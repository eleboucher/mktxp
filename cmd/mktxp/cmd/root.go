package cmd

import (
	"fmt"
	"os"

	"github.com/eleboucher/mktxp/internal/config"
	"github.com/eleboucher/mktxp/internal/version"
	"github.com/spf13/cobra"
)

var (
	cfgDir  string
	rootCmd = &cobra.Command{
		Use:   "mktxp",
		Short: "Prometheus Exporter for Mikrotik RouterOS",
		Long: `MKTXP (Mikrotik Traffic Exporter for Prometheus) collects metrics 
from Mikrotik RouterOS devices and exports them in Prometheus format.`,
		Version: version.Version,
	}
)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgDir, "cfg-dir", "", "MKTXP config files directory (default is ~/mktxp)")

	cobra.OnInitialize(initConfig)
}

func initConfig() {
	if err := config.Handler.Init(cfgDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing config: %v\n", err)
		os.Exit(1)
	}
}
