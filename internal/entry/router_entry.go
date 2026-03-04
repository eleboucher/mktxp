package entry

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/eleboucher/mktxp/internal/config"
	"github.com/eleboucher/mktxp/internal/routeros"
	"github.com/eleboucher/mktxp/internal/utils"
)

// WirelessType indicates the type of wireless stack installed on a RouterOS device.
type WirelessType int

const (
	WirelessTypeNone      WirelessType = 0
	WirelessTypeWireless  WirelessType = 1
	WirelessTypeWiFiWave2 WirelessType = 2
	WirelessTypeWiFi      WirelessType = 3
	WirelessTypeDual      WirelessType = 4
)

// ConnectionState represents the overall connectivity status of a RouterEntry.
type ConnectionState int

const (
	ConnectionStateNotConnected       ConnectionState = 0
	ConnectionStatePartiallyConnected ConnectionState = 1
	ConnectionStateConnected          ConnectionState = 2
)

// dhcpCacheEntry stores a cached DHCP lease record.
type dhcpCacheEntry struct {
	entryType string // "mac_address" or "address"
	record    map[string]string
}

// RouterEntry represents a single configured RouterOS device.
// It holds the API connection, cached DHCP data, and state for one scrape cycle.
type RouterEntry struct {
	RouterName  string
	ConfigEntry *config.RouterConfigEntry
	APIConn     *routeros.Connection
	RouterID    map[string]string
	TimeSpent   map[string]time.Duration

	mu           sync.Mutex
	dhcpCache    map[string]*dhcpCacheEntry // keyed by mac_address or ip_address
	wirelessType WirelessType
	dhcpEntry    *RouterEntry
	capsmanEntry *RouterEntry
}

// New creates a RouterEntry for the named router using the package-level config handler.
func New(name string) *RouterEntry {
	cfg := config.Handler.RouterEntry(name)
	sysCfg := config.Handler.SystemEntry()

	backoff := routeros.BackoffConfig{
		InitialDelay: time.Duration(sysCfg.InitialDelayOnFailure) * time.Second,
		MaxDelay:     time.Duration(sysCfg.MaxDelayOnFailure) * time.Second,
		Divisor:      sysCfg.DelayIncDiv,
	}

	port := cfg.Port
	if port == 0 {
		if cfg.UseSSL {
			port = 8729
		} else {
			port = 8728
		}
	}

	connCfg := routeros.ConnectionConfig{
		RouterName:           name,
		Hostname:             cfg.Hostname,
		Port:                 port,
		Username:             cfg.Username,
		Password:             cfg.Password,
		CredentialsFile:      cfg.CredentialsFile,
		PlaintextLogin:       cfg.PlaintextLogin,
		UseSSL:               cfg.UseSSL,
		NoSSLCertificate:     cfg.NoSSLCertificate,
		SSLCertificateVerify: cfg.SSLCertificateVerify,
		SSLCheckHostname:     cfg.SSLCheckHostname,
		SSLCAFile:            cfg.SSLCAFile,
		SocketTimeout:        time.Duration(sysCfg.SocketTimeout) * time.Second,
		Backoff:              backoff,
	}

	return &RouterEntry{
		RouterName:  name,
		ConfigEntry: cfg,
		APIConn:     routeros.NewConnection(connCfg),
		RouterID: map[string]string{
			"routerboard_name":    name,
			"routerboard_address": cfg.Hostname,
		},
		TimeSpent: make(map[string]time.Duration),
	}
}

// Connect attempts to connect this entry and its child entries (dhcp, capsman).
func (e *RouterEntry) Connect(ctx context.Context) {
	if !e.APIConn.IsConnected() {
		if err := e.APIConn.Connect(ctx); err != nil {
			slog.Error("Connection failed", "name", e.RouterName, "err", err)
		}
	}
	e.mu.Lock()
	dhcp := e.dhcpEntry
	caps := e.capsmanEntry
	e.mu.Unlock()

	if dhcp != nil && !dhcp.APIConn.IsConnected() {
		if err := dhcp.APIConn.Connect(ctx); err != nil {
			slog.Error("DHCP entry connection failed", "name", dhcp.RouterName, "err", err)
		}
	}
	if caps != nil && !caps.APIConn.IsConnected() {
		if err := caps.APIConn.Connect(ctx); err != nil {
			slog.Error("CAPsMAN entry connection failed", "name", caps.RouterName, "err", err)
		}
	}
}

// IsReady returns true if the entry is connected enough to be scraped.
// It triggers a connection attempt if not already connected.
func (e *RouterEntry) IsReady(ctx context.Context) bool {
	state := e.connectionState()
	if state == ConnectionStateNotConnected || state == ConnectionStatePartiallyConnected {
		e.Connect(ctx)
	}
	state = e.connectionState()
	return state == ConnectionStateConnected || state == ConnectionStatePartiallyConnected
}

