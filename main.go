// Final unified exporter with full metrics restored

package main

import (
	"crypto/md5"
	"encoding/hex"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/yaml.v3"
)

// ================= CONFIG =================

type RootConfig struct {
	Profiles []map[string]ProfileConfig `yaml:"profiles"`
}

type ProfileConfig struct {
	Profile  string            `yaml:"profile"`
	Address  string            `yaml:"address"`
	Username string            `yaml:"username"`
	Password string            `yaml:"password"`
	Timeout  int               `yaml:"timeout_seconds"`
	PoE      int               `yaml:"poe"`
	Comments map[string]string `yaml:"comments"`
}

type NamedProfile struct {
	Name   string
	Config ProfileConfig
}

// ================= MODELS =================

type Port struct {
	Name       string
	State      string
	LinkStatus string
	TxGood     float64
	TxBad      float64
	RxGood     float64
	RxBad      float64
}

type PortStatus struct {
	Name       string
	SpeedMbps  float64
	FullDuplex float64
}

type PoEPort struct {
	Name    string
	State   float64
	Power   float64
	Type    float64
	Watts   float64
	Voltage float64
	Current float64
}

// ================= COLLECTOR =================

type PortStatsCollector struct {
	profiles []NamedProfile

	portState      *prometheus.Desc
	portLink       *prometheus.Desc
	portTxGood     *prometheus.Desc
	portTxBad      *prometheus.Desc
	portRxGood     *prometheus.Desc
	portRxBad      *prometheus.Desc
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

	mu sync.Mutex
}

