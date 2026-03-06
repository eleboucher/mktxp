package config

import (
	"encoding/json"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/viper"
)

const envPrefix = "MKTXP"

type EnvConfigurator struct{}

func NewEnvConfigurator() *EnvConfigurator {
	return &EnvConfigurator{}
}

// ApplyRouterOverrides applies MKTXP_{ROUTERNAME}_{FIELD} env vars to each router entry.
// Router name matching is case-insensitive.
func (e *EnvConfigurator) ApplyRouterOverrides(h *ConfigHandler) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Build a case-normalised lookup (UPPERCASE_KEY → value) so that router
	// names in env var keys are matched case-insensitively on Linux.
	envMap := buildEnvMap()

	for routerName, entry := range h.entryCache {
		prefix := envPrefix + "_" + strings.ToUpper(routerName) + "_"
		rv := viperFromEnvMap(prefix, envMap)
		applyRouterOverridesFromViper(rv, routerName, entry)
	}
	return nil
}

// ApplySystemOverrides applies MKTXP_{FIELD} env vars to system config.
func (e *EnvConfigurator) ApplySystemOverrides(h *ConfigHandler) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	v := viper.New()
	v.SetEnvPrefix(envPrefix)
	v.AutomaticEnv()
	for fieldName := range systemFieldMap {
		_ = v.BindEnv(fieldName)
	}

	for fieldName, fi := range systemFieldMap {
		if !v.IsSet(fieldName) {
			continue
		}
		switch {
		case fi.IsString:
			setSystemStringField(h.sysConfig, fieldName, v.GetString(fieldName))
		case fi.IsInt:
			setSystemIntField(h.sysConfig, fieldName, v.GetInt(fieldName))
		case fi.IsBool:
			setSystemBoolField(h.sysConfig, fieldName, v.GetBool(fieldName))
		}
		slog.Info("Applied system env override", "field", fieldName)
	}
	return nil
}

// buildEnvMap scans os.Environ() and returns an uppercase-keyed map of all
// env vars that start with the MKTXP_ prefix.
func buildEnvMap() map[string]string {
	pfx := envPrefix + "_"
	m := make(map[string]string)
	for _, env := range os.Environ() {
		key, value, found := strings.Cut(env, "=")
		if found && strings.HasPrefix(strings.ToUpper(key), pfx) {
			m[strings.ToUpper(key)] = value
		}
	}
	return m
}

// viperFromEnvMap creates a Viper instance pre-populated with all entries in
// envMap whose key starts with prefix. Keys are stripped of the prefix and
// lowercased so callers use plain field names (e.g. "hostname").
func viperFromEnvMap(prefix string, envMap map[string]string) *viper.Viper {
	v := viper.New()
	for key, val := range envMap {
		if strings.HasPrefix(key, prefix) {
			fieldName := strings.ToLower(key[len(prefix):])
			v.Set(fieldName, val)
		}
	}
	return v
}

func applyRouterOverridesFromViper(v *viper.Viper, routerName string, entry *RouterConfigEntry) {
	for fieldName, getPtr := range routerFieldPointers {
		if !v.IsSet(fieldName) {
			continue
		}
		*getPtr(entry) = v.GetBool(fieldName)
		slog.Info("Applied env override", "router", routerName, "field", fieldName)
	}

	for fieldName, getPtr := range routerStringFieldPointers {
		if !v.IsSet(fieldName) {
			continue
		}
		*getPtr(entry) = v.GetString(fieldName)
		slog.Info("Applied env override", "router", routerName, "field", fieldName, "value", "***")
	}

	if v.IsSet("port") {
		port := v.GetInt("port")
		if port >= 1 && port <= 65535 {
			entry.Port = port
			slog.Info("Applied env override", "router", routerName, "field", "port", "value", port)
		} else {
			slog.Warn("Port out of range in environment variable", "router", routerName, "port", port)
		}
	}

	if v.IsSet("custom_labels") {
		var labels map[string]string
		if err := json.Unmarshal([]byte(v.GetString("custom_labels")), &labels); err != nil {
			slog.Warn("Failed to parse custom_labels from env", "router", routerName, "error", err)
		} else {
			if entry.CustomLabels == nil {
				entry.CustomLabels = make(map[string]string)
			}
			for k, val := range labels {
				entry.CustomLabels[k] = val
			}
			slog.Info("Applied env override", "router", routerName, "field", "custom_labels")
		}
	}
}

