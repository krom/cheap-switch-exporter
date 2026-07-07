package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// ================= HELPER FUNCTION TESTS =================

func TestState(t *testing.T) {
	cases := []struct {
		in   string
		want float64
	}{
		{"Enable", 1},
		{"Disable", 0},
		{"", 0},
	}
	for _, c := range cases {
		if got := state(c.in); got != c.want {
			t.Errorf("state(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestLink(t *testing.T) {
	cases := []struct {
		in   string
		want float64
	}{
		{"Link Up", 1},
		{"Link Down", 0},
		{"", 0},
	}
	for _, c := range cases {
		if got := link(c.in); got != c.want {
			t.Errorf("link(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestDuplex(t *testing.T) {
	cases := []struct {
		in   string
		want float64
	}{
		{"Full", 1},
		{"Half", 0},
		{"", 0},
	}
	for _, c := range cases {
		if got := duplex(c.in); got != c.want {
			t.Errorf("duplex(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestOnOff(t *testing.T) {
	cases := []struct {
		in   string
		want float64
	}{
		{"On", 1},
		{"Off", 0},
		{"", 0},
	}
	for _, c := range cases {
		if got := onoff(c.in); got != c.want {
			t.Errorf("onoff(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestPoeType(t *testing.T) {
	cases := []struct {
		in   string
		want float64
	}{
		{"Class1", 1},
		{"Class2", 2},
		{"Class3", 3},
		{"Class4", 4},
		{"Unknown", 0},
	}
	for _, c := range cases {
		if got := poeType(c.in); got != c.want {
			t.Errorf("poeType(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestSpeed(t *testing.T) {
	cases := []struct {
		in   string
		want float64
	}{
		{"10M", 10},
		{"100M", 100},
		{"1000M", 1000},
		{"2500M", 2500},
		{"5000M", 5000},
		{"10G", 10000},
		{" 1000m ", 1000},
		{"Unknown", 0},
	}
	for _, c := range cases {
		if got := speed(c.in); got != c.want {
			t.Errorf("speed(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseNum(t *testing.T) {
	cases := []struct {
		in   string
		want float64
	}{
		{"1-901525430", 1*4294967296 + 901525430},
		{"0-2549448529", 2549448529},
		{"71817", 71817},
		{"", 0},
		{"-", 0},
		{"-123", -123},
	}
	for _, c := range cases {
		if got := parseNum(c.in); got != c.want {
			t.Errorf("parseNum(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseLinkStatus(t *testing.T) {
	cases := []struct {
		in             string
		wantUp         bool
		wantSpeedMbps  float64
		wantFullDuplex bool
		wantOk         bool
	}{
		{"Link Down", false, 0, false, true},
		{"1000MbpsFull", true, 1000, true, true},
		{"2500MbpsFull", true, 2500, true, true},
		{"100MbpsFull", true, 100, true, true},
		{"10GbpsFull", true, 10000, true, true},
		{"garbage", false, 0, false, false},
	}
	for _, c := range cases {
		gotUp, gotSpeed, gotFull, gotOk := parseLinkStatus(c.in)
		if gotUp != c.wantUp || gotSpeed != c.wantSpeedMbps || gotFull != c.wantFullDuplex || gotOk != c.wantOk {
			t.Errorf("parseLinkStatus(%q) = (%v, %v, %v, %v), want (%v, %v, %v, %v)",
				c.in, gotUp, gotSpeed, gotFull, gotOk, c.wantUp, c.wantSpeedMbps, c.wantFullDuplex, c.wantOk)
		}
	}
}

func TestValidateProfileConfig(t *testing.T) {
	cases := []struct {
		name    string
		pc      ProfileConfig
		wantErr bool
	}{
		{"empty profile ok", ProfileConfig{Profile: ""}, false},
		{"default profile ok", ProfileConfig{Profile: "default"}, false},
		{"v2 profile ok", ProfileConfig{Profile: "v2"}, false},
		{"invalid profile rejected", ProfileConfig{Profile: "bogus"}, true},
		{"v2 with poe rejected", ProfileConfig{Profile: "v2", PoE: 1}, true},
		{"default with poe ok", ProfileConfig{Profile: "default", PoE: 1}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := validateProfileConfig("test-profile", c.pc)
			if (err != nil) != c.wantErr {
				t.Errorf("validateProfileConfig(%+v) error = %v, wantErr %v", c.pc, err, c.wantErr)
			}
		})
	}
}

// ================= FIXTURE-DRIVEN FETCH TESTS =================

// newFixtureServer starts an httptest.Server that serves fixturePath for any request
// matching path and page query param, and returns a ProfileConfig pointed at it.
func newFixtureServer(t *testing.T, routes map[string]string) (*httptest.Server, ProfileConfig) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Path + "?page=" + r.URL.Query().Get("page")
		fixture, ok := routes[key]
		if !ok {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, fixture)
	}))
	t.Cleanup(srv.Close)

	cfg := ProfileConfig{
		Address: strings.TrimPrefix(srv.URL, "http://"),
		Timeout: 5,
	}
	return srv, cfg
}

func portCounters(t *testing.T, ports []Port, name string) map[CounterKind]float64 {
	t.Helper()
	for _, p := range ports {
		if p.Name == name {
			return p.Counters
		}
	}
	t.Fatalf("port %q not found", name)
	return nil
}

func TestFetchPorts_PacketOnlyLayout(t *testing.T) {
	_, cfg := newFixtureServer(t, map[string]string{
		"/port.cgi?page=stats": "examples/4.html",
	})

	ports, err := fetchPorts(cfg)
	if err != nil {
		t.Fatalf("fetchPorts: %v", err)
	}

	want := map[string]map[CounterKind]float64{
		"Port 1": {TxGoodPkt: 94452802, TxBadPkt: 0, RxGoodPkt: 37704960, RxBadPkt: 1},
		"Port 2": {TxGoodPkt: 48029475, TxBadPkt: 0, RxGoodPkt: 12326209, RxBadPkt: 0},
		"Port 3": {TxGoodPkt: 93227737, TxBadPkt: 0, RxGoodPkt: 45039604, RxBadPkt: 0},
		"Port 4": {TxGoodPkt: 210870280, TxBadPkt: 0, RxGoodPkt: 172636180, RxBadPkt: 0},
		"Port 5": {TxGoodPkt: 0, TxBadPkt: 0, RxGoodPkt: 0, RxBadPkt: 0},
		"Port 6": {TxGoodPkt: 321472130, TxBadPkt: 0, RxGoodPkt: 279670430, RxBadPkt: 0},
		"Port 7": {TxGoodPkt: 0, TxBadPkt: 0, RxGoodPkt: 0, RxBadPkt: 0},
		"Port 8": {TxGoodPkt: 0, TxBadPkt: 0, RxGoodPkt: 0, RxBadPkt: 0},
		"Port 9": {TxGoodPkt: 531136375, TxBadPkt: 0, RxGoodPkt: 647023724, RxBadPkt: 0},
	}
	if len(ports) != len(want) {
		t.Fatalf("got %d ports, want %d", len(ports), len(want))
	}
	for name, wantCounters := range want {
		got := portCounters(t, ports, name)
		if !reflect.DeepEqual(got, wantCounters) {
			t.Errorf("%s: Counters = %v, want %v", name, got, wantCounters)
		}
	}
}

// combine mirrors the switch firmware's own split 32-bit-half recombination
// (high*4294967296 + low), used to build expected values for the mixed-layout fixtures.
func combine(high, low uint64) float64 {
	return float64(high*4294967296 + low)
}

func TestFetchPorts_MixedPacketAndByteLayout(t *testing.T) {
	fixtures := map[string]map[string]map[CounterKind]float64{
		"examples/1.html": {
			"Port 1": {TxGoodPkt: 0, RxGoodPkt: 0, TxGoodBytes: 0, RxGoodBytes: 0},
			"Port 6": {TxGoodPkt: 71817, RxGoodPkt: 20357564, TxGoodBytes: 4759071, RxGoodBytes: 2423989663},
		},
		"examples/2.html": {
			"Port 1": {
				TxGoodPkt: combine(1, 901525430), RxGoodPkt: combine(0, 2549448529),
				TxGoodBytes: combine(1063, 946136370), RxGoodBytes: combine(282, 1722221385),
			},
			"Port 4": {TxGoodPkt: 0, RxGoodPkt: 0, TxGoodBytes: 0, RxGoodBytes: 0},
			"Port 5": {
				TxGoodPkt: combine(0, 30699438), RxGoodPkt: combine(0, 7086695),
				TxGoodBytes: combine(3, 2941890757), RxGoodBytes: combine(0, 1246160469),
			},
		},
		"examples/3.html": {
			"Port 2": {TxGoodPkt: 0, RxGoodPkt: 0, TxGoodBytes: 0, RxGoodBytes: 0},
			"Port 3": {
				TxGoodPkt: combine(1, 889979427), RxGoodPkt: combine(0, 2543295741),
				TxGoodBytes: combine(1057, 893168878), RxGoodBytes: combine(279, 2841856798),
			},
		},
	}

	for fixture, want := range fixtures {
		t.Run(fixture, func(t *testing.T) {
			_, cfg := newFixtureServer(t, map[string]string{
				"/port.cgi?page=stats": fixture,
			})
			ports, err := fetchPorts(cfg)
			if err != nil {
				t.Fatalf("fetchPorts: %v", err)
			}
			for name, wantCounters := range want {
				got := portCounters(t, ports, name)
				if !reflect.DeepEqual(got, wantCounters) {
					t.Errorf("%s: Counters = %v, want %v", name, got, wantCounters)
				}
				for kind := range got {
					if kind == TxBadPkt || kind == RxBadPkt {
						t.Errorf("%s: unexpected bad-packet counter %v present", name, kind)
					}
				}
			}
		})
	}
}

func TestFetchPortStatus(t *testing.T) {
	_, cfg := newFixtureServer(t, map[string]string{
		"/port.cgi?page=status": "testdata/port_status.html",
	})

	got, err := fetchPortStatus(cfg)
	if err != nil {
		t.Fatalf("fetchPortStatus: %v", err)
	}

	want := map[string]PortStatus{
		"Port 1": {Name: "Port 1", SpeedMbps: 1000, FullDuplex: 1},
		"Port 2": {Name: "Port 2", SpeedMbps: 100, FullDuplex: 0},
		"Port 3": {Name: "Port 3", SpeedMbps: 10000, FullDuplex: 1},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d statuses, want %d", len(got), len(want))
	}
	for _, s := range got {
		w, ok := want[s.Name]
		if !ok {
			t.Fatalf("unexpected port %q", s.Name)
		}
		if s != w {
			t.Errorf("%s: got %+v, want %+v", s.Name, s, w)
		}
	}
}

func TestFetchPoE(t *testing.T) {
	_, cfg := newFixtureServer(t, map[string]string{
		"/pse_system.cgi?page=stats": "testdata/pse_system.html",
		"/pse_port.cgi?page=stats":   "testdata/pse_port.html",
	})

	cons, ports, err := fetchPoE(cfg)
	if err != nil {
		t.Fatalf("fetchPoE: %v", err)
	}
	if cons != 15.7 {
		t.Errorf("consumption = %v, want 15.7", cons)
	}

	want := map[string]PoEPort{
		"Port 1": {Name: "Port 1", State: 1, Power: 1, Type: 2, Watts: 5.4, Voltage: 53.2, Current: 102},
		"Port 2": {Name: "Port 2", State: 0, Power: 0, Type: 1, Watts: 0, Voltage: 0, Current: 0},
	}
	if len(ports) != len(want) {
		t.Fatalf("got %d PoE ports, want %d", len(ports), len(want))
	}
	for _, p := range ports {
		w, ok := want[p.Name]
		if !ok {
			t.Fatalf("unexpected port %q", p.Name)
		}
		if p != w {
			t.Errorf("%s: got %+v, want %+v", p.Name, p, w)
		}
	}
}

func TestV2ClientFetchPorts(t *testing.T) {
	var requestOrder []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestOrder = append(requestOrder, r.URL.Path)
		switch r.URL.Path {
		case "/authorize":
			http.SetCookie(w, &http.Cookie{Name: "user", Value: "u"})
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "s"})
			w.WriteHeader(http.StatusOK)
		case "/port_statistics.json":
			http.ServeFile(w, r, "examples/sks3200-8e2x_port_statistics.json")
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := v2Client{
		name: "sw1",
		cfg: ProfileConfig{
			Address:  strings.TrimPrefix(srv.URL, "http://"),
			Username: "admin",
			Password: "admin",
			Timeout:  5,
		},
	}

	ports, statuses, err := client.FetchPorts()
	if err != nil {
		t.Fatalf("FetchPorts: %v", err)
	}

	if len(requestOrder) < 2 || requestOrder[0] != "/authorize" || requestOrder[1] != "/port_statistics.json" {
		t.Fatalf("request order = %v, want [/authorize /port_statistics.json ...]", requestOrder)
	}

	if len(ports) != 10 {
		t.Fatalf("got %d ports, want 10", len(ports))
	}

	port1 := portCounters(t, ports, "Port 1")
	wantPort1 := map[CounterKind]float64{TxGoodPkt: 136463660, TxBadPkt: 0, RxGoodPkt: 208345310, RxBadPkt: 0}
	if !reflect.DeepEqual(port1, wantPort1) {
		t.Errorf("port 1 Counters = %v, want %v", port1, wantPort1)
	}

	statusByName := map[string]PortStatus{}
	for _, s := range statuses {
		statusByName[s.Name] = s
	}
	if s := statusByName["Port 1"]; s.SpeedMbps != 1000 || s.FullDuplex != 1 {
		t.Errorf("port 1 status = %+v, want SpeedMbps=1000 FullDuplex=1", s)
	}
	if s := statusByName["Port 9"]; s.SpeedMbps != 10000 || s.FullDuplex != 1 {
		t.Errorf("port 9 status = %+v, want SpeedMbps=10000 FullDuplex=1", s)
	}
	if s, ok := statusByName["Port 2"]; !ok || s.SpeedMbps != 0 || s.FullDuplex != 0 {
		t.Errorf("port 2 (Link Down) status = %+v, ok=%v, want SpeedMbps=0 FullDuplex=0 ok=true", s, ok)
	}

	for _, p := range ports {
		if p.Name == "Port 1" && link(p.LinkStatus) != 1 {
			t.Errorf("port 1 LinkStatus = %q, want link()==1", p.LinkStatus)
		}
		if p.Name == "Port 2" && link(p.LinkStatus) != 0 {
			t.Errorf("port 2 LinkStatus = %q, want link()==0", p.LinkStatus)
		}
		if p.Name == "Port 1" && state(p.State) != 1 {
			t.Errorf("port 1 State = %q, want state()==1", p.State)
		}
	}
}

// ================= CONFIG RESOLUTION TESTS =================

func TestResolveProfileConfig(t *testing.T) {
	cases := []struct {
		name         string
		pc           ProfileConfig
		settings     GlobalSettings
		wantTimeout  int
		wantPollRate int
	}{
		{
			name:         "nothing set uses hardcoded defaults",
			pc:           ProfileConfig{},
			settings:     GlobalSettings{},
			wantTimeout:  10,
			wantPollRate: 60,
		},
		{
			name:         "global settings used when profile unset",
			pc:           ProfileConfig{},
			settings:     GlobalSettings{PollRateSeconds: 30, TimeoutSeconds: 8},
			wantTimeout:  8,
			wantPollRate: 30,
		},
		{
			name:         "profile value takes precedence over global settings",
			pc:           ProfileConfig{Timeout: 20, PollRateSeconds: 15},
			settings:     GlobalSettings{PollRateSeconds: 30, TimeoutSeconds: 8},
			wantTimeout:  20,
			wantPollRate: 15,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := resolveProfileConfig(c.pc, c.settings)
			if got.Timeout != c.wantTimeout {
				t.Errorf("Timeout = %d, want %d", got.Timeout, c.wantTimeout)
			}
			if got.PollRateSeconds != c.wantPollRate {
				t.Errorf("PollRateSeconds = %d, want %d", got.PollRateSeconds, c.wantPollRate)
			}
		})
	}
}

// ================= CACHED-COLLECT TESTS =================

func TestCollectFromCacheAndPollingLifecycle(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		http.ServeFile(w, r, "examples/4.html")
	}))
	defer srv.Close()

	profile := NamedProfile{
		Name: "sw1",
		Config: ProfileConfig{
			Address:         strings.TrimPrefix(srv.URL, "http://"),
			Timeout:         5,
			PollRateSeconds: 60,
		},
	}
	collector := NewCollector([]NamedProfile{profile})

	// Simulate what a background poller does: populate the cache with one scrape.
	scrapeOnce(profile, collector.states[profile.Name], collector.errors)
	afterScrape := atomic.LoadInt32(&hits)
	if afterScrape == 0 {
		t.Fatal("scrapeOnce made no HTTP requests")
	}

	ch := make(chan prometheus.Metric, 1024)
	collector.Collect(ch)
	close(ch)
	count := 0
	for range ch {
		count++
	}
	if count == 0 {
		t.Fatal("Collect() emitted no metrics")
	}
	if got := atomic.LoadInt32(&hits); got != afterScrape {
		t.Errorf("Collect() triggered a live scrape: hits went from %d to %d", afterScrape, got)
	}

	// StartPolling should keep scraping until its context is cancelled, then stop.
	// Reuses this test's collector rather than creating a new one, since NewCollector
	// registers metrics into Prometheus's global default registry and a second call
	// within the same test binary would panic on duplicate registration.
	ctx, cancel := context.WithCancel(context.Background())
	collector.StartPolling(ctx)

	time.Sleep(50 * time.Millisecond)
	cancel()
	afterCancel := atomic.LoadInt32(&hits)
	if afterCancel <= afterScrape {
		t.Fatal("StartPolling made no HTTP requests before cancellation")
	}

	// Long enough that a 60s-interval tick could never legitimately fire again;
	// this just confirms the goroutine exited rather than continuing to poll.
	time.Sleep(50 * time.Millisecond)
	if got := atomic.LoadInt32(&hits); got != afterCancel {
		t.Errorf("poller kept scraping after ctx cancel: hits went from %d to %d", afterCancel, got)
	}
}
