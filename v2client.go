package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

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
		return nil, fmt.Errorf("v2 login: %w", err)
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
		return nil, nil, fmt.Errorf("v2 fetch port_statistics.json: %w", err)
	}
	defer resp.Body.Close()

	var raw map[string]json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, nil, fmt.Errorf("v2 decode port_statistics.json: %w", err)
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
			Name:  "Port " + p.PortId,
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
