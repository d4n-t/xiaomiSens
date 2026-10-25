# xiaomiSens

A Go application that scans Bluetooth Low Energy (BLE) advertisements from
reprogrammed Xiaomi BLE temp2 sensors and exposes their readings for Prometheus.

The application runs on a Raspberry Pi 4 and is developed and tested on an Ubuntu
laptop. The refactored application was deployed and verified on the Pi on
2026-10-25. See the [deployment runbook](docs/deployment.md) for the tested build
settings, monitoring configuration, verification steps, and rollback procedure.

All six sensors currently run custom firmware version 5.9 from
[pvvx/ATC_MiThermometer](https://github.com/pvvx/ATC_MiThermometer), configured for
unencrypted **Custom/PVVX** advertisements. Live readings and metrics were verified
after the upgrade. BTHome and ATC advertisement formats are not supported by this
decoder.

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
system. The existing Raspberry Pi 4 deployment uses **32-bit armhf userspace**
with a 64-bit kernel. The verified production build was compiled natively on the
Pi with Go 1.27.3, `GOARM=6`, and `CGO_ENABLED=1`, preserving the previous build's
settings. Follow the runbook to build in staging and replace the service safely.

A static ARMv7 cross-build is also available for that Pi:

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -mod=vendor -o xiaomiSens-linux-arm .
```

For a 64-bit Raspberry Pi OS userspace, use:

```bash
GOOS=linux GOARCH=arm64 go build -mod=vendor -o xiaomiSens-linux-arm64 .
```

The target computer needs its own Bluetooth hardware and BlueZ setup. Check the
userspace architecture rather than relying only on the kernel's `uname -m`
output. The refactored application cross-compiles for the Pi; its live deployment
was verified with all six sensors and the existing Prometheus/Grafana setup.

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

Sensor MAC addresses are currently hardcoded in `sensorMACs` in `config.go`;
the HTTP port is also defined there. Edit these values and rebuild when necessary.

The following improvements are deferred:

- Update the project's Bluetooth libraries and improve payload handling and
  validation. Sensor firmware has already been updated to 5.9. The current
  decoder assumes the expected layout and does not check payload length.
- Add freshness metrics to identify sensors that have stopped transmitting.
  Currently, the last readings remain exposed while the exporter is running.
- Add recovery from Bluetooth errors and graceful shutdown.
- Move sensor addresses and server settings into external configuration.

The application currently uses `tinygo.org/x/bluetooth` for Bluetooth access and
`github.com/prometheus/client_golang` for metrics.

## Code structure

The application remains a single Go package, split by responsibility so changes
can be made without navigating one large scan callback:

| File | Responsibility |
| --- | --- |
| `main.go` | Wire the real Bluetooth adapter, registry, output, and HTTP server together. |
| `config.go` | Default sensor addresses, service UUID, and listen address. |
| `payload.go` | Decode firmware bytes into a measurement, preserving raw units. |
| `exporter.go` | Filter advertisements, print readings, and update metrics. |
| `metrics.go` | Define Prometheus gauges and the `/metrics` HTTP handler. |
| `app.go` | Enable Bluetooth and run scanning alongside HTTP serving; report failures. |
| `*_test.go` | Unit tests and an advertisement-to-HTTP integration test. |

Use `go run -mod=vendor .` to run the whole package during development. Running
only `main.go` does not include the other application files.

## Tests

Run all tests, including the race detector, and static checks:

```bash
go test -mod=vendor -race -cover ./...
go vet -mod=vendor ./...
```

The tests use a fake Bluetooth adapter and in-memory HTTP requests. They require
no Bluetooth hardware, sensor access, network listener, or running Prometheus.
The race detector requires a supported host toolchain with CGO and a C compiler.

Coverage includes payload byte order, signed temperatures and field boundaries,
raw metric scaling, label isolation, repeated readings and sequence wraparound,
advertisement filtering, console formatting, HTTP scrapes, concurrent metric
updates, and adapter/scanner/server errors. An integration test exercises a
sensor advertisement through the application to a Prometheus scrape.

When changing the decoder, add a hand-written byte fixture and expected reading
in `payload_test.go`. When changing metrics, update the expected scrape output
in `metrics_test.go`. Shared fixtures and metric assertions live in
`test_helpers_test.go`.

The small `main` wiring function, the Bluetooth library, and real radio behavior
are not exercised by the automated tests. Verify actual sensor reception on the
target computer separately. Payload validation, freshness, retries, and graceful
shutdown remain deferred as described above.
