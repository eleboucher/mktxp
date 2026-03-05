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

		metricMap := map[string]struct {
			name       string
			help       string
			parseFloat bool
		}{
			"private_mem_blocks": {"routing_stats_private_mem", "Private Memory Blocks Used", true},
			"shared_mem_blocks":  {"routing_stats_shared_mem", "Shared Memory Blocks Used", true},
			"kernel_time":        {"routing_stats_kernel_time", "Process Kernel Time", true},
			"process_time":       {"routing_stats_process_time", "Process Time", true},
			"max_busy":           {"routing_stats_max_busy", "Max Busy Time", true},
			"max_calc":           {"routing_stats_max_calc", "Max Calc Time", true},
		}

		for key, meta := range metricMap {
			if val, ok := rec[key]; ok && val != "" {
				var value float64
				if meta.parseFloat {
					value = ParseFloat(val)
				} else {
					value = 1
				}
				mb.GaugeVal(ch, meta.name, meta.help, value, labelKeys, labelKeys)
			}
		}
	}

	return nil
}
