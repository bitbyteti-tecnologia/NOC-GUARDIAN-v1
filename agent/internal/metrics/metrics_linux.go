//go:build linux

package metrics

import (
	"bufio"
	"errors"
	"os"
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
	mem, err := memUsedPercent()
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

	sys, hasSys := collectSystemInfo()

	return Snapshot{
		TS:          time.Now().UTC(),
		CPUPercent:  cpu,
		MemUsedPct:  mem,
		DiskUsedPct: disk,
		DiskPath:    diskPath,
		HasSys:      hasSys,
		System:      sys,
	}, nil
}
