# Implementation plan

Work in independently reviewable commits. On September 24 the user authorized
autonomous progress across checkpoints wherever validation can be performed
programmatically. Record the evidence and distinguish fixture results from live
deployment evidence. Pause only for essential manual intervention, with exact
steps, readable output, expected results, and cleanup. Never use a production
pier or identity for an automated test.

## Progress

- Architecture and sequence: recorded.
- Checkpoint 1 protocol/simulator: implemented; automated tests and user acceptance
  of the [local walkthrough](checkpoints/01-local-protocol.md) passed.
- Protocol contract: [experimental version 1](protocol.md).
- Real Docker/runtime qualification: lifecycle, maintenance, local host HTTP,
  management services and clean host reboot/resume have passed on the development
  fleet; Ames/remote networking and broader failure gates remain.
- Qualification pill: use the selected Vere runtime's upstream default bootstrap
  pill until the Omabit release is available. Verify the expected base, webterm,
  and groups desks after boot; this does not qualify the future Omabit bundle.
- Default deployment is confirmed: dedicated-account rootless Docker, with the
  same intended five-step plugin onboarding. Existing Docker services, contexts,
  and desktop group membership remain unchanged. No automatic privileged fallback.
- Checkpoint 2A rootless setup: accepted by the user. The dedicated account is
  UID/GID 959 on this host, with socket `/run/user/959/urbit-docker.sock` and
  `rootless_verified: true`. Runtime qualification remains false; the UID is not
  a portable configuration default.
- Checkpoint 2B is accepted: run `35aac1083d71188d9782ec64b49ba771` verified
  image/pill pins, live cgroup limits, fake boot, clean shutdown, replacement
  restart, offline pack, and post-pack restart. All five containers were removed.
- Checkpoint 2C is accepted: run `e9f766ae9854e04f35eb9991d39fb9ba` verified
  loopback HTTP on ports 32768/32769 across replacement, then removed all three
  containers and the per-run network with resolved outcomes.
- Checkpoint 3A implements the persistent development runner/collector workflow,
  restricted to fresh fake ships and the qualified image/HTTP/stop behavior.
  SQLite holds registrations, configuration revisions, idempotency, and operation
  history. Unqualified real-key bootstrap, Ames, meld, and control-socket monitoring
  remain disabled rather than delaying the bounded control-plane milestone. The
  [persistent-runner walkthrough](checkpoints/05-persistent-runner.md) has passed
  the exercised live lifecycle gate; developer tests use real SQLite and an
  injected fake runtime.
- The user's 3A snapshot confirms alpha registration, first start, a revision-2
  HTTP-port change, stop, and successful initialized-pier resume. A reported
  two-minute delay was traced to the preflight sleep probe's PID-1 signal handling;
  see the [diagnostic record](diagnostics/2026-09-24-resume-delay.md). Two live
  checks after the fix completed resume in 6.81 s and 6.80 s, with stops in 0.70 s
  and 0.75 s. Stable runner identity and prior history remained after management
  restart; alpha stayed healthy at revision 2 with no incidents. The timing-fix
  gate is complete. The user accepted this bounded workflow and authorized the
  next phase; the snapshot does not establish every failure/qualification scenario.
- Checkpoint 4A implements persistent collector enrollment, named runner selection,
  per-connection identity verification, and independently polled inventory over
  private local sockets. Its [manual gate](checkpoints/06-collector-enrollment.md)
  passed: the user's 04:41 UTC watch observation confirms the enrolled workstation
  and healthy alpha at revision 2. `scripts/validate-collection.py` verified two
  actual simulator processes, independent collector registries, selected routing,
  stale-revision rejection, unavailable-runner isolation, and replacement refusal
  before mutation. `--format text` now provides readable tables, including watches.
  The process-level validation passed on this host; no further manual 4A fixture
  work is required.
- Checkpoint 4B implements OpenSSH fixed-command transport, host verification,
  SSH enrollment, forced-command key attribution, operator/observer grants,
  per-request revocation, and a restricted account shell. Actual OpenSSH fixture
  validation passed with two operators and an observer against real SQLite and an
  injected runtime. The [dedicated-account installation gate](checkpoints/07-ssh-access.md)
  passed on September 24 at 19:00 UTC: live Hello identified the operator key,
  inventory retained healthy alpha at revision 2, and `register-ssh-check` replayed
  operation `op-77c182d25396669c51322c117e114b5a` with SSH-key attribution. The user
  stopped the foreground SSH server afterward. Remote/tailnet reachability remains
  a separate qualification gate.
