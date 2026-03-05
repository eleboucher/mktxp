package collector

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/eleboucher/mktxp/internal/config"
	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/showwin/speedtest-go/speedtest"
)

type BandwidthCollector struct {
	mu              sync.Mutex
	lastTestTime    time.Time
	internetResults *speedtest.Server
}

func NewBandwidthCollector() *BandwidthCollector { return &BandwidthCollector{} }

func (c *BandwidthCollector) Name() string { return "bandwidth" }

func (c *BandwidthCollector) Describe(_ chan<- *prometheus.Desc) {}

func (c *BandwidthCollector) Collect(ctx context.Context, e *entry.RouterEntry, ch chan<- prometheus.Metric) error {
	sysCfg := config.Handler.SystemEntry()
	if !sysCfg.Bandwidth {
		return nil
	}

	mb := NewMetricBuilder(e)

	c.mu.Lock()
	result := c.internetResults
	defer c.mu.Unlock()

	if result == nil {
		slog.Debug("No bandwidth results available yet", "router", e.RouterName)
		return nil
	}

	dlBits := float64(result.DLSpeed) * 8
	ulBits := float64(result.ULSpeed) * 8

	metrics := []struct {
		direction string
		value     float64
	}{
		{"download", dlBits},
		{"upload", ulBits},
	}

	for _, m := range metrics {
		mb.GaugeVal(ch, "internet_bandwidth_"+m.direction,
			"Internet "+m.direction+" bandwidth in bits per second",
			m.value, []string{"direction"}, []string{m.direction})
	}

	mb.GaugeVal(ch, "internet_latency", "Internet latency in milliseconds",
		float64(result.Latency.Milliseconds()), nil, nil)

	for _, m := range metrics {
		mb.GaugeVal(ch, "internet_bandwidth", "Internet bandwidth in bits per second",
			m.value, []string{"direction"}, []string{m.direction})
	}

	return nil
}

func (c *BandwidthCollector) StartBackgroundTest(ctx context.Context, collectorName string) {
	go func() {
		sysCfg := config.Handler.SystemEntry()
		interval := time.Duration(sysCfg.BandwidthTestInterval) * time.Second

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		slog.Info("Starting background bandwidth test", "collector", collectorName, "interval_seconds", interval.Seconds())

		runSpeedtestOnce := func() {
			result, err := c.runSpeedtest()
			if err != nil {
				slog.Debug("bandwidth speedtest failed", "collector", collectorName, "err", err)
				return
			}

			c.mu.Lock()
			c.internetResults = result
			c.lastTestTime = time.Now()
			defer c.mu.Unlock()
			slog.Info("Bandwidth test completed successfully", "collector", collectorName,
				"download_mbps", result.DLSpeed.Mbps(),
				"upload_mbps", result.ULSpeed.Mbps(),
				"latency_ms", result.Latency.Milliseconds())
		}

		runSpeedtestOnce()

		for {
			select {
			case <-ctx.Done():
				slog.Info("Stopping background bandwidth test", "collector", collectorName)
				return
			case <-ticker.C:
				runSpeedtestOnce()
			}
		}
	}()
}

func (c *BandwidthCollector) runSpeedtest() (*speedtest.Server, error) {
	client := speedtest.New()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	servers, err := client.FetchServerListContext(ctx)
	if err != nil {
		return nil, err
	}

	if len(servers) == 0 {
		return nil, speedtest.ErrServerNotFound
	}

	server := servers[0]

	err = server.PingTestContext(ctx, nil)
	if err != nil {
		return nil, err
	}

	err = server.DownloadTestContext(ctx)
	if err != nil {
		return nil, err
	}

	err = server.UploadTestContext(ctx)
	if err != nil {
		return nil, err
	}

	return server, nil
}
