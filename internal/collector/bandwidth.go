package collector

import (
	"context"
	"log/slog"

	"github.com/eleboucher/mktxp/internal/config"
	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/prometheus/client_golang/prometheus"
)

type BandwidthCollector struct{}

func NewBandwidthCollector() *BandwidthCollector { return &BandwidthCollector{} }

func (c *BandwidthCollector) Name() string { return "bandwidth" }

func (c *BandwidthCollector) Describe(_ chan<- *prometheus.Desc) {}

func (c *BandwidthCollector) Collect(ctx context.Context, e *entry.RouterEntry, ch chan<- prometheus.Metric) error {
	sysCfg := config.Handler.SystemEntry()
	if !sysCfg.Bandwidth {
		return nil
	}

	records, err := e.APIConn.Run(ctx, "/tool/benchmark/print")
	if err != nil {
		slog.Debug("bandwidth collect failed", "router", e.RouterName, "err", err)
		return nil
	}

	mb := NewMetricBuilder(e)
	labelKeysWithRouter := []string{"router_id"}

	for _, raw := range records {
		rec := TrimRecord(raw, nil)

		if _, ok := rec["cpu"]; ok && rec["cpu"] != "" {
			mb.GaugeVal(ch, "bandwidth_cpu_score", "Bandwidth test CPU score", ParseFloat(rec["cpu"]), labelKeysWithRouter, []string{e.RouterID["router_id"]})
		}

		if _, ok := rec["write"]; ok && rec["write"] != "" {
			mb.GaugeVal(ch, "bandwidth_write_speed", "Bandwidth test write speed in MB/s", ParseFloat(rec["write"]), labelKeysWithRouter, []string{e.RouterID["router_id"]})
		}

		if _, ok := rec["read"]; ok && rec["read"] != "" {
			mb.GaugeVal(ch, "bandwidth_read_speed", "Bandwidth test read speed in MB/s", ParseFloat(rec["read"]), labelKeysWithRouter, []string{e.RouterID["router_id"]})
		}

		if _, ok := rec["latency"]; ok && rec["latency"] != "" {
			mb.GaugeVal(ch, "bandwidth_latency", "Bandwidth test latency in ms", ParseFloat(rec["latency"]), labelKeysWithRouter, []string{e.RouterID["router_id"]})
		}

		if _, ok := rec["comment"]; ok && rec["comment"] != "" {
			mb.Info(ch, "bandwidth_info", "Information about bandwidth test results",
				[]string{"cpu", "write", "read"},
				rec)
		}
	}

	return nil
}
