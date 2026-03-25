package discovery

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"strings"
	"time"

	gosnmp "github.com/gosnmp/gosnmp"

	"discovery/internal/db"
	"discovery/internal/snmp"
)

type Service struct {
	MasterHost string
	MasterPort string
	MasterUser string
	MasterPass string
	MasterDB   string

	LogPrefix string

	SNMPCommunity string
	SNMPVersion   string
	SNMPPort      uint16
	SNMPTimeout   time.Duration
	SNMPRetries   int
}

func (s *Service) RunOnce(ctx context.Context) {
	start := time.Now()

	master, err := db.Open(ctx, db.DSN(s.MasterHost, s.MasterPort, s.MasterUser, s.MasterPass, s.MasterDB))
	if err != nil {
		log.Printf("%sERROR master db: %v", s.LogPrefix, err)
		return
	}
	defer master.Close()

	tenants, err := listTenants(ctx, master)
	if err != nil {
		log.Printf("%sERROR list tenants: %v", s.LogPrefix, err)
		return
	}
	log.Printf("%sdiscovery start tenants=%d", s.LogPrefix, len(tenants))

	for _, t := range tenants {
		if err := s.processTenant(ctx, t); err != nil {
			log.Printf("%sERROR tenant=%s: %v", s.LogPrefix, t.ID, err)
		}
	}

	log.Printf("%sdiscovery done in %s", s.LogPrefix, time.Since(start))
}

func (s *Service) processTenant(ctx context.Context, t Tenant) error {
	tenantDB, err := db.Open(ctx, db.DSN(s.MasterHost, s.MasterPort, s.MasterUser, s.MasterPass, t.DBName))
	if err != nil {
		return err
	}
	defer tenantDB.Close()

	if err := ensureRelationshipTable(ctx, tenantDB, t.ID); err != nil {
		return err
	}

	devices, err := listDevices(ctx, tenantDB)
	if err != nil {
		return err
	}

	if len(devices) == 0 {
		seeded, err := seedDevicesFromMetrics(ctx, tenantDB)
		if err != nil {
			return err
		}
		log.Printf("%stenant=%s devices=0 seeded=%d", s.LogPrefix, t.ID, seeded)
		devices, err = listDevices(ctx, tenantDB)
		if err != nil {
			return err
		}
		if len(devices) == 0 {
			return nil
		}
	}

	mapByHost := make(map[string]Device)
	mapByIP := make(map[string]Device)
	for _, d := range devices {
		mapByHost[strings.ToLower(d.Hostname)] = d
		if d.IP != "" {
			mapByIP[d.IP] = d
		}
	}

	relationships := 0
	for _, d := range devices {
		if d.IP == "" {
			continue
		}
		neighbors, err := s.snmpNeighbors(d.IP)
		if err != nil {
			log.Printf("%stenant=%s device=%s snmp error: %v", s.LogPrefix, t.ID, d.Hostname, err)
			continue
		}
		if len(neighbors) == 0 {
			continue
		}
		for _, n := range neighbors {
			if strings.TrimSpace(n.SysName) == "" {
				continue
			}
			target, ok := mapByHost[strings.ToLower(n.SysName)]
			if !ok {
				target = mapByIP[n.SysName]
			}
			if target.ID == "" {
				created, err := getOrCreateDevice(ctx, tenantDB, n.SysName)
				if err != nil {
					log.Printf("%stenant=%s neighbor=%s create device error: %v", s.LogPrefix, t.ID, n.SysName, err)
					continue
				}
				target = created
				mapByHost[strings.ToLower(created.Hostname)] = created
			}
			if d.ID == target.ID {
				continue
			}
			if err := insertRelationship(ctx, tenantDB, t.ID, d.ID, target.ID, n.Proto); err != nil {
				return err
			}
			relationships++
		}
	}

	log.Printf("%stenant=%s devices=%d relationships=%d", s.LogPrefix, t.ID, len(devices), relationships)
	return nil
}

func (s *Service) snmpNeighbors(ip string) ([]Neighbor, error) {
	if s.SNMPVersion != "2c" && s.SNMPVersion != "2" {
		return nil, ErrUnsupportedVersion
	}

	g := &gosnmp.GoSNMP{
		Target:    ip,
		Port:      s.SNMPPort,
		Community: s.SNMPCommunity,
		Version:   gosnmp.Version2c,
		Timeout:   s.SNMPTimeout,
		Retries:   s.SNMPRetries,
	}

	if err := g.Connect(); err != nil {
		return nil, err
	}
	defer g.Conn.Close()

	neighbors, err := snmp.DiscoverNeighbors(g)
	if err != nil {
		return nil, err
	}

	out := make([]Neighbor, 0, len(neighbors))
	for _, n := range neighbors {
		out = append(out, Neighbor{SysName: strings.TrimSpace(n.SysName), Port: n.Port, Proto: n.Proto})
	}
	return out, nil
}

