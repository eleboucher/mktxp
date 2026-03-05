package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/eleboucher/mktxp/internal/collector"
	"github.com/eleboucher/mktxp/internal/config"
	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Server struct {
	config       *config.SystemConfig
	registry     *collector.Registry
	httpServer   *http.Server
	entries      map[string]*entry.RouterEntry
	mu           sync.RWMutex
	semaphore    chan struct{}
	totalTimeout time.Duration
}

type Options struct {
	ListenOverride string
}

func New(cfg *config.SystemConfig, opts *Options) *Server {
	listen := cfg.Listen
	if opts != nil && opts.ListenOverride != "" {
		listen = opts.ListenOverride
	}

	semSize := cfg.MaxWorkerThreads
	if semSize <= 0 {
		semSize = 5
	}

	return &Server{
		config:       cfg,
		registry:     collector.NewRegistry(),
		entries:      make(map[string]*entry.RouterEntry),
		semaphore:    make(chan struct{}, semSize),
		totalTimeout: time.Duration(cfg.TotalMaxScrapeDuration) * time.Second,
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
	const msg = `MKTXP - Mikrotik RouterOS Prometheus Exporter

Endpoints:
  /metrics  - All router metrics
  /probe    - Target-specific metrics (use ?target=<router>)
`
	if _, err := w.Write([]byte(msg)); err != nil {
		slog.Error("Failed to write root response", "error", err)
	}
}

// getScrapeTimeout safely determines the timeout for the current HTTP request
func (s *Server) getScrapeTimeout(r *http.Request) time.Duration {
	timeout := s.totalTimeout
	if promTimeout := r.Header.Get("X-Prometheus-Scrape-Timeout-Seconds"); promTimeout != "" {
		if parsed, err := strconv.ParseFloat(promTimeout, 64); err == nil {
			timeout = time.Duration(parsed*float64(time.Second)) - (500 * time.Millisecond)
		}
	}
	return timeout
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	slog.Debug("handleMetrics called", "num_entries", len(s.entries))
	s.mu.RLock()
	entries := make([]*entry.RouterEntry, 0, len(s.entries))
	for _, e := range s.entries {
		entries = append(entries, e)
	}
	s.mu.RUnlock()

	timeout := s.getScrapeTimeout(r)
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	registry := prometheus.NewRegistry()
	if s.registry == nil {
		promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(w, r)
		return
	}

	allCollectors := s.registry.All()
	var wg sync.WaitGroup

	for _, e := range entries {
		if e == nil || !e.ConfigEntry.Enabled {
			continue
		}

		wg.Add(1)
		go func(routerEntry *entry.RouterEntry) {
			defer wg.Done()

			select {
			case s.semaphore <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-s.semaphore }()

			if !routerEntry.IsReady(ctx) {
				slog.Warn("Router not ready, skipping", "router", routerEntry.RouterName)
				return
			}

			for _, c := range allCollectors {
				_ = registry.Register(&routerCollector{
					collector:      c,
					entry:          routerEntry,
					ctx:            ctx,
					scrapeDuration: time.Duration(s.config.MaxScrapeDuration) * time.Second,
				})
			}
		}(e)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		http.Error(w, "Scrape timed out during setup", http.StatusGatewayTimeout)
		return
	case <-done:
		promhttp.HandlerFor(registry, promhttp.HandlerOpts{
			Timeout: timeout,
		}).ServeHTTP(w, r)
	}
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

	timeout := s.getScrapeTimeout(r)
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	select {
	case s.semaphore <- struct{}{}:
	case <-ctx.Done():
		http.Error(w, "Request cancelled", http.StatusGatewayTimeout)
		return
	}
	defer func() { <-s.semaphore }()

	if !e.IsReady(ctx) {
		http.Error(w, "Target router is unreachable", http.StatusBadGateway)
		return
	}

	registry := prometheus.NewRegistry()
	if s.registry != nil {
		for _, c := range s.registry.All() {
			_ = registry.Register(&routerCollector{
				collector:      c,
				entry:          e,
				ctx:            ctx,
				scrapeDuration: time.Duration(s.config.MaxScrapeDuration) * time.Second,
			})
		}
	}

	promhttp.HandlerFor(registry, promhttp.HandlerOpts{
		Timeout: timeout,
	}).ServeHTTP(w, r)
}

type routerCollector struct {
	collector      collector.Collector
	entry          *entry.RouterEntry
	ctx            context.Context
	scrapeDuration time.Duration
}

func (rc *routerCollector) Describe(ch chan<- *prometheus.Desc) {
	rc.collector.Describe(ch)
}

func (rc *routerCollector) Collect(ch chan<- prometheus.Metric) {
	start := time.Now()

	scrapeCtx, cancel := context.WithTimeout(rc.ctx, rc.scrapeDuration)
	defer cancel()

	if err := rc.collector.Collect(scrapeCtx, rc.entry, ch); err != nil {
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
	ch <- prometheus.MustNewConstMetric(metricDesc, prometheus.GaugeValue, float64(duration), rc.collector.Name(), rc.getRouterboardName())
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
