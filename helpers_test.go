package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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

// combine mirrors the switch firmware's own split 32-bit-half recombination
// (high*4294967296 + low), used to build expected values for the mixed-layout fixtures.
func combine(high, low uint64) float64 {
	return float64(high*4294967296 + low)
}
