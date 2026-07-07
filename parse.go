package main

import (
	"crypto/md5"
	"encoding/hex"
	"strconv"
	"strings"
)

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
