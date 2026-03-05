package collector

import (
	"context"
	"log/slog"

	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/prometheus/client_golang/prometheus"
)

// RouteCollector collects routing table metrics from RouterOS.
type RouteCollector struct{}

func NewRouteCollector() *RouteCollector {
	return &RouteCollector{}
}

func (c *RouteCollector) Name() string { return "route" }

// Describe intentionally sends nothing — unchecked collector with dynamic labels.
func (c *RouteCollector) Describe(_ chan<- *prometheus.Desc) {}

func (c *RouteCollector) Collect(ctx context.Context, e *entry.RouterEntry, ch chan<- prometheus.Metric) error {
	mb := NewMetricBuilder(e)

	if e.ConfigEntry.Route {
		if err := collectRouteMetrics(ctx, e, mb, ch,
			"/ip/route/print",
			"routes_total_routes", "Overall number of routes in RIB",
			"routes_protocol_count", "Number of routes per protocol in RIB",
		); err != nil {
			slog.Error("route collect failed", "router", e.RouterName, "err", err)
		}
	}

	if e.ConfigEntry.IPv6Route {
		if err := collectRouteMetrics(ctx, e, mb, ch,
			"/ipv6/route/print",
			"routes_total_routes_ipv6", "Overall number of IPv6 routes in RIB",
			"routes_protocol_count_ipv6", "Number of IPv6 routes per protocol in RIB",
		); err != nil {
			slog.Error("ipv6 route collect failed", "router", e.RouterName, "err", err)
		}
	}

	return nil
}

var routeProtocols = []string{"dynamic", "connect", "static", "bgp", "ospf"}

func collectRouteMetrics(
	ctx context.Context,
	e *entry.RouterEntry,
	mb *MetricBuilder,
	ch chan<- prometheus.Metric,
	api string,
	totalMetric, totalHelp string,
	protocolMetric, protocolHelp string,
) error {
	records, err := e.APIConn.Run(ctx, api, "=.proplist=dynamic,connect,static,bgp,ospf")
	if err != nil {
		return err
	}

	protocolCounts := make(map[string]float64, len(routeProtocols))
	for _, p := range routeProtocols {
		protocolCounts[p] = 0
	}

	total := float64(len(records))
	for _, raw := range records {
		record := TrimRecord(raw, routeProtocols)
		for _, proto := range routeProtocols {
			if record[proto] == trueStr {
				protocolCounts[proto]++
			}
		}
	}

	mb.GaugeVal(ch, totalMetric, totalHelp, total, nil, nil)

	for _, proto := range routeProtocols {
		mb.GaugeVal(ch, protocolMetric, protocolHelp,
			protocolCounts[proto],
			[]string{"protocol"}, []string{proto},
		)
	}

	return nil
}
