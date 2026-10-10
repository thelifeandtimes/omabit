# Omabit

A local Urbit fleet for Omarchy: run ships with the official Vere runtime, monitor
them from the desktop, and manage maintenance without keeping terminal sessions open.

This repository is the ship-running and orchestration project at
[thelifeandtimes/omabit](https://github.com/thelifeandtimes/omabit).
[Tend](https://github.com/thelifeandtimes/tend), the Urbit reminders application,
has its own repository, CLI, and release lifecycle.

**Status:** [**0.1.0-rc.1 is published**](https://github.com/thelifeandtimes/omabit/releases/tag/v0.1.0-rc.1),
consolidating development checkpoints through 0.23 into the first public prerelease.
See [binary bundle installation](docs/release-install.md),
[qualification and known issues](docs/diagnostics/2026-10-10-rc1-pill.md),
and the [draft pill plan](docs/pill-release.md). Final 0.1.0 is still pending.

## Get started

From a source checkout on a supported Omarchy machine:

```bash
bash scripts/install.sh
```

Then open **~+ → Setup → Set up this computer**. The guided flow checks prerequisites,
prepares private desktop access, asks for OS authorization, installs the background
services and connects your desktop automatically. Choose **Add a ship → New comet**
when it reports ready.

Prefer a terminal?

```bash
go build -o ./build/ ./cmd/...
./build/omabit --format text setup
```

See the [fresh installation guide](docs/install.md) for prerequisites, reruns and
troubleshooting. Source builds need Go 1.25+; prebuilt release packages are not yet
published. The current host implementation targets Linux x86-64, systemd/cgroup v2
and Docker 29.7.2.

## What works

- Dedicated-account rootless Docker and the pinned official Vere release.
- New comets and development fakes, with durable start/stop/resume operations.
- Owned-key bootstrap, including a live-tested moon, and stopped in-tree pier enrollment.
- Permanent per-ship mount directories: first boot creates `pier/` in place and keeps running.
- Management and restricted SSH services, including verified clean-reboot resume.
- A compact fleet menu and monitoring window with container RAM, tasks and health.
- Per-ship lifecycle/error/exit diagnostics and bounded read-only runtime logs.
- Pack, meld, roll, epoch inspection, confirmed prefix pruning and UTC schedules.
- Identity-pinned collector connections, operator/observer grants and Eyre sessions.
- Local first-run setup with visible progress and automatic collector enrollment.
- Stopped cross-host handoff with verified contents and permanent source fencing.
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
go build -o ./build/ ./cmd/...
node scripts/test-panel-model.js
python3 -B scripts/test-panel.py --windows
```

Existing ships and unrelated Docker installations are preserved during setup.
Host changes use explicit OS authorization; desktop clients do not receive Docker
group access. Development data/report formats are not a backwards-compatible
release contract.

## Documentation

- [Fresh installation and first ship](docs/install.md)
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
