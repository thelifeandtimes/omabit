# Omabit

A local Urbit fleet for Omarchy: run ships with the official Vere runtime, monitor
them from the desktop, and manage maintenance without keeping terminal sessions open.

This repository is the ship-running and orchestration project at
[thelifeandtimes/omabit](https://github.com/thelifeandtimes/omabit).
[Tend](https://github.com/thelifeandtimes/tend), the Urbit reminders application,
has its own repository, CLI, and release lifecycle.

**Published:** [**0.1.0-rc.7**](https://github.com/thelifeandtimes/omabit/releases/tag/v0.1.0-rc.7), a Linux/AMD64 development release candidate
using the existing pier-owning Unix user for the runner and its rootless engine.
See [binary bundle installation](docs/release-install.md),
[release notes and qualification scope](docs/release-rc7.md),
and [Recommended Apps pill provenance](docs/pill-release.md). Final 0.1.0 is still pending.

RC.3 makes `|mount` → ordinary copy/editor saves → `|commit` work directly on
user-owned files. Default piers live at **`/srv/urbit/piers/`**; custom locations
use ordinary user permissions. See the [owner-user runtime](docs/owner-user-runtime.md).
RC.2 storage/import/deletion and bounded reboot-resume behavior continue.
RC.4 adds a first-run welcome and laptop/always-on choice, host-checked port
suggestions, confirmed automatic first boot, healthy-only access buttons and
corrected monitor-window activation.
RC.5 adds explicit software/source information, stable inline progress, public comet
names in runtime output and the user-supplied Recommended Apps pill. Application
Connections is marked Coming Soon.
RC.6 corrects the effective pill default when the shell supplies only explicit
widget settings; this is verified in the running sourcebox shell.
RC.7 adds persistent log following, bounded reconnect backoff and reader lifecycle
cleanup to remove per-refresh SSH/PAM/logind sessions.
Prior dedicated-account installations were
[retired for the user-driven fresh-install exercise](docs/diagnostics/2026-10-15-rc2-release-retirement.md).

## Get started

In a terminal inside your Omarchy desktop:

```bash
omarchy plugin install https://github.com/thelifeandtimes/omabit.git --enable
```

This repository is directly installable through Omarchy's native GitHub plugin installer. The plugin downloads
and verifies the pinned prebuilt tools on first use; no Go compiler or local bundle
directory is required. See [native plugin distribution](docs/native-plugin.md).

Then open **~+ → Setup → Set up this computer**. The guided flow checks prerequisites,
prepares private desktop access, asks for OS authorization, installs the background
services and connects your desktop automatically. Choose **Add a ship → New comet**
when it reports ready.

For development from a source checkout:

```bash
bash scripts/install.sh
go build -o ./build/ ./cmd/... ./backend/cmd/...
./build/omabit --format text setup
```

See the [fresh installation guide](docs/install.md) for prerequisites, reruns and
troubleshooting, or [binary bundle installation](docs/release-install.md) for the
published release assets. Source builds need Go 1.25+. The current host implementation
targets Linux x86-64, systemd/cgroup v2 and Docker 29.7.2.

## What works

The generic, non-Omarchy runner and collector entry points are in
[`backend/`](backend/README.md). The desktop-specific Go helper is `cmd/omabit`;
shared implementation remains under `internal/` in one Go module.

- A separate owner-user rootless Docker engine and the pinned official Vere release.
- New comets and development fakes, with durable start/stop/resume operations.
- Owned-key bootstrap, including a live-tested moon, and stopped in-tree pier enrollment.
- Permanent per-ship mount directories: first boot creates `pier/` in place and keeps running.
- Management and restricted SSH services, including verified clean and unclean host-reboot resume.
- A compact fleet menu and monitoring window with container RAM, tasks and health.
- Per-ship lifecycle/error/exit diagnostics and bounded read-only runtime logs.
- Pack, meld, roll, epoch inspection, confirmed prefix pruning and UTC schedules.
- Identity-pinned collector connections, operator/observer grants and Eyre sessions.
- Local first-run setup with visible progress and automatic collector enrollment.
- Stopped cross-host handoff with verified contents and permanent source fencing.
- Stock Vere 4.6 epoch import checks before source movement, plus explicit recovery of retained enrollment failures.
  See [import compatibility and recovery verification](docs/diagnostics/import-recovery-verification.md).
- New-ship selection between bare bootstrap and a supplied Recommended Apps pill.

Owned-pier import is implemented as explicit enrollment of a fully transferred,
stopped pier in the [storage convention](docs/pier-storage.md); broader live import
and network qualification remain. Loom-internal telemetry and custom runtime builds
are out of scope; the UI reports container resources as container resources.

## Components

| Component | Responsibility |
| --- | --- |
| `urbit-runner` | Host lifecycle, resource observations, maintenance, storage and durable operations |
| `urbit-collector` | Per-user runner connections, credentials and generic Eyre transport |
| `omabit` and the Omarchy plugin | Local onboarding, fleet/setup/monitoring UI and desktop/application policy |

This source monorepo is authoritative. Optional peer, gateway, social and agent
features follow a reliable core ship-running experience.

See [repository ownership and publishing](docs/repository.md) for the Tend split,
Git remotes, and the current Go module identity.

## Development

```bash
go test ./...
go vet ./...
go build -o ./build/ ./cmd/... ./backend/cmd/...
node scripts/test-panel-model.js
python3 -B scripts/test-panel.py --windows
python3 -B scripts/test-log-watch.py
```

RC.7 source includes a [persistent runtime-log watcher](docs/protocol.md#persistent-collector-log-watch-rc7)
to eliminate per-refresh SSH/PAM sessions. The updated collector and plugin ship
together. See the [RC.7 release notes](docs/release-rc7.md) and
[D-Bus incident follow-up](docs/diagnostics/2026-10-16-dbus-log-watch.md).

Existing ships and unrelated Docker installations are preserved during setup.
Host changes use explicit OS authorization; desktop clients do not receive Docker
group access. Development data/report formats are not a backwards-compatible
release contract.

## Documentation

- [Fresh installation and first ship](docs/install.md)
- [Sourcebox fresh native installation](docs/sourcebox-clean-install.md)
- [Owner-user runtime and desk editing](docs/owner-user-runtime.md)
- [Storage locations, import and deletion](docs/storage-management.md)
- [Services, shutdown and reboot behavior](docs/services.md)
- [Official-runtime policy](docs/runtime-policy.md)
- [Release pill composition and provisioning](docs/pill-release.md)
- [Current status and remaining release work](docs/project-status.md)
- [Architecture](docs/architecture.md)
- [Implementation plan](docs/implementation-plan.md)
- [Runner protocol](docs/protocol.md)
- [Ship sessions and credential storage](docs/ship-sessions.md)
- [Runtime qualification evidence](docs/runtime-qualification.md)

Historical development walkthroughs are under `docs/checkpoints/`; new installations
should use the installation guide above.
