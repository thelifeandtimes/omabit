# Stable-release upgrade requirement

## Review development: immutable per-ship runtimes, schema66

Issue#22 introduces additive runtime-profile fields and a schema66 older-reader
fence. An omitted historical profile keeps the exact official Vere4.6 artifact;
migration does not rewrite original instance, operation, permission, recording or
handoff bodies. Existing clients that omit a runtime setting preserve the stored
selection. The JSON-fix correction requirement prevents an incompatible stock
downgrade; changes of runtime need a separate reviewed checkpoint/replay plan.

Source fixtures cover preservation, per-ship bindings and signed handoffs.
The [frozen0.5.0→rc.3 rehearsal](diagnostics/2026-11-24-runtime-profiles-upgrade.json)
now passes the actual local owner-user service upgrade, schema65→66, preserved
ship/app history, access and running processes, manager reopen and genuine0.5.0
writer refusal. [Native core runtime checks](diagnostics/2026-11-24-runtime-profiles-native.json)
pass separately. This review window covers that exact published0.5.0 starting
state; earlier users first follow their documented stable upgrade chain. Remote
updates, active recording-worker upgrades and power-loss durability are outside
this scenario. Full shell activation and immutable review delivery remain required. Do not install this development branch
on production until the [review guide](runtime-profiles-review.md) records those
gates. No marker editing, state reset, re-enrollment or protected-pier testing is
part of the procedure. Preserve runtime-profile archives alongside runner state.

## 0.5.0: qualified local0.4.0→0.5.0 support window

The [exact qualification](releases/0.5.0/qualification.json) binds frozen59d83aa
artifacts and the actual published0.4.0/schema59→0.5.0/schema65 owner-user upgrade.
It preserves identities, piers, access, settings, intent and original operation/app
history; manager restart, denial/retry and actual older-reader refusal pass.
Separate native desktop and preserved review040 DEV evidence pass, including a
fresh Gall sample's reload/restart state and final Stop after closing temporary
permissions. This supersedes historical pending final-artifact/DEV statements
only for the linked scenarios. Active recording-worker upgrades, remote service
upgrades, power-loss durability and arbitrary development snapshots remain outside
this window. Earlier stable versions follow their documented path to0.4.0 first.
Never downgrade markers, reset piers or discard history to update a manager.
Final protected publication remains separate and authorizes no production update.

## Development: native closed output-coverage response, schema65 unchanged

Matching5344c5c [native output coverage/export](diagnostics/2026-11-19-recording-output-coverage-native.json)
passes actual CLI/SDK and independent raw-frame/SQL checks. The new summary does
not change JSONL index files or complete redacted manifests versus4c. No original
recording/job/control/schema65 encoding, admission or lifecycle change is made.
This selected closed-prefix gate does not qualify active pressure/retirement,
other factories, final59 upgrade, plugin packaging or preserved DEV.


## Development: optional derived output coverage, persisted schema65 unchanged

The current core adds optional completed `output_coverage` to timing/index
summaries. Original/index/redacted files, metadata, jobs, controls, endpoints and
schema65 stay unchanged. Older omission is unavailable, not zero; current values
cover only the exact verified retained prefix. First/last sequence time differs
from earliest/latest wall time when clocks reverse. [Focused races/interfaces](diagnostics/2026-11-19-recording-output-coverage-source.json)
pass; matching artifacts/reference and current native verification remain gates.
This adds no retention, runtime or installed upgrade authority.


## Development: closed indexed/query/redacted native gate, schema65 unchanged

Matching4c0b16e tools pass the [closed original/index/timing/redacted export case](diagnostics/2026-11-19-recording-derived-export-native.json)
with actual CLI/stdio, independent raw-frame/archive/span/hash checks and exact
original capture/control/table/intent preservation except one fresh observation.
No persistent format, schema65, admission or lifecycle change is introduced.
The failed earlier whole case is retained; its framing correction passes a fresh
retry. This qualifies these closed development reads only. Active/retention/
pressure, final previous-stable59 upgrade, plugin packaging and preserved DEV
still require evidence; no production service change is implied.


## Development: original recording exports, schema65 unchanged

The new original manifest/part reads and private CLI ZIP writer change no stored
recording, job, receipt, control or schema. Operator/selected adapter and explicit
original/private choice remain required; older runners lacking the new feature
refuse. Source tests preserve stationary original bytes and report verified
prefix/torn-tail evidence separately from whole retained files. The source
[evidence](diagnostics/2026-11-19-recording-original-export-source.json) includes
actual CLI/SDK parity, role/path/hash/privacy and archive failure checks. Matching3fc6 [native closed original exports](diagnostics/2026-11-19-recording-original-export-native.json) now pass, including independent original/ZIP/chain/SQL preservation checks. Installed stable upgrade, preserved DEV and a final support window remain pending. This does not qualify active capture/pressure/retirement or redacted bundles; no installed or production change is implied.

## Development: optional recording storage observations, schema65 unchanged