- Checkpoint 4C implements independent ship enrollment, exact-origin Eyre sessions,
  Secret Service and explicit headless-file credential adapters, authenticated
  scries, SSE acknowledgement/replay, and reconciliation-required channel loss.
  The [ship-session gate](checkpoints/08-ship-sessions.md) passed programmatically
  against the running fake with the SSH listener stopped, using both credential
  adapters and separate CLI processes. Actual desks: base/groups/kids/landscape/
  webterm. Only the test's own HTTP sessions/channels were created and cleaned up.
  The CLI/store tests also retain retry intent for failed credential cleanup.
- Checkpoint 5A implements the initial Omabit management bar panel: independently
  polled inventory, stale/unavailable/observer controls, explicit runtime start/stop,
  identity- and revision-bound configuration edits, fake registration, SSH runner
  enrollment, and separate application-enrollment display. Manifest/model/QML
  fixture and installer checks passed. The user completed all
  [visual/interactive acceptance steps](checkpoints/09-management-panel.md).
  The September 25 live inventory shows alpha stopped, panel-check running/healthy
  on port 18085 at revision 1, and ssh-check uninitialized. Existing theme settings/
  credentials require the explicit 5B migration.
- Checkpoint 5B adds application sign-in/status/logout UI, immutable ship selection,
  generic acknowledged Eyre pokes, an Omabit-specific theme publisher, bounded
  per-ship retries, change-based desktop notifications, and digest-bound legacy
  migration. [Development validation](checkpoints/10-applications-and-themes.md)
  passed on panel-check at port 18085; migration recovery was tested with fixtures.
  The real legacy ~sarlev handoff remains an explicit user decision, not a test.
  Theme testing/refinement is paused at the user's request while runner work proceeds.
- Checkpoint 6A implements explicit durable offline pack for stopped initialized
  development piers, with host admission, before/after logical file-size evidence,
  and persistent interruption quarantine. Driver, journal, and race tests passed.
  [The live gate](checkpoints/11-offline-pack.md) passed on September 25: the active
  runner advertises offline-pack, alpha packed in 2.36 s and resumed in 5.91 s,
  and exact replay returned the original completed operation while alpha remained
  healthy at revision 2/2. Logical pier bytes changed from 1,046,691,852 to
  637,779,980. No incidents or recovery requirement were reported; see the
  [evidence record](diagnostics/2026-09-25-managed-pack.md).
  Scheduling, meld, epoch pruning, and loom advice remain separately gated.
- Checkpoint 6B implements disposable offline-meld qualification with pinned
  utility invocation, a fresh fake pier, pre-utility resource re-admission,
  before/after logical file-size evidence, same-pier restart, and durable
  interruption quarantine. It holds the management-process lock through cleanup
  to exclude concurrent managed start/pack work. Qualification plans/progress/
  results now support `--format text`. [Live qualification](checkpoints/12-offline-meld.md)
  passed on September 25: run `9b5dfcb57a677f7cf1a2b833e6f65274` melded in 17.32 s,
  restarted the same pier, and removed all four containers with resolved outcomes.
  Logical bytes changed from 1,016,840,204 to 571,572,236. The restored management
  inventory was independently checked; see the
  [acceptance record](diagnostics/2026-09-25-offline-meld.md).
- Checkpoint 6C implements explicit managed meld with `offline-meld` capability
  negotiation, durable actor-scoped replay, stopped/initialized-pier admission,
  shared start/pack/meld serialization, and interruption quarantine. Pack and
  meld now re-admit immediately before utility dispatch and expose the admission
  sample. Durable retained-utility checks also prevent later expensive work after
  a failed worker releases the in-memory gate. Driver, SQLite/RPC, collector, and
  race tests passed. The [live gate](checkpoints/13-managed-meld.md) passed on
  September 25: alpha melded in 20.59 s, resumed in 5.94 s, and exact replay returned
  the same operation both before and after resume. Final health was healthy at
  revision 2/2, with no incidents/recovery requirement and direct HTTP identity
  `~zod`; see the [evidence](diagnostics/2026-09-25-managed-meld.md).
