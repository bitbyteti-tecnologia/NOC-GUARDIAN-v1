// diagnostics.go
// - Exemplo de ferramenta de diagnóstico (ping) acionada via UI.
// - A Central orquestra execução: chama o Proxy do cliente (outbound) ou
//   usa Ansible para rodar no host de destino.

package main

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
)

func DiagnosticPingHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Target string `json:"target"`
		Count  int    `json:"count"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Count <= 0 {
		req.Count = 4
	}

	mode := strings.TrimSpace(strings.ToLower(os.Getenv("DIAG_MODE")))
	if mode == "" {
		mode = "simulate"
	}
	if mode == "disabled" {
		http.Error(w, "diagnostics disabled", http.StatusNotImplemented)
		return
	}

	// TODO: enfileirar pedido p/ Proxy do tenant executar e retornar output
	// Por enquanto, resposta simulada:
	out := map[string]any{
		"target":         req.Target,
		"sent":           req.Count,
		"received":       req.Count - 1,
		"loss_percent":   25.0,
		"avg_rtt_ms":     120.4,
		"stdout":         "PING ...",
		"simulated":      true,
		"execution_mode": mode,
		"note":           "Simulado. Configure o runner do Proxy para execução real.",
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}
