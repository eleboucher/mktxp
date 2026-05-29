package collector

import (
	"context"
	"fmt"
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
		return fmt.Errorf("dhcp lease: %w", err)
	}

	mb := NewMetricBuilder(e)

	wantedKeys := []string{
		"active_address", "address", "mac_address", "host_name",
		"comment", "server", "expires_after", "client_id", "active_mac_address",
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
		mb.GaugeVal(ch, "dhcp_lease_active_count", "Number of active leases per DHCP server",
			count,
			[]string{"server"}, []string{server},
		)
	}

	if e.ConfigEntry.DHCPLease {
		leaseLabels := []string{
			"active_address", "address", "mac_address", "host_name",
			"comment", "server", "client_id", "active_mac_address",
		}

		metricMapLease := map[string]struct {
			name       string
			help       string
			parseFloat bool
		}{
			"expires_after": {"dhcp_lease_info", "DHCP Active Leases", false},
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

			for key, meta := range metricMapLease {
				if key == "expires_after" {
					mb.GaugeVal(ch, meta.name, meta.help, expiresAfter, leaseLabels, labelVals)
				}
			}
		}
	}

	if e.ConfigEntry.DHCPLease {
		for _, record := range trimmed {
			dhcpName := record["host_name"]
			mb.GaugeVal(ch, "dhcp_name", "DHCP host name", 1,
				[]string{"address", "mac_address", "dhcp_name"},
				[]string{record["address"], record["mac_address"], dhcpName})
		}
	}

	return nil
}

// dhcpResolveKeys are the lease fields cached for client MAC/IP resolution.
var dhcpResolveKeys = []string{
	"address", "mac_address", "host_name", "comment",
	"active_address", "server", "client_id", "active_mac_address",
}

// ensureDHCPCache lazily populates the entry's DHCP lease cache for client name resolution.
func ensureDHCPCache(ctx context.Context, e *entry.RouterEntry) {
	e.EnsureDHCPRecords(ctx, func(ctx context.Context, de *entry.RouterEntry) ([]map[string]string, error) {
		raw, err := de.APIConn.Run(
			ctx,
			"/ip/dhcp-server/lease/print",
			"=.proplist=active-address,address,mac-address,host-name,comment,server,client-id,active-mac-address",
		)
		if err != nil {
			return nil, err
		}
		out := make([]map[string]string, 0, len(raw))
		for _, r := range raw {
			out = append(out, TrimRecord(r, dhcpResolveKeys))
		}
		return out, nil
	})
}

// resolveDHCPName formats a lease's display name, falling back to the MAC address.
func resolveDHCPName(lease map[string]string, format string) string {
	name := lease["host_name"]
	if name == "" {
		name = lease["mac_address"]
	}
	return FormatInterfaceName(name, lease["comment"], format)
}

// resolveClientDHCP sets dhcp_name and dhcp_address on a registration record from
// the DHCP cache, falling back to the MAC when no lease matches.
func resolveClientDHCP(rec map[string]string, de *entry.RouterEntry, format string) {
	mac := rec["mac_address"]
	if lease := de.DHCPRecord(mac); lease != nil {
		rec["dhcp_name"] = resolveDHCPName(lease, format)
		rec["dhcp_address"] = lease["address"]
		return
	}
	rec["dhcp_name"] = mac
	rec["dhcp_address"] = "No DHCP Record"
}
