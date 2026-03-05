package config

import (
	"reflect"
)

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

// FieldInfo describes a configurable field.
type FieldInfo struct {
	Name     string
	YAMLTag  string
	IsBool   bool
	IsInt    bool
	IsString bool
	IsSlice  bool
	IsMap    bool
}

// routerFieldMap maps YAML field names to RouterConfigEntry fields.
var routerFieldMap = map[string]FieldInfo{
	"enabled":                {"Enabled", "enabled", true, false, false, false, false},
	"module_only":            {"ModuleOnly", "module_only", true, false, false, false, false},
	"hostname":               {"Hostname", "hostname", false, false, true, false, false},
	"port":                   {"Port", "port", false, true, false, false, false},
	"username":               {"Username", "username", false, false, true, false, false},
	"password":               {"Password", "password", false, false, true, false, false},
	"credentials_file":       {"CredentialsFile", "credentials_file", false, false, true, false, false},
	"custom_labels":          {"CustomLabels", "custom_labels", false, false, false, false, true},
	"use_ssl":                {"UseSSL", "use_ssl", true, false, false, false, false},
	"no_ssl_certificate":     {"NoSSLCertificate", "no_ssl_certificate", true, false, false, false, false},
	"ssl_certificate_verify": {"SSLCertificateVerify", "ssl_certificate_verify", true, false, false, false, false},
	"ssl_check_hostname":     {"SSLCheckHostname", "ssl_check_hostname", true, false, false, false, false},
	"ssl_ca_file":            {"SSLCAFile", "ssl_ca_file", false, false, true, false, false},
	"plaintext_login":        {"PlaintextLogin", "plaintext_login", true, false, false, false, false},
	"health":                 {"Health", "health", true, false, false, false, false},
	"installed_packages":     {"InstalledPackages", "installed_packages", true, false, false, false, false},
	"dhcp":                   {"DHCP", "dhcp", true, false, false, false, false},
	"dhcp_lease":             {"DHCPLease", "dhcp_lease", true, false, false, false, false},
	"connections":            {"Connections", "connections", true, false, false, false, false},
	"connection_stats":       {"ConnectionStats", "connection_stats", true, false, false, false, false},
	"interface":              {"Interface", "interface", true, false, false, false, false},
	"route":                  {"Route", "route", true, false, false, false, false},
	"pool":                   {"Pool", "pool", true, false, false, false, false},
	"firewall":               {"Firewall", "firewall", true, false, false, false, false},
	"address_list":           {"AddressList", "address_list", false, false, false, true, false},
	"neighbor":               {"Neighbor", "neighbor", true, false, false, false, false},
	"dns":                    {"DNS", "dns", true, false, false, false, false},
	"ipv6_route":             {"IPv6Route", "ipv6_route", true, false, false, false, false},
	"ipv6_pool":              {"IPv6Pool", "ipv6_pool", true, false, false, false, false},
	"ipv6_firewall":          {"IPv6Firewall", "ipv6_firewall", true, false, false, false, false},
	"ipv6_neighbor":          {"IPv6Neighbor", "ipv6_neighbor", true, false, false, false, false},
	"ipv6_address_list":      {"IPv6AddressList", "ipv6_address_list", false, false, false, true, false},
	"poe":                    {"POE", "poe", true, false, false, false, false},
	"monitor":                {"Monitor", "monitor", true, false, false, false, false},
	"netwatch":               {"Netwatch", "netwatch", true, false, false, false, false},
	"public_ip":              {"PublicIP", "public_ip", true, false, false, false, false},
	"wireless":               {"Wireless", "wireless", true, false, false, false, false},
	"wireless_clients":       {"WirelessClients", "wireless_clients", true, false, false, false, false},
	"capsman":                {"CAPsMAN", "capsman", true, false, false, false, false},
	"capsman_clients":        {"CAPsMANClients", "capsman_clients", true, false, false, false, false},
	"w60g":                   {"W60G", "w60g", true, false, false, false, false},
	"eoip":                   {"EOIP", "eoip", true, false, false, false, false},
	"gre":                    {"GRE", "gre", true, false, false, false, false},
	"ipip":                   {"IPIP", "ipip", true, false, false, false, false},
	"lte":                    {"LTE", "lte", true, false, false, false, false},
	"ipsec":                  {"IPSec", "ipsec", true, false, false, false, false},
	"switch_port":            {"SwitchPort", "switch_port", true, false, false, false, false},
	"kid_control_assigned":   {"KidControlAssigned", "kid_control_assigned", true, false, false, false, false},
	"kid_control_dynamic":    {"KidControlDynamic", "kid_control_dynamic", true, false, false, false, false},
	"user":                   {"User", "user", true, false, false, false, false},
	"queue":                  {"Queue", "queue", true, false, false, false, false},
	"bfd":                    {"BFD", "bfd", true, false, false, false, false},
	"bgp":                    {"BGP", "bgp", true, false, false, false, false},
	"routing_stats":          {"RoutingStats", "routing_stats", true, false, false, false, false},
	"certificate":            {"Certificate", "certificate", true, false, false, false, false},
	"container":              {"Container", "container", true, false, false, false, false},
	"remote_dhcp_entry":      {"RemoteDHCPEntry", "remote_dhcp_entry", false, false, true, false, false},
	"remote_capsman_entry":   {"RemoteCAPsMANEntry", "remote_capsman_entry", false, false, true, false, false},
	"interface_name_format":  {"InterfaceNameFormat", "interface_name_format", false, false, true, false, false},
	"check_for_updates":      {"CheckForUpdates", "check_for_updates", true, false, false, false, false},
}