`capacity.usage` adds transient read evidence without changing stored jobs,
recordings, receipts, policy or schema65. New clients accept older omission as
unavailable, and an actual frozen703de9f2 CLI ignores newer measured/unknown
fields while preserving its old settings/commitments. Source race/API/SDK/UI and
reply-bound checks pass; see the [evidence and limits](diagnostics/2026-11-19-recording-usage-source.json).
Matching frozen f37 tools now pass the [closed-capture native measurement case](diagnostics/2026-11-19-recording-allocation-native.json): actual CLI/stdio measurements
match independent filesystem counts, with original captures, controls, history
and stopped intent preserved. Only one fresh observation timestamp/revision is
permitted. Active capture growth, pressure/retirement, installed upgrades and
preserved DEV still require qualification. The earlier703de9f2 native pass is not
evidence for those later measurement behaviors.

## Development: recording policy/API native64→65 gate

The [matching-artifact native rehearsal](diagnostics/2026-11-19-recording-policy-native-upgrade.json)
passes at703de9f2: online activation preserves the old detached worker, native
container/engine and saved history; raw API windows match independent retained
bytes; audited policy/replay/lookup/pin and immutable old/new worker limits pass.
Two same-pier sessions Stop cleanly. The genuine old64 executable refuses the
closed65 database unchanged, and candidate reopen preserves controls. This
supersedes prior deferral only for that explicit development pilot/scenario.
Published0.4.0/schema59→65, installed/preserved DEV and the remaining recording
features still require their own qualification. No production service or stable
release changes are implied.

## Development: recording launch intents, schema64 — gates pending

Schema64 adds empty private recording intent/head tables without rewriting older
instances, operations, app journals, leases, audits or access. New intents bind the
authenticated original Start operation and exact saved runtime/configuration state
to one job, executable and admission digest. Insertion and independent count/digest
head advancement commit atomically; a missing tail or original ancestry refuses.
Saved replay is historical and cannot authorize another worker submission. Opening
performs no job claim, service submission or native effect.

Focused source/race gates preserve original59/60/61/62 and synthetic11–63 stores,
and cover concurrent admission, process exit after committed/no-response intent,
head-write rollback, corruption and mixed-schema refusal. Actual frozen63 writer/
reader refusal, relevant native/installed upgrades, final matching artifacts and
support window remain required. Normal launch still submits no recording jobs.
Never lower markers or remove recording history to run an older manager.

The [actual frozen63 writer/reader rehearsal](diagnostics/2026-11-18-recording-schema64-source-upgrade.json)
now passes for candidate6fd35d6d and unchanged1589158b with a declared synthetic
helper. Original rows/DDL/access, pending Start and earlier linked audit ancestry
stay exact; one new recording intent/head replays unchanged. The genuine63 reader
refuses populated64 unchanged. A compact actual63 SQL fixture verifies repeated
preservation and rollback in ordinary race tests. This supersedes only the source
writer/reader deferral above. Native capture, active-worker/installed upgrades and
final matching artifacts remain unqualified; normal submission is still disabled.

M694 adds an explicit unqualified development-only `--record-runtime` pilot.
Default service startup still submits nothing. Its source manager/server/CLI
gates do not qualify native capture or an installed/active-worker upgrade. Do not
select it for preserved manual-review state until those gates have been assessed.

## Development: explicit recording bootstrap identity, metadata4 — gates pending

New comet creation has no ship name before mining, and a first launch need not
yet have a pier/.urb object. The prepared metadata4 model distinguishes a
registered expected name from pending-comet-mining, and an existing-pier object
binding from a prepared-storage contract. Pending identity never receives a
guessed ship name or future inode. This is attachment intent, not proof of a
successful process, initialized pier or answered native identity. Later verified
observations need separate linked evidence; original metadata is immutable.

Metadata1–3 retain their exact canonical bytes and omitted basis fields. The
segmented format introduced by3 is also used by4; frame codec1 is unchanged.
The source coordinator selects current4 for new captures; generic legacy-format
source fixtures remain explicitly readable without promotion. No installed file
or runner database changes, and no recording feature is advertised.

The [actual original3 writer/reader rehearsal](diagnostics/2026-11-18-recording-metadata4-source-upgrade.json)
passes for candidate0b28bac and unchanged frozen6e810bc9 code with a declared
helper in `scripts/test-recording-bootstrap-upgrade.py`. Original3 segments,
cross-segment cursors, omitted identity basis and repeated reopen stay exact. The
actual old reader refuses populated4 without changing it. A compact actual
original3 fixture with source/helper/writer pins runs in ordinary tests.
Full recording race tests, focused ordinary native
factory/process/attachment race tests and vet now pass with inert engine/private
filesystem fixtures. Native capture factories,
worker lifecycle, later observed identity, real Docker/Vere, installed upgrade
and final artifact/support-window gates remain required. This is an unreleased
development format, not qualified recording support.

