package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnvOnlyConfig(t *testing.T) {
	t.Setenv("MKTXP_ROUTERS", "home, Office_router")
	t.Setenv("MKTXP_HOME_HOSTNAME", "192.168.88.1")
	t.Setenv("MKTXP_HOME_USERNAME", "metrics")
	t.Setenv("MKTXP_HOME_PASSWORD", "secret")
	t.Setenv("MKTXP_HOME_HEALTH", "false")
	t.Setenv("MKTXP_OFFICE_ROUTER_PORT", "8729")
	t.Setenv("MKTXP_LISTEN", ":9090")
	// A file where a config directory would be proves Init skips filesystem access.
	dir := filepath.Join(t.TempDir(), "blocked")
	if err := os.WriteFile(dir, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	h := &ConfigHandler{}
	if err := h.Init(dir); err != nil {
		t.Fatal(err)
	}
	if len(h.RegisteredEntries()) != 2 || h.RouterEntry("Office_router").Port != 8729 {
		t.Fatal("expected both env routers with overrides")
	}
	r := h.RouterEntry("home")
	if r.Hostname != "192.168.88.1" || r.Username != "metrics" || r.Password != "secret" || r.Health || !r.Enabled || r.Port != 8728 {
		t.Fatalf("unexpected router settings: hostname=%q username=%q health=%v enabled=%v port=%d", r.Hostname, r.Username, r.Health, r.Enabled, r.Port)
	}
	if h.SystemEntry().Listen != ":9090" || h.SystemEntry().SocketTimeout != hardcodedSystemDefaults().SocketTimeout {
		t.Fatal("expected system defaults and env overrides")
	}
	if h.ConfigDir() != "" || h.MainConfPath() != "" || h.SysConfPath() != "" {
		t.Fatal("env-only mode should have no config paths")
	}
	t.Setenv("MKTXP_ROUTERS", "home")
	t.Setenv("MKTXP_HOME_HOSTNAME", "192.168.88.2")
	t.Setenv("MKTXP_LISTEN", ":9091")
	if err := h.Reload(); err != nil {
		t.Fatal(err)
	}
	if len(h.RegisteredEntries()) != 1 || h.RouterEntry("home").Hostname != "192.168.88.2" || h.SystemEntry().Listen != ":9091" {
		t.Fatal("reload should re-read env routers and settings")
	}
}

func TestEnvOnlyInvalidRouterNames(t *testing.T) {
	for _, names := range []string{"", " ", "home,", ",home", "home,HOME", "home-router", "home router"} {
		t.Run(names, func(t *testing.T) {
			t.Setenv("MKTXP_ROUTERS", names)
			if err := (&ConfigHandler{}).Init(t.TempDir()); err == nil {
				t.Fatalf("expected invalid MKTXP_ROUTERS %q to fail", names)
			}
		})
	}
}

func setenv(t *testing.T, key, value string) {
	t.Helper()
	if err := os.Setenv(key, value); err != nil {
		t.Fatalf("setenv %s: %v", key, err)
	}
	t.Cleanup(func() { os.Unsetenv(key) }) // nolint:errcheck
}

func newTestHandler(routers map[string]*RouterConfigEntry) *ConfigHandler {
	return &ConfigHandler{entryCache: routers}
}

func newTestSystemHandler(sc *SystemConfig) *ConfigHandler {
	return &ConfigHandler{sysConfig: sc}
}

func TestEnvOverrideUsername(t *testing.T) {
	if err := os.Setenv("MKTXP_TestRouter_USERNAME", "testuser"); err != nil {
		t.Fatalf("failed to set env: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("MKTXP_TestRouter_USERNAME"); err != nil {
			t.Fatalf("failed to unset env: %v", err)
		}
	}()

	h := &ConfigHandler{
		entryCache: map[string]*RouterConfigEntry{
			"TestRouter": {Username: "default"},
		},
	}

	configurator := NewEnvConfigurator()
	if err := configurator.ApplyRouterOverrides(h); err != nil {
		t.Fatalf("ApplyRouterOverrides failed: %v", err)
	}

	if h.entryCache["TestRouter"].Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", h.entryCache["TestRouter"].Username)
	}
}