func setSystemStringField(sc *SystemConfig, field, value string) {
	switch field {
	case "listen":
		sc.Listen = value
	case "bandwidth_test_dns_server":
		sc.BandwidthTestDNSServer = value
	}
}

func setSystemIntField(sc *SystemConfig, field string, value int) {
	switch field {
	case "socket_timeout":
		sc.SocketTimeout = value
	case "initial_delay_on_failure":
		sc.InitialDelayOnFailure = value
	case "max_delay_on_failure":
		sc.MaxDelayOnFailure = value
	case "delay_inc_div":
		sc.DelayIncDiv = value
	case "bandwidth_test_interval":
		sc.BandwidthTestInterval = value
	case "minimal_collect_interval":
		sc.MinimalCollectInterval = value
	case "max_worker_threads":
		sc.MaxWorkerThreads = value
	case "max_scrape_duration":
		sc.MaxScrapeDuration = value
	case "total_max_scrape_duration":
		sc.TotalMaxScrapeDuration = value
	case "probe_connection_pool_ttl":
		sc.ProbeConnectionPoolTTL = value
	case "probe_connection_pool_max_size":
		sc.ProbeConnectionPoolMaxSize = value
	}
}

func setSystemBoolField(sc *SystemConfig, field string, value bool) {
	switch field {
	case "bandwidth":
		sc.Bandwidth = value
	case "verbose_mode":
		sc.VerboseMode = value
	case "fetch_routers_in_parallel":
		sc.FetchRoutersInParallel = value
	case "persistent_router_connection_pool":
		sc.PersistentRouterConnectionPool = value
	case "persistent_dhcp_cache":
		sc.PersistentDHCPCache = value
	case "probe_connection_pool":
		sc.ProbeConnectionPool = value
	}
}

type fieldPointer[T any] func(*RouterConfigEntry) *T