The private recording-worker job codec is version1 and currently unwired. It has
no installed predecessor, state database entry or running worker. Unknown or
noncanonical versions refuse. Before enabling persistent jobs, qualify durable
claim/readiness and exact-executable admission, add the runner's downgrade guard
and preserve old jobs/recordings through upgrade. Do not allow a schema63 binary
to ignore newly active recording authority or replace an existing session.

The unwired job-file source layer now writes immutable version1 jobs, readiness
and finite results plus claim2 under an explicitly supplied private test root.
Claim2 adds worker process-start ticks and service invocation. Legacy claim1
lookup leaves those fields unknown and cannot grant current worker authority;
the [actual original-writer gate](diagnostics/2026-11-18-recording-claim2-source-upgrade.json)
now passes for candidate252aa548 and unchanged85ac7c6 plus a declared synthetic
helper. Original1 bytes survive reopen/refused reclaim; the genuine old reader
refuses claim2 unchanged. A compact exact fixture runs in ordinary tests. This is
source filesystem compatibility only, not active-job/installed upgrade support.
Unknown/corrupt or
partial originals refuse and remain retained. Lookups preserve those bytes, and
closed claims cannot be retried as new captures. Source fsync/locking tests are
not native crash/power-loss or supervision qualification. The source worker now
observes only its kernel-selected self executable/process/namespaces; source
composition checks do not qualify independent process lifetime. These gates
and the active-job runner downgrade/upgrade guard are still required before
enabling this layer in installed launch.

File visibility is not proof that fsync returned. Installed launch must require a
separate post-sync supervisor acknowledgement and exact live-worker observation;
it may not treat a readable marker or Type=exec submission as that authority.

## Development: separate app runtime recovery audits, schema63

Schema63 adds immutable linked runtime recovery receipts and exact per-operation
heads. Migration seeds only empty heads, preserving every original app plan,
event, lease, framework receipt, runtime, access and other history row/DDL.
Opening never rebinds, starts, signals or redispatches a runtime. Missing/changed
audit ancestry or unowned successor tables refuse without repair; failed
migration rolls back. The internal apply transaction admits an exact prefix/head/
lease CAS only, with at most64 receipts. Saved exact replay returns the original
audit after expiration and manager restart without modifying original uncertainty.

The actual frozen62 writer at3e780879 contains acknowledged plan2 uncertainty
and original59/60 ancestry. Its synthetic metadata fixture is retained under
`internal/managed/testdata/grubbery-app-original62`. Focused race checks cover
preservation, rollback, process exit after committed audit/no response, replay,
concurrent admission, corruption and bounded history. The actual62 reader has
refused a populated63 store unchanged. The clean frozen217275f rehearsal passes
through `scripts/test-app-runtime-recovery-upgrade.py`, with
[source/binary/fixture pins](diagnostics/2026-11-18-app-runtime-audit63-source-upgrade.json).
No downgrade through marker/table edits is
supported. Native recovery, matching final artifacts, installed upgrades and
preserved DEV qualification remain required; no workflow readiness is advertised.

## Development: native version signals and rejected app writes, schema62

Schema62 fences explicit app plan/journal2 while retaining the schema61 table
shape and every original plan1/header/event/lease byte. New plans carry a native
`version` value plus source/epoch/original-request metadata. The first plain-root
marker uses the pinned native file-make parent behavior, avoiding a separate
unmarked-root phase. Fresh admission must still refuse existing unowned roots.
A complete validated native pre-effect rejection is separately retained as
`refused`; transport errors and5xx never become that evidence. Refused writes
cannot acquire ownership through later matching content or observation.

Legacy plan1 is retained for saved replay, exact read-only recovery and narrowly
safe preactivation abandonment. A pending legacy phase cannot be dispatched or
silently converted into the corrected payload, even with source bytes available.
Unknown activation remains fenced. The original61 writer fixture and original
59/60 history preserve all rows and uncertainty across reopen. The actual60 and
61 readers refuse real populated62 without modifying the closed store; see the
[source evidence](diagnostics/2026-11-18-grubbery-app-store62-migration.json).
These gates extend development source evidence only. Matching final artifacts,
native corrected source following/state tests, installed upgrade and preserved
DEV qualification remain required; published releases and support pins stay
unchanged. Never relabel a user database or edit old receipts to downgrade.

## Development: immutable app phase journals, schema61

Schema61 transactionally adds Grubbery app creation order, immutable intent
headers, append-only phase events and exact namespace leases. Opening validates
ancestry and changes no native state. Unknown mixed schemas, missing tables,
receipts, events, clock entries or leases refuse; opening does not repair them.
Applied ownership is derived from original observed marker receipts rather than
an independently editable current-owner row. Completed or narrowly abandoned
preactivation operations release their exact lease without deleting staged code
or live app data. Unknown writes and dispatched activation remain fenced.

