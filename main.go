// Updated version with profiles, comments, port status parsing
// NOTE: This is a refactor of the original file

package main

import (
    "crypto/md5"
    "encoding/hex"
    "log"
    "net/http"
    "net/url"
    "os"
    "os/signal"
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

// ---------------- Config ----------------

type RootConfig struct {
	Profiles []map[string]ProfileConfig `yaml:"profiles"`
}

type ProfileConfig struct {
	Profile  string            `yaml:"profile"`
	Address  string            `yaml:"address"`
	Username string            `yaml:"username"`
	Password string            `yaml:"password"`
	PollRate int               `yaml:"poll_rate_seconds"`
	Timeout  int               `yaml:"timeout_seconds"`
	PoE      int               `yaml:"poe"`
	Comments map[string]string `yaml:"comments"`
}

// ---------------- Models ----------------

type Port struct {
	Name       string
	State      string
	LinkStatus string
	TxGoodPkt  uint64
	TxBadPkt   uint64
	RxGoodPkt  uint64
	RxBadPkt   uint64
}

type PortStatus struct {
	Name       string
	SpeedMbps  float64
	FullDuplex float64
}

type PortStatistics struct {
	Ports []Port
}

type PoEStatistics struct {
	Consumption float64
}

// ---------------- Collector ----------------

type PortStatsCollector struct {
	profiles []NamedProfile

	portLinkStatus *prometheus.Desc
	portLinkSpeed  *prometheus.Desc
	portFullDuplex *prometheus.Desc

	lastScrapeDuration prometheus.Gauge
	scrapeErrorsTotal  prometheus.Counter

	mutex sync.Mutex
}

type NamedProfile struct {
	Name   string
	Config ProfileConfig
}

func NewPortStatsCollector(profiles []NamedProfile) *PortStatsCollector {
	labels := []string{"port", "comment", "switch", "address"}

	return &PortStatsCollector{
		profiles: profiles,
		portLinkStatus: prometheus.NewDesc(
			"port_link_status",
			"Link status of the port",
			labels, nil,
		),
		portLinkSpeed: prometheus.NewDesc(
			"port_link_speed",
			"Port link speed in Mbps",
			labels, nil,
		),
		portFullDuplex: prometheus.NewDesc(
			"port_link_full_duplex",
			"Port full duplex (1=Full, 0=Half)",
			labels, nil,
		),
		lastScrapeDuration: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "exporter_last_scrape_duration_seconds",
			Help: "Duration of the last scrape",
		}),
		scrapeErrorsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "exporter_scrape_errors_total",
			Help: "Total number of scrape errors",
		}),
	}
}

func (c *PortStatsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.portLinkStatus
	ch <- c.portLinkSpeed
	ch <- c.portFullDuplex
}

func (c *PortStatsCollector) Collect(ch chan<- prometheus.Metric) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	start := time.Now()

	for _, p := range c.profiles {
		ports, err := fetchPortStatistics(p.Config)
		if err != nil {
			c.scrapeErrorsTotal.Inc()
			continue
		}

		status, err := fetchPortStatus(p.Config)
		if err != nil {
			c.scrapeErrorsTotal.Inc()
			continue
		}

		for _, ps := range ports.Ports {
			comment := p.Config.Comments[ps.Name]

			ch <- prometheus.MustNewConstMetric(
				c.portLinkStatus,
				prometheus.GaugeValue,
				linkStatusToFloat(ps.LinkStatus),
				ps.Name, comment, p.Name, p.Config.Address,
			)
		}

		for _, st := range status {
			comment := p.Config.Comments[st.Name]

			ch <- prometheus.MustNewConstMetric(
				c.portLinkSpeed,
				prometheus.GaugeValue,
				st.SpeedMbps,
				st.Name, comment, p.Name, p.Config.Address,
			)

			ch <- prometheus.MustNewConstMetric(
				c.portFullDuplex,
				prometheus.GaugeValue,
				st.FullDuplex,
				st.Name, comment, p.Name, p.Config.Address,
			)
		}
	}

	c.lastScrapeDuration.Set(time.Since(start).Seconds())
}

