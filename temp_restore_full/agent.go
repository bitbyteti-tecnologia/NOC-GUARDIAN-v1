// agent.go
// - Coleta CPU, Mem, Disco, Rede locais e envia ao Proxy/Central.
// - Cross-platform (Linux/Windows).
// - Em produção: assinar mensagens, TLS mTLS, etc.

package main

import (
    "encoding/json"
    "os"
    "time"

    "github.com/joho/godotenv"
    "github.com/shirou/gopsutil/v3/cpu"
    "github.com/shirou/gopsutil/v3/disk"
    "github.com/shirou/gopsutil/v3/mem"
    "github.com/valyala/fasthttp"
)

func main() {
    _ = godotenv.Load()
    interval := 30
    if v := os.Getenv("INTERVAL_SEC"); v != "" { /* parse... */ }
    client := &fasthttp.Client{}

    for {
        payload := gather()
        b, _ := json.Marshal(payload)
        // TODO: enviar para endpoint local do Proxy; por ora: só imprime
        _ = b
        time.Sleep(time.Duration(interval) * time.Second)
    }
}

func gather() map[string]any {
    tm := time.Now().UTC().Format(time.RFC3339)
    c, _ := cpu.Percent(time.Second, false)
    m, _ := mem.VirtualMemory()
    d, _ := disk.Usage("/")
    return map[string]any{
        "time": tm,
        "cpu_percent": c,
        "mem_percent": m.UsedPercent,
        "disk_percent": d.UsedPercent,
    }
}