func TestEnvOverridePassword(t *testing.T) {
	if err := os.Setenv("MKTXP_TestRouter_PASSWORD", "secret123"); err != nil {
		t.Fatalf("failed to set env: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("MKTXP_TestRouter_PASSWORD"); err != nil {
			t.Fatalf("failed to unset env: %v", err)
		}
	}()

	h := &ConfigHandler{
		entryCache: map[string]*RouterConfigEntry{
			"TestRouter": {Password: "default"},
		},
	}

	configurator := NewEnvConfigurator()
	if err := configurator.ApplyRouterOverrides(h); err != nil {
		t.Fatalf("ApplyRouterOverrides failed: %v", err)
	}

	if h.entryCache["TestRouter"].Password != "secret123" {
		t.Errorf("Expected password 'secret123', got '%s'", h.entryCache["TestRouter"].Password)
	}
}

func TestEnvOverridePort(t *testing.T) {
	if err := os.Setenv("MKTXP_TestRouter_PORT", "8729"); err != nil {
		t.Fatalf("failed to set env: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("MKTXP_TestRouter_PORT"); err != nil {
			t.Fatalf("failed to unset env: %v", err)
		}
	}()

	h := &ConfigHandler{
		entryCache: map[string]*RouterConfigEntry{
			"TestRouter": {Port: 8728},
		},
	}

	configurator := NewEnvConfigurator()
	if err := configurator.ApplyRouterOverrides(h); err != nil {
		t.Fatalf("ApplyRouterOverrides failed: %v", err)
	}

	if h.entryCache["TestRouter"].Port != 8729 {
		t.Errorf("Expected port 8729, got %d", h.entryCache["TestRouter"].Port)
	}
}

func TestEnvOverrideHostname(t *testing.T) {
	if err := os.Setenv("MKTXP_TestRouter_HOSTNAME", "192.168.1.100"); err != nil {
		t.Fatalf("failed to set env: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("MKTXP_TestRouter_HOSTNAME"); err != nil {
			t.Fatalf("failed to unset env: %v", err)
		}
	}()

	h := &ConfigHandler{
		entryCache: map[string]*RouterConfigEntry{
			"TestRouter": {Hostname: "localhost"},
		},
	}

	configurator := NewEnvConfigurator()
	if err := configurator.ApplyRouterOverrides(h); err != nil {
		t.Fatalf("ApplyRouterOverrides failed: %v", err)
	}

	if h.entryCache["TestRouter"].Hostname != "192.168.1.100" {
		t.Errorf("Expected hostname '192.168.1.100', got '%s'", h.entryCache["TestRouter"].Hostname)
	}
}

func TestEnvOverrideBoolean(t *testing.T) {
	if err := os.Setenv("MKTXP_TestRouter_HEALTH", "false"); err != nil {
		t.Fatalf("failed to set env: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("MKTXP_TestRouter_HEALTH"); err != nil {
			t.Fatalf("failed to unset env: %v", err)
		}
	}()

	h := &ConfigHandler{
		entryCache: map[string]*RouterConfigEntry{
			"TestRouter": {Health: true},
		},
	}

	configurator := NewEnvConfigurator()
	if err := configurator.ApplyRouterOverrides(h); err != nil {
		t.Fatalf("ApplyRouterOverrides failed: %v", err)
	}

	if h.entryCache["TestRouter"].Health != false {
		t.Errorf("Expected health false, got %v", h.entryCache["TestRouter"].Health)
	}
}

func TestEnvOverrideCustomLabels(t *testing.T) {
	if err := os.Setenv("MKTXP_TestRouter_CUSTOM_LABELS", `{"region":"us-west","team":"network"}`); err != nil {
		t.Fatalf("failed to set env: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("MKTXP_TestRouter_CUSTOM_LABELS"); err != nil {
			t.Fatalf("failed to unset env: %v", err)
		}
	}()

	h := &ConfigHandler{
		entryCache: map[string]*RouterConfigEntry{
			"TestRouter": {CustomLabels: nil},
		},
	}

	configurator := NewEnvConfigurator()
	if err := configurator.ApplyRouterOverrides(h); err != nil {
		t.Fatalf("ApplyRouterOverrides failed: %v", err)
	}

	labels := h.entryCache["TestRouter"].CustomLabels
	if labels["region"] != "us-west" {
		t.Errorf("Expected region 'us-west', got '%s'", labels["region"])
	}
	if labels["team"] != "network" {
		t.Errorf("Expected team 'network', got '%s'", labels["team"])
	}
}