// systemFieldMap maps YAML field names to SystemConfig fields.
var systemFieldMap = map[string]FieldInfo{
	"listen":                            {"Listen", "listen", false, false, true, false, false},
	"socket_timeout":                    {"SocketTimeout", "socket_timeout", false, true, false, false, false},
	"initial_delay_on_failure":          {"InitialDelayOnFailure", "initial_delay_on_failure", false, true, false, false, false},
	"max_delay_on_failure":              {"MaxDelayOnFailure", "max_delay_on_failure", false, true, false, false, false},
	"delay_inc_div":                     {"DelayIncDiv", "delay_inc_div", false, true, false, false, false},
	"bandwidth":                         {"Bandwidth", "bandwidth", true, false, false, false, false},
	"bandwidth_test_dns_server":         {"BandwidthTestDNSServer", "bandwidth_test_dns_server", false, false, true, false, false},
	"bandwidth_test_interval":           {"BandwidthTestInterval", "bandwidth_test_interval", false, true, false, false, false},
	"minimal_collect_interval":          {"MinimalCollectInterval", "minimal_collect_interval", false, true, false, false, false},
	"verbose_mode":                      {"VerboseMode", "verbose_mode", true, false, false, false, false},
	"fetch_routers_in_parallel":         {"FetchRoutersInParallel", "fetch_routers_in_parallel", true, false, false, false, false},
	"max_worker_threads":                {"MaxWorkerThreads", "max_worker_threads", false, true, false, false, false},
	"max_scrape_duration":               {"MaxScrapeDuration", "max_scrape_duration", false, true, false, false, false},
	"total_max_scrape_duration":         {"TotalMaxScrapeDuration", "total_max_scrape_duration", false, true, false, false, false},
	"persistent_router_connection_pool": {"PersistentRouterConnectionPool", "persistent_router_connection_pool", true, false, false, false, false},
	"persistent_dhcp_cache":             {"PersistentDHCPCache", "persistent_dhcp_cache", true, false, false, false, false},
	"prometheus_headers_deduplication":  {"PrometheusHeadersDeduplication", "prometheus_headers_deduplication", true, false, false, false, false},
	"probe_connection_pool":             {"ProbeConnectionPool", "probe_connection_pool", true, false, false, false, false},
	"probe_connection_pool_ttl":         {"ProbeConnectionPoolTTL", "probe_connection_pool_ttl", false, true, false, false, false},
	"probe_connection_pool_max_size":    {"ProbeConnectionPoolMaxSize", "probe_connection_pool_max_size", false, true, false, false, false},
}

// hardcodedDefaults returns the baseline RouterConfigEntry with default values.
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

// mergeEntry merges a rawEntry into a RouterConfigEntry, using reflection.
func mergeEntry(base RouterConfigEntry, raw rawEntry) RouterConfigEntry {
	baseVal := reflect.ValueOf(&base).Elem()
	rawVal := reflect.ValueOf(&raw).Elem()

	baseType := baseVal.Type()

	for i := 0; i < baseVal.NumField(); i++ {
		fieldName := baseType.Field(i).Name
		rawField := rawVal.FieldByName(fieldName)

		if !rawField.IsValid() {
			continue
		}

		val := rawField.Interface()
		switch v := val.(type) {
		case *bool:
			if v != nil {
				baseVal.Field(i).SetBool(*v)
			}
		case *string:
			if v != nil {
				baseVal.Field(i).SetString(*v)
			}
		case *int:
			if v != nil {
				baseVal.Field(i).SetInt(int64(*v))
			}
		case map[string]string:
			if v != nil {
				mapVal := reflect.MakeMap(reflect.TypeOf(map[string]string{}))
				for k, vv := range v {
					mapVal.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(vv))
				}
				baseVal.Field(i).Set(mapVal)
			}
		case []string:
			if v != nil {
				slice := reflect.MakeSlice(baseVal.Field(i).Type(), len(v), len(v))
				reflect.Copy(slice, reflect.ValueOf(v))
				baseVal.Field(i).Set(slice)
			}
		}
	}

	return base
}
