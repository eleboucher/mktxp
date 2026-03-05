package collector

import (
	"context"
	"log/slog"

	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/prometheus/client_golang/prometheus"
)

type DNSCollector struct{}

func NewDNSCollector() *DNSCollector { return &DNSCollector{} }

func (c *DNSCollector) Name() string { return "dns" }

func (c *DNSCollector) Describe(_ chan<- *prometheus.Desc) {}

func (c *DNSCollector) Collect(ctx context.Context, e *entry.RouterEntry, ch chan<- prometheus.Metric) error {
	if !e.ConfigEntry.DNS {
		return nil
	}

	records, err := e.APIConn.Run(ctx, "/ip/dns/print", "=.proplist=cache-size,cache-used")
	if err != nil {
		slog.Error("dns collect failed", "router", e.RouterName, "err", err)
		return nil
	}

	if len(records) == 0 {
		return nil
	}

	mb := NewMetricBuilder(e)
	rec := TrimRecord(records[0], nil)

	metricMap := map[string]struct {
		name       string
		help       string
		parseFloat bool
	}{
		"cache_size": {"dns_info", "DNS cache size in bytes", true},
		"cache_used": {"dns_info", "DNS cache usage in bytes", true},
	}

	for key, meta := range metricMap {
		if val, ok := rec[key]; ok && val != "" {
			var value float64
			if meta.parseFloat {
				value = ParseFloat(val) * 1024
			} else {
				value = 1
			}
			mb.GaugeVal(ch, meta.name, meta.help, value, []string{"property"}, []string{key})
		}
	}

	return nil
}
