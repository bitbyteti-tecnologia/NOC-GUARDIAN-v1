//go:build linux

package metrics

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
)

func ExtraMetrics(diskPath string, services []string, pingTargets []string, topN int) map[string]float64 {
	out := map[string]float64{}
	_ = diskPath

	// CPU cores
	if perCore, err := cpu.Percent(900*time.Millisecond, true); err == nil && len(perCore) > 0 {
		out["cpu_cores"] = float64(len(perCore))
		var sum float64
		for i, v := range perCore {
			out[fmt.Sprintf("cpu_core_%d_pct", i)] = v
			sum += v
		}
		avg := sum / float64(len(perCore))
		out["cpu_available_pct"] = clampPct(100.0 - avg)
	}

	// Memória e Swap
	if vm, err := mem.VirtualMemory(); err == nil {
		out["mem_total_bytes"] = float64(vm.Total)
		out["mem_available_bytes"] = float64(vm.Available)
		out["mem_used_bytes"] = float64(vm.Used)
	}
	if sm, err := mem.SwapMemory(); err == nil {
		out["swap_total_bytes"] = float64(sm.Total)
		out["swap_free_bytes"] = float64(sm.Free)
		out["swap_used_bytes"] = float64(sm.Used)
	}

	// Discos/partições
	if parts, err := disk.Partitions(true); err == nil {
		for _, p := range parts {
			usage, err := disk.Usage(p.Mountpoint)
			if err != nil {
				continue
			}
			key := sanitizeMetricKey(p.Mountpoint)
			out["disk_"+key+"_used_pct"] = usage.UsedPercent
			out["disk_"+key+"_total_bytes"] = float64(usage.Total)
			out["disk_"+key+"_free_bytes"] = float64(usage.Free)
			out["disk_"+key+"_used_bytes"] = float64(usage.Used)
		}
	}

	// Processos Top N (CPU e Mem)
	topCPU, topMem := topProcesses(topN)
	for i, p := range topCPU {
		key := sanitizeMetricKey(p.Name)
		out[fmt.Sprintf("proc_cpu_top%d_%s_pct", i+1, key)] = p.CPU
	}
	for i, p := range topMem {
		key := sanitizeMetricKey(p.Name)
		out[fmt.Sprintf("proc_mem_top%d_%s_bytes", i+1, key)] = float64(p.MemBytes)
	}

	// Serviços (status + recurso best-effort)
	for _, svc := range services {
		sname := sanitizeMetricKey(svc)
		status := checkServiceStatus(svc)
		out["service_"+sname+"_status"] = boolToFloat(status == "active")

		cpuPct, memBytes := processUsageByName(svc)
		if cpuPct >= 0 {
			out["service_"+sname+"_cpu_pct"] = cpuPct
		}
		if memBytes >= 0 {
			out["service_"+sname+"_mem_bytes"] = memBytes
		}
	}

	// Top serviços por CPU/Mem (systemd MainPID)
	topSvcCPU, topSvcMem := topServicesLinux(topN)
	for i, s := range topSvcCPU {
		key := sanitizeMetricKey(s.Name)
		out[fmt.Sprintf("service_cpu_top%d_%s_pct", i+1, key)] = s.CPU
	}
	for i, s := range topSvcMem {
		key := sanitizeMetricKey(s.Name)
		out[fmt.Sprintf("service_mem_top%d_%s_bytes", i+1, key)] = float64(s.MemBytes)
	}

	// Ping (latência e perda)
	for _, tgt := range pingTargets {
		avgMs, lossPct := pingLinux(tgt)
		key := sanitizeMetricKey(tgt)
		if avgMs >= 0 {
			out["ping_"+key+"_avg_ms"] = avgMs
		}
		if lossPct >= 0 {
			out["ping_"+key+"_loss_pct"] = lossPct
		}
	}

	// Atualizações pendentes
	if n := updatesPendingLinux(); n >= 0 {
		out["updates_pending"] = float64(n)
	}

	// Temperatura
	for name, tempC := range readThermalZones() {
		key := sanitizeMetricKey(name)
		out["temp_"+key+"_c"] = tempC
	}

	// Ventoinhas (RPM)
	for name, rpm := range readFanRpm() {
		key := sanitizeMetricKey(name)
		out["fan_"+key+"_rpm"] = rpm
	}

	// Energia (Watts) via RAPL, best-effort
	if w := readPowerWatts(); w >= 0 {
		out["power_pkg_watts"] = w
	}

	// Auditoria (erros últimos 5m)
	if n := journalErrorCount(); n >= 0 {
		out["log_errors_5m"] = float64(n)
	}

	return out
}

type procStat struct {
	Name     string
	CPU      float64
	MemBytes uint64
}

