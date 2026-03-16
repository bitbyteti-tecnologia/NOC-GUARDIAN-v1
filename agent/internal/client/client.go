package client

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"
)

type Client struct {
	BaseURL  string
	TenantID string
	Timeout  time.Duration
}

type Metric struct {
	T string  `json:"t"` // ISO timestamp
	V float64 `json:"v"` // valor
}

type Payload struct {
	AgentID  string            `json:"agent_id"`
	Hostname string            `json:"hostname"`
	OS       string            `json:"os"`
	Version  string            `json:"version"`
	DiskPath string            `json:"disk_path"`
	Metrics  map[string]Metric `json:"metrics"`
}

func (c *Client) Ingest(ingestURL string, payload Payload) (*http.Response, error) {
	if c.Timeout == 0 {
		c.Timeout = 10 * time.Second
	}
	body, _ := json.Marshal(payload)

	url := ingestURL
	if url == "" {
		// default: /api/v1/{tenantID}/metrics/ingest
		url = c.BaseURL + "/api/v1/" + c.TenantID + "/metrics/ingest"
	}

	req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	hc := &http.Client{Timeout: c.Timeout}
	return hc.Do(req)
}
