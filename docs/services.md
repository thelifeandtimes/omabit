# Management startup and reboot behavior

**Accepted live: service handoff, management-only restart, controlled engine restart
and clean host reboot with automatic ship resume.**

The schema-15 unclean-reboot policy is
[deployed on both runners](diagnostics/2026-10-15-reboot-resume-deployment.md), with
post-hard-reset pier replay and durable eligibility verified. Automatic unclean
resume now also [passed a real sourcebox hard reset](diagnostics/2026-10-15-hard-reboot-acceptance.md):
three automatic resumes, preserved identities, and the intentional stop retained.

The sourcebox permanent-layout fleet also passed a user-managed reboot on October
10 UTC: two previously running ships resumed through journaled startup operations,
and the transferred stopped ship stayed stopped. See the
[remote reboot acceptance](diagnostics/2026-10-10-sourcebox-reboot.md).

After a later unclean host restart, sourcebox's retained fake/comet containers
required explicit review and Stop/Start reconciliation. Both were recovered on
October 13, and the btrfs stable-identity hotfix from PR #1 was deployed with their
running containers preserved. See the
[recovery and hotfix record](diagnostics/2026-10-13-remote-recovery.md).

Omabit uses two small service units on the configured development host:

- **`urbit-runner.service`**, in the lingering `urbit-runner` user manager: starts
  after its dedicated rootless Docker service and runs the current management
  binary with `--resume-running`.
- **`urbit-runner-ssh.service`**, a system unit: runs the existing dedicated,
  loopback-only SSH configuration. It preserves the existing host key, grants and
  restricted bridge; no system-wide SSH configuration is changed.

Both restart on process failure with a bounded start rate. The management unit
wants Docker but is not stopped when Docker is temporarily unavailable, so the
API can still expose durable inventory and errors. Stopping management leaves
running ship containers alone.

## Startup resume

The user selected automatic resume of previously running ships. The service opts
into that policy with `serve-development --acknowledge-development --resume-running`.
Foreground management without that flag retains manual-only startup behavior.

**Schema 15 adds unclean-reboot resume.** The runner records the Linux boot ID,
container and configuration revision when an initialized ship is healthy. On a
later host boot it may resume that same pier even after a nonzero/OOM exit. Vere's
normal startup performs event-log replay; a clean exit is not required. See the
[incident and policy analysis](diagnostics/2026-10-14-hard-reset-policy.md).

At management startup:

1. Reconcile interrupted operations without replaying them.
2. Select initialized, resolved instances whose desired state is running and whose
   applied/configuration revisions match. Skip intentional stops, incomplete
   provisioning, unapplied changes and previously known failures. Persist the
   container/revision-bound pending boot before polling, so an exited-container
   observation or a management restart before dispatch cannot lose eligibility.
3. Wait through temporary engine unavailability using bounded read-only observation.
   Already-running ships need no operation. A candidate must have exactly one
   authenticated retained runtime with fresh terminal evidence. A healthy record
   from a different host boot authorizes nonzero/OOM exits; legacy records and
   management-only restarts retain the clean-exit-only rule. Missing, active, paused,
   restarting or dead containers do not qualify.
4. Journal a `startup-resume` operation attributed to `system:startup`. Each pier
   gets at most one attempt per Linux host boot ID, including failed attempts.
5. Under the existing host work lock, recheck ownership, containment, terminal state
   and RAM/disk admission. Preserve terminal evidence, remove the exited container and create a normal
   initialized-pier replacement. Never restart a creation command with `-c`/`-F`.
6. Verify the same ship identity and persist the outcome. A failure remains visible
   for explicit stop/review/start; service restarts do not create a retry loop.
   A failed/interrupted attempt discards automatic eligibility until a successful
   healthy runtime observation establishes a new record.

Startup resume is internal, not a new user RPC action. It uses the official runtime,
existing port/revision/identity rules and current SQLite operation journal. It never
bootstraps an uninitialized registration or continues a maintenance job.

