// Final unified exporter with full metrics restored

package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"sort"
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
	Settings GlobalSettings             `yaml:"settings"`
}

// GlobalSettings provides defaults for poll_rate_seconds/timeout_seconds, applied to any
// profile that doesn't set its own value (see main()'s resolution loop).
type GlobalSettings struct {
	PollRateSeconds int `yaml:"poll_rate_seconds"`
	TimeoutSeconds  int `yaml:"timeout_seconds"`
}

type ProfileConfig struct {
	Profile         string            `yaml:"profile"`
	Address         string            `yaml:"address"`
	Username        string            `yaml:"username"`
	Password        string            `yaml:"password"`
	Timeout         int               `yaml:"timeout_seconds"`
	PollRateSeconds int               `yaml:"poll_rate_seconds"`
	PoE             int               `yaml:"poe"`
	Comments        map[string]string `yaml:"comments"`
}

type NamedProfile struct {
	Name   string
	Config ProfileConfig
}

// resolveProfileConfig fills in a profile's Timeout/PollRateSeconds using the precedence
// profile value -> global settings value -> hardcoded default (10s timeout, 60s poll rate).
func resolveProfileConfig(pc ProfileConfig, settings GlobalSettings) ProfileConfig {
	if pc.Timeout == 0 {
		if settings.TimeoutSeconds != 0 {
			pc.Timeout = settings.TimeoutSeconds
		} else {
			pc.Timeout = 10
		}
	}
	if pc.PollRateSeconds == 0 {
		if settings.PollRateSeconds != 0 {
			pc.PollRateSeconds = settings.PollRateSeconds
		} else {
			pc.PollRateSeconds = 60
		}
	}
	return pc
}

// validateProfileConfig checks a resolved profile's Profile/PoE combination is valid,
// returning a descriptive error naming the profile if not.
func validateProfileConfig(name string, pc ProfileConfig) error {
	switch pc.Profile {
	case "", "default", "v2":
	default:
		return fmt.Errorf("profile %q: invalid profile %q (must be \"\", \"default\", or \"v2\")", name, pc.Profile)
	}
	if pc.Profile == "v2" && pc.PoE == 1 {
		return fmt.Errorf("profile %q: poe: 1 is not supported with profile: v2", name)
	}
	return nil
}

// ================= MODELS =================

// CounterKind identifies one of the 8 Tx/Rx x Good/Bad x Pkt/Bytes counter columns a
// switch's port.cgi?page=stats table may report. Its string value is also the exact
// header text used to recognize that column, e.g. <th>TxGoodPkt</th>.
type CounterKind string

const (
	TxGoodPkt   CounterKind = "TxGoodPkt"
	TxBadPkt    CounterKind = "TxBadPkt"
	TxGoodBytes CounterKind = "TxGoodBytes"
	TxBadBytes  CounterKind = "TxBadBytes"
	RxGoodPkt   CounterKind = "RxGoodPkt"
	RxBadPkt    CounterKind = "RxBadPkt"
	RxGoodBytes CounterKind = "RxGoodBytes"
	RxBadBytes  CounterKind = "RxBadBytes"
)

var counterMetricNames = map[CounterKind]string{
	TxGoodPkt:   "port_tx_good_pkt",
	TxBadPkt:    "port_tx_bad_pkt",
	TxGoodBytes: "port_tx_good_bytes",
	TxBadBytes:  "port_tx_bad_bytes",
	RxGoodPkt:   "port_rx_good_pkt",
	RxBadPkt:    "port_rx_bad_pkt",
	RxGoodBytes: "port_rx_good_bytes",
	RxBadBytes:  "port_rx_bad_bytes",
}

var counterMetricHelp = map[CounterKind]string{
	TxGoodPkt:   "TX good packets",
	TxBadPkt:    "TX bad packets",
	TxGoodBytes: "TX good bytes",
	TxBadBytes:  "TX bad bytes",
	RxGoodPkt:   "RX good packets",
	RxBadPkt:    "RX bad packets",
	RxGoodBytes: "RX good bytes",
	RxBadBytes:  "RX bad bytes",
}