- Checkpoint 6D implements read-only one-shot container resource polling, with
  exact raw memory usage/limit and task count, container identity, original sample
  timestamps, and fresh/stale/unavailable semantics independent of ship health.
  Missing counters are not zero and late polls cannot resurrect stopped-container
  samples. Stats use the authenticated private Docker socket, without guest exec
  or Vere control-socket access. Driver, Unix HTTP transport, SQLite freshness,
  collector, and race checks passed. The [live gate](checkpoints/14-container-resources.md)
  passed on September 25: two fresh samples advanced for alpha and panel-check
  with stable container IDs, 4 GiB limits, and healthy applied revisions. The
  uninitialized instance had no invented sample; see the
  [evidence](diagnostics/2026-09-25-container-resources.md).
- Checkpoint 6E implements stopped-pier epoch inventory and non-executable
  digest-bound cutoff previews. Selection is an oldest contiguous prefix with
  the newest two epochs protected. Inspection is bounded, metadata-only, and
  rooted at the registered pier; lifecycle/configuration changes invalidate its
  cache. Driver, planner, SQLite/RPC, and collector fixtures cover replay, stale
  selection, unsafe layouts, and protected cutoffs. Live inspection, exact replay,
  and protected/digest/absent-cutoff refusal passed on September 25. The subsequent
  initial resume was refused before runtime work because available RAM was below
  5 GiB. After the user freed memory, a new request resumed alpha in 6.15 s;
  healthy 2/2 state, direct `~zod` HTTP identity, cache invalidation, and exact
  running-state replay passed. The [6E live gate](checkpoints/15-epoch-preview.md)
  is complete; see the [record](diagnostics/2026-09-25-epoch-inspection.md).
  Actual pruning and multi-epoch recovery remain separately gated.
- Checkpoint 6F implements disposable offline-roll/multi-epoch qualification:
  one fresh isolated fake, three explicit pinned roll invocations, a new snapshot
  and monotonically newer epoch after each, same-pier restart after every roll,
  and an oldest-only preview retaining three epochs. Admission is refreshed before
  each boot/roll; persistent uncertainty prevents automatic retry/restart after an
  unresolved roll. Eight owned containers are tracked through cleanup. Driver,
  report-history, CLI, and race tests passed. [Live qualification](checkpoints/16-epoch-lifecycle.md)
  passed September 26 UTC: run `c81cf35049c0b2f6b78b728af5bb9802` created epochs
  0i0/0i66/0i101/0i118, verified all restarts, selected only 0i0 in the preview,
  and removed all eight containers. Restored management health was independently
  verified; see the [evidence](diagnostics/2026-09-26-epoch-lifecycle.md).
- Checkpoint 6G implements destructive qualification confined to a newly created
  disposable fake: four epochs, a revalidated oldest-only manifest, exact unlink
  of genesis epoch 0i0, three unchanged retained epoch fingerprints, ordinary
  restart, then controlled removal of the current checkpoint and pinned
  `play --full` recovery from the latest retained epoch followed by restart.
  Partial deletion and uncertain replay stay quarantined after cleanup. Fixture,
  Unix/filesystem, history, CLI, and race checks passed. The
  [live gate](checkpoints/17-prefix-prune-recovery.md) passed September 26 UTC:
  run `e31441cd9345af32611b0153afc0cff7` removed only 0i0, preserved the three
  retained epoch fingerprints, restored an 803,799,040-byte checkpoint from
  latest epoch 0i128, and passed both restart checks. All eleven containers were
  removed; see the [evidence](diagnostics/2026-09-26-prefix-prune-recovery.md).
  Managed pruning and older-retained-epoch recovery remain unqualified.
- Checkpoint 6H implements explicit managed offline roll with `offline-roll`
  negotiation, clean stopped-pier admission, shared expensive-work serialization,
  durable replay, fresh resource checks, exact one-epoch growth and snapshot
  validation, and persistent interruption quarantine. The report includes before/
  after epoch names and the created epoch, and success refreshes cached inventory.
  A durable runtime-cycle guard refuses another new roll until the pier has run;
  exact replay remains available. Driver, journal/RPC, collector, and race tests
  passed. The [6H live gate](checkpoints/18-managed-roll.md) passed September 26 UTC:
  three rolls created 0i22185/0i22199/0i22213 on alpha; stopped/running replay,
  runtime-cycle refusal, four-epoch preview, and healthy 2/2 resume passed. See the
  [evidence](diagnostics/2026-09-26-managed-roll.md).
