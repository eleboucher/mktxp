package collector

import (
	"context"
	"log/slog"
	"strings"

	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/prometheus/client_golang/prometheus"
)

type HealthCollector struct{}

func NewHealthCollector() *HealthCollector {
	return &HealthCollector{}
}

func (c *HealthCollector) Name() string { return "health" }

func (c *HealthCollector) Describe(_ chan<- *prometheus.Desc) {}

func (c *HealthCollector) Collect(ctx context.Context, e *entry.RouterEntry, ch chan<- prometheus.Metric) error {
	mb := NewMetricBuilder(e)
	value := 0.0
	if e.APIConn.IsConnected() {
		value = 1
	}
	mb.GaugeVal(ch, "health_up", "Indicates if the router is reachable and responding", value, nil, nil)

	if !e.ConfigEntry.Health {
		return nil
	}

	records, err := e.APIConn.Run(ctx, "/system/health/print")
	if err != nil {
		slog.Debug("health collect failed", "router", e.RouterName, "err", err)
		return nil
	}

	if len(records) == 0 {
		return nil
	}

	record := TrimRecord(records[0], nil)

	labels := []string{
		"voltage", "temperature", "phy_temperature", "cpu_temperature", "switch_temperature",
		"fan1_speed", "fan2_speed", "fan3_speed", "fan4_speed", "power_consumption",
		"board_temperature1", "board_temperature2",
		"psu1_voltage", "psu2_voltage", "psu1_current", "psu2_current",
		"psu1_state", "psu2_state",
		"poe_out_consumption", "jack_voltage", "2pin_voltage", "poe_in_voltage",
	}

	metricMap := map[string]struct {
		name       string
		help       string
		parseFloat bool
	}{
		"voltage":             {"system_voltage", "System voltage value", true},
		"temperature":         {"system_temperature", "System temperature value", true},
		"phy_temperature":     {"system_phy_temperature", "System PHY temperature value", true},
		"cpu_temperature":     {"system_cpu_temperature", "System CPU temperature value", true},
		"switch_temperature":  {"system_switch_temperature", "System switch temperature value", true},
		"fan1_speed":          {"system_fan1_speed", "System fan1 speed value", true},
		"fan2_speed":          {"system_fan2_speed", "System fan2 speed value", true},
		"fan3_speed":          {"system_fan3_speed", "System fan3 speed value", true},
		"fan4_speed":          {"system_fan4_speed", "System fan4 speed value", true},
		"power_consumption":   {"system_power_consumption", "System power consumption value", true},
		"board_temperature1":  {"system_board_temperature1", "System board temperature1 value", true},
		"board_temperature2":  {"system_board_temperature2", "System board temperature2 value", true},
		"psu1_voltage":        {"system_psu1_voltage", "System PSU1 voltage value", true},
		"psu2_voltage":        {"system_psu2_voltage", "System PSU2 voltage value", true},
		"psu1_current":        {"system_psu1_current", "System PSU1 current value", true},
		"psu2_current":        {"system_psu2_current", "System PSU2 current value", true},
		"poe_out_consumption": {"system_poe_out_consumption", "System POE out consumption value", true},
		"jack_voltage":        {"system_jack_voltage", "System jack voltage value", true},
		"2pin_voltage":        {"system_2pin_voltage", "System 2pin voltage value", true},
		"poe_in_voltage":      {"system_poe_in_voltage", "System POE in voltage value", true},
	}

	stateKeys := map[string]string{
		"psu1_state": "system_psu1_state",
		"psu2_state": "system_psu2_state",
	}

	for _, key := range labels {
		if val, ok := record[key]; ok && val != "" {
			if _, isState := stateKeys[key]; isState {
				state := 0.0
				if strings.ToLower(val) == "ok" || val == trueStr {
					state = 1
				}
				mb.GaugeVal(ch, stateKeys[key], "System "+strings.ReplaceAll(key, "_", " ")+" state", state, []string{"routerboard_name", "routerboard_address"}, []string{e.RouterID["routerboard_name"], e.RouterID["routerboard_address"]})
			} else if meta, ok := metricMap[key]; ok {
				num := ParseFloat(val)
				if num != 0 || key == "voltage" || key == "temperature" {
					mb.GaugeVal(ch, meta.name, meta.help, num, []string{"routerboard_name", "routerboard_address"}, []string{e.RouterID["routerboard_name"], e.RouterID["routerboard_address"]})
				}
			}
		}
	}

	return nil
}