var ErrUnsupportedVersion = errors.New("snmp version not supported")

func listTenants(ctx context.Context, master *sql.DB) ([]Tenant, error) {
	rows, err := master.QueryContext(ctx, `SELECT id::text, db_name FROM tenants`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Tenant, 0)
	for rows.Next() {
		var t Tenant
		if err := rows.Scan(&t.ID, &t.DBName); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func listDevices(ctx context.Context, dbConn *sql.DB) ([]Device, error) {
	rows, err := dbConn.QueryContext(ctx, `SELECT id::text, hostname, ip FROM devices WHERE ip <> ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Device, 0)
	for rows.Next() {
		var d Device
		if err := rows.Scan(&d.ID, &d.Hostname, &d.IP); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func getOrCreateDevice(ctx context.Context, dbConn *sql.DB, hostname string) (Device, error) {
	hostname = strings.TrimSpace(hostname)
	if hostname == "" {
		return Device{}, errors.New("hostname vazio")
	}
	row := dbConn.QueryRowContext(ctx, `
INSERT INTO devices (hostname, ip, type, os)
VALUES ($1, '', 'network', 'unknown')
ON CONFLICT (hostname) DO UPDATE SET hostname = EXCLUDED.hostname
RETURNING id::text, hostname, ip`, hostname)
	var d Device
	if err := row.Scan(&d.ID, &d.Hostname, &d.IP); err != nil {
		return Device{}, err
	}
	return d, nil
}

func ensureRelationshipTable(ctx context.Context, dbConn *sql.DB, tenantID string) error {
	_, err := dbConn.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS device_relationships (
  id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
  tenant_id UUID,
  parent_device_id UUID NOT NULL,
  child_device_id UUID NOT NULL,
  relation_type TEXT NOT NULL DEFAULT 'uplink',
  discovered_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE device_relationships
  ADD COLUMN IF NOT EXISTS tenant_id UUID,
  ADD COLUMN IF NOT EXISTS relation_type TEXT NOT NULL DEFAULT 'uplink',
  ADD COLUMN IF NOT EXISTS discovered_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

UPDATE device_relationships
SET tenant_id = $1
WHERE tenant_id IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS device_relationships_unique
  ON device_relationships (tenant_id, parent_device_id, child_device_id);

CREATE INDEX IF NOT EXISTS idx_device_relationships_parent
  ON device_relationships (tenant_id, parent_device_id);

CREATE INDEX IF NOT EXISTS idx_device_relationships_child
  ON device_relationships (tenant_id, child_device_id);
`, tenantID)
	return err
}

func insertRelationship(ctx context.Context, dbConn *sql.DB, tenantID, parentID, childID, relation string) error {
	_, err := dbConn.ExecContext(ctx, `
INSERT INTO device_relationships (tenant_id, parent_device_id, child_device_id, relation_type)
VALUES ($1, $2, $3, $4)
ON CONFLICT (tenant_id, parent_device_id, child_device_id) DO NOTHING`, tenantID, parentID, childID, relation)
	return err
}

func seedDevicesFromMetrics(ctx context.Context, dbConn *sql.DB) (int, error) {
	// cria devices a partir de labels das métricas (quando tabela devices está vazia)
	res, err := dbConn.ExecContext(ctx, `
WITH src AS (
  SELECT DISTINCT
    NULLIF(labels->>'hostname','') AS hostname,
    NULLIF(COALESCE(labels->>'ip', labels->>'host_ip', labels->>'ip_address'), '') AS ip,
    NULLIF(labels->>'os','') AS os
  FROM metrics
  WHERE labels ? 'hostname'
)
INSERT INTO devices (hostname, ip, type, os)
SELECT hostname, COALESCE(ip,''), 'server', COALESCE(os, 'linux')
FROM src
WHERE hostname IS NOT NULL
ON CONFLICT (hostname) DO NOTHING;
`)
	if err != nil {
		return 0, err
	}
	rows, _ := res.RowsAffected()
	return int(rows), nil
}