The development source gate preserves the actual0.4.0/schema59 fixture and an
actual d24c960/schema60 writer's populated state, including its linked Stop
resolution. Repeated open/replay, migration rollback, corruption and process death
after committed dispatch are covered. The genuine60 reader refuses populated61
without changing the closed database bytes. See the [source evidence](diagnostics/2026-11-18-grubbery-app-store-migration.json)
and `scripts/test-grubbery-app-upgrade.py`. The older RC.12/schema20 qualification
record and release pins remain frozen; its historical source tests now target61.
Synthetic compatibility tests explicitly omit only empty successor app tables;
they are not original-writer or user downgrade procedures.

The supported source-reader fixtures include the stable0.4.0/schema59 boundary
and the unreleased60 predecessor alongside the separately frozen older inventory.
This does not qualify installed59→61, interrupted native projects, managed app
state, DEV installation or a stable release. Build matching collector/runner
artifacts and rehearse the relevant non-destructive native/installed paths before
promotion. Never lower markers, remove tables or reset a user store to downgrade.
The live Grubbery workflow remains unadvertised pending the native adapter gates.

## Development: interrupted project Stop reconciliation, schema60

Schema60 adds separately attributed clean-stop reconciliation receipts and
pre-signal process bindings for new expanded-member stops. Migration preserves
all prior rows, grants no permissions, and signals/starts no runtime. Historical
omitted stop-intent fields remain omitted. Reconciliation preserves the original
interrupted operation and resolves only its exact fence through a new audited
receipt. Unknown, corrupt or missing resolution ancestry refuses unchanged.

The frozen synthetic schema59 fixture was written by actual0.4.0 source
`a9623329a538ebf83d63365053d2d1011ade091e`, with the declared test helper. The
original-writer rehearsal preserves every original table across repeated
schema60 opening and tests exact resolution replay after reopen. The actual old
reader refuses populated60 without modifying it. See
[project member recovery](project-member-recovery.md) and
`scripts/test-project-stop-recovery-upgrade.py` for reproducing these gates.

The later exact development snapshot at d24c960 passes a local Linux/AMD64
installed0.4.0/schema59→60 gate with two fresh ordinary fakes: running/stopped
intent, identity, configuration, history, access, engine invocation and the
running container process remain unchanged. The actual old reader refuses60
unchanged and the new manager restarts without touching that process. See the
[exact artifact evidence and limits](diagnostics/2026-11-17-issues-installed-upgrade-section.json).
That capsule stops later at a harness duplicate-enrollment refusal before its
recovery project. The same four frozen artifacts later pass a separate fresh
[native clean-stop recovery case](diagnostics/2026-11-17-issues-native-stop-recovery.json)
with exact reviewed reconciliation, store/manager reopen, original receipt
preservation, one SIGTERM, same-pier resume and unchanged peer. Neither case
qualifies migration of an already interrupted native project, managed application
state, desktop/DEV deployment, later artifacts or a stable release. Matching final artifacts and relevant disposable runtime
recovery/upgrade gates remain required before promotion.
Never lower the schema marker or reset a store to run an older manager.

## 0.4.0 candidate: installed0.3.7→0.4.0 qualification

The [exact-artifact rehearsal](releases/0.4.0/rehearsal.json) passes the local
Linux/AMD64 owner-user upgrade from schema47 to59, preserving running/stopped
intent, access, identity and history. The actual older runner refuses59 unchanged.
This supersedes the installed-upgrade deferral in the historical feature notes
below only for this documented path and artifact. Their individual fixture limits
remain valid. Manual password entry is left to the user's separate review VM;
remote service upgrades and live provider configuration remain outside this gate.
Stable publication and any production upgrade remain separate actions.


## Development: container resource limits, schema59

Schema59 fences optional RAM/CPU configuration and immutable launch limits.
Historical omitted values mean 4 GiB / 2 CPUs; migration rewrites no instance,
operation, application policy, frontend or ship-URL history. Invalid limits refuse
opening and roll migration back. Old clients omitting limits preserve the current
selection. Nondefault handoffs bind limits in the signed offer and require
`container-resources-v1` at both ends before fencing the source.

`scripts/test-container-resources-upgrade.py` uses the actual schema58 writer at
66c641b, preserves original receipts across repeated migration/reopen, retains
new resource settings, and verifies that the original writer refuses59 unchanged.
This is isolated feature evidence, not installed upgrade or release qualification.
Never lower the schema marker or reset a store to downgrade.


## Development: shared ship URLs, schema58

Schema58 adds current ship URL settings and contiguous authoring history keyed by
runner, target generation, native ship and verified pier identity. Migration invents
no URL, route approval or network configuration. Existing desk/source/receiver and
frontend records remain byte-preserved. Unknown or corrupt current/history state
refuses reopening; missing tables in an already58 store are not recreated.

`scripts/test-ship-routes-upgrade.py` uses the actual schema57 writer atbdb81b8.
Migration/reopen, rollback and that older reader's refusal pass with observed and
dispatched frontend uploads retained. Negotiate `ship-routes-v1` with matching
clients; never lower a schema marker. This is feature fixture evidence, not the
separately deferred installed upgrade/release qualification.

