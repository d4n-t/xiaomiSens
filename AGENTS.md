# Project guidance

Read `README.md` for the application contract. Before deployment or Raspberry Pi
work, read `docs/deployment.md` and, when present, `.local/deployment-state.md`.
The latter contains local operational context and must remain untracked.

## Behavior to preserve

- Temperature and humidity metrics intentionally retain raw hundredths. Scaling
  happens in Prometheus/Grafana; do not "fix" this scaling in the exporter.
- Preserve the six configured sensors, metric names, `mac`/`name` labels,
  `/metrics` endpoint, and port unless the task explicitly changes them.
- Sensors were verified with pvvx firmware 5.9 and unencrypted Custom/PVVX
  advertisements. BTHome, ATC, and encrypted advertisements are unsupported.
- Payload validation, freshness metrics, retries, graceful shutdown, and external
  configuration are deferred work. Keep unrelated changes out of a deployment.

## Verified deployment baseline

On 2026-10-25, commit `fb61017e6157224c52a21dfe115955a616e36a87` was built natively
on a Raspberry Pi 4 with 32-bit armhf userspace, Raspbian 11, a 64-bit Linux 6.1
kernel, and BlueZ 5.55. Go was upgraded from 1.20.4 to 1.27.3. The working build
used `GOOS=linux GOARCH=arm GOARM=6 CGO_ENABLED=1`.

The installed Bluetooth dependencies remain TinyGo Bluetooth 0.6.0 and the 2022
go-bluetooth revision. A migration to newer Bluetooth libraries was investigated
but has not been implemented or deployed.

## Deployment practice

- Recheck the live machine and GitHub revision; this baseline is historical,
  not evidence that the current remote state is unchanged.
- Use a fresh staging checkout and preserve the old source, binary, toolchain,
  configuration, and a usable rollback path before replacement.
- Build the whole package (`go build .`), never just `main.go`.
- Run race tests on the Ubuntu laptop. On the 32-bit Pi, use native tests/vet
  without `-race`, and limit compile parallelism as described in the runbook.
- The successful rollout included a live comparison of the exact candidate
  against production before switching. Preserve that check for future releases.
- The exporter unit has a deliberate 40-second startup delay. Allow for it in
  health checks. Verify advancing sensor sequences, not just HTTP status.
- For an exporter-only release, preserve Prometheus/Grafana configurations and
  processes. Backups contain sensitive account/configuration data; keep them
  outside the repository.
- Update the deployment record after each completed rollout and report any
  remaining uncertainty. Historical deployment instructions do not expand the
  user's currently authorized task scope.
