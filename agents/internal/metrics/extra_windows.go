//go:build windows

package metrics

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
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

	// Processos Top N
	topCPU, topMem := topProcessesWindows(topN)
	for i, p := range topCPU {
		key := sanitizeMetricKey(p.Name)
		out[fmt.Sprintf("proc_cpu_top%d_%s_pct", i+1, key)] = p.CPU
	}
	for i, p := range topMem {
		key := sanitizeMetricKey(p.Name)
		out[fmt.Sprintf("proc_mem_top%d_%s_bytes", i+1, key)] = float64(p.MemBytes)
	}

	// Serviços (status) via sc query
	for _, name := range services {
		status := serviceStatusWindows(name)
		key := sanitizeMetricKey(name)
		out["service_"+key+"_status"] = boolToFloat(status)
	}

	// Top serviços por CPU/Mem
	topSvcCPU, topSvcMem := topServicesWindows(topN)
	for i, s := range topSvcCPU {
		key := sanitizeMetricKey(s.Name)
		out[fmt.Sprintf("service_cpu_top%d_%s_pct", i+1, key)] = s.CPU
	}
	for i, s := range topSvcMem {
		key := sanitizeMetricKey(s.Name)
		out[fmt.Sprintf("service_mem_top%d_%s_bytes", i+1, key)] = float64(s.MemBytes)
	}

	// Ping
	for _, tgt := range pingTargets {
		avgMs, lossPct := pingWindows(tgt)
		key := sanitizeMetricKey(tgt)
		if avgMs >= 0 {
			out["ping_"+key+"_avg_ms"] = avgMs
		}
		if lossPct >= 0 {
			out["ping_"+key+"_loss_pct"] = lossPct
		}
	}

	// Atualizações pendentes (best-effort via COM)
	if n := updatesPendingWindows(); n >= 0 {
		out["updates_pending"] = float64(n)
	}

	return out
}

type procStatWin struct {
	Name     string
	CPU      float64
	MemBytes uint64
}

func topProcessesWindows(n int) ([]procStatWin, []procStatWin) {
	procs, err := process.Processes()
	if err != nil || len(procs) == 0 || n <= 0 {
		return nil, nil
	}
	var items []procStatWin
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
		items = append(items, procStatWin{Name: name, CPU: cpuPct, MemBytes: memBytes})
	}
	byCPU := append([]procStatWin(nil), items...)
	sort.Slice(byCPU, func(i, j int) bool { return byCPU[i].CPU > byCPU[j].CPU })
	if len(byCPU) > n {
		byCPU = byCPU[:n]
	}
	byMem := append([]procStatWin(nil), items...)
	sort.Slice(byMem, func(i, j int) bool { return byMem[i].MemBytes > byMem[j].MemBytes })
	if len(byMem) > n {
		byMem = byMem[:n]
	}
	return byCPU, byMem
}

type svcStatWin struct {
	Name     string
	PID      int32
	CPU      float64
	MemBytes uint64
}

func topServicesWindows(n int) ([]svcStatWin, []svcStatWin) {
	if n <= 0 {
		return nil, nil
	}
	services := listRunningServicesWindows()
	if len(services) == 0 {
		// Fallback: use top processes to avoid empty Top Services
		topCPU, topMem := topProcessesWindows(n)
		return convertProcToSvc(topCPU), convertProcToSvc(topMem)
	}

	var items []svcStatWin
	for _, s := range services {
		if s.PID <= 0 {
			continue
		}
		p, err := process.NewProcess(s.PID)
		if err != nil {
			continue
		}
		cpuPct, _ := p.CPUPercent()
		memInfo, _ := p.MemoryInfo()
		var memBytes uint64
		if memInfo != nil {
			memBytes = memInfo.RSS
		}
		items = append(items, svcStatWin{Name: s.Name, PID: s.PID, CPU: cpuPct, MemBytes: memBytes})
	}

	byCPU := append([]svcStatWin(nil), items...)
	sort.Slice(byCPU, func(i, j int) bool { return byCPU[i].CPU > byCPU[j].CPU })
	if len(byCPU) > n {
		byCPU = byCPU[:n]
	}

	byMem := append([]svcStatWin(nil), items...)
	sort.Slice(byMem, func(i, j int) bool { return byMem[i].MemBytes > byMem[j].MemBytes })
	if len(byMem) > n {
		byMem = byMem[:n]
	}

	return byCPU, byMem
}

