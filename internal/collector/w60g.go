package collector

import (
	"context"
	"log/slog"

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
	labelKeysWithRouter := labelKeys

	metricMap := map[string]struct {
		name       string
		help       string
		parseFloat bool
	}{
		"disabled":             {"w60g_disabled", "W60G interface disabled status", false},
		"frequency":            {"w60g_frequency", "W60G operating frequency in GHz", true},
		"bandwidth":            {"w60g_bandwidth", "W60G channel bandwidth in MHz", true},
		"tx-power":             {"w60g_tx_power", "W60G transmit power in dBm", true},
		"signal-strength":      {"w60g_signal_strength", "W60G signal strength in dBm", true},
		"mode":                 {"w60g_mode", "W60G operating mode (station/p2p/ap)", false},
		"ssid":                 {"w60g_ssid", "W60G SSID configuration", false},
		"country":              {"w60g_country", "W60G regulatory domain country", false},
		"channel":              {"w60g_channel", "W60G CHANNEL", false},
		"antenna-gain":         {"w60g_antenna_gain", "W60G ANTENNA-GAIN", false},
		"rssi":                 {"w60g_rssi", "W60G RSSI", false},
		"snr":                  {"w60g_snr", "W60G SNR", false},
		"baseband-temperature": {"w60g_baseband_temperature", "Baseband unit temperature", true},
		"connected":            {"w60g_connected", "Connected status", false},
		"distance":             {"w60g_distance", "Distance to W60G peer", true},
		"rf-temperature":       {"w60g_rf_temperature", "RF module temperature", true},
		"signal":               {"w60g_signal", "Link Signal Strength", true},
		"tx-mcs":               {"w60g_tx_mcs", "Transmission MCS", true},
		"tx-packet-error-rate": {"w60g_tx_packet_error_rate", "Transmission error rate (percentage)", true},
		"tx-phy-rate":          {"w60g_tx_phy_rate", "Transmission PHY rate", true},
		"tx-sector":            {"w60g_tx_sector", "Transmission sector number", true},
	}

	for _, raw := range records {
		rec := TrimRecord(raw, nil)
		if rec["name"] == "" {
			continue
		}

		rec["name"] = FormatInterfaceName(rec["name"], "", e.ConfigEntry.InterfaceNameFormat)

		if rec["running"] == trueStr && rec["disabled"] != trueStr {
			mb.GaugeVal(ch, "w60g_status", "W60G interface status (1=up, 0=down)", 1.0, labelKeysWithRouter, []string{rec["name"]})
		} else {
			mb.GaugeVal(ch, "w60g_status", "W60G interface status (1=up, 0=down)", 0.0, labelKeysWithRouter, []string{rec["name"]})
		}

		for rosKey, metric := range metricMap {
			if val, ok := rec[rosKey]; ok && val != "" {
				var metricValue float64
				if metric.parseFloat {
					metricValue = ParseFloat(val)
				} else {
					if val == trueStr {
						metricValue = 1.0
					} else {
						metricValue = 0.0
					}
				}
				mb.GaugeVal(ch, metric.name, metric.help, metricValue, labelKeysWithRouter, []string{rec["name"]})
			}
		}

		if _, ok := rec["comment"]; ok && rec["comment"] != "" {
			mb.Info(ch, "w60g_info", "Information about W60G interface",
				[]string{"name", "mode"},
				rec)
		}
	}

	return nil
}
