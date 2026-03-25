package snmp

import (
	"fmt"
	"strings"

	gosnmp "github.com/gosnmp/gosnmp"
)

const (
	lldpRemSysName  = ".1.0.8802.1.1.2.1.4.1.1.9"
	lldpRemPortDesc = ".1.0.8802.1.1.2.1.4.1.1.8"

	cdpCacheDeviceID   = ".1.3.6.1.4.1.9.9.23.1.2.1.1.6"
	cdpCacheDevicePort = ".1.3.6.1.4.1.9.9.23.1.2.1.1.7"
)

type Neighbor struct {
	SysName string
	Port    string
	Proto   string
}

func DiscoverNeighbors(g *gosnmp.GoSNMP) ([]Neighbor, error) {
	lldp, err := walkLLDP(g)
	if err == nil && len(lldp) > 0 {
		return lldp, nil
	}
	cdp, err := walkCDP(g)
	if err == nil && len(cdp) > 0 {
		return cdp, nil
	}
	if err != nil {
		return nil, err
	}
	return []Neighbor{}, nil
}

func walkLLDP(g *gosnmp.GoSNMP) ([]Neighbor, error) {
	names, err := walkAsStringMap(g, lldpRemSysName)
	if err != nil {
		return nil, err
	}
	ports, err := walkAsStringMap(g, lldpRemPortDesc)
	if err != nil {
		return nil, err
	}

	out := make([]Neighbor, 0)
	for idx, name := range names {
		port := ports[idx]
		out = append(out, Neighbor{SysName: name, Port: port, Proto: "lldp"})
	}
	return out, nil
}

func walkCDP(g *gosnmp.GoSNMP) ([]Neighbor, error) {
	names, err := walkAsStringMap(g, cdpCacheDeviceID)
	if err != nil {
		return nil, err
	}
	ports, err := walkAsStringMap(g, cdpCacheDevicePort)
	if err != nil {
		return nil, err
	}

	out := make([]Neighbor, 0)
	for idx, name := range names {
		port := ports[idx]
		out = append(out, Neighbor{SysName: name, Port: port, Proto: "cdp"})
	}
	return out, nil
}

func walkAsStringMap(g *gosnmp.GoSNMP, baseOID string) (map[string]string, error) {
	result := make(map[string]string)
	err := g.BulkWalk(baseOID, func(pdu gosnmp.SnmpPDU) error {
		idx := strings.TrimPrefix(pdu.Name, baseOID+".")
		val := ""
		switch v := pdu.Value.(type) {
		case []byte:
			val = string(v)
		case string:
			val = v
		default:
			val = fmt.Sprint(v)
		}
		result[idx] = strings.TrimSpace(val)
		return nil
	})
	return result, err
}
