package entry_test

import (
	"testing"

	"github.com/eleboucher/mktxp/internal/config"
	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/eleboucher/mktxp/internal/routeros"
)

func newTestEntry() *entry.RouterEntry {
	return &entry.RouterEntry{
		RouterName:  "test",
		ConfigEntry: &config.RouterConfigEntry{},
		APIConn:     routeros.NewConnection(routeros.ConnectionConfig{RouterName: "test", Hostname: "localhost"}),
		RouterID: map[string]string{
			"routerboard_name":    "test",
			"routerboard_address": "localhost",
		},
	}
}

func TestDHCPRecord_NilCache(t *testing.T) {
	t.Parallel()

	e := newTestEntry()
	if got := e.DHCPRecord("any-key"); got != nil {
		t.Errorf("expected nil from uninitialized cache, got %v", got)
	}
}

func TestDHCPRecord_NotFound(t *testing.T) {
	t.Parallel()

	e := newTestEntry()
	e.SetDHCPRecords([]map[string]string{
		{"mac_address": "AA:BB:CC:DD:EE:FF", "address": "192.168.1.1"},
	})

	if got := e.DHCPRecord("00:00:00:00:00:00"); got != nil {
		t.Errorf("expected nil for unknown MAC, got %v", got)
	}
	if got := e.DHCPRecord("10.0.0.1"); got != nil {
		t.Errorf("expected nil for unknown IP, got %v", got)
	}
}

func TestSetDHCPRecords_LookupByMAC(t *testing.T) {
	t.Parallel()

	e := newTestEntry()
	e.SetDHCPRecords([]map[string]string{
		{"mac_address": "AA:BB:CC:DD:EE:FF", "address": "192.168.1.100", "host-name": "myhost"},
	})

	got := e.DHCPRecord("AA:BB:CC:DD:EE:FF")
	if got == nil {
		t.Fatal("expected record by MAC, got nil")
	}
	if got["mac_address"] != "AA:BB:CC:DD:EE:FF" {
		t.Errorf("mac_address = %q, want AA:BB:CC:DD:EE:FF", got["mac_address"])
	}
}

func TestSetDHCPRecords_LookupByIP(t *testing.T) {
	t.Parallel()

	e := newTestEntry()
	e.SetDHCPRecords([]map[string]string{
		{"mac_address": "AA:BB:CC:DD:EE:FF", "address": "192.168.1.100"},
	})

	got := e.DHCPRecord("192.168.1.100")
	if got == nil {
		t.Fatal("expected record by IP, got nil")
	}
	if got["type"] != "address" {
		t.Errorf("type = %q, want address", got["type"])
	}
}

func TestSetDHCPRecords_IPEntryDoesNotModifyMAC(t *testing.T) {
	t.Parallel()

	e := newTestEntry()
	e.SetDHCPRecords([]map[string]string{
		{"mac_address": "AA:BB:CC:DD:EE:FF", "address": "192.168.1.100"},
	})

	// The IP lookup record should have type=address added, but the MAC record should not.
	byMAC := e.DHCPRecord("AA:BB:CC:DD:EE:FF")
	if byMAC == nil {
		t.Fatal("MAC record not found")
	}
	if byMAC["type"] == "address" {
		t.Error("MAC record should not have type=address; that field is only on IP entries")
	}
}

func TestDHCPRecords_ReturnsOnlyMACEntries(t *testing.T) {
	t.Parallel()

	e := newTestEntry()
	e.SetDHCPRecords([]map[string]string{
		{"mac_address": "AA:BB:CC:DD:EE:01", "address": "192.168.1.1"},
		{"mac_address": "AA:BB:CC:DD:EE:02", "address": "192.168.1.2"},
		{"mac_address": "AA:BB:CC:DD:EE:03"}, // no address
	})

	all := e.DHCPRecords()
	if len(all) != 3 {
		t.Errorf("expected 3 MAC records, got %d", len(all))
	}
}

func TestDHCPRecords_EmptyCache(t *testing.T) {
	t.Parallel()

	e := newTestEntry()
	all := e.DHCPRecords()
	if all != nil {
		t.Errorf("expected nil from uninitialized cache, got %v", all)
	}
}

func TestSetDHCPRecords_OverwritesExistingEntries(t *testing.T) {
	t.Parallel()

	e := newTestEntry()
	e.SetDHCPRecords([]map[string]string{
		{"mac_address": "AA:BB:CC:DD:EE:FF", "address": "192.168.1.100"},
	})
	// Overwrite with updated address.
	e.SetDHCPRecords([]map[string]string{
		{"mac_address": "AA:BB:CC:DD:EE:FF", "address": "192.168.1.200"},
	})

	got := e.DHCPRecord("AA:BB:CC:DD:EE:FF")
	if got == nil {
		t.Fatal("record should still be accessible")
	}
	if got["address"] != "192.168.1.200" {
		t.Errorf("address = %q, want 192.168.1.200 (updated)", got["address"])
	}
}

func TestIsConnected_NewEntry(t *testing.T) {
	t.Parallel()

	e := newTestEntry()
	if e.APIConn.IsConnected() {
		t.Error("new entry should not be connected")
	}
}
