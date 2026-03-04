package collector

import (
	"testing"
)

func TestNormalizeHealthRecords_V7Format(t *testing.T) {
	t.Parallel()

	records := []map[string]string{
		{"name": "temperature", "value": "45", "type": "C"},
		{"name": "voltage", "value": "24.5", "type": "V"},
		{"name": "fan1-speed", "value": "1200", "type": "RPM"},
	}

	got := normalizeHealthRecords(records)

	if got["temperature"] != "45" {
		t.Errorf("temperature = %q, want 45", got["temperature"])
	}
	if got["voltage"] != "24.5" {
		t.Errorf("voltage = %q, want 24.5", got["voltage"])
	}
	if got["fan1_speed"] != "1200" {
		t.Errorf("fan1_speed = %q, want 1200 (key should be normalized)", got["fan1_speed"])
	}
}

func TestNormalizeHealthRecords_V6Format(t *testing.T) {
	t.Parallel()

	// v6: one record with all metrics as flat key-value pairs.
	records := []map[string]string{
		{"temperature": "45", "voltage": "24.5"},
	}

	got := normalizeHealthRecords(records)

	if got["temperature"] != "45" {
		t.Errorf("temperature = %q, want 45", got["temperature"])
	}
	if got["voltage"] != "24.5" {
		t.Errorf("voltage = %q, want 24.5", got["voltage"])
	}
}

func TestNormalizeHealthRecords_Empty(t *testing.T) {
	t.Parallel()

	got := normalizeHealthRecords(nil)
	if len(got) != 0 {
		t.Errorf("expected empty map for nil input, got %v", got)
	}

	got = normalizeHealthRecords([]map[string]string{})
	if len(got) != 0 {
		t.Errorf("expected empty map for empty input, got %v", got)
	}
}

func TestNormalizeHealthRecords_V7KeyNormalization(t *testing.T) {
	t.Parallel()

	// RouterOS v7 health names use dashes; NormalizeKey converts them to underscores.
	records := []map[string]string{
		{"name": "poe-in-voltage", "value": "52.1"},
		{"name": "cpu-temperature", "value": "38"},
	}

	got := normalizeHealthRecords(records)

	if got["poe_in_voltage"] != "52.1" {
		t.Errorf("poe_in_voltage = %q, want 52.1", got["poe_in_voltage"])
	}
	if got["cpu_temperature"] != "38" {
		t.Errorf("cpu_temperature = %q, want 38", got["cpu_temperature"])
	}
}

func TestSplitSimpleQueue(t *testing.T) {
	t.Parallel()

	t.Run("splits_slash_separated_values", func(t *testing.T) {
		t.Parallel()
		rec := map[string]string{
			"bytes": "1000/2000",
			"rate":  "100/200",
		}
		got := splitSimpleQueue(rec)
		if got["bytes_up"] != "1000" {
			t.Errorf("bytes_up = %q, want 1000", got["bytes_up"])
		}
		if got["bytes_down"] != "2000" {
			t.Errorf("bytes_down = %q, want 2000", got["bytes_down"])
		}
		if got["rate_up"] != "100" {
			t.Errorf("rate_up = %q, want 100", got["rate_up"])
		}
		if got["rate_down"] != "200" {
			t.Errorf("rate_down = %q, want 200", got["rate_down"])
		}
	})

	t.Run("non_slash_values_kept_as_is", func(t *testing.T) {
		t.Parallel()
		rec := map[string]string{"name": "myqueue"}
		got := splitSimpleQueue(rec)
		if got["name"] != "myqueue" {
			t.Errorf("name = %q, want myqueue", got["name"])
		}
		if _, ok := got["name_up"]; ok {
			t.Error("name_up should not exist for a non-split value")
		}
	})

	t.Run("zero_values", func(t *testing.T) {
		t.Parallel()
		rec := map[string]string{"bytes": "0/0"}
		got := splitSimpleQueue(rec)
		if got["bytes_up"] != "0" || got["bytes_down"] != "0" {
			t.Errorf("zero split: up=%q down=%q", got["bytes_up"], got["bytes_down"])
		}
	})
}

func TestFloatStr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input float64
		want  string
	}{
		{0, "0"},
		{1, "1"},
		{1.5, "1.5"},
		{0.00025, "0.00025"},
		{1000, "1000"},
		{-1.5, "-1.5"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			if got := floatStr(tt.input); got != tt.want {
				t.Errorf("floatStr(%v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
