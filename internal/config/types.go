package config

// RouterConfigEntry holds the fully-resolved configuration for a single router entry.
type RouterConfigEntry struct {
	Enabled              bool              `yaml:"enabled"`
	ModuleOnly           bool              `yaml:"module_only"`
	Hostname             string            `yaml:"hostname"`
	Port                 int               `yaml:"port"`
	Username             string            `yaml:"username"`
	Password             string            `yaml:"password"`
	CredentialsFile      string            `yaml:"credentials_file"`
	CustomLabels         map[string]string `yaml:"custom_labels"`
	UseSSL               bool              `yaml:"use_ssl"`
	NoSSLCertificate     bool              `yaml:"no_ssl_certificate"`
	SSLCertificateVerify bool              `yaml:"ssl_certificate_verify"`
	SSLCheckHostname     bool              `yaml:"ssl_check_hostname"`
	SSLCAFile            string            `yaml:"ssl_ca_file"`
	PlaintextLogin       bool              `yaml:"plaintext_login"`

	// Feature flags
	Health              bool     `yaml:"health"`
	InstalledPackages   bool     `yaml:"installed_packages"`
	DHCP                bool     `yaml:"dhcp"`
	DHCPLease           bool     `yaml:"dhcp_lease"`
	Connections         bool     `yaml:"connections"`
	ConnectionStats     bool     `yaml:"connection_stats"`
	Interface           bool     `yaml:"interface"`
	Route               bool     `yaml:"route"`
	Pool                bool     `yaml:"pool"`
	Firewall            bool     `yaml:"firewall"`
	AddressList         []string `yaml:"address_list"`
	Neighbor            bool     `yaml:"neighbor"`
	DNS                 bool     `yaml:"dns"`
	IPv6Route           bool     `yaml:"ipv6_route"`
	IPv6Pool            bool     `yaml:"ipv6_pool"`
	IPv6Firewall        bool     `yaml:"ipv6_firewall"`
	IPv6Neighbor        bool     `yaml:"ipv6_neighbor"`
	IPv6AddressList     []string `yaml:"ipv6_address_list"`
	POE                 bool     `yaml:"poe"`
	Monitor             bool     `yaml:"monitor"`
	Netwatch            bool     `yaml:"netwatch"`
	PublicIP            bool     `yaml:"public_ip"`
	Wireless            bool     `yaml:"wireless"`
	WirelessClients     bool     `yaml:"wireless_clients"`
	CAPsMAN             bool     `yaml:"capsman"`
	CAPsMANClients      bool     `yaml:"capsman_clients"`
	W60G                bool     `yaml:"w60g"`
	EOIP                bool     `yaml:"eoip"`
	GRE                 bool     `yaml:"gre"`
	IPIP                bool     `yaml:"ipip"`
	LTE                 bool     `yaml:"lte"`
	IPSec               bool     `yaml:"ipsec"`
	SwitchPort          bool     `yaml:"switch_port"`
	KidControlAssigned  bool     `yaml:"kid_control_assigned"`
	KidControlDynamic   bool     `yaml:"kid_control_dynamic"`
	User                bool     `yaml:"user"`
	Queue               bool     `yaml:"queue"`
	BFD                 bool     `yaml:"bfd"`
	BGP                 bool     `yaml:"bgp"`
	RoutingStats        bool     `yaml:"routing_stats"`
	Certificate         bool     `yaml:"certificate"`
	Container           bool     `yaml:"container"`
	RemoteDHCPEntry     string   `yaml:"remote_dhcp_entry"`
	RemoteCAPsMANEntry  string   `yaml:"remote_capsman_entry"`
	InterfaceNameFormat string   `yaml:"interface_name_format"`
	CheckForUpdates     bool     `yaml:"check_for_updates"`
}

