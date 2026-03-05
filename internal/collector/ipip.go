package collector

import (
	"context"
	"log/slog"

	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/prometheus/client_golang/prometheus"
)

type IPIPCollector struct{}

func NewIPIPCollector() *IPIPCollector { return &IPIPCollector{} }

func (c *IPIPCollector) Name() string { return "ipip" }

func (c *IPIPCollector) Describe(_ chan<- *prometheus.Desc) {}

func (c *IPIPCollector) Collect(ctx context.Context, e *entry.RouterEntry, ch chan<- prometheus.Metric) error {
	if !e.ConfigEntry.IPIP {
		return nil
	}

	records, err := e.APIConn.Run(ctx, "/interface/ipip/print")
	if err != nil {
		slog.Debug("ipip collect failed", "router", e.RouterName, "err", err)
		return nil
	}

	mb := NewMetricBuilder(e)
	labelKeys := []string{"name", "remote_address"}
	labelKeysWithRouter := append([]string{"router_id"}, labelKeys...)

	metricMap := map[string]struct {
		name       string
		help       string
		parseFloat bool
	}{
		"disabled":       {"ipip_disabled", "IPIP tunnel disabled status", false},
		"fast-path":      {"ipip_fast_path", "IPIP fast path enabled status", false},
		"arp":            {"ipip_arp", "IPIP ARP configuration", false},
		"mtu":            {"ipip_mtu", "IPIP MTU size in bytes", true},
		"ttl":            {"ipip_ttl", "IPIP TTL value", true},
		"dscp":           {"ipip_dscp", "IPIP DSCP value", true},
		"remote-address": {"ipip_remote_address", "IPIP REMOTE-ADDRESS", false},
		"local-address":  {"ipip_local_address", "IPIP LOCAL-ADDRESS", false},
		"interface":      {"ipip_interface", "IPIP INTERFACE", false},
		"mpls-ttl":       {"ipip_mpls_ttl", "IPIP MPLS-TTL", false},
	}

	for _, raw := range records {
		rec := TrimRecord(raw, nil)
		if rec["name"] == "" {
			continue
		}

		rec["name"] = FormatInterfaceName(rec["name"], "", e.ConfigEntry.InterfaceNameFormat)

		if rec["running"] == trueStr && rec["disabled"] != trueStr {
			mb.GaugeVal(ch, "ipip_status", "IPIP tunnel status (1=running, 0=stopped)", 1.0, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["remote_address"]})
		} else {
			mb.GaugeVal(ch, "ipip_status", "IPIP tunnel status (1=running, 0=stopped)", 0.0, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["remote_address"]})
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
				mb.GaugeVal(ch, metric.name, metric.help, metricValue, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["remote_address"]})
			}
		}

		if _, ok := rec["comment"]; ok && rec["comment"] != "" {
			mb.Info(ch, "ipip_info", "Information about IPIP tunnel",
				[]string{"name", "remote_address"},
				rec)
		}
	}

	return nil
}
