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
	SNMPCredKey   string
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

func (s *Service) RunTenant(ctx context.Context, tenantID string) error {
	master, err := db.Open(ctx, db.DSN(s.MasterHost, s.MasterPort, s.MasterUser, s.MasterPass, s.MasterDB))
	if err != nil {
		return err
	}
	defer master.Close()

	row := master.QueryRowContext(ctx, `SELECT id::text, db_name FROM tenants WHERE id=$1`, tenantID)
	var t Tenant
	if err := row.Scan(&t.ID, &t.DBName); err != nil {
		return err
	}
	return s.processTenant(ctx, t)
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

	creds, err := listCredentials(ctx, tenantDB, t.ID, s.SNMPCredKey)
	if err != nil {
		return err
	}
	credByID := map[string]SNMPCredential{}
	for _, c := range creds {
		credByID[c.ID] = c
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
		cred, ok := credByID[d.CredID]
		if !ok {
			cred = SNMPCredential{Version: "v2c", Community: s.SNMPCommunity}
		}
		neighbors, err := s.snmpNeighbors(d.IP, cred)
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

func (s *Service) snmpNeighbors(ip string, cred SNMPCredential) ([]Neighbor, error) {
	g, err := buildSNMPClient(ip, s.SNMPPort, s.SNMPTimeout, s.SNMPRetries, cred)
	if err != nil {
		return nil, err
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

func buildSNMPClient(ip string, port uint16, timeout time.Duration, retries int, cred SNMPCredential) (*gosnmp.GoSNMP, error) {
	version := strings.ToLower(strings.TrimSpace(cred.Version))
	if version == "" {
		version = "v2c"
	}
	switch version {
	case "v2", "v2c", "2c":
		return &gosnmp.GoSNMP{
			Target:    ip,
			Port:      port,
			Community: cred.Community,
			Version:   gosnmp.Version2c,
			Timeout:   timeout,
			Retries:   retries,
		}, nil
	case "v3":
		authProto := parseAuthProto(cred.AuthProtocol)
		privProto := parsePrivProto(cred.PrivProtocol)
		params := &gosnmp.UsmSecurityParameters{
			UserName:                 cred.Username,
			AuthenticationProtocol:   authProto,
			AuthenticationPassphrase: cred.AuthPassword,
			PrivacyProtocol:          privProto,
			PrivacyPassphrase:        cred.PrivPassword,
		}
		secLevel := gosnmp.NoAuthNoPriv
		if authProto != gosnmp.NoAuth && privProto != gosnmp.NoPriv {
			secLevel = gosnmp.AuthPriv
		} else if authProto != gosnmp.NoAuth {
			secLevel = gosnmp.AuthNoPriv
		}

		return &gosnmp.GoSNMP{
			Target:             ip,
			Port:               port,
			Version:            gosnmp.Version3,
			SecurityModel:      gosnmp.UserSecurityModel,
			MsgFlags:           secLevel,
			SecurityParameters: params,
			Timeout:            timeout,
			Retries:            retries,
		}, nil
	default:
		return nil, ErrUnsupportedVersion
	}
}

func parseAuthProto(v string) gosnmp.SnmpV3AuthProtocol {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "md5":
		return gosnmp.MD5
	case "sha", "sha1":
		return gosnmp.SHA
	case "sha224":
		return gosnmp.SHA224
	case "sha256":
		return gosnmp.SHA256
	case "sha384":
		return gosnmp.SHA384
	case "sha512":
		return gosnmp.SHA512
	default:
		return gosnmp.NoAuth
	}
}

func parsePrivProto(v string) gosnmp.SnmpV3PrivProtocol {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "des":
		return gosnmp.DES
	case "aes", "aes128":
		return gosnmp.AES
	case "aes192":
		return gosnmp.AES192
	case "aes256":
		return gosnmp.AES256
	default:
		return gosnmp.NoPriv
	}
}

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
	hasIPAddr, err := columnExists(ctx, dbConn, "devices", "ip_address")
	if err != nil {
		return nil, err
	}
	hasCred, err := columnExists(ctx, dbConn, "devices", "snmp_credential_id")
	if err != nil {
		return nil, err
	}

	q := "SELECT id::text, hostname, ip"
	if hasIPAddr {
		q = "SELECT id::text, hostname, COALESCE(NULLIF(ip_address,''), ip) AS ip"
	}
	if hasCred {
		q += ", snmp_credential_id::text"
	} else {
		q += ", ''::text"
	}
	q += " FROM devices WHERE "
	if hasIPAddr {
		q += "COALESCE(NULLIF(ip_address,''), ip) <> ''"
	} else {
		q += "ip <> ''"
	}
	rows, err := dbConn.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Device, 0)
	for rows.Next() {
		var d Device
		if err := rows.Scan(&d.ID, &d.Hostname, &d.IP, &d.CredID); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func listCredentials(ctx context.Context, dbConn *sql.DB, tenantID string, key string) ([]SNMPCredential, error) {
	exists, err := tableExists(ctx, dbConn, "snmp_credentials")
	if err != nil || !exists {
		return []SNMPCredential{}, err
	}
	query := `
SELECT id::text, version, community, username, auth_protocol, auth_password, priv_protocol, priv_password
FROM snmp_credentials
WHERE tenant_id = $1`
	args := []any{tenantID}
	if strings.TrimSpace(key) != "" {
		query = `
SELECT id::text, version,
  pgp_sym_decrypt(decode(community,'base64'), $2) AS community,
  username, auth_protocol,
  pgp_sym_decrypt(decode(auth_password,'base64'), $2) AS auth_password,
  priv_protocol,
  pgp_sym_decrypt(decode(priv_password,'base64'), $2) AS priv_password
FROM snmp_credentials
WHERE tenant_id = $1`
		args = append(args, key)
	}
	rows, err := dbConn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]SNMPCredential, 0)
	for rows.Next() {
		var c SNMPCredential
		if err := rows.Scan(&c.ID, &c.Version, &c.Community, &c.Username, &c.AuthProtocol, &c.AuthPassword, &c.PrivProtocol, &c.PrivPassword); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func tableExists(ctx context.Context, dbConn *sql.DB, table string) (bool, error) {
	row := dbConn.QueryRowContext(ctx, `SELECT to_regclass($1)`, table)
	var reg sql.NullString
	if err := row.Scan(&reg); err != nil {
		return false, err
	}
	return reg.Valid, nil
}

func columnExists(ctx context.Context, dbConn *sql.DB, table, col string) (bool, error) {
	row := dbConn.QueryRowContext(ctx, `
SELECT 1 FROM information_schema.columns
WHERE table_schema='public' AND table_name=$1 AND column_name=$2
LIMIT 1`, table, col)
	var one int
	if err := row.Scan(&one); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
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