// SystemConfig holds system-level configuration from _mktxp.yaml.
type SystemConfig struct {
	Listen                         string `yaml:"listen"`
	SocketTimeout                  int    `yaml:"socket_timeout"`
	InitialDelayOnFailure          int    `yaml:"initial_delay_on_failure"`
	MaxDelayOnFailure              int    `yaml:"max_delay_on_failure"`
	DelayIncDiv                    int    `yaml:"delay_inc_div"`
	Bandwidth                      bool   `yaml:"bandwidth"`
	BandwidthTestDNSServer         string `yaml:"bandwidth_test_dns_server"`
	BandwidthTestInterval          int    `yaml:"bandwidth_test_interval"`
	MinimalCollectInterval         int    `yaml:"minimal_collect_interval"`
	VerboseMode                    bool   `yaml:"verbose_mode"`
	FetchRoutersInParallel         bool   `yaml:"fetch_routers_in_parallel"`
	MaxWorkerThreads               int    `yaml:"max_worker_threads"`
	MaxScrapeDuration              int    `yaml:"max_scrape_duration"`
	TotalMaxScrapeDuration         int    `yaml:"total_max_scrape_duration"`
	PersistentRouterConnectionPool bool   `yaml:"persistent_router_connection_pool"`
	PersistentDHCPCache            bool   `yaml:"persistent_dhcp_cache"`
	PrometheusHeadersDeduplication bool   `yaml:"prometheus_headers_deduplication"`
	ProbeConnectionPool            bool   `yaml:"probe_connection_pool"`
	ProbeConnectionPoolTTL         int    `yaml:"probe_connection_pool_ttl"`
	ProbeConnectionPoolMaxSize     int    `yaml:"probe_connection_pool_max_size"`
}

// rawEntry uses pointers for boolean and optional fields so we can detect
// which fields were explicitly set in YAML vs which were absent (nil = use default).
type rawEntry struct {
	Enabled              *bool             `yaml:"enabled"`
	ModuleOnly           *bool             `yaml:"module_only"`
	Hostname             *string           `yaml:"hostname"`
	Port                 *int              `yaml:"port"`
	Username             *string           `yaml:"username"`
	Password             *string           `yaml:"password"`
	CredentialsFile      *string           `yaml:"credentials_file"`
	CustomLabels         map[string]string `yaml:"custom_labels"`
	UseSSL               *bool             `yaml:"use_ssl"`
	NoSSLCertificate     *bool             `yaml:"no_ssl_certificate"`
	SSLCertificateVerify *bool             `yaml:"ssl_certificate_verify"`
	SSLCheckHostname     *bool             `yaml:"ssl_check_hostname"`
	SSLCAFile            *string           `yaml:"ssl_ca_file"`
	PlaintextLogin       *bool             `yaml:"plaintext_login"`
	Health               *bool             `yaml:"health"`
	InstalledPackages    *bool             `yaml:"installed_packages"`
	DHCP                 *bool             `yaml:"dhcp"`
	DHCPLease            *bool             `yaml:"dhcp_lease"`
	Connections          *bool             `yaml:"connections"`
	ConnectionStats      *bool             `yaml:"connection_stats"`
	Interface            *bool             `yaml:"interface"`
	Route                *bool             `yaml:"route"`
	Pool                 *bool             `yaml:"pool"`
	Firewall             *bool             `yaml:"firewall"`
	AddressList          []string          `yaml:"address_list"`
	Neighbor             *bool             `yaml:"neighbor"`
	DNS                  *bool             `yaml:"dns"`
	IPv6Route            *bool             `yaml:"ipv6_route"`
	IPv6Pool             *bool             `yaml:"ipv6_pool"`
	IPv6Firewall         *bool             `yaml:"ipv6_firewall"`
	IPv6Neighbor         *bool             `yaml:"ipv6_neighbor"`
	IPv6AddressList      []string          `yaml:"ipv6_address_list"`
	POE                  *bool             `yaml:"poe"`
	Monitor              *bool             `yaml:"monitor"`
	Netwatch             *bool             `yaml:"netwatch"`
	PublicIP             *bool             `yaml:"public_ip"`
	Wireless             *bool             `yaml:"wireless"`
	WirelessClients      *bool             `yaml:"wireless_clients"`
	CAPsMAN              *bool             `yaml:"capsman"`
	CAPsMANClients       *bool             `yaml:"capsman_clients"`
	W60G                 *bool             `yaml:"w60g"`
	EOIP                 *bool             `yaml:"eoip"`
	GRE                  *bool             `yaml:"gre"`
	IPIP                 *bool             `yaml:"ipip"`
	LTE                  *bool             `yaml:"lte"`
	IPSec                *bool             `yaml:"ipsec"`
	SwitchPort           *bool             `yaml:"switch_port"`
	KidControlAssigned   *bool             `yaml:"kid_control_assigned"`
	KidControlDynamic    *bool             `yaml:"kid_control_dynamic"`
	User                 *bool             `yaml:"user"`
	Queue                *bool             `yaml:"queue"`
	BFD                  *bool             `yaml:"bfd"`
	BGP                  *bool             `yaml:"bgp"`
	RoutingStats         *bool             `yaml:"routing_stats"`
	Certificate          *bool             `yaml:"certificate"`
	Container            *bool             `yaml:"container"`
	RemoteDHCPEntry      *string           `yaml:"remote_dhcp_entry"`
	RemoteCAPsMANEntry   *string           `yaml:"remote_capsman_entry"`
	InterfaceNameFormat  *string           `yaml:"interface_name_format"`
	CheckForUpdates      *bool             `yaml:"check_for_updates"`
}

