package collector

import (
	"context"
	"log/slog"
	"strings"

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

	for _, raw := range records {
		rec := TrimRecord(raw, nil)
		if rec["name"] == "" {
			continue
		}

		rec["name"] = FormatInterfaceName(rec["name"], "", e.ConfigEntry.InterfaceNameFormat)
		labelKeysWithRouter := append([]string{"router_id"}, labelKeys...)

		mb.GaugeVal(ch, "gre_status", "GRE tunnel status (1=running, 0=stopped)", func() float64 {
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
			mb.GaugeVal(ch, "gre_disabled", "GRE tunnel disabled status", disabled, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["remote_address"]})
		}

		if _, ok := rec["fast-path"]; ok {
			fastPath := 0.0
			if rec["fast-path"] == "true" {
				fastPath = 1
			}
			mb.GaugeVal(ch, "gre_fast_path", "GRE fast path enabled status", fastPath, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["remote_address"]})
		}

		if _, ok := rec["arp"]; ok && rec["arp"] != "" {
			mb.GaugeVal(ch, "gre_arp", "GRE ARP configuration", 1.0, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["remote_address"]})
		}

		if _, ok := rec["mtu"]; ok && rec["mtu"] != "" {
			mb.GaugeVal(ch, "gre_mtu", "GRE MTU size in bytes", ParseFloat(rec["mtu"]), labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["remote_address"]})
		}

		if _, ok := rec["ttl"]; ok && rec["ttl"] != "" {
			mb.GaugeVal(ch, "gre_ttl", "GRE TTL value", ParseFloat(rec["ttl"]), labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["remote_address"]})
		}

		if _, ok := rec["dscp"]; ok && rec["dscp"] != "" {
			mb.GaugeVal(ch, "gre_dscp", "GRE DSCP value", ParseFloat(rec["dscp"]), labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["remote_address"]})
		}

		if _, ok := rec["comment"]; ok && rec["comment"] != "" {
			mb.Info(ch, "gre_info", "Information about GRE tunnel",
				[]string{"name", "remote_address"},
				rec)
		}

		for _, metric := range []struct{ name, key string }{
			{"gre_remote_address", "remote-address"},
			{"gre_local_address", "local-address"},
			{"gre_interface", "interface"},
			{"gre_mpls_ttl", "mpls-ttl"},
			{"gre_mpls_label", "mpls-label"},
		} {
			if val, ok := rec[metric.key]; ok && val != "" {
				mb.GaugeVal(ch, metric.name, "GRE "+strings.ToUpper(metric.key), 1.0, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["remote_address"]})
			}
		}
	}

	return nil
}
