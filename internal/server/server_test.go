package server

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eleboucher/mktxp/internal/collector"
	"github.com/eleboucher/mktxp/internal/config"
)

// newTestServer creates a Server with real config from .dev/ and registers routes.
// It does NOT call initEntries or Run — the mux is set up with registerRoutes only.
func newTestServer(t *testing.T) *Server {
	t.Helper()

	devDir := filepath.Join("..", "..", ".dev")
	if err := config.Handler.Init(devDir); err != nil {
		t.Fatalf("config init failed: %v", err)
	}

	sysCfg := config.Handler.SystemEntry()
	s := New(sysCfg, nil)
	for _, c := range collector.AllCollectors() {
		s.RegisterCollector(c)
	}
	s.initEntries()
	s.registerRoutes()
	return s
}

func TestHandleRoot(t *testing.T) {
	t.Parallel()

	devDir := filepath.Join("..", "..", ".dev")
	if err := config.Handler.Init(devDir); err != nil {
		t.Skip(".dev/ config not available:", err)
	}

	sysCfg := config.Handler.SystemEntry()
	s := New(sysCfg, nil)
	s.registerRoutes()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

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

	devDir := filepath.Join("..", "..", ".dev")
	if err := config.Handler.Init(devDir); err != nil {
		t.Skip(".dev/ config not available:", err)
	}

	sysCfg := config.Handler.SystemEntry()
	s := New(sysCfg, nil)
	s.registerRoutes()

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	w := httptest.NewRecorder()

	s.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestHandleProbe_UnknownTarget(t *testing.T) {
	t.Parallel()

	devDir := filepath.Join("..", "..", ".dev")
	if err := config.Handler.Init(devDir); err != nil {
		t.Skip(".dev/ config not available:", err)
	}

	sysCfg := config.Handler.SystemEntry()
	s := New(sysCfg, nil)
	s.initEntries()
	s.registerRoutes()

	req := httptest.NewRequest(http.MethodGet, "/probe?target=nonexistent-router", nil)
	w := httptest.NewRecorder()

	s.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestHandleProbe_KnownTargetReturns200(t *testing.T) {
	t.Parallel()

	s := newTestServer(t)

	// Sample-Router is configured in .dev/mktxp.yaml but unreachable.
	// The server should still return 200 — it just logs "not ready" and emits no metrics.
	req := httptest.NewRequest(http.MethodGet, "/probe?target=Sample-Router", nil)
	w := httptest.NewRecorder()

	s.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestHandleMetrics_Returns200WithHealthUp(t *testing.T) {
	t.Parallel()

	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()

	s.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	// health collector always emits mktxp_health_up, even for unreachable routers.
	if !strings.Contains(body, "mktxp_health_up") {
		t.Errorf("expected mktxp_health_up in /metrics output, got:\n%s", body)
	}
}

func TestOptionsListenOverride(t *testing.T) {
	t.Parallel()

	devDir := filepath.Join("..", "..", ".dev")
	if err := config.Handler.Init(devDir); err != nil {
		t.Skip(".dev/ config not available:", err)
	}

	sysCfg := config.Handler.SystemEntry()
	s := New(sysCfg, &Options{ListenOverride: ":19090"})

	if s.httpServer.Addr != ":19090" {
		t.Errorf("Addr = %q, want :19090", s.httpServer.Addr)
	}
}