- Checkpoint 6I implements explicitly confirmed managed genesis pruning using the
  qualified four-to-three epoch scope. Admission binds runner/instance selection,
  revision, cutoff, digest, confirmation and actor-scoped request ID. The shared
  6G unlink engine revalidates disk state and journals exact progress; interrupted
  deletion stays quarantined. Success refreshes inventory and leaves the pier
  stopped. Local driver, journal/RPC, confirmation/capability, observer, and race
  checks passed. The [6I live gate](checkpoints/19-managed-genesis-prune.md) passed
  September 26 UTC: alpha's 0i0 was removed, all three retained records matched,
  exact stopped/running replay passed, and healthy 2/2 resume completed in 5.97 s.
  See the [evidence](diagnostics/2026-09-26-managed-genesis-prune.md).
- Checkpoint 6J implements fresh-disposable repeated-prefix qualification: four
  roll/restart pairs create five epochs, a first deletion removes genesis plus a
  snapshot-bearing epoch while retaining three, and a second deletion removes the
  oldest remaining epoch while retaining two. Each round checks retained metadata,
  ordinary restart, controlled checkpoint loss, latest-epoch replay, and restart.
  Whole-prefix intent and per-file/per-epoch progress remain quarantined on failure.
  Sixteen containers and both recovery records are bounded and journaled. Tests,
  vet, and race checks passed. The [live gate](checkpoints/20-repeated-prefix.md)
  passed September 27 UTC: run `5c72c70d9ed83d9b9f199d48c24c860c` completed the
  five-to-three-to-two sequence, both recovery checks, and sixteen-container
  cleanup. Restored management health was independently checked; see the
  [evidence](diagnostics/2026-09-27-repeated-prefix.md).
- Checkpoint 6K implements confirmed managed oldest-prefix pruning using the 6J
  engine, including snapshot-bearing epochs and repeated jobs. Selection retains
  the newest two, checks the pinned layout/exclusive links, and is bounded by the
  128-epoch inventory limit. Each job installs a new durable manifest; older job
  evidence and exact replay remain intact. Legacy genesis reports/capability are
  preserved. Driver, journal/RPC, capability/fallback, boundary and race tests
  passed. The [6K live gate](checkpoints/21-managed-prefix.md) passed September 27
  UTC: two confirmed pruning jobs completed five-to-three-to-two, preserved
  retained metadata, and kept both historical replay results intact. Alpha resumed
  healthy at 2/2 with 0i33536/0i33561 retained; see the
  [evidence](diagnostics/2026-09-27-managed-prefix.md).
- Checkpoint 6L adds maintenance and retention to panel version 0.3.0: resource
  freshness/source/container identity, latest reports and operation history,
  explicit pack/meld/roll confirmation, stopped epoch inspection, protected cutoff
  selection, validated collector preview and separate irreversible-history consent.
  Reviews bind runner/instance/revision/digest/sample/entries and expire on changed
  observations. Model and isolated real-QML fixtures validate immutable selection,
  full uint64 names, ambiguous-result request retention, and no automatic retry.
  [Visual review](checkpoints/22-maintenance-panel.md) exposed stale shell QML;
  a shell restart resolved it. The user accepted the data coverage on September 28
  and requested a separate visual/interaction pass.
- Checkpoint 6M / panel 0.4 reorganizes the unified plugin into a compact fleet
  menu, setup pane and comprehensive monitoring window. It adds status indicators,
  local runner presentation/browser preferences, RAM bars/session trends, epoch
  summaries and a runner/ship-filtered activity feed. Existing confirmed maintenance
  moves into the window. Unsupported code retrieval, provisioning, connection
  editing and scheduling are explicitly identified. See the
  [design and backend decisions](checkpoints/23-fleet-desktop.md).
- The user accepted 6M and authorized backend implementation. Checkpoint 6N adds
  atomic identity-preserving collector connection edits with revision conflicts,
  and a bounded operator-only login-code RPC using container-private Eyre HTTP.
  Codes are excluded from snapshots, journals, errors and QML; the desktop helper
  offers a sensitive clipboard for at most 30 seconds. Go/vet/race and model/QML
  checks pass. The [runner upgrade/live gate](checkpoints/24-runner-edit-and-code.md)
  passed for code retrieval and temporary Eyre login/logout on September 28 UTC;
  no code/cookie was printed or persisted in the validation script. The existing
  desktop clipboard was preserved, so live clipboard expiry remains a follow-up.
