package collector

import (
	"context"
	"log/slog"

	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/prometheus/client_golang/prometheus"
)

type PublicIPCollector struct{}

func NewPublicIPCollector() *PublicIPCollector                  { return &PublicIPCollector{} }
func (c *PublicIPCollector) Name() string                       { return "public_ip" }
func (c *PublicIPCollector) Describe(_ chan<- *prometheus.Desc) {}

func (c *PublicIPCollector) Collect(ctx context.Context, e *entry.RouterEntry, ch chan<- prometheus.Metric) error {
	if !e.ConfigEntry.PublicIP {
		return nil
	}

	records, err := e.APIConn.Run(ctx, "/ip/cloud/print", "=.proplist=public-address,public-address-ipv6,dns-name")
	if err != nil {
		slog.Error("public_ip collect failed", "router", e.RouterName, "err", err)
		return nil
	}

	mb := NewMetricBuilder(e)
	labels := []string{"public_address", "public_address_ipv6", "dns_name"}

	for _, raw := range records {
		rec := TrimRecord(raw, labels)
		if rec["dns_name"] == "" {
			rec["dns_name"] = "ddns disabled"
		}
		mb.Info(ch, "public_ip_address", "Public IP address", labels, rec)
	}

	return nil
}