type Port struct {
	Name       string
	State      string
	LinkStatus string
	// Counters holds only the counter kinds this switch's page actually reported;
	// a kind absent here was not present in the table header, and must not be
	// emitted (not even as 0).
	Counters map[CounterKind]float64
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

// ================= POLLING STATE =================

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

// ================= COLLECTOR =================

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

// Collect reads each profile's cached snapshot rather than scraping live; background
// pollProfile goroutines (started via StartPolling) are what keep those snapshots fresh.
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

// ================= SWITCH CLIENT =================

// switchClient abstracts the per-profile fetch mechanics for one switch dialect, so
// scrapeOnce doesn't need to know whether it's talking to the default cookie+HTML-scrape
// dialect or the v2 JSON dialect.
type switchClient interface {
	FetchPorts() ([]Port, []PortStatus, error)
	FetchPoE() (float64, []PoEPort, error)
}

// defaultClient wraps the original cookie+HTML-scrape fetch functions, unchanged.
type defaultClient struct {
	cfg ProfileConfig
}

func (c defaultClient) FetchPorts() ([]Port, []PortStatus, error) {
	ports, err := fetchPorts(c.cfg)
	if err != nil {
		return nil, nil, err
	}
	status, _ := fetchPortStatus(c.cfg)
	return ports, status, nil
}

func (c defaultClient) FetchPoE() (float64, []PoEPort, error) {
	return fetchPoE(c.cfg)
}

// newSwitchClient resolves which dialect a profile uses (validated at startup in main(),
// so the default case here is unreachable in practice).
func newSwitchClient(name string, cfg ProfileConfig) switchClient {
	if cfg.Profile == "v2" {
		return v2Client{name: name, cfg: cfg}
	}
	return defaultClient{cfg: cfg}
}

// ================= BACKGROUND POLLING =================

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

// ================= FETCH / PARSE =================

func fetchDoc(cfg ProfileConfig, path, page string) (*goquery.Document, error) {
	client := &http.Client{Timeout: time.Duration(cfg.Timeout) * time.Second}
	req, _ := http.NewRequest("GET", "http://"+cfg.Address+path, nil)
	req.URL.RawQuery = url.Values{"page": {page}}.Encode()
	req.AddCookie(&http.Cookie{Name: "admin", Value: md5hex(cfg.Username + cfg.Password)})
	req.Header.Set("Referer", "http://"+cfg.Address+"/menu.cgi")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return goquery.NewDocumentFromReader(resp.Body)
}

func fetchPorts(cfg ProfileConfig) ([]Port, error) {
	doc, err := fetchDoc(cfg, "/port.cgi", "stats")
	if err != nil {
		return nil, err
	}
	var res []Port
	columnKinds := map[int]CounterKind{}
	doc.Find("table tr").Each(func(i int, s *goquery.Selection) {
		if i == 0 {
			s.Find("th").Each(func(col int, th *goquery.Selection) {
				kind := CounterKind(strings.TrimSpace(th.Text()))
				if _, ok := counterMetricNames[kind]; ok {
					columnKinds[col] = kind
				}
			})
			return
		}
		tds := s.Find("td")
		if tds.Length() < 3+len(columnKinds) {
			return
		}
		pt := Port{
			Name:       tds.Eq(0).Text(),
			State:      tds.Eq(1).Text(),
			LinkStatus: tds.Eq(2).Text(),
			Counters:   map[CounterKind]float64{},
		}
		for col, kind := range columnKinds {
			pt.Counters[kind] = parseNum(tds.Eq(col).Text())
		}
		res = append(res, pt)
	})
	return res, nil
}

func fetchPortStatus(cfg ProfileConfig) ([]PortStatus, error) {
	doc, err := fetchDoc(cfg, "/port.cgi", "status")
	if err != nil {
		return nil, err
	}
	var res []PortStatus
	doc.Find("table tr").Each(func(i int, s *goquery.Selection) {
		if i == 0 {
			return
		}
		tds := s.Find("td")
		if tds.Length() < 6 {
			return
		}
		res = append(res, PortStatus{
			Name:       tds.Eq(0).Text(),
			SpeedMbps:  speed(tds.Eq(5).Text()),
			FullDuplex: duplex(tds.Eq(3).Text()),
		})
	})
	return res, nil
}

func fetchPoE(cfg ProfileConfig) (float64, []PoEPort, error) {
	docSys, err := fetchDoc(cfg, "/pse_system.cgi", "stats")
	if err != nil {
		return 0, nil, err
	}
	cons := parseNum(docSys.Find(`input[name="pse_con_pwr"]`).AttrOr("value", "0"))

	doc, err := fetchDoc(cfg, "/pse_port.cgi", "stats")
	if err != nil {
		return 0, nil, err
	}

	var ports []PoEPort
	doc.Find("table tbody tr").Each(func(_ int, s *goquery.Selection) {
		tds := s.Find("td")
		if tds.Length() != 7 {
			return
		}
		ports = append(ports, PoEPort{
			Name:    tds.Eq(0).Text(),
			State:   state(tds.Eq(1).Text()),
			Power:   onoff(tds.Eq(2).Text()),
			Type:    poeType(tds.Eq(3).Text()),
			Watts:   parseNum(tds.Eq(4).Text()),
			Voltage: parseNum(tds.Eq(5).Text()),
			Current: parseNum(tds.Eq(6).Text()),
		})
	})

	return cons, ports, nil
}

// ================= V2 CLIENT =================

// v2Client talks to the JSON-based "v2" device family (e.g. SKS3200-8E2X): a real login
// call establishing a session, and a JSON status endpoint instead of HTML tables. It has
// no PoE support (FetchPoE is a no-op), matching that this device family exposes no PoE
// pages; profiles combining profile: v2 with poe: 1 are rejected at startup instead.
type v2Client struct {
	name string
	cfg  ProfileConfig
}

// v2Login performs the v2 dialect's login (GET /authorize with the username and password
// MD5-hashed separately) and returns the session cookies from its response. Per design,
// this is called fresh on every FetchPorts call rather than cached/reused, since v2
// sessions are short-lived.
func v2Login(cfg ProfileConfig) ([]*http.Cookie, error) {
	client := &http.Client{Timeout: time.Duration(cfg.Timeout) * time.Second}
	req, _ := http.NewRequest("GET", "http://"+cfg.Address+"/authorize", nil)
	req.URL.RawQuery = url.Values{
		"loginusr": {md5hex(cfg.Username)},
		"loginpwd": {md5hex(cfg.Password)},
	}.Encode()
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return resp.Cookies(), nil
}

// v2PortJSON matches one port entry's shape within /port_statistics.json's response,
// e.g. the "Port_1" value in {"PortNum":"10","Port_1":{...},...,"Port_10":{...}}.
type v2PortJSON struct {
	PortId     string `json:"Port_Id"`
	PortStatus string `json:"Port_Status"`
	LinkStatus string `json:"Link_Status"`
	TxGoodPkt  string `json:"TxGoodPkt"`
	TxBadPkt   string `json:"TxBadPkt"`
	RxGoodPkt  string `json:"RxGoodPkt"`
	RxBadPkt   string `json:"RxBadPkt"`
}

// linkStatusRe matches the v2 dialect's combined link-up/speed/duplex format, e.g.
// "1000MbpsFull" or "10GbpsFull". "Link Down" is handled separately in parseLinkStatus.
var linkStatusRe = regexp.MustCompile(`^(\d+)(Mbps|Gbps)(Full|Half)$`)

// parseLinkStatus splits a v2 Link_Status string into link-up/down, speed (Mbps), and
// duplex. ok is false if s doesn't match any recognized format (including documented-but-
// unverified ones like Half duplex or 10Mbps/5000Mbps, which follow the same pattern but
// have never been observed live).
func parseLinkStatus(s string) (linkUp bool, speedMbps float64, fullDuplex bool, ok bool) {
	s = strings.TrimSpace(s)
	if s == "Link Down" {
		return false, 0, false, true
	}
	m := linkStatusRe.FindStringSubmatch(s)
	if m == nil {
		return false, 0, false, false
	}
	n, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return false, 0, false, false
	}
	if m[2] == "Gbps" {
		n *= 1000
	}
	return true, n, m[3] == "Full", true
}

