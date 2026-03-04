package cmd

import (
	"fmt"

	"github.com/eleboucher/mktxp/internal/config"
	"github.com/eleboucher/mktxp/internal/version"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Shows base MKTXP info",
	Long:  `Displays MKTXP version, configuration paths, and router statistics.`,
	Run:   runInfo,
}

func init() {
	rootCmd.AddCommand(infoCmd)
}

func runInfo(cmd *cobra.Command, args []string) {
	info := version.BuildInfo()
	sysCfg := config.Handler.SystemEntry()
	routers := config.Handler.RegisteredEntries()

	fmt.Println("MKTXP - Mikrotik RouterOS Prometheus Exporter")
	fmt.Println()
	fmt.Printf("Version:        %s\n", info["version"])
	fmt.Printf("Git Commit:     %s\n", info["git_commit"])
	fmt.Printf("Build Date:     %s\n", info["build_date"])
	fmt.Println()
	fmt.Printf("Config Dir:     %s\n", config.Handler.ConfigDir())
	fmt.Printf("Main Config:    %s\n", config.Handler.MainConfPath())
	fmt.Println()
	fmt.Printf("Listen Address: %s\n", sysCfg.Listen)
	fmt.Printf("Socket Timeout: %ds\n", sysCfg.SocketTimeout)
	fmt.Printf("Routers:        %d configured\n", len(routers))

	if len(routers) > 0 {
		fmt.Println()
		fmt.Println("Configured Routers:")
		for _, name := range routers {
			cfg := config.Handler.RouterEntry(name)
			if cfg != nil {
				status := "enabled"
				if !cfg.Enabled {
					status = "disabled"
				}
				fmt.Printf("  - %s (%s) [%s]\n", name, cfg.Hostname, status)
			}
		}
	}
}
