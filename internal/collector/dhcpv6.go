package collector

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/eleboucher/mktxp/internal/utils"
	"github.com/prometheus/client_golang/prometheus"
)

// DHCPv6Collector collects DHCPv6 binding metrics from RouterOS.
type DHCPv6Collector struct{}

func NewDHCPv6Collector() *DHCPv6Collector {
	return &DHCPv6Collector{}
}

func (c *DHCPv6Collector) Name() string { return "dhcpv6" }

func (c *DHCPv6Collector) Describe(_ chan<- *prometheus.Desc) {}

func (c *DHCPv6Collector) Collect(ctx context.Context, e *entry.RouterEntry, ch chan<- prometheus.Metric) error {
	if !e.ConfigEntry.DHCPv6 {
		return nil
	}

	records, err := e.APIConn.Run(
		ctx,
		"/ipv6/dhcp-server/binding/print",
		"=.proplist=address,duid,iaid,server,status,expires-after,comment",
	)
	if err != nil {
		slog.Error("dhcpv6 binding collect failed", "router", e.RouterName, "err", err)
		return fmt.Errorf("dhcpv6 binding: %w", err)
	}

	mb := NewMetricBuilder(e)

	wantedKeys := []string{
		"address", "duid", "iaid", "server", "status", "expires_after", "comment",
	}

	serverCounts := make(map[string]float64)
	trimmed := make([]map[string]string, 0, len(records))
	for _, raw := range records {
		record := TrimRecord(raw, wantedKeys)
		trimmed = append(trimmed, record)
		server := record["server"]
		serverCounts[server]++
	}

	for server, count := range serverCounts {
		mb.GaugeVal(ch, "dhcpv6_binding_active_count", "Number of active bindings per DHCPv6 server",
			count,
			[]string{"server"}, []string{server},
		)
	}

	if e.ConfigEntry.DHCPv6Lease {
		leaseLabels := []string{
			"address", "duid", "iaid", "server", "status", "comment",
		}

		for _, record := range trimmed {
			expiresAfter := float64(utils.ParseMktUptime(record["expires_after"]))
			labelVals := []string{
				record["address"],
				record["duid"],
				record["iaid"],
				record["server"],
				record["status"],
				record["comment"],
			}
			mb.GaugeVal(ch, "dhcpv6_binding_info", "DHCPv6 Active Bindings", expiresAfter, leaseLabels, labelVals)
		}
	}

	return nil
}