// hardcodedDefaults returns the baseline RouterConfigEntry with default values,
func hardcodedDefaults() RouterConfigEntry {
	return RouterConfigEntry{
		Enabled:              true,
		ModuleOnly:           false,
		Hostname:             "localhost",
		Port:                 8728,
		Username:             "username",
		Password:             "password",
		CredentialsFile:      "",
		CustomLabels:         nil,
		UseSSL:               false,
		NoSSLCertificate:     false,
		SSLCertificateVerify: false,
		SSLCheckHostname:     true,
		SSLCAFile:            "",
		PlaintextLogin:       true,
		Health:               true,
		InstalledPackages:    true,
		DHCP:                 true,
		DHCPLease:            true,
		Connections:          true,
		ConnectionStats:      false,
		Interface:            true,
		Route:                true,
		Pool:                 true,
		Firewall:             true,
		AddressList:          nil,
		Neighbor:             true,
		DNS:                  false,
		IPv6Route:            false,
		IPv6Pool:             false,
		IPv6Firewall:         false,
		IPv6Neighbor:         false,
		IPv6AddressList:      nil,
		POE:                  true,
		Monitor:              true,
		Netwatch:             true,
		PublicIP:             true,
		Wireless:             true,
		WirelessClients:      true,
		CAPsMAN:              true,
		CAPsMANClients:       true,
		W60G:                 false,
		EOIP:                 false,
		GRE:                  false,
		IPIP:                 false,
		LTE:                  false,
		IPSec:                false,
		SwitchPort:           false,
		KidControlAssigned:   false,
		KidControlDynamic:    false,
		User:                 true,
		Queue:                true,
		BFD:                  false,
		BGP:                  false,
		RoutingStats:         false,
		Certificate:          false,
		Container:            false,
		RemoteDHCPEntry:      "",
		RemoteCAPsMANEntry:   "",
		InterfaceNameFormat:  "name",
		CheckForUpdates:      false,
	}
}

// hardcodedSystemDefaults returns the default SystemConfig values.
func hardcodedSystemDefaults() SystemConfig {
	return SystemConfig{
		Listen:                         "0.0.0.0:49090",
		SocketTimeout:                  2,
		InitialDelayOnFailure:          120,
		MaxDelayOnFailure:              900,
		DelayIncDiv:                    5,
		Bandwidth:                      false,
		BandwidthTestDNSServer:         "8.8.8.8",
		BandwidthTestInterval:          420,
		MinimalCollectInterval:         5,
		VerboseMode:                    false,
		FetchRoutersInParallel:         false,
		MaxWorkerThreads:               5,
		MaxScrapeDuration:              30,
		TotalMaxScrapeDuration:         90,
		PersistentRouterConnectionPool: true,
		PersistentDHCPCache:            true,
		PrometheusHeadersDeduplication: false,
		ProbeConnectionPool:            false,
		ProbeConnectionPoolTTL:         300,
		ProbeConnectionPoolMaxSize:     128,
	}
}