The existing UI displays the startup operation and its reboot/replay message.
Readiness allows up to 10 minutes for fake ships and 20 minutes for networked ships;
HTTP unavailability during this interval does not restart a replaying process.
Insufficient RAM/disk or a readiness failure produces one failed operation with
diagnostics, not a retry loop. Continuous same-boot crash restart is not enabled.

The additive schema-15 migration preserves schema 11–14 records but does not invent
historical boot evidence. Existing healthy runtimes become eligible after the new
runner observes them. Ships already in recovery at upgrade need one reviewed
Stop/Start; future host reboots use their newly recorded healthy boot. Older runners
refuse the upgraded database.

## Shutdown ordering and budgets

The dependency order stops management before Docker, leaving Docker to shut down
the ships cleanly. The installed budgets are:

| Layer | Stop budget |
| --- | ---: |
| Vere container | 300 seconds |
| Dedicated Docker daemon | 310 seconds |
| `urbit-docker.service` | 360 seconds |
| Dedicated `user@UID.service` | 480 seconds |

The last override applies only to the `urbit-runner` account. This host's original
user-manager budget was only five seconds, which could cut short the inner graceful
shutdown. Abrupt power loss can use the reboot-resume policy above. Interrupted
management operations and failed resume attempts retain explicit review.

## Install on this configured host

For a fresh installation, use the [guided local setup](install.md), which includes
the account, rootless engine, SSH access, services and collector enrollment. The
service-only script below remains available for an already configured host.

Build and inspect the units under `deploy/systemd/`. Ensure no maintenance or other
operation is active, then authorize the installer:

```bash
sudo /usr/bin/bash /home/thelifeandtimes/repos/omabit/scripts/install-services.sh
```

The installer requires the existing account, rootless engine and dedicated SSH
configuration. It enables lingering, stops the exact foreground management/listener
commands used by the walkthrough, installs the current runner and app-owned service
files, and enables/starts both services. It does not restart Docker or stop ships.
Collector connections briefly reconnect during the management/SSH handoff.

The dedicated Docker config must be the generated empty config or the new shutdown
config; unrelated edits are refused. On an installation error, inspect the reported
state before retrying; there is no automatic rollback or data deletion.

## Local verification

Full Go tests and vet passed, along with race checks for the managed server, runtime
driver, rootless bundle and runner CLI. Native binaries and a Linux/ARM64 runner
cross-build were produced. The latter does not qualify ARM64 runtime execution.
Both service units passed `systemd-analyze verify`; the installer passed `bash -n`
and the dedicated Docker configuration passed `dockerd --validate`.

Fixtures cover initialized-only resume, no creation-flag replay, exact identity,
clean/non-OOM terminal evidence, resource/journal refusal, stale observation and
operator-stop races, skipped interrupted work, host-work serialization, temporary
engine unavailability, explicit opt-in and once-per-boot deduplication across
management restarts. They do not substitute for the live checks below.

## Live installation evidence — September 30 UTC

The user ran the installer successfully. At **05:42:49 UTC**, independent checks
confirmed the expected runner identity/mode and the new `startup-resume` capability.
Management PID 3731903 was in the dedicated user manager's
`app.slice/urbit-runner.service` cgroup, running with `--resume-running`. The SSH
system service was enabled/active, PID 3732055, with zero automatic restarts.
Lingering was enabled and the dedicated user-manager stop budget was eight minutes.
The installed Docker drop-in specified 360 seconds and its config specified 310;
the daemon config takes effect at the next engine start.

All four initialized ships were healthy at their applied configuration revisions,
with the **same container IDs as before installation**:

| Instance | Container before/after service handoff |
| --- | --- |
| alpha | `78116edfda0f9f10ae5ee0d385699b3656dc87639fbac7fa0176e095d38850a7` |
| comet-check | `5a69955d97a86bde1dc91e9aed72e8eb010cecc3fc53f92cc3985f15d4b374ac` |
| import-check | `c869eeef2de0f538591bed9567ed30e8947b0e9d569b2749b540293ee95040a9` |
| panel-check | `59be6fba033c898e22b35a057390f78f5ab5c0335f528de27805b1a7a6433d89` |

`ssh-check` remained uninitialized. Both schedules were disabled, with no active
operations, startup-resume attempts, recovery requirements or incidents. This
accepts the foreground-to-service handoff, not engine-restart or reboot behavior.

