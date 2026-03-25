package topology

type Node struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Status string `json:"status"`
	Root   bool   `json:"root"`
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