func TestRouterNameCaseInsensitive(t *testing.T) {
	if err := os.Setenv("MKTXP_testrouter_USERNAME", "lowercaseuser"); err != nil {
		t.Fatalf("failed to set env: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("MKTXP_testrouter_USERNAME"); err != nil {
			t.Fatalf("failed to unset env: %v", err)
		}
	}()

	h := &ConfigHandler{
		entryCache: map[string]*RouterConfigEntry{
			"TestRouter": {Username: "default"},
		},
	}

	configurator := NewEnvConfigurator()
	if err := configurator.ApplyRouterOverrides(h); err != nil {
		t.Fatalf("ApplyRouterOverrides failed: %v", err)
	}

	if h.entryCache["TestRouter"].Username != "lowercaseuser" {
		t.Errorf("Expected username 'lowercaseuser', got '%s'", h.entryCache["TestRouter"].Username)
	}
}

func TestUnknownRouterIgnored(t *testing.T) {
	if err := os.Setenv("MKTXP_UnknownRouter_USERNAME", "unknownuser"); err != nil {
		t.Fatalf("failed to set env: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("MKTXP_UnknownRouter_USERNAME"); err != nil {
			t.Fatalf("failed to unset env: %v", err)
		}
	}()

	h := &ConfigHandler{
		entryCache: map[string]*RouterConfigEntry{
			"TestRouter": {Username: "default"},
		},
	}

	configurator := NewEnvConfigurator()
	if err := configurator.ApplyRouterOverrides(h); err != nil {
		t.Fatalf("ApplyRouterOverrides failed: %v", err)
	}

	if h.entryCache["TestRouter"].Username != "default" {
		t.Errorf("Expected username unchanged 'default', got '%s'", h.entryCache["TestRouter"].Username)
	}
}

func TestInvalidPortIgnored(t *testing.T) {
	if err := os.Setenv("MKTXP_TestRouter_PORT", "invalid"); err != nil {
		t.Fatalf("failed to set env: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("MKTXP_TestRouter_PORT"); err != nil {
			t.Fatalf("failed to unset env: %v", err)
		}
	}()

	h := &ConfigHandler{
		entryCache: map[string]*RouterConfigEntry{
			"TestRouter": {Port: 8728},
		},
	}

	configurator := NewEnvConfigurator()
	if err := configurator.ApplyRouterOverrides(h); err != nil {
		t.Fatalf("ApplyRouterOverrides failed: %v", err)
	}

	if h.entryCache["TestRouter"].Port != 8728 {
		t.Errorf("Expected port unchanged 8728, got %d", h.entryCache["TestRouter"].Port)
	}
}

func TestSystemEnvOverrideListen(t *testing.T) {
	if err := os.Setenv("MKTXP_LISTEN", "0.0.0.0:49091"); err != nil {
		t.Fatalf("failed to set env: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("MKTXP_LISTEN"); err != nil {
			t.Fatalf("failed to unset env: %v", err)
		}
	}()

	h := &ConfigHandler{
		sysConfig: &SystemConfig{Listen: "0.0.0.0:49090"},
	}

	configurator := NewEnvConfigurator()
	if err := configurator.ApplySystemOverrides(h); err != nil {
		t.Fatalf("ApplySystemOverrides failed: %v", err)
	}

	if h.sysConfig.Listen != "0.0.0.0:49091" {
		t.Errorf("Expected listen '0.0.0.0:49091', got '%s'", h.sysConfig.Listen)
	}
}

func TestSystemEnvOverrideInt(t *testing.T) {
	if err := os.Setenv("MKTXP_MAX_WORKER_THREADS", "20"); err != nil {
		t.Fatalf("failed to set env: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("MKTXP_MAX_WORKER_THREADS"); err != nil {
			t.Fatalf("failed to unset env: %v", err)
		}
	}()

	h := &ConfigHandler{
		sysConfig: &SystemConfig{MaxWorkerThreads: 5},
	}

	configurator := NewEnvConfigurator()
	if err := configurator.ApplySystemOverrides(h); err != nil {
		t.Fatalf("ApplySystemOverrides failed: %v", err)
	}

	if h.sysConfig.MaxWorkerThreads != 20 {
		t.Errorf("Expected max_worker_threads 20, got %d", h.sysConfig.MaxWorkerThreads)
	}
}

