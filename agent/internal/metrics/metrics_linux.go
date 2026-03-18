//go:build linux

package metrics

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type Snapshot struct {
	TS          time.Time
	CPUPercent  float64
	MemUsedPct  float64
	DiskUsedPct float64
	DiskPath    string

	MemTotalBytes float64
	MemUsedBytes  float64

	NetRxBps      float64
	NetTxBps      float64
	DiskReadBps   float64
	DiskWriteBps  float64

	Services map[string]string

	HasSys bool
	System SystemInfo
}

type cpuTimes struct {
	idle  uint64
	total uint64
}

func readCPUTimes() (cpuTimes, error) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return cpuTimes{}, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "cpu ") {
			fields := strings.Fields(line)
			if len(fields) < 5 {
				return cpuTimes{}, errors.New("unexpected /proc/stat format")
			}
			// cpu user nice system idle iowait irq softirq steal guest guest_nice
			var vals []uint64
			for i := 1; i < len(fields); i++ {
				n, _ := strconv.ParseUint(fields[i], 10, 64)
				vals = append(vals, n)
			}
			idle := vals[3]
			if len(vals) > 4 { // iowait
				idle += vals[4]
			}
			var total uint64
			for _, v := range vals {
				total += v
			}
			return cpuTimes{idle: idle, total: total}, nil
		}
	}
	return cpuTimes{}, errors.New("cpu line not found in /proc/stat")
}

func cpuPercent(prev, cur cpuTimes) float64 {
	dIdle := float64(cur.idle - prev.idle)
	dTotal := float64(cur.total - prev.total)
	if dTotal <= 0 {
		return 0
	}
	used := (dTotal - dIdle) / dTotal * 100.0
	if used < 0 {
		return 0
	}
	if used > 100 {
		return 100
	}
	return used
}

func memUsedPercent() (float64, error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, err
	}
	defer f.Close()

	var total, avail float64
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			total, _ = strconv.ParseFloat(fields[1], 64)
		}
		if strings.HasPrefix(line, "MemAvailable:") {
			fields := strings.Fields(line)
			avail, _ = strconv.ParseFloat(fields[1], 64)
		}
	}
	if total <= 0 {
		return 0, errors.New("mem total not found")
	}
	used := (total - avail) / total * 100.0
	if used < 0 {
		used = 0
	}
	if used > 100 {
		used = 100
	}
	return used, nil
}

func diskUsedPercent(path string) (float64, error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 0, err
	}
	total := float64(st.Blocks) * float64(st.Bsize)
	free := float64(st.Bfree) * float64(st.Bsize)
	if total <= 0 {
		return 0, errors.New("disk total invalid")
	}
	used := (total - free) / total * 100.0
	if used < 0 {
		used = 0
	}
	if used > 100 {
		used = 100
	}
	return used, nil
}

func Collect(diskPath string) (Snapshot, error) {
	ts := time.Now().UTC()

	prev, err := readCPUTimes()
	if err != nil {
		return Snapshot{}, err
	}
	time.Sleep(900 * time.Millisecond)
	cur, err := readCPUTimes()
	if err != nil {
		return Snapshot{}, err
	}

	cpu := cpuPercent(prev, cur)
	memPct, totalBytes, usedBytes, err := memInfo()
	if err != nil {
		return Snapshot{}, err
	}
	if diskPath == "" {
		diskPath = "/"
	}
	disk, err := diskUsedPercent(diskPath)
	if err != nil {
		return Snapshot{}, err
	}

	netRxBps, netTxBps := netBps(ts)
	diskReadBps, diskWriteBps := diskIoBps(ts, diskPath)

	sys, hasSys := collectSystemInfo()

	// Monitorar serviços específicos
	services := make(map[string]string)
	checkServices := []string{"docker", "nginx", "postgresql", "central", "dashboard"}
	for _, s := range checkServices {
		services[s] = checkServiceStatus(s)
	}

	return Snapshot{
		TS:            ts,
		CPUPercent:    cpu,
		MemUsedPct:    memPct,
		MemTotalBytes: totalBytes,
		MemUsedBytes:  usedBytes,
		DiskUsedPct:   disk,
		DiskPath:      diskPath,
		NetRxBps:      netRxBps,
		NetTxBps:      netTxBps,
		DiskReadBps:   diskReadBps,
		DiskWriteBps:  diskWriteBps,
		Services:      services,
		HasSys:        hasSys,
		System:        sys,
	}, nil
}

func checkServiceStatus(name string) string {
	// Tenta via systemctl
	cmd := exec.Command("systemctl", "is-active", name)
	out, err := cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(out))
	}
	return "inactive"
}

