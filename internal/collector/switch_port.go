package collector

import (
	"context"
	"log/slog"
	"strconv"
	"strings"

	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/prometheus/client_golang/prometheus"
)

type SwitchPortCollector struct{}

func NewSwitchPortCollector() *SwitchPortCollector { return &SwitchPortCollector{} }

func (c *SwitchPortCollector) Name() string { return "switch_port" }

func (c *SwitchPortCollector) Describe(_ chan<- *prometheus.Desc) {}

func (c *SwitchPortCollector) Collect(ctx context.Context, e *entry.RouterEntry, ch chan<- prometheus.Metric) error {
	if !e.ConfigEntry.SwitchPort {
		return nil
	}

	records, err := e.APIConn.Run(ctx, "/interface/ethernet/switch/port/print", "=stats=")
	if err != nil {
		slog.Debug("switch port stats collect failed", "router", e.RouterName, "err", err)
		return nil
	}

	mb := NewMetricBuilder(e)
	labelKeys := []string{"name"}

	for _, raw := range records {
		rec := TrimRecord(raw, nil)

		for k, v := range rec {
			if strings.Contains(v, ",") {
				parts := strings.Split(v, ",")
				var total int64
				for _, p := range parts {
					if val, err := strconv.ParseInt(strings.TrimSpace(p), 10, 64); err == nil {
						total += val
					}
				}
				rec[k] = strconv.FormatInt(total, 10)
			}
		}

		metricMap := map[string]struct {
			name       string
			help       string
			parseFloat bool
		}{
			"rx_bytes":         {"switch_rx_bytes", "Total count of received bytes", true},
			"tx_bytes":         {"switch_tx_bytes", "Total count of transmitted bytes", true},
			"rx_packet":        {"switch_rx_packet", "Total count of received packets", true},
			"tx_packet":        {"switch_tx_packet", "Total count of transmitted packets", true},
			"rx_broadcast":     {"switch_rx_broadcast", "Total count of received broadcast frames", true},
			"tx_broadcast":     {"switch_tx_broadcast", "Total count of transmitted broadcast frames", true},
			"rx_multicast":     {"switch_rx_multicast", "Total count of received multicast frames", true},
			"tx_multicast":     {"switch_tx_multicast", "Total count of transmitted multicast frames", true},
			"rx_pause":         {"switch_rx_pause", "Total count of received pause frames", true},
			"tx_pause":         {"switch_tx_pause", "Total count of transmitted pause frames", true},
			"rx_drop":          {"switch_rx_drop", "Total count of received dropped frames", true},
			"tx_drop":          {"switch_tx_drop", "Total count of transmitted dropped frames", true},
			"rx_fcs_error":     {"switch_rx_fcs_error", "Total count of received frames with incorrect checksum", true},
			"rx_align_error":   {"switch_rx_align_error", "Total count of received align error event", true},
			"tx_collision":     {"switch_tx_collision", "Total count of transmitted frames that made collisions", true},
			"rx_fragment":      {"switch_rx_fragment", "Total count of received fragment frames", true},
			"rx_overflow":      {"switch_rx_overflow", "Total count of received overflow frames", true},
			"tx_underrun":      {"switch_tx_underrun", "Total count of transmitted underrun frames", true},
			"tx_deferred":      {"switch_tx_deferred", "Total count of transmitted deferred frames", true},
			"driver_rx_byte":   {"switch_driver_rx_byte", "Total count of received bytes (driver)", true},
			"driver_tx_byte":   {"switch_driver_tx_byte", "Total count of transmitted bytes (driver)", true},
			"driver_rx_packet": {"switch_driver_rx_packet", "Total count of received packets (driver)", true},
			"driver_tx_packet": {"switch_driver_tx_packet", "Total count of transmitted packets (driver)", true},
		}

		labelVals := []string{rec["name"]}

		for key, meta := range metricMap {
			if val, ok := rec[key]; ok && val != "" {
				var value float64
				if meta.parseFloat {
					value = ParseFloat(val)
				} else {
					value = 1
				}
				mb.GaugeVal(ch, meta.name, meta.help, value, labelKeys, labelVals)
			}
		}
	}

	return nil
}