// ---------------- Fetch & Parse ----------------

func fetchPortStatistics(cfg ProfileConfig) (PortStatistics, error) {
	doc, err := fetchDocument(cfg, "/port.cgi", "stats")
	if err != nil {
		return PortStatistics{}, err
	}
	return parsePortStatistics(doc)
}

func fetchPortStatus(cfg ProfileConfig) ([]PortStatus, error) {
	doc, err := fetchDocument(cfg, "/port.cgi", "status")
	if err != nil {
		return nil, err
	}
	return parsePortStatus(doc)
}

func fetchDocument(cfg ProfileConfig, path, page string) (*goquery.Document, error) {
	params := url.Values{}
	params.Set("page", page)

	client := &http.Client{Timeout: time.Duration(cfg.Timeout) * time.Second}
	req, _ := http.NewRequest("GET", "http://"+cfg.Address+path, nil)

	req.AddCookie(&http.Cookie{Name: "admin", Value: getMD5Hash(cfg.Username + cfg.Password)})
	req.Header.Set("Referer", "http://"+cfg.Address+"/menu.cgi")
	req.URL.RawQuery = params.Encode()

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return goquery.NewDocumentFromReader(resp.Body)
}

func parsePortStatistics(doc *goquery.Document) (PortStatistics, error) {
	var stats PortStatistics
	doc.Find("table tr").Each(func(i int, s *goquery.Selection) {
		if i == 0 {
			return
		}
		p := Port{}
		s.Find("td").Each(func(j int, td *goquery.Selection) {
			text := strings.TrimSpace(td.Text())
			switch j {
			case 0:
				p.Name = text
			case 2:
				p.LinkStatus = text
			}
		})
		if p.Name != "" {
			stats.Ports = append(stats.Ports, p)
		}
	})
	return stats, nil
}

func parsePortStatus(doc *goquery.Document) ([]PortStatus, error) {
	var res []PortStatus

	doc.Find("table tr").Each(func(i int, s *goquery.Selection) {
		if i == 0 {
			return
		}

		tds := s.Find("td")
		if tds.Length() < 6 {
			return
		}

		name := strings.TrimSpace(tds.Eq(0).Text())
		speed := speedToMbps(strings.TrimSpace(tds.Eq(5).Text()))
		duplex := duplexToFloat(strings.TrimSpace(tds.Eq(3).Text()))

		res = append(res, PortStatus{
			Name:       name,
			SpeedMbps:  speed,
			FullDuplex: duplex,
		})
	})

	return res, nil
}

// ---------------- Helpers ----------------

func linkStatusToFloat(s string) float64 {
	if s == "Link Up" {
		return 1
	}
	return 0
}

func duplexToFloat(s string) float64 {
	if strings.Contains(s, "Full") {
		return 1
	}
	return 0
}

func speedToMbps(s string) float64 {
	s = strings.ToUpper(strings.TrimSpace(s))
	switch s {
	case "10M":
		return 10
	case "100M":
		return 100
	case "1000M":
		return 1000
	case "2500M":
		return 2500
	case "5000M":
		return 5000
	case "10G":
		return 10000
	default:
		return 0
	}
}

func getMD5Hash(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}

// ---------------- Main ----------------

func main() {
	cfg, err := readConfig("config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	var profiles []NamedProfile
	for _, item := range cfg.Profiles {
		for name, pc := range item {
			if pc.Profile == "" {
				pc.Profile = "default"
			}
			if pc.Timeout == 0 {
				pc.Timeout = 5
			}
			profiles = append(profiles, NamedProfile{Name: name, Config: pc})
		}
	}

	collector := NewPortStatsCollector(profiles)
	prometheus.MustRegister(collector)

	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(":8080", nil)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
}

func readConfig(path string) (RootConfig, error) {
	var cfg RootConfig
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	return cfg, yaml.Unmarshal(data, &cfg)
}
