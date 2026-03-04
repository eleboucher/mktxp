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

	if val, ok := rec["cache_size"]; ok {
		mb.GaugeVal(ch, "dns_info", "DNS info", ParseFloat(val)*1024, []string{"property"}, []string{"cache_size"})
	}

	if val, ok := rec["cache_used"]; ok {
		mb.GaugeVal(ch, "dns_info", "DNS info", ParseFloat(val)*1024, []string{"property"}, []string{"cache_used"})
	}

	return nil
}
