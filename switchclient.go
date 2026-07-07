package main

// switchClient abstracts the per-profile fetch mechanics for one switch dialect, so
// scrapeOnce doesn't need to know whether it's talking to the default cookie+HTML-scrape
// dialect or the v2 JSON dialect.
type switchClient interface {
	FetchPorts() ([]Port, []PortStatus, error)
	FetchPoE() (float64, []PoEPort, error)
}

// newSwitchClient resolves which dialect a profile uses (validated at startup in main(),
// so the default case here is unreachable in practice).
func newSwitchClient(name string, cfg ProfileConfig) switchClient {
	if cfg.Profile == "v2" {
		return v2Client{name: name, cfg: cfg}
	}
	return defaultClient{cfg: cfg}
}
