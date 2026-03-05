package collector

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/prometheus/client_golang/prometheus"
)

// PoolCollector collects IP pool usage metrics from RouterOS.
type PoolCollector struct{}

func NewPoolCollector() *PoolCollector {
	return &PoolCollector{}
}

func (c *PoolCollector) Name() string { return "pool" }

// Describe intentionally sends nothing — unchecked collector with dynamic labels.
func (c *PoolCollector) Describe(_ chan<- *prometheus.Desc) {}

func (c *PoolCollector) Collect(ctx context.Context, e *entry.RouterEntry, ch chan<- prometheus.Metric) error {
	mb := NewMetricBuilder(e)

	if e.ConfigEntry.Pool {
		if err := collectPoolMetrics(ctx, e, mb, ch,
			"/ip/pool/print", "/ip/pool/used/print",
			"ip_pool_used", "Number of used addresses per IP pool (IPv4)",
		); err != nil {
			slog.Error("ip pool collect failed", "router", e.RouterName, "err", err)
			return fmt.Errorf("ipv4 pool: %w", err)
		}
	}

	if e.ConfigEntry.IPv6Pool {
		if err := collectPoolMetrics(ctx, e, mb, ch,
			"/ipv6/pool/print", "/ipv6/pool/used/print",
			"ip_pool_used_ipv6", "Number of used addresses per IP pool (IPv6)",
		); err != nil {
			slog.Error("ipv6 pool collect failed", "router", e.RouterName, "err", err)
			return fmt.Errorf("ipv6 pool: %w", err)
		}
	}

	return nil
}

// collectPoolMetrics fetches pool names and used counts, then emits a gauge per pool.
func collectPoolMetrics(
	ctx context.Context,
	e *entry.RouterEntry,
	mb *MetricBuilder,
	ch chan<- prometheus.Metric,
	poolListAPI, poolUsedAPI string,
	metricName, helpText string,
) error {
	poolRecords, err := e.APIConn.Run(ctx, poolListAPI, "=.proplist=name")
	if err != nil {
		return err
	}

	poolCounts := make(map[string]float64, len(poolRecords))
	for _, raw := range poolRecords {
		record := TrimRecord(raw, []string{"name"})
		poolCounts[record["name"]] = 0
	}

	usedRecords, err := e.APIConn.Run(ctx, poolUsedAPI, "=.proplist=pool")
	if err != nil {
		return err
	}
	for _, raw := range usedRecords {
		record := TrimRecord(raw, []string{"pool"})
		pool := record["pool"]
		poolCounts[pool]++
	}

	metricMap := map[string]struct {
		name       string
		help       string
		parseFloat bool
	}{
		"count": {metricName, helpText, true},
	}

	for pool, count := range poolCounts {
		rec := map[string]string{"count": strconv.FormatFloat(count, 'f', 0, 64)}
		labelKeys := []string{"pool"}
		labelVals := []string{pool}

		for key, meta := range metricMap {
			if val, ok := rec[key]; ok && val != "" {
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
