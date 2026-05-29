package collector

import (
	"testing"

	"github.com/eleboucher/mktxp/internal/config"
	"github.com/eleboucher/mktxp/internal/entry"
)

func resolveTestEntry(t *testing.T) *entry.RouterEntry {
	t.Helper()
	return &entry.RouterEntry{
		RouterName:  "test",
		ConfigEntry: &config.RouterConfigEntry{},
		RouterID:    map[string]string{"routerboard_name": "test", "routerboard_address": "localhost"},
	}
}

func TestResolveClientDHCP_LookupByMAC(t *testing.T) {
	t.Parallel()

	e := resolveTestEntry(t)
	e.SetDHCPRecords([]map[string]string{
		{"mac_address": "AA:BB:CC:DD:EE:FF", "host_name": "laptop", "address": "192.168.1.50"},
	})

	rec := map[string]string{"mac_address": "AA:BB:CC:DD:EE:FF"}
	resolveClientDHCP(rec, e.DHCPEntry(), "name")

	if rec["dhcp_name"] != "laptop" {
		t.Errorf("dhcp_name = %q, want laptop", rec["dhcp_name"])
	}
	if rec["dhcp_address"] != "192.168.1.50" {
		t.Errorf("dhcp_address = %q, want 192.168.1.50", rec["dhcp_address"])
	}
}

func TestResolveClientDHCP_NoLeaseFallsBackToMAC(t *testing.T) {
	t.Parallel()

	e := resolveTestEntry(t)
	e.SetDHCPRecords([]map[string]string{
		{"mac_address": "AA:BB:CC:DD:EE:01", "host_name": "other", "address": "192.168.1.1"},
	})

	rec := map[string]string{"mac_address": "AA:BB:CC:DD:EE:FF"}
	resolveClientDHCP(rec, e.DHCPEntry(), "name")

	if rec["dhcp_name"] != "AA:BB:CC:DD:EE:FF" {
		t.Errorf("dhcp_name = %q, want MAC fallback", rec["dhcp_name"])
	}
	if rec["dhcp_address"] != "No DHCP Record" {
		t.Errorf("dhcp_address = %q, want 'No DHCP Record'", rec["dhcp_address"])
	}
}

func TestParseSignalStrength(t *testing.T) {
	t.Parallel()

	cases := map[string]float64{
		"-55@6Mbps": -55,
		"-55dBm":    -55,
		"-55.5":     -55.5,
		"-55":       -55,
		"":          0,
		"@6Mbps":    0,
	}
	for in, want := range cases {
		if got := ParseSignalStrength(in); got != want {
			t.Errorf("ParseSignalStrength(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestResolveDHCPName_HostNameFallback(t *testing.T) {
	t.Parallel()

	// No host_name: falls back to MAC.
	got := resolveDHCPName(map[string]string{"mac_address": "AA:BB"}, "name")
	if got != "AA:BB" {
		t.Errorf("resolveDHCPName fallback = %q, want AA:BB", got)
	}

	// comment format uses the comment when present.
	got = resolveDHCPName(map[string]string{"host_name": "h", "comment": "kitchen"}, "comment")
	if got != "kitchen" {
		t.Errorf("resolveDHCPName comment = %q, want kitchen", got)
	}
}
