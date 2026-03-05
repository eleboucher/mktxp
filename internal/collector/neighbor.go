package collector

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/prometheus/client_golang/prometheus"
)

type NeighborCollector struct{}

func NewNeighborCollector() *NeighborCollector                  { return &NeighborCollector{} }
func (c *NeighborCollector) Name() string                       { return "neighbor" }
func (c *NeighborCollector) Describe(_ chan<- *prometheus.Desc) {}

func (c *NeighborCollector) Collect(ctx context.Context, e *entry.RouterEntry, ch chan<- prometheus.Metric) error {
	mb := NewMetricBuilder(e)

	if e.ConfigEntry.Neighbor {
		records, err := e.APIConn.Run(ctx, "/ip/neighbor/print", "=.proplist=address,interface,mac-address,identity")
		if err != nil {
			slog.Error("neighbor collect failed", "router", e.RouterName, "err", err)
			return fmt.Errorf("ipv4 neighbor: %w", err)
		} else {
			labels := []string{"address", "interface", "mac_address", "identity"}
			for _, raw := range records {
				mb.Info(ch, "neighbor", "Reachable neighbors (IPv4)", labels, TrimRecord(raw, labels))
			}
		}
	}

	if e.ConfigEntry.IPv6Neighbor {
		records, err := e.APIConn.Run(ctx, "/ipv6/neighbor/print", "=.proplist=address,interface,mac-address,status,comment")
		if err != nil {
			slog.Error("ipv6 neighbor collect failed", "router", e.RouterName, "err", err)
			return fmt.Errorf("ipv6 neighbor: %w", err)
		} else {
			labels := []string{"address", "interface", "mac_address", "status", "comment"}
			for _, raw := range records {
				rec := TrimRecord(raw, labels)
				if rec["status"] == "reachable" {
					mb.Info(ch, "ipv6_neighbor", "Reachable neighbors (IPv6)", labels, rec)
				}
			}
		}
	}

	return nil
}
