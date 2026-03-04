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

	// Queue tree
	if err := c.collectTree(ctx, e, mb, ch); err != nil {
		slog.Error("queue tree collect failed", "router", e.RouterName, "err", err)
	}

	// Simple queue
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
	for _, raw := range records {
		rec := TrimRecord(raw, keys)
		mb.Counter(ch, "queue_tree_rates", "Average passing data rate (bytes/s)", "rate", []string{"name"}, rec)
		mb.Counter(ch, "queue_tree_bytes", "Number of processed bytes", "bytes", []string{"name"}, rec)
		mb.Counter(ch, "queue_tree_queued_bytes", "Number of queued bytes", "queued_bytes", []string{"name"}, rec)
		mb.Counter(ch, "queue_tree_dropped", "Number of dropped bytes", "dropped", []string{"name"}, rec)
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

		// Simple queue fields are "upload/download" — split them.
		split := splitSimpleQueue(rec)
		split["name"] = name

		mb.Counter(ch, "queue_simple_rates_upload", "Average upload data rate (bytes/s)", "rate_up", []string{"name"}, split)
		mb.Counter(ch, "queue_simple_rates_download", "Average download data rate (bytes/s)", "rate_down", []string{"name"}, split)
		mb.Counter(ch, "queue_simple_bytes_upload", "Upload processed bytes", "bytes_up", []string{"name"}, split)
		mb.Counter(ch, "queue_simple_bytes_download", "Download processed bytes", "bytes_down", []string{"name"}, split)
		mb.Counter(ch, "queue_simple_queued_bytes_upload", "Upload queued bytes", "queued_bytes_up", []string{"name"}, split)
		mb.Counter(ch, "queue_simple_queued_bytes_download", "Download queued bytes", "queued_bytes_down", []string{"name"}, split)
		mb.Counter(ch, "queue_simple_dropped_upload", "Upload dropped bytes", "dropped_up", []string{"name"}, split)
		mb.Counter(ch, "queue_simple_dropped_download", "Download dropped bytes", "dropped_down", []string{"name"}, split)
	}
	return nil
}

// splitSimpleQueue expands "value_up/value_down" RouterOS fields into separate keys.
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
