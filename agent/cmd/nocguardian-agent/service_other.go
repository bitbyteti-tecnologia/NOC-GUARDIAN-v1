//go:build !windows

package main

func isWindowsService() bool {
	return false
}

func runAsWindowsService(cfgPath, diskPath string, intervalSec int, serverURL, tenantID string) {
	// no-op on non-Windows
}

