package collector

import (
	"context"
	"fmt"
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
		return fmt.Errorf("hw_health: %w", err)
	}

	flat := normalizeHealthRecords(records)
	if len(flat) == 0 {
		return nil
	}

	mb := NewMetricBuilder(e)

	metricMap := map[string]struct {
		name       string
		help       string
		parseFloat bool
	}{
		"voltage":             {"system_routerboard_voltage", "Supplied routerboard voltage", true},
		"temperature":         {"system_routerboard_temperature", "Routerboard current temperature", true},
		"phy_temperature":     {"system_phy_temperature", "Current PHY temperature", true},
		"cpu_temperature":     {"system_cpu_temperature", "Current CPU temperature", true},
		"switch_temperature":  {"system_switch_temperature", "Current switch temperature", true},
		"fan1_speed":          {"system_fan_one_speed", "System fan 1 current speed", true},
		"fan2_speed":          {"system_fan_two_speed", "System fan 2 current speed", true},
		"fan3_speed":          {"system_fan_three_speed", "System fan 3 current speed", true},
		"fan4_speed":          {"system_fan_four_speed", "System fan 4 current speed", true},
		"power_consumption":   {"system_power_consumption", "System power consumption", true},
		"board_temperature1":  {"system_board_temperature1", "System board temperature 1", true},
		"board_temperature2":  {"system_board_temperature2", "System board temperature 2", true},
		"psu1_voltage":        {"system_psu1_voltage", "System PSU1 voltage", true},
		"psu2_voltage":        {"system_psu2_voltage", "System PSU2 voltage", true},
		"psu1_current":        {"system_psu1_current", "System PSU1 current", true},
		"psu2_current":        {"system_psu2_current", "System PSU2 current", true},
		"poe_out_consumption": {"system_poe_out_consumption", "System POE-out consumption", true},
		"jack_voltage":        {"system_jack_voltage", "System jack voltage", true},
		"2pin_voltage":        {"system_2pin_voltage", "System 2-pin voltage", true},
		"poe_in_voltage":      {"system_poe_in_voltage", "System POE-in voltage", true},
	}

	stateKeys := map[string]string{
		"psu1_state": "system_psu1_state",
		"psu2_state": "system_psu2_state",
	}

	for key, val := range flat {
		if _, isState := stateKeys[key]; isState {
			state := ParseBool(val)
			mb.GaugeVal(ch, stateKeys[key], "System "+strings.ReplaceAll(key, "_", " ")+" state", state, nil, nil)
		} else if meta, ok := metricMap[key]; ok {
			num := ParseFloat(val)
			mb.GaugeVal(ch, meta.name, meta.help, num, nil, nil)
		}
	}

	return nil
}

func normalizeHealthRecords(records []map[string]string) map[string]string {
	flat := make(map[string]string)

	for _, rec := range records {
		if name, ok := rec["name"]; ok && name != "" {
			key := NormalizeKey(strings.ToLower(name))
			flat[key] = rec["value"]
		} else {
			for k, v := range rec {
				flat[NormalizeKey(k)] = v
			}
		}
	}

	return flat
}