## Development: reviewed frontend uploads, schema57

Schema57 adds immutable `desk_frontends` linked atomically to desk operations.
Reviews bind the exact staged artifact/MIME digest, predicted native glob hash,
source revision, source excluding the generated docket, workflow settings and pier.
Receipts retain upload and owner-session cleanup separately. Missing/corrupt or
rebound records refuse; migration grants no permissions and sends no request.
Older step bodies remain byte-preserved through optional omitted frontend fields.

The actual schema56 writer at656e64d supplies the original fixture used by
`scripts/test-desk-frontend-upgrade.py`. Migration/reopen, rollback and original
older-reader refusal pass, including observed/pending receiver history and source
ownership. Negotiate `desk-frontend-v1` with matching collector/runner code. Never
lower a schema marker or reinterpret a lost upload as permission to send it again.
These are feature checks in synthetic state; installed upgrade/release qualification
remains deferred. No installed runner was changed.

## Development: reviewed receiver installation, schema56

Schema56 adds immutable `desk_receivers` linked to the existing operation journal.
The record binds the reviewed unused name, continuing publisher and expected hash,
project member generations, workflow settings and receiver pier. Migration creates
no install, policy grant or subscription. Journal/record writes are atomic, and
startup checks complete matching ancestry. Interrupted effects become uncertain;
recovery never repeats installation.

The actual schema55 writer atc912982 produces the original fixture used by
`scripts/test-desk-receiver-upgrade.py`. Prior policy, settings, source baselines,
deployment ownership, releases and adoptions remain unchanged. Preservation,
reopen, rollback, new receiver history and actual older-reader refusal pass.
Negotiate `desk-receiver-v1` with matching collector/runner code; never lower a
schema marker. These are development feature checks, not stable upgrade or release
qualification. No installed runner has been changed.

## Development: standing application/ship approval format1

M550 adds a separate `application-ship-approvals.json` file in the collector profile.
Existing runner schema55, `ships.json` and per-attempt application journals are
unchanged. Missing state means no standing permission. Existing approvals never
migrate into standing grants. Old collectors ignore the new file and retain their
explicit per-attempt workflow; they cannot use it to initiate automatic attempts.
New readers require version1, exact current/history agreement and contiguous
revision history. Unknown/corrupt state refuses unchanged, including writes.
Do not reset the file, drop revocations or convert a cached grant into authority.
M558 exposes explicit standing-approval review and authoring without changing
format1 or any attempt/session journal. Offline revocation remains possible after
a runner or plugin becomes unavailable. Public automatic connection and installed
migration qualification remain pending.

## Development: explicit source ownership adoption, schema55

Schema55 adds `desk_adoptions` records linked to the existing desk operation
journal. An adoption receipt and the selected source-ownership baseline commit in
one runner database transaction; no ship effect is dispatched. Existing ownership
rows keep their original encoding and use their referenced operation's type to
validate commit or adoption evidence. Missing, corrupt or rebound evidence refuses
startup and cannot be interpreted as an empty ownership baseline. Migration creates
no ownership, permission, ship effect or pending adoption.

The actual schema54 writer at829cfed supplies the checked-in fixture and
`scripts/test-desk-adoption-upgrade.py`. Tests preserve prior policy, settings,
source baselines, deployment ownership and release receipts; reopening a new
adoption and rollback pass. The actual older reader refuses55 without changing
state. Negotiate `desk-source-adoption-v1`; never lower a schema number for a
downgrade. This is development feature evidence, not release qualification.


## Development: reviewed activation/publication, schema54

Schema54 adds immutable `desk_releases` linked to the existing operation journal,
and observed application-stage evidence on release receipts. Original policy,
workflow, source baselines, deployments, ownership and history remain byte-preserved.
Migration does not invent release records or enable any gate. Opening checks every
release record against its journal; unknown, incomplete or inconsistent metadata
refuses reopen. Recovery never redispatches an unfinished release.

`scripts/test-desk-release-upgrade.py` exercises the actual schema53 writer at
`3a5a5c52a396089231d12998f7316a8d24a48ac9`, migration/reopen with new release
records, and that actual older reader's refusal without changing54 state. The
checked-in original53 SQL fixture additionally covers preservation and rollback.
Negotiate `desk-release-v1` with matching collector/runner code. Never downgrade
the database schema. This is development functionality and fixture evidence;
release qualification, supported stable upgrade artifacts and installed upgrades
remain separate and deferred.

## Development: verified source deployment, schema53

