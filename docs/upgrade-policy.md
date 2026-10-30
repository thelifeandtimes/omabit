# Stable-release upgrade requirement

## 0.3.3 maintenance release

The [0.3.3 qualification](releases/0.3.3/qualification.json) adds the tested
0.3.2 → 0.3.3 desktop and local service path. Runner schema47, desktop state1 and
bootstrap API1 are unchanged. The existing earlier paths below compose through
0.3.2; no reset or automatic remote upgrade is implied. See the
[release instructions](release-0.3.3.md).

## Qualified0.3.0 support window

The0.3.0 [qualification](releases/0.3.0/qualification.json) and
[upgrade guide](releases-0.3-upgrade.md) supersede the historical unreleased
qualification status in the dated sections below. The actual published-binary
0.1.0→0.2.0→0.3.0 path is qualified on the documented Linux/AMD64 owner-user
stack. Users on0.1.0 should take0.2.0 first. Schema21→47 preserves existing
state and refuses older writers. Expanded project recovery and other explicitly
listed experimental capabilities remain outside this release guarantee.

## Original stable upgrade contract

The user requires seamless upgrades from the proper **0.1.0 stable release** to
future supported versions. Treat this as a release gate, not a post-release task.
The current repeated fresh-install exercises are development qualification only.

## User contract

- Upgrade in place: preserve piers, stable runner/instance identities, configured
  ports, storage roots and directory identities, enrollments, SSH grants/host-key
  pins, secrets, preferences, operation receipts and replay history.
- A management/UI update should preserve running ships and stopped intent. Runtime
  upgrades that require downtime need a coordinated graceful maintenance path;
  "seamless" does not mean pretending every Vere upgrade is zero-downtime.
- Coordinate collector, runner and QML compatibility. Older remote runners must
  remain observable and expose clear capability-based upgrade guidance.
- Keep migrations versioned, transactional and restart-safe. Interrupted upgrades
  must be resumable without bootstrapping over, deleting or re-enrolling a pier.
- Verify staged artifacts before atomic activation. Define tested rollback rules;
  refuse unsupported downgrades rather than letting old code discard new state.
- Preserve the Unix owner, ordinary filesystem access, unrelated ships/services
  and preexisting host settings. The protected-ship policy applies throughout.

## Required qualification before stable 0.1.0

1. Maintain fixtures from every supported prior schema/release, with populated
   instances, custom roots, access grants, policies and completed/interrupted jobs.
2. Exercise N → N+1, repeated execution, interruption at each migration/activation
   boundary, and the supported rollback/downgrade-refusal path.
3. Verify unchanged identities, data, permissions, ports, intended running/stopped
   behavior and actor-scoped replay. Exercise mismatched client/server/UI versions.
4. Run an upgrade smoke test on disposable ships without resetting the host.
5. Publish a clear support window and upgrade command/UI, including any necessary
   original-runtime preparation and maintenance window.

## Development runtime settings

The0.2.0 remote-onboarding work keeps runner schema21 and desktop state/API1.
Pairing an existing runner modifies only the dedicated access grant and its exact
forced-command SSH key entry. New version1 attempt journals are separate from the
runner database and native-package evidence. Existing operator keys, port22222
connections, piers and running/stopped intent remain valid. Fresh installation and
known interrupted installation use the established owner-user installer; a missing
profile/path never authorizes recreating an existing runtime. See
[remote setup](remote-runner-setup.md) and its development verification record.

RC.15 separates reuse of existing OS-delegated subordinate IDs from allocation of
new ranges. Fresh owner setup validates the local account binding and both
root-owned grant files without requiring an existing Omabit profile. Existing
explicit reservation policies still apply; unknown identity domains do not gain
automatic allocation authority. No runner/desktop schema change is involved.

RC.14 adds a separate version1 desktop activation and
profile store. Default user paths remain unchanged; named profiles have private
XDG roots. Immutable UI/tool generations share a bootstrap/profile contract, and
unknown formats are refused rather than relabeled. Desktop selection never opens
or downgrades a runner database. See [desktop development](desktop-development.md)
for activation, recovery and stock-Omarchy upgrade scenarios.

The **RC.13** host-compatibility work advances to **schema21**. It
accepts schemas11–20 without rewriting historical instance bodies, signatures or
receipts, and adds genuine RC.12/schema20 writer fixtures. New container records
carry the observed engine version; historical omitted fields remain omitted.
Older runners must refuse the successor rather than discard this evidence or
apply obsolete engine-admission assumptions. See
[host compatibility](host-compatibility.md) and its
[upgrade procedure](host-compatibility-upgrades.md). Source and isolated
native-package verification do not qualify every new runtime/distro combination;
the [RC.13 notes](release-rc13.md) and attached qualification record define its scope.

The post-RC.11 source remediation advances to **schema20**, accepting existing
schemas11–19 without resetting instances or receipts. The new version fences
older runners from ignoring durable management-update admission, object-bound
deletion/staging evidence, maintenance dispatch phases and handoff audit receipts.
Historical signature bytes and actor/request fingerprints remain unchanged.
Already-persisted interrupted imports have explicit evidence-checked recovery;
unexplained replacement objects and ambiguous legacy maintenance stay preserved.

Quiescence-aware managers use a durable local-root update fence, verified retained
executable generations and a restart-safe activation journal. Runtime processes
remain independent. A previously published schema19 manager lacks that admission
contract: use the explicit [one-time offline upgrade](management-offline-upgrade.md),
which holds the existing runtime lock, migrates and fences in one transaction,
and preserves owner, piers, configuration, history and running containers. An
older binary is never restored across an incompatible schema migration.

