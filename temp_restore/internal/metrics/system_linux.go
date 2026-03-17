//go:build linux

package metrics

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type SystemInfo struct {
	UptimeSec    float64
	Load1        float64
	Load5        float64
	Load15       float64
	ProcCount    float64
	ThreadCount  float64
	KThreadCount float64
	RunningProcs float64
}

// Retorna (info, ok). ok=false se falhar geral.
func collectSystemInfo() (SystemInfo, bool) {
	var info SystemInfo

	// uptime
	if b, err := os.ReadFile("/proc/uptime"); err == nil {
		parts := strings.Fields(string(b))
		if len(parts) >= 1 {
			if v, err := strconv.ParseFloat(parts[0], 64); err == nil {
				info.UptimeSec = v
			}
		}
	}

	// loadavg
	if b, err := os.ReadFile("/proc/loadavg"); err == nil {
		parts := strings.Fields(string(b))
		if len(parts) >= 3 {
			if v, err := strconv.ParseFloat(parts[0], 64); err == nil {
				info.Load1 = v
			}
			if v, err := strconv.ParseFloat(parts[1], 64); err == nil {
				info.Load5 = v
			}
			if v, err := strconv.ParseFloat(parts[2], 64); err == nil {
				info.Load15 = v
			}
		}
	}

	// processos + threads + kthreads + running
	procDirEntries, err := os.ReadDir("/proc")
	if err != nil {
		return info, false
	}

	var procCount, thrCount, kthrCount, runningCount int

	for _, e := range procDirEntries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		// somente pids numéricos
		if name == "" || name[0] < 0 || name[0] > 9 {
			continue
		}
		procCount++

		pidPath := filepath.Join("/proc", name)

		// kthread heurística: comm com [xxx]
		if b, err := os.ReadFile(filepath.Join(pidPath, "comm")); err == nil {
			comm := strings.TrimSpace(string(b))
			if strings.HasPrefix(comm, "[") && strings.HasSuffix(comm, "]") {
				kthrCount++
			}
		}

		// status: Threads e State
		if b, err := os.ReadFile(filepath.Join(pidPath, "status")); err == nil {
			sc := bufio.NewScanner(bytes.NewReader(b))
			var threads int
			var isRunning bool
			for sc.Scan() {
				line := sc.Text()
				if strings.HasPrefix(line, "Threads:") {
					fields := strings.Fields(line)
					if len(fields) >= 2 {
						if n, err := strconv.Atoi(fields[1]); err == nil {
							threads = n
						}
					}
				}
				if strings.HasPrefix(line, "State:") {
					// exemplo: "State:\tR (running)"
					fields := strings.Fields(line)
					if len(fields) >= 2 && fields[1] == "R" {
						isRunning = true
					}
				}
			}
			thrCount += threads
			if isRunning {
				runningCount++
			}
		}
	}

	info.ProcCount = float64(procCount)
	info.ThreadCount = float64(thrCount)
	info.KThreadCount = float64(kthrCount)
	info.RunningProcs = float64(runningCount)

	return info, true
}

// Export: usado pelo main.go para anexar métricas extras
func GetSystemInfo() (SystemInfo, bool) {
    return collectSystemInfo()
}