// applySystemDefaults fills zero values in sc with the hardcoded defaults.
func applySystemDefaults(sc SystemConfig) SystemConfig {
	d := hardcodedSystemDefaults()
	if sc.Listen == "" {
		sc.Listen = d.Listen
	}
	if sc.SocketTimeout == 0 {
		sc.SocketTimeout = d.SocketTimeout
	}
	if sc.InitialDelayOnFailure == 0 {
		sc.InitialDelayOnFailure = d.InitialDelayOnFailure
	}
	if sc.MaxDelayOnFailure == 0 {
		sc.MaxDelayOnFailure = d.MaxDelayOnFailure
	}
	if sc.DelayIncDiv == 0 {
		sc.DelayIncDiv = d.DelayIncDiv
	}
	if sc.BandwidthTestDNSServer == "" {
		sc.BandwidthTestDNSServer = d.BandwidthTestDNSServer
	}
	if sc.BandwidthTestInterval == 0 {
		sc.BandwidthTestInterval = d.BandwidthTestInterval
	}
	if sc.MinimalCollectInterval == 0 {
		sc.MinimalCollectInterval = d.MinimalCollectInterval
	}
	if sc.MaxWorkerThreads == 0 {
		sc.MaxWorkerThreads = d.MaxWorkerThreads
	}
	if sc.MaxScrapeDuration == 0 {
		sc.MaxScrapeDuration = d.MaxScrapeDuration
	}
	if sc.TotalMaxScrapeDuration == 0 {
		sc.TotalMaxScrapeDuration = d.TotalMaxScrapeDuration
	}
	if sc.ProbeConnectionPoolTTL == 0 {
		sc.ProbeConnectionPoolTTL = d.ProbeConnectionPoolTTL
	}
	if sc.ProbeConnectionPoolMaxSize == 0 {
		sc.ProbeConnectionPoolMaxSize = d.ProbeConnectionPoolMaxSize
	}
	// Note: boolean fields default to false in Go, which is correct for most system booleans.
	// The ones that default to true are handled explicitly:
	if !sc.PersistentRouterConnectionPool {
		sc.PersistentRouterConnectionPool = d.PersistentRouterConnectionPool
	}
	if !sc.PersistentDHCPCache {
		sc.PersistentDHCPCache = d.PersistentDHCPCache
	}
	return sc
}

// mergeWithDefaults applies the `default` YAML section on top of the hardcoded defaults.
func mergeWithDefaults(def rawEntry) RouterConfigEntry {
	base := hardcodedDefaults()
	return mergeEntry(base, def)
}

