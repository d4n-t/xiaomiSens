package main

import (
	"fmt"
	"io"

	"tinygo.org/x/bluetooth"
)

type exporter struct {
	allowedMACs map[string]struct{}
	metrics     *metrics
	output      io.Writer
}

func newExporter(macs []string, metrics *metrics, output io.Writer) *exporter {
	allowed := make(map[string]struct{}, len(macs))
	for _, mac := range macs {
		allowed[mac] = struct{}{}
	}
	return &exporter{allowedMACs: allowed, metrics: metrics, output: output}
}

func (e *exporter) handleAdvertisement(_ *bluetooth.Adapter, device bluetooth.ScanResult) {
	if _, allowed := e.allowedMACs[device.Address.String()]; !allowed {
		return
	}
	data, found := device.ServiceData()[environmentServiceUUID]
	if !found {
		return
	}
	reading := decodePayload(data)
	writeMeasurement(e.output, reading, device.RSSI)
	e.metrics.update(reading, device.LocalName(), device.RSSI)
}

func writeMeasurement(output io.Writer, reading measurement, rssi int16) {
	fmt.Fprintf(output, "MAC address: %s\nTemperature: %.2f°C\nHumidity: %.2f%%\nBattery voltage: %d mV\nBattery level: %d%%\nMeasurement count: %d\nRSSI: %d\nFlags: %08b\n\n",
		reading.MAC, float64(reading.Temperature)/100, float64(reading.Humidity)/100,
		reading.BatteryMillivolts, reading.BatteryPercentage, reading.Sequence, rssi, reading.Flags)
}
