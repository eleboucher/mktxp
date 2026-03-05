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
		labelVals := []string{e.RouterID["router_id"], rec["name"], rec["ssid"]}

		metricMap := map[string]struct {
			name       string
			help       string
			parseFloat bool
		}{
			"ssid":             {"wlan_ssid", "WLAN SSID", false},
			"bssid":            {"wlan_bssid", "WLAN BSSID", false},
			"channel":          {"wlan_channel", "WLAN Channel", false},
			"bridge-mode":      {"wlan_bridge_mode", "WLAN Bridge Mode", false},
			"security-profile": {"wlan_security_profile", "WLAN Security Profile", false},
			"channel-width":    {"wlan_channel_width", "WLAN channel width in MHz", true},
			"frequency":        {"wlan_frequency", "WLAN operating frequency in MHz", true},
			"tx-power":         {"wlan_tx_power", "WLAN transmit power in dBm", true},
		}

		for key, meta := range metricMap {
			if val, ok := rec[key]; ok && val != "" {
				var value float64
				if meta.parseFloat {
					value = ParseFloat(val)
				} else {
					value = 1
				}
				mb.GaugeVal(ch, meta.name, meta.help, value, labelKeysWithRouter, labelVals)
			}
		}

		if status, ok := rec["running"]; ok {
			wlanStatus := 0.0
			if status == trueStr && rec["disabled"] != trueStr {
				wlanStatus = 1
			}
			mb.GaugeVal(ch, "wlan_status", "WLAN interface status (1=up, 0=down)", wlanStatus, labelKeysWithRouter, labelVals)
		}

		if _, ok := rec["disabled"]; ok {
			disabled := 0.0
			if rec["disabled"] == trueStr {
				disabled = 1
			}
			mb.GaugeVal(ch, "wlan_disabled", "WLAN interface disabled status", disabled, labelKeysWithRouter, labelVals)
		}

		infoFields := map[string]string{
			"band":              "WLAN frequency band (2g/5g)",
			"country":           "WLAN regulatory domain country",
			"mode":              "WLAN operating mode (ap-bridge/station/etc)",
			"wireless-protocol": "WLAN protocol (802.11a/b/g/n/ac/ax)",
		}

		for key, help := range infoFields {
			if _, ok := rec[key]; ok && rec[key] != "" {
				mb.GaugeVal(ch, "wlan_"+strings.ReplaceAll(key, "-", "_"), help, 1, labelKeysWithRouter, labelVals)
			}
		}

		if _, ok := rec["comment"]; ok && rec["comment"] != "" {
			mb.Info(ch, "wlan_info", "Information about WLAN interface",
				[]string{"name", "ssid", "band", "mode"},
				rec)
		}
	}

	return nil
}
