package collector

import (
	"context"
	"log/slog"

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
		collectLTE(mb, ch, rec, labelKeysWithRouter, e.RouterID)
	}

	return nil
}

func collectLTE(mb *MetricBuilder, ch chan<- prometheus.Metric, rec map[string]string, labelKeysWithRouter []string, routerID map[string]string) {
	metricMap := map[string]struct {
		name       string
		help       string
		parseFloat bool
	}{
		"signal-strength":     {"signal_strength", "LTE signal strength in dBm", true},
		"rssi":                {"rssi", "LTE RSSI in dBm", true},
		"rsrp":                {"rsrp", "LTE RSRP in dBm", true},
		"rsrq":                {"rsrq", "LTE RSRQ in dB", true},
		"sinr":                {"sinr", "LTE SINR in dB", true},
		"operator":            {"current_operator", "LTE operator", false},
		"cell-id":             {"cell_id", "LTE cell ID", false},
		"tac":                 {"tac", "LTE TAC", false},
		"uplink-speed":        {"rate_up", "LTE uplink speed", true},
		"downlink-speed":      {"rate_down", "LTE downlink speed", true},
		"session-uptime":      {"session_uptime", "LTE session uptime", true},
		"pin-status":          {"pin_status", "LTE PIN status", false},
		"registration-status": {"registration_status", "LTE registration status", false},
		"nr-rsrp":             {"nr_rsrp", "5G NR RSRP in dBm", true},
		"nr-rsrq":             {"nr_rsrq", "5G NR RSRQ in dB", true},
		"nr-sinr":             {"nr_sinr", "5G NR SINR in dB", true},
	}

	for key, meta := range metricMap {
		if val, ok := rec[key]; ok && val != "" {
			var value float64
			if meta.parseFloat {
				value = ParseFloat(val)
			} else {
				value = 1.0
			}
			mb.GaugeVal(ch, meta.name, meta.help, value, labelKeysWithRouter, []string{routerID["router_id"], rec["name"], rec["device_name"]})
		}
	}

	if rec["running"] == trueStr && rec["connected"] == trueStr {
		mb.GaugeVal(ch, "lte_status", "LTE interface status (1=up, 0=down)", 1.0, labelKeysWithRouter, []string{routerID["router_id"], rec["name"], rec["device_name"]})
	} else {
		mb.GaugeVal(ch, "lte_status", "LTE interface status (1=up, 0=down)", 0.0, labelKeysWithRouter, []string{routerID["router_id"], rec["name"], rec["device_name"]})
	}

	if disabledVal, ok := rec["disabled"]; ok {
		disabled := 0.0
		if disabledVal == trueStr {
			disabled = 1.0
		}
		mb.GaugeVal(ch, "lte_disabled", "LTE interface disabled status", disabled, labelKeysWithRouter, []string{routerID["router_id"], rec["name"], rec["device_name"]})
	}

	if connectedVal, ok := rec["connected"]; ok {
		connected := 0.0
		if connectedVal == trueStr {
			connected = 1.0
		}
		mb.GaugeVal(ch, "lte_connected", "LTE connection status (1=connected, 0=disconnected)", connected, labelKeysWithRouter, []string{routerID["router_id"], rec["name"], rec["device_name"]})
	}

	for _, key := range []string{"imei", "iccid", "apn", "mode", "ip-address"} {
		if val, ok := rec[key]; ok && val != "" {
			var help string
			switch key {
			case "imei":
				help = "LTE IMEI number"
			case "iccid":
				help = "LTE ICCID number"
			case "apn":
				help = "LTE APN configuration"
			case "mode":
				help = "LTE mode (2G/3G/4G)"
			case "ip-address":
				help = "LTE assigned IP address"
			}
			mb.GaugeVal(ch, "lte_"+key, help, 1.0, labelKeysWithRouter, []string{routerID["router_id"], rec["name"], rec["device_name"]})
		}
	}

	if comment, ok := rec["comment"]; ok && comment != "" {
		mb.Info(ch, "lte_info", "Information about LTE interface",
			[]string{"name", "device_name", "apn", "mode"},
			rec)
	}
}