### Management-only restart accepted

After the user restarted `urbit-runner.service`, checks at **05:51:16 UTC** showed
management PID **3769036** (replacing 3731903), with `--resume-running` still enabled.
All four ship container IDs in the table above were unchanged and healthy, with
matched applied/configuration revisions and preserved identities. `ssh-check`
remained uninitialized; both schedules stayed disabled. There were no startup-resume
jobs, active operations, recovery requirements or incidents. SSH remained enabled
and active at PID 3732055 with zero automatic restarts.

Before the controlled engine test, the host reported about **6.4 GiB available RAM**
and **395 GiB free disk**; actual startup admission will recheck both.

### Controlled engine restart accepted

The user restarted `urbit-docker.service` and `urbit-runner.service` together.
Docker returned at PID **3827102** with the updated configuration and management at
PID **3827407** with `--resume-running`. Independent snapshots at **06:07:53–06:08:03
UTC** confirmed four successful startup-resume operations, attributed to
`system:startup`, and all four initialized ships healthy:

| Instance | Startup operation | Finished UTC | Replacement container |
| --- | --- | --- | --- |
| import-check | `op-7bc31aaee45d1d8b0f22eea439317699` | 06:07:19.930645799 | `65443e5dc66886fd05e684dc8a76d55855f989bdb51c2b1f5b3133323998310b` |
| panel-check | `op-a947c9113c2cc98776d7c559abe01730` | 06:07:26.059522951 | `452d121ede26353ece7f8f7afcbf508a0b9dede1d0a2f970f5e0e48a98eb5763` |
| alpha | `op-90a8bd26caabed5b0502376e035c1445` | 06:07:32.202248908 | `4f514543a375dfcaf86dd098d449fad524b15bf2a887c71c544fd068ba8188de` |
| comet-check | `op-fc550bdf7837d97db59ca677fd4d0dad` | 06:07:38.22270205 | `2e012bdab435a10f436054f2c57e80e86ba70815baa9ee5d8b8c7a771f5a08b5` |

Each instance had exactly one startup attempt and a new container. Alpha retained
revision 2/2; the other initialized instances retained 1/1. The fake identities
remained `~zod`, and comet-check retained
`~nolpel-modsyd-livsep-novmeg--sallen-tassul-padlud-litzod`. The first startup job was
queued at 06:07:13.694495549 UTC and the last completed about 25 seconds later;
queue timestamps are not worker-dispatch timestamps.

`ssh-check` remained stopped/uninitialized, both schedules remained disabled, and
there were no active operations, recovery requirements or incidents. SSH remained
enabled/active at PID 3732055 with zero automatic restarts. Available RAM afterward
was about 5.7 GiB.

This was an engine restart within host boot
`76106296-4bcf-4d82-85fa-35ce22484217`. It verifies clean shutdown/replacement and
automatic resume on this fleet. The separate actual-reboot result follows.

### Host reboot accepted

The user rebooted normally and returned without manually starting management or
ships. The host boot ID changed to **`6f75f84d-7c19-4783-af44-94286ab79d5f`**.
Independent checks at **06:36:20 UTC on September 30** confirmed:

- Dedicated account state `lingering`, `Linger=yes`, runtime path `/run/user/959`.
- Docker PID **1051** in `app.slice/urbit-docker.service` and management PID **1525**
  in `app.slice/urbit-runner.service`, both under `user@959.service`.
- Management ran with `--resume-running`; Hello retained the same runner ID,
  development mode, startup-resume capability and expected SSH operator identity.
- Dedicated SSH was enabled/active at PID **896**, with zero automatic restarts,
  started at **06:34:58 UTC**. The dedicated user-manager stop budget remained eight minutes.

Exactly four new `system:startup` operations succeeded in the new boot:

| Instance | Startup operation | Finished UTC | Post-reboot container |
| --- | --- | --- | --- |
| alpha | `op-07b29bf4c1fc0434f34d02494624d4ae` | 06:35:06.228364363 | `075247b30dca16f9296e12b3d27487f5eb256a8c5e0334985ea2294f47403a73` |
| import-check | `op-d32ba733fa1826ffa737f767037e0fbb` | 06:35:12.171209476 | `4d13cbd7720fd5a7e129ea694bcb917e3c08469b79f3b7b16aec2a25c555a06b` |
| panel-check | `op-a9e0e83dbf83604500088120cfc4ab87` | 06:35:18.086218043 | `e95e648f9c0ea796e2c4a460fc07ee12ac4294c33e80ae8e73eb1f260312d345` |
| comet-check | `op-fb79183bfa092d9ec5b032a69b72a683` | 06:35:23.980343723 | `a844bba35e64eed0a3730e852b8ef31aa987c14942bdf92c2fa29884874bdad4` |

The first new job was queued at 06:35:00.118471267 UTC; all four completed about
24 seconds later. Their request IDs differ from the previous boot's attempts,
which remain recorded as earlier successful operations, not duplicate attempts.

All four initialized instances were running/healthy with preserved ship identities
and configuration revisions (alpha 2/2, others 1/1). Comet-check retained
`~nolpel-modsyd-livsep-novmeg--sallen-tassul-padlud-litzod`. The uninitialized
`ssh-check` remained stopped at 1/0, both schedules remained disabled, and there
were no active operations, recovery requirements or incidents. This accepts the
normal clean-reboot path for the current official-runtime development fleet.

## Verify installation

```bash
loginctl show-user urbit-runner -p Linger -p RuntimePath
sudo /usr/bin/systemctl --user --machine=urbit-runner@.host status urbit-runner.service --no-pager
sudo /usr/bin/systemctl status urbit-runner-ssh.service --no-pager
sudo /usr/bin/systemctl --user --machine=urbit-runner@.host is-enabled urbit-runner.service urbit-docker.service
sudo /usr/bin/systemctl is-enabled urbit-runner-ssh.service
```

Check the fleet through its pinned connection:

```bash
/home/thelifeandtimes/repos/omabit/build/urbit-collector \
  --runner workstation \
  --expected-runner-id runner-f496feed1f7b08f31a5d82b94ac2f5fd snapshot
```

A management-only restart should preserve running ship container IDs and create no
startup-resume operations for those ships:

```bash
sudo /usr/bin/systemctl --user --machine=urbit-runner@.host restart urbit-runner.service
```

## Restart/reboot checks — accepted workflow

On first installation, the daemon shutdown timeout takes effect at the next engine
start. The checks below have now passed on this host. On this
development fleet, a controlled restart of **both** services tests shutdown order
and startup resume; it briefly interrupts ship HTTP access:

```bash
sudo /usr/bin/systemctl --user --machine=urbit-runner@.host \
  restart urbit-docker.service urbit-runner.service
```

The runner should resume eligible ships serially, show succeeded `startup-resume`
operations, preserve identities/configuration and leave the uninitialized or
intentionally stopped registrations alone. Stop and inspect any failed attempt;
do not clear its journal or retry it by repeatedly restarting services. Repeating an
engine shutdown in the same boot after a startup attempt requires explicit ship
starts; startup resume is deliberately once per boot.

An actual host reboot is a separate manual check when revalidation is needed. Afterward,
the services should be enabled/active without logging in as `urbit-runner`, and the
same fleet/operation checks should pass. The desktop collector still needs its
ordinary SSH credentials; a locked desktop keyring is not a server startup failure.

## Operations and diagnostics

```bash
sudo /usr/bin/journalctl _UID=959 _SYSTEMD_USER_UNIT=urbit-runner.service --no-pager -n 80
sudo /usr/bin/journalctl -u urbit-runner-ssh.service --no-pager -n 80
sudo /usr/bin/journalctl _UID=959 _SYSTEMD_USER_UNIT=urbit-docker.service --no-pager -n 80
```

UID 959 is specific to this host. A service start-limit failure can be inspected and
then reset with `systemctl reset-failed` in the corresponding system/user manager.
For disposable qualification work, stop the management service rather than starting
a second foreground manager. Restore it with `systemctl start urbit-runner.service`
in the dedicated user manager afterward.
