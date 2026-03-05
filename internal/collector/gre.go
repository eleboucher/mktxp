package collector

import (
	"context"
	"log/slog"

	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/prometheus/client_golang/prometheus"
)

type GRECollector struct{}

func NewGRECollector() *GRECollector { return &GRECollector{} }

func (c *GRECollector) Name() string { return "gre" }

func (c *GRECollector) Describe(_ chan<- *prometheus.Desc) {}

func (c *GRECollector) Collect(ctx context.Context, e *entry.RouterEntry, ch chan<- prometheus.Metric) error {
	if !e.ConfigEntry.GRE {
		return nil
	}

	records, err := e.APIConn.Run(ctx, "/interface/gre/print")
	if err != nil {
		slog.Debug("gre collect failed", "router", e.RouterName, "err", err)
		return nil
	}

	mb := NewMetricBuilder(e)
	labelKeys := []string{"name", "remote_address"}
	labelKeysWithRouter := append([]string{"routerboard_name"}, labelKeys...)

	metricMap := map[string]struct {
		name       string
		help       string
		parseFloat bool
	}{
		"disabled":       {"gre_disabled", "GRE tunnel disabled status", false},
		"fast-path":      {"gre_fast_path", "GRE fast path enabled status", false},
		"arp":            {"gre_arp", "GRE ARP configuration", false},
		"mtu":            {"gre_mtu", "GRE MTU size in bytes", true},
		"ttl":            {"gre_ttl", "GRE TTL value", true},
		"dscp":           {"gre_dscp", "GRE DSCP value", true},
		"remote-address": {"gre_remote_address", "GRE REMOTE-ADDRESS", false},
		"local-address":  {"gre_local_address", "GRE LOCAL-ADDRESS", false},
		"interface":      {"gre_interface", "GRE INTERFACE", false},
		"mpls-ttl":       {"gre_mpls_ttl", "GRE MPLS-TTL", false},
		"mpls-label":     {"gre_mpls_label", "GRE MPLS-LABEL", false},
	}

	for _, raw := range records {
		rec := TrimRecord(raw, nil)
		if rec["name"] == "" {
			continue
		}

		rec["name"] = FormatInterfaceName(rec["name"], "", e.ConfigEntry.InterfaceNameFormat)

		if rec["running"] == trueStr && rec["disabled"] != trueStr {
			mb.GaugeVal(ch, "gre_status", "GRE tunnel status (1=running, 0=stopped)", 1.0, labelKeysWithRouter, []string{e.RouterID["routerboard_name"], rec["name"], rec["remote_address"]})
		} else {
			mb.GaugeVal(ch, "gre_status", "GRE tunnel status (1=running, 0=stopped)", 0.0, labelKeysWithRouter, []string{e.RouterID["routerboard_name"], rec["name"], rec["remote_address"]})
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
				mb.GaugeVal(ch, metric.name, metric.help, metricValue, labelKeysWithRouter, []string{e.RouterID["routerboard_name"], rec["name"], rec["remote_address"]})
			}
		}

		if _, ok := rec["comment"]; ok && rec["comment"] != "" {
			mb.Info(ch, "gre_info", "Information about GRE tunnel",
				[]string{"name", "remote_address"},
				rec)
		}
	}

	return nil
}
