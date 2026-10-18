# Install Omabit on this computer

Omabit has a single local setup flow for the owner's separate rootless engine, background
services, private desktop access and runner enrollment. The panel guides you through
it, and the same flow is available from a terminal.

**Implemented today:** Linux x86-64 with systemd/cgroup v2, Docker 29.7.2, the official
Vere 4.6 runtime, new comets/fakes, owned-key boot and stopped in-tree pier enrollment.
The local moon has passed key boot/recovery; broader owned import/network and clean
remote-host acceptance remain. This page describes source installation; published
release binaries have a separate [bundle guide](release-install.md). For the disposable
sourcebox reset, use the [clean-reinstall walkthrough](sourcebox-clean-install.md).
See also the [remote trial plan](remote-runner-acceptance.md).

## 1. Install the desktop component

For the normal GitHub installation in an Omarchy desktop:

```bash
omarchy plugin install https://github.com/thelifeandtimes/omabit.git --enable
```

The canonical repository's native entry points prepare checksum-pinned release binaries in the user's cache
on first use. It requires no Go build and performs no privileged host setup until
the user selects it. See [native plugin distribution](native-plugin.md).

From the repository checkout, as your ordinary desktop user:

```bash
bash scripts/install.sh
```

This builds the three Go binaries, installs the owned panel snapshot and enables it
in Omarchy. It needs Go 1.25+, Python and the Omarchy shell. Locally modified panel
files are preserved rather than overwritten.

The runner helper is bundled with the panel alongside the collector and desktop
helper, so the UI does not depend on the checkout path after installation. If an
existing shell session still displays the older panel, use `omarchy restart shell`
to load the new snapshot.

## 2. Set up the local host

### Build and install from source

Source for the three installed Omabit executables is at
<https://github.com/thelifeandtimes/omabit>: `cmd/omabit/` is the desktop/setup
helper, and `backend/cmd/urbit-collector/` and `backend/cmd/urbit-runner/` contain
the generic client and host service. Build with Go 1.25+:

```bash
git clone https://github.com/thelifeandtimes/omabit.git
cd omabit
CGO_ENABLED=0 go build -trimpath -o build/ ./cmd/... ./backend/cmd/...
python3 -B scripts/install-panel.py --enable
./build/omabit --format text setup
```

This installs your locally built tools. Headless installation is documented in
[owner-user runtime](owner-user-runtime.md). Native prebuilt downloads normally
live in `~/.cache/omabit/native-binaries/`; authorized host setup installs the
runner at `/usr/local/lib/urbit-runner/current/urbit-runner` and the pinned rootless
launcher under `/usr/local/lib/urbit-runner/rootless/29.7.2/`.

Docker/RootlessKit/slirp4netns/OpenSSH come from the host's OS packages. Upstream
Docker source is <https://github.com/moby/moby>, including the verified rootless
launcher. Ships run the pinned official Vere image; its source is
<https://github.com/urbit/vere>. See [runtime policy](runtime-policy.md) for the
qualified runtime artifact and version constraints.

### Guided local setup

Open **~+ → Setup → Set up this computer**.

The guided flow shows:

1. **Prerequisites** — platform, kernel, Docker and helper checks, with concrete
   missing/unsupported items. Install missing packages using Omarchy's package
   manager, then choose **Check again**. Setup does not downgrade packages or enable
   a system-wide Docker daemon.
2. **Desktop access** — creates a dedicated local Ed25519 key in private per-user
   storage. Existing setup keys are reused. Personal SSH configuration is untouched.
3. **Verified host helper** — downloads the pinned rootless launcher and verifies
   its content before requesting host authorization.
4. **Host services** — asks for OS authorization, selects the existing desktop user,
   creates `/srv/urbit/piers/` for that owner, installs the separate rootless engine
   and runner, configures restricted SSH access and enables lingering/reboot services.
5. **Desktop connection** — uses the locally authorized SSH host public key for
   strict host checking, verifies the runner/operator identities, and enrolls the
   connection automatically. An existing enrollment of that same runner is reused
   or updated without duplicating it.
6. **Ready** — establishes local browser defaults and takes you to **Add a ship**.

The terminal equivalent, from the checkout, is:

```bash
./build/omabit --format text setup
```

Run it as the desktop user. The helper uses `sudo` in an interactive terminal or
the system authorization prompt from the panel. Do not run the whole desktop setup
as root. To inspect readiness without installing:

```bash
./build/omabit --format text setup-status
```

## 3. Create and start a ship

Choose **New comet**, enter a required local name and find an available host HTTP
port. Confirm **Boot a new comet identity now with these settings**, then choose
**Boot comet**. Omabit verifies registration and submits one revision-bound Start,
opening the ship's management view. First boot can take several minutes while the runtime creates
the identity and boots the verified upstream pill.

First-run setup asks whether this is an always-on machine or a laptop. The choice
is saved locally. Laptop users also get **Set up remote server** in the fleet menu.
The empty collector displays a welcome and **Start Setup**, rather than an SSH error.

**Find an available port** asks the selected runner to check both registered port
reservations and real host listeners. Docker still rechecks binding races at Start.
The guest HTTP port is 8080; the chosen host port is its loopback publication.
Browser and +code buttons enable only after fresh, healthy runtime observations.

Startup requires **5 GiB available physical RAM and 8 GiB free disk**. The current
container limits remain 4 GiB RAM, two CPUs and 256 PIDs. Loom defaults to 31.
Source builds with `runtime-settings` offer exponents 30–34 (1–16 GiB). Loom and
storage selection are under collapsed **Advanced settings**; storage defaults to
`/srv/urbit/piers`. Native control defaults off and is CLI-only. The
`native-control-all-ships` capability supports CLI opt-in for every
ship class. Owned-key boot links to [Get an Urbit ID](https://urbit.org/overview/running-urbit/get-urbit-id).
See [development runtime settings](development-runtime.md) for
address-space labels, risks and recorded managed-fake acceptance. Once healthy, open
the ship and use the operator-only code-copy action for sign-in.

Long-running Vere processes default to **60-minute periodic snapshots**
(`--snap-time 60`). Runtime status shows the configured interval. A running process
adopts a changed default on its next explicit stop/start, without a host reboot.

Development fakes are clearly labelled as test ships. The key and import choices
describe their current limitations rather than accepting unsupported inputs.

### Development fake names

Fake registration now separates three names:

- **Instance name:** a unique local identifier on the runner, such as `bus-lab`.
- **Fake Urbit identity:** the identity passed to Vere, such as `~bus` or
  `~sampel-palnet`. Fake ships run with Ames disabled and do not use real ship keys.
- **Ship directory name:** the exclusive permanent parent containing `pier/`, such
  as `bus-data`. A full convention-relative pier path may include project groups.

For example:

```bash
./build/urbit-collector --runner local --expected-runner-id YOUR_RUNNER_ID \
  create-fake bus-lab --ship '~bus' --pier-name bus-data \
  --http-port 18088 --request-id register-bus-lab
```

The current host path is below `/srv/urbit/piers`. Use `--pier-path
fakeships/project/bus/pier` to select a project path; a basename-only `--pier-name
bus-data` defaults to `fakeships/bus-data/pier`. Existing-pier enrollment and stopped
moves are explicit actions in **Setup → Pier storage & livenet import**; see the
[storage convention](pier-storage.md).

The owner runner advertises `owner-user-runtime`, `human-pier-storage` plus
`fake-ship-options`, `pier-storage` and `host-handoff`. RC.3 uses schema 16 and the
permanent mount-root/pier layout. Legacy account installations require explicit
retirement/migration; schema upgrade does not move piers. Update both runners before using
[stopped host moves](host-handoff.md).

The form suggests unused instance names and unreserved HTTP ports, validates them
again before submission, and keeps input visible on a rejected or uncertain result.
The backend remains authoritative for races and actual host-port binding.

HTTP ports shown in the fleet are host bindings. Fake Ames is confined to container
loopback; outbound HTTP remains available. Guest Ames labels are internal settings;
explicitly published Ames labels describe host UDP bindings.

## Existing installations and reruns

- A verified, already-connected local setup returns ready without requesting host
  authorization again.
- An incomplete first install can be rerun: validated account/engine artifacts,
  host identity and desktop keys are reused. Setup never reinitializes a pier.
- Existing SSH grants are preserved; the new desktop grant is added once. Observer
  grants are not silently promoted. SSH host keys are not rotated to resolve errors.
- Custom service/SSH configuration is preserved with an actionable error. Finish
  active runner operations before asking setup to update the management program.
- Concurrent setup attempts are locked. A failed/uncertain host step is reported;
  no automatic mutating retry or data deletion is performed.
- A changed runner or SSH host identity stops enrollment rather than replacing its
  pin. Use **Manage runners** to explicitly review a changed connection.

Per-user setup files live under `$XDG_CONFIG_HOME/omabit/setup` (normally
`~/.config/omabit/setup`), mode 0700 with private files mode 0600. The dedicated
desktop key is unencrypted for unattended local access and constrained to the
runner's fixed RPC command; only its public key is sent to host installation.
Private key material is never returned in progress, snapshots or logs.

The canonical management/SSH-shell binary is
`/usr/local/lib/urbit-runner/current/urbit-runner`. Fresh installation does not require
checkpoint-specific binaries or copying a shell command out of an old walkthrough.

## Daily operation

The management and dedicated SSH services start without a desktop login. Eligible
previously healthy running ships resume after a clean or unclean host reboot; intentionally stopped,
uninitialized or uncertain instances remain stopped. See [services.md](services.md)
for the tested startup policy, diagnostics and service commands.

Use **Connect another runner** for an existing remote installation. Automatic host
setup currently targets this computer only; it does not modify remote hosts.

Panel 0.14 puts **Setup** in the full monitoring window as well as the fleet menu.
Ship views expose **Copy +code** for a current running ship and operator connection;
the helper owns a sensitive clipboard selection for up to 30 seconds. The code is
not passed through QML or saved in management history. Saved maintenance cards show
the UTC schedule, next run in local time, downtime/resume behavior and explicit
enable/disable/edit actions. From panel 0.15, **Save policy** enables the policy and
scheduled maintenance may stop a running ship. The resume checkbox remains an
explicit choice. Use the saved card's **Disable policy** action to suspend it;
existing saved policies are not changed merely by updating the panel.

The monitoring charts are **Container CPU** and **Container Memory**, with local
time and numeric unit axes. CPU uses Docker's per-core convention (100% = one CPU,
200% = the current two-CPU quota), averaged over successive runner observations.
The first CPU reading needs a second fresh sample before a rate can be shown.

