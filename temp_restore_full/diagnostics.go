// diagnostics.go
// - Exemplo de ferramenta de diagnóstico (ping) acionada via UI.
// - A Central orquestra execução: chama o Proxy do cliente (outbound) ou
//   usa Ansible para rodar no host de destino.

package main

import (
    "encoding/json"
    "net/http"
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

    // TODO: enfileirar pedido p/ Proxy do tenant executar e retornar output
    // Por enquanto, resposta simulada:
    out := map[string]any{
        "target": req.Target,
        "sent": req.Count,
        "received": req.Count - 1,
        "loss_percent": 25.0,
        "avg_rtt_ms": 120.4,
        "stdout": "PING ...",
    }
    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(out)
}
