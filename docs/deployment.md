# Raspberry Pi deployment runbook

This records the procedure that worked on 2026-10-25. Recheck the target before
using it again; versions and active paths can change. Connection details and
private backup locations are in `.local/deployment-state.md` on the development
machine, when available. No credentials or recovery database dumps belong here.

## Last verified deployment

| Item | Verified value |
| --- | --- |
| Application revision | `fb61017e6157224c52a21dfe115955a616e36a87` |
| Target | Raspberry Pi 4, approximately 4 GB RAM |
| Userspace | Raspbian 11 Bullseye, **32-bit armhf** |
| Kernel | Linux 6.1.21-v8+, aarch64 |
| Bluetooth stack | BlueZ 5.55 |
| Installed Go | **1.27.3 linux/arm**, upgraded from 1.20.4 |
| Native build | `GOOS=linux GOARCH=arm GOARM=6 CGO_ENABLED=1` |
| Application dependencies | TinyGo Bluetooth 0.6.0; existing 2022 go-bluetooth revision |
| Sensors | Six LYWSD03MMC sensors; reported firmware 5.9; unencrypted Custom/PVVX |

The 64-bit kernel does not imply 64-bit userspace. Check `getconf LONG_BIT`,
`dpkg --print-architecture`, and the existing executable's `file`/`go version -m`
output. ARM64 and static ARMv7 cross-builds succeeded on the laptop, but the
production deployment deliberately matched the old native ARMv6/CGO build.

The Bluetooth library upgrade discussed during investigation is still future
work. Updating Go alone does not replace the old mapping library.

## Monitoring contract

The paths and service names below are placeholders. Resolve them from the
private operational note or inspect the target; do not treat them as literal
commands or assume another installation uses the same layout.

| Component | Service | Executable/configuration/data |
| --- | --- | --- |
| Exporter | `<exporter-unit>` | `<app-dir>/mitempmonitor`; port 8081 |
| Prometheus 2.43.1 | `<prometheus-unit>` | `<prometheus-dir>/prometheus`; configuration and data directories recorded privately |
| Grafana Enterprise RPi 9.5.1 | `grafana-server.service` | `/usr/share/grafana`; `/etc/grafana/grafana.ini`; SQLite `/var/lib/grafana/grafana.db` |

Preserve the exporter and Prometheus service users and groups recorded in the
private operational note; Grafana uses its package-provided service account.
Exporter/Prometheus units require and start after the network-online and Bluetooth
targets, use `Restart=always`, and are enabled under `multi-user.target`.
The exporter unit has **`ExecStartPre=/bin/sleep 40`**. It was preserved.

Prometheus scrapes the exporter at `localhost:8081` and itself at `localhost:9090`,
with 30-second scrape/evaluation intervals and explicit two-year retention.
Do not confuse its active storage directory with the older `data/` directory
under the Prometheus download directory.

The Grafana dashboard divides raw temperature and humidity by 100.
Its saved temperature/humidity visibility override plots only one selected
sensor; hidden curves are not evidence that other sensors stopped reporting.
The dashboard and data source are stored in SQLite, not active provisioning
files. Preserve their UIDs when restoring from exported definitions.

## Procedure that worked

1. **Inventory and record the baseline.** Check service state, restart counts,
   process IDs, configuration hashes, executable hash/build information, Git
   status/revision, architecture, available memory/disk, Go version, and Bluetooth
   state. Read live metrics, Prometheus targets, and Grafana health. Confirm that
   all six sensor sequences advance. Record any existing dirty source changes.
2. **Back up before modifying active files.** Save the working executable and
   complete source checkout, unit/configuration files, baseline metrics, and old
   Go installation. Keep a local recovery copy too. Configuration and Grafana
   SQL snapshots do not include Prometheus historical measurement data.
3. **Stage Go separately.** Read official Go release metadata, select the
   `linux-armv6l` archive, verify SHA-256, extract into a fresh staging directory,
   and execute that Go binary on the Pi. Never unpack a new Go archive over the
   existing Go directory. Keep the old installation until validation completes.
4. **Fetch the release into a fresh checkout.** Clone GitHub `main`, record its
   full commit ID, and confirm it is the reviewed release. Do not pull/build over
   a running executable. Dependencies were built using the checked-in vendor
   directory; no dependency upgrade was included in the rollout.