- Checkpoint 6O implements bounded rooted stopped-pier copying and a disposable
  transfer qualifier. It reserves a cleanly stopped fresh fake as source, verifies
  a private destination copy, resumes only that copy twice, and verifies source
  content unchanged. Partial transfers/resumes remain quarantined. Unit/workflow,
  CLI, vet, race and cross-build checks passed. The
  [live copy/resume gate](checkpoints/25-stopped-pier-copy.md) passed for run
  `185ff2ed58b9abdd6861dfe64d6650f5`: 7 files / 1,016,840,204 logical bytes,
  both resumes verified, source unchanged and all four containers removed.
- Checkpoint 6P implements managed import of those qualified stopped fake sources:
  pinned metadata catalogue, confirmed digest selection, transactional instance /
  port / source reservation, durable host-gated copy, progress, replay, and
  initialized-but-stopped completion. Incomplete imports cannot bootstrap, and a
  private SQLite version guard prevents older runners discarding quarantine.
  Collector/Setup integration and Go/vet/race/model/QML checks pass. The
  [managed import live gate](checkpoints/26-managed-qualified-import.md) passed:
  import-check copied all 7 source files, completed stopped at 1/0, and passed
  exact replay, duplicate-source refusal and start/stop/resume through pinned RPC.
  It finished running/healthy at 1/1 on HTTP 18086 with no incidents.
- Checkpoint 6Q implements fresh-comet bootstrap qualification using the pinned
  image/pill, internally generated identity, an owned outbound NAT bridge and only
  loopback HTTP publication. Exact public comet identity must survive a clean
  stop/replacement resume. Three containers and the bridge are removed; private
  pier/report are retained and uncertain provisioning stays quarantined. Go,
  vet/race and CLI checks pass. The [live gate](checkpoints/27-fresh-comet.md) passed
  for run `cb31b1d255ffe45103cf90ed7458a593`, completed September 29 at
  01:03:35.707058531 UTC. Public identity stayed
  `~mapwyc-hadrup-listyc-haddeb--dansed-nodlen-narbel-samzod`; all three containers
  and the bridge were removed with uncertainty false.
- Checkpoint 6R adds opt-in mixed development management and durable comet
  registration, internally generated identity, pinned resume/observation and
  interrupted-bootstrap quarantine. The collector explicitly accepts the new mode
  at the same pinned endpoint; private schema version 3 prevents unsafe downgrade.
  Panel 0.7 exposes acknowledged comet registration. Local Go/vet/race and QML/model
  checks pass. The [managed live gate](checkpoints/28-managed-comets.md) passed on
  September 29: mode acceptance retained the runner ID and advanced enrollment to
  revision 2; comet-check generated an identity, replayed exactly, and retained that
  identity through stop/replacement resume. Telemetry, code redaction and comet
  maintenance refusal passed; see the acceptance record.
- Checkpoint 6S implements durable UTC pack/meld policies for qualified fake piers,
  explicit downtime/resume consent, configuration/policy revision binding, operator
  revalidation, atomic per-slot admission, missed/unsafe-slot reporting and one
  host-gated multi-stage operation. Restart does not continue interrupted work.
  Panel 0.8 exposes policy editing and activity. The
  [timed live gate](checkpoints/29-routine-maintenance.md) passed September 29:
  policy f702d36eabb8f6a03aaed0252df8dea1 dispatched one 03:37 UTC stop/pack/resume
  cycle on import-check; healthy replacement and no duplicate slot execution were
  verified. The test policy was then disabled at revision 2.
- Checkpoint 6T implements fresh-disposable comet pack/meld qualification with
  before/after logical sizes, isolated utilities and exact-identity restart after
  each utility plus a final replacement resume. Seven containers and the bridge
  are cleaned; uncertainty remains explicit. Local Go/vet/race and CLI checks pass.
  The [live gate](checkpoints/30-comet-maintenance.md) passed September 29 for
  5418c87ccd3aac4ef8a1597adedeae0c: both utilities and all identity-preserving
  restarts resolved; seven containers and the bridge removed.
- Checkpoint 6U adds managed comet pack/meld and UTC routines under the existing
  host/instance gates, with private provisioning proof, identity-pinned resume,
  failed-outcome quarantine and monotonic schema-5 compatibility guard. Panel 0.9
  uses the new `comet-compaction` capability. The
  [managed live gate](checkpoints/31-managed-comet-compaction.md) passed September 29:
  comet-check retained its identity through manual pack/replay/resume and one
  14:46 UTC timed meld/resume. The test policy was disabled at revision 2; panel
  0.9 was installed and its loaded version verified. External imports, keys, host
  Ames, comet retention and richer policy remain queued.
