package collector

import (
	"context"
	"log/slog"

	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/prometheus/client_golang/prometheus"
)

type RoutingStatsCollector struct{}

func NewRoutingStatsCollector() *RoutingStatsCollector { return &RoutingStatsCollector{} }

func (c *RoutingStatsCollector) Name() string { return "routing_stats" }

func (c *RoutingStatsCollector) Describe(_ chan<- *prometheus.Desc) {}

func (c *RoutingStatsCollector) Collect(ctx context.Context, e *entry.RouterEntry, ch chan<- prometheus.Metric) error {
	if !e.ConfigEntry.RoutingStats {
		return nil
	}

	records, err := e.APIConn.Run(ctx, "/routing/stats/process/print")
	if err != nil {
		slog.Error("routing stats collect failed", "router", e.RouterName, "err", err)
		return nil
	}

	mb := NewMetricBuilder(e)
	labelKeys := []string{"id", "pid", "tasks"}

	for _, raw := range records {
		rec := TrimRecord(raw, nil)

		mb.Info(ch, "routing_stats_processes", "Routing Process Stats", labelKeys, rec)

		mb.Gauge(ch, "routing_stats_private_mem", "Private Memory Blocks Used", "private_mem_blocks", labelKeys, rec)
		mb.Gauge(ch, "routing_stats_shared_mem", "Shared Memory Blocks Used", "shared_mem_blocks", labelKeys, rec)

		mb.Counter(ch, "routing_stats_kernel_time", "Process Kernel Time", "kernel_time", labelKeys, rec)
		mb.Counter(ch, "routing_stats_process_time", "Process Time", "process_time", labelKeys, rec)
		mb.Counter(ch, "routing_stats_max_busy", "Max Busy Time", "max_busy", labelKeys, rec)
		mb.Counter(ch, "routing_stats_max_calc", "Max Calc Time", "max_calc", labelKeys, rec)
	}

	return nil
}
