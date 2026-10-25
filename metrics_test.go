package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestMetricsUpdateAndSensorIsolation(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := newMetrics(reg)
	reading := sampleMeasurement()
	m.update(reading, "Kitchen", -67)
	assertMetrics(t, reg, reading.MAC, "Kitchen", sampleMetricValues(), 1)

	second := measurement{MAC: "a4:c1:38:6e:21:d7", Temperature: -1, Humidity: 10000,
		BatteryMillivolts: 2500, BatteryPercentage: 0, Sequence: 255}
	m.update(second, "", -90)
	secondValues := map[string]float64{"hub_temperature_celsius": -1, "hub_humidity": 10000,
		"hub_battery_mv": 2500, "hub_battery_percentage": 0, "hub_rssi": -90, "hub_msg_cnt": 255}
	assertMetrics(t, reg, second.MAC, "", secondValues, 2)
	assertMetrics(t, reg, reading.MAC, "Kitchen", sampleMetricValues(), 2)

	// New readings replace previous values, including a wrapped sequence.
	second.Temperature, second.Humidity, second.Sequence = -1250, 5001, 0
	m.update(second, "", -80)
	secondValues["hub_temperature_celsius"] = -1250
	secondValues["hub_humidity"] = 5001
	secondValues["hub_msg_cnt"] = 0
	secondValues["hub_rssi"] = -80
	assertMetrics(t, reg, second.MAC, "", secondValues, 2)
	assertMetrics(t, reg, reading.MAC, "Kitchen", sampleMetricValues(), 2)
}

func TestMetricsHTTPHandler(t *testing.T) {
	reg := prometheus.NewRegistry()
	newMetrics(reg).update(sampleMeasurement(), "Kitchen", -67)
	handler := newMetricsHandler(reg)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d", response.Code)
	}
	if !strings.Contains(response.Header().Get("Content-Type"), "text/plain") {
		t.Fatalf("unexpected content type %q", response.Header().Get("Content-Type"))
	}
	for _, line := range []string{
		`hub_temperature_celsius{mac="a4:c1:38:e8:a7:92",name="Kitchen"} 2345`,
		`hub_humidity{mac="a4:c1:38:e8:a7:92",name="Kitchen"} 5678`,
		`hub_battery_mv{mac="a4:c1:38:e8:a7:92",name="Kitchen"} 3000`,
		`hub_battery_percentage{mac="a4:c1:38:e8:a7:92",name="Kitchen"} 87`,
		`hub_rssi{mac="a4:c1:38:e8:a7:92",name="Kitchen"} -67`,
		`hub_msg_cnt{mac="a4:c1:38:e8:a7:92",name="Kitchen"} 42`,
		"# TYPE hub_msg_cnt gauge",
	} {
		if !strings.Contains(response.Body.String(), line+"\n") {
			t.Errorf("scrape missing %q", line)
		}
	}
	missing := httptest.NewRecorder()
	handler.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/other", nil))
	if missing.Code != http.StatusNotFound {
		t.Errorf("unexpected route status = %d", missing.Code)
	}
}

func TestMetricsRegistriesAreIndependent(t *testing.T) {
	first, second := prometheus.NewRegistry(), prometheus.NewRegistry()
	newMetrics(first).update(sampleMeasurement(), "Kitchen", -67)
	newMetrics(second)
	response := httptest.NewRecorder()
	newMetricsHandler(second).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if response.Code != http.StatusOK || strings.Contains(response.Body.String(), "hub_") {
		t.Fatal("empty registry leaked another exporter's sensor metrics")
	}
}

func TestMetricsConcurrentUpdatesAndScrapes(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := newMetrics(reg)
	handler := newMetricsHandler(reg)
	var workers sync.WaitGroup
	workers.Add(1)
	go func() {
		defer workers.Done()
		reading := sampleMeasurement()
		for i := 0; i < 100; i++ {
			reading.Sequence = uint8(i)
			m.update(reading, "Kitchen", -67)
		}
	}()
	for i := 0; i < 100; i++ {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/metrics", nil))
		if response.Code != http.StatusOK {
			t.Errorf("concurrent scrape status = %d", response.Code)
		}
	}
	workers.Wait()
	want := sampleMetricValues()
	want["hub_msg_cnt"] = 99
	assertMetrics(t, reg, sampleMeasurement().MAC, "Kitchen", want, 1)
}
