package main

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"tinygo.org/x/bluetooth"
)

type fakeScanner struct {
	enable func() error
	scan   func(func(*bluetooth.Adapter, bluetooth.ScanResult)) error
}

func (s fakeScanner) Enable() error { return s.enable() }
func (s fakeScanner) Scan(callback func(*bluetooth.Adapter, bluetooth.ScanResult)) error {
	return s.scan(callback)
}

func awaitRun(t *testing.T, adapter scanner, exporter *exporter, handler http.Handler, serve func(http.Handler) error) error {
	t.Helper()
	result := make(chan error, 1)
	go func() { result <- run(adapter, exporter, handler, serve) }()
	select {
	case err := <-result:
		return err
	case <-time.After(3 * time.Second):
		t.Fatal("application did not return after an injected failure or completion")
		return nil
	}
}

func TestRunEnableFailure(t *testing.T) {
	want := errors.New("no Bluetooth adapter")
	adapter := fakeScanner{
		enable: func() error { return want },
		scan: func(func(*bluetooth.Adapter, bluetooth.ScanResult)) error {
			t.Error("scanning started despite enable failure")
			return nil
		},
	}
	err := awaitRun(t, adapter, nil, nil, func(http.Handler) error {
		t.Error("HTTP server started despite enable failure")
		return nil
	})
	if !errors.Is(err, want) || !strings.Contains(err.Error(), "enable BLE stack") {
		t.Fatalf("enable error = %v", err)
	}
}

func TestRunReturnsWhenWorkerEnds(t *testing.T) {
	failure := errors.New("injected failure")
	for _, tt := range []struct {
		name   string
		worker string
		err    error
	}{
		{"scan failure", "scan", failure},
		{"HTTP failure", "serve", failure},
		{"scan completion", "scan", nil},
		{"HTTP completion", "serve", nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			stop := make(chan struct{})
			finished := make(chan struct{})
			t.Cleanup(func() {
				close(stop)
				select {
				case <-finished:
				case <-time.After(3 * time.Second):
					t.Error("fake worker failed to finish")
				}
			})
			reg := prometheus.NewRegistry()
			exporter := newExporter(sensorMACs, newMetrics(reg), io.Discard)
			adapter := fakeScanner{
				enable: func() error { return nil },
				scan: func(func(*bluetooth.Adapter, bluetooth.ScanResult)) error {
					if tt.worker == "scan" {
						return tt.err
					}
					defer close(finished)
					<-stop
					return nil
				},
			}
			serve := func(http.Handler) error {
				if tt.worker == "serve" {
					return tt.err
				}
				defer close(finished)
				<-stop
				return nil
			}
			err := awaitRun(t, adapter, exporter, newMetricsHandler(reg), serve)
			if !errors.Is(err, tt.err) {
				t.Fatalf("run error = %v, want %v", err, tt.err)
			}
			if tt.err != nil {
				context := "scan sensors"
				if tt.worker == "serve" {
					context = "serve metrics"
				}
				if !strings.Contains(err.Error(), context) {
					t.Errorf("error missing context %q: %v", context, err)
				}
			}
		})
	}
}

func TestRunAdvertisementToHTTPScrape(t *testing.T) {
	reg := prometheus.NewRegistry()
	var output bytes.Buffer
	exporter := newExporter([]string{testMAC}, newMetrics(reg), &output)
	device := advertisement(t, testMAC, "Kitchen", map[string][]byte{environmentServiceUUID: samplePayload()})
	ready, stop, finished := make(chan struct{}), make(chan struct{}), make(chan struct{})
	t.Cleanup(func() {
		close(stop)
		select {
		case <-finished:
		case <-time.After(3 * time.Second):
			t.Error("fake scan failed to finish")
		}
	})
	adapter := fakeScanner{
		enable: func() error { return nil },
		scan: func(callback func(*bluetooth.Adapter, bluetooth.ScanResult)) error {
			defer close(finished)
			callback(nil, device)
			close(ready)
			<-stop
			return nil
		},
	}
	responses := make(chan *httptest.ResponseRecorder, 1)
	serverStopped := errors.New("test server finished")
	serve := func(handler http.Handler) error {
		<-ready
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/metrics", nil))
		responses <- response
		return serverStopped
	}
	if err := awaitRun(t, adapter, exporter, newMetricsHandler(reg), serve); !errors.Is(err, serverStopped) {
		t.Fatalf("run error = %v", err)
	}
	response := <-responses
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `hub_temperature_celsius{mac="a4:c1:38:e8:a7:92",name="Kitchen"} 2345`) {
		t.Fatalf("unexpected scrape: status %d, body %s", response.Code, response.Body.String())
	}
	assertMetrics(t, reg, sampleMeasurement().MAC, "Kitchen", sampleMetricValues(), 1)
	if !strings.Contains(output.String(), "Temperature: 23.45°C") {
		t.Errorf("unexpected console output: %s", output.String())
	}
}
