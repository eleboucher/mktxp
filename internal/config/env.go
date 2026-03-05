package config

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

const envPrefix = "MKTXP_"

// EnvConfigurator handles environment variable configuration overrides.
type EnvConfigurator struct {
	v *viper.Viper
}

// NewEnvConfigurator creates a new EnvConfigurator instance.
func NewEnvConfigurator() *EnvConfigurator {
	return &EnvConfigurator{
		v: viper.New(),
	}
}

// ApplyRouterOverrides applies environment variable overrides to router entries.
// Pattern: MKTXP_{ROUTERNAME}_{FIELD}
// Priority: Environment > Credentials File > YAML Config > Defaults
func (e *EnvConfigurator) ApplyRouterOverrides(h *ConfigHandler) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	routerNames := make([]string, 0, len(h.entryCache))
	for name := range h.entryCache {
		routerNames = append(routerNames, name)
	}

	envVars := os.Environ()

	for _, envVar := range envVars {
		key, value, found := strings.Cut(envVar, "=")
		if !found || !strings.HasPrefix(key, envPrefix) {
			continue
		}

		parts := strings.SplitN(key[len(envPrefix):], "_", 2)
		if len(parts) != 2 {
			continue
		}

		routerName, field := parts[0], parts[1]

		matchedRouter := e.findRouterByName(routerName, routerNames)
		if matchedRouter == "" && routerName != "*" {
			slog.Debug("Environment variable doesn't match any router",
				"var", key, "router", routerName)
			continue
		}

		e.applyFieldOverride(h, matchedRouter, field, value)
	}

	return nil
}

// ApplySystemOverrides applies environment variable overrides to system config.
// Pattern: MKTXP_{FIELD} (no router name)
func (e *EnvConfigurator) ApplySystemOverrides(h *ConfigHandler) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	routerNames := make([]string, 0, len(h.entryCache))
	for name := range h.entryCache {
		routerNames = append(routerNames, name)
	}
	_ = routerNames // Used for consistency, though not needed here

	envVars := os.Environ()
	for _, envVar := range envVars {
		key, value, found := strings.Cut(envVar, "=")
		if !found || !strings.HasPrefix(key, envPrefix) {
			continue
		}

		field := key[len(envPrefix):]
		e.applySystemFieldOverride(h, field, value)
	}

	return nil
}

// findRouterByName performs case-insensitive exact matching of router names.
func (e *EnvConfigurator) findRouterByName(input string, known []string) string {
	inputUpper := strings.ToUpper(input)
	for _, name := range known {
		if strings.ToUpper(name) == inputUpper {
			return name
		}
	}
	return ""
}

// applyFieldOverride applies a single field override to a router entry.
func (e *EnvConfigurator) applyFieldOverride(h *ConfigHandler, routerName, field, value string) {
	entry, exists := h.entryCache[routerName]
	if !exists {
		return
	}

	fieldLower := strings.ToLower(field)

	switch fieldLower {
	case "username":
		entry.Username = value
		slog.Info("Applied env override",
			"router", routerName,
			"field", field,
			"value", "***")

	case "password":
		entry.Password = value
		slog.Info("Applied env override",
			"router", routerName,
			"field", field,
			"value", "***")

	case "hostname":
		entry.Hostname = value
		slog.Info("Applied env override",
			"router", routerName,
			"field", field,
			"value", value)

	case "port":
		if port, err := parsePort(value); err == nil {
			entry.Port = port
			slog.Info("Applied env override",
				"router", routerName,
				"field", field,
				"value", value)
		} else {
			slog.Warn("Invalid port value in environment variable",
				"router", routerName,
				"field", field,
				"value", value,
				"error", err)
		}

	case "credentials_file":
		entry.CredentialsFile = value
		slog.Info("Applied env override",
			"router", routerName,
			"field", field,
			"value", "***")

	case "custom_labels":
		if err := e.parseAndSetCustomLabels(entry, value); err != nil {
			slog.Warn("Failed to parse custom_labels from env",
				"router", routerName,
				"field", field,
				"error", err)
		}

	default:
		if e.setBoolField(entry, fieldLower, value) {
			slog.Info("Applied env override",
				"router", routerName,
				"field", field,
				"value", value)
		} else {
			slog.Debug("Unknown environment variable field",
				"router", routerName,
				"field", field)
		}
	}
}

// applySystemFieldOverride applies a single field override to system config.
func (e *EnvConfigurator) applySystemFieldOverride(h *ConfigHandler, field, value string) {
	fieldLower := strings.ToLower(field)

	switch fieldLower {
	case "listen":
		h.sysConfig.Listen = value
		slog.Info("Applied system env override",
			"field", field,
			"value", value)

	case "bandwidth_test_dns_server":
		h.sysConfig.BandwidthTestDNSServer = value
		slog.Info("Applied system env override",
			"field", field,
			"value", value)

	case "socket_timeout", "initial_delay_on_failure",
		"max_delay_on_failure", "delay_inc_div",
		"bandwidth_test_interval", "minimal_collect_interval",
		"max_worker_threads", "max_scrape_duration",
		"total_max_scrape_duration", "probe_connection_pool_ttl",
		"probe_connection_pool_max_size":
		if num, err := parseInt(value); err == nil {
			e.setSystemIntField(h.sysConfig, fieldLower, num)
			slog.Info("Applied system env override",
				"field", field,
				"value", value)
		} else {
			slog.Warn("Invalid numeric value in environment variable",
				"field", field,
				"value", value,
				"error", err)
		}

	case "verbose_mode", "fetch_routers_in_parallel",
		"persistent_router_connection_pool",
		"persistent_dhcp_cache", "prometheus_headers_deduplication",
		"probe_connection_pool":
		if b, err := parseBool(value); err == nil {
			e.setSystemBoolField(h.sysConfig, fieldLower, b)
			slog.Info("Applied system env override",
				"field", field,
				"value", value)
		} else {
			slog.Warn("Invalid boolean value in environment variable",
				"field", field,
				"value", value,
				"error", err)
		}

	default:
		slog.Debug("Unknown system environment variable field",
			"field", field)
	}
}