Use **Add a ship →** for new comets, owned key boot and existing-pier enrollment.
The owned-key form derives the instance name from its @p. Confirming **Boot ship**
registers it and submits its first Start once. An uncertain registration result does
not trigger a blind Start, and a failed/uncertain boot needs explicit review.
After reload, the existing ship can be managed without supplying its key again.
Each ship overview includes **Runtime status** and **Runtime / dojo output**.
Logs are read-only, operator-only and bounded, refreshing every three seconds while
the overview is visible. Use Pause or Refresh to inspect output. Status includes
the lifecycle phase, operation/error and container exit/OOM evidence during boot
and recovery.
**Manage runners** concerns this collector's saved runner connections, including
offline hosts; use it to add or edit a connection. Ship HTTP configuration is in
the selected ship's **Connectivity & runtime** section.

## Where data lives

Host data belongs to the selected existing Unix user. Container UID0 maps to that
same owner, so `|mount`, ordinary `cp`/editor saves and `|commit` work directly.
See [owner-user runtime](owner-user-runtime.md). Default and private paths:

```text
/srv/urbit/piers/
├── fakeships/[project/]<ship-directory>/pier/
├── livenet/{galaxies,stars,planets,moons,comets}/[group/]<ship-directory>/pier/
└── archive/                      # Inactive backups, excluded from discovery
~/.local/share/urbit-runner/managed/
├── runner.sqlite                 # Omabit management database (+ WAL/SHM sidecars)
├── keys/                         # Private pending boot-key files
└── <private-instance-id>/         # Runtime metadata and pinned artifacts; no pier data
```

For example `bus-lab` (`~bus`) may use `/srv/urbit/piers/fakeships/bus/pier/`. The
private ID remains independent of the visible path, instance name and Urbit identity. Equal
fake identities have separate directories. Each pier contains its own `.urb/`.

**`runner.sqlite` is the management state**, including the stable runner ID,
registrations, desired/applied configuration, runtime ownership and recovery
markers, attributed operations/replay history and maintenance policies. Urbit's
own ship state lives inside each pier's `.urb` checkpoint and event-log storage
(including epoch LMDB files), not in this SQLite database.

