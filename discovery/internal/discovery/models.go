package discovery

type Tenant struct {
	ID     string
	DBName string
}

type Device struct {
	ID       string
	Hostname string
	IP       string
}

type Neighbor struct {
	SysName string
	Port    string
	Proto   string
}
