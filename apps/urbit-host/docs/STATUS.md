# Implementation status — 0.1.0-pilot

Created September 8, 2026. This is source and packaged Linux executables, not an
installed Omarchy plugin. See TEST-RESULTS.txt for the actual automated run and CLI-SMOKE-RESULTS.txt
for a compiled-binary installer/daemon/restart smoke test in a temporary HOME.

## Implemented in code

- JSON-first CLI with local host setup and remote SSH host aliases.
- Persistent per-user host service; owner and fixed-role agent Unix sockets.
- Atomic durable metadata, request deduplication, asynchronous lifecycle jobs,
  per-host lifecycle serialization, interrupted-job reporting, single writer.
- Docker creation, graceful stop, restart, bounded logs, inventory, archive and
  separately confirmed purge. Container ownership labels checked on use.
- Explicit fake/keyed/comet paths; real creation gated off in this pilot.
- Independent same-host fake-galaxy groups with shared loopback per group.
- Per-instance bind mounts, immutable image resolution and pill overrides.
- Private client browser forwarding, local clipboard copy and tmux Dojo attach.
- HTTP-based readiness, on-request CPU/memory and shared-budget warnings.
- Hourly runtime digest checks and persistent, explicitly dismissible records.
- One-time comet identity discovery and boot-key cleanup logic.
- Agent/operator skills, service unit, installer, preflight and pilot script.

## Automated validation actually performed

Go unit and integration tests, including real local Unix sockets/TCP byte streams,
exercise policy, persistence, request deduplication, archive/purge, independent
fake identities, unavailable Docker status, fixed-role RPC, strict JSON parsing,
CLI input validation, shell-free SSH command construction and loopback proxies.
Lifecycle tests use a **fake Docker executor**, not a real Docker daemon.
All 25 included tests pass with the race detector. Vet, compiler and shell
syntax checks pass. A separate compiled-amd64 CLI smoke test verified client
installation, daemon startup, fixed-role RPC, the real-ship pilot gate, and
development-group persistence across a manager restart without Docker installed.

## Not yet validated with real external systems

- GroundSeg manifest retrieval and selected Native Planet image compatibility.
- Real Docker lifecycle, UID/GID permissions and container restart behavior.
- Actual Vere fake/keyed/comet boot, pill override, Hoon loopback response shape,
  login-code retrieval, and tmux attachment.
- SSH/Tailscale transport, forwarding reconnect behavior and Wayland clipboard.
- Workstation reboot ordering for a fake group's namespace keeper/member ships.
- Same-group fake-ship traffic and isolation with the chosen Ames/Mesa runtime.
- Real-ship key cleanup, identity discovery and graceful termination semantics.
- ARM64 execution (cross-compiled only).

Do not interpret an automated test pass as a successful Urbit boot.

## Deliberately unfinished

- Cross-tailnet fake-network relay and distributed environment enrollment.
- Quickshell/QML fleet widget, pane, colored icons and desktop notifications.
- Deep/kernel metrics and urbtop integration.
- Runtime image replacement/upgrade, channel reassignment UI and rollback.
- Real-identity cross-host fencing, migration and adoption of existing piers.
- Group deletion, automatic namespace repair and creation cancellation.
- Developer source mounts, desks and reset-to-baseline conveniences.
- Anchor fork/deployment, StarTram integration and hosted-Eyre-only connections.
- All backups, application or pier level (out of scope by user decision).

The image-change notice store exists; **persistent Omarchy notification delivery
is not implemented**. Metric limits are warnings; no resource cgroups are applied.
An absent or unreachable host is never a confirmed stopped ship.
