# Sourcebox hard reset and automatic resume policy — 2026-10-14

**Diagnosis:** management restarted correctly. The three previously running ships
were withheld by Omabit's clean-exit-only startup policy. **Recommendation:** allow
one runner-controlled initialized-pier resume after a host reboot, including an
unclean exit. The exit-0 restriction is an Omabit policy, not a Vere requirement.

This record describes the initial read-only investigation. Its recommendation was
subsequently authorized and **implemented and deployed in schema 15**; see the
[deployment and recovery record](2026-10-15-reboot-resume-deployment.md) and
[services and startup behavior](../services.md) for the current contract. The
incident observations below precede that implementation and deployment.

## Observed incident

The user reported a hard reset for an unrelated host issue. Inspection at
**23:27–23:30 UTC (16:27–16:30 PDT)** found:

- New boot `13e98cc6-433b-4060-97a3-a147044cf4be`, first journal entry **23:09:02 UTC**.
  The prior boot's last retained journal entry was **23:01:18 UTC**. The inspected
  prior system-manager and runner-account log tails contained no graceful shutdown
  sequence. The investigation did not attempt to diagnose the unrelated reset cause.
- Dedicated `urbit-docker.service`: enabled, active/running, PID 2135, started
  **23:09:04 UTC**, zero service restarts.
- `urbit-runner.service`: enabled, active/running, PID 3784, started **23:09:06 UTC**,
  zero service restarts, with `--resume-running` still present.
- Installed runner hash matches the full development deployment:
  `76b42bb5c26c682a95e914c1274a6c318623a1cd1a48860f498552e45f0dd49b`.
- All three retained containers are exited, not running, not dead, not OOM-killed,
  with empty Docker state errors, exit **255**, restart policy **no**, restart count 0.

| Instance | Retained container prefix | Docker finished time (UTC) | Desired / observed |
| --- | --- | --- | --- |
| `remote-smoke` | `2d22a600b6e93` | 23:09:04.855582975 | running / failed |
| `remote-comet` | `ee714d6b60fa` | 23:09:04.860062175 | running / failed |
| `panel-check` | `fc835d755dba` | 23:09:04.860154674 | running / failed |

The clustered finished times coincide with Docker startup, consistent with daemon
reconciliation of processes lost in the hard reset. They should not be treated as
proof of three new Vere application crashes after boot. The current runner-account
journal tail was dominated by SSH reads and did not include the daemon's startup
messages, so this record does not claim direct restore-log confirmation.

All four remote piers' recorded pier, `.urb` and exclusive parent identities match
the current filesystem, including inode and nanosecond birth time. This is not a
missing/replaced-pier case. There are no active operations or independent incomplete
provisioning/maintenance markers on the three affected instances. The reviewed
kernel log search found no matching OOM or filesystem/I/O error reports; available
RAM was about 89 GiB and free disk about 460.5 GB.

Retained Vere logs show the earlier successful initialized boots, with no new boot
after this reset. `remote-comet` also has earlier Groups glob HTTP 504 messages;
these do not establish a cause for the reset or exit-255 states. `rc1-pill-check`
remains intentionally stopped. The workstation's moved `panel-check` source remains
fenced. No managed ship was operated on during this investigation.

## Why the current implementation waits

1. Docker restart is disabled and enforced by container containment validation
   (`internal/runtimequalify/docker_linux.go`). The runner owns lifecycle decisions.
2. `StartupResumeReady` in `internal/runtimequalify/startup.go` accepts only a freshly
   observed stopped runtime with exit **0**, no OOM flag and no active/dead state.
3. `applyStartupResume` rechecks that same terminal-state restriction immediately
   before removing the old container (`startup_linux.go`).
4. `observeManaged` classifies a nonzero exited container as failed, unhealthy and
   recovery-required (`managed_linux.go`). There is no automatic operation to fail
   here: the startup candidate is rejected before a `startup-resume` job is queued.
5. Subsequent management starts exclude recovery-required records, so restarting
   the service repeatedly is not a recovery mechanism (`internal/managed/startup.go`).

The available explicit Stop reconciles/removes an authenticated exited container
and preserves terminal evidence; it does not repair or rebuild the event log.
Start then launches a replacement against the same initialized pier. An automatic
path can perform this reconciliation without requiring the user to toggle desired
state through stopped.

## Vere persistence and normal recovery

Reviewed the deployed runtime's source revision
**`5a7614f69bbc783736b08965fdc0a2534bb42e82`** (Vere `4.6-5a7614f`):