var routerFieldPointers = map[string]fieldPointer[bool]{
	"enabled":                func(e *RouterConfigEntry) *bool { return &e.Enabled },
	"module_only":            func(e *RouterConfigEntry) *bool { return &e.ModuleOnly },
	"use_ssl":                func(e *RouterConfigEntry) *bool { return &e.UseSSL },
	"no_ssl_certificate":     func(e *RouterConfigEntry) *bool { return &e.NoSSLCertificate },
	"ssl_certificate_verify": func(e *RouterConfigEntry) *bool { return &e.SSLCertificateVerify },
	"ssl_check_hostname":     func(e *RouterConfigEntry) *bool { return &e.SSLCheckHostname },
	"plaintext_login":        func(e *RouterConfigEntry) *bool { return &e.PlaintextLogin },
	"health":                 func(e *RouterConfigEntry) *bool { return &e.Health },
	"installed_packages":     func(e *RouterConfigEntry) *bool { return &e.InstalledPackages },
	"dhcp":                   func(e *RouterConfigEntry) *bool { return &e.DHCP },
	"dhcp_lease":             func(e *RouterConfigEntry) *bool { return &e.DHCPLease },
	"connections":            func(e *RouterConfigEntry) *bool { return &e.Connections },
	"connection_stats":       func(e *RouterConfigEntry) *bool { return &e.ConnectionStats },
	"interface":              func(e *RouterConfigEntry) *bool { return &e.Interface },
	"route":                  func(e *RouterConfigEntry) *bool { return &e.Route },
	"pool":                   func(e *RouterConfigEntry) *bool { return &e.Pool },
	"firewall":               func(e *RouterConfigEntry) *bool { return &e.Firewall },
	"neighbor":               func(e *RouterConfigEntry) *bool { return &e.Neighbor },
	"dns":                    func(e *RouterConfigEntry) *bool { return &e.DNS },
	"ipv6_route":             func(e *RouterConfigEntry) *bool { return &e.IPv6Route },
	"ipv6_pool":              func(e *RouterConfigEntry) *bool { return &e.IPv6Pool },
	"ipv6_firewall":          func(e *RouterConfigEntry) *bool { return &e.IPv6Firewall },
	"ipv6_neighbor":          func(e *RouterConfigEntry) *bool { return &e.IPv6Neighbor },
	"poe":                    func(e *RouterConfigEntry) *bool { return &e.POE },
	"monitor":                func(e *RouterConfigEntry) *bool { return &e.Monitor },
	"netwatch":               func(e *RouterConfigEntry) *bool { return &e.Netwatch },
	"public_ip":              func(e *RouterConfigEntry) *bool { return &e.PublicIP },
	"wireless":               func(e *RouterConfigEntry) *bool { return &e.Wireless },
	"wireless_clients":       func(e *RouterConfigEntry) *bool { return &e.WirelessClients },
	"capsman":                func(e *RouterConfigEntry) *bool { return &e.CAPsMAN },
	"capsman_clients":        func(e *RouterConfigEntry) *bool { return &e.CAPsMANClients },
	"w60g":                   func(e *RouterConfigEntry) *bool { return &e.W60G },
	"eoip":                   func(e *RouterConfigEntry) *bool { return &e.EOIP },
	"gre":                    func(e *RouterConfigEntry) *bool { return &e.GRE },
	"ipip":                   func(e *RouterConfigEntry) *bool { return &e.IPIP },
	"lte":                    func(e *RouterConfigEntry) *bool { return &e.LTE },
	"ipsec":                  func(e *RouterConfigEntry) *bool { return &e.IPSec },
	"switch_port":            func(e *RouterConfigEntry) *bool { return &e.SwitchPort },
	"kid_control_assigned":   func(e *RouterConfigEntry) *bool { return &e.KidControlAssigned },
	"kid_control_dynamic":    func(e *RouterConfigEntry) *bool { return &e.KidControlDynamic },
	"user":                   func(e *RouterConfigEntry) *bool { return &e.User },
	"queue":                  func(e *RouterConfigEntry) *bool { return &e.Queue },
	"bfd":                    func(e *RouterConfigEntry) *bool { return &e.BFD },
	"bgp":                    func(e *RouterConfigEntry) *bool { return &e.BGP },
	"routing_stats":          func(e *RouterConfigEntry) *bool { return &e.RoutingStats },
	"certificate":            func(e *RouterConfigEntry) *bool { return &e.Certificate },
	"container":              func(e *RouterConfigEntry) *bool { return &e.Container },
	"check_for_updates":      func(e *RouterConfigEntry) *bool { return &e.CheckForUpdates },
}

var routerStringFieldPointers = map[string]fieldPointer[string]{
	"hostname":              func(e *RouterConfigEntry) *string { return &e.Hostname },
	"username":              func(e *RouterConfigEntry) *string { return &e.Username },
	"password":              func(e *RouterConfigEntry) *string { return &e.Password },
	"credentials_file":      func(e *RouterConfigEntry) *string { return &e.CredentialsFile },
	"remote_dhcp_entry":     func(e *RouterConfigEntry) *string { return &e.RemoteDHCPEntry },
	"remote_capsman_entry":  func(e *RouterConfigEntry) *string { return &e.RemoteCAPsMANEntry },
	"interface_name_format": func(e *RouterConfigEntry) *string { return &e.InterfaceNameFormat },
}