- Checkpoint 6V adds fresh-comet epoch lifecycle qualification: three isolated
  rolls with snapshot-bearing inventories and exact-identity restarts, a final
  replacement, and a four-epoch oldest-only preview retaining three. All nine
  containers and the bridge are cleaned; no epoch is deleted. The
  [standalone live gate](checkpoints/32-comet-epochs.md) passed September 29 for
  e84ffd83afcfbbe71e6cceca2d69c30f with epochs 0i0/0i78/0i178/0i287 and exact identity
  preserved. All nine containers and the bridge were removed.
- Checkpoint 6W adds managed comet inspection, explicit roll and preview-only
  retention under the existing revision/replay/host gates. Comet previews cannot
  inherit fake pruning permission; both admission and dispatch reject deletion.
  Schema 6 prevents unsafe downgrade, and panel 0.10 exposes capability-gated epoch
  controls. The [managed live gate](checkpoints/33-managed-comet-epochs.md) passed
  September 29: three roll/replay/resume cycles plus final inspection retained
  0i0/0i7046/0i7064/0i7082 and the original identity. Confirmed pruning was refused,
  disabled policies were unchanged, and the final runtime was healthy at 1/1.
- Checkpoint 6X adds disposable comet repeated-prefix deletion and latest-epoch
  recovery: four roll/restart pairs prepare five epochs, two exact deletions retain
  three then two, with ordinary restart, controlled checkpoint loss, replay and
  identity-preserving restart after each. A final replacement verifies the newest
  two; 17 containers and the bridge are cleaned. The
  [standalone live gate](checkpoints/34-comet-prefix-recovery.md) passed September 29
  for 198fec0f0e49df2bcb0215789f2b5f1d, retaining 0i557/0i592 with both recovery checks
  and unchanged identity; all 17 containers plus the bridge were removed.
- Checkpoint 6Y adds confirmed managed comet prefix pruning, separate capability
  gates, exact stopped inventory/cutoff/revision binding, per-file durability,
  independent failure quarantine and schema-7 downgrade protection. Panel 0.11
  exposes fresh review plus history-loss confirmation. The
  [managed live gate](checkpoints/35-managed-comet-pruning.md) passed September 29:
  a preparation roll added 0i7847; cutoffs 0i7046 then 0i7064 retained 0i7082/0i7847.
  Exact immediate metadata, protected-cutoff refusal, historical replay and
  same-identity resumes passed, ending healthy at 1/1 with disabled policies unchanged.
- Panel 0.11 was installed after 6Y acceptance, and a shell restart plus live status
  IPC verified the loaded version.
- The subsequent custom-runtime and control-socket experiments have been removed.
  Use the pinned official release, keep --no-conn and defer loom-internal telemetry
  and pressure advice. No historical report compatibility or custom-candidate
  adoption layer is required before a first release.
- Management/SSH service installation, management-only restart, controlled engine
  restart and a clean host reboot passed live checks. The full reboot restored all
  four initialized ships through one new startup-resume job each, preserving identity
  and configuration; the uninitialized registration remained stopped.
- Next work prioritizes lifecycle recovery, released-runtime maintenance and
  real-ship setup. If runtime choices are needed
  later, use a simple upstream `live`/`edge` model. See [runtime-policy.md](runtime-policy.md).
- No production deployment is supported yet.

## 1. Contracts and safe simulator

Define host/instance/connection identifiers, protocol negotiation, snapshots,
errors, configuration revisions, idempotent requests, operations, and incidents.
Create a small explicitly simulated runner and collector CLI before managing real
resources. Use a private local socket, no Docker, no key input, and no pier access.

Validation: independent clients see the same snapshots; stale configuration
updates fail; duplicate requests do not execute twice; incompatible operations
conflict; disconnecting a client does not cancel the simulated job; unsupported
protocol versions fail explicitly. Be honest about memory-only demo state.

Manual gate: a documented two-terminal walkthrough using only development data.

## 2. Runtime qualification

Qualify a pinned upstream image on x86-64 with dedicated-account rootless Docker.
Verify direct Vere invocation, permissions, persistence, HTTP/Ames networking,
custom URL/local pill boot, clean shutdown, reboot, offline utilities, and safe
control-socket collection. See runtime-qualification.md.

