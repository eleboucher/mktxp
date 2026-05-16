package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/eleboucher/mktxp/internal/config"
)

func baseSysCfg() *config.SystemConfig {
	return &config.SystemConfig{
		Listen:                         ":0",
		SocketTimeout:                  2,
		InitialDelayOnFailure:          120,
		MaxDelayOnFailure:              900,
		DelayIncDiv:                    5,
		BandwidthTestDNSServer:         "8.8.8.8",
		BandwidthTestInterval:          420,
		MinimalCollectInterval:         0,
		VerboseMode:                    false,
		FetchRoutersInParallel:         true,
		MaxWorkerThreads:               7,
		MaxScrapeDuration:              30,
		TotalMaxScrapeDuration:         90,
		PersistentRouterConnectionPool: true,
		PersistentDHCPCache:            true,
	}
}

func TestNew_FetchRoutersInParallel_TrueRespectsMaxWorkerThreads(t *testing.T) {
	t.Parallel()

	cfg := baseSysCfg()
	cfg.FetchRoutersInParallel = true
	cfg.MaxWorkerThreads = 7

	s := New(cfg, nil)

	if cap(s.semaphore) != 7 {
		t.Errorf("semaphore cap = %d, want 7", cap(s.semaphore))
	}
}

func TestNew_FetchRoutersInParallel_FalseClampsSemaphoreToOne(t *testing.T) {
	t.Parallel()

	cfg := baseSysCfg()
	cfg.FetchRoutersInParallel = false
	cfg.MaxWorkerThreads = 7

	s := New(cfg, nil)

	if cap(s.semaphore) != 1 {
		t.Errorf("semaphore cap = %d, want 1 (serialized mode)", cap(s.semaphore))
	}
}

func TestReserveCollectSlot_ZeroIntervalAlwaysAllows(t *testing.T) {
	t.Parallel()

	cfg := baseSysCfg()
	cfg.MinimalCollectInterval = 0
	s := New(cfg, nil)

	for i := range 5 {
		if !s.reserveCollectSlot() {
			t.Errorf("call %d: reserveCollectSlot returned false with zero interval", i)
		}
	}
}

func TestReserveCollectSlot_RateLimitsWithinWindow(t *testing.T) {
	t.Parallel()

	cfg := baseSysCfg()
	cfg.MinimalCollectInterval = 10 // seconds
	s := New(cfg, nil)

	if !s.reserveCollectSlot() {
		t.Fatal("first call should be allowed")
	}
	if s.reserveCollectSlot() {
		t.Error("second call within window should be denied")
	}
	if s.reserveCollectSlot() {
		t.Error("third call within window should still be denied")
	}
}

func TestReserveCollectSlot_AllowsAfterWindow(t *testing.T) {
	t.Parallel()

	cfg := baseSysCfg()
	cfg.MinimalCollectInterval = 1 // seconds (smallest unit)
	s := New(cfg, nil)

	if !s.reserveCollectSlot() {
		t.Fatal("first call should be allowed")
	}
	// Backdate the timestamp to simulate window expiry rather than sleeping.
	s.lastCollectMu.Lock()
	s.lastCollectAt = time.Now().Add(-2 * time.Second)
	s.lastCollectMu.Unlock()

	if !s.reserveCollectSlot() {
		t.Error("call after window should be allowed")
	}
}

func TestHandleMetrics_RateLimitedReturns429(t *testing.T) {
	t.Parallel()

	cfg := baseSysCfg()
	cfg.MinimalCollectInterval = 60 // generous window so the second call is denied
	s := New(cfg, nil)
	s.registerRoutes()
	config.Handler.RegisterTestSystemConfig(cfg)

	// First request consumes the slot.
	req1 := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w1 := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w1, req1.WithContext(t.Context()))

	// Second request within the window must be deferred.
	req2 := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w2 := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w2, req2.WithContext(t.Context()))

	if w2.Code != http.StatusTooManyRequests {
		t.Errorf("second /metrics status = %d, want 429", w2.Code)
	}
}