func (c v2Client) FetchPoE() (float64, []PoEPort, error) {
	return 0, nil, nil
}

func (c v2Client) FetchPorts() ([]Port, []PortStatus, error) {
	cookies, err := v2Login(c.cfg)
	if err != nil {
		return nil, nil, err
	}

	client := &http.Client{Timeout: time.Duration(c.cfg.Timeout) * time.Second}
	req, _ := http.NewRequest("GET", "http://"+c.cfg.Address+"/port_statistics.json", nil)
	for _, ck := range cookies {
		req.AddCookie(ck)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	var raw map[string]json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, nil, err
	}

	// Iterate the actual "Port_N" keys present rather than trusting PortNum as a loop
	// bound (PortNum could disagree with the keys actually present).
	var keys []string
	for k := range raw {
		if strings.HasPrefix(k, "Port_") {
			keys = append(keys, k)
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		ni, erri := strconv.Atoi(strings.TrimPrefix(keys[i], "Port_"))
		nj, errj := strconv.Atoi(strings.TrimPrefix(keys[j], "Port_"))
		if erri == nil && errj == nil {
			return ni < nj
		}
		return keys[i] < keys[j]
	})

	var ports []Port
	var statuses []PortStatus
	for _, k := range keys {
		var p v2PortJSON
		if err := json.Unmarshal(raw[k], &p); err != nil {
			continue
		}
		pt := Port{
			Name:  p.PortId,
			State: p.PortStatus,
			Counters: map[CounterKind]float64{
				TxGoodPkt: parseNum(p.TxGoodPkt),
				TxBadPkt:  parseNum(p.TxBadPkt),
				RxGoodPkt: parseNum(p.RxGoodPkt),
				RxBadPkt:  parseNum(p.RxBadPkt),
			},
		}

		linkUp, speedMbps, fullDuplex, ok := parseLinkStatus(p.LinkStatus)
		if !ok {
			log.Printf("v2 switch client: profile %q port %q: unrecognized Link_Status %q, omitting link/speed/duplex", c.name, pt.Name, p.LinkStatus)
			ports = append(ports, pt)
			continue
		}

		if linkUp {
			pt.LinkStatus = "Link Up"
		} else {
			pt.LinkStatus = "Link Down"
		}
		ports = append(ports, pt)

		fd := 0.0
		if fullDuplex {
			fd = 1
		}
		statuses = append(statuses, PortStatus{Name: pt.Name, SpeedMbps: speedMbps, FullDuplex: fd})
	}

	return ports, statuses, nil
}

