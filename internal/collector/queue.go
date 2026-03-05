package collector

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/prometheus/client_golang/prometheus"
)

type QueueCollector struct{}

func NewQueueCollector() *QueueCollector                     { return &QueueCollector{} }
func (c *QueueCollector) Name() string                       { return "queue" }
func (c *QueueCollector) Describe(_ chan<- *prometheus.Desc) {}

func (c *QueueCollector) Collect(ctx context.Context, e *entry.RouterEntry, ch chan<- prometheus.Metric) error {
	if !e.ConfigEntry.Queue {
		return nil
	}

	mb := NewMetricBuilder(e)

	if err := c.collectTree(ctx, e, mb, ch); err != nil {
		slog.Error("queue tree collect failed", "router", e.RouterName, "err", err)
		return fmt.Errorf("queue tree: %w", err)
	}

	if err := c.collectSimple(ctx, e, mb, ch); err != nil {
		slog.Error("queue simple collect failed", "router", e.RouterName, "err", err)
		return fmt.Errorf("queue simple: %w", err)
	}

	return nil
}

func (c *QueueCollector) collectTree(ctx context.Context, e *entry.RouterEntry, mb *MetricBuilder, ch chan<- prometheus.Metric) error {
	records, err := e.APIConn.Run(ctx, "/queue/tree/print",
		"=.proplist=name,parent,packet-mark,limit-at,max-limit,priority,bytes,queued-bytes,dropped,rate,disabled")
	if err != nil {
		return err
	}

	keys := []string{"name", "bytes", "queued_bytes", "dropped", "rate"}

	metricCounters := map[string]string{
		"bytes":        "queue_tree_bytes",
		"queued_bytes": "queue_tree_queued_bytes",
		"dropped":      "queue_tree_dropped",
	}

	metricGauges := map[string]struct {
		name       string
		help       string
		parseFloat bool
	}{
		"rate": {"queue_tree_rates", "Average passing data rate (bytes/s)", true},
	}

	for _, raw := range records {
		rec := TrimRecord(raw, keys)
		labelKeys := []string{"name"}
		labelVals := []string{rec["name"]}

		for rosKey, metricName := range metricCounters {
			if val, ok := rec[rosKey]; ok && val != "" {
				mb.CounterVal(ch, metricName, "Number of processed bytes", ParseFloat(val), labelKeys, labelVals)
			}
		}

		for key, meta := range metricGauges {
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

func (c *QueueCollector) collectSimple(ctx context.Context, e *entry.RouterEntry, mb *MetricBuilder, ch chan<- prometheus.Metric) error {
	records, err := e.APIConn.Run(ctx, "/queue/simple/print",
		"=.proplist=name,parent,packet-mark,limit-at,max-limit,priority,bytes,packets,queued-bytes,queued-packets,dropped,rate,packet-rate,disabled")
	if err != nil {
		return err
	}

	for _, raw := range records {
		rec := TrimRecord(raw, nil)
		name := rec["name"]

		split := splitSimpleQueue(rec)
		split["name"] = name

		metricCountersUpload := map[string]string{
			"bytes_up":        "queue_simple_bytes_upload",
			"queued_bytes_up": "queue_simple_queued_bytes_upload",
			"dropped_up":      "queue_simple_dropped_upload",
		}

		metricGaugesUpload := map[string]struct {
			name       string
			help       string
			parseFloat bool
		}{
			"rate_up": {"queue_simple_rates_upload", "Average upload data rate (bytes/s)", true},
		}

		metricCountersDownload := map[string]string{
			"bytes_down":        "queue_simple_bytes_download",
			"queued_bytes_down": "queue_simple_queued_bytes_download",
			"dropped_down":      "queue_simple_dropped_download",
		}

		metricGaugesDownload := map[string]struct {
			name       string
			help       string
			parseFloat bool
		}{
			"rate_down": {"queue_simple_rates_download", "Average download data rate (bytes/s)", true},
		}

		labelKeys := []string{"name"}

		for rosKey, metricName := range metricCountersUpload {
			if val, ok := split[rosKey]; ok && val != "" {
				mb.CounterVal(ch, metricName, "Upload processed bytes", ParseFloat(val), labelKeys, []string{name})
			}
		}

		for key, meta := range metricGaugesUpload {
			if val, ok := split[key]; ok && val != "" {
				var value float64
				if meta.parseFloat {
					value = ParseFloat(val)
				} else {
					value = 1.0
				}
				mb.GaugeVal(ch, meta.name, meta.help, value, labelKeys, []string{name})
			}
		}

		for rosKey, metricName := range metricCountersDownload {
			if val, ok := split[rosKey]; ok && val != "" {
				mb.CounterVal(ch, metricName, "Download processed bytes", ParseFloat(val), labelKeys, []string{name})
			}
		}

		for key, meta := range metricGaugesDownload {
			if val, ok := split[key]; ok && val != "" {
				var value float64
				if meta.parseFloat {
					value = ParseFloat(val)
				} else {
					value = 1.0
				}
				mb.GaugeVal(ch, meta.name, meta.help, value, labelKeys, []string{name})
			}
		}
	}
	return nil
}

func splitSimpleQueue(rec map[string]string) map[string]string {
	out := make(map[string]string, len(rec)*2)
	for k, v := range rec {
		parts := strings.SplitN(v, "/", 2)
		if len(parts) == 2 {
			out[k+"_up"] = parts[0]
			out[k+"_down"] = parts[1]
		} else {
			out[k] = v
		}
	}
	return out
}