Historical fixtures produced by original RC.1–11 writers and the recovered
schema14/schema18 previews now cover schemas13,14,15,16,18,19. Every published
release has its own pinned original-writer fixture. Original writers for unpublished
schemas11,12,17 remain unavailable; their synthetic migration coverage is not
equivalent to original-writer qualification. These prototypes are outside the
supported published-release window. See the fixture manifest in
`internal/managed/testdata/historical-upgrades/` and the
[release-by-release support matrix](reviews/2026-10-18-rc12-historical-upgrades.md).
The frozen RC.12 artifacts subsequently passed the documented supported-host
upgrade, runtime, guest power-cut and desktop gates. See the
[release qualification record](diagnostics/2026-10-18-rc12-release-qualification.md)
for results and limits; this does not replace the separate stable0.1.0 requirement.

The integrated RC.10 uses schema19 and accepts schemas11–18 transactionally.
This fences runtime-settings-aware handoffs from schema18 preview binaries that
would discard their settings. Historical handoffs retain their original signature
bytes. New nondefault transfers sign loom/native settings and require capability
checks before source fencing. Legacy configuration requests preserve omitted
native-control values; explicit false still disables them on the next replacement.
Release metadata obtains the schema from the built runner's `state-schema` command.

The subsequent boot-form refinement restricts new loom selections to30–34 and
makes native control CLI-only. Existing smaller RC.10 configurations/launches,
signed transfers and actor-scoped creation replays retain their historical contract.
Observation, Stop, resume and maintenance must remain possible without relabeling
or clamping stored values. New configuration requests use the narrower bounds.
Desktop edits omit native control and preserve an existing CLI choice. This is an
in-place source update with no persistent format change or reset requirement.

The contributed schema18 preview broadens explicit native control to all ship classes. Its version-only
transaction accepts schemas11–17 without rewriting bodies or receipts. Schema17
runners refuse the successor because their runtime validator forbids enabled livenet
sockets. Source tests cover repeated schema17 upgrades and preserved request replay.
A separately approved schema18 management update also preserved the four current
instance configurations, running/stopped intent and both running container/process
identities. It did not enable or exercise any livenet socket. Livenet runtime checks
remain separate from both management updates.

Schema17 records loom selection and development-fake native-control opt-in.
The version-only transaction accepts schemas11–16 and leaves instance and operation
bodies unchanged. Missing private launch fields mean the historical loom31 and
native-control-off contract. Existing running containers retain that contract until
an explicit replacement. Older schema16 runners refuse the new schema.

Source tests cover legacy bodies, repeated migration, defaults, desired versus
recorded settings and future-schema refusal. A separately approved management-only
upgrade preserved two running ships' containers, processes, configuration and
operation history. Fresh-fake native-thread acceptance also passed. This is not
full qualification of every stable-release upgrade path. See [the guide](development-runtime.md)
and [verification report](runtime-settings-validation.md).
No database reset or downgrade-marker edit is an upgrade or rollback procedure.

## Current RC integration decisions

The Ames enrollment fix applies to new live enrollments. Opening an existing
database must not silently publish a new UDP port or change a running container's
network contract. Legacy unpublished enrollments remain readable with their current
settings. The post-RC.11 source provides explicit `publish-ames` for a cleanly
stopped enrolled live ship, with capability, identity/provenance, filesystem and
host/reserved-port checks. It advances the configuration revision, records the
authenticated actor/request and prior launch contract, and leaves the ship stopped;
publication happens on the next explicit Start. Exact replay survives restart.
It requires neither deletion nor re-enrollment. See
[legacy Ames publication](legacy-ames-publication.md) for the CLI contract and
qualification limits.

Import recovery is an explicit stopped-state journal operation, not an automatic
upgrade side effect. Existing schema16 records and their private recovery fences
must survive management restarts. New recovery behavior must handle legacy records
as well as guided/named/in-place imports, and must never infer that a missing
historical source path authorizes deleting or replacing another copy.


## Unreleased application connection journal

The platform experiment adds a separate private version-1 `applications.json`
and per-attempt locks beside the collector's existing registries. It does not
change `ships.json`, `runners.json`, desktop state or the runner database. Existing
readers leave the new file alone; application cleanup retains its own origin,
identity, adapter and opaque reference if a parent enrollment is removed. New
readers refuse unknown formats without resetting them. Source interruption tests
cover issuance, handoff and cleanup; deployment/rollback and installed receiver
qualification remain future release gates. See the
[durable issuer record](diagnostics/2026-10-21-durable-issuer.md).


Application approval development adds `application-receivers/DIGEST` beneath the
collector directory. Retain these verified helper generations while an attempt
may need receiver reconciliation; plugin upgrades do not replace the approved
generation. `ships` gains an additive `revision` field for stale-selection checks.
The declared helper file is additive to both plugin packages. Theme's installer
continues to recognize the earlier receiver-recovery inventory. Completed-record
and cache retention must be qualified before general release; never delete
uncertain lifecycle evidence as an upgrade/migration shortcut.


Completed application archival advances `applications.json` from version 1 to 2
at the first removal, after its private `application-history/REQUEST_ID.json`
receipt has been written and synced. New code reads both versions. Old issuers
refuse version 2, preventing replay of IDs removed from the active list. A crash
before removal leaves the active evidence, receipt, or both; same-revision retry
is idempotent. Never lower the version or erase receipts to downgrade.

Theme/DMs retain their version-1 active receiver journal so already retained older
helpers can finish cleanup. New helpers compact reconciled entries into permanent
private receipts before removing them, including legacy journals at the 64-entry
limit. Old helpers do not understand these receipts; the upgraded issuer must
remain the approval authority. This is a compatibility path for cleanup, not
permission to replay old offers manually. Pending/accepted entries and all
retained helper generations remain preserved. Helper-cache pruning is not yet
implemented or qualified.


## Unreleased schema22 source restoration

The platform work advances the development runner to schema22 for durable
`import_restore` request/journal state and completed restoration receipts. A
version-only transaction accepts schemas11–21, preserving existing instance
bodies, identities, ports, grants, policies, replay records and runtime intent.
It does not restore or restart an import as an upgrade side effect.