// ================= HELPERS =================

func state(s string) float64 {
	if strings.Contains(s, "Enable") {
		return 1
	}
	return 0
}
func link(s string) float64 {
	if strings.Contains(s, "Up") {
		return 1
	}
	return 0
}
func duplex(s string) float64 {
	if strings.Contains(s, "Full") {
		return 1
	}
	return 0
}
func onoff(s string) float64 {
	if s == "On" {
		return 1
	}
	return 0
}
func poeType(s string) float64 {
	switch s {
	case "Class1":
		return 1
	case "Class2":
		return 2
	case "Class3":
		return 3
	case "Class4":
		return 4
	}
	return 0
}

func speed(s string) float64 {
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
	}
	return 0
}

// parseNum parses a table cell's numeric text. Some switch firmware splits a 64-bit
// counter into two 32-bit halves joined by "-" (e.g. "1-901525430"), recombined by the
// page's own JS as high*4294967296 + low; recombine the same way here.
func parseNum(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "-" {
		return 0
	}
	if high, low, ok := strings.Cut(s, "-"); ok {
		if hv, err := strconv.ParseUint(high, 10, 64); err == nil {
			if lv, err := strconv.ParseUint(low, 10, 64); err == nil {
				return float64(hv*4294967296 + lv)
			}
		}
	}
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func md5hex(s string) string { h := md5.Sum([]byte(s)); return hex.EncodeToString(h[:]) }

// ================= MAIN =================

func main() {
	cfg := RootConfig{}
	b, err := os.ReadFile("config.yaml")
	if err != nil {
		log.Fatal(err)
	}
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		log.Fatal(err)
	}

	var profiles []NamedProfile
	for _, m := range cfg.Profiles {
		for name, pc := range m {
			rc := resolveProfileConfig(pc, cfg.Settings)
			if err := validateProfileConfig(name, rc); err != nil {
				log.Fatal(err)
			}
			profiles = append(profiles, NamedProfile{Name: name, Config: rc})
		}
	}

	collector := NewCollector(profiles)
	prometheus.MustRegister(collector)

	ctx, cancel := context.WithCancel(context.Background())
	collector.StartPolling(ctx)

	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(":8080", nil)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	cancel()
}