5. **Test and compile natively as the service user.** The successful commands used the staged
   Go executable, `GOTOOLCHAIN=local`, `GOMAXPROCS=2`, `GOARM=6`, `CGO_ENABLED=1`,
   and an isolated build cache. Compilation was run at nice level 10 with two
   package workers while the monitoring services continued running.

   ```bash
   # Run in the fresh staging checkout. Use the verified staged Go in PATH.
   GOTOOLCHAIN=local GOMAXPROCS=2 GOARM=6 CGO_ENABLED=1 \
     go test -p=2 -mod=vendor -cover ./...
   GOTOOLCHAIN=local GOMAXPROCS=2 GOARM=6 CGO_ENABLED=1 \
     go vet -p=2 -mod=vendor ./...
   GOTOOLCHAIN=local GOMAXPROCS=2 GOARM=6 CGO_ENABLED=1 \
     go build -p=2 -mod=vendor -o ../mitempmonitor.candidate .
   ```

   Tests passed at 88.5% coverage, vet passed, and `file`, `ldd`, and
   `go version -m` confirmed the expected 32-bit executable, resolved libraries,
   build flags, commit, and clean tracked source. Race tests passed separately
   on the Ubuntu laptop; `-race` is unsupported for 32-bit Linux ARM.
6. **Compare live behavior before replacement.** Run the exact candidate as the service user
   in a temporary network namespace with loopback enabled. This isolates port
   8081 while keeping the real host D-Bus/Bluetooth adapter available. It does
   not isolate the radio, so monitor the production service as well. The saved
   deployment's `shadow_check.py` is a reference, not a reusable script with
   universally valid paths.

   Require the same 36 sensor metric series (six metrics per sensor), matching
   names and labels, raw sensor values matching at equal sequence numbers, and
   a new sequence number from every sensor. Exclude RSSI from exact equality:
   signal strength can change between receptions. Verify zero HTTP metric error
   counters. Stop the temporary candidate and confirm the original service
   processes stayed unchanged. This check passed after about 43 seconds.
7. **Prepare rollback, then switch.** Install Go into a separate versioned
   directory, preserve the old directory, and point `/usr/local/go` at the new
   installation. Verify it works as the service user. Stage the tested executable
   in the new checkout with the existing service ownership and executable
   permissions. Stop only the exporter unit; preserve the old application
   directory and move the staged
   checkout into the original application path. Start the unchanged unit.
   No systemd daemon reload is needed if no unit file changed.
8. **Wait for real recovery.** Preserve the 40-second startup delay. First sensor
   series appeared at about 45 seconds; all 36 appeared by about 61 seconds.
   The rollout passed all fresh-reading checks by about 96 seconds. A temporary
   scrape failure during the restart is expected; record the collection gap.
   The successful verifier allowed up to 180 seconds and would restore the old
   application and Go if health checks failed.
9. **Verify the complete system.** Require every sensor sequence to advance,
   exact metric names/labels/raw units, exporter and Prometheus self-scrape targets UP
   after the switch, Grafana database OK, zero unexpected exporter restarts, and
   the actual running executable's hash matching the candidate. Confirm that
   Prometheus/Grafana process IDs and unit/configuration hashes stayed unchanged.
   Ensure no temporary exporter is left running. Save the result and update this
   record and local operational notes.

## Rollback and recovery

The completed rollout retained the previous entire application directory, the
previous Go installation, original installers, source/vendor archive, old/new
binaries, test logs, configuration hashes, health samples, and a rollback script.
The exact locations are in the ignored local deployment note and private recovery
report. The existing rollback script is specific to that deployment; do not
blindly rerun it after later deployments.

On verification failure, stop the new exporter, restore the previous application
directory and Go installation, restart the same unit, allow its startup delay,
and verify all six sensors and Prometheus scraping again. Preserve the failed
candidate for inspection. Prometheus and Grafana should keep running during an
exporter rollback.

## Findings to carry forward

- Native Go 1.27.3 works on the existing Raspbian/BlueZ installation; an OS or
  userspace architecture change was not needed for this release.
- The existing sensor firmware 5.9 configuration works with this decoder.
- Raw temperature/humidity scaling is intentional and part of the contract.
- The mapping warnings originated from old Go BlueZ bindings and existed before
  the sensor firmware upgrade. This deployment did not change those bindings.
- A 32-bit native build cannot run Go's race detector; keep that check on the
  development laptop.
- Prometheus API timestamps can contain nanoseconds. The Pi's Python 3.9
  deployment verifier normalized fractional seconds before parsing timestamps.
- These finite checks establish a working baseline, not a guarantee against all
  future radio failures or malformed payloads. Previously deferred features
  remain separate work.
