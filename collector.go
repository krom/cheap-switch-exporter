package main

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// profileState holds the most recent successfully scraped data for one profile,
// swapped in by that profile's background poller and read by Collect(). A nil
// error means the last scrape succeeded; a non-nil error means the previous
// successful snapshot (if any) is kept and served until the next successful poll.
type profileState struct {
	mu       sync.RWMutex
	ports    []Port
	status   []PortStatus
	poeCons  float64
	poePorts []PoEPort
	lastErr  error
}

func (s *profileState) snapshot() ([]Port, []PortStatus, float64, []PoEPort, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ports, s.status, s.poeCons, s.poePorts, s.lastErr
}

// PortStatsCollector implements prometheus.Collector, emitting port and PoE metrics for
// every configured profile from each profile's cached snapshot (see profileState and
// StartPolling) rather than scraping live on every /metrics request.
type PortStatsCollector struct {
	profiles []NamedProfile
	states   map[string]*profileState

	portState      *prometheus.Desc
	portLink       *prometheus.Desc
	portCounters   map[CounterKind]*prometheus.Desc
	portSpeed      *prometheus.Desc
	portFullDuplex *prometheus.Desc

	poeSystem *prometheus.Desc
	poeState  *prometheus.Desc
	poePower  *prometheus.Desc
	poeType   *prometheus.Desc
	poeWatts  *prometheus.Desc
	poeVolt   *prometheus.Desc
	poeCurr   *prometheus.Desc

	lastScrape prometheus.Gauge
	errors     prometheus.Counter
}

// NewCollector builds a PortStatsCollector for the given profiles, registering its
// Prometheus metric descriptors. It does not perform any scraping itself; call
// StartPolling to begin populating the cache each profile's Collect() call reads from.
func NewCollector(p []NamedProfile) *PortStatsCollector {
	labels := []string{"port", "comment", "switch", "address"}

	portCounters := map[CounterKind]*prometheus.Desc{}
	for kind, name := range counterMetricNames {
		portCounters[kind] = prometheus.NewDesc(name, counterMetricHelp[kind], labels, nil)
	}

	states := map[string]*profileState{}
	for _, np := range p {
		states[np.Name] = &profileState{}
	}

	return &PortStatsCollector{
		profiles:       p,
		states:         states,
		portState:      prometheus.NewDesc("port_state", "Port admin state", labels, nil),
		portLink:       prometheus.NewDesc("port_link_status", "Link up/down", labels, nil),
		portCounters:   portCounters,
		portSpeed:      prometheus.NewDesc("port_link_speed", "Link speed Mbps", labels, nil),
		portFullDuplex: prometheus.NewDesc("port_link_full_duplex", "Full duplex", labels, nil),

		poeSystem: prometheus.NewDesc("poe_system_consumption_watts", "Total PoE consumption", nil, nil),
		poeState:  prometheus.NewDesc("poe_port_state", "PoE port enabled", labels, nil),
		poePower:  prometheus.NewDesc("poe_port_power_on", "PoE power on", labels, nil),
		poeType:   prometheus.NewDesc("poe_port_type", "PoE class", labels, nil),
		poeWatts:  prometheus.NewDesc("poe_port_watts", "PoE watts", labels, nil),
		poeVolt:   prometheus.NewDesc("poe_port_voltage", "PoE voltage", labels, nil),
		poeCurr:   prometheus.NewDesc("poe_port_current_ma", "PoE current", labels, nil),

		lastScrape: promauto.NewGauge(prometheus.GaugeOpts{Name: "exporter_last_scrape_duration_seconds"}),
		errors:     promauto.NewCounter(prometheus.CounterOpts{Name: "exporter_scrape_errors_total"}),
	}
}

// Describe implements prometheus.Collector.
func (c *PortStatsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.portState
	ch <- c.portLink
	for _, desc := range c.portCounters {
		ch <- desc
	}
	ch <- c.portSpeed
	ch <- c.portFullDuplex
	ch <- c.poeSystem
	ch <- c.poeState
	ch <- c.poePower
	ch <- c.poeType
	ch <- c.poeWatts
	ch <- c.poeVolt
	ch <- c.poeCurr
}

// Collect implements prometheus.Collector. It reads each profile's cached snapshot rather
// than scraping live; background pollProfile goroutines (started via StartPolling) are
// what keep those snapshots fresh.
func (c *PortStatsCollector) Collect(ch chan<- prometheus.Metric) {
	start := time.Now()

	for _, p := range c.profiles {
		ports, status, cons, poePorts, err := c.states[p.Name].snapshot()
		if err != nil && ports == nil {
			continue
		}
		statusMap := map[string]PortStatus{}
		for _, s := range status {
			statusMap[s.Name] = s
		}

		for _, pt := range ports {
			comment := p.Config.Comments[pt.Name]
			labels := []string{pt.Name, comment, p.Name, p.Config.Address}

			ch <- prometheus.MustNewConstMetric(c.portState, prometheus.GaugeValue, state(pt.State), labels...)
			ch <- prometheus.MustNewConstMetric(c.portLink, prometheus.GaugeValue, link(pt.LinkStatus), labels...)
			for kind, v := range pt.Counters {
				ch <- prometheus.MustNewConstMetric(c.portCounters[kind], prometheus.CounterValue, v, labels...)
			}

			if st, ok := statusMap[pt.Name]; ok {
				ch <- prometheus.MustNewConstMetric(c.portSpeed, prometheus.GaugeValue, st.SpeedMbps, labels...)
				ch <- prometheus.MustNewConstMetric(c.portFullDuplex, prometheus.GaugeValue, st.FullDuplex, labels...)
			}
		}

		if p.Config.PoE == 1 && poePorts != nil {
			ch <- prometheus.MustNewConstMetric(c.poeSystem, prometheus.GaugeValue, cons)
			for _, pp := range poePorts {
				comment := p.Config.Comments[pp.Name]
				labels := []string{pp.Name, comment, p.Name, p.Config.Address}
				ch <- prometheus.MustNewConstMetric(c.poeState, prometheus.GaugeValue, pp.State, labels...)
				ch <- prometheus.MustNewConstMetric(c.poePower, prometheus.GaugeValue, pp.Power, labels...)
				ch <- prometheus.MustNewConstMetric(c.poeType, prometheus.GaugeValue, pp.Type, labels...)
				ch <- prometheus.MustNewConstMetric(c.poeWatts, prometheus.GaugeValue, pp.Watts, labels...)
				ch <- prometheus.MustNewConstMetric(c.poeVolt, prometheus.GaugeValue, pp.Voltage, labels...)
				ch <- prometheus.MustNewConstMetric(c.poeCurr, prometheus.GaugeValue, pp.Current, labels...)
			}
		}
	}

	c.lastScrape.Set(time.Since(start).Seconds())
}
