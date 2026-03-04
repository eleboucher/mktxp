package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/eleboucher/mktxp/internal/config"
	"github.com/eleboucher/mktxp/internal/entry"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
	"github.com/spf13/cobra"
)

var printCmd = &cobra.Command{
	Use:   "print",
	Short: "Displays selected metrics on the command line",
	Long:  `Connects to a RouterOS device and prints metrics to stdout in Prometheus format.`,
	Run:   runPrint,
}

var (
	printEntryName string
	printFormat    string
)

func init() {
	rootCmd.AddCommand(printCmd)
	printCmd.Flags().StringVarP(&printEntryName, "entry-name", "e", "", "Config entry name (required)")
	printCmd.Flags().StringVarP(&printFormat, "format", "f", "prometheus", "Output format: prometheus, json")
	if err := printCmd.MarkFlagRequired("entry-name"); err != nil {
		panic(err)
	}
}

func runPrint(cmd *cobra.Command, args []string) {
	cfg := config.Handler.RouterEntry(printEntryName)
	if cfg == nil {
		fmt.Fprintf(os.Stderr, "Router entry '%s' not found\n", printEntryName)
		os.Exit(1)
	}

	if !cfg.Enabled {
		fmt.Fprintf(os.Stderr, "Router entry '%s' is disabled\n", printEntryName)
		os.Exit(1)
	}

	e := entry.New(printEntryName)

	ctx := context.Background()
	if !e.IsReady(ctx) {
		fmt.Fprintf(os.Stderr, "Failed to connect to router '%s'\n", printEntryName)
		os.Exit(1)
	}
	defer e.IsDone()

	registry := prometheus.NewRegistry()

	upDesc := prometheus.NewDesc(
		"mktxp_router_up",
		"Indicates if the router is reachable",
		[]string{"router", "hostname"},
		nil,
	)

	registry.MustRegister(&printCollector{
		entry:  e,
		upDesc: upDesc,
	})

	switch printFormat {
	case "prometheus":
		printPrometheusMetrics(registry)
	case "json":
		printJSONMetrics(e)
	default:
		fmt.Fprintf(os.Stderr, "Unknown format: %s (use 'prometheus' or 'json')\n", printFormat)
		os.Exit(1)
	}
}

type printCollector struct {
	entry  *entry.RouterEntry
	upDesc *prometheus.Desc
}

func (pc *printCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- pc.upDesc
}

func (pc *printCollector) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(
		pc.upDesc,
		prometheus.GaugeValue,
		1,
		pc.entry.RouterName,
		pc.entry.ConfigEntry.Hostname,
	)
}

func printPrometheusMetrics(registry *prometheus.Registry) {
	metricFamilies, err := registry.Gather()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error gathering metrics: %v\n", err)
		os.Exit(1)
	}

	for _, mf := range metricFamilies {
		if _, err := expfmt.MetricFamilyToText(os.Stdout, mf); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing metric: %v\n", err)
		}
	}
}

func printJSONMetrics(e *entry.RouterEntry) {
	slog.Info("JSON format not yet implemented, router info",
		"router", e.RouterName,
		"hostname", e.ConfigEntry.Hostname,
		"connected", e.APIConn.IsConnected())
}
