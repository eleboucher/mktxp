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
		labelKeysWithRouter := append([]string{"router_id"}, labelKeys...)

		mb.GaugeVal(ch, "eoip_status", "EOIP tunnel status (1=running, 0=stopped)", func() float64 {
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
			mb.GaugeVal(ch, "eoip_disabled", "EOIP tunnel disabled status", disabled, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["remote_address"]})
		}

		if _, ok := rec["fast-path"]; ok {
			fastPath := 0.0
			if rec["fast-path"] == "true" {
				fastPath = 1
			}
			mb.GaugeVal(ch, "eoip_fast_path", "EOIP fast path enabled status", fastPath, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["remote_address"]})
		}

		if _, ok := rec["arp"]; ok && rec["arp"] != "" {
			mb.GaugeVal(ch, "eoip_arp", "EOIP ARP configuration", 1.0, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["remote_address"]})
		}

		if _, ok := rec["interface-type"]; ok && rec["interface-type"] != "" {
			mb.GaugeVal(ch, "eoip_interface_type", "EOIP interface type", 1.0, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["remote_address"]})
		}

		if _, ok := rec["mtu"]; ok && rec["mtu"] != "" {
			mb.GaugeVal(ch, "eoip_mtu", "EOIP MTU size in bytes", ParseFloat(rec["mtu"]), labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["remote_address"]})
		}

		if _, ok := rec["comment"]; ok && rec["comment"] != "" {
			mb.Info(ch, "eoip_info", "Information about EOIP tunnel",
				[]string{"name", "remote_address", "interface_type"},
				rec)
		}

		for _, metric := range []struct{ name, key string }{
			{"eoip_remote_address", "remote-address"},
			{"eoip_local_address", "local-address"},
			{"eoip_interface", "interface"},
			{"eoip_bandwidth", "bandwidth"},
		} {
			if val, ok := rec[metric.key]; ok && val != "" {
				mb.GaugeVal(ch, metric.name, "EOIP "+strings.ToUpper(metric.key), 1.0, labelKeysWithRouter, []string{e.RouterID["router_id"], rec["name"], rec["remote_address"]})
			}
		}
	}

	return nil
}
