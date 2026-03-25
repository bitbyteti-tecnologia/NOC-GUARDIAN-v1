package discovery

type Tenant struct {
	ID     string
	DBName string
}

type Device struct {
	ID       string
	Hostname string
	IP       string
	CredID   string
}

type Neighbor struct {
	SysName string
	Port    string
	Proto   string
}

type SNMPCredential struct {
	ID           string
	Version      string
	Community    string
	Username     string
	AuthProtocol string
	AuthPassword string
	PrivProtocol string
	PrivPassword string
}
