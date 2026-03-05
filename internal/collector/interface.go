package collector

import (
	"context"
	"fmt"
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
		return fmt.Errorf("interface: %w", err)
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

	metricGauges := map[string]struct {
		name       string
		help       string
		parseFloat bool
	}{
		"running":  {"interface_running", "Current running status of the interface", false},
		"disabled": {"interface_disabled", "Current disabled status of the interface", false},
	}

	metricCounters := map[string]string{
		"rx_byte":    "interface_rx_byte",
		"tx_byte":    "interface_tx_byte",
		"rx_packet":  "interface_rx_packet",
		"tx_packet":  "interface_tx_packet",
		"rx_error":   "interface_rx_error",
		"tx_error":   "interface_tx_error",
		"rx_drop":    "interface_rx_drop",
		"tx_drop":    "interface_tx_drop",
		"link_downs": "link_downs",
	}

	for _, raw := range records {
		record := TrimRecord(raw, wantedKeys)

		formattedName := FormatInterfaceName(record["name"], record["comment"], e.ConfigEntry.InterfaceNameFormat)
		record["name"] = formattedName

		mb.Info(ch, "interface_comment", "The interface comment", []string{"name", "comment"}, record)
		mb.Info(ch, "interface_type", "Interface type like ether, vrrp, eoip, gre-tunnel, ...", []string{"name", "type"}, record)

		labelKeys := []string{"name"}

		for key, meta := range metricGauges {
			if val, ok := record[key]; ok && val != "" {
				var value float64
				if meta.parseFloat {
					value = ParseFloat(val)
				} else {
					value = ParseBool(val)
				}
				mb.GaugeVal(ch, meta.name, meta.help, value, labelKeys, []string{record["name"]})
			}
		}

		for rosKey, metricName := range metricCounters {
			if val, ok := record[rosKey]; ok && val != "" {
				mb.CounterVal(ch, metricName, "Interface counter", ParseFloat(val), labelKeys, []string{record["name"]})
			}
		}
	}

	return nil
}