Schema21's original writer at `b23312876c611e1d083bdcc9468469f7e716d749`
now has a reproducible synthetic fixture, separately classified as a baseline
without extending the frozen RC.12 release or host-support claims. That original
reader refuses a database produced by the real schema22 migrator without changing
it. Earlier fixture coverage and explicitly unavailable prototype writers remain.

Restoration persists every move phase before publication. Only a verified,
durably completed source restoration can atomically retire the failed instance
and publish its terminal receipt/tombstone revision. Crash or publication failure
preserves the journal and recovery fence. A later registration advances the
revision; old actor/request IDs retain their original receipts. Installed upgrade,
UI and real VM restoration qualification remain separate release gates.

## Unreleased schema23 project reservations

Schema23 adds `projects` and `project_members` in the migration transaction,
retaining legacy bodies and receipts. The reader accepts schemas11–22, and opening
never creates/adopts runtime resources. Project member IDs/HTTP ports are reserved
atomically with the actor-scoped operation; standalone admission consults these
claims. Missing/altered ownership refuses opening or allocation. Do not lower the
version or delete reservations to make an older runner open the database.

The unreleased schema22 writer is retained as a reproducible source snapshot,
separate from release commits and frozen RC.12 claims. It reproduces a synthetic
fixture and refuses actual schema23 state containing project reservations without
modifying it. See [source evidence](diagnostics/2026-10-23-project-store.md). Runtime,
installed upgrades and project lifecycle integration remain separate gates.

## Unreleased schema24 project preparation

Schema24 adds `project_jobs` transactionally, preserving schemas11–23 and existing
project/member reservations. The original operation and its runtime mirror retain
engine/contract bindings and each dispatch/returned-ID/inspection step. Restart
interrupts pending work without erasing its journal or recreating resources.
Older schema23 readers must refuse it. Do not delete the journal or lower the
schema marker to permit a downgrade. The frozen schema23 writer and real-schema24
refusal test are documented in [the journal evidence](diagnostics/2026-10-23-project-journal.md).
No installed database was migrated; runtime dispatch and member lifecycle remain
separate gates.


## Unreleased schema25 initial anchor Start

Schema25 adds `project_anchor_starts` transactionally and retains every existing
preparation/member reservation and receipt. Start intent, acknowledgement and
boot-bound namespace proof remain separate commits. Startup interrupts unfinished
work without replay or resource adoption. The original schema24 writer reproduces
populated preparation and refuses actual schema25 dispatched Start unchanged.
Never lower the marker or delete retained uncertainty to permit downgrade. No
installed database was migrated; actual runtime and release gates remain. See
[the source evidence](diagnostics/2026-10-23-anchor-start-journal.md).


The private platform development slice now uses **schema26**, accepting schemas11–25
without rewriting earlier bodies or receipts. It adds durable project-member storage
claims and mkdir dispatch/identity journals; previous readers must refuse these
claims rather than permit a conflicting standalone import or move. An original
schema25 source snapshot reproduces completed creation/Start state and refuses an
actual schema26 dispatched member-storage operation unchanged. This is source and
temporary-filesystem qualification, not a deployed/released migration. See
[member storage evidence](diagnostics/2026-10-23-member-storage.md).


## Unreleased schema27 initial member launch

The current development reader uses **schema27**, accepting schemas11–26 while
preserving historical bodies, identities and receipts. Initial member launch has
a separate journal and atomic operation mirror; workspace/Create/Start uncertainty
cannot be discarded by an older reader. Startup interrupts without replay/adoption.
An original schema26 snapshot reproduces completed anchor/storage receipts and
refuses actual schema27 dispatched launch unchanged. Never lower the marker or
remove this evidence to downgrade. No installed database was migrated; see
[source qualification](diagnostics/2026-10-23-member-launch-store.md). Concrete
runtime, readiness and release/installation qualification remain separate gates.


## Unreleased schema28 readiness observations

Schema28 adds immutable per-attempt initial member readiness journals. It accepts
schemas11–27 without rewriting prior project, launch, storage or operation bodies.
Restart interrupts unfinished observations and never resumes runtime effects.
The original schema27 writer/reader is frozen and pinned; populated fixtures
reproduce exactly, and the old reader refuses real schema28 readiness state without
modification. These are source database checks, not release or runtime readiness
qualification. See [the readiness record](diagnostics/2026-10-23-member-readiness-store.md).


## Unreleased schema29 graceful member Stop

Schema29 adds per-operation Stop journals bound uniquely to each member/container.
It accepts schemas11–28 and preserves prior project, launch, readiness, storage,
legacy instance and operation bodies. Restart retains uncertain Stop intent;
it never re-signals or removes resources. The original schema28 writer/reader is
frozen and pinned, with reproducible populated readiness fixtures and refusal of
real schema29 Stop state without modification. These are source database checks,
not runtime or release qualification. See [the Stop persistence record](diagnostics/2026-10-23-member-stop-store.md).

## Unreleased schema30 read-only Stop exit observations

Schema30 adds per-attempt exit journals while preserving original Stop receipts,
history and uncertainty. It accepts schemas11–29; startup interrupts observations
without signaling or resuming any runtime. The frozen original schema29 writer
reproduces populated readiness and interrupted Stop evidence, and its reader
refuses actual schema30 state unchanged. The160KiB fixture bound applies only to
the larger schema28/29 originals; earlier fixture bounds remain unchanged. These
are source database checks, not an installed upgrade or real recovery. See
[recovery persistence](diagnostics/2026-10-23-member-stop-exit-store.md).

## Unreleased schema31 stopped-member retirement

Schema31 adds exact-container retirement journals, preserving original Stop/exit
receipts and piers. It accepts schemas11–30; restart interrupts unfinished removal
without replay or replacement. The pinned original schema30 writer reproduces its
179499-byte complete exit fixture and refuses real schema31 state unchanged.
Its192KiB fixture bound is specific to that original; older limits stay unchanged.
These are source database checks, not an installed upgrade or real retirement.
See [retirement persistence](diagnostics/2026-10-23-member-retirement-store.md).


