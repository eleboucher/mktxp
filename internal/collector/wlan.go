package collector

import (
	"context"
	"log/slog"
	"strings"

	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/prometheus/client_golang/prometheus"
)

type WLANCollector struct{}

func NewWLANCollector() *WLANCollector { return &WLANCollector{} }

func (c *WLANCollector) Name() string { return "wlan" }

func (c *WLANCollector) Describe(_ chan<- *prometheus.Desc) {}

func (c *WLANCollector) Collect(ctx context.Context, e *entry.RouterEntry, ch chan<- prometheus.Metric) error {
	if !e.ConfigEntry.Wireless {
		return nil
	}

	records, err := e.APIConn.Run(ctx, "/interface/wireless/print")
	if err != nil {
		slog.Debug("wlan collect failed", "router", e.RouterName, "err", err)
		return nil
	}

	mb := NewMetricBuilder(e)
	labelKeys := []string{"name", "ssid"}

	for _, raw := range records {
		rec := TrimRecord(raw, nil)
		if rec["name"] == "" {
			continue
		}

		rec["name"] = FormatInterfaceName(rec["name"], "", e.ConfigEntry.InterfaceNameFormat)
		labelKeysWithRouter := append([]string{"router_id"}, labelKeys...)

		mb.GaugeVal(ch, "wlan_status", "WLAN interface status (1=up, 0=down)", func() float64 {
			if rec["running"] == "true" && rec["disabled"] != "true" {
				return 1
			}
			return 0
		}(), labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["ssid"]})

		if _, ok := rec["disabled"]; ok {
			disabled := 0.0
			if rec["disabled"] == "true" {
				disabled = 1
			}
			mb.GaugeVal(ch, "wlan_disabled", "WLAN interface disabled status", disabled, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["ssid"]})
		}

		if _, ok := rec["band"]; ok && rec["band"] != "" {
			mb.GaugeVal(ch, "wlan_band", "WLAN frequency band (2g/5g)", 1.0, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["ssid"]})
		}

		if _, ok := rec["channel-width"]; ok && rec["channel-width"] != "" {
			mb.GaugeVal(ch, "wlan_channel_width", "WLAN channel width in MHz", ParseFloat(rec["channel-width"]), labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["ssid"]})
		}

		if _, ok := rec["frequency"]; ok && rec["frequency"] != "" {
			mb.GaugeVal(ch, "wlan_frequency", "WLAN operating frequency in MHz", ParseFloat(rec["frequency"]), labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["ssid"]})
		}

		if _, ok := rec["tx-power"]; ok && rec["tx-power"] != "" {
			mb.GaugeVal(ch, "wlan_tx_power", "WLAN transmit power in dBm", ParseFloat(rec["tx-power"]), labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["ssid"]})
		}

		if _, ok := rec["country"]; ok && rec["country"] != "" {
			mb.GaugeVal(ch, "wlan_country", "WLAN regulatory domain country", 1.0, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["ssid"]})
		}

		if _, ok := rec["mode"]; ok && rec["mode"] != "" {
			mb.GaugeVal(ch, "wlan_mode", "WLAN operating mode (ap-bridge/station/etc)", 1.0, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["ssid"]})
		}

		if _, ok := rec["wireless-protocol"]; ok && rec["wireless-protocol"] != "" {
			mb.GaugeVal(ch, "wlan_protocol", "WLAN protocol (802.11a/b/g/n/ac/ax)", 1.0, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["ssid"]})
		}

		if _, ok := rec["comment"]; ok && rec["comment"] != "" {
			mb.Info(ch, "wlan_info", "Information about WLAN interface",
				[]string{"name", "ssid", "band", "mode"},
				rec)
		}

		for _, metric := range []struct{ name, key string }{
			{"wlan_ssid", "ssid"},
			{"wlan_bssid", "bssid"},
			{"wlan_channel", "channel"},
			{"wlan_bridge_mode", "bridge-mode"},
			{"wlan_security_profile", "security-profile"},
		} {
			if val, ok := rec[metric.key]; ok && val != "" {
				mb.GaugeVal(ch, metric.name, "WLAN "+strings.ToUpper(metric.key), 1.0, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["ssid"]})
			}
		}
	}

	return nil
}
