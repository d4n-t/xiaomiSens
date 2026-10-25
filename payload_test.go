package main

import "testing"

func TestDecodePayload(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want measurement
	}{
		{"typical reading", samplePayload(), sampleMeasurement()},
		{"zero reading", make([]byte, 15), measurement{MAC: "00:00:00:00:00:00"}},
		{"negative temperature", []byte{0x92, 0xa7, 0xe8, 0x38, 0xc1, 0xa4, 0xcb, 0xfb, 0x2e, 0x16, 0xb8, 0x0b, 87, 42, 0xa5},
			measurement{MAC: "a4:c1:38:e8:a7:92", Temperature: -1077, Humidity: 5678, BatteryMillivolts: 3000, BatteryPercentage: 87, Sequence: 42, Flags: 0xa5}},
		{"maximum fields", []byte{255, 255, 255, 255, 255, 255, 255, 127, 255, 255, 255, 255, 255, 255, 255},
			measurement{MAC: "ff:ff:ff:ff:ff:ff", Temperature: 32767, Humidity: 65535, BatteryMillivolts: 65535, BatteryPercentage: 255, Sequence: 255, Flags: 255}},
		{"minimum temperature", []byte{0, 0, 0, 0, 0, 0, 0, 128, 0, 0, 0, 0, 0, 0, 0},
			measurement{MAC: "00:00:00:00:00:00", Temperature: -32768}},
		{"trailing bytes", append(samplePayload(), 0xde, 0xad), sampleMeasurement()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := append([]byte(nil), tt.data...)
			if got := decodePayload(tt.data); got != tt.want {
				t.Fatalf("decoded %+v, want %+v", got, tt.want)
			}
			for i := range before {
				if tt.data[i] != before[i] {
					t.Fatal("decoder changed the advertisement bytes")
				}
			}
		})
	}
}
