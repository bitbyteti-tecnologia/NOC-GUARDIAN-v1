package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	MasterHost string
	MasterPort string
	MasterUser string
	MasterPass string
	MasterDB   string

	Interval time.Duration

	SNMPCommunity string
	SNMPVersion   string
	SNMPPort      uint16
	SNMPTimeout   time.Duration
	SNMPRetries   int
}

func getenv(k, def string) string {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return def
	}
	return v
}

func atoi(raw string, def int) int {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n <= 0 {
		return def
	}
	return n
}

func MustLoad() Config {
	intervalSec := atoi(getenv("DISCOVERY_INTERVAL_SEC", "600"), 600)
	port := atoi(getenv("SNMP_PORT", "161"), 161)
	timeoutMs := atoi(getenv("SNMP_TIMEOUT_MS", "2000"), 2000)
	retries := atoi(getenv("SNMP_RETRIES", "1"), 1)

	return Config{
		MasterHost:    getenv("MASTER_DB_HOST", "db"),
		MasterPort:    getenv("MASTER_DB_PORT", "5432"),
		MasterUser:    getenv("MASTER_DB_USER", "guardian"),
		MasterPass:    getenv("MASTER_DB_PASS", ""),
		MasterDB:      getenv("MASTER_DB_NAME", "guardian_master"),
		Interval:      time.Duration(intervalSec) * time.Second,
		SNMPCommunity: getenv("SNMP_COMMUNITY", "public"),
		SNMPVersion:   strings.ToLower(getenv("SNMP_VERSION", "2c")),
		SNMPPort:      uint16(port),
		SNMPTimeout:   time.Duration(timeoutMs) * time.Millisecond,
		SNMPRetries:   retries,
	}
}
