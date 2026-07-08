package main

import (
	"reflect"
	"testing"
)

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

func TestFetchPortStatus_SeparateDuplexSpeedGroups(t *testing.T) {
	_, cfg := newFixtureServer(t, map[string]string{
		"/port.cgi?page=status": "examples/sks3200-5e1x_port_status.html",
	})

	got, err := fetchPortStatus(cfg)
	if err != nil {
		t.Fatalf("fetchPortStatus: %v", err)
	}

	want := map[string]PortStatus{
		"Port 1": {Name: "Port 1", SpeedMbps: 10, FullDuplex: 0},
		"Port 2": {Name: "Port 2", SpeedMbps: 10, FullDuplex: 0},
		"Port 3": {Name: "Port 3", SpeedMbps: 10, FullDuplex: 0},
		"Port 4": {Name: "Port 4", SpeedMbps: 10, FullDuplex: 0},
		"Port 5": {Name: "Port 5", SpeedMbps: 10, FullDuplex: 0},
		"Port 6": {Name: "Port 6", SpeedMbps: 10000, FullDuplex: 1},
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

func TestFetchPortStatus_MergedSpeedDuplexGroup(t *testing.T) {
	_, cfg := newFixtureServer(t, map[string]string{
		"/port.cgi?page=status": "examples/sks3200-8e1x-p_port_status.html",
	})

	got, err := fetchPortStatus(cfg)
	if err != nil {
		t.Fatalf("fetchPortStatus: %v", err)
	}

	want := map[string]PortStatus{
		"Port 1": {Name: "Port 1", SpeedMbps: 1000, FullDuplex: 1},
		"Port 2": {Name: "Port 2", SpeedMbps: 1000, FullDuplex: 1},
		"Port 3": {Name: "Port 3", SpeedMbps: 1000, FullDuplex: 1},
		"Port 4": {Name: "Port 4", SpeedMbps: 2500, FullDuplex: 1},
		"Port 5": {Name: "Port 5", SpeedMbps: 0, FullDuplex: 0},
		"Port 6": {Name: "Port 6", SpeedMbps: 2500, FullDuplex: 1},
		"Port 7": {Name: "Port 7", SpeedMbps: 0, FullDuplex: 0},
		"Port 8": {Name: "Port 8", SpeedMbps: 0, FullDuplex: 0},
		"Port 9": {Name: "Port 9", SpeedMbps: 10000, FullDuplex: 1},
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
