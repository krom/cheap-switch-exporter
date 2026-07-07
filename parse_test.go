package main

import "testing"

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
