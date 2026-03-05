package collector

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/prometheus/client_golang/prometheus"
)

// FirewallCollector collects firewall rule byte-counter metrics from RouterOS.
type FirewallCollector struct{}

func NewFirewallCollector() *FirewallCollector {
	return &FirewallCollector{}
}

func (c *FirewallCollector) Name() string { return "firewall" }

// Describe intentionally sends nothing — unchecked collector with dynamic labels.
func (c *FirewallCollector) Describe(_ chan<- *prometheus.Desc) {}

type firewallChain struct {
	name   string
	metric string
	help   string
}

var firewallChains = []firewallChain{
	{"filter", "firewall_filter", "Total amount of bytes matched by firewall rules"},
	{"raw", "firewall_raw", "Total amount of bytes matched by raw firewall rules"},
	{"nat", "firewall_nat", "Total amount of bytes matched by NAT rules"},
	{"mangle", "firewall_mangle", "Total amount of bytes matched by Mangle rules"},
}

func (c *FirewallCollector) Collect(ctx context.Context, e *entry.RouterEntry, ch chan<- prometheus.Metric) error {
	mb := NewMetricBuilder(e)

	if e.ConfigEntry.Firewall {
		for _, chain := range firewallChains {
			if err := collectFirewallChain(ctx, e, mb, ch, "/ip/firewall/"+chain.name+"/print", chain.metric, chain.help); err != nil {
				slog.Error("firewall collect failed", "router", e.RouterName, "chain", chain.name, "err", err)
				return fmt.Errorf("ipv4 firewall %s: %w", chain.name, err)
			}
		}
	}

	if e.ConfigEntry.IPv6Firewall {
		for _, chain := range firewallChains {
			if err := collectFirewallChain(ctx, e, mb, ch, "/ipv6/firewall/"+chain.name+"/print", chain.metric+"_ipv6", chain.help+" (IPv6)"); err != nil {
				slog.Error("ipv6 firewall collect failed", "router", e.RouterName, "chain", chain.name, "err", err)
				return fmt.Errorf("ipv6 firewall %s: %w", chain.name, err)
			}
		}
	}

	return nil
}

func collectFirewallChain(
	ctx context.Context,
	e *entry.RouterEntry,
	mb *MetricBuilder,
	ch chan<- prometheus.Metric,
	api, metricName, helpText string,
) error {
	records, err := e.APIConn.Run(
		ctx,
		api,
		"=.proplist=chain,action,bytes,comment,log,out-interface,protocol",
	)
	if err != nil {
		return err
	}

	wantedKeys := []string{"chain", "action", "bytes", "comment", "log", "out_interface", "protocol"}

	metricMap := map[string]struct {
		name       string
		help       string
		parseFloat bool
	}{
		"bytes": {metricName, helpText, true},
	}

	for _, raw := range records {
		record := TrimRecord(raw, wantedKeys)

		ruleName := fmt.Sprintf("| %s | %s | %s", record["chain"], record["action"], record["comment"])
		if record["out_interface"] != "" {
			ruleName += fmt.Sprintf(" | %s", record["out_interface"])
		}
		if record["protocol"] != "" {
			ruleName += fmt.Sprintf(" | %s", record["protocol"])
		}

		labelKeys := []string{"name", "log"}
		labelVals := []string{ruleName, record["log"]}

		for key, meta := range metricMap {
			if val, ok := record[key]; ok && val != "" {
				var value float64
				if meta.parseFloat {
					value = ParseFloat(val)
				} else {
					value = 1.0
				}
				mb.GaugeVal(ch, meta.name, meta.help, value, labelKeys, labelVals)
			}
		}
	}

	return nil
}
