package collector

import (
	"context"
	"log/slog"
	"strings"

	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/prometheus/client_golang/prometheus"
)

type W60GCollector struct{}

func NewW60GCollector() *W60GCollector { return &W60GCollector{} }

func (c *W60GCollector) Name() string { return "w60g" }

func (c *W60GCollector) Describe(_ chan<- *prometheus.Desc) {}

func (c *W60GCollector) Collect(ctx context.Context, e *entry.RouterEntry, ch chan<- prometheus.Metric) error {
	if !e.ConfigEntry.W60G {
		return nil
	}

	records, err := e.APIConn.Run(ctx, "/interface/w60g/print")
	if err != nil {
		slog.Debug("w60g collect failed", "router", e.RouterName, "err", err)
		return nil
	}

	mb := NewMetricBuilder(e)
	labelKeys := []string{"name"}

	for _, raw := range records {
		rec := TrimRecord(raw, nil)
		if rec["name"] == "" {
			continue
		}

		rec["name"] = FormatInterfaceName(rec["name"], "", e.ConfigEntry.InterfaceNameFormat)
		labelKeysWithRouter := append([]string{"router_id"}, labelKeys...)

		mb.GaugeVal(ch, "w60g_status", "W60G interface status (1=up, 0=down)", func() float64 {
			if rec["running"] == "true" && rec["disabled"] != "true" {
				return 1
			}
			return 0
		}(), labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"]})

		if _, ok := rec["disabled"]; ok {
			disabled := 0.0
			if rec["disabled"] == "true" {
				disabled = 1
			}
			mb.GaugeVal(ch, "w60g_disabled", "W60G interface disabled status", disabled, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"]})
		}

		if _, ok := rec["frequency"]; ok && rec["frequency"] != "" {
			mb.GaugeVal(ch, "w60g_frequency", "W60G operating frequency in GHz", ParseFloat(rec["frequency"]), labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"]})
		}

		if _, ok := rec["bandwidth"]; ok && rec["bandwidth"] != "" {
			mb.GaugeVal(ch, "w60g_bandwidth", "W60G channel bandwidth in MHz", ParseFloat(rec["bandwidth"]), labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"]})
		}

		if _, ok := rec["tx-power"]; ok && rec["tx-power"] != "" {
			mb.GaugeVal(ch, "w60g_tx_power", "W60G transmit power in dBm", ParseFloat(rec["tx-power"]), labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"]})
		}

		if _, ok := rec["signal-strength"]; ok && rec["signal-strength"] != "" {
			mb.GaugeVal(ch, "w60g_signal_strength", "W60G signal strength in dBm", ParseFloat(rec["signal-strength"]), labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"]})
		}

		if _, ok := rec["mode"]; ok && rec["mode"] != "" {
			mb.GaugeVal(ch, "w60g_mode", "W60G operating mode (station/p2p/ap)", 1.0, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"]})
		}

		if _, ok := rec["ssid"]; ok && rec["ssid"] != "" {
			mb.GaugeVal(ch, "w60g_ssid", "W60G SSID configuration", 1.0, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"]})
		}

		if _, ok := rec["country"]; ok && rec["country"] != "" {
			mb.GaugeVal(ch, "w60g_country", "W60G regulatory domain country", 1.0, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"]})
		}

		if _, ok := rec["comment"]; ok && rec["comment"] != "" {
			mb.Info(ch, "w60g_info", "Information about W60G interface",
				[]string{"name", "mode"},
				rec)
		}

		for _, metric := range []struct{ name, key string }{
			{"w60g_channel", "channel"},
			{"w60g_antenna_gain", "antenna-gain"},
			{"w60g_rssi", "rssi"},
			{"w60g_snr", "snr"},
		} {
			if val, ok := rec[metric.key]; ok && val != "" {
				mb.GaugeVal(ch, metric.name, "W60G "+strings.ToUpper(metric.key), 1.0, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"]})
			}
		}
	}

	return nil
}