// parseAndSetCustomLabels parses JSON and sets custom_labels.
func (e *EnvConfigurator) parseAndSetCustomLabels(entry *RouterConfigEntry, value string) error {
	if entry.CustomLabels == nil {
		entry.CustomLabels = make(map[string]string)
	}

	var labels map[string]string
	if err := json.Unmarshal([]byte(value), &labels); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	for k, v := range labels {
		entry.CustomLabels[k] = v
	}

	return nil
}

// setBoolField sets a boolean field on the router entry.
func (e *EnvConfigurator) setBoolField(entry *RouterConfigEntry, field, value string) bool {
	b, err := parseBool(value)
	if err != nil {
		return false
	}

	switch field {
	case "enabled":
		entry.Enabled = b
	case "module_only":
		entry.ModuleOnly = b
	case "use_ssl":
		entry.UseSSL = b
	case "no_ssl_certificate":
		entry.NoSSLCertificate = b
	case "ssl_certificate_verify":
		entry.SSLCertificateVerify = b
	case "ssl_check_hostname":
		entry.SSLCheckHostname = b
	case "plaintext_login":
		entry.PlaintextLogin = b
	case "health":
		entry.Health = b
	case "installed_packages":
		entry.InstalledPackages = b
	case "dhcp":
		entry.DHCP = b
	case "dhcp_lease":
		entry.DHCPLease = b
	case "connections":
		entry.Connections = b
	case "connection_stats":
		entry.ConnectionStats = b
	case "interface":
		entry.Interface = b
	case "route":
		entry.Route = b
	case "pool":
		entry.Pool = b
	case "firewall":
		entry.Firewall = b
	case "neighbor":
		entry.Neighbor = b
	case "dns":
		entry.DNS = b
	case "ipv6_route":
		entry.IPv6Route = b
	case "ipv6_pool":
		entry.IPv6Pool = b
	case "ipv6_firewall":
		entry.IPv6Firewall = b
	case "ipv6_neighbor":
		entry.IPv6Neighbor = b
	case "poe":
		entry.POE = b
	case "monitor":
		entry.Monitor = b
	case "netwatch":
		entry.Netwatch = b
	case "public_ip":
		entry.PublicIP = b
	case "wireless":
		entry.Wireless = b
	case "wireless_clients":
		entry.WirelessClients = b
	case "capsman":
		entry.CAPsMAN = b
	case "capsman_clients":
		entry.CAPsMANClients = b
	case "w60g":
		entry.W60G = b
	case "eoip":
		entry.EOIP = b
	case "gre":
		entry.GRE = b
	case "ipip":
		entry.IPIP = b
	case "lte":
		entry.LTE = b
	case "ipsec":
		entry.IPSec = b
	case "switch_port":
		entry.SwitchPort = b
	case "kid_control_assigned":
		entry.KidControlAssigned = b
	case "kid_control_dynamic":
		entry.KidControlDynamic = b
	case "user":
		entry.User = b
	case "queue":
		entry.Queue = b
	case "bfd":
		entry.BFD = b
	case "bgp":
		entry.BGP = b
	case "routing_stats":
		entry.RoutingStats = b
	case "certificate":
		entry.Certificate = b
	case "container":
		entry.Container = b
	case "check_for_updates":
		entry.CheckForUpdates = b
	default:
		return false
	}

	return true
}

// setSystemIntField sets an integer field on the system config.
func (e *EnvConfigurator) setSystemIntField(sc *SystemConfig, field string, value int) {
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

// setSystemBoolField sets a boolean field on the system config.
func (e *EnvConfigurator) setSystemBoolField(sc *SystemConfig, field string, value bool) {
	switch field {
	case "verbose_mode":
		sc.VerboseMode = value
	case "fetch_routers_in_parallel":
		sc.FetchRoutersInParallel = value
	case "persistent_router_connection_pool":
		sc.PersistentRouterConnectionPool = value
	case "persistent_dhcp_cache":
		sc.PersistentDHCPCache = value
	case "prometheus_headers_deduplication":
		sc.PrometheusHeadersDeduplication = value
	case "probe_connection_pool":
		sc.ProbeConnectionPool = value
	}
}

// parsePort parses a string to an integer port number.
func parsePort(s string) (int, error) {
	var p int
	if _, err := fmt.Sscanf(s, "%d", &p); err != nil {
		return 0, fmt.Errorf("invalid port: %w", err)
	}
	if p < 1 || p > 65535 {
		return 0, fmt.Errorf("port out of range: %d", p)
	}
	return p, nil
}

// parseInt parses a string to an integer.
func parseInt(s string) (int, error) {
	var i int
	if _, err := fmt.Sscanf(s, "%d", &i); err != nil {
		return 0, fmt.Errorf("invalid integer: %w", err)
	}
	return i, nil
}

// parseBool parses a string to a boolean.
func parseBool(s string) (bool, error) {
	return strconv.ParseBool(s)
}