Validation: record exact artifacts and environment; pass lifecycle and resource
tests on disposable ships. Resolve the known socket-disconnect hazard before
enabling continuous control-socket telemetry.

Manual gate: user-approved setup and disposable-ship qualification. If rootless
fails, explain evidence and agree on an explicit alternative; no silent privilege
changes.

## 3. Minimal real runner and bootstrap

Add explicit instance registration, desired configuration, transactional journal,
start/stop/status, diagnostics, locks, and authenticated access attribution. Boot
from URL by default or imported local pill, with digest verification and temporary
key handling. Reconcile incomplete bootstrap and interrupted jobs on restart.

Validation: two disposable instances; existing/partial pier refusal; download
failure/hash mismatch; idempotent bootstrap; clean stop/restart; journal recovery;
no secret-bearing arguments or logs. Test simultaneous configuration changes.

Manual gate: manage a disposable ship using the CLI, including a runner restart.

## 4. Collector, SSH, and generic ship sessions

Incremental gates:

- **4A: collection foundation.** Persist explicitly enrolled local runners with
  immutable name/address/identity bindings. Route commands through a selected
  connection, verify identity on every new connection, and poll runners
  independently. Qualify connection isolation with two local simulator runners
  before introducing remote transport and credentials.
- **4B: SSH access.** Add reviewed access grants, fixed-command transport, verified
  host keys, authentication-derived key attribution, and remote reconnect tests.
- **4C: ship sessions.** Add unmanaged ship enrollment and origin-bound Eyre
  authentication, credential adapters, and subscription recovery.

Add persistent per-user enrollment, verified SSH connections, inventory,
snapshots/events, polling, and isolation between unavailable connections. Add
origin-bound Eyre authentication/subscriptions and desktop/headless credential
adapters. Keep application access distinct from host management.

Validation: two collectors and two runner hosts; host-key errors; reconnects;
cross-operator conflicts; external ship without a runner; origin isolation;
expired sessions; subscription replay/reconciliation; redacted diagnostics.

Manual gate: laptop CLI manages a remote disposable instance and separately
connects to an unmanaged ship.

## 5. Omabit management interface

Incremental gates: **5A** validates the initial management panel and collector
command binding in the actual bar. **5B** adds the reviewed legacy-theme migration,
application sign-in UI, notification policy, and richer onboarding. Hosting setup
and real-key bootstrap remain constrained by their runtime qualification gates.

Build Omaship collection, enrollment, configuration, operations, health, incident
indicators, bootstrap, and reviewed dependency setup. Add laptop acknowledgement.
Migrate the existing theme integration without moving Omarchy behavior into core
services. Preserve shipped settings and credential behavior through an explicit
migration.

Validation: isolated QML fixtures, desktop/mobile-sized layouts as relevant,
sleep/reconnect, inaccessible hosts, multiple instances, notification behavior,
and non-destructive plugin update/removal.

Manual gate: real Omarchy plugin controls a remote disposable instance.

## 6. Maintenance and epoch retention

Incremental gates: **6A** is the accepted managed offline-pack lifecycle and
idempotent replay. **6B** qualified offline meld on a fresh disposable pier;
**6C** adds managed meld admission/replay/recovery. **6D** establishes exact,
freshness-aware container resource measurements. **6E** adds stopped-pier epoch
inventory and protected-prefix previews. Loom metrics/advice, scheduling, and
actual epoch pruning follow their own qualification and recovery gates. **6F**
qualifies explicit roll and multi-epoch restart before selected-prefix deletion
and retained-suffix replay/recovery are attempted.
**6G** qualifies exact oldest-prefix deletion and latest-retained-epoch recovery
after losing the current checkpoint, on a fresh disposable pier only.
**6H** exposes explicit, durable offline roll for registered development piers,
enabling managed multi-epoch validation before managed pruning.
**6I** adds explicitly confirmed managed removal of genesis 0i0 only from a
four-epoch pier, retaining three; broader selected cutoffs remain separately gated.
**6J** qualifies snapshot-bearing and repeated prefix deletion with a five-to-three-
to-two epoch sequence before broader managed pruning is enabled.
**6K** integrates confirmed, bounded oldest-prefix selection and repeated deletion
into managed operations, preserving the protected newest-two suffix.
**6L** exposes the accepted explicit maintenance/retention workflows in the panel,
with immutable review and history-loss confirmation before pruning.

