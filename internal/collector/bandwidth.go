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
	labelKeysWithRouter := []string{"routerboard_name"}

	for _, raw := range records {
		rec := TrimRecord(raw, nil)

		metricMap := map[string]struct {
			name       string
			help       string
			parseFloat bool
		}{
			"cpu":     {"bandwidth_cpu_score", "Bandwidth test CPU score", true},
			"write":   {"bandwidth_write_speed", "Bandwidth test write speed in MB/s", true},
			"read":    {"bandwidth_read_speed", "Bandwidth test read speed in MB/s", true},
			"latency": {"bandwidth_latency", "Bandwidth test latency in ms", true},
		}

		for key, meta := range metricMap {
			if val, ok := rec[key]; ok && val != "" {
				var value float64
				if meta.parseFloat {
					value = ParseFloat(val)
				} else {
					value = 1
				}
				mb.GaugeVal(ch, meta.name, meta.help, value, labelKeysWithRouter, []string{e.RouterID["routerboard_name"]})
			}
		}

		if _, ok := rec["comment"]; ok && rec["comment"] != "" {
			mb.Info(ch, "bandwidth_info", "Information about bandwidth test results",
				[]string{"cpu", "write", "read"},
				rec)
		}
	}

	return nil
}
