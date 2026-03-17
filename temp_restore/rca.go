// rca.go
// - Motor de correlação (RCA) inicial baseado em regras simples.
// - Exemplos de regras:
//   * Se muitos devices relatam perda>30% e latência alta logo após "WAN link" cair => WAN raiz.
//   * Se 1 switch core fica DOWN e vários servidores/aps dependem dele => switch é raiz.
// - Pode evoluir para grafos/ML. Aqui geramos sugerindo causa e recomendação.

package main

import (
    "encoding/json"
    "net/http"
)

func RCAHandler(w http.ResponseWriter, r *http.Request) {
    // TODO: Consultar métricas recentes + topologia (labels: depends_on, site, vlan, uplink)
    // TODO: Implementar regras. Aqui respondemos exemplo estático.

    type RCA struct {
        RootCause       string            `json:"root_cause"`
        Confidence      float64           `json:"confidence"`
        Why             []string          `json:"why"`
        Recommendation  string            `json:"recommendation"`
        ImpactedDevices []string          `json:"impacted_devices"`
    }

    resp := RCA{
        RootCause:  "Degradação no Link WAN principal (ISP-1)",
        Confidence: 0.82,
        Why: []string{
            "Perda de pacote > 35% nos gateways de borda",
            "Aumento de latência médio > 200ms em toda matriz",
            "Alertas simultâneos em dispositivos dependentes",
        },
        Recommendation:  "Redirecionar tráfego para link WAN-2; Abrir chamado com ISP-1; Forçar failover em roteador.",
        ImpactedDevices: []string{"gw-matriz", "core-sw-01", "srv-app-01", "srv-db-02"},
    }

    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(resp)
}
