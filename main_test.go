package main

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
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
