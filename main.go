package main

import (
	"log"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"tinygo.org/x/bluetooth"
)

func main() {
	registry := prometheus.NewRegistry()
	exporter := newExporter(sensorMACs, newMetrics(registry), os.Stdout)
	serve := func(handler http.Handler) error {
		return http.ListenAndServe(listenAddress, handler)
	}
	if err := run(bluetooth.DefaultAdapter, exporter, newMetricsHandler(registry), serve); err != nil {
		log.Fatal(err)
	}
}
