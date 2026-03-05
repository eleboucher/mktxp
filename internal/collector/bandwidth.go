package collector

import (
	"context"
	"log/slog"
	"net"
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
	now := time.Now()
	interval := time.Duration(sysCfg.BandwidthTestInterval) * time.Second

	if c.internetResults == nil || now.Sub(c.lastTestTime) > interval {
		if c.checkInternetConnectivity(sysCfg.BandwidthTestDNSServer) {
			result, err := c.runSpeedtest()
			if err != nil {
				slog.Debug("bandwidth speedtest failed", "router", e.RouterName, "err", err)
				c.internetResults = &speedtest.Server{}
			} else {
				c.internetResults = result
				c.lastTestTime = now
			}
		}
	}
	result := c.internetResults
	c.mu.Unlock()

	if result != nil && result.DLSpeed > 0 || result.ULSpeed > 0 {
		mb.GaugeVal(ch, "internet_bandwidth_download", "Internet download bandwidth in bits per second",
			float64(result.DLSpeed)*8, []string{"direction"}, []string{"download"})
		mb.GaugeVal(ch, "internet_bandwidth_upload", "Internet upload bandwidth in bits per second",
			float64(result.ULSpeed)*8, []string{"direction"}, []string{"upload"})
		mb.GaugeVal(ch, "internet_latency", "Internet latency in milliseconds",
			float64(result.Latency.Milliseconds()), nil, nil)
	}

	records, err := e.APIConn.Run(ctx, "/tool/benchmark/print")
	if err != nil {
		slog.Debug("bandwidth collect failed", "router", e.RouterName, "err", err)
		return nil
	}

	labelKeysWithRouter := []string{"routerboard_name"}

	for _, raw := range records {
		rec := TrimRecord(raw, nil)

		metricMap := map[string]struct {
			name       string
			help       string
			parseFloat bool
		}{
			"cpu":     {"bandwidth_cpu_score", "Bandwidth test CPU score", true},
			"write":   {"bandwidth_write_speed", "Bandwidth test write speed in MB/s", true},
			"read":    {"bandwidth_read_speed", "Bandwidth test read speed in MB/s", true},
			"latency": {"bandwidth_latency", "Bandwidth test latency in ms", true},
		}

		for key, meta := range metricMap {
			if val, ok := rec[key]; ok && val != "" {
				var value float64
				if meta.parseFloat {
					value = ParseFloat(val)
				} else {
					value = 1
				}
				mb.GaugeVal(ch, meta.name, meta.help, value, labelKeysWithRouter, []string{e.RouterID["routerboard_name"]})
			}
		}

		if _, ok := rec["comment"]; ok && rec["comment"] != "" {
			mb.Info(ch, "bandwidth_info", "Information about bandwidth test results",
				[]string{"cpu", "write", "read"},
				rec)
		}
	}

	return nil
}

func (c *BandwidthCollector) checkInternetConnectivity(dnsServer string) bool {
	conn, err := net.DialTimeout("tcp", dnsServer+":80", 5*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close() //nolint:errcheck
	return true
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