## Unreleased schema32 retained-pier replacement

Schema32 adds an independent journal for the first explicit member replacement
following a completed retirement. It preserves original launch, readiness, Stop,
exit and retirement receipts and never restarts a ship while opening/migrating.
Unfinished replacement dispatch remains interrupted and fences conflicting project
work. Original schema31 fixtures and downgrade refusal pass source checks; actual
replacement and stable-release upgrade qualification remain separate. See
[the persistence evidence](diagnostics/2026-10-23-member-resume-store.md).

## Unreleased schema33 replacement-readiness observations

Schema33 adds per-attempt read-only readiness bound to a completed replacement.
It preserves all original bodies, interrupts pending observations at restart, and
never starts or adopts a runtime while opening the store. The frozen original32
writer reproduces its populated fixture exactly and its reader refuses real33
state unchanged. These are source database checks; runtime and stable-release
qualification remain separate. See [the readiness journal evidence](diagnostics/2026-10-23-member-resumed-readiness-store.md).

## Unreleased schema34 replacement-bound Stop

Schema34 admits Stop receipts bound to completed replacement readiness and causal
member history. Existing bodies and old uncertain Stops remain unchanged; startup
never signals or resumes a container. The original33 source and populated fixture
are pinned, reproduce exactly, and refuse real34 state unchanged. This is source
qualification, not an installed upgrade or runtime pass. See
[the persistence record](diagnostics/2026-10-23-member-current-stop-store.md).

## Unreleased schema35 repeated retained-pier replacement

Schema35 accepts schemas11–34 and permits multiple resumes per member, with one
resume per completed retirement. It transactionally rebuilds the resume parent and
readiness child while foreign keys stay enabled, retaining original body bytes.
Unknown references or validation failures roll back. Opening never repeats runtime
effects; unfinished work stays fenced. Original34 reproduces and refuses real35
unchanged. These are source checks, not an installed upgrade or later-generation
runtime pass. See [the persistence evidence](diagnostics/2026-10-23-member-repeated-resume-store.md).


## Unreleased schema36 current-context readiness

Schema36 adds a per-attempt current-readiness table without rewriting earlier
launch, readiness, Stop, retirement or replacement bodies. It accepts11–35 and
interrupts pending observations at restart without runtime effects. Original35
source/fixtures reproduce and refuse actual36 state unchanged. Focused store,
queue and migration checks and full managed race814.392s pass; runtime qualification
remains pending. No installed database was migrated. See
[current readiness evidence](diagnostics/2026-10-23-member-context-readiness-store.md).

## Unreleased schema37 current-context lifecycle

Schema37 permits Stop authority from completed current-context readiness, retaining
its exact creator, membership and process evidence through later lifecycle jobs.
This semantic fence changes no historical receipt bodies. Original36 reproduces
its populated fixture and refuses actual37 unchanged; migration tests pass.
Focused persistence/queue checks pass, while full regression and runtime validation
remain separate. See [the source gate](diagnostics/2026-10-23-current-context-lifecycle-store.md).

Schema37 follow-up: full managed race passes in972.505s with recorded sources unchanged. Runtime qualification remains separate.

Schema36/37 runtime follow-up: independently verified VM25 (M132) and VM26 (M137)
now pass their respective readiness and lifecycle gates. VM26 also verifies sender
replacement, original sessions and three persistent-state/subscription recovery
cycles. This supersedes the pending runtime statements above; it is not a live
installation or coordinated anchor-recovery qualification. See
[VM26 evidence](diagnostics/2026-10-23-current-context-lifecycle-runtime.md).

## Unreleased schema38 coordinated recovery

Schema38 accepts11–37 and adds parent recovery/member creator journals, preserving
all historical operation bodies. Admission and each transition validate their
actual source references, actor and mirrors. Incomplete recovery remains fenced
at restart without repeating runtime actions. Original37 reproduces its fixture
and refuses genuine38 state unchanged; historical migration tests pass. Full
managed regression and actual coordinated runtime qualification remain pending.
No installed database was migrated. See [M165 evidence](diagnostics/2026-10-24-persistent-anchor-recovery.md).

Schema38 follow-up (M167): full managed baseline race1307.093s plus later focused queue/fixture checks pass. Original37 artifacts remain unchanged. Actual coordinated runtime qualification and installed upgrades remain separate.

Schema39 (M169) adds explicit recovery observation receipts with immutable parent
and checkpoint ancestry. Existing recovery rows are preserved; restart interrupts
queued/running observations without replay. The accepted original-writer window
now includes38. Its frozen source and populated fixture reproduce and the original
reader refuses real populated39 unchanged. Focused migration/store/queue/race gates
pass; full39 regression and installed/runtime upgrades are not yet qualified.


## Unreleased schema40 explicit recovery continuation

Schema40 adds actor/request-owned continuation receipts bound to the exact parent
checkpoint and source history. It preserves original parent/member attribution,
prior interruption records and read-only observations. Parent/creator/continuation
progress commits atomically. Startup interrupts active continuation ownership
before generic parent recovery; it does not dispatch runtime work. Ordinary parent
callbacks cannot bypass the continuation's latest-checkpoint mirror.

Original39 observation fixtures reproduce; migration and original39 refusal of
actual populated40 state pass. Focused managed race861.164s and vet pass (M175).
Full managed race2164.666s and vet also pass (M178); real-runtime continuation remains pending. The current VM32 is frozen39 and excludes40.
See diagnostics/2026-10-24-schema39-original-writer.md.

Schema40 runtime follow-up (M243): VM42 passes independently audited interrupted
public recovery and three actual owner-service restart cycles against the installed
runner/store. Restart preserves checkpoint/history and containers without replay.
Application state and original sessions survive four recovery cycles. This
supersedes the pending runtime continuation statement above; installed production
upgrades and release qualification remain separate.
See diagnostics/2026-10-24-service-restart42-runtime.md.


