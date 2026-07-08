package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

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

func fetchDoc(cfg ProfileConfig, path, page string) (*goquery.Document, error) {
	client := &http.Client{Timeout: time.Duration(cfg.Timeout) * time.Second}
	req, _ := http.NewRequest("GET", "http://"+cfg.Address+path, nil)
	req.URL.RawQuery = url.Values{"page": {page}}.Encode()
	req.AddCookie(&http.Cookie{Name: "admin", Value: md5hex(cfg.Username + cfg.Password)})
	req.Header.Set("Referer", "http://"+cfg.Address+"/menu.cgi")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s?page=%s: %w", path, page, err)
	}
	defer resp.Body.Close()
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse %s?page=%s response: %w", path, page, err)
	}
	return doc, nil
}

func fetchPorts(cfg ProfileConfig) ([]Port, error) {
	doc, err := fetchDoc(cfg, "/port.cgi", "stats")
	if err != nil {
		return nil, fmt.Errorf("fetch ports: %w", err)
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

// speedDuplexColumns locates, within a port.cgi?page=status data table's group-header row,
// which data column holds the live Duplex/Speed value(s). Two real layouts are known: one
// with separate "Duplex Mode" and "Negotiation Speed" colspan-2 groups, and one with a single
// merged "Speed/Duplex" colspan-2 group. Each group's second (colspan) column is its "Actual"
// value; the first is the configured target and is not used here.
type speedDuplexColumns struct {
	duplexActual      int
	speedActual       int
	speedDuplexActual int
	merged            bool
	found             bool
}

func findSpeedDuplexColumns(headerRow *goquery.Selection) speedDuplexColumns {
	var cols speedDuplexColumns
	offset := 0
	headerRow.Find("th").Each(func(_ int, th *goquery.Selection) {
		label := strings.TrimSpace(th.Text())
		span := 1
		if cs, ok := th.Attr("colspan"); ok {
			if n, err := strconv.Atoi(cs); err == nil && n > 0 {
				span = n
			}
		}
		switch label {
		case "Duplex Mode":
			cols.duplexActual = offset + span - 1
			cols.found = true
		case "Negotiation Speed":
			cols.speedActual = offset + span - 1
			cols.found = true
		case "Speed/Duplex":
			cols.speedDuplexActual = offset + span - 1
			cols.merged = true
			cols.found = true
		}
		offset += span
	})
	return cols
}

func fetchPortStatus(cfg ProfileConfig) ([]PortStatus, error) {
	doc, err := fetchDoc(cfg, "/port.cgi", "status")
	if err != nil {
		return nil, fmt.Errorf("fetch port status: %w", err)
	}

	dataTable := doc.Find("table").Last()
	cols := findSpeedDuplexColumns(dataTable.Find("tr").First())

	var res []PortStatus
	dataTable.Find("tr").Each(func(_ int, s *goquery.Selection) {
		tds := s.Find("td")
		if tds.Length() == 0 {
			return
		}
		ps := PortStatus{Name: tds.Eq(0).Text()}
		if cols.found {
			if cols.merged {
				ps.SpeedMbps, ps.FullDuplex = parseSpeedDuplex(tds.Eq(cols.speedDuplexActual).Text())
			} else {
				ps.SpeedMbps = speed(tds.Eq(cols.speedActual).Text())
				ps.FullDuplex = duplex(tds.Eq(cols.duplexActual).Text())
			}
		}
		res = append(res, ps)
	})
	return res, nil
}

func fetchPoE(cfg ProfileConfig) (float64, []PoEPort, error) {
	docSys, err := fetchDoc(cfg, "/pse_system.cgi", "stats")
	if err != nil {
		return 0, nil, fmt.Errorf("fetch PoE system stats: %w", err)
	}
	cons := parseNum(docSys.Find(`input[name="pse_con_pwr"]`).AttrOr("value", "0"))

	doc, err := fetchDoc(cfg, "/pse_port.cgi", "stats")
	if err != nil {
		return 0, nil, fmt.Errorf("fetch PoE port stats: %w", err)
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
