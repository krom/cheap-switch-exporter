// Command cheap-switch-exporter is a Prometheus exporter for budget network switches
// that don't support SNMP: it scrapes port/PoE statistics from each configured switch's
// web management interface and serves them on /metrics.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	profiles, err := loadConfig("config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	collector := NewCollector(profiles)
	prometheus.MustRegister(collector)

	ctx, cancel := context.WithCancel(context.Background())
	collector.StartPolling(ctx)

	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(":8080", nil)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	cancel()
}
