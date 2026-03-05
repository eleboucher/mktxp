package collector

import (
	"context"
	"log/slog"

	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/prometheus/client_golang/prometheus"
)

type AddressListCollector struct{}

func NewAddressListCollector() *AddressListCollector { return &AddressListCollector{} }

func (c *AddressListCollector) Name() string { return "address_list" }

func (c *AddressListCollector) Describe(_ chan<- *prometheus.Desc) {}

func (c *AddressListCollector) Collect(ctx context.Context, e *entry.RouterEntry, ch chan<- prometheus.Metric) error {
	if len(e.ConfigEntry.AddressList) == 0 {
		return nil
	}

	mb := NewMetricBuilder(e)

	if err := c.collectIPv4(ctx, e, mb, ch); err != nil {
		slog.Debug("address_list ipv4 collect failed", "router", e.RouterName, "err", err)
	}

	return nil
}

func (c *AddressListCollector) collectIPv4(ctx context.Context, e *entry.RouterEntry, mb *MetricBuilder, ch chan<- prometheus.Metric) error {
	records, err := e.APIConn.Run(ctx, "/ip/firewall/address-list/print")
	if err != nil {
		return err
	}

	wantedKeys := []string{"list", "address", "dynamic", "timeout", "disabled", "comment"}

	var trimmed []map[string]string
	for _, raw := range records {
		rec := TrimRecord(raw, wantedKeys)
		trimmed = append(trimmed, rec)
	}

	if len(trimmed) == 0 {
		return nil
	}

	labels := []string{"list", "address", "dynamic", "timeout", "disabled", "comment"}
	for _, rec := range trimmed {
		mb.Gauge(ch, "firewall_address_list", "Firewall IPv4 Address List Entry", "timeout", labels, rec)
	}

	return nil
}
