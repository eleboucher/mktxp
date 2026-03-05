package collector

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/prometheus/client_golang/prometheus"
)

type PackageCollector struct{}

func NewPackageCollector() *PackageCollector                   { return &PackageCollector{} }
func (c *PackageCollector) Name() string                       { return "package" }
func (c *PackageCollector) Describe(_ chan<- *prometheus.Desc) {}

func (c *PackageCollector) Collect(ctx context.Context, e *entry.RouterEntry, ch chan<- prometheus.Metric) error {
	if !e.ConfigEntry.InstalledPackages {
		return nil
	}

	records, err := e.APIConn.Run(ctx, "/system/package/print", "=.proplist=name,version,build-time,disabled")
	if err != nil {
		slog.Error("package collect failed", "router", e.RouterName, "err", err)
		return fmt.Errorf("package: %w", err)
	}

	mb := NewMetricBuilder(e)
	labels := []string{"name", "version", "build_time", "disabled"}

	for _, raw := range records {
		mb.Info(ch, "installed_packages", "Installed packages", labels, TrimRecord(raw, labels))
	}

	return nil
}
