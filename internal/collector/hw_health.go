package collector

import (
	"context"
	"log/slog"
	"strings"

	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/prometheus/client_golang/prometheus"
)

// HWHealthCollector collects hardware health metrics (voltage, temperature, fans, PSU).
type HWHealthCollector struct{}

func NewHWHealthCollector() *HWHealthCollector                  { return &HWHealthCollector{} }
func (c *HWHealthCollector) Name() string                       { return "hw_health" }
func (c *HWHealthCollector) Describe(_ chan<- *prometheus.Desc) {}

func (c *HWHealthCollector) Collect(ctx context.Context, e *entry.RouterEntry, ch chan<- prometheus.Metric) error {
	if !e.ConfigEntry.Health {
		return nil
	}

	records, err := e.APIConn.Run(ctx, "/system/health/print")
	if err != nil {
		slog.Error("hw_health collect failed", "router", e.RouterName, "err", err)
		return nil
	}

	// RouterOS v7+ returns [{name:"temperature", value:"33", type:"C"}, ...]
	// RouterOS v6  returns [{voltage:"24.0", temperature:"45"}, ...]
	// Normalize to a flat map either way.
	flat := normalizeHealthRecords(records)
	if len(flat) == 0 {
		return nil
	}

	mb := NewMetricBuilder(e)

	for key, val := range flat {
		switch key {
		case "voltage":
			mb.GaugeVal(ch, "system_routerboard_voltage", "Supplied routerboard voltage", ParseFloat(val), nil, nil)
		case "temperature":
			mb.GaugeVal(ch, "system_routerboard_temperature", "Routerboard current temperature", ParseFloat(val), nil, nil)
		case "phy_temperature":
			mb.GaugeVal(ch, "system_phy_temperature", "Current PHY temperature", ParseFloat(val), nil, nil)
		case "cpu_temperature":
			mb.GaugeVal(ch, "system_cpu_temperature", "Current CPU temperature", ParseFloat(val), nil, nil)
		case "switch_temperature":
			mb.GaugeVal(ch, "system_switch_temperature", "Current switch temperature", ParseFloat(val), nil, nil)
		case "fan1_speed":
			mb.GaugeVal(ch, "system_fan_one_speed", "System fan 1 current speed", ParseFloat(val), nil, nil)
		case "fan2_speed":
			mb.GaugeVal(ch, "system_fan_two_speed", "System fan 2 current speed", ParseFloat(val), nil, nil)
		case "fan3_speed":
			mb.GaugeVal(ch, "system_fan_three_speed", "System fan 3 current speed", ParseFloat(val), nil, nil)
		case "fan4_speed":
			mb.GaugeVal(ch, "system_fan_four_speed", "System fan 4 current speed", ParseFloat(val), nil, nil)
		case "power_consumption":
			mb.GaugeVal(ch, "system_power_consumption", "System power consumption", ParseFloat(val), nil, nil)
		case "board_temperature1":
			mb.GaugeVal(ch, "system_board_temperature1", "System board temperature 1", ParseFloat(val), nil, nil)
		case "board_temperature2":
			mb.GaugeVal(ch, "system_board_temperature2", "System board temperature 2", ParseFloat(val), nil, nil)
		case "psu1_voltage":
			mb.GaugeVal(ch, "system_psu1_voltage", "System PSU1 voltage", ParseFloat(val), nil, nil)
		case "psu2_voltage":
			mb.GaugeVal(ch, "system_psu2_voltage", "System PSU2 voltage", ParseFloat(val), nil, nil)
		case "psu1_current":
			mb.GaugeVal(ch, "system_psu1_current", "System PSU1 current", ParseFloat(val), nil, nil)
		case "psu2_current":
			mb.GaugeVal(ch, "system_psu2_current", "System PSU2 current", ParseFloat(val), nil, nil)
		case "psu1_state":
			mb.GaugeVal(ch, "system_psu1_state", "System PSU1 state", ParseBool(val), nil, nil)
		case "psu2_state":
			mb.GaugeVal(ch, "system_psu2_state", "System PSU2 state", ParseBool(val), nil, nil)
		case "poe_out_consumption":
			mb.GaugeVal(ch, "system_poe_out_consumption", "System POE-out consumption", ParseFloat(val), nil, nil)
		case "jack_voltage":
			mb.GaugeVal(ch, "system_jack_voltage", "System jack voltage", ParseFloat(val), nil, nil)
		case "2pin_voltage":
			mb.GaugeVal(ch, "system_2pin_voltage", "System 2-pin voltage", ParseFloat(val), nil, nil)
		case "poe_in_voltage":
			mb.GaugeVal(ch, "system_poe_in_voltage", "System POE-in voltage", ParseFloat(val), nil, nil)
		}
	}

	return nil
}

// normalizeHealthRecords handles both RouterOS v6 (flat) and v7 (name/value/type) formats.
func normalizeHealthRecords(records []map[string]string) map[string]string {
	flat := make(map[string]string)

	for _, rec := range records {
		if name, ok := rec["name"]; ok && name != "" {
			// RouterOS v7 format
			key := NormalizeKey(strings.ToLower(name))
			flat[key] = rec["value"]
		} else {
			// RouterOS v6 format — each record already has the metric as a key
			for k, v := range rec {
				flat[NormalizeKey(k)] = v
			}
		}
	}

	return flat
}
