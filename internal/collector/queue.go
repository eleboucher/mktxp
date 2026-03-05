package collector

import (
	"context"
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
	}

	if err := c.collectSimple(ctx, e, mb, ch); err != nil {
		slog.Error("queue simple collect failed", "router", e.RouterName, "err", err)
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

	metricMap := map[string]struct {
		name       string
		help       string
		parseFloat bool
	}{
		"bytes":        {"queue_tree_bytes", "Number of processed bytes", true},
		"queued_bytes": {"queue_tree_queued_bytes", "Number of queued bytes", true},
		"dropped":      {"queue_tree_dropped", "Number of dropped bytes", true},
		"rate":         {"queue_tree_rates", "Average passing data rate (bytes/s)", true},
	}

	for _, raw := range records {
		rec := TrimRecord(raw, keys)
		labelKeys := []string{"name"}
		labelVals := []string{rec["name"]}

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

		metricMapUpload := map[string]struct {
			name       string
			help       string
			parseFloat bool
		}{
			"rate_up":         {"queue_simple_rates_upload", "Average upload data rate (bytes/s)", true},
			"bytes_up":        {"queue_simple_bytes_upload", "Upload processed bytes", true},
			"queued_bytes_up": {"queue_simple_queued_bytes_upload", "Upload queued bytes", true},
			"dropped_up":      {"queue_simple_dropped_upload", "Upload dropped bytes", true},
		}

		metricMapDownload := map[string]struct {
			name       string
			help       string
			parseFloat bool
		}{
			"rate_down":         {"queue_simple_rates_download", "Average download data rate (bytes/s)", true},
			"bytes_down":        {"queue_simple_bytes_download", "Download processed bytes", true},
			"queued_bytes_down": {"queue_simple_queued_bytes_download", "Download queued bytes", true},
			"dropped_down":      {"queue_simple_dropped_download", "Download dropped bytes", true},
		}

		labelKeys := []string{"name"}

		for key, meta := range metricMapUpload {
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

		for key, meta := range metricMapDownload {
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
