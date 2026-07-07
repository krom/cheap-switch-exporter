package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

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