Add offline pack/meld with schedules, host resource admission, safe interruption
and before/after disk measurements using the official released runtime. Add epoch
inventory, cutoff preview, confirmation, protected suffix, stopped-pier
revalidation, and qualified prefix pruning. Never substitute blanket chop for a
user-selected cutoff.

Loom-internal telemetry and utilization-based advice are deferred.

Validation: stale/unknown observations do not trigger work; bounded retry;
insufficient resources; stale previews; protected epochs; exact deletion set;
restart and claimed recovery paths on disposable multi-epoch piers.

Manual gate: maintenance followed by selected-prefix cleanup and recovery checks.

## 7. Monitoring peers

Add persistent incidents, polling plus streams, observer attribution, restricted
directional peer grants, maintenance windows, deduplication, and recovery events.
Integrate independent relay-ship notifications only through a qualified path.

Validation: process failure, runner failure, partition, sleeping laptop, stale
metrics, and recovery. Do not turn unreachability into a confirmed crash.

Manual gate: two hosts observe each other with one deliberately disconnected.

## 8. Personal operator gateway

Serve a management-only web interface from the collector: inventory, metrics,
configuration, jobs, incidents, and storage cleanup. Do not expose application
transport or credentials. Use authenticated sessions and browser-origin/CSRF
protections. Private reachability is not authentication.

Validation: browser and native collector concurrency; session expiry; unauthorized
requests; CSRF; operation continuity after closing the browser; credential absence
from browser payloads. Reuse configuration revision and operation contracts.

Manual gate: a non-Omarchy browser operates the same runner as an Omarchy client.

## 9. Social utilities and Omarchy gossip

Implement small Contacts, Pals, DM, group-join, and channel connectors with Talon
handoff. Build separate Omarchy gossip with tag enforcement on every delivery
path, bounded tag-revocation delay, limits, deduplication, and retention. Read the
Talon theme schema directly from Settings in the gossip frontend.

Validation: wire fixtures plus disposable multi-ship acceptance; first DM, join
completion, contact-name layers, permitted/blocked forwarding, tag removal,
backlog behavior, and direct Settings theme updates. Tend details remain separate.

Manual gate: multi-ship rumor and theme walkthrough; no claims of cryptographic
anonymity or guaranteed confidentiality.

## 10. Agents and release distribution

Deliver stable CLI output, reviewed top-level skill installation, qualified MCP
integration, prebuilt binaries, service packages, generated microrepos, pill
releases, and upgrade/uninstall/recovery documentation.

Validation: clean x86-64 install without a compiler; digest/version compatibility;
upgrade/uninstall preserves user data; no secrets in agent config, logs, or release
artifacts. Generated repositories identify their source revision.

Manual gate: clean-machine installation using only published artifacts.

## First supported release gates

- Clean install, reboot, and explicit privileges.
- Independent access keys/collectors operating the same runner.
- Configuration conflicts and duplicate-request handling.
- Client and runner interruption during work.
- Verified URL/local custom-pill bootstrap.
- Qualified maintenance and post-maintenance advice.
- Prefix-only epoch deletion and tested recovery claims.
- Laptop sleep/reconnect and truthful partition reporting.
- CLI, Omabit, and gateway use compatible contracts.
- Secret-redacted diagnostics and non-destructive uninstall.

## Research baseline

Source research is not execution evidence and must be refreshed when pinning
release artifacts.

- Omarchy Docker policy: https://github.com/omacom/omarchy/pull/8056
- Pending rootless proposal: https://github.com/omacom/omarchy/pull/11386
- Vere Docker: https://github.com/urbit/vere/tree/develop/docker
- Vere runtime reference: https://docs.urbit.org/user-manual/running/vere
- Urbtop reference: https://github.com/nisfeb/urbtop/tree/7f5ba606ef84e9fdeb11f0a9949936f012313290
- Reported socket crash: https://github.com/urbit/vere/issues/1100
- Proposed socket fix: https://github.com/urbit/vere/pull/1101
- Pals/Rumors: https://github.com/Fang-/suite/tree/b69ce154ae6ce906c2598cd46c806158b51b8a0f
- Talon: https://github.com/nisfeb/talon/tree/8dccb37424e961a542784232b8e50c5097e0bfb8
- Tlon backend: https://github.com/tloncorp/tlon-apps/tree/3f0e5e90e280b67311f7932be8ee7c519f7b0cb8/desk
