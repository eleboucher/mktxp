package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/eleboucher/mktxp/internal/collector"
	"github.com/eleboucher/mktxp/internal/config"
	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Server struct {
	config     *config.SystemConfig
	registry   *collector.Registry
	httpServer *http.Server
	entries    map[string]*entry.RouterEntry
	mu         sync.RWMutex
}

type Options struct {
	ListenOverride string
}

func New(cfg *config.SystemConfig, opts *Options) *Server {
	listen := cfg.Listen
	if opts != nil && opts.ListenOverride != "" {
		listen = opts.ListenOverride
	}

	return &Server{
		config:   cfg,
		registry: collector.NewRegistry(),
		entries:  make(map[string]*entry.RouterEntry),
		httpServer: &http.Server{
			Addr: listen,
		},
	}
}

func (s *Server) RegisterCollector(c collector.Collector) {
	s.registry.Register(c)
}

func (s *Server) Run(ctx context.Context) error {
	s.initEntries()
	s.registerRoutes()

	slog.Info("Starting MKTXP server", "listen", s.httpServer.Addr)

	errCh := make(chan error, 1)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("server listen: %w", err)
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		slog.Info("Shutting down server gracefully")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return s.httpServer.Shutdown(shutdownCtx)
	}
}

func (s *Server) initEntries() {
	names := config.Handler.RegisteredEntries()
	for _, name := range names {
		cfg := config.Handler.RouterEntry(name)
		if cfg == nil || !cfg.Enabled {
			continue
		}
		e := entry.New(name)
		s.entries[name] = e
		slog.Debug("Initialized router entry", "name", name)
	}
}

func (s *Server) registerRoutes() {
	mux := http.NewServeMux()

	mux.HandleFunc("/", s.handleRoot)
	mux.HandleFunc("/metrics", s.handleMetrics)
	mux.HandleFunc("/probe", s.handleProbe)

	s.httpServer.Handler = mux
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	_, err := fmt.Fprintf(w, "MKTXP - Mikrotik RouterOS Prometheus Exporter\n\n")
	if err != nil {
		slog.Error("Failed to write root response", "error", err)
		return
	}
	_, err = fmt.Fprintf(w, "Endpoints:\n")
	if err != nil {
		slog.Error("Failed to write root response", "error", err)
		return
	}
	_, err = fmt.Fprintf(w, "  /metrics  - All router metrics\n")
	if err != nil {
		slog.Error("Failed to write root response", "error", err)
		return
	}
	_, err = fmt.Fprintf(w, "  /probe    - Target-specific metrics (use ?target=<router>)\n")
	if err != nil {
		slog.Error("Failed to write root response", "error", err)
		return
	}
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	slog.Debug("handleMetrics called", "num_entries", len(s.entries), "registry_nil", s.registry == nil)
	s.mu.RLock()
	entries := make([]*entry.RouterEntry, 0, len(s.entries))
	for _, e := range s.entries {
		entries = append(entries, e)
	}
	s.mu.RUnlock()

	registry := prometheus.NewRegistry()
	if s.registry != nil {
		allCollectors := s.registry.All()
		slog.Debug("Registering collectors", "num_collectors", len(allCollectors))

		for _, e := range entries {
			if e == nil {
				slog.Warn("Entry is nil, skipping")
				continue
			}
			e.Connect(r.Context())

			for _, c := range allCollectors {
				registry.MustRegister(&routerCollector{collector: c, entry: e})
			}
		}
	}

	promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(w, r)
}

func (s *Server) handleProbe(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Query().Get("target")
	if target == "" {
		http.Error(w, "Missing 'target' parameter", http.StatusBadRequest)
		return
	}

	s.mu.RLock()
	e, exists := s.entries[target]
	s.mu.RUnlock()

	if !exists {
		http.Error(w, fmt.Sprintf("Unknown target: %s", target), http.StatusNotFound)
		return
	}

	registry := prometheus.NewRegistry()
	s.collectRouterMetrics(r.Context(), e, registry)

	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	handler.ServeHTTP(w, r)
}

func (s *Server) collectRouterMetrics(ctx context.Context, e *entry.RouterEntry, registry *prometheus.Registry) {
	if !e.IsReady(ctx) {
		slog.Warn("Router not ready, skipping collection", "router", e.RouterName)
		return
	}

	if s.registry != nil {
		collectors := s.registry.All()
		for _, c := range collectors {
			registry.MustRegister(&routerCollector{
				collector: c,
				entry:     e,
			})
		}
	}
}

type routerCollector struct {
	collector collector.Collector
	entry     *entry.RouterEntry
}

func (rc *routerCollector) Describe(ch chan<- *prometheus.Desc) {
	rc.collector.Describe(ch)
}

func (rc *routerCollector) Collect(ch chan<- prometheus.Metric) {
	start := time.Now()
	ctx := context.Background()
	if err := rc.collector.Collect(ctx, rc.entry, ch); err != nil {
		routerName := "unknown"
		if rc.entry != nil {
			routerName = rc.entry.RouterName
		}
		slog.Error("Collector failed", "collector", rc.collector.Name(), "router", routerName, "error", err)
	}
	duration := time.Since(start).Milliseconds()
	metricDesc := prometheus.NewDesc(
		"mktxp_collection_time_total",
		"Total time spent collecting metrics in milliseconds",
		[]string{"collector", "routerboard_name"},
		nil,
	)
	ch <- prometheus.MustNewConstMetric(metricDesc, prometheus.CounterValue, float64(duration), rc.collector.Name(), rc.getRouterboardName())
}

func (rc *routerCollector) getRouterboardName() string {
	if rc.entry != nil {
		if name, ok := rc.entry.RouterID["routerboard_name"]; ok {
			return name
		}
		return rc.entry.RouterName
	}
	return "unknown"
}
