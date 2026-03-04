package collector

// AllCollectors returns all standard collectors to register with the server.
func AllCollectors() []Collector {
	return []Collector{
		NewHealthCollector(),
		NewSystemResourceCollector(),
		NewIdentityCollector(),
		NewInterfaceCollector(),
		NewMonitorCollector(),
		NewHWHealthCollector(),
		NewPOECollector(),
		NewDHCPCollector(),
		NewPoolCollector(),
		NewRouteCollector(),
		NewFirewallCollector(),
		NewNeighborCollector(),
		NewNetwatchCollector(),
		NewPackageCollector(),
		NewUserCollector(),
		NewQueueCollector(),
		NewConnectionCollector(),
		NewPublicIPCollector(),
		NewSwitchPortCollector(),
		NewBGPCollector(),
		NewDNSCollector(),
		NewRoutingStatsCollector(),
		NewSystemUpdateCollector(),
	}
}
