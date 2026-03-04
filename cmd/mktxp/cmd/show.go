package cmd

import (
	"fmt"
	"strings"

	"github.com/eleboucher/mktxp/internal/config"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Shows MKTXP configuration entries",
	Long:  `Displays MKTXP config router entries on the command line.`,
	Run:   runShow,
}

var (
	showEntryName string
	showConfig    bool
)

func init() {
	rootCmd.AddCommand(showCmd)
	showCmd.Flags().StringVarP(&showEntryName, "entry-name", "e", "", "Config entry name to display")
	showCmd.Flags().BoolVarP(&showConfig, "config", "c", false, "Shows MKTXP config files paths")
}

func runShow(cmd *cobra.Command, args []string) {
	if showConfig {
		showConfigPaths()
		return
	}

	if showEntryName != "" {
		showRouterEntry(showEntryName)
		return
	}

	showAllEntries()
}

func showConfigPaths() {
	fmt.Println("MKTXP Configuration Files:")
	fmt.Printf("  Config Directory: %s\n", config.Handler.ConfigDir())
	fmt.Printf("  Main Config:      %s\n", config.Handler.MainConfPath())
	fmt.Printf("  System Config:    %s\n", config.Handler.SysConfPath())
}

func showRouterEntry(name string) {
	cfg := config.Handler.RouterEntry(name)
	if cfg == nil {
		fmt.Printf("Router entry '%s' not found\n", name)
		return
	}

	printRouterConfig(name, cfg)
}

func showAllEntries() {
	routers := config.Handler.RegisteredEntries()
	if len(routers) == 0 {
		fmt.Println("No routers configured")
		return
	}

	fmt.Printf("Configured Routers (%d):\n", len(routers))
	fmt.Println()
	for _, name := range routers {
		cfg := config.Handler.RouterEntry(name)
		if cfg != nil {
			printRouterConfig(name, cfg)
			fmt.Println()
		}
	}
}

func printRouterConfig(name string, cfg *config.RouterConfigEntry) {
	fmt.Printf("Router: %s\n", name)
	fmt.Printf("  Hostname:       %s\n", cfg.Hostname)
	fmt.Printf("  Port:           %d\n", cfg.Port)
	fmt.Printf("  Username:       %s\n", cfg.Username)
	fmt.Printf("  Enabled:        %v\n", cfg.Enabled)
	fmt.Printf("  Module Only:    %v\n", cfg.ModuleOnly)
	fmt.Printf("  Use SSL:        %v\n", cfg.UseSSL)
	fmt.Printf("  Plaintext Auth: %v\n", cfg.PlaintextLogin)

	if cfg.CredentialsFile != "" {
		fmt.Printf("  Credentials:    %s\n", cfg.CredentialsFile)
	}

	if len(cfg.CustomLabels) > 0 {
		fmt.Printf("  Custom Labels:  %v\n", cfg.CustomLabels)
	}

	enabledFeatures := getEnabledFeatures(cfg)
	if len(enabledFeatures) > 0 {
		fmt.Printf("  Features:       %s\n", strings.Join(enabledFeatures, ", "))
	}
}

func getEnabledFeatures(cfg *config.RouterConfigEntry) []string {
	var features []string

	if cfg.Health {
		features = append(features, "health")
	}
	if cfg.InstalledPackages {
		features = append(features, "packages")
	}
	if cfg.DHCP {
		features = append(features, "dhcp")
	}
	if cfg.DHCPLease {
		features = append(features, "dhcp_lease")
	}
	if cfg.Connections {
		features = append(features, "connections")
	}
	if cfg.Interface {
		features = append(features, "interface")
	}
	if cfg.Route {
		features = append(features, "route")
	}
	if cfg.Pool {
		features = append(features, "pool")
	}
	if cfg.Firewall {
		features = append(features, "firewall")
	}
	if cfg.Neighbor {
		features = append(features, "neighbor")
	}
	if cfg.Wireless {
		features = append(features, "wireless")
	}
	if cfg.CAPsMAN {
		features = append(features, "capsman")
	}
	if cfg.LTE {
		features = append(features, "lte")
	}
	if cfg.BGP {
		features = append(features, "bgp")
	}

	return features
}