## Supported collection intent prototype (M269)

The new internal collection journal uses a separate format1 and explicit root;
it neither migrates the Omabit-only desktop store nor opens runner databases.
Committed, rolled-back, pending-enable and uncertain-helper original-writer
fixtures are retained with source hashes. Future changes must preserve these
histories or refuse them intact. Missing journals cannot recreate prior state.
No user-facing activation or supported upgrade claim follows from model tests.
See [the collection journal contract](collection-journal.md).


## Unreleased DMs channel ownership format1

M273 adds a separate per-plugin/profile `transport/channels.json`, retaining exact
channel/session fingerprints, creator and cleanup identities, and closed receipts.
It stores no cookies or application bodies. Six original-writer snapshots and
writer sources are pinned in the M273 evidence; unknown/missing existing journals
refuse intact. This cannot reconstruct older unjournaled channels: adoption needs
verified old-helper drain. Downgrading to a helper that ignores the journal is
unsupported. History refuses at1024 records/2MiB; supported archival and expired/
missing-credential recovery remain release gates. No installed format was changed.


## Unreleased Theme transport format1

M276 introduces per-operation owner records and watch/poke channel history under
profile state `transport/`. Intent persists before HTTP; remote cleanup does not
resolve uncertain application writes. Six original writer snapshots/source are
retained. Unknown/missing existing state refuses intact, and caps2048 channels/
4MiB/8192 owner records never auto-prune. The installer includes the module while
recognizing its prior checkpoint inventory; no existing state was migrated.
Old-helper adoption, unsupported-reader rollback, safe live-client disconnect,
credential expiry and archival remain release gates. Do not remove this journal
as a recovery shortcut or use transport cleanup to discard pending appearance work.


## Unreleased Theme transport format2

M277 supersedes format1 for cooperating development clients. It adds bounded
retired-session fingerprints and shared read/channel admission versus exclusive
disconnect. Original M276 writer/snapshot migration preserves channels and refuses
live earlier owners; that original reader refuses the successor. Session limits
are2048, with existing channel/owner/byte caps unchanged. No evidence is pruned.
Receiver cancellation preserves credentials while old transport is unresolved.
This does not qualify pre-journal helper rollback, remote expiry or archival.

## Unreleased collection recovery format2

M282 reads format1 without rewriting; the next new collection event writes2 and
preserves every historical event. Exact replay stays byte-identical. The original
M281 writer/populated history is pinned, reproduces, and refuses actual format2
without mutation. New prepared/verified recovery records cannot be erased or
relabeled for downgrade. Pending app work, credentials and original problem history
remain distinct from transport closure. Model tests do not qualify installed
activation or runtime cleanup; see the collection recovery evidence.


## Unreleased collection artifact format1

Prepared supported runtimes use a separate content-derived generation descriptor
in an explicitly supplied private store. The pinned descriptor binds input/payload
inventories, plugin/profile identity and Omabit's environment. Verification refuses
unknown formats or changed bytes; existing generations are retained. Preparation
has no bootstrap, shell configuration, runner database or application-state effects.
Interrupted hidden stages remain distinct from published generations; exact retry
re-verifies and re-syncs a published generation. This internal preparation API is
not a native installation/activation command. M286 qualifies file/crash and native
identity behavior with synthetic runtime bodies; real plugins and durable admission
remain required before public deployment.

## Unreleased native admission format1

M288 adds a separate explicit immutable-activation permit root with sealed issuance,
one-use role claims and irreversible closing. It does not migrate runner, desktop,
artifact or collection journals. Preserve ended/dead-owner claims and unknown or
damaged records; never delete a permit to authorize a replacement. All78 private
tests and five synthetic native-loader cases pass. Future changes must retain the
original writer and populated cases. Real artifact/coordinator integration is pending.

## Unreleased guarded collection artifact format2

M289 adds pinned admission support and widget metadata. The format1 writer remains
unchanged and reproduces original bytes. The new reader preserves legacy readability
marked non-admitting; the original reader refuses real format2 unchanged. No
generation migrates in place. Prepare a distinct complete generation; never relabel
old unguarded bytes as format2. Actual body/coordinator rollout is a separate gate.


## Unreleased shared runtime authority and admission format2 (M301)

The separate authority/reservation format1 pins its runtime context and canonical
journal. Admission format2 binds its issuance seal to that reservation. Existing
format1 admission is never rewritten; original sources remain available for
cleanup, and old readers refuse actual format2 unchanged. Four original-writer
snapshots and exact sources are retained in M301 evidence. Unknown/missing/replaced
evidence refuses intact. There is no automatic provisioning repair or handoff, and
no native artifact version currently advertises this contract. Package a new
generation before native use; never relabel an existing format2 artifact as shared
authority capable. Cross-session bootstrap and original-guardian-backed handoff
remain release gates. See diagnostics/2026-10-24-shared-authority.md.


## Unreleased shared-authority artifact format3 (M302)

New generations pin SharedHost and the complete shared-admission bundle. Formats1/2
remain readable and explicitly lack shared-authority capability; no generation is
rewritten. The original format2 writer/support/generation fixture is retained at
tests/fixtures/collection-artifact-v2, reproduces exactly, and refuses format3.
The refactored format2 writer also preserves original bytes. Native metadata and
grant validation do not fall back to legacy permits. Shared-authority handoff and
cross-session bootstrap are still unqualified; initial native admission does not
authorize deployment or migration of an active collection. See M302 evidence.


## Unreleased shared reservation history format2 (M303)

