package collector

import (
	"context"
	"log/slog"
	"strings"

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

	for _, raw := range records {
		rec := TrimRecord(raw, nil)
		if rec["name"] == "" {
			continue
		}

		rec["name"] = FormatInterfaceName(rec["name"], "", e.ConfigEntry.InterfaceNameFormat)
		labelKeysWithRouter := append([]string{"router_id"}, labelKeys...)

		mb.GaugeVal(ch, "ipip_status", "IPIP tunnel status (1=running, 0=stopped)", func() float64 {
			if rec["running"] == "true" && rec["disabled"] != "true" {
				return 1
			}
			return 0
		}(), labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["remote_address"]})

		if _, ok := rec["disabled"]; ok {
			disabled := 0.0
			if rec["disabled"] == "true" {
				disabled = 1
			}
			mb.GaugeVal(ch, "ipip_disabled", "IPIP tunnel disabled status", disabled, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["remote_address"]})
		}

		if _, ok := rec["fast-path"]; ok {
			fastPath := 0.0
			if rec["fast-path"] == "true" {
				fastPath = 1
			}
			mb.GaugeVal(ch, "ipip_fast_path", "IPIP fast path enabled status", fastPath, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["remote_address"]})
		}

		if _, ok := rec["arp"]; ok && rec["arp"] != "" {
			mb.GaugeVal(ch, "ipip_arp", "IPIP ARP configuration", 1.0, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["remote_address"]})
		}

		if _, ok := rec["mtu"]; ok && rec["mtu"] != "" {
			mb.GaugeVal(ch, "ipip_mtu", "IPIP MTU size in bytes", ParseFloat(rec["mtu"]), labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["remote_address"]})
		}

		if _, ok := rec["ttl"]; ok && rec["ttl"] != "" {
			mb.GaugeVal(ch, "ipip_ttl", "IPIP TTL value", ParseFloat(rec["ttl"]), labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["remote_address"]})
		}

		if _, ok := rec["dscp"]; ok && rec["dscp"] != "" {
			mb.GaugeVal(ch, "ipip_dscp", "IPIP DSCP value", ParseFloat(rec["dscp"]), labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["remote_address"]})
		}

		if _, ok := rec["comment"]; ok && rec["comment"] != "" {
			mb.Info(ch, "ipip_info", "Information about IPIP tunnel",
				[]string{"name", "remote_address"},
				rec)
		}

		for _, metric := range []struct{ name, key string }{
			{"ipip_remote_address", "remote-address"},
			{"ipip_local_address", "local-address"},
			{"ipip_interface", "interface"},
			{"ipip_ttl", "ttl"},
			{"ipip_mpls_ttl", "mpls-ttl"},
		} {
			if val, ok := rec[metric.key]; ok && val != "" {
				mb.GaugeVal(ch, metric.name, "IPIP "+strings.ToUpper(metric.key), 1.0, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["remote_address"]})
			}
		}
	}

	return nil
}
