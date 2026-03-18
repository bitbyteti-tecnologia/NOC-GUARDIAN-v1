package main

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/bitbyteti/noc-guardian/agent/internal/client"
	"github.com/bitbyteti/noc-guardian/agent/internal/config"
	"github.com/bitbyteti/noc-guardian/agent/internal/metrics"
)

const Version = "0.1.1"

func genID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Escreve/atualiza agent_id no YAML (simples e robusto)
func writeAgentID(path, agentID string) {
	in, err := os.ReadFile(path)
	if err != nil {
		return
	}
	lines := strings.Split(string(in), "\n")
	out := make([]string, 0, len(lines)+1)

	found := false
	for _, ln := range lines {
		t := strings.ToLower(strings.TrimSpace(ln))
		if strings.HasPrefix(t, "agent_id:") {
			out = append(out, "agent_id: "+agentID)
			found = true
		} else {
			out = append(out, ln)
		}
	}
	if !found {
		out = append(out, "agent_id: "+agentID)
	}
	_ = os.WriteFile(path, []byte(strings.Join(out, "\n")), 0644)
}

func main() {
	var cfgPath string
	var diskPath string
	var intervalSec int

	// Para ServiceInstall no Windows (Arguments)
	var serverURL string
	var tenantID string

	// Modo opcional: gerar config e sair
	var initCfgPath string

	flag.StringVar(&cfgPath, "config", "/etc/nocguardian/agent.yml", "config file")

	defDisk := "/"
	if runtime.GOOS == "windows" {
		defDisk = `C:\`
	}
	flag.StringVar(&diskPath, "disk", defDisk, "disk path to monitor (Linux: /, Windows: C:\\)")
	flag.IntVar(&intervalSec, "interval", 30, "seconds between sends")

	flag.StringVar(&serverURL, "server-url", "", "server url (optional if config exists)")
	flag.StringVar(&tenantID, "tenant-id", "", "tenant id (optional if config exists)")

	flag.StringVar(&initCfgPath, "init-config", "", "write config to path and exit (optional)")
	flag.Parse()

	// Execução como serviço no Windows
	if runtime.GOOS == "windows" && isWindowsService() {
		runAsWindowsService(cfgPath, diskPath, intervalSec, serverURL, tenantID)
		return
	}

	// init-config manual (opcional para troubleshooting)
	if initCfgPath != "" {
		if err := initConfig(initCfgPath, serverURL, tenantID); err != nil {
			log.Fatalf("init-config error: %v", err)
		}
		log.Printf("init-config OK: %s", initCfgPath)
		return
	}

	if err := runAgent(cfgPath, diskPath, intervalSec, serverURL, tenantID, nil); err != nil {
		log.Fatalf("agent error: %v", err)
	}
}

func runAgent(cfgPath, diskPath string, intervalSec int, serverURL, tenantID string, stop <-chan struct{}) error {
	// Carrega config
	cfg, err := config.Load(cfgPath)

	// Se config não existe e server-url/tenant-id foram passados (MSI/Service), cria e carrega
	if err != nil {
		if os.IsNotExist(err) && serverURL != "" && tenantID != "" {
			if err2 := initConfig(cfgPath, serverURL, tenantID); err2 != nil {
				return fmt.Errorf("auto init-config error: %w", err2)
			}
			cfg, err = config.Load(cfgPath)
		}
	}
	if err != nil {
		return fmt.Errorf("config error: %w", err)
	}

	// AgentID: gera e persiste se não existir
	if cfg.AgentID == "" {
		cfg.AgentID = genID()
		writeAgentID(cfgPath, cfg.AgentID)
	}

	hostname, _ := os.Hostname()
	osName := fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)

	cl := &client.Client{
		BaseURL:  strings.TrimRight(cfg.ServerURL, "/"),
		TenantID: cfg.TenantID,
		Timeout:  10 * time.Second,
	}

	log.Printf("nocguardian-agent v%s starting | tenant=%s | agent=%s | server=%s | os=%s",
		Version, cfg.TenantID, cfg.AgentID, cfg.ServerURL, osName)

	ticker := time.NewTicker(time.Duration(intervalSec) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			log.Printf("shutdown requested")
			return nil
		default:
		}

		snap, err := metrics.Collect(diskPath)
		if err != nil {
			log.Printf("collect error: %v", err)
			select {
			case <-ticker.C:
				continue
			case <-stop:
				log.Printf("shutdown requested")
				return nil
			}
		}

		ts := snap.TS.Format(time.RFC3339)

		metricMap := map[string]client.Metric{
			"agent_heartbeat": {T: ts, V: 1},
			"cpu_percent":     {T: ts, V: snap.CPUPercent},
			"mem_used_pct":    {T: ts, V: snap.MemUsedPct},
			"disk_used_pct":   {T: ts, V: snap.DiskUsedPct},
			"net_rx_bps":      {T: ts, V: snap.NetRxBps},
			"net_tx_bps":      {T: ts, V: snap.NetTxBps},
			"disk_read_bps":   {T: ts, V: snap.DiskReadBps},
			"disk_write_bps":  {T: ts, V: snap.DiskWriteBps},
		}

		// ✅ Novas métricas de sistema (Linux e Windows)
		// Windows: sem load average (não envia load1/load5/load15)
		if snap.HasSys {
			metricMap["uptime_sec"] = client.Metric{T: ts, V: snap.System.UptimeSec}
			metricMap["proc_count"] = client.Metric{T: ts, V: snap.System.ProcCount}
			metricMap["thread_count"] = client.Metric{T: ts, V: snap.System.ThreadCount}
			metricMap["mem_total_bytes"] = client.Metric{T: ts, V: snap.MemTotalBytes}
			metricMap["mem_used_bytes"] = client.Metric{T: ts, V: snap.MemUsedBytes}

			// Monitoramento de Serviços (0=inactive, 1=active)
			for name, status := range snap.Services {
				val := 0.0
				if status == "active" {
					val = 1.0
				}
				metricMap["service_"+name+"_status"] = client.Metric{T: ts, V: val}
			}

			// Linux: load average + kthreads + running
			if runtime.GOOS != "windows" {
				metricMap["load1"] = client.Metric{T: ts, V: snap.System.Load1}
				metricMap["load5"] = client.Metric{T: ts, V: snap.System.Load5}
				metricMap["load15"] = client.Metric{T: ts, V: snap.System.Load15}
				metricMap["kthread_count"] = client.Metric{T: ts, V: snap.System.KThreadCount}
				metricMap["running_procs"] = client.Metric{T: ts, V: snap.System.RunningProcs}
			}
		}

		payload := client.Payload{
			AgentID:  cfg.AgentID,
			Hostname: hostname,
			OS:       osName,
			Version:  Version,
			DiskPath: snap.DiskPath,
			Metrics:  metricMap,
		}

		resp, err := cl.Ingest(cfg.IngestURL, payload)
		if err != nil {
			log.Printf("send error: %v", err)
		} else {
			_ = resp.Body.Close()
			if resp.StatusCode >= 300 {
				log.Printf("send status: %s", resp.Status)
			}
		}

		select {
		case <-ticker.C:
		case <-stop:
			log.Printf("shutdown requested")
			return nil
		}
	}
}
