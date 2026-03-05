package config

import (
	"os"
	"testing"
)

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