func memInfo() (pct float64, total float64, used float64, err error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0, 0, err
	}
	defer f.Close()

	var memTotal, memFree, buffers, cached float64
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		val, _ := strconv.ParseFloat(parts[1], 64)
		switch parts[0] {
		case "MemTotal:":
			memTotal = val * 1024
		case "MemFree:":
			memFree = val * 1024
		case "Buffers:":
			buffers = val * 1024
		case "Cached:":
			cached = val * 1024
		}
	}
	if memTotal == 0 {
		return 0, 0, 0, fmt.Errorf("could not parse MemTotal")
	}
	// Usado = Total - Livre - Buffers - Cached (estilo htop/free)
	used = memTotal - memFree - buffers - cached
	pct = (used / memTotal) * 100
	return pct, memTotal, used, nil
}

var (
	lastNetTS    time.Time
	lastNetRx    float64
	lastNetTx    float64
	lastDiskTS   time.Time
	lastDiskRead float64
	lastDiskWrite float64
)

func netBps(now time.Time) (float64, float64) {
	rx, tx, err := readNetBytes()
	if err != nil {
		return 0, 0
	}
	if lastNetTS.IsZero() {
		lastNetTS = now
		lastNetRx = rx
		lastNetTx = tx
		return 0, 0
	}
	dt := now.Sub(lastNetTS).Seconds()
	if dt <= 0 {
		return 0, 0
	}
	drx := rx - lastNetRx
	dtx := tx - lastNetTx
	if drx < 0 {
		drx = 0
	}
	if dtx < 0 {
		dtx = 0
	}
	lastNetTS = now
	lastNetRx = rx
	lastNetTx = tx
	return drx / dt, dtx / dt
}

func diskIoBps(now time.Time, diskPath string) (float64, float64) {
	readB, writeB, err := readDiskBytes(diskPath)
	if err != nil {
		return 0, 0
	}
	if lastDiskTS.IsZero() {
		lastDiskTS = now
		lastDiskRead = readB
		lastDiskWrite = writeB
		return 0, 0
	}
	dt := now.Sub(lastDiskTS).Seconds()
	if dt <= 0 {
		return 0, 0
	}
	dread := readB - lastDiskRead
	dwrite := writeB - lastDiskWrite
	if dread < 0 {
		dread = 0
	}
	if dwrite < 0 {
		dwrite = 0
	}
	lastDiskTS = now
	lastDiskRead = readB
	lastDiskWrite = writeB
	return dread / dt, dwrite / dt
}

func readNetBytes() (float64, float64, error) {
	f, err := os.Open("/proc/net/dev")
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()

	var rx, tx float64
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "Inter-") || strings.HasPrefix(line, "face") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 10 {
			continue
		}
		iface := strings.TrimSuffix(parts[0], ":")
		if iface == "lo" {
			continue
		}
		rxVal, _ := strconv.ParseFloat(parts[1], 64)
		txVal, _ := strconv.ParseFloat(parts[9], 64)
		rx += rxVal
		tx += txVal
	}
	if err := sc.Err(); err != nil {
		return 0, 0, err
	}
	return rx, tx, nil
}

func readDiskBytes(diskPath string) (float64, float64, error) {
	device, err := resolveDeviceForPath(diskPath)
	if err != nil {
		return 0, 0, err
	}

	f, err := os.Open("/proc/diskstats")
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 14 {
			continue
		}
		name := fields[2]
		if name != device {
			continue
		}
		// sectors read (6) and written (10)
		reads, _ := strconv.ParseFloat(fields[5], 64)
		writes, _ := strconv.ParseFloat(fields[9], 64)
		return reads * 512.0, writes * 512.0, nil
	}
	if err := sc.Err(); err != nil {
		return 0, 0, err
	}
	return 0, 0, fmt.Errorf("device not found in /proc/diskstats: %s", device)
}

func resolveDeviceForPath(p string) (string, error) {
	if p == "" {
		p = "/"
	}
	mounts, err := os.Open("/proc/mounts")
	if err != nil {
		return "", err
	}
	defer mounts.Close()

	var bestDev string
	var bestMount string
	sc := bufio.NewScanner(mounts)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 2 {
			continue
		}
		dev := fields[0]
		mp := fields[1]
		if !strings.HasPrefix(p, mp) {
			continue
		}
		if len(mp) > len(bestMount) {
			bestMount = mp
			bestDev = dev
		}
	}
	if err := sc.Err(); err != nil {
		return "", err
	}
	if bestDev == "" {
		return "", fmt.Errorf("mount not found for path: %s", p)
	}

	base := strings.TrimPrefix(bestDev, "/dev/")
	return base, nil
}