Only an explicitly guardian-bound new reservation uses format2. Existing format1
authority descriptors/tickets and initial-only reservations retain their original
contract and bytes; populated v1 reservations cannot acquire a later invented
guardian. The frozen M301 writer/dependencies/four original snapshots live at
tests/fixtures/runtime-authority-v1. Old readers refuse reservation2. Handoff keeps
all prior reservation/binding/journal-prefix/closure evidence; it never resets a
gate or erases uncertain work. Missing partial candidate files require original
preconstruction guardian closure and canonical rollback receipts, and remain
missing in retained evidence. Initial provisioning/adoption and cross-session
recovery still need explicit qualification. Updated Host EOF release is shipped in
new immutable generations only; historical support bundles are never rewritten.
See diagnostics/2026-10-25-authority-handoff.md.

## Unreleased collection candidate recovery format3 (M305)

Readers accept1/2/3 without rewriting historical events. New fenced rollback or
candidate prepared/verified events write3; ordinary histories still write2. The
frozen original format2 writer and actual M303 history are retained and reproduce;
that reader refuses3 without mutation. Candidate recovery binds its exact original
reservation/guardian and all selected namespace cohorts, even without a complete
loaded receipt. Outgoing recovery retains its separate history. Never erase events,
lower markers or replace the original owner to authorize a downgrade or retry.
Native qualification is currently a private DMs-only adapter, not an installed
migration or public profile command. See candidate cleanup recovery evidence.

## Unreleased initial recovery record1 and reservation3 (M308)

Initial cleanup records are separate private prepared/verified receipts under the
runtime authority. They bind the unchanged canonical registration history and
original reservation/guardian/admission/application cohorts. The first explicit
initial restart writes reservation3 with an immutable initial-recovery history
reference; ordinary initial reservation/handoff still write2. New readers preserve
1/2 histories and exact replay. The original reservation2 writer and actual populated
M306/M307 states are frozen; its reader refuses3 unchanged. Descriptor/ticket1 and
collection journal1/2/3 remain unchanged. Do not lower markers, erase receipts,
replace proof inodes or reset prior ownership to permit an old reader.

Historical native generations remain byte-identical/readable; older bundled
authority2 helpers cannot admit reservation3. Prepare compatible complete generations
and qualify public capability/upgrade preflight before deployment. This slice is
private same-compositor pre-adoption recovery, not cross-session bootstrap or an
installed migration. See initial-owner recovery evidence.

## Historical initial binding reconciliation (M309)

Verified canonical original-owner cleanup may reconcile the historical binding of
an already adopted initial reservation before unload. Existing binding/history
shapes and reservation markers remain2/3; the original reader2 accepts reconciled
format2 state. Guard claim state in the binding is historical admission, not a new
live grant. Exact replays preserve bytes. The bounded admission snapshot reopen
changes no writers/formats and does not repair missing or corrupt evidence.
Ship these source changes in new immutable generations; preserve historical
bundles. Compatible-generation/public upgrade preflight remains a separate gate.

## Unreleased explicit-runtime artifact4 (M311)

Only new complete generations carry the pinned runtime-protocol declaration.
Formats1–3 and their writers remain unchanged/readable but do not advertise that
contract. Frozen original writer3 reproduces and refuses4. Never edit an old
artifact marker, bundle or selected manifest to manufacture compatibility.
Preparation verifies declared versions against bundled reader literals before
publication; complete-set preflight binds exact artifact hashes. Existing runtime
state formats do not change. Readiness is not activation or cleanup authority;
required candidate cleanup must remain possible. Public activation and migration
from a loaded legacy collection remain unqualified. See runtime preflight docs.

## Unreleased authority provisioning intent1 (M312)

Only explicit new attempts use the separate authority-provision.json and runtime
claim. Existing authority descriptor1, ready seal1 and journal/reservation formats
remain unchanged. Staging inode identities and abandoned objects are immutable
history; never relabel, adopt unbound objects, remove the runtime claim or recreate
missing published evidence. Completed replay preserves later application history.
The original sidecar writer and real populated attempts are retained in M312.
Future formats must preserve them or refuse unchanged. Historical unrecorded partial
setup and compositor/session replacement remain unsupported. Ship changes in new
immutable generations; no installed state or old bundle was migrated.

## Unreleased desktop cleanup receipt1 (M315)

A separate canonical-coordinator desktop-recovery-UUID.json retains its prepared
original authority, reservation/history/closure and application/session/channel
cohort plus verified transport receipt. It changes no runner, authority, reservation,
admission or collection-journal format. Original writer and populated records are
retained in M315. Unknown or changed records refuse; never delete/relabel a receipt
or original evidence to retry. Missing cleanup records/runtime evidence are not a
qualified recovery path. Verified cleanup does not authorize new desktop admission
or downgrade. Public delivery and durable cross-session bootstrap remain separate.

## Unreleased desktop checkpoint1 (M316)

An immutable desktop-checkpoint-UUID.json in the canonical coordinator retains
verified original cleanup and exact authority/history metadata outside the runtime.
Original writer/populated records are retained in M316. Unknown/altered records
refuse unchanged. Original coordinator, lock, verified cleanup and canonical journal
prefix remain required; neither archives nor retired runtime objects are active
authority. Existing authority/admission/reservation/journal formats are unchanged.
Never erase/relabel original records or use an archive as a new live descriptor.
Successor ownership and installed/public upgrade remain separate gates.

## Unreleased desktop successor intent1 (M317)

An immutable desktop-successor-UUID.json in the canonical coordinator links its
checkpoint and original history/cohort to a new display and a staged runtime claim.
Its distinct desktop-successor-intent claim uses the existing exclusive provision
claim pathname; legacy fresh setup refuses the occupied path. It is not authority
format1 or a ticket and changes no existing journal/admission/runtime schema.
Original writer and populated records are retained. Lost replies preserve recorded
inodes; missing bound evidence and unknown formats refuse. Never overwrite the
old mirror, reset canonical history, relabel the claim as a descriptor, or delete
intent/claim to permit fresh setup. A dead new compositor leaves the intent pending;
second-session-loss recovery and compatible successor admission remain unqualified.

## Unreleased linked authority descriptor2 and publication1 (M319)

