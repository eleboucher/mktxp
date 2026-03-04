package collector_test

import (
	"context"
	"testing"

	dto "github.com/prometheus/client_model/go"

	"github.com/eleboucher/mktxp/internal/collector"
	"github.com/eleboucher/mktxp/internal/config"
	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/eleboucher/mktxp/internal/routeros"
	"github.com/prometheus/client_golang/prometheus"
)

func TestNormalizeKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"rx-byte", "rx_byte"},
		{"cpu.load", "cpu_load"},
		{"a-b.c-d", "a_b_c_d"},
		{"already_normalized", "already_normalized"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			if got := collector.NormalizeKey(tt.input); got != tt.want {
				t.Errorf("NormalizeKey(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestTrimRecord(t *testing.T) {
	t.Parallel()

	t.Run("nil_keys_returns_all_normalized", func(t *testing.T) {
		t.Parallel()
		record := map[string]string{"rx-byte": "100", "tx-byte": "200"}
		got := collector.TrimRecord(record, nil)
		if got["rx_byte"] != "100" || got["tx_byte"] != "200" || len(got) != 2 {
			t.Errorf("unexpected result: %v", got)
		}
	})

	t.Run("empty_keys_returns_all_normalized", func(t *testing.T) {
		t.Parallel()
		record := map[string]string{"name": "ether1"}
		got := collector.TrimRecord(record, []string{})
		if got["name"] != "ether1" {
			t.Errorf("unexpected result: %v", got)
		}
	})

	t.Run("filters_to_wanted_keys", func(t *testing.T) {
		t.Parallel()
		record := map[string]string{"rx-byte": "100", "tx-byte": "200", "name": "ether1"}
		got := collector.TrimRecord(record, []string{"rx-byte", "name"})
		if len(got) != 2 {
			t.Fatalf("expected 2 keys, got %d: %v", len(got), got)
		}
		if got["rx_byte"] != "100" {
			t.Errorf("rx_byte = %q, want 100", got["rx_byte"])
		}
		if got["name"] != "ether1" {
			t.Errorf("name = %q, want ether1", got["name"])
		}
		if _, ok := got["tx_byte"]; ok {
			t.Error("tx_byte should not be present")
		}
	})

	t.Run("missing_wanted_key_absent_from_result", func(t *testing.T) {
		t.Parallel()
		record := map[string]string{"name": "ether1"}
		got := collector.TrimRecord(record, []string{"name", "missing-key"})
		if _, ok := got["missing_key"]; ok {
			t.Error("missing_key should not appear in result")
		}
	})

	t.Run("normalizes_wanted_keys", func(t *testing.T) {
		t.Parallel()
		record := map[string]string{"rx-byte": "42"}
		got := collector.TrimRecord(record, []string{"rx-byte"})
		if got["rx_byte"] != "42" {
			t.Errorf("key not normalized: %v", got)
		}
	})
}

func TestParseFloat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  float64
	}{
		{"", 0},
		{"0", 0},
		{"42", 42},
		{"3.14", 3.14},
		{"-1", -1},
		{"invalid", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input+"_"+tt.input, func(t *testing.T) {
			t.Parallel()
			if got := collector.ParseFloat(tt.input); got != tt.want {
				t.Errorf("ParseFloat(%q) = %f, want %f", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseBool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  float64
	}{
		{"true", 1},
		{"True", 1},
		{"TRUE", 1},
		{"yes", 1},
		{"YES", 1},
		{"false", 0},
		{"no", 0},
		{"", 0},
		{"unknown", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			if got := collector.ParseBool(tt.input); got != tt.want {
				t.Errorf("ParseBool(%q) = %f, want %f", tt.input, got, tt.want)
			}
		})
	}
}

// testEntry builds a minimal RouterEntry for MetricBuilder tests.
func testEntry(customLabels map[string]string) *entry.RouterEntry {
	return &entry.RouterEntry{
		RouterName: "test",
		ConfigEntry: &config.RouterConfigEntry{
			CustomLabels: customLabels,
		},
		RouterID: map[string]string{
			"routerboard_name":    "test-router",
			"routerboard_address": "192.168.1.1",
		},
	}
}

// gatherMetric collects exactly one metric from fn and writes it to a dto.Metric.
func gatherMetric(t *testing.T, fn func(ch chan<- prometheus.Metric)) *dto.Metric {
	t.Helper()
	ch := make(chan prometheus.Metric, 1)
	fn(ch)
	m := <-ch
	var dm dto.Metric
	if err := m.Write(&dm); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	return &dm
}

// labelMap converts a dto.Metric label list into a name→value map.
func labelMap(dm *dto.Metric) map[string]string {
	out := make(map[string]string, len(dm.GetLabel()))
	for _, lp := range dm.GetLabel() {
		out[lp.GetName()] = lp.GetValue()
	}
	return out
}

func TestMetricBuilderGaugeVal(t *testing.T) {
	t.Parallel()

	mb := collector.NewMetricBuilder(testEntry(nil))

	dm := gatherMetric(t, func(ch chan<- prometheus.Metric) {
		mb.GaugeVal(ch, "test_metric", "help", 42.0, []string{"name"}, []string{"ether1"})
	})

	if got := dm.GetGauge().GetValue(); got != 42.0 {
		t.Errorf("value = %f, want 42.0", got)
	}

	labels := labelMap(dm)
	if labels["name"] != "ether1" {
		t.Errorf("name label = %q, want ether1", labels["name"])
	}
	if labels["routerboard_name"] != "test-router" {
		t.Errorf("routerboard_name = %q, want test-router", labels["routerboard_name"])
	}
	if labels["routerboard_address"] != "192.168.1.1" {
		t.Errorf("routerboard_address = %q, want 192.168.1.1", labels["routerboard_address"])
	}
}

func TestMetricBuilderCounterVal(t *testing.T) {
	t.Parallel()

	mb := collector.NewMetricBuilder(testEntry(nil))

	dm := gatherMetric(t, func(ch chan<- prometheus.Metric) {
		mb.CounterVal(ch, "test_counter", "help", 100.0, nil, nil)
	})

	if got := dm.GetCounter().GetValue(); got != 100.0 {
		t.Errorf("value = %f, want 100.0", got)
	}
}

func TestMetricBuilderInfo(t *testing.T) {
	t.Parallel()

	mb := collector.NewMetricBuilder(testEntry(nil))
	record := map[string]string{"version": "7.14", "board_name": "RB4011"}

	dm := gatherMetric(t, func(ch chan<- prometheus.Metric) {
		mb.Info(ch, "system_identity", "help", []string{"version", "board_name"}, record)
	})

	if got := dm.GetGauge().GetValue(); got != 1.0 {
		t.Errorf("info gauge value = %f, want 1.0", got)
	}

	labels := labelMap(dm)
	if labels["version"] != "7.14" {
		t.Errorf("version = %q, want 7.14", labels["version"])
	}
	if labels["board_name"] != "RB4011" {
		t.Errorf("board_name = %q, want RB4011", labels["board_name"])
	}
}

// TestMetricBuilderCustomLabelOrder verifies that custom labels are emitted correctly
// regardless of map iteration order. Prometheus sorts labels alphabetically in output,
// so we verify correctness by name, not by position.
func TestMetricBuilderCustomLabelOrder(t *testing.T) {
	t.Parallel()

	customLabels := map[string]string{
		"zzz": "last",
		"aaa": "first",
		"mmm": "middle",
	}
	mb := collector.NewMetricBuilder(testEntry(customLabels))

	// Run multiple times to catch any label/value mismatch that would cause a panic
	// or wrong values — the critical bug was non-deterministic map iteration causing
	// labelNames and labelVals to diverge.
	for range 20 {
		dm := gatherMetric(t, func(ch chan<- prometheus.Metric) {
			mb.GaugeVal(ch, "test_metric", "help", 1.0, nil, nil)
		})

		labels := labelMap(dm)

		if len(labels) != 5 {
			t.Fatalf("expected 5 labels, got %d: %v", len(labels), labels)
		}
		if labels["aaa"] != "first" {
			t.Errorf("aaa = %q, want first", labels["aaa"])
		}
		if labels["mmm"] != "middle" {
			t.Errorf("mmm = %q, want middle", labels["mmm"])
		}
		if labels["zzz"] != "last" {
			t.Errorf("zzz = %q, want last", labels["zzz"])
		}
		if labels["routerboard_name"] != "test-router" {
			t.Errorf("routerboard_name = %q, want test-router", labels["routerboard_name"])
		}
		if labels["routerboard_address"] != "192.168.1.1" {
			t.Errorf("routerboard_address = %q, want 192.168.1.1", labels["routerboard_address"])
		}
	}
}

func TestMetricBuilderGaugeFromRecord(t *testing.T) {
	t.Parallel()

	mb := collector.NewMetricBuilder(testEntry(nil))
	record := map[string]string{"cpu_load": "75", "name": "ether1"}

	dm := gatherMetric(t, func(ch chan<- prometheus.Metric) {
		mb.Gauge(ch, "cpu_load", "CPU load", "cpu_load", []string{"name"}, record)
	})

	if got := dm.GetGauge().GetValue(); got != 75.0 {
		t.Errorf("value = %f, want 75.0", got)
	}
}

func TestMetricBuilderMissingValueKey(t *testing.T) {
	t.Parallel()

	mb := collector.NewMetricBuilder(testEntry(nil))
	record := map[string]string{"name": "ether1"} // no "bytes" key

	dm := gatherMetric(t, func(ch chan<- prometheus.Metric) {
		mb.Gauge(ch, "bytes", "bytes", "bytes", []string{"name"}, record)
	})

	// Missing key → ParseFloat("") → 0.
	if got := dm.GetGauge().GetValue(); got != 0 {
		t.Errorf("value = %f, want 0 for missing key", got)
	}
}

// connectedEntry builds a RouterEntry with a disconnected (but non-nil) APIConn.
func connectedEntry() *entry.RouterEntry {
	return &entry.RouterEntry{
		RouterName:  "test",
		ConfigEntry: &config.RouterConfigEntry{},
		APIConn:     routeros.NewConnection(routeros.ConnectionConfig{RouterName: "test", Hostname: "localhost"}),
		RouterID: map[string]string{
			"routerboard_name":    "test-router",
			"routerboard_address": "192.168.1.1",
		},
	}
}

func TestHealthCollector_Disconnected(t *testing.T) {
	t.Parallel()

	e := connectedEntry() // APIConn exists but client=nil → IsConnected()=false

	ch := make(chan prometheus.Metric, 1)
	if err := collector.NewHealthCollector().Collect(context.Background(), e, ch); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dm := gatherMetric(t, func(out chan<- prometheus.Metric) { out <- <-ch })
	if got := dm.GetGauge().GetValue(); got != 0 {
		t.Errorf("health_up = %f, want 0 (disconnected)", got)
	}
}

func TestInterfaceCollector_Disabled(t *testing.T) {
	t.Parallel()

	e := &entry.RouterEntry{
		RouterName:  "test",
		ConfigEntry: &config.RouterConfigEntry{Interface: false},
		RouterID: map[string]string{
			"routerboard_name":    "test",
			"routerboard_address": "localhost",
		},
	}

	ch := make(chan prometheus.Metric, 10)
	if err := collector.NewInterfaceCollector().Collect(context.Background(), e, ch); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ch) != 0 {
		t.Errorf("expected no metrics when interface is disabled, got %d", len(ch))
	}
}

func TestDHCPCollector_Disabled(t *testing.T) {
	t.Parallel()

	e := &entry.RouterEntry{
		RouterName:  "test",
		ConfigEntry: &config.RouterConfigEntry{DHCP: false},
		RouterID: map[string]string{
			"routerboard_name":    "test",
			"routerboard_address": "localhost",
		},
	}

	ch := make(chan prometheus.Metric, 10)
	if err := collector.NewDHCPCollector().Collect(context.Background(), e, ch); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ch) != 0 {
		t.Errorf("expected no metrics when DHCP is disabled, got %d", len(ch))
	}
}

func TestFirewallCollector_BothDisabled(t *testing.T) {
	t.Parallel()

	e := &entry.RouterEntry{
		RouterName: "test",
		ConfigEntry: &config.RouterConfigEntry{
			Firewall:     false,
			IPv6Firewall: false,
		},
		RouterID: map[string]string{
			"routerboard_name":    "test",
			"routerboard_address": "localhost",
		},
	}

	ch := make(chan prometheus.Metric, 10)
	if err := collector.NewFirewallCollector().Collect(context.Background(), e, ch); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ch) != 0 {
		t.Errorf("expected no metrics when both firewall flags are disabled, got %d", len(ch))
	}
}