Schema53 adds immutable staged-commit reviews linked to the existing desk journal,
and verified source ownership linked to an observed completion. Migration preserves
policy, workflow, pending operation and canonical source-baseline records. Ownership
is not invented during migration. Completion and ownership write in one transaction;
unknown or inconsistent metadata refuses reopen. Original schema52 writer fixtures,
repeated migration, rollback and older-reader refusal are exercised by
`scripts/test-desk-deployment-upgrade.py` (M544). Runtime continuity uses the same pier
object, rather than a container ID, so a restart need not lose owned paths. Different
pier/source bindings start without ownership. The new CLI requires `desk-commit-v1`;
publication remains unavailable. This is functional source/fixture evidence, not a
release qualification or authorization to upgrade any installed runner.

## Development: private source staging

`desk-source-staging-v1` uses a versioned disposable `desk-stages-v1` runner cache;
SQLite remains schema52. It adds no live mount, automatic commit or policy grant.
Matching capability-aware clients verify frozen transfer receipts; older clients
cannot invoke it and existing workflow/policy/journal records are unchanged.
Cache metadata is strict and immutable. Interrupted uploads/discards retain their
original identity; unknown/corrupt contents fail visibly rather than being repaired.
Private API/collector/core fixtures cover recovery and quota/ownership boundaries
(M543). Managed-runtime deployment and release artifact qualification remain pending.

## Development: canonical source baselines, schema52

Schema52 adds a bounded canonical baseline cache, shared by collectors. A baseline
is recorded only after Unix and Clay match and is bound to the exact workflow and
runtime selection. Migration creates an empty cache; it does not infer source
equality or change existing workflow/policy/journal records. Original schema51
writer fixtures, repeated migration, corrupt-cache refusal and older-reader
refusal are exercised by `scripts/test-desk-source-upgrade.py` and the source tests
(M542). Negotiate `desk-source-status-v1` before using the live comparison API.
This remains development-source evidence, not exact-artifact release qualification.

## Development: workflow settings, schema51

Schema51 adds versioned ship/desk workflow metadata and append-only history.
Original schema50 writer state, including pending desk-operation receipts,
upgrades without changing its registry or policy rows. Repeated opens validate
the metadata and its history; failed migration rolls back. The schema50 writer
refuses actual schema51 state unchanged. The original-writer fixture and
`scripts/test-desk-settings-upgrade.py` record this narrow path (M541).
`desk-workflow-settings-v1` must be negotiated by matching collectors and runners.
Ordinary edits preserve these separate records. This does not qualify a released
0.3.7 → 0.4.0 artifact or authorize updating an installed runner.

## Development: typed desk operations, schema50

Schema50 adds separately journaled desk-operation intents and append-only history.
Original schema49 writer state upgrades without changing prior registry/policy
records; repeated opens validate the journal and never redispatch an interrupted
effect. Older schema49 writers refuse actual schema50 state without modifying it.
The original-writer fixture and `scripts/test-desk-upgrade.py` record this narrow
source compatibility path (M538). Discovery adds no persistent format; its
`desk-discovery-v1` capability and typed creation's `desk-create-v1` must be checked
before use. These are development fixtures, not the deferred exact-artifact
0.3.7 → 0.4.0 release qualification.

## Development: application permissions, schema49

The 0.4.0 [application policy layer](app-development-policy.md) adds runner
schema49, version2 policy and version2 collector rollout journals. Schema47
fixtures start with desk creation, commit and publication blocked. Schema48
fixtures preserve commit/publication grants and all original audit bytes, append
one migration receipt, and close desk creation. Reopening is idempotent; corrupt
audit receipts roll migration back. Historical database migrations remain tested.
Schema48 and earlier writers refuse schema49. Do not decrement `user_version` or
reset a store. Ordinary settings edits preserve policy. Legacy collector journal
reads preserve their bytes and report creation blocked; subsequent authoring
writes version2. Use matching artifacts and the `app-development-policy-v2`
capability. This is source/fixture qualification; the full exact-artifact
0.3.7 → 0.4.0 upgrade remains a release gate.

## 0.3.7 one-click setup and identity recovery

The [0.3.7 qualification](releases/0.3.7/qualification.json) covers the published
0.3.6 desktop and local runner upgrading to 0.3.7 on Linux/AMD64, schema 47.
The guided update preserves running ship and engine processes, stopped intent,
access and operation history across retry and a manager restart. See the
[release notes](release-0.3.7.md) for commands and support limits.

## Development: keyed first-boot identity correction

The local owner-only [identity repair](boot-identity-repair.md) preserves schema47,
existing instance/storage IDs, ports and operation history. A version1 opaque meta
journal supports interrupted correction; completed correction uses existing
initialized-runtime fields, so the currently installed manager can resume the same
pier. Synthetic commit-failure/replay/reopen and normal-resume tests cover this
targeted recovery, not a new release-wide upgrade claim. Its support window is the
current schema47/pinned keyed-runtime contract; older/future schemas are refused.

## Development: one-click native prerequisites

