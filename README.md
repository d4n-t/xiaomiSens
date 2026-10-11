# xiaomiSens

A Go application that scans Bluetooth Low Energy (BLE) advertisements from
reprogrammed Xiaomi BLE temp2 sensors and exposes their readings for Prometheus.

The intended deployment is a small computer with Bluetooth hardware, such as a
Raspberry Pi with a build for its architecture. For now, the application runs on
an Ubuntu laptop.

## How it works

The sensors publish their measurements in BLE advertisements. The application
uses the host's default Bluetooth adapter to scan for six configured sensor MAC
addresses, then decodes their advertisement service data under UUID
`0000181a-0000-1000-8000-00805f9b34fb`. It receives readings without pairing or
connecting to each sensor.

Readings are printed to the console and exposed at `/metrics` on port `8081`.
Prometheus scrapes this endpoint to collect measurements; Grafana can visualize
the data stored in Prometheus.

The current decoder expects a 15-byte service-data layout containing the sensor
MAC address, temperature, humidity, battery voltage, battery percentage,
measurement sequence number, and flags. Flags are printed to the console.

## Build and run on Ubuntu

Requirements:

- Go compatible with the module's `go 1.20` declaration.
- A BLE-capable Bluetooth adapter.
- BlueZ running, with Bluetooth enabled and permission to use its system D-Bus
  interface.
- Sensors running the reprogrammed firmware with the expected advertisement
  format.

Dependencies are included in `vendor/`. From the project directory:

```bash
go build -mod=vendor -o xiaomiSens .
./xiaomiSens
```

Check the exported metrics:

```bash
curl http://localhost:8081/metrics
```

The HTTP server listens on all network interfaces. A Prometheus instance on
another machine can scrape `<exporter-host>:8081`. Sensor metric series appear
after matching advertisements are received.

For a Raspberry Pi, build for the Linux architecture matching its operating
system. For example, a 64-bit Raspberry Pi OS build can be cross-compiled with:

```bash
GOOS=linux GOARCH=arm64 go build -mod=vendor -o xiaomiSens-linux-arm64 .
```

The target computer needs its own Bluetooth hardware and BlueZ setup. Raspberry
Pi deployment remains an intended target rather than a verified deployment.

## Metrics and scaling

All sensor metrics are gauges labelled with `mac` and `name`.

| Metric | Exported value |
| --- | --- |
| `hub_temperature_celsius` | Raw temperature in hundredths of a degree Celsius: `2345` represents `23.45°C`. |
| `hub_humidity` | Raw relative humidity in hundredths of a percent: `5678` represents `56.78%`. |
| `hub_battery_mv` | Battery voltage in millivolts. |
| `hub_battery_percentage` | Battery level from 0 to 100 percent. |
| `hub_rssi` | Received signal strength reported by the Bluetooth adapter. |
| `hub_msg_cnt` | Sensor's 8-bit measurement sequence number; it wraps and is not a cumulative received-message count. |

Exporting temperature and humidity as raw integers is intentional: it retains
the sensor's full reported precision, with scaling handled later in Prometheus
or Grafana. Console output already displays these readings divided by 100.
Despite its name, `hub_temperature_celsius` therefore needs scaling before it
is displayed as degrees Celsius.

Example PromQL expressions for display:

```promql
hub_temperature_celsius / 100
hub_humidity / 100
```

## Prometheus setup

Add a scrape job to your Prometheus configuration. This example assumes
Prometheus runs on the same computer as the exporter:

```yaml
scrape_configs:
  - job_name: xiaomi_sensors
    metrics_path: /metrics
    static_configs:
      - targets: ['localhost:8081']
```

Use the exporter's hostname or IP address instead of `localhost` when Prometheus
runs elsewhere, including in a separate container.

## Current configuration and future work

Sensor MAC addresses are currently hardcoded in `macWhitelist` in `main.go`;
the HTTP port is also hardcoded. Edit these values and rebuild when necessary.

The following improvements are deferred:

- Update sensor firmware and the project's libraries, then improve payload
  handling and validation for the resulting advertisement format. The current
  decoder assumes the expected layout and does not check payload length.
- Add freshness metrics to identify sensors that have stopped transmitting.
  Currently, the last readings remain exposed while the exporter is running.
- Add recovery from Bluetooth errors and graceful shutdown.
- Move sensor addresses and server settings into external configuration.

The application currently uses `tinygo.org/x/bluetooth` for Bluetooth access and
`github.com/prometheus/client_golang` for metrics. There are no automated tests
yet.
