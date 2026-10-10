# Checkpoint 6M: fleet menu, setup and monitoring workspace

Omabit **0.4.0** reorganizes the unified plugin around three surfaces. The user
accepted the data coverage of 6L and requested this visual/interaction pass.
Version 0.4.0 is installed; see the
[validation and visual-inspection record](../diagnostics/2026-09-28-fleet-desktop.md).

## Information architecture

### Fleet menu

Runners form compact groups with a reachability dot, display name and host/location.
Each ship has a descriptive status indicator, ship/instance names, Ames and HTTP
ports, `+code`, browser, start/stop and monitor controls. Status combines connection
freshness, recovery, active work, runtime observation and health; color is accompanied
by text/tooltips. Offline runners retain last-known ships with unknown status and
disabled mutation controls. Raw runner/container identifiers are not in this view.

### Setup pane

- Add a prepared runner through SSH or a local Unix socket.
- Manage runner display names, location labels and browser hosts. These are
  desktop preferences keyed by runner identity in `$XDG_STATE_HOME/omabit/panel.ini`.
- Register fake ships under available operator runners, then boot explicitly.
- Stage supported HTTP-port changes using the selected configuration revision.
- Manage independent application enrollments, masked sign-in, sessions and the
  existing theme controls in their own section.

Browser hosts use `http://host` or `https://host` without a port or path. The observed
ship endpoint's port is appended, so a staged but unapplied port does not silently
reroute a browser link. Remote loopback endpoints require an explicit browser host;
SSH aliases are not assumed to be browser-resolvable addresses. This UI does not
establish a tunnel or configure a proxy.

### Monitor & maintenance window

A normal desktop window has fleet navigation, overview, maintenance and unified
activity views. It follows the window manager's layout and has an expand control
(also F11) for full-screen viewing. It can be resized, closed and reopened.

- RAM/limit bars and session-only sample trends, task counts, source freshness.
  Samples are deduplicated by daemon timestamp, capped at 120 per ship / 256 ships,
  split at missing observations and reset when the container identity changes.
- An epoch timeline distinguishing a measured inventory from historical retained
  names in the last completed pruning report. It does not invent current sizes or
  backup integrity.
- Runtime/network configuration, last maintenance/pruning results, and expandable
  technical detail for source timestamps, container and runner IDs.
- Manual pack/meld/roll, inspection and pruning review retain the existing immutable
  selections, exact evidence checks, confirmation and no-automatic-retry behavior.
- The activity feed combines operation and incident journals across runners, with
  runner, ship and event-type filters, plus expandable request/actor/operation IDs.

## Backend decisions intentionally deferred

| Requested feature | Current UI behavior | Decision / backend work |
| --- | --- | --- |
| Copy `+code` | Opens a clear availability explanation; copies nothing | Authenticated retrieval, authorization, redaction and clipboard lifetime |
| Change runner SSH/socket/name binding | Current enrollment stays pinned; display preferences are editable | Identity-preserving atomic collector connection update |
| Import an existing pier | Provisioning choice explains unavailable support | Ownership/layout validation, import/adoption and recovery contract |
| Boot from key | Provisioning choice accepts no key yet | Secret intake and lifecycle, identity checks, qualified production bootstrap |
| Boot a comet | Provisioning choice explains unavailable support | Runtime bootstrap/resource qualification |
| Publish or change Ames networking | Shows configured port and current publication limitation | Networking exposure and qualification |
| Routine maintenance | Dedicated policy-design view; no effective schedule is saved | Window/timezone, missed jobs, resource admission, retention, failure/resume policy |
| Loom utilization and resize advice | Container RAM is explicitly labeled as different | Separately qualified loom telemetry |

## Validation and use

```bash
node /home/thelifeandtimes/repos/omabit/scripts/test-panel-model.js
python3 -B /home/thelifeandtimes/repos/omabit/scripts/test-panel.py
python3 -B /home/thelifeandtimes/repos/omabit/scripts/test-panel.py --windows
python3 -B /home/thelifeandtimes/repos/omabit/scripts/test-panel-install.py
```

The QML fixture uses isolated state and CLI helpers. It exercises maintenance and
pruning buttons, changed identity/cutoff invalidation, persisted runner presentation,
browser host composition, no malformed-ack retry, and stdin-only application login.
Model checks include filters, offline state, remote loopback refusal, stale metrics,
container replacement and epoch uint64 precision.

Update the owned snapshot, then force a fresh shell process because hot rescans
have demonstrably retained old QML on this workstation:

```bash
python3 -B /home/thelifeandtimes/repos/omabit/scripts/install-panel.py --replace --enable
omarchy restart shell
```

View-only shortcuts:

```bash
omarchy shell thelifeandtimes.omabit open
omarchy shell thelifeandtimes.omabit.setup open
omarchy shell thelifeandtimes.omabit.monitor open
omarchy shell thelifeandtimes.omabit.monitor activity
omarchy shell thelifeandtimes.omabit.monitor maintenance
```

The optional `--windows` check briefly maps the isolated fixture window and checks
overview, maintenance and activity rendering, including the chart canvas.

Visual review should cover the fleet row density, setup flow, windowed/full-screen
window, charts, scrolling and activity filters. Existing backend capabilities were
reused; this checkpoint is not new runtime, key, networking or scheduling qualification.
