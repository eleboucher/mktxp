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

// EnvConfigurator handles environment variable configuration overrides using Viper.
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

	e.v = viper.New()
	e.v.SetEnvPrefix(envPrefix)
	e.v.AutomaticEnv()
	e.v.SetEnvKeyReplacer(strings.NewReplacer("_", "_"))

	envVars := os.Environ()
	for _, envVar := range envVars {
		key, value, found := strings.Cut(envVar, "=")
		if !found || !strings.HasPrefix(key, envPrefix) {
			continue
		}

		// Parse MKTXP_ROUTERNAME_FIELD or MKTXP_FIELD
		rest := key[len(envPrefix):]
		parts := strings.SplitN(rest, "_", 2)

		if len(parts) == 2 {
			routerName, field := parts[0], parts[1]
			matchedRouter := e.findRouterByName(routerName, routerNames)
			if matchedRouter == "" && routerName != "*" {
				slog.Debug("Environment variable doesn't match any router",
					"var", key, "router", routerName)
				continue
			}
			e.applyFieldOverride(h, matchedRouter, field, value)
		} else if len(parts) == 1 {
			// System-level env var: MKTXP_FIELD
			e.applySystemFieldOverride(h, parts[0], value)
		}
	}

	return nil
}

// ApplySystemOverrides applies environment variable overrides to system config.
// Pattern: MKTXP_{FIELD} (no router name)
func (e *EnvConfigurator) ApplySystemOverrides(h *ConfigHandler) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	e.v = viper.New()
	e.v.SetEnvPrefix(envPrefix)
	e.v.AutomaticEnv()
	e.v.SetEnvKeyReplacer(strings.NewReplacer("_", "_"))

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

	// Handle special fields
	switch fieldLower {
	case "username", "password", "hostname", "credentials_file",
		"remote_dhcp_entry", "remote_capsman_entry", "interface_name_format":
		*getPtrString(entry, fieldLower) = value
		slog.Info("Applied env override",
			"router", routerName,
			"field", field,
			"value", "***")
		return

	case "custom_labels":
		var labels map[string]string
		if err := json.Unmarshal([]byte(value), &labels); err != nil {
			slog.Warn("Failed to parse custom_labels from env",
				"router", routerName,
				"field", field,
				"error", err)
		} else {
			if entry.CustomLabels == nil {
				entry.CustomLabels = make(map[string]string)
			}
			for k, v := range labels {
				entry.CustomLabels[k] = v
			}
		}
		return

	case "port":
		if port, err := parsePort(value); err == nil {
			*getPtrInt(entry, fieldLower) = port
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
		return
	}

	// Handle boolean fields using field map
	if fi, ok := routerFieldMap[fieldLower]; ok && fi.IsBool {
		if b, err := parseBool(value); err == nil {
			*getPtrBool(entry, fieldLower) = b
			slog.Info("Applied env override",
				"router", routerName,
				"field", field,
				"value", value)
		} else {
			slog.Warn("Invalid boolean value in environment variable",
				"router", routerName,
				"field", field,
				"value", value,
				"error", err)
		}
		return
	}

	slog.Debug("Unknown environment variable field",
		"router", routerName,
		"field", field)
}

// applySystemFieldOverride applies a single field override to system config.
func (e *EnvConfigurator) applySystemFieldOverride(h *ConfigHandler, field, value string) {
	fieldLower := strings.ToLower(field)

	if fi, ok := systemFieldMap[fieldLower]; ok {
		switch {
		case fi.IsString:
			h.sysConfig.Listen = value
			slog.Info("Applied system env override",
				"field", field,
				"value", value)

		case fi.IsInt:
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

		case fi.IsBool:
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
		}
		return
	}

	slog.Debug("Unknown system environment variable field",
		"field", field)
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

// fieldPointer is a helper type for field pointer functions.
type fieldPointer[T any] func(*RouterConfigEntry) *T

// routerFieldPointers maps field names to their pointer functions.
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

var routerIntFieldPointers = map[string]fieldPointer[int]{
	"port": func(e *RouterConfigEntry) *int { return &e.Port },
}

// getPtrBool returns a pointer to a boolean field by name.
func getPtrBool(e *RouterConfigEntry, fieldName string) *bool {
	if fn, ok := routerFieldPointers[fieldName]; ok {
		return fn(e)
	}
	return nil
}

// getPtrString returns a pointer to a string field by name.
func getPtrString(e *RouterConfigEntry, fieldName string) *string {
	if fn, ok := routerStringFieldPointers[fieldName]; ok {
		return fn(e)
	}
	return nil
}

// getPtrInt returns a pointer to an int field by name.
func getPtrInt(e *RouterConfigEntry, fieldName string) *int {
	if fn, ok := routerIntFieldPointers[fieldName]; ok {
		return fn(e)
	}
	return nil
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