- [`u3_lmdb_init` and `u3_lmdb_save`](https://github.com/urbit/vere/blob/5a7614f69bbc783736b08965fdc0a2534bb42e82/pkg/vere/db/lmdb.c):
  the event store uses LMDB write transactions and `mdb_txn_commit`; the environment
  setup does not enable `MDB_NOSYNC`, `MDB_NOMETASYNC` or `MDB_MAPASYNC`.
- [`_mars_fact`, `_mars_flush` and `_mars_disk_cb`](https://github.com/urbit/vere/blob/5a7614f69bbc783736b08965fdc0a2534bb42e82/pkg/vere/mars.c#L235-L270):
  computed events are queued for persistence; their event-associated effects wait
  until the committed event counter reaches them
  ([release condition](https://github.com/urbit/vere/blob/5a7614f69bbc783736b08965fdc0a2534bb42e82/pkg/vere/mars.c#L835-L869)).
- [Normal worker startup](https://github.com/urbit/vere/blob/5a7614f69bbc783736b08965fdc0a2534bb42e82/pkg/vere/main.c#L3062-L3076)
  loads the pier, invokes `u3_mars_play`, then enters normal work. Replay after a
  stale checkpoint is part of ordinary startup.
- The [runtime reference](https://docs.urbit.org/user-manual/running/vere#truncate-event-log)
  describes ship state as a pure function of the ordered event log, and rebuilding
  state from retained events.

Thus an interrupted normal Vere process is ordinarily a **resume/replay** case.
The configured 60-minute snapshot interval is not a 60-minute event-durability
window. It can increase replay work after interruption. Durability applies to
committed events and assumes the storage stack honors its writes; it is not a claim
that physically damaged storage can always be repaired by restarting.

Urbit's event transaction does not encompass Omabit's host-side import, deletion,
handoff or maintenance orchestration. Those operations have separate durable
journals and must retain their phase-specific reconciliation rules. Omabit itself
uses SQLite WAL with `synchronous(FULL)`; its current manual policy is not caused by
the absence of a durable management database.

## Options

| Option | Result | Assessment |
| --- | --- | --- |
| Existing clean-exit-only policy | Services restart; hard-reset ships wait for Stop/Start | More restrictive than normal Vere crash recovery requires |
| Runner-controlled reboot resume | Resume eligible initialized ships once after a host reboot, including nonzero exits | **Recommended**; reuses existing ownership checks, work serialization and durable receipts |
| Runner-controlled continuous restart | Also recover exits during an otherwise running host, with backoff and a retry limit | Reasonable separate policy; not required to solve reboot recovery |
| Docker `always` / `unless-stopped` | Docker restarts the original container command | Possible with architectural changes, but not a drop-in setting for the current runner |

Docker's [restart policy reference](https://docs.docker.com/engine/containers/start-containers-automatically/)
also notes that `on-failure` alone does not restart containers when the daemon
restarts. `always` and `unless-stopped` handle daemon restart differently from
intentional stops.

Changing Docker policy on these containers directly would fail Omabit's current
containment check. More generally, retained first-boot containers can still contain
creation flags/pill/key mounts; maintenance containers must not be re-executed as
if they were normal runtimes. Docker also does not implement Omabit's transfer
fences, revision checks, host-work serialization or per-boot operation accounting.
These are reasons to keep restart ownership in the runner, not reasons to require
manual recovery after every hard reset. The three affected containers already use
resume commands, but a fleet-wide solution must cover first-boot survivors too.

## Recommended behavior and implementation implications

For the existing automatic-resume mode, the expected experience should be:

1. Detect the host boot transition and load desired state. Previously running,
   initialized instances are eligible even if their old container exited nonzero.
   Intentional stops and already-running containers need no restart.
2. Verify the recorded pier, container/network ownership and applied configuration.
   Preserve blocks for handoff fences and interrupted provisioning, import,
   deletion or maintenance. Do not clear a generic recovery flag indiscriminately.
3. Persist one automatic attempt per runtime generation and host boot. Capture the
   old terminal state/logs, remove the exited container, then use initialized resume
   against the existing pier with the pinned runtime. Do not replay creation flags
   or restore an old pier checkpoint as a separate management action.
4. Start serially with resource admission. Let Vere perform its replay and verify
   the expected ship identity/HTTP readiness. Present **Resuming after host reboot**
   while startup is in progress; if replay progress is available, present it rather
   than treating a temporarily unavailable HTTP endpoint as an immediate crash.
5. If that actual resume fails, preserve its diagnostics and request intervention.
   Restarting the management service must not reset the attempt budget or repeatedly
   restart a process that is already replaying.

The existing code already provides most of the replacement path, host lock,
identity checking and once-per-boot journal mechanism. The change is more than
removing a single `ExitCode != 0` check: both admission and final removal gates must
agree, observation currently reports `failed` rather than `stopped`, and the generic
recovery flag mixes ordinary runtime exits with unrelated unresolved operations.

For a strictly reboot-specific policy, persist the boot identity alongside runtime
launch/healthy observations, rather than using it only in startup request hashes.
That distinguishes a host reboot from a later management-only restart and avoids
silently converting a known pre-existing failure into a new retry. Preserve the
candidate/attempt across interrupted management startup, and retain operator Stop
and deletion/name-reuse race protections.

Verification should cover clean and unclean exits (including 255/137), a changed
host boot versus a management restart, retained creation containers becoming resume
containers, intentional stops, moved sources, partial host operations, storage
identity changes, long replay, resource delays and a failed attempt without a loop.
A disposable-pier unclean-exit test and a separately scheduled real host hard-reboot
check are distinct acceptance steps. No such destructive qualification was performed
as part of this investigation.

**Bottom line:** there is no identified Vere-specific reason to require manual
Stop/Start for these three otherwise resolved piers solely because the host reset
uncleanly. Automatic initialized-pier resume is the recommended policy change.