The `0.3.7-dev.20261105.1` local candidate adds explicitly authorized missing-package
installation to initial owner setup on Arch/Omarchy. Metadata inspection remains
read-only. The installer recomputes its package plan locally, rechecks the result,
then performs the existing native payload authentication and owner setup. It does
not add package installation to guided management upgrades or the offline bridge.
Runner schema 47, desktop state 1 and bootstrap API 1 are unchanged. Existing
OpenSSH-format Ed25519 desktop keys are read in place and reused; key generation
now uses the bundled SSH library. Source builds require Go 1.26 or later.

The earlier development candidate exercised fresh installation on the operator’s fleet.
The stable qualification above covers the subsequent exact-artifact upgrade.

## 0.3.6 local import preview patch

The [0.3.6 qualification](releases/0.3.6/qualification.json) records the exact
0.3.5 → 0.3.6 desktop/local service path and large logical preview regression.
The collector uses the existing 8 TiB local bound instead of the cross-host
100 GB bound for guided import and restoration previews. Schema 47, desktop
state 1, bootstrap API 1, runtime and pill are unchanged. No automatic import,
service restart or remote upgrade is implied by plugin publication. See the
[install steps and test limits](release-0.3.6.md).

## 0.3.5 performance release

The [0.3.5 qualification](releases/0.3.5/qualification.json) records the exact
0.3.4 → 0.3.5 desktop and local service upgrade path. Runner schema 47, desktop
state 1, bootstrap API 1 and the runtime image/pill are unchanged. Update the
plugin for bootstrap/UI/collector improvements, then **Update Dependencies** for
the runner's snapshot improvements. Earlier supported paths compose through
0.3.4. See [instructions and measurement limits](release-0.3.5.md).


## 0.3.4 macOS metadata compatibility release

The [0.3.4 qualification](releases/0.3.4/qualification.json) adds the tested
0.3.3 → 0.3.4 desktop and local service path. Runner schema 47, desktop state 1
and bootstrap API 1 are unchanged. Upgrade the runner to obtain the import fix.
The fresh offline rehearsal preserves existing running/stopped fakes and imports
a separate stopped fake with synthetic macOS companions, retaining every byte.
Earlier paths compose through 0.3.3. See [release instructions](release-0.3.4.md).

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


## Unreleased Theme replacement sidecar (M568)

The isolated Theme receiver retains application journal1 and transport journal2
for unrelated cached cleanup helpers. Replacement adds private version1
`application-replacements/active.json` plus immutable bound history receipts;
existing directory with missing/corrupt state is not silently reinitialized.
Request-specific Secret Service application namespaces retain old credentials
before normal-slot overwrite. No secrets enter JSON history. Pending installation
is fenced; only exact accepted ancestry admits a fresh independent successor.
Predecessor cleanup operates only on its original session and preserves the new
account. An original receiver journal writer and prior installer inventory are
covered in private source fixtures, not installed upgrade qualification.

Older predecessor helpers without replacement cleanup are not eligible for silent
handoff. Current helper approval, issuer lineage/migration and assembled recovery
qualification remain prerequisites before advertising automatic reconnection.
Do not delete sidecar/credential/transport history or downgrade formats to unblock
an attempt. See `diagnostics/2026-11-07-theme-session-replacement.json`.


## Unreleased DMs replacement and bridge coordination (M569)

DMs keeps account/attempt formats1 and reads transport1/2. Only explicit retirement
upgrades transport1 to2; original-writer preservation/old-reader refusal remains
qualified in private fixtures. Quiescence identifies the exact retired generation,
allowing it to remain alive after releasing its exclusive lease. It does not admit
unrelated live cleanup actors, resolve app writes or authorize a successor.

The new version1 replacement sidecar and request-specific Secret Service namespace
follow Theme's retention rules. Pending installation is fenced; only accepted exact
ancestry permits a new channel while old remote cleanup remains pending. Private
bridge coordination stores no credential and creates no socket file. An existing
bridge can adopt a successor without restart, but older cached helpers must not
be treated as replacement-capable. Source fixture evidence does not qualify an
installed/native upgrade. See `diagnostics/2026-11-07-dms-session-replacement.json`.


## Unreleased collector replacement ancestry (M570)

The collector reads application journals1–4. Explicit replacement publishes4
atomically with the predecessor retirement and successor intent; read-only open
does not rewrite prior state. Accepted successors link atomically, while original
origin/credential references and cleanup remain intact. Replacement receipts use
archive3; older archive1/2 receipts remain readable. Active attempts remain bounded
to64 and reachable ancestry to2048. Missing history, cycles and divergent active
branches are errors, not a reason to delete state or retry a login.

The actual previous3 collection sources are retained under
`internal/collection/testdata/v3writer` with source hashes. Private migration tests
use that writer, preserve its original evidence, and prove its refusal of actual4
state without mutation. Complete replacement requires declaration2 from the exact
approved helper; an installed update cannot widen a cached1 approval. Public
receiver advertisement and assembled qualification are still pending. Do not
downgrade journal markers or remove ancestry to unblock work.


