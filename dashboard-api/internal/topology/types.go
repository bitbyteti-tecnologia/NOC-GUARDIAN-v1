package topology

import "time"

type Node struct {
	ID            string     `json:"id"`
	Label         string     `json:"label"`
	Status        string     `json:"status"`
	Root          bool       `json:"root"`
	LastSeen      *time.Time `json:"last_seen"`
	IncidentCount int        `json:"incident_count"`
	Metrics       Metrics    `json:"metrics"`
}

type Metrics struct {
	CPUPercent  *float64 `json:"cpu_percent"`
	MemUsedPct  *float64 `json:"mem_used_pct"`
	DiskUsedPct *float64 `json:"disk_used_pct"`
}

type Edge struct {
	Source       string `json:"source"`
	Target       string `json:"target"`
	RelationType string `json:"relation_type"`
}

type Response struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}
