package collector

import (
	"context"
	"log/slog"

	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/prometheus/client_golang/prometheus"
)

// InterfaceCollector collects interface traffic metrics from RouterOS.
type InterfaceCollector struct{}

func NewInterfaceCollector() *InterfaceCollector {
	return &InterfaceCollector{}
}

func (c *InterfaceCollector) Name() string { return "interface" }

// Describe intentionally sends nothing — unchecked collector with dynamic labels.
func (c *InterfaceCollector) Describe(_ chan<- *prometheus.Desc) {}

func (c *InterfaceCollector) Collect(ctx context.Context, e *entry.RouterEntry, ch chan<- prometheus.Metric) error {
	if !e.ConfigEntry.Interface {
		return nil
	}

	records, err := e.APIConn.Run(ctx,
		"/interface/print",
		"=.proplist=name,disabled,running,rx-byte,tx-byte,rx-packet,tx-packet,rx-error,tx-error,rx-drop,tx-drop,link-downs,comment,type",
	)
	if err != nil {
		slog.Error("interface collect failed", "router", e.RouterName, "err", err)
		return nil
	}

	mb := NewMetricBuilder(e)

	wantedKeys := []string{
		"name", "disabled", "running",
		"rx_byte", "tx_byte",
		"rx_packet", "tx_packet",
		"rx_error", "tx_error",
		"rx_drop", "tx_drop",
		"link_downs", "comment", "type",
	}

	for _, raw := range records {
		record := TrimRecord(raw, wantedKeys)

		// Apply interface name format per config.
		formattedName := FormatInterfaceName(record["name"], record["comment"], e.ConfigEntry.InterfaceNameFormat)
		record["name"] = formattedName

		mb.Info(ch, "interface_comment", "The interface comment", []string{"name", "comment"}, record)
		mb.Info(ch, "interface_type", "Interface type like ether, vrrp, eoip, gre-tunnel, ...", []string{"name", "type"}, record)

		mb.GaugeVal(ch, "interface_running", "Current running status of the interface",
			ParseBool(record["running"]),
			[]string{"name"}, []string{record["name"]},
		)
		mb.GaugeVal(ch, "interface_disabled", "Current disabled status of the interface",
			ParseBool(record["disabled"]),
			[]string{"name"}, []string{record["name"]},
		)

		mb.Counter(ch, "interface_rx_byte", "Number of received bytes", "rx_byte", []string{"name"}, record)
		mb.Counter(ch, "interface_tx_byte", "Number of transmitted bytes", "tx_byte", []string{"name"}, record)
		mb.Counter(ch, "interface_rx_packet", "Number of packets received", "rx_packet", []string{"name"}, record)
		mb.Counter(ch, "interface_tx_packet", "Number of transmitted packets", "tx_packet", []string{"name"}, record)
		mb.Counter(ch, "interface_rx_error", "Number of packets received with an error", "rx_error", []string{"name"}, record)
		mb.Counter(ch, "interface_tx_error", "Number of packets transmitted with an error", "tx_error", []string{"name"}, record)
		mb.Counter(ch, "interface_rx_drop", "Number of received packets being dropped", "rx_drop", []string{"name"}, record)
		mb.Counter(ch, "interface_tx_drop", "Number of transmitted packets being dropped", "tx_drop", []string{"name"}, record)
		mb.Counter(ch, "link_downs", "Number of times link went down", "link_downs", []string{"name"}, record)
	}

	return nil
}