func listRunningServicesWindows() []svcStatWin {
	ps := findPowerShell()
	cmd := exec.Command(ps, "-NoProfile", "-Command", "Get-CimInstance Win32_Service | where {$_.State -eq 'Running' -and $_.ProcessId -ne 0} | select Name,ProcessId | ConvertTo-Csv -NoTypeInformation")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) <= 1 {
		return nil
	}

	var services []svcStatWin
	for _, ln := range lines[1:] { // skip header
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		// CSV: "Name","ProcessId"
		parts := splitCSVLine(ln)
		if len(parts) < 2 {
			continue
		}
		name := strings.Trim(parts[0], `"`)
		pidStr := strings.Trim(parts[1], `"`)
		pid64, _ := strconv.ParseInt(pidStr, 10, 32)
		if pid64 <= 0 {
			continue
		}
		services = append(services, svcStatWin{Name: name, PID: int32(pid64)})
		if len(services) >= 200 {
			break
		}
	}
	return services
}

func findPowerShell() string {
	candidates := []string{
		`C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`,
		`C:\Windows\Sysnative\WindowsPowerShell\v1.0\powershell.exe`,
		`C:\Windows\System32\WindowsPowerShell\v1.0\powershell`,
		`powershell.exe`,
		`powershell`,
		`pwsh.exe`,
		`pwsh`,
	}
	for _, c := range candidates {
		if strings.Contains(c, `C:\`) {
			if _, err := os.Stat(c); err == nil {
				return c
			}
			continue
		}
		if p, err := exec.LookPath(c); err == nil {
			return p
		}
	}
	return "powershell"
}

func convertProcToSvc(items []procStatWin) []svcStatWin {
	if len(items) == 0 {
		return nil
	}
	out := make([]svcStatWin, 0, len(items))
	for _, p := range items {
		out = append(out, svcStatWin{
			Name:     p.Name,
			PID:      0,
			CPU:      p.CPU,
			MemBytes: p.MemBytes,
		})
	}
	return out
}

func splitCSVLine(line string) []string {
	var out []string
	var buf strings.Builder
	inQuotes := false
	for i := 0; i < len(line); i++ {
		ch := line[i]
		if ch == '"' {
			inQuotes = !inQuotes
			continue
		}
		if ch == ',' && !inQuotes {
			out = append(out, buf.String())
			buf.Reset()
			continue
		}
		buf.WriteByte(ch)
	}
	out = append(out, buf.String())
	return out
}

func serviceStatusWindows(name string) bool {
	if name == "" {
		return false
	}
	cmd := exec.Command("sc", "query", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	s := strings.ToUpper(string(out))
	return strings.Contains(s, "STATE") && strings.Contains(s, "RUNNING")
}

func pingWindows(target string) (avgMs float64, lossPct float64) {
	avgMs, lossPct = -1, -1
	if target == "" {
		return
	}
	cmd := exec.Command("ping", "-n", "3", "-w", "1000", target)
	out, err := cmd.CombinedOutput()
	if err != nil && len(out) == 0 {
		return
	}
	s := string(out)
	// loss: "Lost = 0 (0% loss)"
	if idx := strings.Index(strings.ToLower(s), "% loss"); idx != -1 {
		start := strings.LastIndex(s[:idx], "(")
		if start != -1 {
			lossStr := strings.TrimSpace(s[start+1 : idx])
			lossStr = strings.TrimSuffix(lossStr, "%")
			if v, err := strconv.ParseFloat(lossStr, 64); err == nil {
				lossPct = v
			}
		}
	}
	// average: "Average = 12ms"
	if idx := strings.Index(strings.ToLower(s), "average"); idx != -1 {
		line := s[idx:]
		if eq := strings.Index(line, "="); eq != -1 {
			part := strings.TrimSpace(line[eq+1:])
			part = strings.TrimSuffix(part, "ms")
			part = strings.Fields(part)[0]
			if v, err := strconv.ParseFloat(part, 64); err == nil {
				avgMs = v
			}
		}
	}
	return
}

func updatesPendingWindows() int {
	// Best-effort via COM object; if PowerShell not available, return -1
	cmd := exec.Command("powershell", "-NoProfile", "-Command", "(New-Object -ComObject Microsoft.Update.Session).CreateUpdateSearcher().Search('IsInstalled=0 and Type=''Software''').Updates.Count")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return -1
	}
	s := strings.TrimSpace(string(bytes.TrimSpace(out)))
	n, err := strconv.Atoi(s)
	if err != nil {
		return -1
	}
	return n
}