// mergeEntry applies override values from raw onto base, returning the merged result.
func mergeEntry(base RouterConfigEntry, raw rawEntry) RouterConfigEntry {
	r := base // copy
	if raw.Enabled != nil {
		r.Enabled = *raw.Enabled
	}
	if raw.ModuleOnly != nil {
		r.ModuleOnly = *raw.ModuleOnly
	}
	if raw.Hostname != nil {
		r.Hostname = *raw.Hostname
	}
	if raw.Port != nil {
		r.Port = *raw.Port
	}
	if raw.Username != nil {
		r.Username = *raw.Username
	}
	if raw.Password != nil {
		r.Password = *raw.Password
	}
	if raw.CredentialsFile != nil {
		r.CredentialsFile = *raw.CredentialsFile
	}
	if raw.CustomLabels != nil {
		r.CustomLabels = raw.CustomLabels
	}
	if raw.UseSSL != nil {
		r.UseSSL = *raw.UseSSL
	}
	if raw.NoSSLCertificate != nil {
		r.NoSSLCertificate = *raw.NoSSLCertificate
	}
	if raw.SSLCertificateVerify != nil {
		r.SSLCertificateVerify = *raw.SSLCertificateVerify
	}
	if raw.SSLCheckHostname != nil {
		r.SSLCheckHostname = *raw.SSLCheckHostname
	}
	if raw.SSLCAFile != nil {
		r.SSLCAFile = *raw.SSLCAFile
	}
	if raw.PlaintextLogin != nil {
		r.PlaintextLogin = *raw.PlaintextLogin
	}
	if raw.Health != nil {
		r.Health = *raw.Health
	}
	if raw.InstalledPackages != nil {
		r.InstalledPackages = *raw.InstalledPackages
	}
	if raw.DHCP != nil {
		r.DHCP = *raw.DHCP
	}
	if raw.DHCPLease != nil {
		r.DHCPLease = *raw.DHCPLease
	}
	if raw.Connections != nil {
		r.Connections = *raw.Connections
	}
	if raw.ConnectionStats != nil {
		r.ConnectionStats = *raw.ConnectionStats
	}
	if raw.Interface != nil {
		r.Interface = *raw.Interface
	}
	if raw.Route != nil {
		r.Route = *raw.Route
	}
	if raw.Pool != nil {
		r.Pool = *raw.Pool
	}
	if raw.Firewall != nil {
		r.Firewall = *raw.Firewall
	}
	if raw.AddressList != nil {
		r.AddressList = raw.AddressList
	}
	if raw.Neighbor != nil {
		r.Neighbor = *raw.Neighbor
	}
	if raw.DNS != nil {
		r.DNS = *raw.DNS
	}
	if raw.IPv6Route != nil {
		r.IPv6Route = *raw.IPv6Route
	}
	if raw.IPv6Pool != nil {
		r.IPv6Pool = *raw.IPv6Pool
	}
	if raw.IPv6Firewall != nil {
		r.IPv6Firewall = *raw.IPv6Firewall
	}
	if raw.IPv6Neighbor != nil {
		r.IPv6Neighbor = *raw.IPv6Neighbor
	}
	if raw.IPv6AddressList != nil {
		r.IPv6AddressList = raw.IPv6AddressList
	}
	if raw.POE != nil {
		r.POE = *raw.POE
	}
	if raw.Monitor != nil {
		r.Monitor = *raw.Monitor
	}
	if raw.Netwatch != nil {
		r.Netwatch = *raw.Netwatch
	}
	if raw.PublicIP != nil {
		r.PublicIP = *raw.PublicIP
	}
	if raw.Wireless != nil {
		r.Wireless = *raw.Wireless
	}
	if raw.WirelessClients != nil {
		r.WirelessClients = *raw.WirelessClients
	}
	if raw.CAPsMAN != nil {
		r.CAPsMAN = *raw.CAPsMAN
	}
	if raw.CAPsMANClients != nil {
		r.CAPsMANClients = *raw.CAPsMANClients
	}
	if raw.W60G != nil {
		r.W60G = *raw.W60G
	}
	if raw.EOIP != nil {
		r.EOIP = *raw.EOIP
	}
	if raw.GRE != nil {
		r.GRE = *raw.GRE
	}
	if raw.IPIP != nil {
		r.IPIP = *raw.IPIP
	}
	if raw.LTE != nil {
		r.LTE = *raw.LTE
	}
	if raw.IPSec != nil {
		r.IPSec = *raw.IPSec
	}
	if raw.SwitchPort != nil {
		r.SwitchPort = *raw.SwitchPort
	}
	if raw.KidControlAssigned != nil {
		r.KidControlAssigned = *raw.KidControlAssigned
	}
	if raw.KidControlDynamic != nil {
		r.KidControlDynamic = *raw.KidControlDynamic
	}
	if raw.User != nil {
		r.User = *raw.User
	}
	if raw.Queue != nil {
		r.Queue = *raw.Queue
	}
	if raw.BFD != nil {
		r.BFD = *raw.BFD
	}
	if raw.BGP != nil {
		r.BGP = *raw.BGP
	}
	if raw.RoutingStats != nil {
		r.RoutingStats = *raw.RoutingStats
	}
	if raw.Certificate != nil {
		r.Certificate = *raw.Certificate
	}
	if raw.Container != nil {
		r.Container = *raw.Container
	}
	if raw.RemoteDHCPEntry != nil {
		r.RemoteDHCPEntry = *raw.RemoteDHCPEntry
	}
	if raw.RemoteCAPsMANEntry != nil {
		r.RemoteCAPsMANEntry = *raw.RemoteCAPsMANEntry
	}
	if raw.InterfaceNameFormat != nil {
		r.InterfaceNameFormat = *raw.InterfaceNameFormat
	}
	if raw.CheckForUpdates != nil {
		r.CheckForUpdates = *raw.CheckForUpdates
	}
	return r
}
