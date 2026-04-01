package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func initConfig(path, serverURL, tenantID string) error {
	if path == "" {
		return fmt.Errorf("init-config: path is required")
	}
	if serverURL == "" || tenantID == "" {
		return fmt.Errorf("init-config: server_url and tenant_id are required")
	}

	// cria diretórios
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	content := fmt.Sprintf(`# NOC Guardian Agent (Windows)
server_url: %s
tenant_id: %s
# agent_id será preenchido automaticamente na primeira execução
`, serverURL, tenantID)

	return os.WriteFile(path, []byte(content), 0644)
}