Only the new explicit successor reader accepts descriptor2. Existing descriptor1
readers and old shared gates refuse it. The new reader retains both authority
contracts; it does not rewrite old runtime bundles. A separate successor mirror and
object-bound publication record retain the original coordinator mirror/checkpoint.
Ticket1, admission2, reservation1/2/3 and journal1/2/3 formats remain unchanged.
Original writer/populated records are retained in M319. Never relabel a descriptor,
replace bound objects, erase old mirrors or reset the canonical journal to downgrade.
Native generations must explicitly bundle and declare the successor reader before
admission; existing artifact4 declarations do not imply authority2 support. Installed
and native application activation remain separate qualification gates.

## Unreleased successor artifact5 / protocol2 (M320)

Artifact5 is a distinct immutable package with explicit authority readers1/2 and a
complete successor-guard dependency bundle. Artifact1–4 writers/bytes remain unchanged;
new readers retain their original declared or undeclared capabilities. The frozen
original4 writer reproduces and refuses5. Never change an old generation's marker,
protocol file or hash to promote it. New preflight requires2 unless an explicit caller
requests1; the public read-only command keeps1 as its default. Mixed4/5 can serve1
but must refuse2 before effects. Cleanup must not depend on forward compatibility.
This packaging step changes no persistent authority/ticket/admission/reservation/
journal format. Native activation and installed upgrade qualification remain separate.

M322 qualifies the private artifact4-to5 packaging/guard upgrade across explicit
linked desktop continuation. Original immutable generations and application payloads
remain unchanged; the canonical journal retains both collection hashes. Actual native
apps survive lost controller replies without new login or pending-message replay.
This does not qualify installed-user migration or changing application data formats;
see diagnostics/2026-10-25-native-successor-upgrade.md.


## Unreleased linked desktop cleanup2 (M324)

New `desktop-successor-recovery-*` receipts require authority2 and bind lineage in
the evidence digest. Existing desktop cleanup1/checkpoint1 and runtime writers remain
unchanged. Original cleanup readers refuse2; never lower markers or rewrite old
claims to manufacture closure. Cleanup2 grants no successor admission. Its durable
checkpoint and public installed-profile integration remain separate gates.


## Unreleased linked desktop checkpoint2 (M325)

New `desktop-successor-checkpoint-*` records copy current authority2 evidence and
reference checkpoint1 without embedding ancestor bodies. Both canonical cleanups,
ancestor and immutable links remain mandatory after runtime retirement. Existing
writers remain byte-identical; original checkpoint readers refuse2. Never relabel
checkpoints, recreate missing evidence or infer admission from successful reading.
A future repeated-admission contract must explicitly advertise checkpoint2 support.


## Unreleased checkpoint2 continuation intent2 (M326)

Only the new `desktop-continuation-*` writer consumes linked checkpoint2; its separate
format2 runtime claim is never an authority descriptor. Existing intents/checkpoints
and runtime writers remain unchanged; original intent readers refuse2. Do not reuse
an old bundle or relabel a claim as admitting. Compatible authority/guard publication
and handling a lost claimed successor remain separate gates.


## Unreleased continuation authority3 (M327)

New desktop-activation publication records bind descriptor3 to continuation-intent2
and checkpoint2 with its retained ancestor. Older authority readers refuse3; new
readers preserve1/2. Ticket/admission/reservation/journal formats are unchanged.
Do not relabel an existing artifact5 bundle as authority3-capable or replace old
mirrors/claims. Compatible immutable packaging and native application activation
remain required before exposing this contract outside the private fixtures.


## Unreleased continuation artifact6 (M328)

Prepare new immutable generations for authority3. Artifact6 adds complete continuation
readers and an explicit authority1/2/3 list; historical declarations and writers remain
unchanged. Never relabel artifact5 as supporting3. New preflight accepts historical
bundles only for their declared contracts; mixed5/6 refuses3 before effects. The public
read-only CLI still defaults to1. Native activation and installed upgrades are separate.

M329 qualifies private package5-to6 continuation with unchanged application payloads
and state/session preservation. This is not installed-user or app-data-format migration
evidence. General repeated sessions require bounded reusable lineage before release.

M330 adds output3 read-only authority3 inspection with complete checkpoint ancestry.
Existing inspector outputs1/2 and persistent writers remain unchanged. Reading evidence
does not authorize cleanup or another activation.

M331 introduces cleanup3 with a separate desktop-chain-cleanup record prefix.
Original cleanup1/2 writers remain unchanged; earlier readers refuse3. Guardian
closure and transport cleanup stay distinct; neither permits another activation.

M333 adds reference-only checkpoint3 with23 focused tests and19 ancestry regressions.
Original checkpoint writers remain unchanged; earlier readers refuse3. Reusable
intent/admission and same-format repeated sessions remain unqualified. See
[checkpoint evidence](diagnostics/2026-10-25-reusable-checkpoint-envelope.md).

M334 qualifies reusable intent3 and dual legacy-reader refusal fences with20
private tests; no authority or application admission is granted. See
[ownership evidence](diagnostics/2026-10-25-reusable-session-ownership.md).

M335 qualifies one reusable recovery family across three further sessions with608
core tests and actual guards. Application observations are synthetic; packaging
and copied native plugins remain next. See
[repeated core evidence](diagnostics/2026-10-25-reusable-runtime-core.md).

M336 qualifies artifact7 and explicit authority4 preflight while retaining the
authority1 default, exact old writer6 reproduction and old-reader refusal.
See [package evidence](diagnostics/2026-10-25-reusable-runtime-packages.md).

## Unreleased schema41 private-chain reservations

Schema41 adds separate private project/member tables in the migration transaction.
Existing fake-project bodies, fingerprints and recovery receipts remain unchanged.
Shared allocation checks prevent private/fake/standalone name or HTTP-port overlap.
Opening validates both ownership representations and private registration receipts;
it creates no keys, chains or runtime resources. Old40 must refuse41 rather than
ignore reservations. The frozen original40 writer reproduces populated synthetic
continuation state and refuses actual41 unchanged (M353). Do not lower markers,
erase claims or reset a database to downgrade. Installed upgrade and runtime
qualification remain separate; RC.12 release/host-support claims remain frozen.

