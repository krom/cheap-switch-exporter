package main

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// scrapeOnce fetches fresh data for one profile and swaps it into st. On a FetchPorts
// failure, the previous successful snapshot (if any) is left in place and st.lastErr
// records the failure, so Collect() keeps serving the last-known-good data.
func scrapeOnce(profile NamedProfile, st *profileState, errs prometheus.Counter) {
	client := newSwitchClient(profile.Name, profile.Config)

	ports, status, err := client.FetchPorts()
	if err != nil {
		errs.Inc()
		st.mu.Lock()
		st.lastErr = err
		st.mu.Unlock()
		return
	}

	var cons float64
	var poePorts []PoEPort
	if profile.Config.PoE == 1 {
		cons, poePorts, _ = client.FetchPoE()
	}

	st.mu.Lock()
	st.ports = ports
	st.status = status
	st.poeCons = cons
	st.poePorts = poePorts
	st.lastErr = nil
	st.mu.Unlock()
}

// pollProfile scrapes profile immediately, then again on every tick of interval, until
// ctx is cancelled.
func pollProfile(ctx context.Context, profile NamedProfile, st *profileState, interval time.Duration, errs prometheus.Counter) {
	scrapeOnce(profile, st, errs)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			scrapeOnce(profile, st, errs)
		}
	}
}

// StartPolling launches one background poller goroutine per profile, each scraping on
// its own resolved PollRateSeconds interval, until ctx is cancelled.
func (c *PortStatsCollector) StartPolling(ctx context.Context) {
	for _, p := range c.profiles {
		interval := time.Duration(p.Config.PollRateSeconds) * time.Second
		go pollProfile(ctx, p, c.states[p.Name], interval, c.errors)
	}
}
