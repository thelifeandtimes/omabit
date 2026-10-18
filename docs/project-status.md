# Project status

## Where we are

**[RC.11 is published](https://github.com/thelifeandtimes/omabit/releases/tag/v0.1.0-rc.11)**
and simplifies boot forms: loom31 and default storage are under
collapsed Advanced settings, new loom choices are30–34, native control is CLI-only,
and owned-key boot links to Get an Urbit ID. Existing runtime settings and CLI
native choices survive in-place updates; historical smaller looms remain manageable.
The final runner was updated in place on sourceport, preserving the user's healthy
`bare-comet`, its container and king/worker processes, settings and history.
The cold-loaded desktop now reports RC.11 with the original bar position and panel
preferences preserved. See the [in-place update record](diagnostics/2026-10-18-rc11-in-place-update.md).

**[RC.10 is published](https://github.com/thelifeandtimes/omabit/releases/tag/v0.1.0-rc.10).**
It integrates [PR #4](diagnostics/pr4-integration.md): loom20–34 and default-off
native control, with immutable launch settings, legacy-client preservation,
signed settings-aware host moves and schema19 downgrade protection. The frozen
binaries passed real fake/comet native file threads and two in-place updates from
RC.9, preserving the running container, stopped intent, settings, receipts and SSH
identities. See [qualification](diagnostics/2026-10-18-rc10-qualification.md).
Earlier on October18, sourceport was prepared for the user's fresh RC.10 installation:
the qualification installation was retired, all three stopped piers retained with
unchanged fingerprints, and the cold shell/empty setup probe verified. See the
[local handoff](diagnostics/2026-10-18-rc10-local-reset.md).

RC.9 integrates [PRs #2/#3](diagnostics/pr2-pr3-integration.md): Ames
publication for new live imports, universal pre-move epoch checks, and explicit
non-destructive enrollment recovery adapted to named/guided/in-place imports.
Stable 0.1.0 is gated on the [upgrade contract](upgrade-policy.md); fresh resets
are development exercises, not the supported user update path.

**[RC.9 is published](https://github.com/thelifeandtimes/omabit/releases/tag/v0.1.0-rc.9), with matching native binary pins.**
The local sourceport install was retired for the user's final fresh-setup check.
Its test comet was cleanly stopped and preserved; the cold-reloaded desktop and
isolated RC.9 setup probe report an empty collector, no installed host and ready
prerequisites. See the [local RC.9 handoff](diagnostics/2026-10-17-rc9-local-reset.md).
RC.8 simplifies setup approval and machine selection, adds source/location-only
guided imports with progress, recognizes ship-named pier leaves, and accepts intact
historical native-pier epochs and self-contained docked binaries for import.
Sourcebox is reset for the next user-driven installation. The relocated protected
`~lismyl-lismyl` pier's tree fingerprint is unchanged; twenty unrelated pier
directories and ten Urbit processes were preserved. See the [RC.8 handoff](diagnostics/2026-10-16-rc8-sourcebox-reset.md).
RC.7 removes per-refresh log-viewer SSH/PAM/logind sessions with a persistent
identity-pinned watcher, cancellation and bounded reconnect backoff. Its frozen
collector delivered 23 updates over 66 seconds through one actual sourcebox SSH
process and reaped it on cancellation. See the [D-Bus follow-up](diagnostics/2026-10-16-dbus-log-watch.md).
RC.6 fixed the effective Recommended Apps default found during RC.5 live deployment;
the sourcebox fresh-install UI confirmed the supplied S3 default was active.
The latest feedback adds software/source explanations, stable inline port/log
progress, public comet identities, HTTP fleet labels and a Coming Soon application
connections stub. Recommended Apps now uses the supplied S3 pill; its frozen copy
passed production download and a disposable fake boot/healthy/stop check. See
[RC.5 qualification](diagnostics/2026-10-15-rc5-feedback.md).

RC.4 established the first onboarding improvements below.
It incorporates the user's first sourcebox onboarding feedback: welcome/setup,
machine-use preference, automatic paths and first boot, real host-port suggestions,
healthy-only access and fixed full-window activation. The sourcebox test installation
was retired for another fresh pass, preserving all ten unrelated Urbit processes.
See the [feedback/reset record](diagnostics/2026-10-15-onboarding-feedback.md).

RC.3 established the owner-user runtime described below.
It runs the runner/rootless engine as the existing user, with `/srv/urbit/piers/`
and ordinary custom workspace paths. Default fake and custom-root comet desk
copy/edit/commit/remount gates passed locally, plus pack/resume content readback.
Frozen binaries also passed management-update preservation, controlled owner-engine
restart/resume and ordinary-file import/start/delete/revision-safe name reuse.
See [owner-user runtime](owner-user-runtime.md) and [RC.3 notes](release-rc3.md).

RC.2 is published; the old sourceport/sourcebox dedicated installations are retired.
The verified binary bundle is staged on sourcebox. The selected repository model is
[one directly installable `/omabit` monorepo](native-plugin.md), with generic Go
tools under `backend/cmd/` and shared implementation under `internal/`.
The user will use the normal Omarchy plugin install command and host setup manually. See the
[publication/retirement handoff](diagnostics/2026-10-15-rc2-release-retirement.md).
Sourceport's earlier disposable owner-user qualification resources are archived. Earlier
on October 16, sourcebox completed the user's RC.6 fresh installation. Its `guac-demo` comet was gracefully
stopped and its Omabit installation retired on October 16 for a user-driven RC.7
pass. Data and all ten unrelated Urbit processes were preserved; a cold shell
reload verified fresh-install readiness. See the [RC.7 reset receipt](diagnostics/2026-10-16-rc7-sourcebox-reset.md).
The older running-fleet results below are historical acceptance evidence.

Release candidate source **0.1.0-rc.2 / schema 15** adds alpha-requested storage roots,
GUI host-path move/import and confirmed pier deletion, with a dedicated SVG bar
mark. Default storage and the parent-mount contract remain intact. Name reuse after
deletion advances configuration revisions to fence stale requests. See
[storage management](storage-management.md) for permission requirements and the
scope of fixture/QML verification. These changes were
[deployed to the desktop and both runners on October 14](diagnostics/2026-10-14-development-update.md),
preserving all 12 registrations and seven running containers at that deployment.
RC.2 release packaging now includes these changes; see [release notes](release-rc2.md)
and the [user-driven sourcebox installation](sourcebox-clean-install.md).

Schema 15 additionally implements durable **resume after an unclean host reboot**:
last-healthy boot/container/revision evidence, pending authorization surviving
management restart, and one journaled attempt per runtime generation and host boot.
Same-boot failures and interrupted host operations retain their separate handling.
This is [deployed on both hosts and bundled in the desktop](diagnostics/2026-10-15-reboot-resume-deployment.md).
The three remote hard-reset failures were resumed and verified healthy; all seven
running ships now have boot-bound eligibility records. See [services](services.md)
for the exact policy and migration behavior.

A subsequent [real sourcebox hard reset passed automatic resume](diagnostics/2026-10-15-hard-reboot-acceptance.md)
for all three running remote ships without operator intervention. The proposed next
phase is [RC.2 publication, fresh installation and multi-collector qualification](rc2-next-phase.md).

The project has a working, live-tested **development management stack**. Current
work is in **phase 6 of the ten-phase implementation sequence: maintenance and
epoch retention**. The immediate priority is reliable ship operation with official
released runtimes; see [runtime-policy.md](runtime-policy.md). Phase numbers indicate
order, not percentage of total effort.

| Area | Current state |
| --- | --- |
| Contracts and simulator | Accepted protocol, revision/conflict handling, idempotency, and multi-client fixtures |
| Runtime foundation | Owner-user rootless Docker, default/custom-root fake/comet desk editing and lifecycle accepted locally; broader production qualification remains |
| Persistent runner | Durable SQLite registrations, operations, attribution, reconciliation, resource admission, and development lifecycle accepted |
| Collector and access | Named identity-pinned connections, SSH grants/revocation, inventory isolation, origin-bound Eyre sessions and subscriptions accepted in exercised development gates |
| Omabit UI | Published RC.11 native plugin: compact boot defaults, advanced loom/storage, CLI-only native control, retained import recovery, guided/named imports and persistent logs |
| Maintenance and retention | Official-runtime fake/comet pack/meld/roll, inventory/previews, prefix pruning and UTC schedules accepted in exercised live gates; interruption/recovery hardening remains; loom-internal telemetry deferred |
| Peer monitoring | Planned; current single-runner health/incident basics do not yet provide the phase-7 multi-host peer system |
| Personal operator gateway | Planned |
| Social utilities / Omarchy gossip | Planned; architecture and delivery/tag policy recorded |
| Agents and release distribution | RC.11 assets and supplied pill published; RC.9→10 and RC.10→11 management preservation verified; full stable-upgrader, application/OTA and agent/MCP qualification remain |

## Current milestone

6H passed three managed roll/replay/resume cycles on alpha. At that point it contained
`0i0`, `0i22185`, `0i22199`, and `0i22213`; alpha finished running/healthy at
configuration/applied revision 2/2. Its oldest-only preview retains three epochs.

6I passed confirmed managed genesis pruning: alpha's `0i0` was removed, its three
newer epochs were unchanged, and replay plus healthy resume passed. Alpha then
retained `0i22185`, `0i22199`, and `0i22213`. 6J passed snapshot-bearing and repeated
prefix deletion/recovery on a fresh disposable pier, using a five-to-three-to-two
sequence. 6K passed that sequence on managed alpha after two explicit preparation
rolls, including repeated pruning and preserved historical replay. Alpha resumed
running/healthy at 2/2 with `0i33536` and `0i33561` retained.

6L exposes the accepted workflows in the management panel: source/freshness-aware
container resources, maintenance results and confirmation, epoch inspection,
protected-prefix selection, exact collector preview and explicit history-loss
confirmation. Model and isolated QML checks cover changed/stale selections and
non-retry behavior. Version 0.3.0 was installed and enabled. The next essential user
gate was visual review of the updated panel. A stale-QML reload issue was resolved
with a shell restart; the user accepted the data coverage and requested a visual
redesign. [Checkpoint 6M](checkpoints/23-fleet-desktop.md) now organizes the unified
plugin into fleet, setup and monitoring surfaces, with charts and a filterable
activity feed. Its backend-gap table records the next refinement decisions.

The user accepted that design and authorized backend implementation. **6N** adds
identity-preserving enrollment edits and operator-only login-code access with a
30-second sensitive clipboard handoff. The upgraded runner passed live code
retrieval, temporary Eyre authentication, logout/revocation and secret-free audit
checks. Real desktop clipboard expiry remains a follow-up.

**6O** now implements a bounded stopped-pier copier and a fresh-disposable
copy/resume/replacement qualification workflow. Its
[live gate](checkpoints/25-stopped-pier-copy.md) passed for run
`185ff2ed58b9abdd6861dfe64d6650f5`: 7 files / 1,016,840,204 logical bytes,
two target resumes, original unchanged, all four containers removed.

**6P / panel 0.6** integrates that source into durable managed registration,
source/port reservation, progress, exact replay and explicit initialized-pier
start. The [live gate](checkpoints/26-managed-qualified-import.md) passed: import-check
is running/healthy at 1/1 on port 18086 after verified copy, replay, duplicate-source
refusal and a stop/resume cycle.

**6Q** implements new disposable-comet bootstrap and exact-identity replacement
qualification. [Live qualification](checkpoints/27-fresh-comet.md) passed for
`cb31b1d255ffe45103cf90ed7458a593`, retaining the same comet identity and removing
all containers/network.

**6R / panel 0.7** integrates durable managed comet registration and identity-pinned
lifecycle behind an explicitly accepted mixed development mode. The
[managed comet live gate](checkpoints/28-managed-comets.md) passed: comet-check is
running/healthy at 1/1 on port 18087, with a stable generated identity after resume.

**6S / panel 0.8** adds durable UTC pack/meld policies for qualified fake piers, with
explicit downtime/resume settings and no missed-slot catch-up or interrupted-job
continuation. The [timed live gate](checkpoints/29-routine-maintenance.md) passed:
one stop/pack/resume cycle restored healthy import-check and did not repeat. The
test policy is disabled at revision 2.

**6T** passed fresh-comet pack/meld qualification with isolated utilities, measured
logical sizes and exact-identity restarts. Run `5418c87ccd3aac4ef8a1597adedeae0c`
removed all seven containers and the bridge; see its
[accepted evidence](diagnostics/2026-09-29-comet-maintenance.md).

**6U / panel 0.9** integrates managed comet pack/meld and UTC routines, with pinned
identity, uncertain-outcome guards and private schema 5. The
[managed live gate](checkpoints/31-managed-comet-compaction.md) passed: manual
pack/replay/resume and one timed meld/resume restored the same comet identity and
healthy 1/1 state. The test policy is disabled at revision 2, and panel 0.9 is loaded.
External imports, keys, host Ames, comet retention and richer policy remain.

**6V** passed fresh-comet epoch lifecycle qualification: three isolated rolls,
snapshot-bearing inventories and exact-identity restarts, followed by one final
replacement and preview-only selection retaining three of four epochs. The
[standalone live gate](checkpoints/32-comet-epochs.md) passed for
`e84ffd83afcfbbe71e6cceca2d69c30f`, retaining `0i0`, `0i78`, `0i178`, `0i287` and
removing all nine containers plus the bridge.

**6W / panel 0.10** integrates managed comet inspection, explicit roll and
preview-only retention behind `comet-epochs`, with schema-6 downgrade protection.
The [managed live gate](checkpoints/33-managed-comet-epochs.md) passed: comet-check
retained `0i0`, `0i7046`, `0i7064`, `0i7082` through three identity-preserving cycles
and final inspection/resume. Confirmed deletion was refused, historical replay
preserved cache invalidation, and disabled policies were unchanged. Comet
deletion/recovery remain separately gated.

**6X** passed fresh-disposable comet repeated-prefix deletion and latest-epoch
recovery: five → three → two epochs, controlled current-checkpoint loss and replay
after each deletion, exact-identity restarts and a final replacement. The
[standalone live gate](checkpoints/34-comet-prefix-recovery.md) passed for
`198fec0f0e49df2bcb0215789f2b5f1d`, retaining protected epochs `0i557` and `0i592`;
both recoveries restored checkpoints and all 17 containers plus the bridge were
removed with resolved outcomes.

**6Y / panel 0.11** integrates confirmed managed comet prefix pruning behind its
own capability, with exact evidence binding, retained metadata checks, independent
partial-deletion quarantine and schema-7 compatibility protection. Its
[managed live gate](checkpoints/35-managed-comet-pruning.md) passed: preparation
added `0i7847`, then confirmed cutoffs `0i7046` and `0i7064` retained `0i7082` and
`0i7847`. Immediate metadata, protected-cutoff refusal, historical replay and
same-identity resumes passed; comet-check ended healthy at 1/1 with policies unchanged.
Panel 0.11 is installed and its loaded version is verified.

## Official-release focus

The exploratory custom-runtime/control-socket branch has been removed: client,
patched executable overlays, test commands, build recipe and rejection-review
exceptions. There is no compatibility layer for those disposable reports.

The sole runtime is the pinned official Vere 4.6 release. Native control stays off
by default and is configured only through the CLI. Source offers a
[loom30–34 dropdown under Advanced settings](development-runtime.md). The earlier
managed-fake native/Clay gate and final RC.10 fake/comet native checks passed,
alongside repeated in-place management updates. See the linked qualification
records for the exact tested boundaries.
Existing lifecycle, maintenance, pruning, schedules, import, access and
container-monitoring features remain. Live loom-internal metrics and pressure-based
advice are deferred.

Any future runtime choice should follow a simple `live` (upstream release) / `edge`
(upstream develop) model. It is not being implemented now. Work is focused on reliable
released-runtime operation, lifecycle recovery and real-ship setup.

Dedicated management and SSH units, ordered shutdown budgets and opt-in journaled
startup resume are now implemented. The selected policy resumes only initialized,
previously running, cleanly stopped ships once per host boot; interrupted work and
intentional stops are skipped. Service installation and the foreground handoff were
verified live September 30: supervised management, enabled/active SSH, all four
initialized ships healthy with unchanged container IDs and no startup operations.
The management-only service restart also passed: a new runner process preserved
all ship container IDs and health without issuing startup-resume jobs. The controlled
engine/management restart then passed: four automatic `startup-resume` operations
succeeded, each replacing a container while preserving identity and configuration.
The fleet returned healthy with no recovery requirements; `ssh-check` stayed
uninitialized and schedules stayed disabled. A full host reboot then passed: the
lingering account started Docker and management, SSH started automatically, and four
new once-per-boot resume jobs restored the same healthy fleet in about 24 seconds
from the first queued job. Runner/ship identities and configuration revisions were
preserved, with no active work, recovery requirements or incidents. The startup/reboot
milestone is complete for this development fleet; see [services.md](services.md).

## What remains before a supported release

**0.1.0-rc.1:** versioned Linux/AMD64 binary packaging and a configured Recommended
Apps release URL consolidate the development checkpoints through 0.23. The frozen
draft pill passed fresh-comet first boot, 60-second health observation, clean stop,
same-identity initialized resume and final stop using the exact packaged binaries.
The sourcebox runner is now rc.1; the local desktop/runner remain 0.22.3. Grubbery
compile failures and Azimuth snapshot/HTTP failures were observed; full app/OTA
acceptance is incomplete. `%mcp` and draft `%omachan` await the next supplied pill.
See the [qualification record](diagnostics/2026-10-10-rc1-pill.md) and
[prerelease notes](release-rc1.md).

The Tend repository migration is complete, and the new public runner repository is
<https://github.com/thelifeandtimes/omabit>. Reviewed source history is published
through the `github` remote; the original self-hosted `origin` is retained. See
[repository.md](repository.md) for command ownership and the cutover boundaries.
The supplied pill's bytes, hash and scoped lifecycle qualification are recorded.
This supports an explicitly limited prerelease, not final 0.1.0 app/OTA acceptance.

**0.23 source** implements the two-option first-boot selector: **bare bootstrap**
uses Vere's canonical default, while **Recommended Apps** uses a supplied release
pill URL. This covers comets, owned-key ships and development fakes, with immutable
content-bound registration and initialized-pier resume. Schema 13 preserves older
runtime records and prevents downgrade ignoring the new selection. The user
supplied `omabit-brass.pill` (57,011,409 bytes); its measured digest, reported desk
inventory and rc.1 qualification are recorded in `pill-plan.json`. `%mcp` and draft
`%omachan` await a separate replacement artifact. The automated pill builder is explicitly
deferred to 0.2.0. See [pill-release.md](pill-release.md).

**0.22.3 is deployed.** Inventory watches now reuse a pinned transport per runner
instead of creating a fresh SSH/PAM/logind session for every two-second poll.
Sourcebox measurements fell from 421 runner sessions in 15 minutes to 6 during a
15-minute window including update/diagnostic activity. All managed fleet states,
containers, revisions, policies and move fences were preserved. The reported
backed-up D-Bus sockets belonged to separately launched native Urbit processes under
the desktop account, not Omabit's managed containers. The user managed those ships
and rebooted sourcebox manually. Post-reboot verification passed: both previously
running managed ships resumed, `panel-check` stayed stopped, the user confirmed
normal Brave operation, and broker resource errors were absent. Some nonempty bus
queues were stable over the measured minute; longer soak checks remain. See the
[incident record](diagnostics/2026-10-10-ssh-polling-dbus.md) and
[reboot acceptance](diagnostics/2026-10-10-sourcebox-reboot.md).

The next live gate also passed: discarded caller acknowledgment, observer disconnect
during remote fake Pack, running/completed request replay with no duplicate job,
and healthy initialized resume. See the
[caller-loss record](diagnostics/2026-10-10-remote-client-interruption.md).
Abrupt runner/utility interruption and actual network partitions remain unqualified.

**0.22.2, included in the deployed 0.22.3 builds,** raises Omabit's boot-pill provisioning limit to 4 GB
(4,000,000,000 bytes), including the previously smaller pinned-candidate downloader.
Installed artifacts on both hosts were verified. The
[confirmed pill plan](pill-release.md) now names the 12 required desks and their
publishers. `%pals` stays on `~tocwex` until the future `~paldev` switch is explicitly
made. `%omachan`, intended for `~dister-dozzod-sarlev`, is deferred and does not
block 0.1.0. The pill supplies the initial bundle and retains normal publisher OTA
after boot; recorded revisions are build provenance. Pill construction and combined
application acceptance of the user-provided artifact remain pending; automated
composition/build tooling is deferred to 0.2.0.

**0.22.1, deployed on both runners and desktop**, removes the registration-count cap for all registration/enrollment
and incoming handoff paths, including collector/UI checks. Stopped-pier copy and
handoff bounds are now **100 GB (100,000,000,000 logical bytes)** plus the existing
32 MiB stream-format allowance. Sparse metadata tests cover the boundary without
claiming a full-size live transfer. Installed binary hashes and before/after fleet
comparison passed, preserving running containers, intentional stops, uninitialized
registrations and moved-source fencing. See the
[deployment record](diagnostics/2026-10-09-registration-transfer-limits.md).
Release-pill work and restart/reboot/interruption
qualification are the next planned work.

**0.21** defaults long-running Vere processes to `--snap-time 60`. Tests and an
approved rolling replacement passed on all five running local/remote ships, with
identities, state and inactive registrations preserved. Runtime status distinguishes
the recorded process interval from the new-launch default. See the
[snapshot-policy record](diagnostics/2026-10-08-snapshot-interval.md).

The sourcebox remote trial has passed installation, pinned management over Tailscale,
fake boot/resume, HTTP/+code login, logs/metrics and management-only restart. Full
remote host reboot has now passed on the permanent layout. A fresh remote comet and pack/meld/roll/
inspection/resume on both remote test ships have now passed; see the
[remote maintenance record](diagnostics/2026-10-09-remote-maintenance.md).

**0.22, deployed on both runners and desktop**, implements actual host-port admission and safe launch-error reporting,
plus a [stopped cross-host handoff](host-handoff.md) with durable source/destination
fences, streamed verified contents, signed source commitment and explicit continuation
after interruption. A real 1.05 GB move of stopped `panel-check` to sourcebox passed,
including source Start refusal and destination Start/Stop. Live external-port conflict
refusal and temporary comet Eyre authentication over the tailnet also passed; see
[the 0.22 acceptance record](diagnostics/2026-10-09-stopped-host-handoff.md).
The UI does not offer historical pier backup/restore. Further interruption,
owned-ship movement and longer failure/soak qualification remain live gates.

The broader remote acceptance plan is in
[remote runner assessment and plan](remote-runner-acceptance.md). The earlier
icon/browser-button refinements remain covered by mapped-window/model/installer
checks; additional remote failure, recovery and deployment testing is still needed.

**0.20** removes staging/promotion from first boot. Every managed ship uses one
exclusive permanent mount root and a `pier/` child; the same container keeps running
after identity readiness. All six initialized local piers and eight registrations
were migrated offline to schema 11, with stopped backups and content/epoch/identity
checks. Four running ships were restored healthy; two stopped and two uninitialized
registrations retain those states. Old-layout execution, staging-lock recovery and
incremental schema compatibility paths are removed. A fresh fake passed live
direct-boot and explicit stop/resume acceptance. See the
[permanent-pier migration record](diagnostics/2026-10-02-permanent-piers.md).

Panel/runner 0.19 adds runtime diagnostics and operator-only bounded stdout/stderr
logs. Panel 0.19.1 fixes the live-update crash through cold replacement and owned
deferrals. The first real moon reached HTTP identity readiness but was force-killed
during automatic storage finalization; the updated driver prevents that escalation
and handles the stopped initialized staging pier and its stale lock. The same moon
is now running/healthy at 1/1 in its canonical directory. See the
[recovery/logs record](diagnostics/2026-10-02-moon-recovery-and-logs.md). Broader
application/network qualification and clean-host acceptance remain.

Panel 0.18.2 adds Start beside owned-ship registration, enabled only for the
matching registered, startable ship. State-transition, identity-binding and
single-dispatch fixtures pass, and the installed panel version is verified. The
user's moon is registered but still awaits first boot; see the
[registration Start record](diagnostics/2026-10-02-registration-start.md).

Panel 0.18.1 fixes the native GTK/GVFS Browse crash by selecting Qt's non-native
key/pill dialogs. Real mapped-dialog open/cancel/reopen/selection regression checks
pass, and the installed/loaded patch version and fleet state are verified. See the
[crash diagnosis](diagnostics/2026-10-02-browse-dialog-crash.md).

Panel 0.18 and its host runner are deployed. Add a ship unifies comet, owned-key and
existing-pier setup; Manage runners includes offline collector connections, while
HTTP settings live on the selected ship. The user's confirmed moon name passes
protocol, CLI and store fixtures, and transposed syllables produce a precise error.
Full checks, visual review, installed artifact hashes and live fleet preservation
passed. At that checkpoint the user's real key had not been read or submitted. See the
[unified provisioning record](diagnostics/2026-10-02-unified-provisioning.md).

Panel 0.17 introduced the enrolled-pier visibility/no-op fix, inline source
reviews, shared menu/full-panel +code controls, circled tilde icon and a two-input
key form. The runner allocates HTTP/Ames ports atomically and supports canonical or
hash-bound custom pills; an inert upload passed the real SSH transport. Existing
fleet states and enrollment were preserved. See the
[refinement record](diagnostics/2026-10-02-key-setup-refinement.md). Real owned-key
bootstrap, inbound Ames behavior and a clean-host installation remain acceptance work.

Panel 0.16 implements the user's [pier convention](pier-storage.md). All five
initialized development piers were moved with unchanged epoch metadata; running
ships resumed healthy and intentional stops/uninitialized registrations were
preserved. External rename detection, explicit reconnection, unenrolled discovery
and archive exclusion passed live checks. Key-file bootstrap and in-tree owned
livenet enrollment are implemented and fixture-tested; first supplied real key/pier
acceptance is pending. See the [storage record](diagnostics/2026-10-01-conventional-storage.md).

Panel 0.15 and its runner are deployed. CPU percentages derive from paired private
Docker counters, with genuine zero and unknown/reset semantics tested. Live readings
passed on alpha, bus-lab and comet-check; containers, intentional stops and policy
revisions were preserved. Layout, routine-default and chart-axis checks passed; see
[the 0.15 record](diagnostics/2026-10-01-cpu-and-layout.md). The later storage and
real-ship implementation is described above; fresh-host acceptance remains.

Panel 0.14 implements the user's fleet/activity/policy/history refinements, themed
tooltips, wrapped comet names, full-window Setup and prominent +code access.
Journal-backed roll dates are read-only display metadata; inspection completion
now opens and reveals the bound inventory. Automated checks, screenshot review and
live deployment passed. Version 0.14.0 was loaded, historical roll dates were verified,
the user's running/stopped fleet and disabled policies are preserved, and live +code
access passed without exposing the secret. See the
[refinement record](diagnostics/2026-10-01-desktop-refinement.md).

Panel 0.13 fixes ambiguous fake/Ames labels, checkbox placement, duplicate-registration
validation and asynchronous form feedback. New fake registrations can independently
choose the Urbit identity, local instance name and private pier-directory basename
behind `fake-ship-options` (private schema 8). Existing test and user registrations
are preserved. Backend and QML fixtures cover the new identity/path contract and
failure handling. The updated runner and panel 0.13 were deployed and the loaded UI
version verified before the 0.14 refinement. The [named-fake live gate](diagnostics/2026-10-01-named-fake.md)
passed: `bus-lab` booted as `~bus` in `bus-data`, registration replay returned the
same operation, and stopped epoch inspection plus initialized replacement/resume
preserved the identity. It is healthy at 1/1 on loopback HTTP 18088. All four prior
running containers remain unchanged; `rus` and `ssh-check` remain uninitialized.

The local first-install flow is implemented in `omabit setup` and panel 0.12:
prerequisite checks, private desktop key, verified host-helper download, OS-authorized
account/engine/access/service setup, identity-pinned enrollment and a guided first-ship
handoff. The host helper uses the canonical `current/urbit-runner` shell path and
preserves existing keys, grants, piers and runner identity on validated reruns.
See [install.md](install.md). Fresh-install/partial-rerun fixtures and isolated QML
progress tests pass, as do full Go/vet and targeted race checks. Panel 0.12 was
installed and exercised before the 0.13 update. Live setup reused `workstation` and the existing runner ID,
updating that enrollment to revision 3 with the app-owned `omabit-local` SSH
configuration. A subsequent panel registration created the stopped, uninitialized
`rus` instance. The four previously running ships remain healthy with unchanged
containers. No fresh VM image is currently available, so real clean-host acceptance
is still separate from these existing-host results and isolated installer fixtures.

- Harden the existing maintenance/retention surface: interruption/recovery,
  resource failures and clear operational feedback.
- Close runtime/deployment gates: remote/tailnet and Ames networking,
  resource/failure behavior, real-key/bootstrap workflows,
  supported artifact upgrades, and operational recovery.
- Complete deferred desktop migration/refinement when the user resumes that work.
- Package the reliable core. Peer monitoring, gateways, social/gossip utilities and
  agent integrations are later work, not blockers for the initial ship-running experience.

We are beyond the initial prototype, with a substantial validated development
foundation. Production hosting and the later product surfaces still represent
significant work. No production-runtime qualification or release-ready percentage
is inferred from the number of passed checkpoints.

See [implementation-plan.md](implementation-plan.md) for the full sequence and
[runtime-qualification.md](runtime-qualification.md) for evidence-specific gates.
