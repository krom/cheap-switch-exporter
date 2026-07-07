package main

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

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
