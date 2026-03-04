package utils

import (
	"testing"
)

func TestParseMktUptime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"0s", 0},
		{"1s", 1},
		{"1m", 60},
		{"1h", 3600},
		{"1d", 86400},
		{"1w", 604800},
		{"2w3d4h5m6s", 2*604800 + 3*86400 + 4*3600 + 5*60 + 6},
		{"10h30m", 10*3600 + 30*60},
		{"45s", 45},
		{"1d1s", 86401},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got := ParseMktUptime(tt.input)
			if got != tt.want {
				t.Errorf("ParseMktUptime(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseTimedelta(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		ms    bool
		want  float64
	}{
		{"", false, 0},
		{"1s", false, 1},
		{"1m30s", false, 90},
		{"", true, 0},
		{"1s", true, 1},
		{"500ms", true, 0.5},
		{"1s500ms", true, 1.5},
		{"250us", true, 0.00025},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got := ParseTimedelta(tt.input, tt.ms)
			if got != tt.want {
				t.Errorf("ParseTimedelta(%q, %v) = %f, want %f", tt.input, tt.ms, got, tt.want)
			}
		})
	}
}

func TestStr2Bool(t *testing.T) {
	t.Parallel()

	trueInputs := []string{"y", "yes", "YES", "t", "true", "True", "TRUE", "on", "ok", "1"}
	falseInputs := []string{"n", "no", "NO", "f", "false", "False", "off", "fail", "0"}

	for _, s := range trueInputs {
		s := s
		t.Run("true/"+s, func(t *testing.T) {
			t.Parallel()
			if got := Str2Bool(s, false); !got {
				t.Errorf("Str2Bool(%q, false) = false, want true", s)
			}
		})
	}

	for _, s := range falseInputs {
		s := s
		t.Run("false/"+s, func(t *testing.T) {
			t.Parallel()
			if got := Str2Bool(s, true); got {
				t.Errorf("Str2Bool(%q, true) = true, want false", s)
			}
		})
	}

	t.Run("default_true", func(t *testing.T) {
		t.Parallel()
		if got := Str2Bool("unknown", true); !got {
			t.Error("Str2Bool with unknown input should return default")
		}
	})
}

func TestFormatInterfaceName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name, comment, format string
		want                  string
	}{
		{"ether1", "", "name", "ether1"},
		{"ether1", "WAN", "name", "ether1"},
		{"ether1", "WAN", "comment", "WAN"},
		{"ether1", "", "comment", "ether1"},
		{"ether1", "WAN", "combined", "ether1 (WAN)"},
		{"ether1", "", "combined", "ether1"},
		{"ether1", "WAN", "invalid", "ether1"},
		// Comment truncation at 20 chars
		{"ether1", "a very long interface comment", "comment", "a very long interfac"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.format+"/"+tt.name, func(t *testing.T) {
			t.Parallel()
			got := FormatInterfaceName(tt.name, tt.comment, tt.format)
			if got != tt.want {
				t.Errorf("FormatInterfaceName(%q, %q, %q) = %q, want %q",
					tt.name, tt.comment, tt.format, got, tt.want)
			}
		})
	}
}

func TestRouterOS7Version(t *testing.T) {
	t.Parallel()

	tests := []struct {
		version string
		want    bool
	}{
		{"6.49.10", false},
		{"7.0", true},
		{"7.14.3 (stable)", true},
		{"8.0", true},
		{"5.26", false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.version, func(t *testing.T) {
			t.Parallel()
			got := RouterOS7Version(tt.version)
			if got != tt.want {
				t.Errorf("RouterOS7Version(%q) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}

func TestBuiltinWiFiCAPsMANVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		version string
		want    bool
	}{
		{"7.12", false},
		{"7.13", true},
		{"7.14.3 (stable)", true},
		{"8.0", true},
		{"6.49.10", false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.version, func(t *testing.T) {
			t.Parallel()
			got := BuiltinWiFiCAPsMANVersion(tt.version)
			if got != tt.want {
				t.Errorf("BuiltinWiFiCAPsMANVersion(%q) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}