func NewCollector(p []NamedProfile) *PortStatsCollector {
	labels := []string{"port", "comment", "instance", "address"}

	return &PortStatsCollector{
		profiles: p,
		portState: prometheus.NewDesc("port_state", "Port admin state", labels, nil),
		portLink:  prometheus.NewDesc("port_link_status", "Link up/down", labels, nil),
		portTxGood: prometheus.NewDesc("port_tx_good_pkt", "TX good packets", labels, nil),
		portTxBad:  prometheus.NewDesc("port_tx_bad_pkt", "TX bad packets", labels, nil),
		portRxGood: prometheus.NewDesc("port_rx_good_pkt", "RX good packets", labels, nil),
		portRxBad:  prometheus.NewDesc("port_rx_bad_pkt", "RX bad packets", labels, nil),
		portSpeed: prometheus.NewDesc("port_link_speed", "Link speed Mbps", labels, nil),
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

func (c *PortStatsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.portState
	ch <- c.portLink
	ch <- c.portTxGood
	ch <- c.portTxBad
	ch <- c.portRxGood
	ch <- c.portRxBad
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

func (c *PortStatsCollector) Collect(ch chan<- prometheus.Metric) {
	c.mu.Lock()
	defer c.mu.Unlock()

	start := time.Now()

	for _, p := range c.profiles {
		ports, err := fetchPorts(p.Config)
		if err != nil {
			c.errors.Inc(); continue
		}
		status, _ := fetchPortStatus(p.Config)
		statusMap := map[string]PortStatus{}
		for _, s := range status { statusMap[s.Name] = s }

		for _, pt := range ports {
			comment := p.Config.Comments[pt.Name]
			labels := []string{pt.Name, comment, p.Name, p.Config.Address}

			ch <- prometheus.MustNewConstMetric(c.portState, prometheus.GaugeValue, state(pt.State), labels...)
			ch <- prometheus.MustNewConstMetric(c.portLink, prometheus.GaugeValue, link(pt.LinkStatus), labels...)
			ch <- prometheus.MustNewConstMetric(c.portTxGood, prometheus.CounterValue, pt.TxGood, labels...)
			ch <- prometheus.MustNewConstMetric(c.portTxBad, prometheus.CounterValue, pt.TxBad, labels...)
			ch <- prometheus.MustNewConstMetric(c.portRxGood, prometheus.CounterValue, pt.RxGood, labels...)
			ch <- prometheus.MustNewConstMetric(c.portRxBad, prometheus.CounterValue, pt.RxBad, labels...)

			if st, ok := statusMap[pt.Name]; ok {
				ch <- prometheus.MustNewConstMetric(c.portSpeed, prometheus.GaugeValue, st.SpeedMbps, labels...)
				ch <- prometheus.MustNewConstMetric(c.portFullDuplex, prometheus.GaugeValue, st.FullDuplex, labels...)
			}
		}

		if p.Config.PoE == 1 {
			cons, poePorts, err := fetchPoE(p.Config)
			if err == nil {
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
	}

	c.lastScrape.Set(time.Since(start).Seconds())
}

// ================= FETCH / PARSE =================

func fetchDoc(cfg ProfileConfig, path, page string) (*goquery.Document, error) {
	client := &http.Client{Timeout: time.Duration(cfg.Timeout) * time.Second}
	req, _ := http.NewRequest("GET", "http://"+cfg.Address+path, nil)
	req.URL.RawQuery = url.Values{"page": {page}}.Encode()
	req.AddCookie(&http.Cookie{Name: "admin", Value: md5hex(cfg.Username + cfg.Password)})
	req.Header.Set("Referer", "http://"+cfg.Address+"/menu.cgi")
	resp, err := client.Do(req)
	if err != nil { return nil, err }
	defer resp.Body.Close()
	return goquery.NewDocumentFromReader(resp.Body)
}

func fetchPorts(cfg ProfileConfig) ([]Port, error) {
	doc, err := fetchDoc(cfg, "/port.cgi", "stats")
	if err != nil { return nil, err }
	var res []Port
	doc.Find("table tr").Each(func(i int, s *goquery.Selection) {
		if i == 0 { return }
		tds := s.Find("td")
		if tds.Length() < 7 { return }
		res = append(res, Port{
			Name: tds.Eq(0).Text(),
			State: tds.Eq(1).Text(),
			LinkStatus: tds.Eq(2).Text(),
			TxGood: parseNum(tds.Eq(3).Text()),
			TxBad:  parseNum(tds.Eq(4).Text()),
			RxGood: parseNum(tds.Eq(5).Text()),
			RxBad:  parseNum(tds.Eq(6).Text()),
		})
	})
	return res, nil
}

func fetchPortStatus(cfg ProfileConfig) ([]PortStatus, error) {
	doc, err := fetchDoc(cfg, "/port.cgi", "status")
	if err != nil { return nil, err }
	var res []PortStatus
	doc.Find("table tr").Each(func(i int, s *goquery.Selection) {
		if i == 0 { return }
		tds := s.Find("td")
		if tds.Length() < 6 { return }
		res = append(res, PortStatus{
			Name: tds.Eq(0).Text(),
			SpeedMbps: speed(tds.Eq(5).Text()),
			FullDuplex: duplex(tds.Eq(3).Text()),
		})
	})
	return res, nil
}

func fetchPoE(cfg ProfileConfig) (float64, []PoEPort, error) {
	docSys, err := fetchDoc(cfg, "/pse_system.cgi", "stats")
	if err != nil { return 0, nil, err }
	cons := parseNum(docSys.Find(`input[name="pse_con_pwr"]`).AttrOr("value", "0"))

	doc, err := fetchDoc(cfg, "/pse_port.cgi", "stats")
	if err != nil { return 0, nil, err }

	var ports []PoEPort
	doc.Find("table tbody tr").Each(func(_ int, s *goquery.Selection) {
		tds := s.Find("td")
		if tds.Length() != 7 { return }
		ports = append(ports, PoEPort{
			Name: tds.Eq(0).Text(),
			State: state(tds.Eq(1).Text()),
			Power: onoff(tds.Eq(2).Text()),
			Type: poeType(tds.Eq(3).Text()),
			Watts: parseNum(tds.Eq(4).Text()),
			Voltage: parseNum(tds.Eq(5).Text()),
			Current: parseNum(tds.Eq(6).Text()),
		})
	})

	return cons, ports, nil
}

// ================= HELPERS =================

func state(s string) float64 { if strings.Contains(s, "Enable") { return 1 }; return 0 }
func link(s string) float64 { if strings.Contains(s, "Up") { return 1 }; return 0 }
func duplex(s string) float64 { if strings.Contains(s, "Full") { return 1 }; return 0 }
func onoff(s string) float64 { if s == "On" { return 1 }; return 0 }
func poeType(s string) float64 { switch s { case "Class1": return 1; case "Class2": return 2; case "Class3": return 3; case "Class4": return 4 }; return 0 }

func speed(s string) float64 {
	s = strings.ToUpper(strings.TrimSpace(s))
	switch s { case "10M": return 10; case "100M": return 100; case "1000M": return 1000; case "2500M": return 2500; case "5000M": return 5000; case "10G": return 10000 }
	return 0
}

func parseNum(s string) float64 {
	s = strings.TrimSpace(strings.TrimPrefix(s, "0-"))
	if s == "" || s == "-" { return 0 }
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func md5hex(s string) string { h := md5.Sum([]byte(s)); return hex.EncodeToString(h[:]) }

// ================= MAIN =================

func main() {
	cfg := RootConfig{}
	b, err := os.ReadFile("config.yaml")
	if err != nil { log.Fatal(err) }
	if err := yaml.Unmarshal(b, &cfg); err != nil { log.Fatal(err) }

	var profiles []NamedProfile
	for _, m := range cfg.Profiles {
		for name, pc := range m {
			if pc.Timeout == 0 { pc.Timeout = 5 }
			profiles = append(profiles, NamedProfile{Name: name, Config: pc})
		}
	}

	collector := NewCollector(profiles)
	prometheus.MustRegister(collector)

	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(":8080", nil)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
}
