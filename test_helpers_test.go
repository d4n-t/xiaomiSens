package main

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"tinygo.org/x/bluetooth"
)

const testMAC = "A4:C1:38:E8:A7:92"

// Hand-written firmware bytes provide an independent fixture: 23.45 C,
// 56.78% humidity, 3000 mV, 87% battery, sequence 42, flags 0xa5.
func samplePayload() []byte {
	return []byte{0x92, 0xa7, 0xe8, 0x38, 0xc1, 0xa4, 0x29, 0x09, 0x2e, 0x16, 0xb8, 0x0b, 87, 42, 0xa5}
}

func sampleMeasurement() measurement {
	return measurement{MAC: "a4:c1:38:e8:a7:92", Temperature: 2345, Humidity: 5678,
		BatteryMillivolts: 3000, BatteryPercentage: 87, Sequence: 42, Flags: 0xa5}
}

type testPayload struct {
	name string
	data map[string][]byte
}

func (p testPayload) LocalName() string                   { return p.name }
func (p testPayload) HasServiceUUID(bluetooth.UUID) bool  { return false }
func (p testPayload) Bytes() []byte                       { return nil }
func (p testPayload) ManufacturerData() map[uint16][]byte { return nil }
func (p testPayload) ServiceData() map[string][]byte      { return p.data }

func advertisement(t *testing.T, mac, name string, data map[string][]byte) bluetooth.ScanResult {
	t.Helper()
	address, err := bluetooth.ParseMAC(mac)
	if err != nil {
		t.Fatal(err)
	}
	return bluetooth.ScanResult{
		Address: bluetooth.MACAddress{MAC: address}, RSSI: -67,
		AdvertisementPayload: testPayload{name: name, data: data},
	}
}

func assertMetrics(t *testing.T, reg *prometheus.Registry, mac, name string, want map[string]float64, sensors int) {
	t.Helper()
	families, err := reg.Gather()
	if err != nil {
		t.Fatal(err)
	}
	sensorFamilies := 0
	for _, family := range families {
		// The HTTP handler also registers its own error counter. Only hub_ is
		// the exporter sensor contract checked by this assertion.
		if !strings.HasPrefix(family.GetName(), "hub_") {
			continue
		}
		sensorFamilies++
		expected, ok := want[family.GetName()]
		if !ok {
			t.Errorf("unexpected metric %s", family.GetName())
			continue
		}
		if family.GetType().String() != "GAUGE" {
			t.Errorf("%s must remain a gauge", family.GetName())
		}
		if len(family.Metric) != sensors {
			t.Errorf("%s has %d series, want %d", family.GetName(), len(family.Metric), sensors)
		}
		found := false
		for _, metric := range family.Metric {
			labels := make(map[string]string)
			for _, label := range metric.Label {
				labels[label.GetName()] = label.GetValue()
			}
			if labels["mac"] == mac && labels["name"] == name {
				found = true
				if len(labels) != 2 {
					t.Errorf("unexpected labels: %v", labels)
				}
				if got := metric.GetGauge().GetValue(); got != expected {
					t.Errorf("%s = %v, want %v", family.GetName(), got, expected)
				}
			}
		}
		if !found {
			t.Errorf("%s missing series for mac=%q name=%q", family.GetName(), mac, name)
		}
	}
	if sensorFamilies != len(want) {
		t.Errorf("got %d sensor metric families, want %d", sensorFamilies, len(want))
	}
}

func sampleMetricValues() map[string]float64 {
	return map[string]float64{
		"hub_temperature_celsius": 2345, "hub_humidity": 5678,
		"hub_battery_mv": 3000, "hub_battery_percentage": 87,
		"hub_rssi": -67, "hub_msg_cnt": 42,
	}
}
