package cmd

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/eleboucher/mktxp/internal/collector"
	"github.com/eleboucher/mktxp/internal/config"
	"github.com/eleboucher/mktxp/internal/server"
	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Starts collecting metrics for all enabled RouterOS configuration entries",
	Long: `Starts the MKTXP Prometheus exporter server and begins collecting metrics
from all enabled RouterOS devices. Metrics are exposed at /metrics endpoint.`,
	Run: runExport,
}

var (
	exportListen            string
	exportSocketTimeout     int
	exportVerbose           bool
	exportMaxScrapeDur      int
	exportTotalMaxScrapeDur int
)

func init() {
	rootCmd.AddCommand(exportCmd)

	exportCmd.Flags().StringVar(
		&exportListen,
		"listen",
		"",
		"Override listen address (default from config)",
	)
	exportCmd.Flags().IntVar(
		&exportSocketTimeout,
		"socket-timeout",
		0,
		"Override socket timeout in seconds",
	)
	exportCmd.Flags().BoolVarP(
		&exportVerbose,
		"verbose",
		"v",
		false,
		"Enable verbose/debug logging",
	)
	exportCmd.Flags().IntVar(
		&exportMaxScrapeDur,
		"max-scrape-duration",
		0,
		"Override per-router scrape timeout",
	)
	exportCmd.Flags().IntVar(
		&exportTotalMaxScrapeDur,
		"total-max-scrape-duration",
		0,
		"Override total scrape timeout",
	)
}

func runExport(cmd *cobra.Command, args []string) {
	sysCfg := config.Handler.SystemEntry()

	applyExportOverrides(sysCfg)

	if exportVerbose {
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
		slog.SetDefault(logger)
	}

	opts := &server.Options{
		ListenOverride: exportListen,
	}

	srv := server.New(sysCfg, opts)

	registerCollectors(srv)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		slog.Info("Received signal, shutting down", "signal", sig)
		cancel()
	}()

	slog.Info("Starting MKTXP exporter", "version", cmd.Version)
	if err := srv.Run(ctx); err != nil {
		slog.Error("Server error", "error", err)
		os.Exit(1)
	}
}

func applyExportOverrides(cfg *config.SystemConfig) {
	if exportSocketTimeout > 0 {
		cfg.SocketTimeout = exportSocketTimeout
	}
	if exportMaxScrapeDur > 0 {
		cfg.MaxScrapeDuration = exportMaxScrapeDur
	}
	if exportTotalMaxScrapeDur > 0 {
		cfg.TotalMaxScrapeDuration = exportTotalMaxScrapeDur
	}
}

func registerCollectors(srv *server.Server) {
	for _, c := range collector.AllCollectors() {
		srv.RegisterCollector(c)
	}
}
