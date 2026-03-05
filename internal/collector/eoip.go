package collector

import (
	"context"
	"log/slog"
	"strings"

	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/prometheus/client_golang/prometheus"
)

type EOIPCollector struct{}

func NewEOIPCollector() *EOIPCollector { return &EOIPCollector{} }

func (c *EOIPCollector) Name() string { return "eoip" }

func (c *EOIPCollector) Describe(_ chan<- *prometheus.Desc) {}

func (c *EOIPCollector) Collect(ctx context.Context, e *entry.RouterEntry, ch chan<- prometheus.Metric) error {
	if !e.ConfigEntry.EOIP {
		return nil
	}

	records, err := e.APIConn.Run(ctx, "/interface/eoip/print")
	if err != nil {
		slog.Debug("eoip collect failed", "router", e.RouterName, "err", err)
		return nil
	}

	mb := NewMetricBuilder(e)
	labelKeys := []string{"name", "remote_address"}

	for _, raw := range records {
		rec := TrimRecord(raw, nil)
		if rec["name"] == "" {
			continue
		}

		rec["name"] = FormatInterfaceName(rec["name"], "", e.ConfigEntry.InterfaceNameFormat)
		labelKeysWithRouter := append([]string{"routerboard_name"}, labelKeys...)
		labelVals := []string{e.RouterID["routerboard_name"], rec["name"], rec["remote_address"]}

		metricMap := map[string]struct {
			name       string
			help       string
			parseFloat bool
		}{
			"remote-address": {"eoip_remote_address", "EOIP remote address", false},
			"local-address":  {"eoip_local_address", "EOIP local address", false},
			"interface":      {"eoip_interface", "EOIP interface", false},
			"bandwidth":      {"eoip_bandwidth", "EOIP bandwidth", true},
			"mtu":            {"eoip_mtu", "EOIP MTU size in bytes", true},
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

		if rec["running"] == trueStr && rec["disabled"] != trueStr {
			mb.GaugeVal(ch, "eoip_status", "EOIP tunnel status (1=running, 0=stopped)", 1, labelKeysWithRouter, labelVals)
		} else {
			mb.GaugeVal(ch, "eoip_status", "EOIP tunnel status (1=running, 0=stopped)", 0, labelKeysWithRouter, labelVals)
		}

		if disabledVal, ok := rec["disabled"]; ok {
			disabled := 0.0
			if disabledVal == trueStr {
				disabled = 1
			}
			mb.GaugeVal(ch, "eoip_disabled", "EOIP tunnel disabled status", disabled, labelKeysWithRouter, labelVals)
		}

		if fastPathVal, ok := rec["fast-path"]; ok {
			fastPath := 0.0
			if fastPathVal == trueStr {
				fastPath = 1
			}
			mb.GaugeVal(ch, "eoip_fast_path", "EOIP fast path enabled status", fastPath, labelKeysWithRouter, labelVals)
		}

		infoFields := map[string]string{
			"arp":            "EOIP ARP configuration",
			"interface-type": "EOIP interface type",
		}

		for key, help := range infoFields {
			if _, ok := rec[key]; ok && rec[key] != "" {
				mb.GaugeVal(ch, "eoip_"+strings.ReplaceAll(key, "-", "_"), help, 1, labelKeysWithRouter, labelVals)
			}
		}

		if comment, ok := rec["comment"]; ok && comment != "" {
			mb.Info(ch, "eoip_info", "Information about EOIP tunnel",
				[]string{"name", "remote_address", "interface_type"},
				rec)
		}
	}

	return nil
}
