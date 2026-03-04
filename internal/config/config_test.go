package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHardcodedDefaults(t *testing.T) {
	t.Parallel()

	d := hardcodedDefaults()

	if !d.Enabled {
		t.Error("default Enabled should be true")
	}
	if d.Hostname != "localhost" {
		t.Errorf("default Hostname = %q, want localhost", d.Hostname)
	}
	if d.Port != 8728 {
		t.Errorf("default Port = %d, want 8728", d.Port)
	}
	if !d.Health {
		t.Error("default Health should be true")
	}
	if !d.Interface {
		t.Error("default Interface should be true")
	}
	if d.InterfaceNameFormat != "name" {
		t.Errorf("default InterfaceNameFormat = %q, want name", d.InterfaceNameFormat)
	}
	if d.BGP {
		t.Error("default BGP should be false")
	}
}

func TestHardcodedSystemDefaults(t *testing.T) {
	t.Parallel()

	d := hardcodedSystemDefaults()

	if d.Listen == "" {
		t.Error("default Listen should not be empty")
	}
	if d.SocketTimeout == 0 {
		t.Error("default SocketTimeout should not be zero")
	}
	if d.InitialDelayOnFailure == 0 {
		t.Error("default InitialDelayOnFailure should not be zero")
	}
	if !d.PersistentRouterConnectionPool {
		t.Error("default PersistentRouterConnectionPool should be true")
	}
	if !d.PersistentDHCPCache {
		t.Error("default PersistentDHCPCache should be true")
	}
}

func TestMergeEntry(t *testing.T) {
	t.Parallel()

	base := hardcodedDefaults()

	t.Run("nil_fields_keep_base", func(t *testing.T) {
		t.Parallel()
		raw := rawEntry{} // all nil
		got := mergeEntry(base, raw)
		if got.Hostname != base.Hostname {
			t.Errorf("Hostname = %q, want %q", got.Hostname, base.Hostname)
		}
		if got.Port != base.Port {
			t.Errorf("Port = %d, want %d", got.Port, base.Port)
		}
		if got.Health != base.Health {
			t.Errorf("Health = %v, want %v", got.Health, base.Health)
		}
	})

	t.Run("set_fields_override_base", func(t *testing.T) {
		t.Parallel()
		host := "192.168.1.1"
		port := 8729
		disabled := false
		raw := rawEntry{
			Hostname: &host,
			Port:     &port,
			Enabled:  &disabled,
		}
		got := mergeEntry(base, raw)
		if got.Hostname != host {
			t.Errorf("Hostname = %q, want %q", got.Hostname, host)
		}
		if got.Port != port {
			t.Errorf("Port = %d, want %d", got.Port, port)
		}
		if got.Enabled {
			t.Error("Enabled should be false")
		}
	})

	t.Run("custom_labels_override", func(t *testing.T) {
		t.Parallel()
		labels := map[string]string{"dc": "london"}
		raw := rawEntry{CustomLabels: labels}
		got := mergeEntry(base, raw)
		if got.CustomLabels["dc"] != "london" {
			t.Errorf("CustomLabels[dc] = %q, want london", got.CustomLabels["dc"])
		}
	})

	t.Run("feature_flags_override", func(t *testing.T) {
		t.Parallel()
		bgp := true
		noHealth := false
		raw := rawEntry{BGP: &bgp, Health: &noHealth}
		got := mergeEntry(base, raw)
		if !got.BGP {
			t.Error("BGP should be true")
		}
		if got.Health {
			t.Error("Health should be false")
		}
	})
}

func TestApplySystemDefaults(t *testing.T) {
	t.Parallel()

	t.Run("zero_value_gets_defaults", func(t *testing.T) {
		t.Parallel()
		sc := applySystemDefaults(SystemConfig{})
		d := hardcodedSystemDefaults()
		if sc.Listen != d.Listen {
			t.Errorf("Listen = %q, want %q", sc.Listen, d.Listen)
		}
		if sc.SocketTimeout != d.SocketTimeout {
			t.Errorf("SocketTimeout = %d, want %d", sc.SocketTimeout, d.SocketTimeout)
		}
		if sc.DelayIncDiv != d.DelayIncDiv {
			t.Errorf("DelayIncDiv = %d, want %d", sc.DelayIncDiv, d.DelayIncDiv)
		}
	})

	t.Run("non_zero_values_preserved", func(t *testing.T) {
		t.Parallel()
		sc := applySystemDefaults(SystemConfig{
			Listen:        ":9090",
			SocketTimeout: 10,
		})
		if sc.Listen != ":9090" {
			t.Errorf("Listen = %q, want :9090", sc.Listen)
		}
		if sc.SocketTimeout != 10 {
			t.Errorf("SocketTimeout = %d, want 10", sc.SocketTimeout)
		}
	})
}

func TestConfigInitFromTempDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	h := &ConfigHandler{}
	if err := h.Init(dir); err != nil {
		t.Fatalf("Init(tempdir) failed: %v", err)
	}

	// Template files should have been created.
	if _, err := os.Stat(filepath.Join(dir, "mktxp.yaml")); err != nil {
		t.Error("mktxp.yaml was not created from template")
	}
	if _, err := os.Stat(filepath.Join(dir, "_mktxp.yaml")); err != nil {
		t.Error("_mktxp.yaml was not created from template")
	}

	sysCfg := h.SystemEntry()
	if sysCfg == nil {
		t.Fatal("SystemEntry should not be nil")
	}

	// Second Init does not overwrite existing files.
	h2 := &ConfigHandler{}
	if err := h2.Init(dir); err != nil {
		t.Fatalf("second Init failed: %v", err)
	}
}

func TestConfigReload(t *testing.T) {
	t.Parallel()

	devDir := filepath.Join("..", "..", ".dev")
	if _, err := os.Stat(devDir); os.IsNotExist(err) {
		t.Skip(".dev/ directory not present")
	}

	h := &ConfigHandler{}
	if err := h.Init(devDir); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if err := h.Reload(); err != nil {
		t.Fatalf("Reload failed: %v", err)
	}

	if h.SystemEntry() == nil {
		t.Error("SystemEntry should not be nil after reload")
	}
}
