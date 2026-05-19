package collector

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/eleboucher/mktxp/internal/config"
	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/showwin/speedtest-go/speedtest"
)

var errInternetUnreachable = errors.New("internet unreachable")

func internetReachable(host string, timeout time.Duration) error {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, "53"), timeout)
	if err != nil {
		return err
	}
	return conn.Close()
}

type BandwidthCollector struct {
	mu              sync.Mutex
	lastTestTime    time.Time
	internetResults *speedtest.Server
	startOnce       sync.Once
}

var (
	bandwidthInstance *BandwidthCollector
	bandwidthOnce     sync.Once
)

func NewBandwidthCollector() *BandwidthCollector {
	bandwidthOnce.Do(func() {
		bandwidthInstance = &BandwidthCollector{}
	})
	return bandwidthInstance
}

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
	sysCfg := config.Handler.SystemEntry()
	if !sysCfg.Bandwidth {
		slog.Info("Bandwidth tests disabled by config; not starting background task", "collector", collectorName)
		return
	}

	c.startOnce.Do(func() {
		go func() {
			slog.Info("Starting background bandwidth test", "collector", collectorName)

			runSpeedtestOnce := func() {
				result, err := c.runSpeedtest()
				if err != nil {
					slog.Debug("bandwidth speedtest failed", "collector", collectorName, "err", err)
					return
				}

				c.mu.Lock()
				defer c.mu.Unlock()
				c.internetResults = result
				c.lastTestTime = time.Now()
				slog.Info("Bandwidth test completed successfully", "collector", collectorName,
					"download_mbps", result.DLSpeed.Mbps(),
					"upload_mbps", result.ULSpeed.Mbps(),
					"latency_ms", result.Latency.Milliseconds())
			}

			runSpeedtestOnce()

			for {
				sysCfg := config.Handler.SystemEntry()
				interval := time.Duration(sysCfg.BandwidthTestInterval) * time.Second

				select {
				case <-ctx.Done():
					slog.Info("Stopping background bandwidth test", "collector", collectorName)
					return
				case <-time.After(interval):
					runSpeedtestOnce()
				}
			}
		}()
	})
}

func (c *BandwidthCollector) runSpeedtest() (*speedtest.Server, error) {
	sysCfg := config.Handler.SystemEntry()
	if host := sysCfg.BandwidthTestDNSServer; host != "" {
		if err := internetReachable(host, 3*time.Second); err != nil {
			return nil, errInternetUnreachable
		}
	}

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
