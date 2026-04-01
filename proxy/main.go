// main.go (PROXY)
// - Executa na rede do cliente.
// - Faz scan SNMP local (LAN) e coleta métricas dos agents.
// - Mantém buffer local (SQLite) se internet cair.
// - Envia lotes de métricas para a Central via HTTPs (TLS 1.3).
// - "Outbound-only": nenhuma porta inbound aberta.
// - Opcional: MTLS (cliente->servidor) e payload AES-256-GCM.

package main

import (
	"crypto/tls"
	"encoding/json"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bitbyteti/noc-guardian/proxy/internal"
	"github.com/joho/godotenv"
	"github.com/valyala/fasthttp"
)

func main() {
    _ = godotenv.Load()

    // Inicializa buffer local (SQLite)
    if err := internal.InitBuffer(os.Getenv("BUFFER_DB")); err != nil {
        log.Fatalf("buffer init: %v", err)
    }

    // Cliente HTTP com TLS forte; validação de certs (configurar CA se mTLS)
    client := &fasthttp.Client{
        TLSConfig: &tls.Config{
            MinVersion: tls.VersionTLS13,
        },
        ReadTimeout:  10 * time.Second,
        WriteTimeout: 10 * time.Second,
    }

    // Loop: varredura + envio
    ticker := time.NewTicker(time.Duration(getenvInt("SCAN_INTERVAL_SEC", 60)) * time.Second)
    defer ticker.Stop()
    for {
        runOnce(client)
        <-ticker.C
    }
}

func runOnce(client *fasthttp.Client) {
    // 1) Scan SNMP (subnets)
    subnets := strings.Split(os.Getenv("SNMP_TARGETS"), ",")
    metrics := internal.ScanSNMP(subnets, os.Getenv("SNMP_COMMUNITY"))
    // 2) Coleta dos Agents (se publicar em fila local, etc.) — TODO

    // 3) Empacota e grava no buffer
    payload := MetricsToJSON(metrics)
    if err := internal.BufferAppend(payload); err != nil {
        log.Printf("buffer append err: %v", err)
    }

    // 4) Tenta enviar tudo o que há no buffer
    if err := internal.FlushBuffer(client, os.Getenv("CENTRAL_URL")+os.Getenv("INGEST_ENDPOINT"), os.Getenv("AUTH_TOKEN")); err != nil {
        log.Printf("flush err: %v", err)
    }
}

func MetricsToJSON(points []internal.MetricPoint) []byte {
    b, _ := json.Marshal(points)
    // Opcional: criptografar payload com AES-256-GCM antes de enviar (além de TLS)
    // b = EncryptAESGCM(b, os.Getenv("AES_KEY_BASE64"))
    return b
}

func getenvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
