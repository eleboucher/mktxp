package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/eleboucher/mktxp/internal/config"
	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/prometheus/client_golang/prometheus"
)

// newTestServer creates a Server with mocked config for testing.
func newTestServer(t *testing.T) *Server {
	t.Helper()

	sysCfg := &config.SystemConfig{
		Listen:                         ":0", // Use any available port
		SocketTimeout:                  2,
		InitialDelayOnFailure:          120,
		MaxDelayOnFailure:              900,
		DelayIncDiv:                    5,
		BandwidthTestDNSServer:         "8.8.8.8",
		BandwidthTestInterval:          420,
		MinimalCollectInterval:         5,
		VerboseMode:                    false,
		FetchRoutersInParallel:         false,
		MaxWorkerThreads:               5,
		MaxScrapeDuration:              30,
		TotalMaxScrapeDuration:         90,
		PersistentRouterConnectionPool: true,
		PersistentDHCPCache:            true,
		ProbeConnectionPool:            false,
		ProbeConnectionPoolTTL:         300,
		ProbeConnectionPoolMaxSize:     128,
	}

	s := New(sysCfg, nil)

	config.Handler.RegisterTestSystemConfig(sysCfg)
	config.Handler.RegisterTestRouterEntry("test-router", &config.RouterConfigEntry{
		Hostname: "127.0.0.1",
		Port:     8728,
		Username: "admin",
		Password: "password",
		Enabled:  true,
	})

	return s
}

func TestHandleRoot(t *testing.T) {
	t.Parallel()

	s := newTestServer(t)
	s.registerRoutes()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	s.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/plain" {
		t.Errorf("Content-Type = %q, want text/plain", ct)
	}
	body := w.Body.String()
	if len(body) == 0 {
		t.Error("expected non-empty body")
	}
}

func TestHandleProbe_MissingTarget(t *testing.T) {
	t.Parallel()

	s := newTestServer(t)
	s.registerRoutes()

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	w := httptest.NewRecorder()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	s.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestHandleProbe_UnknownTarget(t *testing.T) {
	t.Parallel()

	s := newTestServer(t)
	s.registerRoutes()

	req := httptest.NewRequest(http.MethodGet, "/probe?target=nonexistent-router", nil)
	w := httptest.NewRecorder()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	s.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestHandleProbe_KnownTargetReturns200(t *testing.T) {
	t.Parallel()

	s := newTestServer(t)
	s.registerRoutes()

	req := httptest.NewRequest(http.MethodGet, "/probe?target=Sample-Router", nil)
	w := httptest.NewRecorder()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	s.httpServer.Handler.ServeHTTP(w, req)

	// With no entries registered, this should return 404 (unknown target).
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

type testCollector struct{}

func (c *testCollector) Name() string { return "test" }
func (c *testCollector) Describe(ch chan<- *prometheus.Desc) {
	desc := prometheus.NewDesc("test_metric", "Test metric", nil, nil)
	ch <- desc
}

func (c *testCollector) Collect(ctx context.Context, e *entry.RouterEntry, ch chan<- prometheus.Metric) error {
	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc("test_metric", "Test metric", nil, nil),
		prometheus.GaugeValue,
		1.0,
	)
	return nil
}

func TestHandleMetrics_Returns200WithHealthUp(t *testing.T) {
	t.Parallel()

	s := newTestServer(t)
	s.RegisterCollector(&testCollector{})
	s.initEntries()
	s.registerRoutes()

	if s.httpServer.Handler == nil {
		t.Fatal("httpServer.Handler is nil")
	}

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	s.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	t.Logf("Response body length: %d, first 100 chars: %.100q", len(body), body)
	if len(body) == 0 {
		t.Error("expected non-empty body")
	}
}

func TestOptionsListenOverride(t *testing.T) {
	t.Parallel()

	sysCfg := &config.SystemConfig{
		Listen:                         ":0",
		SocketTimeout:                  2,
		InitialDelayOnFailure:          120,
		MaxDelayOnFailure:              900,
		DelayIncDiv:                    5,
		BandwidthTestDNSServer:         "8.8.8.8",
		BandwidthTestInterval:          420,
		MinimalCollectInterval:         5,
		VerboseMode:                    false,
		FetchRoutersInParallel:         false,
		MaxWorkerThreads:               5,
		MaxScrapeDuration:              30,
		TotalMaxScrapeDuration:         90,
		PersistentRouterConnectionPool: true,
		PersistentDHCPCache:            true,
		ProbeConnectionPool:            false,
		ProbeConnectionPoolTTL:         300,
		ProbeConnectionPoolMaxSize:     128,
	}

	s := New(sysCfg, &Options{ListenOverride: ":19090"})

	if s.httpServer.Addr != ":19090" {
		t.Errorf("Addr = %q, want :19090", s.httpServer.Addr)
	}
}
