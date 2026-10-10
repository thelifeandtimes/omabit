# Runtime policy: official releases, reliable ships

Omabit uses the published Vere runtime in the pinned `tloncorp/vere` image. The
current version is `4.6-5a7614f`; image and pill digests are defined in
`internal/runtimecandidate/metadata.go`. Updates to that release pin are deliberate.

## Keep the running experience simple

The supported development surface is ship creation/resume, start/stop, HTTP identity
and health, Docker resource monitoring, pack/meld/roll, confirmed epoch pruning,
UTC maintenance schedules, qualified fake import, SSH access and Eyre sessions.
All use the released runtime. Existing managed instances already use this binary.

Vere's control socket stays disabled with `--no-conn`. The tested 4.6 disconnect
fault is avoided by not using that interface. Live loom-internal counters,
utilization percentages and pressure-based maintenance advice are deferred. The UI
continues to show container RAM, limits, tasks and measurement freshness; configured
loom size remains configuration, not a measured utilization denominator.

The experimental patched-runtime build, executable overlays, socket client,
qualification commands, custom upgrade plan and report-review exceptions have been
removed. There is no custom runtime catalogue, build pipeline or adoption API.

## Periodic snapshots

Omabit's default is **60 minutes**, passed as `--snap-time 60` to long-running Vere
boot/resume processes. Vere 4.6's main CLI accepts minutes and converts them to
seconds internally; its omitted-flag default is 120 seconds. Offline utilities
(`pack`, `meld`, `roll`, `play`) keep their own parsers and command contracts.

Each container intent records its snapshot argument before creation. Containment
checks require that exact command. An existing process launched without the option
remains verifiable as a two-minute runtime until an explicit stop/start replaces
it. A manager update alone does not alter running process arguments. Runtime status
reports both the recorded container interval and the current new-start default.

This changes periodic checkpoints, not event-log durability. Normal event logging
continues, and explicit maintenance/shutdown may also write state. An unclean exit
can require replaying a longer event-log interval. No drive-lifetime multiplier is
inferred solely from the lower checkpoint frequency.

## Future channels, only when needed

If multiple runtime choices become useful, follow the simple Groundseg-style model:

- **`live`**: the latest upstream release; the normal/default choice.
- **`edge`**: an upstream `urbit/vere` `develop` build; explicit opt-in for development.

A channel should resolve to a concrete artifact/version for reporting and
repeatable execution. Do not grow a collection of local patches or arbitrary
executable paths. Channel selection is not implemented now; the current release
pin remains the sole runtime.

## Development data

This project has not shipped a production version. Disposable qualification reports
are development artifacts, not a backwards-compatible API. Removed experiment
profiles and older report layouts are unsupported; there are no conversion or
ignore-failure shims. Standalone qualification still refuses unknown/unresolved
reports rather than interpreting them as success. Old scratch reports may need
manual cleanup before running new disposable qualifications.

Managed storage uses the permanent mount-root/pier layout; source version 0.22
adds schema-12 handoff fencing through an additive upgrade from schema 11. Snapshot
interval policy updates preserve the piers and operation history; the recorded
per-container launch interval distinguishes existing and replacement processes.
Source version 0.23 adds schema-13 bootstrap-selection metadata, preserving all
older runtime records. New registrations can choose canonical bare bootstrap or
a supplied Recommended Apps pill; initialized starts never reapply a pill.
Management does not run the standalone prior-qualification gate. No cleanup command
automatically removes piers, qualifications or installed system artifacts.

## Management service

The native build is `build/urbit-runner`, installed at
`/usr/local/lib/urbit-runner/current/urbit-runner`. The dedicated-account management
service and dedicated SSH service replace foreground terminal sessions. The selected
startup policy resumes initialized, previously running ships only after verified
clean shutdown, with one journaled attempt per host boot. See [services.md](services.md)
for installation, controlled restart, reboot verification and operational commands.

Ships continue running across a management-only restart. The existing SSH bridge
and panel use the same management socket, runner identity and operation journal.
Service handoff, management-only restart, controlled engine restart and a full clean
host reboot have passed live checks on the current development fleet.

## Next priorities

1. Clear lifecycle/error recovery and predictable scheduled maintenance on the
   released runtime, including resource-exhaustion and interruption behavior.
2. A straightforward real-ship setup/import path with explicit identity handling.
3. Useful fleet health and actionable UI feedback using the metrics already available.

Peer systems, gateways, social features, richer telemetry and runtime-channel
selection follow a reliable core ship-running experience; they are not prerequisites
for finishing it.
