// snmp.go
// - Usa gosnmp para coletar métricas básicas de dispositivos de rede.
// - Descoberta simples: varredura de hosts e consulta a OIDs comuns.
// - Em produção, utilizar MIBs por vendor/modelo e perfis.

package internal

import (
	"net"
	"time"

	gosnmp "github.com/gosnmp/gosnmp"
	"github.com/google/uuid"
)

type MetricPoint struct {
    Time     string            `json:"time"`
    DeviceID string            `json:"device_id"`
    Metric   string            `json:"metric"`
    Value    float64           `json:"value"`
    Labels   map[string]string `json:"labels"`
}

func ScanSNMP(subnets []string, community string) []MetricPoint {
    var out []MetricPoint
    now := time.Now().UTC().Format(time.RFC3339)
    for _, cidr := range subnets {
        hosts := hostsInCIDR(cidr)
        for _, ip := range hosts {
            cpu, mem := snmpReadBasic(ip, community)
            if cpu >= 0 {
                did := uuid.New().String()
                out = append(out, MetricPoint{Time: now, DeviceID: did, Metric: "cpu", Value: cpu, Labels: map[string]string{"ip": ip}})
            }
            if mem >= 0 {
                did := uuid.New().String()
                out = append(out, MetricPoint{Time: now, DeviceID: did, Metric: "mem", Value: mem, Labels: map[string]string{"ip": ip}})
            }
        }
    }
    return out
}

func snmpReadBasic(ip, community string) (cpu, mem float64) {
    // Consulta OIDs genéricos (exemplo; ajustar conforme MIBs reais)
    g := &gosnmp.GoSNMP{
        Target:    ip,
        Community: community,
        Version:   gosnmp.Version2c,
        Timeout:   2 * time.Second,
        Retries:   1,
    }
    if err := g.Connect(); err != nil {
        return -1, -1
    }
    defer g.Conn.Close()

    // Placeholder OIDs:
    cpuOID := ".1.3.6.1.4.1.9.2.1.58.0" // Exemplo Cisco (ajustar)
    memOID := ".1.3.6.1.4.1.2021.4.6.0" // UCD-SNMP available memory (exemplo)
    oids := []string{cpuOID, memOID}

    pkt, err := g.Get(oids)
    if err != nil || pkt == nil || pkt.Variables == nil {
        return -1, -1
    }
    // Conversões simplificadas:
    var cpuV, memV float64 = -1, -1
    for _, v := range pkt.Variables {
        switch v.Name {
        case cpuOID:
            cpuV = float64(toInt(v.Value))
        case memOID:
            memV = float64(toInt(v.Value))
        }
    }
    return cpuV, memV
}

func hostsInCIDR(cidr string) []string {
    _, ipnet, err := net.ParseCIDR(cidr)
    if err != nil { return nil }
    var ips []string
    for ip := ipnet.IP.Mask(ipnet.Mask); ipnet.Contains(ip); incIP(ip) {
        ips = append(ips, ip.String())
    }
    // remove network/broadcast
    if len(ips) > 2 { return ips[1:len(ips)-1] }
    return nil
}

func incIP(ip net.IP) {
    for j := len(ip)-1; j >= 0; j-- {
        ip[j]++
        if ip[j] > 0 { break }
    }
}

func toInt(v any) int {
    switch t := v.(type) {
    case int: return t
    case uint: return int(t)
    case int64: return int(t)
    case uint64: return int(t)
    case int32: return int(t)
    case uint32: return int(t)
    case byte: return int(t)
    }
    return 0
}
