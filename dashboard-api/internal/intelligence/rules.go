package intelligence

func SeverityWeight(sev string) int {
	switch sev {
	case "critical":
		return -20
	case "warning":
		return -10
	case "info":
		return -5
	default:
		return -3
	}
}

func SeverityRank(sev string) int {
	switch sev {
	case "critical":
		return 3
	case "warning":
		return 2
	case "info":
		return 1
	default:
		return 0
	}
}

func StatusFromScore(score int) string {
	switch {
	case score >= 90:
		return "healthy"
	case score >= 70:
		return "warning"
	default:
		return "critical"
	}
}

func RecommendForEvent(eventType string) string {
	switch eventType {
	case "cpu_high":
		return "Verificar processos, carga e consumo de CPU"
	case "latency_high":
		return "Verificar qualidade do link, perda de pacotes e rota"
	case "memory_high":
		return "Verificar consumo de memória e possíveis vazamentos"
	case "device_offline":
		return "Verificar conectividade, energia e links físicos"
	default:
		return "Investigar causa raiz e validar impacto no serviço"
	}
}
