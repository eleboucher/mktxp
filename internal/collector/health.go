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

	for _, key := range labels {
		if val, ok := record[key]; ok && val != "" {
			switch key {
			case "psu1_state", "psu2_state":
				state := 0.0
				if strings.ToLower(val) == "ok" || val == "true" {
					state = 1
				}
				mb.GaugeVal(ch, "system_"+key+"_state", "System "+strings.ReplaceAll(key, "_", " ")+" state", state, []string{"routerboard_name", "routerboard_address"}, []string{e.RouterID["routerboard_name"], e.RouterID["routerboard_address"]})
			default:
				if num := ParseFloat(val); num != 0 || key == "voltage" || key == "temperature" {
					mb.GaugeVal(ch, "system_"+key, "System "+strings.ReplaceAll(key, "_", " ")+" value", num, []string{"routerboard_name", "routerboard_address"}, []string{e.RouterID["routerboard_name"], e.RouterID["routerboard_address"]})
				}
			}
		}
	}

	return nil
}