func TestSystemEnvOverrideBool(t *testing.T) {
	if err := os.Setenv("MKTXP_VERBOSE_MODE", "true"); err != nil {
		t.Fatalf("failed to set env: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("MKTXP_VERBOSE_MODE"); err != nil {
			t.Fatalf("failed to unset env: %v", err)
		}
	}()

	h := &ConfigHandler{
		sysConfig: &SystemConfig{VerboseMode: false},
	}

	configurator := NewEnvConfigurator()
	if err := configurator.ApplySystemOverrides(h); err != nil {
		t.Fatalf("ApplySystemOverrides failed: %v", err)
	}

	if h.sysConfig.VerboseMode != true {
		t.Errorf("Expected verbose_mode true, got %v", h.sysConfig.VerboseMode)
	}
}

// TestSystemEnvOverrideBandwidthDNSServer covers the previously-missing
// bandwidth_test_dns_server string field.
func TestSystemEnvOverrideBandwidthDNSServer(t *testing.T) {
	setenv(t, "MKTXP_BANDWIDTH_TEST_DNS_SERVER", "1.1.1.1")

	h := newTestSystemHandler(&SystemConfig{BandwidthTestDNSServer: "8.8.8.8"})
	if err := NewEnvConfigurator().ApplySystemOverrides(h); err != nil {
		t.Fatalf("ApplySystemOverrides: %v", err)
	}
	if h.sysConfig.BandwidthTestDNSServer != "1.1.1.1" {
		t.Errorf("BandwidthTestDNSServer = %q, want 1.1.1.1", h.sysConfig.BandwidthTestDNSServer)
	}
}

// TestSystemEnvOverrideBandwidthFlag covers the previously-missing bandwidth bool field.
func TestSystemEnvOverrideBandwidthFlag(t *testing.T) {
	setenv(t, "MKTXP_BANDWIDTH", "true")

	h := newTestSystemHandler(&SystemConfig{Bandwidth: false})
	if err := NewEnvConfigurator().ApplySystemOverrides(h); err != nil {
		t.Fatalf("ApplySystemOverrides: %v", err)
	}
	if !h.sysConfig.Bandwidth {
		t.Error("Bandwidth should be true after env override")
	}
}

// TestSystemEnvOverridePersistentPoolCanBeFalse verifies that the applySystemDefaults
// bug is fixed: PersistentRouterConnectionPool can now be explicitly set to false.
func TestSystemEnvOverridePersistentPoolCanBeFalse(t *testing.T) {
	setenv(t, "MKTXP_PERSISTENT_ROUTER_CONNECTION_POOL", "false")

	h := newTestSystemHandler(&SystemConfig{PersistentRouterConnectionPool: true})
	if err := NewEnvConfigurator().ApplySystemOverrides(h); err != nil {
		t.Fatalf("ApplySystemOverrides: %v", err)
	}
	if h.sysConfig.PersistentRouterConnectionPool {
		t.Error("PersistentRouterConnectionPool should be false after env override")
	}
}

// TestApplySystemDefaultsDoesNotForcePersistentTrue verifies that applySystemDefaults
// no longer overrides an explicit false for PersistentRouterConnectionPool/PersistentDHCPCache.
func TestApplySystemDefaultsDoesNotForcePersistentTrue(t *testing.T) {
	t.Parallel()

	sc := applySystemDefaults(SystemConfig{
		Listen:                         ":9090",
		PersistentRouterConnectionPool: false,
		PersistentDHCPCache:            false,
	})
	if sc.PersistentRouterConnectionPool {
		t.Error("applySystemDefaults must not force PersistentRouterConnectionPool to true")
	}
	if sc.PersistentDHCPCache {
		t.Error("applySystemDefaults must not force PersistentDHCPCache to true")
	}
}

// TestOutOfRangePortIgnored verifies Viper's zero return for a non-integer is rejected.
func TestOutOfRangePortIgnored(t *testing.T) {
	setenv(t, "MKTXP_TestRouter_PORT", "99999")

	h := newTestHandler(map[string]*RouterConfigEntry{
		"TestRouter": {Port: 8728},
	})
	if err := NewEnvConfigurator().ApplyRouterOverrides(h); err != nil {
		t.Fatalf("ApplyRouterOverrides: %v", err)
	}
	if h.entryCache["TestRouter"].Port != 8728 {
		t.Errorf("Port = %d, want 8728 (out-of-range port must be rejected)", h.entryCache["TestRouter"].Port)
	}
}