M571 adds an optional `automatic_disabled` bit to unreleased journal4 replacement
metadata. Explicit disconnect persists cancellation before cleanup, including
unaccepted parent/child intent. It does not change a closed receipt already
published for archival or disable an already accepted successor when cleaning its
parent. Previous stable readers still refuse4. A new explicit reviewed retry can
clear a retired predecessor's cancellation; periodic maintenance cannot do so.
CLI and full registry/issuer fixtures cover the transition and interrupted archive.

## Development: recording attachment metadata2

The unwired recording store now admits only metadata2 for new captures, with
full attachment target/pier/engine/host/project-namespace identity. Version1
canonical metadata and original frame/header bytes remain readable, with missing
identity left unknown; no new capture or generation inference is allowed from it.
No runner database format changes or installed data rewrites are part of this
source change. The frame codec remains1, bound to the metadata's digest.

Focused source race checks pass. The original3218e6f writer/reader rehearsal is
prepared but pending; qualify its exact original files and refusal of successor
metadata before claiming format compatibility. Matching final artifacts, native
capture lifecycle/pressure/retention and installed upgrade remain required. This
is an unreleased development boundary, not a published recording support window.

The later [frozen5041406 original-writer gate](diagnostics/2026-11-18-recording-metadata2-source-upgrade.json)
passes the recording1→metadata2 source boundary: original raw/metadata/lifecycle
files survive repeated new-reader cursor/reopen, new legacy capture is refused,
and the actual original reader refuses metadata2 unchanged. The retained original
fixture covers subsequent source reads. This resolves only the earlier pending
original-writer rehearsal; all integration/native/installed requirements remain.

## Development: recording retention controls, schema65 — gates pending

Schema65 adds empty recording control event/head tables. Policy and exact incident
pin changes use canonical original actor/request receipts, a bounded hash chain
and a revision/digest CAS. Earlier managers must refuse rather than ignore these
storage controls. Opening preserves every prior row and performs no raw retirement
or native effect. Actual64 writer/refusal, original fixture migration, matching
artifacts and installed/active-worker upgrade gates are required before deployment.
Source control storage alone does not qualify deletion, automatic retention or
pressure recovery. Never relabel or remove history to run an older manager.


M701's actual64 writer/current65 source gate and pinned fixture now pass; see
[the scoped report](diagnostics/2026-11-18-recording-schema65-source-upgrade.json).
M702 adds intent2 within development schema65, binding each new launch to its
selected control revision/digest. Intent1 encoding and historical receipts stay
unchanged; historical reads/replay do not check current settings or grant fresh
Start authority. New fresh authority requires the exact current control head.
Old readers that cannot understand an intent must refuse without repairing or
stripping it. Frozen native pilot2d028197 predates controls and their APIs; final
matching artifacts and actual active-worker/native upgrades remain mandatory.

## Development: read-only project allocation — persistent formats unchanged

`project-allocation-read-v1` and shared `project.allocation.v1` add bounded,
operator-private current member storage observations. Schema65 and
original rows/receipts remain unchanged. Growth samples are bounded volatile
memory with monotonic completion times, discarded on manager restart. Missing
capability or an unsupported project phase refuses explicitly. A census never
resolves an operation fence, reserves free space or supplies containment authority.
See the [contract and exclusions](project-allocation.md). Source fixtures do not
qualify matching native measurements or an installed/preserved DEV upgrade. A
future durable incident latch or guard requires its own persistent contract,
old-reader refusal and non-destructive upgrade evidence before deployment.

M736 adds `ledger_kind` to this unshipped read contract, distinguishing an exact
saved expansion from a bounded original-ready history projection. Derivation
preserves all stored rows and does not create an expansion or change a phase.
Matching development tools are required; no persistent schema or downgrade
contract changes. Original replacement/recovery adapters and native project
measurement/installed-upgrade qualification remain pending.

## Development: derived timing index/query — persistent formats unchanged

The optional `recording-export --timing-index` writes a separate derived JSONL
entry in a private ZIP; it does not place a sidecar in immutable recording storage.
The default original exporter retains its existing one-pass behavior. Shared
`runtime.recording-timing.v1` queries use the existing original manifest/part
capability and authorization, with no new runner endpoint, background job,
retention effect or schema65 migration. Stable source identity excludes a volatile
read timestamp and binds exact immutable content; continuation refuses changes.
Focused source/interface checks do not qualify current native indexed exports,
matching artifacts, installed upgrades or preserved DEV. Complete redacted
bundles and pressure/retirement remain separate work.

M739 adds complete derived redacted ZIPs through distinct shared manifest/part
operations and an explicit `--redacted` CLI choice. Both derived representations
use the same bounded original-frame verifier; original/index regression checks
remain required after refactoring. Exact binding/body omissions and escaped
views never rewrite raw storage. No sidecar/cache, persistence schema or new
retention/runtime authority is added. Every derived part re-verifies/re-derives
the source and exact ZIP. Earlier callers remain usable; matching current
tools/reference/native export and installed/preserved DEV gates remain required.
