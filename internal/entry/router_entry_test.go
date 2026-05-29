package entry_test

import (
	"context"
	"sync"
	"sync/atomic"
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

func TestSetDHCPRecords_ReplacesStaleEntries(t *testing.T) {
	t.Parallel()

	e := newTestEntry()
	e.SetDHCPRecords([]map[string]string{
		{"mac_address": "AA:BB:CC:DD:EE:01", "address": "192.168.1.1"},
		{"mac_address": "AA:BB:CC:DD:EE:02", "address": "192.168.1.2"},
	})

	// Second scrape sees only one lease; the stale one must be evicted.
	e.SetDHCPRecords([]map[string]string{
		{"mac_address": "AA:BB:CC:DD:EE:02", "address": "192.168.1.2"},
	})

	if got := e.DHCPRecord("AA:BB:CC:DD:EE:01"); got != nil {
		t.Errorf("stale MAC should be evicted on replace, got %v", got)
	}
	if got := e.DHCPRecord("192.168.1.1"); got != nil {
		t.Errorf("stale IP should be evicted on replace, got %v", got)
	}
	if got := e.DHCPRecord("AA:BB:CC:DD:EE:02"); got == nil {
		t.Error("current lease should still be present")
	}
}

func TestEnsureDHCPRecords_FetchesOnceThenCaches(t *testing.T) {
	t.Parallel()

	e := newTestEntry()
	var calls atomic.Int32
	fetch := func(context.Context, *entry.RouterEntry) ([]map[string]string, error) {
		calls.Add(1)
		return []map[string]string{{"mac_address": "AA:BB:CC:DD:EE:FF", "address": "10.0.0.5"}}, nil
	}

	e.EnsureDHCPRecords(context.Background(), fetch)
	e.EnsureDHCPRecords(context.Background(), fetch)

	if got := calls.Load(); got != 1 {
		t.Errorf("fetch called %d times, want 1 (cache should short-circuit)", got)
	}
	if rec := e.DHCPRecord("AA:BB:CC:DD:EE:FF"); rec == nil {
		t.Error("expected lease populated by EnsureDHCPRecords")
	}
}

func TestEnsureDHCPRecords_ConcurrentFetchesOnce(t *testing.T) {
	t.Parallel()

	e := newTestEntry()
	var calls atomic.Int32
	fetch := func(context.Context, *entry.RouterEntry) ([]map[string]string, error) {
		calls.Add(1)
		return []map[string]string{{"mac_address": "AA:BB:CC:DD:EE:FF", "address": "10.0.0.5"}}, nil
	}

	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			e.EnsureDHCPRecords(context.Background(), fetch)
		})
	}
	wg.Wait()

	if got := calls.Load(); got != 1 {
		t.Errorf("concurrent EnsureDHCPRecords fetched %d times, want 1", got)
	}
}

func TestIsConnected_NewEntry(t *testing.T) {
	t.Parallel()

	e := newTestEntry()
	if e.APIConn.IsConnected() {
		t.Error("new entry should not be connected")
	}
}