func topProcesses(n int) ([]procStat, []procStat) {
	procs, err := process.Processes()
	if err != nil || len(procs) == 0 || n <= 0 {
		return nil, nil
	}

	var items []procStat
	for _, p := range procs {
		name, err := p.Name()
		if err != nil {
			continue
		}
		cpuPct, _ := p.CPUPercent()
		memInfo, _ := p.MemoryInfo()
		var memBytes uint64
		if memInfo != nil {
			memBytes = memInfo.RSS
		}
		items = append(items, procStat{Name: name, CPU: cpuPct, MemBytes: memBytes})
	}

	byCPU := append([]procStat(nil), items...)
	sort.Slice(byCPU, func(i, j int) bool { return byCPU[i].CPU > byCPU[j].CPU })
	if len(byCPU) > n {
		byCPU = byCPU[:n]
	}

	byMem := append([]procStat(nil), items...)
	sort.Slice(byMem, func(i, j int) bool { return byMem[i].MemBytes > byMem[j].MemBytes })
	if len(byMem) > n {
		byMem = byMem[:n]
	}

	return byCPU, byMem
}

func processUsageByName(name string) (float64, float64) {
	if name == "" {
		return -1, -1
	}
	procs, err := process.Processes()
	if err != nil {
		return -1, -1
	}
	var cpuSum float64
	var memSum float64
	var found bool
	for _, p := range procs {
		pname, err := p.Name()
		if err != nil {
			continue
		}
		if !strings.Contains(strings.ToLower(pname), strings.ToLower(name)) {
			continue
		}
		found = true
		if cpuPct, err := p.CPUPercent(); err == nil {
			cpuSum += cpuPct
		}
		if memInfo, err := p.MemoryInfo(); err == nil && memInfo != nil {
			memSum += float64(memInfo.RSS)
		}
	}
	if !found {
		return -1, -1
	}
	return cpuSum, memSum
}

type svcStat struct {
	Name     string
	CPU      float64
	MemBytes uint64
}

func topServicesLinux(n int) ([]svcStat, []svcStat) {
	if n <= 0 {
		return nil, nil
	}

	units := listRunningServices()
	if len(units) == 0 {
		// Fallback: use top processes when systemd is not available
		topCPU, topMem := topProcesses(n)
		return convertProcToSvc(topCPU), convertProcToSvc(topMem)
	}

	var items []svcStat
	for _, unit := range units {
		pid := systemdMainPID(unit)
		if pid <= 0 {
			continue
		}
		p, err := process.NewProcess(int32(pid))
		if err != nil {
			continue
		}
		cpuPct, _ := p.CPUPercent()
		memInfo, _ := p.MemoryInfo()
		var memBytes uint64
		if memInfo != nil {
			memBytes = memInfo.RSS
		}
		name := strings.TrimSuffix(unit, ".service")
		items = append(items, svcStat{Name: name, CPU: cpuPct, MemBytes: memBytes})
	}

	byCPU := append([]svcStat(nil), items...)
	sort.Slice(byCPU, func(i, j int) bool { return byCPU[i].CPU > byCPU[j].CPU })
	if len(byCPU) > n {
		byCPU = byCPU[:n]
	}

	byMem := append([]svcStat(nil), items...)
	sort.Slice(byMem, func(i, j int) bool { return byMem[i].MemBytes > byMem[j].MemBytes })
	if len(byMem) > n {
		byMem = byMem[:n]
	}

	return byCPU, byMem
}

func listRunningServices() []string {
	cmd := exec.Command("systemctl", "list-units", "--type=service", "--state=running", "--no-legend", "--no-pager")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	lines := strings.Split(string(out), "\n")
	var units []string
	for _, ln := range lines {
		fields := strings.Fields(ln)
		if len(fields) == 0 {
			continue
		}
		unit := fields[0]
		if strings.HasSuffix(unit, ".service") {
			units = append(units, unit)
		}
		if len(units) >= 100 {
			break
		}
	}
	return units
}

func systemdMainPID(unit string) int {
	cmd := exec.Command("systemctl", "show", "-p", "MainPID", "--value", unit)
	out, err := cmd.Output()
	if err != nil {
		return 0
	}
	v := strings.TrimSpace(string(out))
	if v == "" {
		return 0
	}
	pid, _ := strconv.Atoi(v)
	return pid
}

