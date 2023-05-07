package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"tinygo.org/x/bluetooth"
)

const environmentServiceUUID = "0000181a-0000-1000-8000-00805f9b34fb"

var (
	adapter      = bluetooth.DefaultAdapter
	macWhitelist = []string{"A4:C1:38:E8:A7:92"}
)

type metrics struct {
	temperature *prometheus.GaugeVec
	humidity    *prometheus.GaugeVec
	bat_mv      *prometheus.GaugeVec
	bat_p       *prometheus.GaugeVec
	rssi        *prometheus.GaugeVec
	cnt         *prometheus.GaugeVec
}

func NewMetrics(reg prometheus.Registerer) *metrics {
	m := &metrics{
		temperature: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "hub",
			Name:      "temperature_celsius",
			Help:      "Current temperature without decimal point and two decimals.",
		},
			[]string{"mac", "name"}),
		humidity: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "hub",
			Name:      "humidity",
			Help:      "humidity",
		},
			[]string{"mac", "name"}),
		bat_mv: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "hub",
			Name:      "battery_mv",
			Help:      "battery in mV",
		},
			[]string{"mac", "name"}),
		bat_p: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "hub",
			Name:      "battery_percentage",
			Help:      "battery in 0-100%",
		},
			[]string{"mac", "name"}),
		rssi: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "hub",
			Name:      "rssi",
			Help:      "rssi",
		},
			[]string{"mac", "name"}),
		cnt: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "hub",
			Name:      "msg_cnt",
			Help:      "Current msg nb",
		},
			[]string{"mac", "name"}),
	}
	reg.MustRegister(m.temperature)
	reg.MustRegister(m.humidity)
	reg.MustRegister(m.bat_mv)
	reg.MustRegister(m.bat_p)
	reg.MustRegister(m.rssi)
	reg.MustRegister(m.cnt)
	return m
}

func main() {
	// Create a non-global registry.
	reg := prometheus.NewRegistry()

	// Create new metrics and register them using the custom registry.
	m := NewMetrics(reg)

	go func() {
		// Expose metrics and custom registry via an HTTP server
		// using the HandleFor function. "/metrics" is the usual endpoint for that.
		http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
		log.Fatal(http.ListenAndServe(":8081", nil))
	}()

	fmt.Printf("Finished setting up server!!!!\n")

	// Enable BLE interface.
	must("enable BLE stack", adapter.Enable())

	// Start scanning.
	println("scanning...")
	err := adapter.Scan(func(adapter *bluetooth.Adapter, device bluetooth.ScanResult) {
		if strings.Contains(strings.Join(macWhitelist, ","), device.Address.String()) {
			scanAdvPay := device.AdvertisementPayload
			for k, v := range scanAdvPay.ServiceData() {
				if k == environmentServiceUUID {
					// Parse MAC address
					mac := make([]byte, 6)
					copy(mac, v[0:6])
					for i := 0; i < len(mac)/2; i++ {
						j := len(mac) - i - 1
						mac[i], mac[j] = mac[j], mac[i]
					}
					hwMac := net.HardwareAddr(mac)

					// Parse temperature
					var temperature int16
					binary.Read(bytes.NewReader(v[6:8]), binary.LittleEndian, &temperature)

					// Parse humidity
					var humidity uint16
					binary.Read(bytes.NewReader(v[8:10]), binary.LittleEndian, &humidity)

					// Parse battery voltage
					var battery_mv uint16
					binary.Read(bytes.NewReader(v[10:12]), binary.LittleEndian, &battery_mv)

					// Parse battery level
					battery_level := v[12]

					// Parse measurement count
					counter := v[13]

					// Parse flags
					flags := v[14]

					// Print the parsed data
					fmt.Printf("MAC address: %s\n", hwMac.String())
					fmt.Printf("Temperature: %.2f°C\n", float64(temperature)/100)
					fmt.Printf("Humidity: %.2f%%\n", float64(humidity)/100)
					fmt.Printf("Battery voltage: %d mV\n", battery_mv)
					fmt.Printf("Battery level: %d%%\n", battery_level)
					fmt.Printf("Measurement count: %d\n", counter)
					fmt.Printf("RSSI: %d\n", device.RSSI)
					fmt.Printf("Flags: %08b\n\n", flags)

					m.temperature.With(prometheus.Labels{"mac": hwMac.String(), "name": device.LocalName()}).Set(float64(temperature))
					m.humidity.With(prometheus.Labels{"mac": hwMac.String(), "name": device.LocalName()}).Set(float64(humidity))
					m.bat_mv.With(prometheus.Labels{"mac": hwMac.String(), "name": device.LocalName()}).Set(float64(battery_mv))
					m.bat_p.With(prometheus.Labels{"mac": hwMac.String(), "name": device.LocalName()}).Set(float64(battery_level))
					m.rssi.With(prometheus.Labels{"mac": hwMac.String(), "name": device.LocalName()}).Set(float64(device.RSSI))
					m.cnt.With(prometheus.Labels{"mac": hwMac.String(), "name": device.LocalName()}).Set(float64(counter))
				}
			}
		}
	})
	must("start scan", err)
}

func must(action string, err error) {
	if err != nil {
		panic("failed to " + action + ": " + err.Error())
	}
}