## Unreleased private runtime input manifest1 (M362)

The separate inert input format is `private-chain-runtime-inputs-v1`. Original
writer, actual manifest, source reader and uninstalled binaries are retained in
M362. The reader refuses unsupported versions and ambiguous/changed bytes. This
does not change runner schema41 or any project/operation body. Never relabel an
input generation or rewrite its reviewed manifest to claim compatibility; prepare
and review a new sealed generation. Inspection grants no initialization, restart,
installation or downgrade authority. Supported runtime delivery remains unqualified.

M369 freezes the original41 writer and a reproducible synthetic fixture with two
private projects/four members plus the legacy fake/standalone history. Its separate
preservation test verifies repeated reopen and actor replay without changing any
original rows or probing host ports. Schema41 and the existing older-reader manifest
remain unchanged. Before activating a successor lifecycle schema, add this original
fixture to that successor's migration and original-reader-refusal qualification;
the baseline alone is not evidence that such a migration or downgrade fence passes.

Schema42 (M371) adds private workspace journals and mirrored operations while
preserving schema41 registrations and shared reservations. Original41 fixtures
join the development support window; the unchanged old writer reproduces its
fixture and refuses actual populated42 without modification. Workspace birth
identities use the existing seconds:nanoseconds spelling before any such state
is persisted. Focused store/replay/migration checks pass; full managed regression
is running. This is not installed upgrade or public runtime qualification.


Schema43 (M378) adds private gateway initialization and separate workspace/gateway
observation tables in one migration transaction. The original42 workspace journals,
old rows and sidecars remain unchanged; recovery creates attributed receipts rather
than rewriting uncertain attempts. The original42 writer/fixture is registered in
the development inventory, reproduces exactly and refuses genuine populated43
without modifying it. Corrupt retained state rolls migration back, including all
three tables and the marker. Focused race/static gates pass; full43 regression is
running. This remains development source evidence, not an installed/release upgrade.


M408 freezes the original43 writer and reproduces completed/uncertain private
gateway and workspace histories with independently attributed observations. The
dedicated preservation test checks unchanged rows, files, replay and readiness
ownership across three reopens. Schema43 and the older-reader inventory remain
unchanged. Before any successor member schema is activated, add this fixture to
actual migration and original-reader-refusal qualification; the baseline alone
does not establish either result.


## Unreleased schema44 first private member launch

Schema44 adds `private_member_launches` transactionally without rewriting earlier
workspace/gateway initialization, separately attributed recovery or actor replay.
Each private project currently permits one initial member journal; second-member
support requires a separately reviewed durable readiness contract. Reopening
validates all bindings; reconciliation interrupts pending work without dispatch,
resource adoption, repair or automatic first-boot retry. No public runtime route
or installed update is enabled by this source change.

The original43 writer reproduces its retained fixture and refuses actual populated
schema44 state unchanged. Migration/reopen checks preserve all original rows and
sidecars; invalid ancestry rolls back the new table and version. This extends the
development database-reader window, not the frozen RC.12 release qualification or
host support claims. Never lower the schema marker to downgrade. See
[source qualification](diagnostics/2026-10-26-private-member-store.md).

M431 freezes and reproduces the original44 first-member launch writer before
durable birth/readiness persistence. Completed and interrupted launch journals,
separate gateway recovery attribution and actor replay remain unchanged across
three opens. A real successor still needs original44 migration preservation and
original-reader downgrade refusal; see [the baseline record](diagnostics/2026-10-26-original44-member-launch.md).

M432 advances development to schema45 for version2 private-member launch capture.
Original44 rows retain their original version and omitted birth field. They remain
readable and replayable without execution or invented identity evidence. Migration
preservation with a new captured project, corruption rollback and genuine original44
reader refusal of real schema45 state pass. Published release qualification is
unchanged; see [durable capture](diagnostics/2026-10-26-private-member-durable-birth.md).


## Unreleased schema46 capture-before-observation

Schema46 adds version3 private-member launch records with a durable `captured`
phase between acknowledged Start and full running observation. The journal and
operation mirror commit atomically. Capture also reserves its exact pier/.urb
objects globally. Restart interrupts unfinished work without repeated Start,
new capture, adoption or repair. New internal queue admissions use version3;
existing version1/2 history remains unchanged and actor-scoped replay works
without a backend. Read-only recovery is a separate subsequent capability.

The actual original45 writer is frozen, reproducible and registered in the
development reader window. Migration tests preserve original rows, sidecars,
identity and replay alongside captured successor records; the actual original45
reader refuses populated46 state without changing it. M445 records this evidence.
M446 covers the concrete adapter/session checks and the preceding failed runtime.
Integrated regression and fresh schema46 runtime evidence remain required before
runtime or release qualification. RC.12/schema20 and host support claims stay
frozen. Never lower the database schema to downgrade.

## Unreleased schema47 — empty project networks

Schema47 permits a project reservation with zero members so the network can be
created before its ships. Existing schemas and immutable member histories are
retained. Earlier runner binaries refuse schema47 rather than interpreting an
empty reservation as a corrupt or disposable project. The creation workflow still
requires shared-administration acknowledgement and an explicit confirmation.


Schema47 also adds `project_expansions` and `project_expansion_members` for native
incremental fake-project membership and live lifecycle controls. Original project
plan/creator/history bytes remain unchanged. Claims are checked against the
ledger, and queued/running expansion work reopens interrupted without dispatch.
Only an explicitly selected preflight that retained phase `reserved` with no
storage/container effects can continue; dispatched unknown outcomes are not
replayed. Older writers must refuse47. Full expanded engine/anchor recovery is
not yet implemented; this development format is not release qualification.
