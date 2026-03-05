package collector

import (
	"context"
	"log/slog"
	"strings"

	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/prometheus/client_golang/prometheus"
)

type LTECollector struct{}

func NewLTECollector() *LTECollector { return &LTECollector{} }

func (c *LTECollector) Name() string { return "lte" }

func (c *LTECollector) Describe(_ chan<- *prometheus.Desc) {}

func (c *LTECollector) Collect(ctx context.Context, e *entry.RouterEntry, ch chan<- prometheus.Metric) error {
	if !e.ConfigEntry.LTE {
		return nil
	}

	records, err := e.APIConn.Run(ctx, "/interface/lte/print")
	if err != nil {
		slog.Debug("lte collect failed", "router", e.RouterName, "err", err)
		return nil
	}

	mb := NewMetricBuilder(e)
	labelKeys := []string{"name", "device_name"}

	for _, raw := range records {
		rec := TrimRecord(raw, nil)
		if rec["name"] == "" {
			continue
		}

		rec["name"] = FormatInterfaceName(rec["name"], "", e.ConfigEntry.InterfaceNameFormat)
		labelKeysWithRouter := append([]string{"router_id"}, labelKeys...)

		mb.GaugeVal(ch, "lte_status", "LTE interface status (1=up, 0=down)", func() float64 {
			if rec["running"] == "true" && rec["connected"] == "true" {
				return 1
			}
			return 0
		}(), labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["device_name"]})

		if _, ok := rec["disabled"]; ok {
			disabled := 0.0
			if rec["disabled"] == "true" {
				disabled = 1
			}
			mb.GaugeVal(ch, "lte_disabled", "LTE interface disabled status", disabled, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["device_name"]})
		}

		if _, ok := rec["connected"]; ok {
			connected := 0.0
			if rec["connected"] == "true" {
				connected = 1
			}
			mb.GaugeVal(ch, "lte_connected", "LTE connection status (1=connected, 0=disconnected)", connected, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["device_name"]})
		}

		if _, ok := rec["signal-strength"]; ok && rec["signal-strength"] != "" {
			mb.GaugeVal(ch, "lte_signal_strength", "LTE signal strength in dBm", ParseFloat(rec["signal-strength"]), labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["device_name"]})
		}

		if _, ok := rec["rssi"]; ok && rec["rssi"] != "" {
			mb.GaugeVal(ch, "lte_rssi", "LTE RSSI in dBm", ParseFloat(rec["rssi"]), labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["device_name"]})
		}

		if _, ok := rec["rsrp"]; ok && rec["rsrp"] != "" {
			mb.GaugeVal(ch, "lte_rsrp", "LTE RSRP in dBm", ParseFloat(rec["rsrp"]), labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["device_name"]})
		}

		if _, ok := rec["rsrq"]; ok && rec["rsrq"] != "" {
			mb.GaugeVal(ch, "lte_rsrq", "LTE RSRQ in dB", ParseFloat(rec["rsrq"]), labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["device_name"]})
		}

		if _, ok := rec["sinr"]; ok && rec["sinr"] != "" {
			mb.GaugeVal(ch, "lte_sinr", "LTE SINR in dB", ParseFloat(rec["sinr"]), labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["device_name"]})
		}

		if _, ok := rec["imei"]; ok && rec["imei"] != "" {
			mb.GaugeVal(ch, "lte_imei", "LTE IMEI number", 1.0, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["device_name"]})
		}

		if _, ok := rec["iccid"]; ok && rec["iccid"] != "" {
			mb.GaugeVal(ch, "lte_iccid", "LTE ICCID number", 1.0, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["device_name"]})
		}

		if _, ok := rec["apn"]; ok && rec["apn"] != "" {
			mb.GaugeVal(ch, "lte_apn", "LTE APN configuration", 1.0, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["device_name"]})
		}

		if _, ok := rec["mode"]; ok && rec["mode"] != "" {
			mb.GaugeVal(ch, "lte_mode", "LTE mode (2G/3G/4G)", 1.0, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["device_name"]})
		}

		if _, ok := rec["ip-address"]; ok && rec["ip-address"] != "" {
			mb.GaugeVal(ch, "lte_ip_address", "LTE assigned IP address", 1.0, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["device_name"]})
		}

		if _, ok := rec["comment"]; ok && rec["comment"] != "" {
			mb.Info(ch, "lte_info", "Information about LTE interface",
				[]string{"name", "device_name", "apn", "mode"},
				rec)
		}

		for _, metric := range []struct{ name, key string }{
			{"lte_operator", "operator"},
			{"lte_cell_id", "cell-id"},
			{"lte_tac", "tac"},
			{"lte_uplink_speed", "uplink-speed"},
			{"lte_downlink_speed", "downlink-speed"},
		} {
			if val, ok := rec[metric.key]; ok && val != "" {
				mb.GaugeVal(ch, metric.name, "LTE "+strings.ToUpper(metric.key), 1.0, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["device_name"]})
			}
		}
	}

	return nil
}
