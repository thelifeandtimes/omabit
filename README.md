# Omabit

A local Urbit fleet for Omarchy: run ships with the official Vere runtime, monitor
them from the desktop, and manage maintenance without keeping terminal sessions open.

This repository is the ship-running and orchestration project at
[thelifeandtimes/omabit](https://github.com/thelifeandtimes/omabit).
[Tend](https://github.com/thelifeandtimes/tend), the Urbit reminders application,
has its own repository, CLI, and release lifecycle.

**Stable release preparation:** [**0.1.0**](https://github.com/thelifeandtimes/omabit/releases/tag/v0.1.0), a Linux/AMD64 release
using the existing pier-owning Unix user for the runner and its rootless engine.
See [binary bundle installation](docs/release-install.md),
[release notes and support scope](docs/release-0.1.0.md),
and [Recommended Apps pill provenance](docs/pill-release.md).

**Channels:** `master` is protected, PR-only and reserved for stable minor/major
releases. `edge` is opt-in bleeding-edge work, explicitly not stable. `develop` has
no stability guarantee. See the [development/versioning policy](docs/development-policy.md)
and [contributor guide](CONTRIBUTING.md), including explicit edge installation/update
commands. Backend changes require a new minor or major release line.

**RC.15** fixes first-time host setup on stock Omarchy by reusing existing
OS-assigned user-namespace IDs without requiring a prior Omabit installation.
The guided checklist and OS authorization flow remain the setup path; technical
host diagnostics are under **View details**.

**RC.14** adds a retained bootstrap for stock Omarchy, verified immutable UI/tool
bundles, automatic desktop activation/recovery and isolated development profiles.
Existing default-profile enrollment and preferences are retained. See
[how desktop releases and profiles work](docs/desktop-development.md).

RC.12 includes the [adversarial-review fixes](docs/reviews/2026-10-18-rc11-remediation.md),
with the same desktop layout and component split. Its frozen artifacts passed
[power-cut, live runtime/upgrade, authenticated Ames and desktop qualification](docs/diagnostics/2026-10-18-rc12-release-qualification.md).
RC.12 uses schema20; existing RC.10/RC.11 owner-user hosts use the documented
[one-time offline management upgrade](docs/management-offline-upgrade.md).

**RC.13:** [distribution-native host compatibility](docs/host-compatibility.md)
adds Pacman and Debian/Ubuntu package authentication, shared engine/API checks,
headless/fleet reports and safe startup-guard migration. It uses schema21 and retains
the existing component boundaries and desktop workflow. Fresh-install testing on
two Omarchy versions and full runtime qualification of new engine/distro profiles
remain pending; see the [publication and verification record](docs/diagnostics/2026-10-19-rc13-publication.md).

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
RC.8 simplifies initial setup and derives import identity, destination and port
from the source/location selection, with progress receipts and ship-named piers.
RC.9 adds Ames publication for new live enrollments, universal pre-move epoch
checks and explicit retained-enrollment recovery, with a stable-upgrade release gate.
RC.10 adds configurable loom and default-off native control, preserves settings
across older-client edits and signed host moves, and verifies an in-place RC.9 upgrade.
RC.11 simplifies boot forms with collapsed Advanced settings, loom30–34 choices
(default31), default storage and CLI-only native control, plus Urbit ID guidance.
RC.12 hardens interruption recovery, identity/replay and credential contracts,
adds transactional management upgrades and explicitly migrates legacy Ames
publication without changing the desktop layout.
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

For an existing native desktop installation, use
`omarchy plugin update thelifeandtimes.omabit`. RC.14's bootstrap prepares the
selected release and activates it when ready; see [updates](docs/native-plugin.md#updates)
for diagnostics, explicit version pins and legacy recovery.

Then open **~+ → Setup → Set up this computer**. The guided flow checks prerequisites,
prepares private desktop access, asks for OS authorization, installs the background
services and connects your desktop automatically. Choose **Add a ship → New comet**
when it reports ready.

To manage an existing remote runner, use **Setup → Manage runners → Connect another
runner** with its authorized SSH connection. Remote-only desktop use needs no local
Docker engine. Local host setup and remote enrollment are separate operations.

For development from a source checkout:

```bash
bash scripts/install.sh
go build -o ./build/ ./cmd/... ./backend/cmd/...
./build/omabit --format text setup
```

See the [fresh installation guide](docs/install.md) for prerequisites, reruns and
troubleshooting, or [binary bundle installation](docs/release-install.md) for the
published release assets. Source builds need Go 1.25+. Hosting targets Linux x86-64
and systemd/cgroup v2. Published RC.12 requires Docker 29.7.2; RC.13
accepts stable Docker 27.2.0–29.x subject to API 1.47 and the documented
[host compatibility and package-authentication checks](docs/host-compatibility.md).

## What works

The generic, non-Omarchy runner and collector entry points are in
[`backend/`](backend/README.md). The desktop-specific Go helper is `cmd/omabit`;
shared implementation remains under `internal/` in one Go module.

- A separate owner-user rootless Docker engine and the pinned official Vere release.
- New comets and development fakes, with durable start/stop/resume operations.
- Owned-key bootstrap, including a live-tested moon, and stopped in-tree pier enrollment.
- Permanent per-instance mount directories with named pier children; first boot creates the child in place and keeps running.
- Management and restricted SSH services, including verified clean and unclean host-reboot resume.
- A compact fleet menu and monitoring window with container RAM, tasks and health.
- Per-ship lifecycle/error/exit diagnostics and bounded read-only runtime logs.
- Pack, meld, roll, epoch inspection, confirmed prefix pruning and UTC schedules.
- Identity-pinned collector connections, operator/observer grants and Eyre sessions.
- Local first-run setup with visible progress and automatic collector enrollment.
- Stopped cross-host handoff with verified contents and permanent source fencing.
- [Advanced loom settings and CLI-only native access](docs/development-runtime.md) for every ship class, with separate requested/recorded settings. RC.10 fake/comet native execution and repeated in-place management upgrades passed; see [qualification](docs/diagnostics/2026-10-18-rc10-qualification.md).
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

RC.14 provides a retained bootstrap, immutable UI/tool releases and isolated
test profiles for stock Omarchy. See [desktop development and upgrade testing](docs/desktop-development.md)
for candidate builds, version selection, fresh profiles and guarded VM scenarios.

```bash
go test ./...
go vet ./...
go build -o ./build/ ./cmd/... ./backend/cmd/...
node scripts/test-panel-model.js
python3 -B scripts/test-rc11-panel-offscreen.py --require-qt
python3 -B scripts/test-panel.py --windows
python3 -B scripts/test-log-watch.py
python3 -B scripts/test-import-recovery.py
```

RC.7 source includes a [persistent runtime-log watcher](docs/protocol.md#persistent-collector-log-watch-rc7)
to eliminate per-refresh SSH/PAM sessions. The updated collector and plugin ship
together. See the [RC.7 release notes](docs/release-rc7.md) and
[D-Bus incident follow-up](docs/diagnostics/2026-10-16-dbus-log-watch.md).

Existing ships and unrelated Docker installations are preserved during setup.
Live operations on the user's hosts follow the [protected-ship policy](docs/protected-ships.md).
Host changes use explicit OS authorization; desktop clients do not receive Docker
group access. RC formats remain developmental; stable 0.1.0 is gated on the
[tested, non-destructive upgrade contract](docs/upgrade-policy.md).

RC.9 integrates the [Ames and import-recovery contributions](docs/diagnostics/pr2-pr3-integration.md)
from PRs #2/#3, adapted to the guided/named-pier flows. It preserves existing
enrollment records and adds explicit recovery rather than requiring a fresh install.

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
