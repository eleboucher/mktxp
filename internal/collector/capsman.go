package collector

import (
	"context"
	"log/slog"
	"strings"

	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/prometheus/client_golang/prometheus"
)

type CAPsMANCollector struct{}

func NewCAPsMANCollector() *CAPsMANCollector { return &CAPsMANCollector{} }

func (c *CAPsMANCollector) Name() string { return "capsman" }

func (c *CAPsMANCollector) Describe(_ chan<- *prometheus.Desc) {}

func (c *CAPsMANCollector) Collect(ctx context.Context, e *entry.RouterEntry, ch chan<- prometheus.Metric) error {
	if !e.ConfigEntry.CAPsMAN {
		return nil
	}

	interfaces, err := e.APIConn.Run(ctx, "/interface/caps-man/print")
	if err != nil {
		slog.Debug("capsman interface collect failed", "router", e.RouterName, "err", err)
		return nil
	}

	mb := NewMetricBuilder(e)
	labelKeys := []string{"name", "mac_address"}

	for _, raw := range interfaces {
		rec := TrimRecord(raw, nil)
		if rec["name"] == "" {
			continue
		}

		rec["name"] = FormatInterfaceName(rec["name"], "", e.ConfigEntry.InterfaceNameFormat)
		labelKeysWithRouter := append([]string{"router_id"}, labelKeys...)
		labelVals := []string{e.RouterID["router_id"], rec["name"], rec["mac_address"]}

		metricMap := map[string]struct {
			name       string
			help       string
			parseFloat bool
		}{
			"mac-address":      {"capsman_mac_address", "CAPsMAN MAC address", false},
			"ip-address":       {"capsman_ip_address", "CAPsMAN IP address", false},
			"remote-interface": {"capsman_remote_interface", "CAPsMAN remote interface", false},
			"channel-width":    {"capsman_channel_width", "CAPsMAN channel width", false},
		}

		for key, meta := range metricMap {
			if val, ok := rec[key]; ok && val != "" {
				mb.GaugeVal(ch, meta.name, meta.help, 1, labelKeysWithRouter, labelVals)
			}
		}

		metricGauges := map[string]struct {
			name       string
			help       string
			parseFloat bool
		}{
			"running":  {"capsman_interface_status", "CAPsMAN interface status (1=running, 0=stopped)", false},
			"disabled": {"capsman_interface_disabled", "CAPsMAN interface disabled status", false},
			"enabled":  {"capsman_interface_enabled", "CAPsMAN interface enabled status", false},
		}

		for key, meta := range metricGauges {
			if val, ok := rec[key]; ok && val != "" {
				var value float64
				if meta.parseFloat {
					value = ParseFloat(val)
				} else {
					if key == "running" {
						if rec["running"] == "true" && rec["disabled"] != "true" {
							value = 1
						} else {
							value = 0
						}
					} else {
						value = ParseBool(val)
					}
				}
				mb.GaugeVal(ch, meta.name, meta.help, value, labelKeysWithRouter, labelVals)
			}
		}

		infoFields := map[string]string{
			"interface-mode": "CAPsMAN interface mode (bridge/access-point/etc)",
			"channel":        "CAPsMAN channel configuration",
		}

		for key, help := range infoFields {
			if _, ok := rec[key]; ok && rec[key] != "" {
				mb.GaugeVal(ch, "capsman_"+strings.ReplaceAll(key, "-", "_"), help, 1, labelKeysWithRouter, labelVals)
			}
		}

		if _, ok := rec["comment"]; ok && rec["comment"] != "" {
			mb.Info(ch, "capsman_interface_info", "Information about CAPsMAN interface",
				[]string{"name", "mac_address", "interface_mode"},
				rec)
		}
	}

	return nil
}
