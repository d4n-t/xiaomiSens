package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type metrics struct {
	temperature       *prometheus.GaugeVec
	humidity          *prometheus.GaugeVec
	batteryMillivolts *prometheus.GaugeVec
	batteryPercentage *prometheus.GaugeVec
	rssi              *prometheus.GaugeVec
	sequence          *prometheus.GaugeVec
}

func newMetrics(reg prometheus.Registerer) *metrics {
	newGauge := func(name, help string) *prometheus.GaugeVec {
		return prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "hub",
			Name:      name,
			Help:      help,
		}, []string{"mac", "name"})
	}
	m := &metrics{
		temperature:       newGauge("temperature_celsius", "Raw temperature in hundredths of a degree Celsius; divide by 100 for display."),
		humidity:          newGauge("humidity", "Raw relative humidity in hundredths of a percent; divide by 100 for display."),
		batteryMillivolts: newGauge("battery_mv", "Battery voltage in millivolts."),
		batteryPercentage: newGauge("battery_percentage", "Battery level in percent."),
		rssi:              newGauge("rssi", "Received signal strength reported by the Bluetooth adapter."),
		sequence:          newGauge("msg_cnt", "Sensor measurement sequence number (0-255), wrapping; not a cumulative counter."),
	}
	reg.MustRegister(m.temperature, m.humidity, m.batteryMillivolts, m.batteryPercentage, m.rssi, m.sequence)
	return m
}

func (m *metrics) update(reading measurement, name string, rssi int16) {
	labels := prometheus.Labels{"mac": reading.MAC, "name": name}
	m.temperature.With(labels).Set(float64(reading.Temperature))
	m.humidity.With(labels).Set(float64(reading.Humidity))
	m.batteryMillivolts.With(labels).Set(float64(reading.BatteryMillivolts))
	m.batteryPercentage.With(labels).Set(float64(reading.BatteryPercentage))
	m.rssi.With(labels).Set(float64(rssi))
	m.sequence.With(labels).Set(float64(reading.Sequence))
}

// A dedicated mux and registry keep the exporter independent of global HTTP
// routes and collectors, including when several instances run in tests.
func newMetricsHandler(reg *prometheus.Registry) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
	return mux
}
