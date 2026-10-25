package main

import (
	"bytes"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestExporterAcceptsConfiguredSensor(t *testing.T) {
	reg := prometheus.NewRegistry()
	var output bytes.Buffer
	exporter := newExporter([]string{testMAC}, newMetrics(reg), &output)
	exporter.handleAdvertisement(nil, advertisement(t, testMAC, "Kitchen", map[string][]byte{
		environmentServiceUUID: samplePayload(), "unrelated-service": {1},
	}))
	assertMetrics(t, reg, sampleMeasurement().MAC, "Kitchen", sampleMetricValues(), 1)
	want := "MAC address: a4:c1:38:e8:a7:92\nTemperature: 23.45°C\nHumidity: 56.78%\nBattery voltage: 3000 mV\nBattery level: 87%\nMeasurement count: 42\nRSSI: -67\nFlags: 10100101\n\n"
	if output.String() != want {
		t.Fatalf("console output:\n%s\nwant:\n%s", output.String(), want)
	}
}

func TestExporterIgnoresUnrelatedAdvertisements(t *testing.T) {
	tests := []struct {
		name string
		macs []string
		data map[string][]byte
	}{
		{"unconfigured sensor", []string{"A4:C1:38:00:00:00"}, map[string][]byte{environmentServiceUUID: {1}}},
		{"empty whitelist", nil, map[string][]byte{environmentServiceUUID: samplePayload()}},
		{"substring is not an address match", []string{testMAC + ":FF"}, map[string][]byte{environmentServiceUUID: samplePayload()}},
		{"no service data", []string{testMAC}, nil},
		{"different service", []string{testMAC}, map[string][]byte{"unrelated-service": {1}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := prometheus.NewRegistry()
			var output bytes.Buffer
			exporter := newExporter(tt.macs, newMetrics(reg), &output)
			exporter.handleAdvertisement(nil, advertisement(t, testMAC, "Kitchen", tt.data))
			families, err := reg.Gather()
			if err != nil {
				t.Fatal(err)
			}
			if len(families) != 0 || output.Len() != 0 {
				t.Fatal("unrelated advertisement produced metrics or console output")
			}
		})
	}
}

func TestWriteMeasurementNegativeTemperature(t *testing.T) {
	reading := sampleMeasurement()
	reading.Temperature = -1
	reading.Humidity = 1
	reading.Flags = 1
	var output bytes.Buffer
	writeMeasurement(&output, reading, -100)
	for _, want := range []string{"Temperature: -0.01°C\n", "Humidity: 0.01%\n", "RSSI: -100\n", "Flags: 00000001\n"} {
		if !bytes.Contains(output.Bytes(), []byte(want)) {
			t.Errorf("console output missing %q", want)
		}
	}
}
