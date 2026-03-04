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

		// Sum each CPU lane
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

		// Standard Counters
		mb.Counter(ch, "switch_rx_bytes", "Total count of received bytes", "rx_bytes", labelKeys, rec)
		mb.Counter(ch, "switch_tx_bytes", "Total count of transmitted bytes", "tx_bytes", labelKeys, rec)
		mb.Counter(ch, "switch_rx_packet", "Total count of received packets", "rx_packet", labelKeys, rec)
		mb.Counter(ch, "switch_tx_packet", "Total count of transmitted packets", "tx_packet", labelKeys, rec)

		// Broadcast/Multicast/Pause
		mb.Counter(ch, "switch_rx_broadcast", "Total count of received broadcast frames", "rx_broadcast", labelKeys, rec)
		mb.Counter(ch, "switch_tx_broadcast", "Total count of transmitted broadcast frames", "tx_broadcast", labelKeys, rec)
		mb.Counter(ch, "switch_rx_multicast", "Total count of received multicast frames", "rx_multicast", labelKeys, rec)
		mb.Counter(ch, "switch_tx_multicast", "Total count of transmitted multicast frames", "tx_multicast", labelKeys, rec)
		mb.Counter(ch, "switch_rx_pause", "Total count of received pause frames", "rx_pause", labelKeys, rec)
		mb.Counter(ch, "switch_tx_pause", "Total count of transmitted pause frames", "tx_pause", labelKeys, rec)

		// Errors & Drops
		mb.Counter(ch, "switch_rx_drop", "Total count of received dropped frames", "rx_drop", labelKeys, rec)
		mb.Counter(ch, "switch_tx_drop", "Total count of transmitted dropped frames", "tx_drop", labelKeys, rec)
		mb.Counter(ch, "switch_rx_fcs_error", "Total count of received frames with incorrect checksum", "rx_fcs_error", labelKeys, rec)
		mb.Counter(ch, "switch_rx_align_error", "Total count of received align error event", "rx_align_error", labelKeys, rec)
		mb.Counter(ch, "switch_tx_collision", "Total count of transmitted frames that made collisions", "tx_collision", labelKeys, rec)

		// Driver Stats (often prefixed with 'driver' in Python output)
		// If keys exist, we map them explicitly to match Python metric names
		if _, ok := rec["driver_rx_byte"]; ok {
			mb.Counter(ch, "switch_driver_rx_byte", "Total count of received bytes (driver)", "driver_rx_byte", labelKeys, rec)
			mb.Counter(ch, "switch_driver_tx_byte", "Total count of transmitted bytes (driver)", "driver_tx_byte", labelKeys, rec)
			mb.Counter(ch, "switch_driver_rx_packet", "Total count of received packets (driver)", "driver_rx_packet", labelKeys, rec)
			mb.Counter(ch, "switch_driver_tx_packet", "Total count of transmitted packets (driver)", "driver_tx_packet", labelKeys, rec)
		}
	}

	return nil
}
