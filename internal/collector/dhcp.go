package collector

import (
	"context"
	"log/slog"

	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/eleboucher/mktxp/internal/utils"
	"github.com/prometheus/client_golang/prometheus"
)

// DHCPCollector collects DHCP lease metrics from RouterOS.
type DHCPCollector struct{}

func NewDHCPCollector() *DHCPCollector {
	return &DHCPCollector{}
}

func (c *DHCPCollector) Name() string { return "dhcp" }

// Describe intentionally sends nothing — unchecked collector with dynamic labels.
func (c *DHCPCollector) Describe(_ chan<- *prometheus.Desc) {}

func (c *DHCPCollector) Collect(ctx context.Context, e *entry.RouterEntry, ch chan<- prometheus.Metric) error {
	if !e.ConfigEntry.DHCP {
		return nil
	}

	records, err := e.APIConn.Run(
		ctx,
		"/ip/dhcp-server/lease/print",
		"=.proplist=active-address,address,mac-address,host-name,comment,server,expires-after,client-id,active-mac-address",
	)
	if err != nil {
		slog.Error("dhcp lease collect failed", "router", e.RouterName, "err", err)
		return nil
	}

	mb := NewMetricBuilder(e)

	wantedKeys := []string{
		"active_address", "address", "mac_address", "host_name",
		"comment", "server", "expires_after", "client_id", "active_mac_address",
	}

	// Count active leases per server.
	serverCounts := make(map[string]float64)
	trimmed := make([]map[string]string, 0, len(records))
	for _, raw := range records {
		record := TrimRecord(raw, wantedKeys)
		trimmed = append(trimmed, record)
		server := record["server"]
		serverCounts[server]++
	}

	// Emit one active-count gauge per server.
	for server, count := range serverCounts {
		mb.GaugeVal(ch, "dhcp_lease_active_count", "Number of active leases per DHCP server",
			count,
			[]string{"server"}, []string{server},
		)
	}

	// Emit per-lease info metrics if DHCPLease flag is set.
	if e.ConfigEntry.DHCPLease {
		leaseLabels := []string{
			"active_address", "address", "mac_address", "host_name",
			"comment", "server", "client_id", "active_mac_address",
		}
		for _, record := range trimmed {
			expiresAfter := float64(utils.ParseMktUptime(record["expires_after"]))
			labelVals := []string{
				record["active_address"],
				record["address"],
				record["mac_address"],
				record["host_name"],
				record["comment"],
				record["server"],
				record["client_id"],
				record["active_mac_address"],
			}
			mb.GaugeVal(ch, "dhcp_lease_info", "DHCP Active Leases",
				expiresAfter,
				leaseLabels,
				labelVals,
			)
		}
	}

	return nil
}