func pingLinux(target string) (avgMs float64, lossPct float64) {
	avgMs, lossPct = -1, -1
	if target == "" {
		return
	}
	cmd := exec.Command("ping", "-c", "3", "-W", "1", target)
	out, err := cmd.CombinedOutput()
	if err != nil && len(out) == 0 {
		return
	}
	s := string(out)
	// packet loss: "0% packet loss"
	if idx := strings.Index(s, "% packet loss"); idx != -1 {
		start := strings.LastIndex(s[:idx], " ")
		if start != -1 {
			lossStr := strings.TrimSpace(s[start:idx])
			if v, err := strconv.ParseFloat(lossStr, 64); err == nil {
				lossPct = v
			}
		}
	}
	// rtt line: "rtt min/avg/max/mdev = 0.026/0.030/0.036/0.004 ms"
	if idx := strings.Index(s, "rtt min/avg/max"); idx != -1 {
		line := s[idx:]
		if eq := strings.Index(line, "="); eq != -1 {
			parts := strings.Split(line[eq+1:], "/")
			if len(parts) >= 2 {
				if v, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64); err == nil {
					avgMs = v
				}
			}
		}
	}
	return
}

func updatesPendingLinux() int {
	if _, err := exec.LookPath("apt"); err == nil {
		cmd := exec.Command("bash", "-lc", "apt list --upgradable 2>/dev/null | grep -v Listing | wc -l")
		out, err := cmd.Output()
		if err != nil {
			return -1
		}
		n, _ := strconv.Atoi(strings.TrimSpace(string(out)))
		return n
	}
	if _, err := exec.LookPath("dnf"); err == nil {
		cmd := exec.Command("bash", "-lc", "dnf -q check-update | grep -E '^[a-zA-Z0-9_.-]+\\.' | wc -l")
		out, err := cmd.Output()
		if err != nil {
			return -1
		}
		n, _ := strconv.Atoi(strings.TrimSpace(string(out)))
		return n
	}
	if _, err := exec.LookPath("yum"); err == nil {
		cmd := exec.Command("bash", "-lc", "yum -q check-update | grep -E '^[a-zA-Z0-9_.-]+\\.' | wc -l")
		out, err := cmd.Output()
		if err != nil {
			return -1
		}
		n, _ := strconv.Atoi(strings.TrimSpace(string(out)))
		return n
	}
	return -1
}

func readThermalZones() map[string]float64 {
	out := map[string]float64{}
	glob, _ := filepath.Glob("/sys/class/thermal/thermal_zone*/temp")
	for _, p := range glob {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		v, err := strconv.ParseFloat(strings.TrimSpace(string(b)), 64)
		if err != nil {
			continue
		}
		// millideg -> deg C
		v = v / 1000.0
		name := filepath.Base(filepath.Dir(p))
		out[name] = v
	}
	return out
}

func readFanRpm() map[string]float64 {
	out := map[string]float64{}
	glob, _ := filepath.Glob("/sys/class/hwmon/hwmon*/fan*_input")
	for _, p := range glob {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		v, err := strconv.ParseFloat(strings.TrimSpace(string(b)), 64)
		if err != nil {
			continue
		}
		name := strings.TrimSuffix(filepath.Base(p), "_input")
		out[name] = v
	}
	return out
}

var (
	lastEnergyUJ float64
	lastEnergyTS time.Time
)

func readPowerWatts() float64 {
	// Intel RAPL path
	path := "/sys/class/powercap/intel-rapl:0/energy_uj"
	b, err := os.ReadFile(path)
	if err != nil {
		return -1
	}
	cur, err := strconv.ParseFloat(strings.TrimSpace(string(b)), 64)
	if err != nil {
		return -1
	}
	now := time.Now()
	if lastEnergyTS.IsZero() {
		lastEnergyTS = now
		lastEnergyUJ = cur
		return -1
	}
	dt := now.Sub(lastEnergyTS).Seconds()
	if dt <= 0 {
		return -1
	}
	dE := cur - lastEnergyUJ
	if dE < 0 {
		// counter reset
		lastEnergyUJ = cur
		lastEnergyTS = now
		return -1
	}
	lastEnergyUJ = cur
	lastEnergyTS = now
	// microjoules per second -> watts
	return (dE / 1e6) / dt
}

func journalErrorCount() int {
	if _, err := exec.LookPath("journalctl"); err != nil {
		return -1
	}
	cmd := exec.Command("journalctl", "-p", "err", "-S", "-5min", "--no-pager")
	out, err := cmd.Output()
	if err != nil {
		return -1
	}
	sc := bufio.NewScanner(bytes.NewReader(out))
	count := 0
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		count++
	}
	return count
}

func convertProcToSvc(items []procStat) []svcStat {
	if len(items) == 0 {
		return nil
	}
	out := make([]svcStat, 0, len(items))
	for _, p := range items {
		out = append(out, svcStat{
			Name:     p.Name,
			CPU:      p.CPU,
			MemBytes: p.MemBytes,
		})
	}
	return out
}