// IsDone is called after a scrape cycle to optionally close connections and clear caches.
func (e *RouterEntry) IsDone() {
	sysCfg := config.Handler.SystemEntry()

	if !sysCfg.PersistentRouterConnectionPool {
		e.APIConn.Disconnect()
		e.mu.Lock()
		if e.dhcpEntry != nil {
			e.dhcpEntry.APIConn.Disconnect()
		}
		if e.capsmanEntry != nil {
			e.capsmanEntry.APIConn.Disconnect()
		}
		e.mu.Unlock()
	}

	if !sysCfg.PersistentDHCPCache {
		e.mu.Lock()
		e.dhcpCache = nil
		e.mu.Unlock()
	}

	e.mu.Lock()
	e.wirelessType = WirelessTypeNone
	e.mu.Unlock()
}

// DHCPEntry returns the entry used for DHCP resolution (self if no remote_dhcp_entry).
func (e *RouterEntry) DHCPEntry() *RouterEntry {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.dhcpEntry != nil {
		return e.dhcpEntry
	}
	return e
}

// SetDHCPEntry sets a remote entry used for DHCP resolution.
func (e *RouterEntry) SetDHCPEntry(d *RouterEntry) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.dhcpEntry = d
}

// CAPsMANEntry returns the entry used for CAPsMAN info (self if no remote_capsman_entry).
func (e *RouterEntry) CAPsMANEntry() *RouterEntry {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.capsmanEntry != nil {
		return e.capsmanEntry
	}
	return e
}

// SetCAPsMANEntry sets a remote entry used for CAPsMAN info.
func (e *RouterEntry) SetCAPsMANEntry(c *RouterEntry) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.capsmanEntry = c
}

// DHCPRecord looks up a cached DHCP lease by MAC address or IP address.
func (e *RouterEntry) DHCPRecord(key string) map[string]string {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.dhcpCache == nil {
		return nil
	}
	if entry, ok := e.dhcpCache[key]; ok {
		return entry.record
	}
	return nil
}

// DHCPRecords returns all cached DHCP lease records (mac_address entries only).
func (e *RouterEntry) DHCPRecords() []map[string]string {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.dhcpCache == nil {
		return nil
	}
	var out []map[string]string
	for _, v := range e.dhcpCache {
		if v.entryType == "mac_address" {
			out = append(out, v.record)
		}
	}
	return out
}

// SetDHCPRecords populates the DHCP cache from a slice of lease records.
func (e *RouterEntry) SetDHCPRecords(records []map[string]string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.dhcpCache == nil {
		e.dhcpCache = make(map[string]*dhcpCacheEntry)
	}
	for _, r := range records {
		if mac := r["mac_address"]; mac != "" {
			e.dhcpCache[mac] = &dhcpCacheEntry{entryType: "mac_address", record: r}
		}
		if addr := r["address"]; addr != "" {
			rec := make(map[string]string, len(r)+1)
			for k, v := range r {
				rec[k] = v
			}
			rec["type"] = "address"
			e.dhcpCache[addr] = &dhcpCacheEntry{entryType: "address", record: rec}
		}
	}
}

// WirelessType returns the wireless stack type, detecting it on first call.
// This requires a live API connection.
func (e *RouterEntry) WirelessType(ctx context.Context) WirelessType {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.wirelessType != WirelessTypeNone {
		return e.wirelessType
	}
	wt := detectWirelessType(ctx, e)
	e.wirelessType = wt
	return wt
}

// connectionState returns the aggregate connection state of this entry and its children.
func (e *RouterEntry) connectionState() ConnectionState {
	primary := e.APIConn.IsConnected()

	e.mu.Lock()
	dhcp := e.dhcpEntry
	caps := e.capsmanEntry
	e.mu.Unlock()

	dhcpOK := dhcp == nil || dhcp.APIConn.IsConnected()
	capsOK := caps == nil || caps.APIConn.IsConnected()

	if primary && dhcpOK && capsOK {
		return ConnectionStateConnected
	}
	if primary || dhcpOK || capsOK {
		return ConnectionStatePartiallyConnected
	}
	return ConnectionStateNotConnected
}

// detectWirelessType queries the router for installed packages and detects the wireless type.
// Must be called with e.mu held.
func detectWirelessType(ctx context.Context, e *RouterEntry) WirelessType {
	if !e.APIConn.IsConnected() {
		return WirelessTypeWireless // default fallback
	}
	records, err := e.APIConn.Run(ctx, "/system/package/print", "=.proplist=name,disabled")
	if err != nil {
		return WirelessTypeWireless
	}
	for _, r := range records {
		if r["disabled"] == "true" {
			continue
		}
		switch r["name"] {
		case "wifi-qcom", "wifi-qcom-ac":
			return WirelessTypeWiFi
		case "wifiwave2":
			return WirelessTypeWiFiWave2
		case "wireless":
			return WirelessTypeDual
		}
	}
	// Check for RouterOS 7.13+ built-in WiFi CAPsMAN
	verRecords, err := e.APIConn.Run(ctx, "/system/resource/print", "=.proplist=version")
	if err == nil && len(verRecords) > 0 {
		if utils.BuiltinWiFiCAPsMANVersion(verRecords[0]["version"]) {
			return WirelessTypeWiFi
		}
	}
	return WirelessTypeWireless
}
