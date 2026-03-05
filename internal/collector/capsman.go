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

	// Collect CAPsMAN interfaces (access points)
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

		mb.GaugeVal(ch, "capsman_interface_status", "CAPsMAN interface status (1=running, 0=stopped)", func() float64 {
			if rec["running"] == "true" && rec["disabled"] != "true" {
				return 1
			}
			return 0
		}(), labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["mac_address"]})

		if _, ok := rec["disabled"]; ok {
			disabled := 0.0
			if rec["disabled"] == "true" {
				disabled = 1
			}
			mb.GaugeVal(ch, "capsman_interface_disabled", "CAPsMAN interface disabled status", disabled, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["mac_address"]})
		}

		if _, ok := rec["enabled"]; ok {
			enabled := 0.0
			if rec["enabled"] == "true" {
				enabled = 1
			}
			mb.GaugeVal(ch, "capsman_interface_enabled", "CAPsMAN interface enabled status", enabled, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["mac_address"]})
		}

		if _, ok := rec["interface-mode"]; ok && rec["interface-mode"] != "" {
			mb.GaugeVal(ch, "capsman_interface_mode", "CAPsMAN interface mode (bridge/access-point/etc)", 1.0, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["mac_address"]})
		}

		if _, ok := rec["channel"]; ok && rec["channel"] != "" {
			mb.GaugeVal(ch, "capsman_channel", "CAPsMAN channel configuration", 1.0, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["mac_address"]})
		}

		if _, ok := rec["comment"]; ok && rec["comment"] != "" {
			mb.Info(ch, "capsman_interface_info", "Information about CAPsMAN interface",
				[]string{"name", "mac_address", "interface_mode"},
				rec)
		}

		for _, metric := range []struct{ name, key string }{
			{"capsman_mac_address", "mac-address"},
			{"capsman_ip_address", "ip-address"},
			{"capsman_remote_interface", "remote-interface"},
			{"capsman_channel_width", "channel-width"},
		} {
			if val, ok := rec[metric.key]; ok && val != "" {
				mb.GaugeVal(ch, metric.name, "CAPsMAN "+strings.ToUpper(metric.key), 1.0, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["mac_address"]})
			}
		}
	}

	return nil
}