Desktop connection records are separate:
`~/.config/urbit-collector/runners.json`, local setup/SSH files under
`~/.config/omabit/setup/`, and display preferences under
`~/.local/state/omabit/panel.ini` (with the corresponding XDG overrides respected).

## Verification status

The following versioned records describe historical dedicated-account deployments.
Current owner-user qualification is recorded separately in the RC.3 release report.

The historical **0.20** acceptance established the permanent layout. Fresh boot uses the permanent
mount-root/pier layout with no automatic shutdown or storage relocation. The local
fleet is migrated to schema 11; old store layouts are refused. A fresh fake passed
real direct-boot/stop/resume checks, and all eight registrations retained their prior
states after the migration. See the
[0.20 record](diagnostics/2026-10-02-permanent-piers.md).

The host installer has isolated fresh-install, partial-install rerun, identity/grant
preservation and refusal tests. Desktop setup tests cover key reuse, prerequisite
and authorization failures, pinned enrollment and changed-identity refusal. The
real QML fixture exercises the progress stream, duplicate-dispatch guard, completion
and local browser defaults. Full Go tests/vet, targeted race tests, panel model,
installation and mapped-window tests pass.

Existing-host setup has also passed live: it reused the `workstation` enrollment and
runner identity with the new app-owned SSH configuration. Panel 0.13 and its updated
runner passed named-fake registration/replay, `~bus` first
boot, stopped epoch inspection and same-identity container replacement/resume; see
[the live record](diagnostics/2026-10-01-named-fake.md). The existing fleet was preserved.

Panel 0.14 passed installation/load checks with reviewed narrow-window layouts,
journal-backed live roll dates and verified +code access. Its
[refinement record](diagnostics/2026-10-01-desktop-refinement.md) covers the checks and
preserved user policies/states.

Panel 0.15 passed installation/load checks. Live CPU readings passed on the three
running development containers, with their identities and saved policies preserved;
see the [CPU/layout record](diagnostics/2026-10-01-cpu-and-layout.md).

Panel 0.16 implements the selected storage convention, discovery/enrollment,
stopped moves, key boot and in-tree livenet import. The existing fleet migration,
direct filesystem rename/reconnection and archive exclusion passed live checks;
see the [storage record](diagnostics/2026-10-01-conventional-storage.md). Owned-key
boot and owned livenet import have fixtures and await supplied real test material.

From panel 0.17, owned key setup asks only for expected Urbit ID
and private key file; Advanced configuration exposes HTTP/Ames port overrides and
a pill URL/local file. Ports are automatically reserved when omitted. Configure
the plugin's `recommendedPillUrl` to the supplied release asset for Recommended
Apps. Bare bootstrap always selects Vere's canonical default. The historical
[0.17 record](diagnostics/2026-10-02-key-setup-refinement.md) documents the enrollment
fixes, desktop checks and live inert pill upload.

Panel 0.18 and its host runner are now installed, with the loaded UI version and
matching built/installed binaries verified. Unified provisioning, collector-only
runner management, activity layout and specific moon-name errors passed fixtures
and visual review. Post-update fleet checks preserved running containers,
intentional stops, uninitialized registrations and disabled policies; see the
[0.18 record](diagnostics/2026-10-02-unified-provisioning.md).

Panel 0.18.1 fixes the key/pill Browse crash using Qt's non-native file dialogs.
Both dialogs pass mapped open/cancel/reopen/selection checks, and the installed
patch is confirmed loaded; see the
[Browse crash record](diagnostics/2026-10-02-browse-dialog-crash.md).

Panel 0.18.2 adds explicit Start to owned registration and is confirmed loaded.
Mapped fixtures verify enablement, runner/instance binding and single dispatch;
the user's registered moon remains uninitialized, awaiting its first Start. See the
[registration Start record](diagnostics/2026-10-02-registration-start.md).

Panel 0.19.1 and the updated runner are deployed. The real moon is healthy after
reviewed same-pier recovery; runtime logs passed the real SSH path. Enabled panel
updates (`--replace --enable`) now cold-replace the shell on an unlocked desktop.
Do not add another shell restart or overlap an OS authorization prompt with that
update. See the [recovery/logs record](diagnostics/2026-10-02-moon-recovery-and-logs.md).

These fixtures do not claim a real blank-machine installation. A clean systemd host
with the supported packages is the remaining fresh-install acceptance environment.
