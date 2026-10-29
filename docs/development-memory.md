# Omabit development memory

Keep stable decision IDs, dates, status, evidence and explicit supersession links.
A newer passing experiment does not erase a failed run or broaden its claim.
Update this log as development proceeds; do not store credentials here.

## M001 — Product and ownership (2026-10-22, current)

Omabit is the ship runner; Tend is separate at thelifeandtimes/tend. The product
anchor is “Multiplayer apps without a subscription”. This supersedes the retired
monorepo's four-workstream description for current platform development.

## M002 — Production exclusions (2026-10-22, current)

Protect managed ~lismyl-lismyl absolutely. Never inspect or operate tmux ships or
anything under ~/temp-ur. Preserve installed plugins and production services.
Use fresh disposable ships, isolated engines/namespaces, and separate experimental
plugin IDs, state, credentials and desktop adapters. This supersedes historical
reset/deployment permissions. See experiment-isolation.md and protected-ships.md.

## M003 — Authentication and application ownership (2026-10-22, current)

Prove the contract with Theme and DMs. Omabit provides optional authentication;
plugins communicate directly with Eyre and support standalone URL/+code login.
Give each plugin its own session, while acknowledging that current Eyre sessions
are not desk-scoped. Backend installation belongs to the plugin; discovery belongs
to Omart. Do not grow an Omabit app store. Private Ethereum chains are the first
happy path, with stable/dev/edge profiles isolated across all state and side effects.
These user decisions supersede the initial Tend-first/shared-data-provider framing.
Wallet/key storage is research only; same-user plugins are not a key-security boundary.

## M004 — Fake transport protocol (2026-10-22, current)

Pinned Vere 4.6-5a7614f uses loopback for fake-galaxy discovery. Shared private
network namespaces pass; a bridge-only topology is not qualified. Direct foreign
Eyre pokes failed HTTP500 because Gall refuses foreign poke-as. Use a typed local
Khan thread to send an ordinary Gall poke and verify its bound acknowledgement.
This supersedes the channel-based peer probe; previous failed reports remain valid.
See diagnostics/2026-10-22-fake-network-channel-trial.md and project-fake-networks.md.

## M005 — Docker restart result (2026-10-22, current)

The two-ship Docker test passed messages in both directions, deliberate rejection,
and same-pier replacement-container recovery with original sessions and namespace.
Resume omits creation flags -F/-B/-c. This supersedes the stale “transport unproven”
summary, not the failed historical trials. It does not prove persistent application
subscriptions, power-loss recovery, anchor recovery or project integration.
Evidence: diagnostics/2026-10-22-docker-peer-recovery.md.

## M006 — Goal execution and next evidence (2026-10-22, execution current; test status superseded by M014/M019)

The user explicitly resumed and requested continuous progress through checkpoints
unless actually blocked. A checkpoint is not a request for another approval.
The goal service still reports an old blocked state; its API cannot resume it.
Continue authorized work and record evidence. The next test uses a disposable Gall
state fixture, queued subscription update, same-pier receiver restart, original
channel/cookie reconnection and fresh subscription reconciliation without replaying
writes. Implementation in progress, no pass claimed yet.

## M007 — Fixture setup transport (2026-10-22, partially superseded by M008/M010)

Native state control12 timed out on Lens `|mount %base`, before state writes or
subscriptions. Both fakes stopped cleanly. Lens expression reads worked in earlier
gates, but a mutating Dojo command did not return usable completion here. Replace
this fixture setup with acknowledged typed Gall pokes and read-only readiness
checks. This supersedes only the initial fixture setup assumption; it is not a
failure of persistent Gall state. Retain /tmp/omabit-thread-probe-20261022-12.

## M008 — Acknowledged fixture setup (2026-10-22, superseded by M010/M011)

Control13 proved the replacement mount and commit requests and read-only Clay
readiness, then returned HTTP500 for the fixture start thread. Both fakes stopped
cleanly. Error diagnostics now request JSON for setup, including bound success
receipts and structured failures. Control14 is fresh; no recovery pass claimed.
This narrows M007: mount/commit are resolved, app startup remains under test.
Original control13 retained beside control12 with hashes and compatibility links.

## M009 — Recovery evidence scope (2026-10-22, current)

Clean process restart proves persistent ship/application state only when read back.
It does not imply an agent upgrade invoked on-save/on-load. Bounded SSE connections
prove channel reconnection and subscription delivery; they do not preserve a TCP
socket across process death. A fresh snapshot must reconcile reads without replaying
writes. Theme currently treats subscription facts as invalidations and re-scries
after each gap; that existing source pattern still needs real write/reconnect tests.

## M010 — Fixture startup diagnostic scope (2026-10-22, resolved by M011)

Controls14/15 still failed at start despite JSON setup responses. Read-only Lens
proved the fresh base extra-app map is empty. A minimal fixed rein poke is now
allowed only behind that explicit empty-map check. Do not apply it as a general
installer: preserve existing overrides. The fixture helper was moved outside the
ten public Gall arms. These two corrections are under test; neither is a proven
root cause yet. This supersedes M008's expectation that JSON alone would reveal a
structured start error. See diagnostics/2026-10-22-application-state-trials.md.

## M011 — Concrete fixture compile failure (2026-10-22, current)

Clay's `%ca` endpoint builds an app source file as a vase; `%cb` builds a mark and
was the wrong diagnostic. Diagnostic03 exposed `-find.$` at the duplicate-entry
guard in omabit-state.hoon. Replace the malformed negation/double-call expression
with `?< (lien ...)`. On diagnostic04, changing that guard in a private copy of the
stopped test pier made the same commit/start requests return HTTP200 and a scry
return revision0/empty entries. Both diagnostic03 and 04 stopped cleanly.
This resolves M010's fixture-start investigation; no evidence supports blaming
Docker, foreign transport, or Clay map access for that setup failure. A fresh full
recovery run is still required. Original failing piers remain preserved.

## M012 — Log bounds must not limit pier files (2026-10-22, current)

Do not apply a low process-wide RLIMIT_FSIZE to Vere to cap a diagnostic log: it
also restricts the pier/database files. Diagnostic02's private clone exited during
startup with that mistaken limit; retain it as failed evidence, not a clean-stop
pass. Use the normal harness's bounded pipe capture and outer deadline. This
supersedes the rejected diagnostic-only file-limit approach. Original source piers
and all production data were untouched.

## M013 — Generic noun pokes across Ames (2026-10-22, current)

Fresh control17 compiled/started the fixture and received the initial subscribe
ack/snapshot. Its first peer write was rejected with `nest-fail`, `-have.*`,
`-need.@t` at the input cast. The generic noun mark does not preserve the caller's
cord aura across the remote conversion. Validate the received noun's atom shape
with `;;(@t q.vase)` rather than expecting a typed @t vase. This supersedes the
fixture's typed-vase input assumption; it is not a networking failure. Both fresh
fakes stopped cleanly. Full state/reconnect acceptance remains pending control18.

## M014 — Native state/subscription recovery (2026-10-22, current)

Fresh control18 passes all four peer writes, persisted state, queued event delivery
on the original channel after same-pier receiver restart, continued updates, and
fresh-subscription reconciliation without write replay. Original sessions survive;
both channels and ships clean up correctly. This supersedes M013's pending native
acceptance status and resolves the fixture work in M006–M011. It does not supersede
those retained failures or qualify Docker yet. Evidence and exact limits:
diagnostics/2026-10-22-application-state-native.md. Proceed to the private Docker
VM gate, then duplicate-identity project isolation, without asking for another
checkpoint approval.

## M015 — Next project isolation gate (2026-10-22, provisional)

The four-fake VM harness is prepared but unrun. It keeps A live while B repeats the
same identities/ports in another namespace, then checks crossed cookies and writes
against independent state. Offline checks reject false namespace, liveness, cookie
and state evidence. Increased memory admission applies only to that future private
VM; do not overlap it with the current two-ship state VM. Runner integration and
endpoint publication remain separate design/qualification work. Existing storage
project labels are not durable network IDs. See project-fake-networks.md and the
qualification README for constraints; these notes are not shipped capabilities.

## M016 — CLI uncertain acknowledgement (2026-10-22, current)

The six-task private simulator walkthrough passes an intentionally lost response
after acceptance. Snapshot reconciliation and exact terminal request replay find
one original operation, with no duplicate work. Keep the original request/actor/
effect binding; a missing reply does not establish rejection. This supersedes the
transport-loss-pending portion of the earlier CLI checkpoint. It is simulator
proof only; the first proxy trial's omitted protocol header was a harness failure.
See diagnostics/2026-10-22-cli-lost-reply.md. Structured guidance is the next step.

## M017 — Machine-readable recovery navigation (2026-10-22, current)

Collector errors now optionally include local `guidance` beside the unchanged
error/code: review_current_revision, reconcile_uncertain_request, or
resolve_runner_identity. Typed client uncertainty preserves causes; never infer
it from remote text. Wrapped identity mismatch takes precedence, offering local
reads only. Guidance performs no recovery or retry. The built six-task private
simulator and affected race/vet checks pass. This supersedes M016's pending
structured-guidance status. See diagnostics/2026-10-22-cli-recovery-guidance.md.

## M018 — Skill distribution ownership (2026-10-22, current)

Keep the operator/developer SKILL.md and references together. Release/development
packagers now add a separate source-bound, checksum-covered skills archive; no
agent configuration or installed skills change automatically. Historical exports
without skills keep their original assets. Four focused packaging fixtures pass;
six existing source-isolation checks pass with two full-build/channel checks
explicitly skipped. Frozen end-to-end release packaging remains unqualified.
This supersedes the source-only distribution plan, not the older published release.
See agent-skills.md. Capability discovery remains necessary with older binaries.

## M019 — Docker state/subscription recovery (2026-10-22, current)

Guest16/appliance15 passes the persistent application and subscription gate on the
pinned runtime/private rootless engine. Four unique peer writes, all 13 observations,
original session/channel/cursor recovery after same-pier container replacement,
fresh reconciliation and both channel deletions independently match. Clean shutdown
and original artifacts are verified and retained. This supersedes the Docker-pending
status in M014 and the current-work portion of M006; native/failed evidence retains
its original scope. It does not qualify the newer project-isolation source, runner
integration or endpoint publication. See diagnostics/2026-10-22-application-state-docker.md.

## M020 — Bind delayed DMs work to its original channel (2026-10-22, current)

Offline tests reproduced three stale-session bugs: old backoff cleanup deleting a
new channel, queued ACK targeting the current channel, and late contact results
assigning before the generation check. The experimental DMs bridge now binds
mutations to generation plus channel, serializes reset, and checks before state
assignment. All 24 affected tests pass. This supersedes any assumption that the
previous account-generation checks alone covered reconnect races; it does not
supersede earlier frozen authentication passes or qualify the new real transport.
See diagnostics/2026-10-22-dms-stream-generation.md. Full profile tests remain open.

## M021 — Qualify publication before choosing topology (2026-10-22, provisional)

Loopback-only namespace transport does not prove host endpoint publication. A
separate no-NIC guest case is prepared using HTTP responders and two private
rootless bridge projects. It checks exact IPv4 guest-loopback mappings, identity,
cross-project reachability and anchor replacement/recovery. Native responder and
offline mapping fixtures pass; Docker behavior remains unqualified. Start with
internal bridges; a separately recorded egress-bridge experiment may be needed.
Never repair an anchor by silently stranding running members in its old namespace.
This supplements M015/M019 and replaces no runtime result. See the fake-network
qualification README. No current host publication or runner behavior changes.

## M022 — Real plugin transport control (2026-10-22, provisional)

A new native pair case stages distinct Theme/DMs `auth-lab` artifacts. It exercises
actual settings writes/watch and DMs acknowledgements, invitations, live events
and history before/after receiver restart, without collector data mediation.
Five evidence-rejection tests and private import/profile preflight pass; no runtime
write/reconnect result yet. Keyring persistence and desktop application are inert
for this control and keep their separate acceptance gates. Temporary storage is
admitted/monitored in addition to the existing eight-GiB scope. This supplements
M020; it does not supersede any frozen authentication result.

## M023 — Remaining native profile collisions (2026-10-22, current finding)

Source inspection still finds Theme's fixed `urbit-theme` IPC target and canonical
hook names/plugin path in Service.qml and client/native.py. Namespaced session
metadata alone cannot authorize enabling experimental copies beside the user's
current plugin. Desktop application itself needs explicit collection ownership.
Keep current runtime controls headless/inert and qualify IPC, hooks, launchers and
watcher teardown before stable/dev/edge collection activation. This confirms the
existing profile limitation; no installed desktop was inspected or changed.

## M024 — Actual plugin writes/reconnect passed (2026-10-22, current)

Native plugin control01 passed four unique DMs, both histories/live events, two
exact appearance writes/facts, original sessions, fresh subscription channels and
independent logout after same-pier receiver restart. Originals and 37 frozen inputs
are verified and retained. This supersedes M022's unrun status and qualifies M020's
stale-stream artifact for this transport case. Desktop, persistent credentials,
hosted compatibility and collection profiles keep separate gates. See
diagnostics/2026-10-22-plugin-transport-native.md. Later source changes are not
covered by this frozen pass.

## M025 — DMs requires an explicit positive receipt (2026-10-22, current)

Four new offline tests reproduced eleven failed subcases: missing/unrecognized
positive acknowledgements were reported as completion, bool/float IDs could match
an integer request, and a late duplicate overwrote the first result. Experimental
DMs now requires an integer ID and exact `ok`, preserves the first receipt, and
keeps malformed outcomes uncertain without replaying writes. All 28 affected tests
pass. This supersedes the permissive receipt-handling assumption, not M024's
frozen runtime evidence. A fresh runtime control is required for this new source.

## M026 — Local-chain compatibility is more than an RPC URL (2026-10-22, research)

Pinned Vere separates gateway/local/fake options. The reference Arvo `%local`
path uses fixed contract addresses and chain ID 1337 tied to Bridge deployment.
Verify the actual pill and deployed code/state; arbitrary Anvil/Ganache defaults
are not a qualified ship devnet. Bridge's convenience scripts include broad
process cleanup and wildcard binding, so use owned children in a private boundary
instead. No scripts were run or keys stored. This refines the private-chain plan
without choosing a shipped implementation; see private-chain-research.md for
source hashes, primary references and evidence limits.

## M027 — Strict receipt artifact qualified on native peers (2026-10-22, current)

Native control02 repeats the complete plugin pass with M025's stricter DMs code.
Independent checks now bind all four DM writes to exact wire poke IDs/channels,
positive receipts, receiver events and both histories. Two Theme writes, original
sessions, new subscriptions and independent logout also pass. Originals and
37 frozen input hashes are retained. This supersedes M025's pending runtime status
and M024's newer-source limitation, without changing their historical evidence.
See diagnostics/2026-10-22-plugin-receipts-native.md; persistent keyring/native
desktop/profile/hosted gates remain separate.

## M028 — Distinguish ship credentials from runner state (2026-10-22, current)

Direct Eyre error hints now distinguish origin/identity refusal, unavailable
credential storage and absent/rejected sessions. Use local ships/offline help
without runner selection; no fallback, credential replacement or retry occurs.
Identity/store failure takes precedence over login advice, and transport failure
is not classified as expired authentication. Collector race/vet and JSON/text
CLI fixtures pass, preserving enrollment with no post-enrollment HTTP dispatch.
This extends M017; ship-status keeps its separate result shape. See
diagnostics/2026-10-22-ship-error-guidance.md, including corrected fixture assumptions.

## M029 — Duplicate identities stay within their Docker projects (2026-10-22, current)

Run17 passed two simultaneous zod/nec pairs in distinct loopback-only namespaces.
Each recovered state/subscriptions and original sessions after receiver replacement.
Four crossed same-identity cookies were refused; interleaved fifth writes/readbacks
kept project state disjoint. Original images and 17 extracted files are verified
and retained. This supersedes M015's provisional runtime status and M019's pending
project-isolation gate. Publication, anchor lifecycle and runner integration remain
separate; liveness is source-asserted rather than a duplicated raw inspect trace.
See diagnostics/2026-10-22-docker-project-isolation.md. No production access occurred.

## M030 — Preserve failed endpoint evidence before changing topology (2026-10-22, current)

Endpoint trial18 failed before any member/HTTP request on a publication mismatch,
then failed its sleep-anchor clean-stop assertion. The old fixture omitted raw
mapping/stop observations, so it cannot diagnose exact Docker behavior. Originals
and 18 extracted files are verified and retained. The successor records both and
uses init for anchor signal forwarding; its internal-bridge rerun is pending.
This supersedes M021's unrun status, not its open publication/lifecycle gate.
See diagnostics/2026-10-22-endpoint-trial18.md. Do not turn clean VM poweroff into
an application-test pass or silently discard the failed fixture's history.

## M031 — Discover and identify administrative endpoints (2026-10-22, source-qualified)

Runner code retrieval now reads a bounded, safe `.http.ports` file from the selected
pier and checks Lens identity before and after requesting a code. Endpoint/inventory
changes discard results; no fallback to12321 or cached port is used. Default/named
fake/comet and refusal race tests plus vet pass. This supersedes the unchanged
hardcoded12321 implementation noted in project-fake-networks.md, while preserving
that historical finding. The newer Go path requires its own real-runtime gate;
earlier Python transport controls do not qualify it. See
diagnostics/2026-10-22-admin-endpoint-discovery.md. Public internal8080 and single-
instance networking remain unchanged until project integration.

## M032 — Internal bridge has requested but inactive publications (2026-10-22, current)

Corrected trial19 records exact requested loopback mappings but null active ports
on the running anchor's sole internal bridge in rootless Docker29.9.1. It cannot
satisfy this direct-publication contract. Init fixed clean anchor shutdown in this
successor (exit143, no cleanup error). Images and 18 files are verified/retained.
This supersedes M030's unresolved publication diagnosis for the corrected source
and rejects M021's internal-bridge candidate on this pinned stack. Next is a
separate non-internal bridge inside the no-NIC VM; production topology remains
unselected. See diagnostics/2026-10-22-endpoint-trial19.md.

## M033 — Published endpoints require coordinated anchor recovery (2026-10-23, current)

Ordinary-bridge fixture run20 passed28 HTTP observations, four exact loopback
publications, four IPv6 refusals and four cross-bridge refusals. Restarting the same
anchor replaced its namespace; old members stayed in the old namespace. Explicit
member stop/replacement restored published endpoints while the second project
remained unchanged. Nine clean stops, VM poweroff and retained artifacts/receipts
were independently verified. This supersedes M021's pending publication fixture
and M032's pending ordinary-bridge experiment, preserving both failed internal-
bridge trials. Actual ships on this topology, runner integration and external
reachability remain unqualified; the VM has no NIC. See
diagnostics/2026-10-23-endpoint-publication.md and project-network-contract.md.

## M034 — Direct published fake traffic is a separate runtime gate (2026-10-23, prepared)

The new single-project publication case sends actual Eyre requests directly from
guest loopback, with independent in-memory cookies and bounded SSE reads. Lens
setup remains inside the selected namespace. All32 offline network checks pass;
the fresh appliance and sources are hash-verified. This extends M033 without
promoting its HTTP fixture to Urbit evidence. Actual execution, published duplicate
projects, real-ship anchor replacement and runner integration remain pending.
See diagnostics/2026-10-23-published-preparation.md. No production access occurred.

## M035 — Completed Start is not yet a login-code selection (2026-10-23, current)

Trial21 passed fresh import/healthy Start but immediately requested a code before
the first resource observation named the container. Existing admission refused;
zero access audits prove the endpoint/code backend was not reached. Original
images and43 extracted receipts are verified/retained. The fixture now polls an
exact fresh selection before its one reveal, with10 offline checks passing. This
supersedes the old fixture assumption; M031's real-runtime gate remains open.
Do not diagnose code-discovery timeout from this trial or weaken admission to make
it pass. See diagnostics/2026-10-23-login-selection-trial21.md.

## M036 — Plan endpoints before persisting project ownership (2026-10-23, source-qualified; CLI gap superseded by M118)

The pure project planner validates fresh distinct galaxies, explicit unreserved
host ports and shared-administration consent; it produces deterministic internal
ports and a canonical review digest. Race checks cover input ordering, reservation
conflicts, duplicate identities and altered contracts. It has no persistence, CLI
capability or engine effects. Schema22 source/binary are retained before any later
migration. This implements a prerequisite of project-network-contract.md, not
runner project support. M035 also informed new offline login-code selection help;
existing help and skill-package checks pass. See diagnostics/2026-10-23-project-planner.md.

## M037 — Published Eyre preserves application state and subscriptions (2026-10-23, current)

Run23-01 passed50 direct guest-loopback Eyre requests, four durable writes and13
ordered observations across same-pier receiver replacement. Existing sessions,
queued subscription recovery, a new subscription and independent logout passed.
Exact publications, IPv6/Lens refusals, clean stops/poweroff and retained hashes
were independently verified. This supersedes M034's prepared-only status and
M033's actual-Urbit-publication gap. Published duplicate projects, ship anchor
replacement and runner integration remain open; it is not an external-network or
hostile-member isolation claim. See diagnostics/2026-10-23-published-state.md.

## M038 — Project ownership is durable before runtime dispatch (2026-10-23, source-qualified)

Schema23 atomically reserves complete reviewed projects with stable IDs and
actor-scoped replay. Standalone IDs/ports and project claims share admission;
corrupt claims refuse opening/allocation. Race suites and vet passed. The frozen
schema22 writer reproduced its fixture and its reader refused actual schema23
project reservations unchanged. Full older-writer regeneration hit a missing
local preview18 object; checked-in migration tests passed. This supersedes M036's
schema22/persistence-pending status only. No RPC/capability or runtime dispatch is
implemented, and no live database was upgraded. See diagnostics/2026-10-23-project-store.md.

## M039 — Compare published projects with memory-only foreign sessions (2026-10-23, prepared)

The combined published/isolation harness now assigns disjoint guest-loopback ports
to two zod/nec pairs. Foreign-session refusal uses ephemeral in-memory handles;
cleanup preserves original cookies even on failure. Both pairs must stay live for
independent writes. All37 offline checks and appliance23-03 hash verification pass.
This prepares M037's published-duplicate gap without claiming runtime completion.
See diagnostics/2026-10-23-published-pairs-preparation.md. No production access.

## M040 — Fresh login selection reaches a still-failing backend (2026-10-23, current)

Trial23-02 selected the correct fresh container after two observations, then
recorded one failed code-access audit. This resolves M035's fixture-selection gap
but does not pass M031's code-retrieval runtime gate. The stage/cause is unknown;
4-second timeout is only a hypothesis. Original images and44 receipts are verified
and retained. The successor adds fixed-stage/outer-deadline-only audit diagnostics,
with all timeouts and identity checks unchanged; race/vet checks pass. A fresh
runtime diagnostic remains required. See diagnostics/2026-10-23-login-backend-trial02.md.

## M041 — Actual-ship anchor recovery requires all old members stopped first (2026-10-23, prepared)

The new separate anchor-recovery gate preserves old-container/namespace evidence,
requires public refusal after anchor restart, then stops every old member before
replacing either. Original piers/sessions and a queued subscription must recover,
followed by another peer write. All41 offline network checks and appliance23-05
hash verification pass. This prepares the actual-ship extension of M033, without
superseding its fixture-only qualification or claiming runner support. See
diagnostics/2026-10-23-anchor-preparation.md.

## M042 — Login code was discarded at the final identity deadline (2026-10-23, current)

Trial23-04's audit confirms `identity-after` with the outer deadline expired,
superseding M040's unconfirmed timeout hypothesis. The secret was correctly not
returned. Original disks/44 receipts are verified and retained. The source fix
allows12seconds for the verified read and15seconds for only its client/response;
ordinary RPCs remain5seconds and all checks/per-call limits remain. Race tests
include a slow real Unix HTTP response past5seconds, prompt caller cancellation,
no replay and redaction; vet passes. The real-runtime success gate remains open.
See diagnostics/2026-10-23-login-deadline.md.

## M043 — Real runner login survives same-pier replacement (2026-10-23, current)

Fresh VM23-06 passed both exact code selections, direct authenticated scries and
session revocation across a same-pier replacement container. Two successful access
audits, unchanged revision, preserved import tree, final Stop and clean poweroff
were independently verified; original disks and51 receipts remain retained. This
supersedes M042's runtime-fix-pending status and closes M031's basic runner gate.
Relocated admin ports remain source-tested only; shared-project runner integration
is separate. Ordinary client timeouts remain unchanged. See
diagnostics/2026-10-23-login-restart.md. No production access occurred.

## M044 — Exact container identity does not preserve its network namespace (2026-10-23, source-qualified)

Project assessment now binds complete reviewed ownership, revision/generation,
engine/resource contracts, exact publications and boot-bound namespace identity.
It retains old membership and requires recovery after an anchor namespace change;
missing resources never imply a clean stop. Race tests/vet pass for incomplete,
foreign and stale evidence. This implements part of M033's recovery contract,
without superseding fixture/runtime limits. It performs no engine calls and is
not persisted; runtime journaling still needs a schema fence and driver integration.
See diagnostics/2026-10-23-project-observation.md.

## M045 — Journal returned resource IDs before inspecting them (2026-10-23, source-qualified)

Schema24 now fences durable bridge/anchor preparation: planned → dispatched →
created with exact returned ID → observed. Losing an inspection result must not
lose the creation ID; losing the create reply remains unresolved dispatch.
Atomic mirrors, sequence/operation guards, eight restart boundaries and source
race/vet tests pass. The original schema23 writer reproduces populated ownership;
its reader refuses actual schema24 pending dispatch unchanged. This supersedes
M038's schema23-current status and M044's preparation-persistence gap only. No
engine driver, member launch or RPC/capability exists yet. See
diagnostics/2026-10-23-project-journal.md; earlier evidence and limits remain.

## M046 — Dispatch and inspection follow separate durable commits (2026-10-23, source-qualified)

The private preparation controller/SQLite worker now enforce intent-before-create,
returned-ID-before-inspection and engine checks around resource calls. Lost replies,
errors with IDs, cancellation and mismatches preserve uncertainty without replay or
inverse actions. Inert-engine race tests pass, including real SQLite readback inside
calls. The Unix engine probe retains an exact optional daemon ID; new project work
requires a matching ID, while historical report behavior stays compatible. This
supersedes M045's missing private-controller wiring only. No real Docker adapter,
server scheduling or member lifecycle is qualified. See
diagnostics/2026-10-23-project-dispatch.md. Schema22→24 original-reader checks pass.

## M047 — The pinned anchor image declares a data volume (2026-10-23, source finding)

Digest-verified OCI config `d6cc5107…b245` for the pinned Vere image declares
`/urbit` as both a volume and working directory, plus image-level ports80/tcp and
34343/udp. Reusing this image as a sleeping namespace anchor must account for the
volume explicitly; a stopped anchor is not evidence that all generated storage
is owned or cleaned up. Prefer an explicit bounded ephemeral mount for the anchor,
then inspect its exact mount contract in the private VM before integration.
Image EXPOSE entries alone are not host publications. Retained config/evidence:
`2026-10-23-anchor-image` under task qualification evidence. This adds a resource-
ownership requirement to M046; it does not supersede earlier transport passes or
claim runtime volume behavior/cleanup qualification. No Docker engine was accessed.

## M048 — Prepare only fresh owned bridge and stopped-anchor resources (2026-10-23, source-qualified)

The private Docker adapter now binds verified owner/daemon/image, exact returned
IDs and fixed resource settings to the journal controller. Fresh setup creates
metadata only and requires a locally installed pinned image. Inert-engine race
checks and vet pass for lost replies, changed ownership, unknown attachments and
mounts. A bounded `/urbit` tmpfs addresses M047's declared image volume in source;
actual engine behavior remains unqualified. This supersedes M046's missing-adapter
source status only. Server scheduling/fences, members and recovery are unfinished.
See diagnostics/2026-10-23-project-adapter.md. No production access occurred.

## M049 — Published duplicate projects retain independent state and sessions (2026-10-23, current)

VM23-03 passed114 direct published requests across two independent zod/nec pairs,
with five writes/17 observations per pair, same-pier subscription recovery and
four cross-project same-identity session401 refusals. Clean ship/anchor stops and
poweroff passed. Journald split the large final result; the original driver failure
is retained. Strict reconstruction equals the disk receipt, and all original
hashes/17 extracted receipts were verified. The bounded parser now has source tests.
This supersedes M039's runtime-pending status and closes M037's published duplicate
gap only. Actual-ship anchor recovery and runner integration remain. See
diagnostics/2026-10-23-published-project-isolation.md.

## M050 — Count queued project workers before admitting management updates (2026-10-23, source-qualified)

Private server admission now holds the runner mutex across receipt/worker creation,
counts queued work for updates and takes the shared resource gate before session
setup. It rechecks ownership and retained maintenance there. Inert race tests cover
queue cancellation, setup failure, replay, busy updates and successful release.
This supersedes M048's missing private server-fence wiring only; no public route or
startup scheduling exists. A guarded real-adapter VM test is compiled, skipped on
host, and being packaged. Actual-engine behavior and members remain separate gates.
See diagnostics/2026-10-23-project-fence.md. Production remains untouched.

## M051 — A late client failure must not erase a returned resource ID (2026-10-23, source-qualified)

The adapter now preserves an exact complete create ID even with client failure or
cancellation, allowing the controller to journal it and stop. Ordinary command
errors still discard output; missing/malformed IDs remain unresolved. Source race
and redaction regressions/vet pass. Initial anchor validation now distinguishes
never-started `created` resources from stopped former processes and refuses unknown
tmpfs options. This refines M048/M050 without runtime qualification. Prepared
appliance23-07 predates these checks; keep its frozen identity and replay its later
receipt against current pure validators before claiming compatibility. See
diagnostics/2026-10-23-project-refinement.md. No production engine was accessed.

## M052 — Restarted forwarding may return no HTTP response without connect refusal (2026-10-23, failed trial/source fix)

Actual-ship VM23-05 failed at its first post-restart endpoint assertion after
proving stopped-anchor refusals and changed namespace with old members retained.
The exact failing curl outcome was not recorded; do not retroactively label it52.
The verified prior endpoint20 fixture did return52/status000 in this situation.
The successor probe now records bounded facts and requires a recognized transport
failure, status000 and no body. All43 offline tests pass. Trial05's18 receipts and
original disks remain; ships/poweroff stopped cleanly but channel/session cleanup
was incomplete. This supersedes M041's unexecuted status, not its unpassed recovery
gate. See diagnostics/2026-10-23-anchor-trial05.md. A fresh retry is needed.

## M053 — Real preparation preserves journal ownership without starting ships (2026-10-23, current)

VM23-07 passed real bridge/stopped-anchor creation through the private SQLite
worker/controller/adapter, exact-ID inspections, duplicate refusal and unchanged
receipt/journal after reopen. No instance/pier or anchor process was started.
Explicit `/urbit` tmpfs configuration created no persistent volume. The17 receipts,
original disks and extracted database were independently verified; current pure
validators also accept the frozen runtime inspections. This closes M048's basic
real-adapter gate and M047's creation-stage volume concern, not Start/mount runtime,
public RPC, real server concurrency, members or recovery. M051's error-path tests
remain source-only. See diagnostics/2026-10-23-project-preparation-runtime.md.

## M054 — Anchor Start needs its own receipt and complete namespace inventory (2026-10-23, source-qualified)

The pure initial-Start controller now requires completed preparation, distinct
operation identity, intent-before-Start and fresh post-ack namespace/publication
proof. It never treats anchor completion as member readiness or retries uncertainty.
The pure inventory resolver includes indirect container joins that bridge endpoint
lists omit, refusing missing/ambiguous/cyclic references. Race tests and vet pass.
This narrows M053's missing Start-model gap only; persistence and actual Start
adapter remain unfinished. See diagnostics/2026-10-23-anchor-start-model.md.

## M055 — Persist initial anchor Start separately from creation (2026-10-23, source-qualified)

Schema25 now retains a distinct Start journal and operation mirror, preserving
completed preparation. Private workers commit dispatch/ack/namespace in order;
reopen interrupts unfinished work without replay. Inert race tests cover every
boundary, corruption, atomic rollback, competing claims and lost replies. The
original schema24 snapshot reproduces populated preparation and refuses actual
schema25 dispatched Start unchanged. This supersedes M054's persistence gap and
M045's schema24-current status only. Actual Start adapter/server scheduling,
members, recovery and release qualification remain. See
diagnostics/2026-10-23-anchor-start-journal.md.

## M056 — Recover every old member after the anchor namespace changes (2026-10-23, current)

VM23-08 independently passed actual-ship published anchor recovery:65 requests,
six durable writes/19 observations/nine facts; unchanged piers and original sessions
and subscription recovered after all old members stopped before replacements.
The restarted anchor retained its container ID but changed namespace; old members
remained in the old namespace. New empty publications returned52/status000 until
recovery. All channels/sessions and processes cleaned up; clean poweroff1349.52s.
Sixteen receipts and original disks remain. This supersedes M052's unpassed actual
recovery gate, not its failed outcome or M033's limits. Runner member recovery is
still unimplemented. See diagnostics/2026-10-23-anchor-state-recovery.md.

## M057 — Inspect all namespace joins before initial anchor Start (2026-10-23, source-qualified)

The private Start adapter now verifies original metadata/engine/image contracts,
complete bounded container joins, stable before/after observations and boot-bound
namespace/publication evidence. The server queues it behind common resource and
update fences; setup failure retains planned intent. Inert race tests and vet pass.
This supersedes M055's missing adapter/scheduling source status only. Appliance23-09
is verified and running its separate real anchor-only test; no runtime pass yet.
Member lifecycle/public routes remain. See diagnostics/2026-10-23-anchor-start-adapter.md.

## M058 — Qualify helper paths against the pinned image (2026-10-23, failed trial/source fix)

Actual anchor-only trial09 retained an acknowledged Start as interrupted because
`/usr/bin/stat` is absent. The fresh anchor stopped cleanly and VM powered off in
229.64s;19 receipts, SQLite and disks remain verified. Digest-verified OCI layers
place BusyBox stat at `/bin/stat`; its short-option probe passes in private
Bubblewrap. The source is corrected and race/vet checks pass. This supersedes
M057's runtime-pending status with a failed result, not a Start pass. The running
mount/publication contract separately replays successfully. Exact two-field Docker
OOM normalization is documented. See diagnostics/2026-10-23-anchor-start-trial09.md.

## M059 — Shared-network members need separate launch and cleanup ownership (2026-10-23, source-qualified)

Standalone managed execution owns a per-instance bridge and assumes internal8080;
pointing it at a shared anchor would violate peer ownership. The first pure member
invocation builder now uses each reviewed internal port and `<ship>` pier leaf,
separates fresh boot from resume flags, and passes focused race tests. This is a
new member-integration slice, not runtime authority or a modification of standalone
ships. Storage/creation/readiness/cleanup remain. See
diagnostics/2026-10-23-member-command.md; M056's actual recovery ordering still applies.

## M060 — Private runner initial anchor Start passes on the pinned image (2026-10-23, current)

Fresh VM23-10 passed real queued Start with SQLite dispatch/ack/namespace ordering,
exact loopback publications and empty complete namespace inventory. Reopen/replay
preserved the original creation receipt and completed Start. No member/pier ran;
the anchor stopped143 without OOM and poweroff took230.38s. Seventeen receipts,
database and original disks remain independently verified. Full managed race tests
passed77.457s. This supersedes M058's fresh-retry requirement and M057's unqualified
initial Start path, not trial09's failed evidence. Member lifecycle/recovery and
public interfaces remain. See diagnostics/2026-10-23-anchor-start-runtime.md.

## M061 — Bind a member container to its exclusive parent and namespace (2026-10-23, source-qualified)

The private contract binds runtime/member/operation/generation, completed anchor
namespace and recorded mount identity into a creation digest. Temporary-filesystem
checks retain partial/replaced/locked piers; inspection tests reject peer mounts,
implicit volumes and independent networking. Resume has no boot flags/pill mount.
Race regressions and vet pass; the initial incomplete synthetic checkpoint fixture
is recorded. This extends M059's command-only slice without engine dispatch or a
schema change. Durable member storage/claims, fresh inventories and lifecycle still
remain. See diagnostics/2026-10-23-member-contract.md.

## M062 — Journal member mkdir separately and reserve its exclusive parent (2026-10-23, source-qualified)

Schema26 retains accepted member/path/runtime claims, dispatch, created directory
identity and observed completion. Late failure preserves known identity; reopen
interrupts without replay/adoption. Legacy storage moves/imports cannot overlap.
Temporary-filesystem and race tests pass; full managed race80.555s. The original
schema25 writer reproduces completed Start state and refuses real schema26 member
dispatch unchanged. New snapshots exclude redundant nested archives while keeping
their hashes and originals. This supersedes M055's schema25-current status and
M061's missing storage-claim slice, not M060's frozen runtime pass. Server scheduling,
actual owner/VM integration and member lifecycle remain. See
diagnostics/2026-10-23-member-storage.md; initial fixture failures remain recorded.

## M063 — Queue member storage behind the same resource/update fences (2026-10-23, source-qualified)

Private queue tests pass cancellation/setup/maintenance failure, actor replay and
busy-update refusal. Setup cannot bypass the shared gate. The guarded VM case uses
the real filesystem driver for two empty member parents, with SQLite readback at
dispatch/inspection and reopen checks. All host guards skip,13 offline tests and
project race/vet checks pass. Appliance23-11 verifies61 source hashes and its embedded
test; the no-NIC VM is running, not yet qualified. This supersedes M062's missing
server-scheduling source gap only. No public route or ship launch exists. See
diagnostics/2026-10-23-member-storage-queue.md.

## M064 — Real queued storage preserves two empty member parents (2026-10-23, current)

Fresh VM23-11 passed real schema26 queue/filesystem preparation, SQLite ordering,
replay refusal and unchanged journals after reopen for both runtime-bound parents.
Offline ext4 inspection independently matches inode/birth,1000:1000 ownership,
0700 permissions and empty contents. No member/pier ran; only the stopped anchor's
container directory remains. All17 receipts and original disks are verified;
clean poweroff217.13s. This supersedes M063's pending VM and M062's unqualified
owner/runtime storage gate only. Container lifecycle/readiness/recovery remain;
new anchor namespaces require new evidence, preserving the original Start receipt.
See diagnostics/2026-10-23-member-storage-runtime.md.

## M065 — Observe actual member namespaces without adopting or rebinding (2026-10-23, source-qualified)

The read-only private adapter verifies exact journaled member contracts and the
complete bounded namespace inventory, bracketing member reads with unchanged
engine/anchor/bridge evidence. Unknown joins and in-flight changes refuse; a
coherent old member namespace remains explicit evidence for recovery. Created
containers have no running namespace. Twenty-two inert scenarios, focused race
regressions and vet pass. This supersedes M061's missing namespace-observation
slice only; it does not qualify dispatch, storage-use checks or readiness. No
schema/receipt change or runtime access. See diagnostics/2026-10-23-member-observation.md.

## M066 — Saved mount sources are a conflict check, not historical mount proof (2026-10-23, source-qualified)

The private bounded inventory rejects current object/path overlaps, aliases,
unresolved volume sources and changing/incomplete observations. Only an exact,
fully verified never-started member gets its own-mount exception. Twenty-four
inert scenarios, ancestry/replacement tests, race regressions and vet pass.
This adds M065's separate mount-conflict slice; it does not supersede lifecycle
ownership, locks or kernel-mount uncertainty. A saved source path can outlive the
object originally mounted. No schema/runtime change. See
diagnostics/2026-10-23-member-mounts.md. Durable member launch remains next.

## M067 — Journal each initial member execution effect before dispatch (2026-10-23, source-qualified)

The pure launch model binds original storage/anchor receipts and exact peers,
orders workspace/Create/Start intent and returned evidence, and retains known
identities after late failure. Complete fresh observations require the original
namespace and peer contracts. Twenty-four controller outcomes, invalid-binding
tests, race regressions and vet pass. This adds the controller slice after M065/
M066; persistence and actual dispatch remain. `running` means container execution,
not initialized storage or Urbit readiness. No schema/runtime change. See
diagnostics/2026-10-23-member-launch-model.md.

## M068 — Persist initial member execution without rewriting earlier receipts (2026-10-23, source-qualified)

Schema27 journals accepted launch, fixed binding and workspace/Create/Start phases
with atomic operation mirrors. Reopen interrupts without replay; serial project
admission preserves exact peer ownership. Nine boundaries, competing claims,
rollback/corruption and SQLite-readback worker tests pass. Full managed race90.780s.
The original schema26 snapshot reproduces completed anchor/two-parent storage and
refuses real schema27 launch dispatch unchanged. This supersedes M062's schema26-
current status and M067's missing persistence slice only. Concrete adapter/queue,
VM runtime and ship readiness remain; running-container success is not readiness.
No installed state changed. See diagnostics/2026-10-23-member-launch-store.md.

## M069 — Initial member dispatch uses fresh private artifacts and shared fences (2026-10-23, source-qualified)

The concrete adapter copies an already staged pinned pill into a new owned
workspace, then binds Create/Start to fresh namespace, parent, artifact and mount-
conflict checks. Known identities survive late failure; partial piers refuse boot.
The private queue holds the common work/update gates and preserves setup failures.
Eleven inert engine/filesystem scenarios, queue tests, race regressions and vet
pass. This supersedes M068's missing adapter/queue source slice only. No downloads,
public routes, schema change or real runtime occurred. Private-VM and readiness
gates remain. See diagnostics/2026-10-23-member-launch-adapter.md.

## M070 — Real member launch requires guarded queue and cleanup evidence (2026-10-23, preparation-qualified)

The new private-VM fixture launches two members through the actual queue, observes
committed SQLite phases, preserves replay/reopen receipts and requires exact-ID
ownership checks before member-first/anchor-last cleanup. Pill staging is explicit.
Ordinary guards, focused race regressions, vet, 96 source hashes and the embedded
appliance binary pass. This supersedes M069's missing VM fixture only; it does not
claim real execution or durable readiness/Stop/recovery. Sources and preparation
proof are retained. See diagnostics/2026-10-23-member-launch-vm-preparation.md.

## M071 — Readiness must bind current identity to stable initialized storage (2026-10-23, source-qualified)

A pure read-only model now reconstructs the full serial launch set, brackets a
credential-free public ship-identity check with unchanged complete runtime and
pier/`.urb` observations, and returns only complete fresh evidence. Its historical
receipt cannot be reused as current authority. Forty-five inert cases, race
regressions and vet pass. This adds the readiness model missing in M069/M070;
concrete collection, persistence and real readiness qualification remain. It does
not supersede any runtime evidence. See diagnostics/2026-10-23-member-readiness-model.md.

## M072 — Trial12 preserves stale dispatch and failed early shutdown (2026-10-23, failed real trial)

The first queued member (nec) reached a committed running-container receipt; the
second (zod) was refused at create-dispatched because final preflight expired.
No second container was created. First-member cleanup exhausted 300 seconds and
exited 137 without OOM; the anchor and VM stopped cleanly. Independent SQLite,
exact-container inventory and evidence hashes confirm the failure, and all
original disks remain retained. Repeated hashing inside the freshness window
needs a stable verified-file check; retain the 30-second bound. The fixture also
omitted the previously passing --init/zod-first pattern. Shutdown cause is not
isolated. This supersedes M070's pending trial status only, not any earlier pass.
See diagnostics/2026-10-23-member-launch-trial12.md.

## M073 — Hash immutable artifacts before the final runtime freshness window (2026-10-23, source/preparation-qualified)

Workspace/preflight still hash the pinned pill, bracketing full file metadata.
Final Create/Start check the identical in-memory verified snapshot without another
full hash; the 30-second runtime/preflight bound is unchanged. Thirteen artifact
cases and concrete post-observation tampering cases pass with race regressions and
vet. The member contract now requires --init; the fresh VM fixture boots zod first,
matching earlier qualified cases. This supersedes M061's no-init contract and
M069's repeated-dispatch-hash implementation only, while preserving their older
source evidence and all failed M072 disks. Appliance13 was independently verified
(100 sources); its runtime remains pending. Shutdown causality is still unisolated.
See diagnostics/2026-10-23-member-launch-refinement.md.

## M074 — Readiness collection performs only bound observations (2026-10-23, source-qualified)

The concrete collector reconstructs complete original member contracts, checks
parent and pier/`.urb` identities, and gates credential-free host identity reads
on fresh complete running-project evidence. Seventeen injected-engine and fresh
filesystem cases plus race regressions/vet pass. No mutation or download command
is available through the collector. This supersedes M071's missing concrete
adapter source slice only; durable storage/queue and real readiness qualification
remain. VM13's earlier frozen binary cannot qualify this collector. See
diagnostics/2026-10-23-member-readiness-adapter.md.

## M075 — Trial13 reached second Create but still refused stale Start (2026-10-23, failed real trial)

The refined immutable-artifact dispatch let nec reach Create after zod launched.
Start then refused stale preflight, leaving a retained exact ID, start-dispatched
journal and independently verified never-started container. Zod's early cleanup
again exited 137 after 300 seconds without OOM, despite --init/zod-first. The
anchor and VM shut down cleanly; all original disks and failed receipts are
preserved. This supersedes M073's pending runtime status and the sufficiency of
its cleanup candidates, not its passed source tests or M072's retained failure.
Batch repeated exact-container reads and expose freshness ages next. No clean
pair or readiness pass is claimed. See diagnostics/2026-10-23-member-launch-trial13.md.

## M076 — Batch exact-container reads without weakening freshness (2026-10-23, source/preparation-qualified)

The observer batches full inspections of recorded IDs while retaining all member
contracts, complete inventories and repeated namespace checks. Malformed batches
and invalid selections refuse; reordered valid responses pass. Dispatch errors
report bounded freshness ages. Focused race suites and vet pass; appliance14's
102 source files and embedded test were independently verified. This supersedes
M075's pending batching source work only. Trial14 is pending, both original failed
trials remain retained, and early-boot cleanup is unresolved. See
diagnostics/2026-10-23-member-observation-batch.md.

## M077 — Retain every readiness attempt without replaying launch (2026-10-23, source-qualified)

Schema28 records separate planned/observing/observed attempts and mirrors their
complete original launch binding. Restart interrupts unfinished observations;
exact actor/request replay returns the original receipt, while a new read-only
attempt preserves the old one. Completed evidence is historical, never current
runtime authority. Twenty-four focused cases, full managed race tests and vet
pass. Frozen original schema27 fixtures reproduce exactly and its reader refuses
real schema28 state unchanged. This supersedes M074's missing persistence slice
only; private scheduling and real readiness remain. VM14 stays schema27. See
diagnostics/2026-10-23-member-readiness-store.md.

## M078 — Schedule readiness behind the existing runner gates (2026-10-23, source-qualified)

The private queue shares work/update admission, validates complete original launch
bindings, and retains interrupted setup/cancellation failures. Five queue cases,
combined project race suites and vet pass. The guarded VM fixture now checks real
queued readiness, replay and reopened receipt equality; ordinary and explicitly
enabled host guards refuse runtime access. This supersedes M077's missing private
queue and readiness fixture only. No public route, startup task or real readiness
pass is claimed; VM14 stays the earlier schema27 binary. See
diagnostics/2026-10-23-member-readiness-queue.md.

## M079 — Trial14 passed dispatch but failed cold-boot/cleanup gates (2026-10-23, failed real trial)

Both members reached committed running receipts with unchanged namespaces and no
stale-dispatch recurrence. The final HTTP wait expired; nec cleanup exited137
without OOM, while zod, anchor and VM stopped cleanly. Independent SQLite/disk
checks and all original evidence are retained. Earlier VM08 needed about405 and
585 seconds for serial cold boots; trial14's480-second final wait and900-second
whole-operation budget were not a suitable baseline. This supersedes M076's
pending runtime status and M075's outstanding stale-dispatch recurrence only.
There is no complete readiness/cleanup pass or proven shutdown cause. Use serial
identity warmup with sufficient bounded time, keeping ownership freshness intact.
See diagnostics/2026-10-23-member-launch-trial14.md.

## M080 — Budget cold boot separately from ownership freshness (2026-10-23, source/preparation-qualified)

The next fixture uses serial exact-identity warmup with 900 seconds per ship and
2400 seconds for setup/launch/readiness, followed by separately bounded cleanup.
Five HTTP fixture cases plus readiness regressions/vet pass. Runtime ownership
and preflight freshness remain 30 seconds; CPU, memory and network isolation are
unchanged. Appliance15's frozen sources and embedded binary are verified. This
supersedes M079's inadequate fixture ordering/time budget and the final login-page
wait from M070, not its failed runtime evidence. Real schema28 readiness remains
pending. See diagnostics/2026-10-23-member-readiness-preparation.md.

## M081 — Graceful Stop preserves intent, exit evidence and peer state (2026-10-23, source-qualified)

The initial Stop model requires fresh exact running ownership and unchanged
initialized storage, commits intent before one graceful signal, and retains
uncertainty without automatic retry, force or removal. A completed observation
must preserve process start identity, pier objects and every peer/project field;
nonzero/OOM exits remain recorded failures. Inert interruption, tampering, peer
and cancellation cases plus project race regressions/vet pass. This adds the Stop
model missing in M067/M078; persistence, concrete dispatch and runtime qualification
remain. It does not supersede any real shutdown failure. VM15 lacks this model.
See diagnostics/2026-10-23-member-stop-model.md.

## M082 — Persist Stop once per exact container and preserve uncertain outcomes (2026-10-23, source-qualified)

Schema29 journals accepted bindings, dispatch, acknowledgement and exit separately.
Only a clean acknowledged worker result finalizes success; unclean exits remain
failed, and lost/canceled/late outcomes remain interrupted with evidence intact.
Replay cannot re-signal, and unresolved project Stops block another Stop. Restart,
concurrency, tampering and worker tests plus full managed race/vet pass. Original
schema28 fixtures reproduce exactly and its reader refuses actual schema29 state
unchanged. This supersedes M081's missing persistence only. Concrete Stop, queue,
recovery and runtime gates remain; VM15 is frozen schema28. See
diagnostics/2026-10-23-member-stop-store.md.

## M083 — Real queued launch and durable readiness pass with serial cold boot (2026-10-23, runtime-qualified)

Private VM15 passed in1231.94 seconds: both original member launches, exact public
identities, queued complete readiness proofs, unchanged replay/reopen receipts,
and member exit0/anchor/guest cleanup. Independent database, filesystem and artifact
checks pass; all original disks are retained. Measured serial warmup was340.14 and
508.19 seconds, with runtime freshness still30 seconds. This supersedes M080's
pending runtime gate and M078's missing real readiness evidence, and completes the
initial-launch/fixture-cleanup gate absent in M072/M075/M079. Those failed originals
and their unproven root causes remain recorded. This does not qualify schema29
Stop, resume/recovery or public routes. See diagnostics/2026-10-23-member-readiness-runtime.md.

## M084 — Bind graceful Stop to exact live and exited processes (2026-10-23, source/preparation-qualified)

The observer retains bracketed full process metadata; exited containers require
an explicit recorded start and unchanged original contract. The concrete Stop
adapter rechecks fresh ownership/storage, sends one explicit TERM, and waits
read-only without force/removal. Completed peer exits remain exact, and the private
queue shares update/maintenance gates with durable phase-before-effect callbacks.
Thirty adapter cases, six queue cases, affected race suites and vet pass. Guarded
VM Stop/replay/reopen coverage and appliance16 are prepared and verified. This
supersedes M082's missing concrete adapter/queue source slice only. Real durable
Stop and recovery remain pending; M083's readiness result and all earlier failed
trials remain unchanged. See diagnostics/2026-10-23-member-stop-preparation.md.

## M085 — Observe an uncertain Stop without inventing its acknowledgement (2026-10-23, source-qualified)

A separate read-only model/adapter records fresh exact-process exit evidence while
preserving the original dispatched Stop and its uncertainty. Clean/nonzero/OOM
outcomes remain observations; no retry, re-signal, removal or resume follows.
Known exit metadata cannot change. Twenty-two model and ten concrete inert cases,
race regressions and vet pass. This fills the read-only recovery-model slice left
open in M082/M084; persistence, scheduling and real recovery remain unimplemented.
It supersedes no runtime result. VM16 predates this code. See
diagnostics/2026-10-23-member-stop-exit-source.md.

## M086 — Persist recovery observations without rewriting Stop (2026-10-23, source-qualified)

Schema30 retains separate read-only attempts and atomic receipt mirrors. Restart
interrupts unfinished observation; replay and new attempts preserve prior evidence.
Fresh exact-process completion can record clean or unclean exit without changing
the original uncertain Stop or its admission fence. Focused tests, full managed
race tests and vet pass. Original schema29 fixtures reproduce exactly and the old
reader refuses real schema30 state unchanged. This supersedes M085's missing
persistence slice only; queue, real recovery and resume remain. VM16 stays frozen
schema29. See diagnostics/2026-10-23-member-stop-exit-store.md.

## M087 — Real queued graceful Stop preserves both ships and receipts (2026-10-23, runtime-qualified)

Private VM16 passed in1293.24 seconds: nec then zod each received one committed
graceful Stop, exited0 without OOM, and retained exact process/pier/peer evidence.
Replay, repeat-dispatch refusal, reopen and clean anchor/guest cleanup passed.
Independent disk/SQLite/artifact verification passed; every original is retained.
This supersedes M084's pending real initial Stop gate, not M085/M086's recovery
limits. VM16 is schema29 and cannot qualify lost-reply recovery, resume or public
integration. See diagnostics/2026-10-23-member-stop-runtime.md.

## M088 — Queue read-only recovery with original Stop uncertainty intact (2026-10-23, source/preparation-qualified)

The private queue shares work/update/maintenance gates and carries exact completed
peer exits. Seven queue cases, affected race suites and vet pass. The guarded
fixture deliberately discards the last member's actual Stop reply, then observes
its exit separately while preserving interrupted dispatch and replay/reopen.
Host guards and appliance17's127-source embedded binary verification pass. This
supersedes M086's missing queue and M085's missing runtime fixture only. Real
recovery remains pending; peer lifecycle changes, resume and public access remain
unsupported. VM16's earlier pass is unchanged. See
diagnostics/2026-10-23-member-stop-exit-preparation.md.

## M089 — Retire only an exact cleanly stopped container before replacement (2026-10-23, source-qualified)

A separate explicit intent commits removal before one exact-ID command, retains
acknowledgement and complete absence separately, and preserves piers, peers and
all old receipts. No force/volume removal, retry or replacement is implied.
Thirty-four model and nineteen injected-adapter cases, race suites and vet pass;
temporary state files retain inode/bytes. Transport cancellation discards its
reply, so that removal remains uncertain. This adds the retirement source slice
missing from M087/M088's resume work; persistence, queue and real retirement remain.
No earlier runtime result is superseded. VM17 predates this code. See
diagnostics/2026-10-23-member-retirement-source.md.

## M090 — Persist retirement boundaries without replay or lost history (2026-10-23, source-qualified)

Schema31 records one retirement per exact member/container, with immutable clean
exit binding, atomic mirrors and separate dispatch/acknowledgement/absence phases.
Restart, concurrency, replay, invalid completion and worker tests pass; original
Stop/exit receipts stay unchanged. Full managed race tests and vet pass. Original
schema30 fixtures reproduce exactly and its reader refuses actual schema31 state
unchanged. This supersedes M089's missing persistence slice only; queue, real
retirement and replacement remain. VM17 stays frozen schema30. See
diagnostics/2026-10-23-member-retirement-store.md.

## M091 — Fence retirement within its project while preserving replay (2026-10-23, source-qualified)

The private queue shares work/update/maintenance gates and preserves the original
contract chain. Previously accepted work blocks retirement; unresolved retirement
blocks new same-project admission and worker execution. Old receipts remain
replayable and unrelated projects are unaffected. Seven queue/fence cases, affected
race suites, historical fixtures and vet pass. Two added adapter cases cover exact
stopped-peer retention, for21 total. This supersedes M090's missing private queue
slice only; no real retirement, replacement or public access is qualified.
See diagnostics/2026-10-23-member-retirement-queue.md.

## M092 — Observe a real exit while retaining the uncertain Stop (2026-10-23, runtime-qualified)

VM17 independently passed the private read-only recovery gate after deliberately
discarding zod's actual Stop reply. Original dispatch remains interrupted; the new
clean exit observation, replay and reopen preserve both receipts. Original disks
are retained. The outer shell returned143 for an unknown reason despite complete
child exit0/clean shutdown; this discrepancy remains recorded separately.
This supersedes M088's pending real recovery gate only. Changed peers/namespace,
retirement, replacement and public integration remain unqualified. See
diagnostics/2026-10-23-member-stop-exit-runtime.md.

## M093 — Hash saved state around explicit container retirement (2026-10-23, preparation-qualified)

The guarded fixture adds one queued retirement after clean exit recovery, retains
original receipts, checks replay/reopen and compares stopped log/checkpoint bytes
before/after. Absent-target cleanup does not send another Stop. Host refusal,
compilation/vet and appliance18's136-source embedded-binary verification pass.
This fills M091's missing runtime fixture only; real retirement and replacement
remain unqualified. See diagnostics/2026-10-23-member-retirement-preparation.md.

## M094 — Resume requires a new container intent over the retained pier (2026-10-23, source-qualified)

The pure replacement model preserves prior retirement/Stop/exit receipts and
commits Create/Start before effects. Returned IDs survive late errors; unfinished
attempts cannot replay. Fresh exact pier, peer and namespace proofs are required.
Thirty fault/dispatch and nine invalid-intent cases, race regression and vet pass.
Previous generation still needs concrete adapter/store binding; no runtime resume
is qualified. This adds the resume model missing from M091/M093, superseding no
runtime evidence. See diagnostics/2026-10-23-member-resume-model.md.

## M095 — Recheck namespace before replacement Create and Start (2026-10-23, source-qualified)

The first-generation resume adapter derives generation2 with the same initialized
pier and no bootstrap flags/pill. It brackets complete ownership and repeats mount,
storage and namespace checks before each dispatch. A failed injected case exposed
a missing pre-Create namespace check; it is fixed and its log retained. All24
adapter cases, full affected race suites and vet pass; synthetic bytes/inodes stay
intact. This supersedes M094's missing concrete first-replacement adapter only.
Persistence, general generations and real resume remain. See
diagnostics/2026-10-23-member-resume-adapter.md.

M095 evidence refinement: same-phase Create/Start reentry is now explicitly tested
and refused, strengthening the earlier post-completion duplicate case. The focused
race check passes; separate hashes preserve both test versions in its report.

## M096 — Real retirement preserves every saved state byte (2026-10-23, runtime-qualified)

VM18 independently passed queued retirement after clean read-only exit recovery.
Seven log/checkpoint files totaling1,016,848,396 bytes retain their names, modes,
sizes and hashes, verified directly from the guest disk. Pier identities, peer
metadata, original receipts, replay/reopen and complete absence pass. Original
disks are retained; clean shutdown/scope inactivity and outer exit0 are confirmed.
This supersedes M093's pending real retirement gate only. VM17's unexplained
outer143 remains recorded; actual replacement remains pending. See
diagnostics/2026-10-23-member-retirement-runtime.md.

## M097 — Persist replacement intent without replaying uncertain effects (2026-10-23, source-qualified)

Schema32 consumes an initial retirement once, preserves all old receipts and
atomically records the new contract and Create/Start phases. Restart retains
unfinished attempts as interrupted; same-project admission stays fenced. Restart,
worker, tamper, concurrency and full managed race tests pass. The original31 fixture
reproduces exactly and its reader refuses actual32 unchanged. This supersedes
M095's missing first-replacement persistence only; general generations and real
resume remain. See diagnostics/2026-10-23-member-resume-store.md.

## M098 — Schedule replacement through the common runner gate (2026-10-23, source-qualified)

The private queue retains the original contract chain and completed peer exits,
shares work/update/maintenance fences, and interrupts cancellation or setup errors.
An altered generation returned by the backend is refused before dispatch. Six
focused race cases and vet pass. This supersedes M097's missing scheduling slice
only; actual replacement, general generations and public access remain. See
diagnostics/2026-10-23-member-resume-queue.md.

## M099 — Qualify a replacement against retained shutdown state (2026-10-23, preparation-qualified)

The guarded schema32 fixture queues the first replacement, preserves original
receipts, checks replay/reopen, verifies HTTP identity and records state hashes after
clean shutdown. Host refusal, compilation/vet and appliance19's145-source embedded
binary verification pass. This fills M098's missing runtime fixture only; real
replacement, durable readiness and general generations remain unqualified. See
diagnostics/2026-10-23-member-resume-preparation.md.

## M100 — Observe the exact replacement process before calling it ready (2026-10-23, source-qualified)

Read-only readiness binds the completed replacement, unchanged pier and process
start time, brackets credential-free identity with complete metadata, and allows
recorded stopped peers. Project and member generations remain distinct.29 model
and22 adapter cases plus24 replacement regressions pass; full affected race suites
and vet pass. Fixture mistakes and their failed logs are retained. This adds the
readiness model/adapter missing from M098/M099; persistence and real readiness are
not yet qualified. See diagnostics/2026-10-23-member-resumed-readiness-source.md.

## M101 — Retain readiness attempts independently of replacement history (2026-10-23, source-qualified)

Schema33 records read-only replacement readiness without rewriting the original
resume. Interrupted observations retain their own receipts and may be followed by
a new request. Seventeen persistence and six private-queue cases, full managed
race tests, vet and original32 reproduction/downgrade refusal pass. This supersedes
M100's missing persistence/scheduling only; runtime readiness and general generations
remain. See diagnostics/2026-10-23-member-resumed-readiness-store.md.

## M102 — Resolve member generations by causal receipts (2026-10-23, source-qualified)

The pure history resolver follows exact prior bindings, creator operations and
generation counters, retaining old receipts and rejecting missing/branched history
or resource reuse. Four valid histories reach generation3 in either input order;
ten invalid histories are refused. Full model race tests and vet pass. This adds
a selector toward the general lifecycle gap noted in M101; it does not supersede
first-replacement-only runtime/store limits. See
diagnostics/2026-10-23-member-generation-selection.md.

M102 implementation follow-up: current Stop admission still selects initial
readiness and fences the original interrupted Stop. Generalized lifecycle work
must distinguish a retired prior generation from unresolved current work using
retained exit/retirement evidence, without rewriting the old interrupted receipt.
This is implementation work, not a need for production access or user permission.

## M103 — First real replacement resumes the retained pier (2026-10-23, runtime-qualified)

VM19 independently passes queued generation2 Create/Start with no bootstrap flags
or pill mount, the original pier/namespace, fixture HTTP identity, unchanged old
receipts, replay/reopen and clean shutdown. Seven post-Stop state files totaling
1,016,852,492 bytes match the guest disk. All17 extracted files and receipt/contract
checks pass; original disks are retained, scope inactive and outer exit0. This
supersedes M099's pending first-replacement runtime gate only. Schema33 readiness,
general generations and runner application recovery remain. VM17's outer143 caveat
remains. See diagnostics/2026-10-23-member-resume-runtime.md.

## M104 — Verify durable readiness after the real replacement (2026-10-23, preparation-qualified)

The guarded schema33 fixture adds queued read-only readiness, exact collection
order, unchanged replacement history, replay and reopen to the first replacement.
Host refusal, compilation/vet and appliance20's156-source embedded-binary check
pass. This fills M101's missing real-runtime fixture only; actual durable readiness
remains pending. VM19's prior first-replacement pass remains distinct. See
diagnostics/2026-10-23-member-resumed-readiness-preparation.md.

## M105 — Separate historical Stop validity from current action authority (2026-10-23, source-qualified)

New replacement Stop intent requires the latest causal binding and its matching
readiness receipt. Historical Stop validation uses the exact old prefix after
validating the whole history, so later replacements do not erase prior evidence.
Generation2/3 execution, stale selection and prefix tests pass, with full model
race/vet checks. This extends M102’s resolver; it does not supersede first-replacement
adapter/store limits. See diagnostics/2026-10-23-member-resumed-stop-model.md.

## M106 — Rebuild current member contracts before Stop (2026-10-23, source-qualified)

The private replacement Stop adapter reconstructs exact current creation contracts
and accepts stopped peer evidence only while that peer retains its recorded
generation. It reuses the committed single-TERM path and checks replacement pier
identity throughout.33 inert-engine cases and the full adapter race suite pass;
state bytes remain intact. This supersedes M105’s missing Stop adapter only.
Persistence/scheduling and actual current-generation Stop remain unqualified;
readiness after unrelated peer replacement remains further work. See
diagnostics/2026-10-23-member-resumed-stop-adapter.md.

## M107 — Keep old Stop uncertainty while admitting the current generation (2026-10-23, source-qualified)

Schema34 persists replacement-bound Stop and schedules it through the shared gates.
Historical validation keeps original receipts; completed causal retirement/resume
evidence accounts for old containers without rewriting their interrupted Stops.
20 persistence and6 queue cases, full managed race tests and vet pass. Original33
fixtures reproduce and its reader refuses real34 unchanged. This supersedes
M102/M106’s missing current Stop persistence/scheduling slice only. Later resumes,
changed-peer readiness and runtime qualification remain. See
diagnostics/2026-10-23-member-current-stop-store.md.

## M108 — Give distinct lifecycle fixture steps distinct request IDs (2026-10-23, failed trial retained)

VM20 reused the initial readiness actor/request for replacement readiness and was
correctly refused before admission. Zero new readiness rows exist. The first
replacement and clean cleanup completed;19 files,13 mirrors and seven saved-state
files independently verify, with original disks retained and outer exit1. Use a
distinct fixture-step ID while preserving exact IDs when replaying uncertain work.
This supersedes M104’s pending trial with a diagnosed failure, not a runtime pass.
The fixture correction is included in the next combined run. M103 remains valid.
See diagnostics/2026-10-23-member-resumed-readiness-trial20.md.

## M109 — Retain full lifecycle evidence without oversized serial events (2026-10-23, preparation-qualified)

The corrected appliance21 combines replacement readiness with queued current Stop.
Its164 sources, embedded test/guest code and host guards verify. Full receipts stay
on disk; serial events carry bounded byte-count/hash references checked by offline
verification. Transport/framing tests pass without raising the256KiB event limit.
This addresses M103's transport-growth concern and includes M108's request-ID fix;
actual readiness/current Stop remains pending. See
diagnostics/2026-10-23-member-current-stop-preparation.md.

## M110 — Recover and replace later generations from retained causal history (2026-10-23, source-qualified)

Private exit/retirement/resume/readiness adapters now accept later replacement
contracts without rewriting earlier receipts.16 inert cases reach generations3/4
and retain uncertainty/state bytes; full adapter race tests pass. Ordered prefix
history selection separately passes full model tests and final vet. This extends M106's Stop adapter and supersedes
M095/M100's first-replacement-only resume/readiness adapter limits; schema34 still stores one resume per
member, and VM21 predates this code. See
diagnostics/2026-10-23-member-later-generation-adapters.md.

## M111 — Replacement readiness and Stop survive real runtime replay/reopen (2026-10-23, runtime-qualified)

VM21's frozen schema34 queue passes generation2 readiness and clean Stop in1449.61s.
All17 extracted files, original receipts and seven retained state files independently
verify; disks and44 metadata files are retained, with outer exit0 and inactive scope.
A quota-interrupted offline verifier was retained and the unchanged retry passed.
This supersedes M101/M104/M109's pending successful readiness/current-Stop runtime
slice only. VM20's diagnosed failure remains; later adapters/schema35 and runner
application/subscription recovery are not qualified by this run. See
diagnostics/2026-10-23-member-current-stop-runtime.md.

## M112 — Budget conversion for the whole virtual disk and serialize large checks (2026-10-23, source/read-only qualified)

VM21 exposed the fixed4GiB extraction estimate: its12GiB virtual disk and concurrent
race tests exhausted temporary quota. Failed logs/dump remain retained; unchanged
runtime verification later passed. Geometry-based admission now reserves virtual
size plus2GiB before output creation. Three tests and actual read-only geometry/
quota refusal pass; no new conversion was executed. This supersedes the fixed
extraction estimate, not earlier runtime evidence. Serialize large conversions and
broad tests; relocate only exact hashed inactive artifacts. See
diagnostics/2026-10-23-extraction-storage.md.

## M113 — Persist repeated replacements without rewriting old lifecycle receipts (2026-10-23, source-qualified)

Schema35 removes only the per-member resume uniqueness limit; each retirement is
still consumed once. Transactional parent/child migration keeps foreign keys enabled
and copies historical bodies unchanged. Generations3/4 pass16 interruption boundaries,
9 migration rollbacks,6 queue cases and2 capability refusals. Original34 reproduces
and refuses real35 unchanged; full managed race passes in662.875s and vet is clean.
Failed fixture/quota/aggregate-timeout trials are retained. This supersedes M110's
one-resume persistence limit only. VM21 remains schema34 evidence; later real runtime
and runner application recovery are pending. See
diagnostics/2026-10-23-member-repeated-resume-store.md.

## M114 — Freeze the second-replacement fixture before real execution (2026-10-23, preparation-qualified)

Appliance22's171 sources and embedded test/guest code independently verify. Its
schema35 fixture extends VM21 through generation3 exit/retirement/resume/readiness/
Stop, replay and reopen. Only the completed-build cache was cleared to free
quota; original evidence remains. This advances M113 to runtime preparation only;
app/subscription recovery is not wired and actual later-generation runtime is pending.
See diagnostics/2026-10-23-member-repeated-preparation.md.

## M115 — Journal application-test phases and keep sessions in the actual anchor tmpfs (2026-10-23, source-qualified)

The opt-in runner app fixture uses four ordered phases, original sessions/channel/
cursor and two recovery snapshots. It refuses uncertain write replay. Eleven probe/
phase and eight routing tests pass; explicit host guards and vet pass. The runner
anchor has a private /urbit tmpfs, not the earlier harness's writable /tmp; cookies
must use its verified mount. The actual fresh root is /srv/urbit/piers. This corrects
those draft fixture assumptions and extends M056's test design into runner wiring,
not runtime qualification. VM22 does not include it. See
diagnostics/2026-10-23-runner-application-recovery-source.md.

## M116 — Give the aggregate race suite its measured CI budget (2026-10-23, configuration update)

M113's662.875s managed race run exceeds Go's default ten-minute package timeout.
The CI race command now uses20m and its whole source job35m, preserving individual
runtime deadlines. CI also runs the19 offline application-probe/state checks from
M115. This supersedes the old aggregate CI budget; it is not a hosted CI pass.
Local suite evidence remains linked in M113, and VM qualification stays separate.

## M117 — Verify the embedded application probe before boot (2026-10-23, preparation-qualified)

Appliance23 verifies174 frozen sources, embedded Go test/guest code, both Python
helpers and Gall fixture. It advances M115 to preparation only and has not booted.
VM22 remains a distinct earlier lifecycle run; no app recovery pass is inferred.
See diagnostics/2026-10-23-runner-application-recovery-preparation.md.

## M118 — Expose review-only fake-project planning without runner contact (2026-10-23, source/CLI-qualified)

`project-plan` and `help projects` expose canonical member/port review and a digest,
with explicit unperformed reservation checks and no runtime authority. Runner/socket
selectors refuse. Focused input/help checks, collector race7.847s, vet and five
built-CLI private-namespace cases pass. This supersedes M036's lack of
a CLI review entry only; registration/lifecycle capabilities remain private.
Prepared VM23 predates it. See diagnostics/2026-10-23-project-plan-cli.md.

M118 follow-on: the packaged operator skill now routes offline project review
before runner selection, preserves the full plan/digest and explains the missing
public runtime path. Four package checks pass; links/frontmatter were reviewed.
The supplied skill validator cannot run because both available Pythons lack
PyYAML; its failed attempts are retained, not counted as validation passes.

## M119 — Two same-pier replacements now pass the real runner queue (2026-10-23, runtime-qualified)

VM22/schema35 passes generations2/3 readiness and clean Stops in1605.95s. Independent
verification checks17 result files, every database/runtime mirror, both retirement
comparisons and seven final pier files totaling1,016,852,492bytes. Replay/reopen
preserve history; outer/QEMU exit0 and exact scope is inactive. Original disks and
44metadata files remain. Geometry-based extraction admission passes its first real
conversion. This supersedes M113/M114's pending generation3 runtime gate and
advances M111; the stopped peer and absent app probe limit the result. App23 is a
separate running gate, not a pass. See diagnostics/2026-10-23-member-repeated-runtime.md.

## M120 — Keep a real peer alive for the next recovery gate (2026-10-23, source-qualified)

The separate running-peer fixture preserves ~nec's exact original process while
~zod is replaced twice; all eight writes come from ~nec. Four phase snapshots and
final cleanup reject container/process/config/attachment changes. Twelve Python
checks, narrow race1.030s, vet and host/appliance guards pass. Peer final shutdown
is exact-ID fixture cleanup, not a queued Stop authorized by stale readiness.
This extends M115's stopped-peer scope without superseding its evidence; frozen
running app23 is unchanged. No schema change or runtime pass is claimed. See
diagnostics/2026-10-23-running-peer-recovery-source.md.

## M121 — Freeze running-peer recovery separately (2026-10-23, preparation-qualified)

Appliance24 verifies175 sources and embedded Go/Python/Gall code. Its original
artifacts are hash-verified and retained outside temporary quota, with exact paths
linked back. It is unbooted; M120's running-peer runtime gate remains pending behind
VM23. Later project-inventory source is outside this snapshot. See
diagnostics/2026-10-23-running-peer-recovery-preparation.md.

## M122 — Expose saved project claims without claiming runtime readiness (2026-10-23, source/CLI-qualified)

`projects`/`GET /v1/projects` now expose a consistent read-only reservation view
under `project-reservations-v1`. Exact observer route, canonical claims/digest,
runner identity and older-capability refusals pass; full client/access/collector
race and six built-CLI namespace cases pass. Two test-setup failures remain in the
evidence; product selection/permissions were unchanged. Runtime observation is
explicitly not-performed, and registration/lifecycle mutations remain private.
This supersedes the lack of a public reservation read, not the pending lifecycle
API gate. Frozen appliances23/24 predate it. See
diagnostics/2026-10-23-project-reservation-inventory.md.

## M123 — Refresh observation context without rewriting historical ownership (2026-10-23, source-qualified)

A pure current-member selector and read-only readiness collector preserve original
pier identity while resolving complete current container history. Both original
and replacement targets pass fresh observation after a peer change; full pure race
4.316s and vet pass. The wrong-port fixture failure remains recorded. This advances
M120's changed-peer follow-on design only; adapters, persistence and lifecycle
admission remain pending. Initial receipts lack process StartedAt, so they cannot
prove uninterrupted execution; a fresh bracket establishes only its own continuity.
No earlier runtime claim is superseded, and frozen23/24 are unchanged. See
diagnostics/2026-10-23-member-current-context.md.

## M124 — Runner replacements preserve application state and original subscriptions (2026-10-23, runtime-qualified)

VM23/schema35 independently passes eight acknowledged writes, two queued-update/
existing-subscription recoveries, four channels and12 matched snapshots across two
same-pier replacements. No reauthentication or write replay occurred. Outer/QEMU
exit0, clean poweroff and inactive exact scope are verified; originals and46
metadata files remain. This supersedes M115/M117's pending runner application gate
and complements M119. Only two writes are remote; six later writes are local while
~nec stays stopped. Running-peer messaging remains the distinct appliance24 gate.
See diagnostics/2026-10-23-runner-application-recovery-runtime.md.

## M125 — Observe current member contracts through a read-only adapter (2026-10-23, source-qualified)

The current-readiness adapter reconstructs latest exact contracts, brackets HTTP
with process/namespace/pier/publication checks, and exposes no mutation methods.
56 cases plus existing readiness/Stop regressions pass race25.715s; vet is clean.
The fixture's retained-path assertion failure is preserved and corrected without
relaxing product checks. This supersedes M123's absent-adapter status only. All
members must be running; durable context, admission and lifecycle integration
remain pending. Running appliance24 predates this work. See
diagnostics/2026-10-23-member-current-readiness-adapter.md.

## M126 — Bind a new Stop to the current readiness process (2026-10-23, source-qualified)

Current-context Stop preserves historical prefix validity while refusing old
readiness for new action after membership changes. Its adapter pins the observed
target start time and reuses the single-TERM protocol. Ten inert cases pass
race4.584s, full pure race4.431s and vet pass. This extends M125 without changing
stored formats or queue admission. Store/replay, downstream lifecycle and runtime
qualification remain pending. See diagnostics/2026-10-23-member-context-stop.md.

## M127 — Persist new readiness without rewriting previous context (2026-10-23, focused-source-qualified; full regression pending)

Schema36 adds private current-readiness journals and fenced read-only scheduling.
Restart/replay, rollback/corruption, historical-prefix preservation and rejection of
current containers with Stop intent pass focused checks. Store race49.217s, queue
race12.504s and vet pass. Frozen35 reproduces twice and its original reader refuses
real36 unchanged. The full managed race suite is still running. This supersedes
M123/M125's absent-persistence status only; M126 Stop admission and runtime remain
pending. VM24 is unchanged schema35. See
diagnostics/2026-10-23-member-context-readiness-store.md.

### M127 follow-up — Full managed regression passed (2026-10-23)

The full race suite exited0 in814.392s; all recorded production source hashes
remain unchanged. This supersedes M127's pending full-regression status. Separate
VM fixture wiring compiles and refuses host execution, but schema36 runtime
qualification is still pending. The original log is retained with M127 evidence.

## M128 — Remote recovery succeeds with the original peer uninterrupted (2026-10-23, runtime-qualified)

VM24/schema35 independently passes all eight remote writes and two queued-update/
subscription recoveries while the original ~nec process remains unchanged through
two ~zod replacements and its final Stop. Four channels and12 snapshots match;
original sessions persist without later login. Outer/QEMU exit0, clean poweroff
and inactive scope are verified in1757.56s. This supersedes M120/M121's pending
runtime status and M124's remote-message limitation. Peer shutdown is still exact-ID
fixture cleanup, not new runner Stop authority after membership changes. Schema36
and general changed-peer lifecycle remain separate gates. See
diagnostics/2026-10-23-running-peer-recovery-runtime.md.

## M129 — Qualify new readiness separately from unchanged peer continuity (2026-10-23, prepared; runtime pending)

Appliance25 freezes192 sources/schema36 and observes both members through the
private queue after two replacements. It checks replay/reopen and preserves all
original readiness, while retaining M128's remote recovery assertions. Compilation,
host refusal, flag prerequisites, vet and embedded-artifact verification pass.
This advances M127's runtime-test preparation only; no runtime result or new Stop
authority is claimed. See diagnostics/2026-10-23-current-context-readiness-preparation.md.

## M130 — Preserve refreshed Stop ownership through exit and replacement (2026-10-23, source-qualified)

Current-context adapters now carry exact readiness/process evidence through Stop,
read-only exit recovery, retirement and same-pier replacement. Both creation modes,
recorded stopped peers, lost replies and changed/unrecorded peer refusals pass12
inert cases plus related race regressions39.285s/vet. This supersedes M126's absent
following-adapter status only; store/queue and runtime remain pending. VM25 is an
earlier frozen readiness experiment. Original36 source/fixture is retained and
reproduces twice; no successor migration is claimed. See
diagnostics/2026-10-23-current-context-lifecycle-adapter.md.

M130 follow-up: concrete post-replacement readiness also passes with the current-context source (race9.981s); preserved under the same evidence directory.

## M131 — Retain current-context authority through private lifecycle queues (2026-10-23, focused-source-qualified; full regression pending)

Schema37 persists refreshed Stop intent and carries its exact historical source
through exit, retirement, replacement and readiness. Both targets pass restart,
process mismatch, lost replies, request replay and full-history reopen; focused
store42.264s, queue80.716s, historical4.332s and vet pass. Original36 reproduces and
refuses actual37 unchanged. This supersedes M130's pending store/queue status only.
The broad managed race is running; VM25 remains earlier schema36. No public
mutation API or new runtime result is claimed. See
diagnostics/2026-10-23-current-context-lifecycle-store.md.

### M131 follow-up — Full managed regression passed (2026-10-23)

The complete managed race suite exited0 in972.505s and all M131 source hashes match. This supersedes M131's pending broad-regression status. Later VM-only fixture additions have separate compile/refusal/probe checks; schema37 runtime qualification remains pending.

## M132 — Refresh both members after peer changes without rewriting history (2026-10-23, runtime-qualified)

VM25/schema36 independently passes both queued current-readiness observations
following two receiver replacements. Exact creator/process/pier evidence, old
readiness, replay, SQLite mirrors and reopen match. All eight writes stay remote;
two recovery cycles and original sessions remain intact. Outer/QEMU exit0, clean
poweroff and inactive scope are verified in1860.46s; originals are retained.
This supersedes M127/M129's pending runtime status only. Later schema37 lifecycle
and sender replacement remain separate gates. See diagnostics/2026-10-23-current-context-readiness-runtime.md.

## M133 — Test the sender's own replacement after its peer changes (2026-10-23, prepared; runtime pending)

Appliance26/schema37 prepares queued current-context Stop and following lifecycle
for the original sender after two receiver replacements. It retains the receiver's
exact process, refreshes both members again, and ends with queued Stops. Its six
application phases check twelve remote writes with original sessions. Compilation,
host/prerequisite refusals, vet,14 probe checks and embedded artifacts pass. This
advances M131's runtime preparation only; no runtime pass is claimed. See
diagnostics/2026-10-23-current-context-lifecycle-preparation.md.

## M134 — Keep test sessions independent of anchor lifetime (2026-10-23, source-qualified; runtime pending)

Future runner app probes use direct guest-loopback Eyre/SSE and private guest
runtime-tmpfs cookies across phase processes. Namespace-only admin discovery is
retained. Twenty inert tests cover exact ports, session/cursor reuse, private-file
refusals and redacted non-replayed failures; entry compile/vet pass. This supersedes
anchor-owned cookie storage for future fixtures only. VM26 is frozen unchanged;
no runtime or anchor-recovery result is implied. New namespace authority remains
separate, and this storage is not a wallet design. See
diagnostics/2026-10-23-runner-published-probe.md.

## M135 — Observe anchor faults without adopting new ownership (2026-10-23, source-qualified)

The pure read-only review brackets stopped/restarted-anchor observations while
requiring every current member and initialized pier unchanged in the old namespace.
Known starts stay pinned; missing original start metadata proves fresh continuity
only. Same-creator historical readiness survives peer changes. Full projectnetwork
race6.752s/vet pass. This advances M134's pending namespace-evidence design only;
no adapter, durable recovery, mutation authority or runtime pass is claimed. See
diagnostics/2026-10-23-anchor-recovery-review-model.md.

## M136 — Inspect failed anchors without relaxing lifecycle validators (2026-10-23, source-qualified)

The concrete review adapter verifies complete exact-ID metadata, every initialized
pier and old member namespaces across two reads. A separate exited-anchor validator
admits diagnosis while ordinary lifecycle inspection still requires a running
anchor and initial Start still requires never-started state. Thirty-three inert
cases9.525s and related race regressions42.027s/vet pass without signals or HTTP.
This supersedes M135's absent-adapter status only; runtime, persistence and
coordinated recovery remain pending. See
diagnostics/2026-10-23-anchor-recovery-review-adapter.md.

M136 design follow-up: `docs/anchor-recovery-design.md` records the next operation's
required history boundary. A changed namespace needs explicit successor evidence;
never rewrite or synthesize an original `AnchorStart`/member receipt. Drain all old
members before replacement, retain uncertainty, and resume exact piers without
bootstrap. These are proposed implementation constraints, not a qualified action
protocol or a change to M136's source-only status.

## M137 — Recover sender and receiver state through current-context queues (2026-10-23, runtime-qualified)

VM26/schema37 independently passes the sender's queued lifecycle after two
receiver replacements, preserving the receiver process and all original history.
Both final Stops use queues. Twelve remote writes, three subscription recovery
cycles, six channels/eighteen snapshots and the original two sessions pass.
Outer/QEMU exit0, clean poweroff and inactive scope are verified in2123.97s;
original disks and metadata are retained. This supersedes M131/M133's pending
runtime status only. Later direct transport and anchor review remain separate.
See diagnostics/2026-10-23-current-context-lifecycle-runtime.md.

## M138 — Freeze direct-client recovery plus read-only anchor fault review (2026-10-23, prepared; runtime pending)

Appliance27/schema37 prepares twelve remote writes with guest-runtime session
storage, then stopped/restarted-anchor reviews after fresh readiness for both
members. It must preserve old processes and history; final shutdown is exact-ID
fixture cleanup. Compile/refusals/vet/20probe tests and embedded artifacts pass.
This advances M134/M136 runtime preparation only, not coordinated recovery. See
diagnostics/2026-10-23-anchor-review-preparation.md.

## M139 — Drain reviewed old processes without replaying uncertain signals (2026-10-23, source-qualified)

The pure coordinated drain validates complete history, fixed member order, exact
piers/processes and the unchanged anchor fault. Every signal has a prior commit;
lost replies need fresh read-only exit evidence, and unclean exits block the next
member. Full projectnetwork race13.805s/vet pass. An initial wait-error handling
bug and test-driver alias were fixed; failed evidence is retained. This advances
M136's action-model design only. Persistence, concrete dispatch and coordinated
runtime recovery remain pending; VM27 is unchanged. See
diagnostics/2026-10-23-anchor-drain-model.md.

## M140 — Recheck complete ownership before one drain signal (2026-10-23, source-qualified)

The concrete drain adapter pins exact old processes/piers, rechecks all ownership
before SIGTERM, and reopens uncertain outcomes only for read-only exit evidence.
Fifty-seven inert race cases34.994s, related lifecycle race31.115s, full pure
race13.787s/vet/diff pass. A JSON-copy representation mismatch was fixed with typed
copies; the failed run is retained. This supersedes M139's absent-adapter status
only. Persistence, namespace succession and coordinated runtime remain pending;
VM27 is an earlier frozen observer fixture. See
diagnostics/2026-10-23-anchor-drain-adapter.md.

## M141 — Record namespace succession separately from original launch authority (2026-10-23, source-qualified)

A complete clean drain now admits a distinct namespace intent. Already restarted
anchors are observed without another Start; stopped anchors receive one committed
Start, with lost acknowledgements resolved only by observation. Old creators,
piers and process exits remain unchanged. Full pure race26.833s/vet/diff pass;
a test-only compile typo and its correction are retained. This advances M140's
pending successor-model status only. Concrete qualification, persistence and
coordinated runtime remain pending. See
diagnostics/2026-10-23-anchor-namespace-model.md.

## M142 — Observe or Start only the exact drained anchor (2026-10-23, source-qualified)

The namespace adapter requires exact clean-exit history and rechecks every old
process/pier before one anchor Start. Already restarted anchors and reopened
uncertain intents are observation-only. Twenty-two inert race cases23.979s,
related regression65.062s/vet/diff pass. This supersedes M141's pending
concrete-adapter status only. Persistence, coordinated replacement and runtime
remain pending; VM27 did not contain this code. See
diagnostics/2026-10-23-anchor-namespace-adapter.md.

## M143 — Preserve interrupted readiness without claiming an anchor runtime pass (2026-10-23, failed runtime trial)

VM27 completed five direct-client phases and three replacements, then sender
readiness interrupted in41.500819s before any anchor fault. No adapter/setup error
was retained; missing observations prevent a proven root cause. Freshness expiry
and repeated fixture audits are hypotheses. Outer1/QEMU0, clean poweroff, inactive
scope,19 extracted files, operation mirrors and both final pier hashes are
independently verified; originals/48metadata files are retained. This supersedes
M138's running/prepared experiment status, not its pending qualification. M137
remains the latest full runtime pass. See
diagnostics/2026-10-23-anchor-review-trial27.md.

## M144 — Keep diagnostic audits outside freshness-sensitive runtime observations (2026-10-23, fixture-source-qualified)

The resumed-readiness fixture now records observations, canonical identity and
per-call timing. Full fixture audits bracket collection; each read still proves
its exact committed/running operation. Production freshness and validation remain
unchanged. Guard race3.004s, compile, host refusal, vet and diff pass. This advances
M143's diagnostic follow-up only, not a proven cause or runtime fix. See
diagnostics/2026-10-23-readiness-timing-fixture.md.

## M145 — Freeze an instrumented rerun without weakening runtime gates (2026-10-23, prepared; runtime pending)

Appliance28/schema37 retains all direct-client and anchor-review assertions and
adds M144's readiness diagnostics/audit placement. Embedded artifacts and frozen
sources are independently verified. The original failed VM27 remains retained;
its timing hypothesis is unproven. Compiled later drain/namespace code is not
exercised by this fixture. This advances rerun preparation only. See
diagnostics/2026-10-23-anchor-review-rerun-preparation.md.

## M146 — Container absence must retain every initialized pier (2026-10-23, source-qualified)

Coordinated retirement now has a pure serial model under the explicit successor
namespace. Each recorded cleanly exited container has one removal intent; lost
replies reconcile by observation only. All pier identities remain required after
container absence, and original history is unchanged. Focused tests2.054s, full
projectnetwork race52.778s and vet/diff pass. This advances the pending pure
retirement portion of M142, without superseding its runtime/persistence limits.
The concrete adapter is still under test. See
diagnostics/2026-10-23-anchor-retirement-model.md.

## M147 — Prove container absence independently of network membership (2026-10-23, source-qualified)

The coordinated retirement adapter now checks complete container-ID inventory,
remaining exact stopped contracts, successor anchor and every retained pier before
and after its one unforced removal. Reopened uncertainty permits only observation.
A container moved to another network is still present and must refuse absence.
Focused inert race36.871s and related race105.032s/vet/diff pass. A missing test
state-directory setup was corrected without weakening production checks; failures
are retained. This supersedes M146's pending concrete-adapter status only.
Replacement, persistence and coordinated runtime remain pending. See
diagnostics/2026-10-23-anchor-retirement-adapter.md.

## M148 — Successor containers retain original anchor history (2026-10-23, source-qualified)

A complete retirement now yields an explicit per-member successor creation
selection. Its separate immutable Docker contract advances the member generation,
resumes the exact initialized pier without a pill, and names successor/retirement
operations while preserving original AnchorStart. Ordinary v1 digests and labels
remain unchanged. Pure focused3.909s/full56.377s, concrete focused4.543s/related
50.161s race and vet/diff pass. This advances M147's pending contract-construction
status only: no replacement dispatch, resolver, persistence or runtime is qualified.
See diagnostics/2026-10-23-recovery-member-contract.md.

## M149 — Replacement running state is not recovered application readiness (2026-10-23, source-qualified)

The pure serial replacement executor now retains unique per-member Create/Start
intents, every pier and earlier replacement process. Returned Create IDs survive
errors; uncertain Start resolves read-only without replay or invented acknowledgement.
The group remains distinct from fresh identity/application readiness. Focused
race60.152s/full122.311s and vet/diff pass; a reused fixture container ID was
corrected and fault-boundary assertions strengthened, with failed evidence retained.
This advances M148's pending pure dispatch model only. Concrete adapter, persistence,
causal history and coordinated runtime remain pending. See
diagnostics/2026-10-23-anchor-replacement-model.md.

## M150 — Concrete replacement must recheck mounts and exact successor processes (2026-10-23, source-qualified)

The separate recovery adapter now creates/starts exact new contracts over the
retained piers after complete old-container absence. It rechecks locks, state trees,
mount inventory, anchor namespace and peer metadata at dispatch. Lost Start replies
reopen observation-only; uncertain Create retains any exact ID and requires review.
Both-member inert recovery and fault-boundary cases pass race69.220s/vet/diff.
This supersedes M149's pending concrete-adapter status only; fresh readiness,
causal history, persistence and coordinated runtime remain open. See
diagnostics/2026-10-23-anchor-replacement-adapter.md.

## M151 — Docker anchor restart retains prior FinishedAt; VM28 partially passes (2026-10-23, independently verified failed runtime trial)

VM28 completes twelve writes, three replacement cycles, six channels/eighteen
subscription snapshots and all replacement readiness using two original sessions
through direct published Eyre. It then fails restarted-anchor review: Docker retains
previous FinishedAt before the new StartedAt, whereas the frozen recovery reader
uses initial-only validation requiring zero. Exact retained metadata reproduces
that source rejection. Cleanup-only flags were also conditional on review success,
causing a secondary missing-Stop fixture panic at reopen. Outer1/QEMU0, clean2036.72s,
inactive scope,19files, database mirrors, application protocol and both final piers
are independently verified; originals/47metadata files are retained.

This supersedes M145's running status and M134's pending three-cycle direct-client
runtime evidence, but not the full recovery gate. VM27's timing cause remains a
hypothesis. M136/M142/M147/M150's synthetic anchor timestamp assumptions are
insufficient for actual Docker restart and are being corrected in separate source
gates; their prior test passes remain historical evidence. M137 remains the latest
full runner runtime pass. See diagnostics/2026-10-23-anchor-review-trial28.md.

## M152 — Accept historical anchor exit times only in recovery (2026-10-23, source-qualified)

The exact VM28 metadata regression now passes recovery validation while initial
Start still rejects it. Recovery accepts a prior FinishedAt only before current
StartedAt, retains the evidence unchanged, and pins a recovery-issued Start to
its observed prior stop. Member checks remain strict. Concrete later recovery
readers use this explicit path. Full pure race118.399s, related adapter race164.596s,
fixture compile/host refusal and vet/diff pass. Cleanup markers now survive failed
review and observer diagnostics retain underlying errors/timings.

This supersedes M151's pending source correction and the insufficient synthetic
anchor timestamp assumptions in M136/M142/M147/M150. It does not supersede their
runtime/persistence limits or turn VM28 into a full pass. See
diagnostics/2026-10-23-anchor-history-fix.md.

## M153 — Freeze recovery timestamp correction in a fresh appliance (2026-10-23, prepared; runtime pending)

Appliance29/schema37 includes M152's exact-evidence timestamp correction and
fixture cleanup/observer diagnostics. All frozen sources, embedded binary/helpers
and artifacts are independently verified and retained. It repeats direct-client
state/subscription recovery plus stopped/restarted anchor reviews. Later coordinated
recovery code is compiled but not exercised. This advances rerun preparation only;
M151's failed full trial remains retained. See
diagnostics/2026-10-23-anchor-history-rerun-preparation.md.

## M154 — Recovery readiness pins the last replacement as strictly as its peers (2026-10-23, source-qualified)

The pure all-member readiness collector derives the explicit successor binding,
then brackets every selected ship identity with fresh complete observations.
Every exact process start and pier remains pinned, including the last replacement
whose creation transition initially admitted its start time. Identity receipts
retain member creator/generation and selected endpoint; original history and anchor
resource generation remain unchanged. Focused race12.566s/full126.710s/vet/diff pass.
This advances M150's pending pure readiness model only; concrete adapter tests,
causal history, persistence and coordinated runtime remain open. See
diagnostics/2026-10-23-anchor-recovery-readiness-model.md.

## M155 — Concrete recovery readiness has no lifecycle side effects (2026-10-23, source-qualified)

The adapter verifies explicit successor contracts, old-container absence, every
pier/start time and unchanged full metadata around all selected identity reads.
Twenty inert scenarios plus incomplete-intent rejection pass race89.024s/vet/diff.
The initial caller-alias fixture failure is retained and corrected by separating
its collector authority from the input it mutates; production code was unchanged.
This supersedes M154's pending concrete-adapter status only. Causal history,
persistence and coordinated runtime remain unqualified. See
diagnostics/2026-10-23-anchor-recovery-readiness-adapter.md.

## M156 — Later member replacements preserve the recovery boundary (2026-10-23, source-qualified)

The pure post-recovery selector now resolves three subsequent ordinary replacements
across both members by exact causal binding, retaining namespace, pier, creator,
generation and process evidence. Full-suffix validation protects historical prefix
reads from ambiguous later history. Focused race6.995s/full132.662s/vet/diff pass.
This advances M155's causal-history requirement for one completed recovery only.
Multiple recovery boundaries, concrete ordinary adapters and persistence remain
pending. The next source model will retain explicit ordered boundaries and their
review prerequisites; it must not synthesize an original anchor receipt. See
diagnostics/2026-10-23-recovered-member-generations.md.

## M157 — Multiple recoveries retain separate causal boundaries (2026-10-23, source-qualified)

Explicit completed boundaries preserve their preceding ordinary replacements and
review prerequisites, while the current suffix remains separate. Two complete pure
recoveries, replacements between/after them, historical prefix reads and rejection
of altered history/reused authority pass focused race61.070s,
full race196.440s and existing adapter regression254.309s/vet/diff.
This supersedes M156's pending multiple-boundary pure model only. Existing adapter
regression is not a second-recovery runtime pass. Concrete later creation/readers,
persistence and coordinated runtime remain pending. See
diagnostics/2026-10-23-recovery-epochs.md.

## M158 — Ordinary post-recovery resumes name their own retirement (2026-10-23, source-qualified)

A distinct pure selection/concrete contract links an ordinary replacement to the
preceding completed recovery, successor namespace and its new ordinary retirement,
while preserving original AnchorStart and earlier contract formats. Exact current
creator/generation/process and initialized pier are required; no pill or runtime
dispatch is introduced. Pure race7.068s, contract race13.021s, legacy race1.063s,
managed compile/skip and vet/diff pass. This advances M157's pending post-recovery
creation contract only. Generalized concrete readers, dispatch, persistence and
coordinated runtime remain pending. See
diagnostics/2026-10-23-recovered-resume-contract.md.

## M159 — VM29 qualifies direct recovery cycles and historical anchor review (2026-10-23, independently verified runtime pass)

VM29/schema37 passes twelve direct Eyre writes, three replacement cycles, six
channels/eighteen snapshots, two original sessions and zero later logins/replays.
Healthy refusal plus stopped/restarted anchor reviews preserve member processes,
piers and original history. Recovery-aware historical FinishedAt handling, observer
diagnostics, freshness bounds and reopen/replay all verify. Outer0/QEMU0, clean2022.2s,
17 result files, database mirrors and both final pier states are independently
verified; original disks and metadata are retained.

This supersedes M153's pending VM and M152's pending runtime rerun only. Failed VM28
remains failed evidence, and VM27's timing cause stays a hypothesis. Final Stops
here use fixture cleanup; M137/VM26 remains the fully queued final-Stop reference.
Coordinated repair after the anchor fault is not exercised. Later source gates
M154–M158 are outside the frozen appliance. See
diagnostics/2026-10-23-anchor-review-runtime.md.

## M160 — Concrete readers retain recovery and ordinary creator history (2026-10-23, source-qualified)

Read-only reconstruction now supports coordinated creators and later ordinary
successors together. Exact contracts, process starts, piers, complete source history
and all prior retired IDs are checked. Six recovery-only/four mixed-creator modes,
focused race30.403s/30.308s, related race307.923s, managed compile/skip and vet/diff
pass. This supersedes M158's pending generalized reader status only. Later recovery
phases still need historical-ID guards and second-cycle concrete qualification;
persistence and real coordinated repair remain open. See
diagnostics/2026-10-23-recovery-history-reader.md.

## M161 — Second recovery preserves mixed creators and rejects old IDs at every phase (2026-10-23, source-qualified)

Complete second recovery passes for stopped/restarted anchors and mixed ordinary/
coordinated creators. A reproduced missing historical-ID guard is fixed across
signal, namespace, retirement, replacement and readiness. Nine inert scenarios pass
race213.241s, related adapters race309.22s, managed compile/skip and vet/diff.
Piers and original histories remain intact; the before-fix failure is retained.
This supersedes M160's pending second concrete recovery/history guards only.
Ordinary post-recovery dispatch, persistent coordination and actual Docker repair
remain open. See diagnostics/2026-10-23-second-recovery-adapters.md.

## M162 — Ordinary lifecycle keeps explicit completed-recovery history (2026-10-23, source-qualified)

Pure current readiness/Stop preserve two recovery boundaries and distinguish fresh
admission from historical validity. Concrete readiness, Stop, exit, retirement,
resume and resumed readiness pass mixed-creator/live-or-stopped-peer cases, lost
Stop acknowledgement and history/process faults. Piers, peer metadata and exact
dispatch counts remain intact; complete inventory guards reject historical IDs.
Full pure race272.389s, related adapter race188.132s, managed compile/skip and vet/diff
pass. This supersedes M161's pending ordinary post-recovery source integration only.
Persistent coordination/public scheduling and actual Docker repair remain open.
See diagnostics/2026-10-23-recovered-current-lifecycle.md.

## M163 — Original schema37 writer and populated lifecycle history are frozen (2026-10-23, prepared)

The original source snapshot (1,569files) and reproducible531,102-byte synthetic
fixture preserve completed/interrupted context-bound Stops, two replacements and
their exact operation mirrors. The initial fixture-helper timestamp mismatch is
retained and corrected without editing original production code. Source SHA57ed6461
and fixture SHA1ba8249a are recorded in the diagnostic. This advances M162's next
upgrade-preparation step only; no persistent-format change, migration or real
successor-reader refusal has occurred. Add the snapshot to the old-reader inventory
only with the actual successor format. See
diagnostics/2026-10-23-schema37-original-writer.md.

## M164 — Aggregate recovery journal and exact history assembly (2026-10-24, source-qualified)

The serial journal retains one nested checkpoint across the entire recovery,
reserves distinct member creator IDs and refuses stale or rewritten transitions.
Completed-record assembly preserves all ordinary generations around two recovery
boundaries and rejects incomplete or branched ancestry. Focused race tests and
static checks pass. This supersedes only M162's pending pure coordination model;
durable ownership, queue integration and real coordinated repair remain pending.
M163 remains the unchanged original schema37 writer. Evidence:
diagnostics/2026-10-24-recovery-journal-model.md.

## M165 — Schema38 persistent recovery, mixed lifecycle and private queue (2026-10-24, focused source-qualified)

Actual SQLite ownership, actor/source references and mirrors now guard the serial
recovery and distinct member creators. Atomic checkpoints, interruption/replay,
management fencing and later ordinary lifecycle pass synthetic tests. Read-only
readiness waits for Eyre without repeating mutations. Queue race122.986s, planner
race2.927s, full pure race228.892s, mixed lifecycle8.627s, historical tests5.557s,
original37 refusal of real38 unchanged and static/harness checks pass. The original
schema37 archive/fixture remain unchanged. Full managed regression is pending:
its first invocation exited143 without output; the rerun remains active.

This supersedes M164's pending persistent ownership/private queue and M163's pending
successor-format inventory only. It does not supersede runtime limits in M159.
Interrupted continuation/public routes and actual coordinated Docker repair remain
open. Evidence: diagnostics/2026-10-24-persistent-anchor-recovery.md.

## M166 — Corrected coordinated-repair appliance is frozen (2026-10-24, prepared)

Image31 independently verifies264 sources and embedded binaries/helpers. Host
refusal,43 harness checks and the cleanup correction's race10.299s pass. It targets
four application/subscription cycles,16 writes and the two original sessions.
Image30 was never booted and remains retained; image31 supersedes its cleanup
assumption that retirement remains a top-level payload after later checkpoints.

This advances M165's pending VM preparation only. Full managed regression is still
running; no coordinated runtime result or continuation queue exists. The new pure
read-only checkpoint dispatcher is outside frozen31. Evidence:
diagnostics/2026-10-24-anchor-recovery-preparation.md.

## M167 — Full schema38 regression and one-checkpoint read-only observation (2026-10-24, source-qualified)

Full managed baseline race1307.093s passes, supplemented by the later queue-readiness
race122.986s and cleanup-fixture race10.299s. This supersedes only M165/M166's pending
full source regression; first-run exit143 remains unexplained, not a test pass.
A new pure dispatcher resolves one uncertain retained outcome via observation-only
interfaces, preserving serial commits and refusing planned work/unknown Creates.
Its race53.505s/static checks pass. It is outside frozen image31 and adds no
persisted continuation queue, public route or runtime claim. Evidence:
diagnostics/2026-10-24-recovery-regression-observation.md.

## M168 — Original schema38 recovery writer is frozen and reproducible (2026-10-24, prepared)

The1603-file source snapshot reproduces959559 bytes across20 tables, including one
completed recovery, a second interrupted drain and four creator operations. Source
SHA8b363b4c and fixture SHA27ae5f54 are retained. This advances M167's preparation
for persisted observation/continuation, without changing active schema38, its
reader inventory or frozen VM31. No real successor-reader refusal is claimed yet.
See diagnostics/2026-10-24-schema38-original-writer.md.

## M169 — Explicit read-only recovery observations persist in schema39 (2026-10-24, source-qualified)

One actor/request/sequence-bound observation atomically advances the exact parent
checkpoint and member mirrors. It never repeats runtime effects or admits the next
mutation; unresolved recovery stays fenced except after complete fresh readiness.
Queued replay/startup are inert. Historical observation evidence must be an exact
ancestor, preserving lost versus received acknowledgements. Store/queue race334.200s,
ancestry race149.324s, migration9.239s and static checks pass. Original38 reproduces
and refuses actual populated39 without modification. The fixture-only monotonic
timestamp mismatch and corrected UTC clock are retained in the evidence.

Supersedes only M167's pending persisted observation and M168's pending successor
refusal. Full39 regression, mutating continuation, public routes and runtime
observation remain open. VM31 is frozen38 and excludes this source. See
diagnostics/2026-10-24-persistent-recovery-observation.md.

## M170 — Known Create replies have a separate Start-only continuation path (2026-10-24, source-qualified)

Pure classification separates new steps from uncertain outcomes. A retained exact
Create ID can be freshly inspected and started through an explicit Start-only
adapter; no Create method is exposed. Unknown Create and pending/already-started
states are refused. Lost Start replies reconcile without replay or invented ack.
Pure race93.618s, concrete race73.741s, related readiness/second-recovery139.211s
and static checks pass. Initial missing-container fixture failures remain recorded.

This advances, but does not supersede, M169's pending mutating-continuation queue.
No new format/public route/runtime authority is wired. Worker/store drafts remain
in /tmp until the full39 baseline and original-writer snapshot are qualified.
See diagnostics/2026-10-24-created-recovery-continuation.md.

## M171 — Full schema39 regression passes (2026-10-24, source-qualified)

The full managed race suite passes in1722.999s, followed by successful vet. The
443-file input audit finds no changed production source; one later inert adapter
test correction has its separate M170 race evidence. This supersedes only the
pending full39 regression in M169/M170. Temporary continuation overlay cases also
pass in12.554s but remain an unintegrated prototype, with migration/restart and
runtime qualification still required. VM31 remains failed at cleanup, with
independent disk inspection pending. No release or production operation occurred.
Evidence: diagnostics/2026-10-24-schema39-full-regression.md.

## M172 — Trial31 partial recovery evidence verified; full trial failed (2026-10-24)

Independent stopped-disk checks confirm coordinated operation/member mirrors,
same-pier successors and sixteen application writes across four subscription
cycles with two original sessions and no later login/replay. Both final members
exited0 during engine shutdown. The fixture rejected their correct extra namespace
and retirement ownership labels before its explicit final Stops, so the overall
trial remains failed and final fixture shutdown is unqualified. Original disks
and58 evidence files are retained. This supersedes M170/M171's pending extraction
and tentative label diagnosis, not the requirement for a clean fresh runtime pass.
Verifier bus-access/variable-shadowing failures are retained separately.
Evidence: diagnostics/2026-10-24-anchor-recovery-trial31.md.

## M173 — Recovery cleanup contract corrected; fresh trial32 running (2026-10-24)

The fixture now requires exact namespace/retirement labels before and after Stop.
Both actual31 ownership records and negative cases pass race10.190s/vet. Frozen
image32/schema39 independently verifies278 sources, artifacts and embedded code;
private host admission refusal also passes. Initial audit directory/order errors
are retained. This supersedes M172's pending source correction/preparation only;
the full runtime rerun is pending and excludes later continuation drafts.
Evidence: diagnostics/2026-10-24-recovery-cleanup-rerun.md.

## M174 — Original39 observations reproduce and refuse populated40 (2026-10-24)

The1625-file original39 archive (SHAe9afaa95) reproduces1095923 bytes across21 tables,
including interrupted/completed observations. Actual40 migration/unknown-Create
checks pass14.460s; original39 refuses populated40 unchanged. Initial generator
method/shallow-copy errors and omitted target assertions are retained. This
supersedes the pending original-writer prerequisite in M170/M171; private
continuation integration now exists, but race/full-regression and real runtime
continuation remain pending. Frozen VM32 stays39 and excludes this work.
Evidence: diagnostics/2026-10-24-schema39-original-writer.md.

## M175 — Explicit continuation passes focused race; runtime fixture ready (2026-10-24)

Schema40 continuation passes the focused managed race suite in 861.164s plus vet
and diff checks. Restart, atomic rollback, stale/corrupt mirrors, callback bypass,
unknown Create and replay cases are covered. The guarded three-reply-loss fixture
compiles; 44 harness checks and host admission refusal pass. It is not yet a VM
pass and covers store reopening, not OS runner-service restart. Full40 regression
is running. This supersedes only M174's pending focused race status. Frozen VM32
remains39. Evidence: diagnostics/2026-10-24-recovery-continuation-source.md.

## M176 — DMs hosted cookie parsing corrected (2026-10-24, source-qualified)

Standalone DMs login now accepts CookieJar's explicit same-host Domain spelling
and refuses ambiguous/wrong-scope sessions before saving. The regression first
reproduced two errors/two failures; 33 affected tests then passed in 0.078s using
real cookie parsing and inert transport/storage. No hosting-provider runtime or
TLS qualification is claimed. Theme/DMs docs now supersede their stale initial
"approval flow unconnected" wording with the existing public-flow/native evidence.
Full desktop profiles and provider tests remain open. Evidence:
diagnostics/2026-10-24-dms-hosted-cookie.md.

## M177 — DMs experimental launchers no longer share the default filename (2026-10-24)

Per-profile filenames, visible labels and exact ownership markers separate stable,
dev and edge Tlon Web entries. Default-only legacy recognition preserves existing
installations. Thirty-nine affected tests pass in 0.080s with temporary launcher
paths. This advances the pending profile gate but does not supersede the remaining
QML/IPC/hook and collection-switch requirements. No installed entry changed.
Evidence: diagnostics/2026-10-24-dms-profile-launchers.md.

## M178 — Full schema40 managed race regression passes (2026-10-24)

Full managed race2164.666s, vet and diff checks pass. All 874 starting inputs are
unchanged and retained byte-for-byte. This supersedes M174/M175's pending full40
regression; the staged public status API/CLI is outside this baseline. VM32 reports
a clean schema39 runtime pass, awaiting independent disk verification. Runtime
continuation remains unqualified. Evidence:
diagnostics/2026-10-24-schema40-full-regression.md.

## M179 — Coordinated recovery passes cleanly with retained state (2026-10-24)

VM32/schema39 passes2345.62s, clean shutdown/outer0 and independent stopped-disk
verification. Exact recovery/creator mirrors, same-pier namespace successors,
replay/reopen and final cleanup pass. Sixteen writes, eight channels and24 facts
span four cycles using two original sessions, no later login/replayed writes.
Original disks and46 metadata files are retained; only the verified raw conversion
was removed. This supersedes M172/M173's pending clean rerun, without relabeling
failed31. Schema40 continuation and public lifecycle runtime remain open.
Evidence: diagnostics/2026-10-24-anchor-recovery-runtime.md.

## M180 — Public saved-status route and CLI pass source gates (2026-10-24)

Runner capability, exact observer-readable route and collector `project-status`
now expose bounded saved operation summaries with explicit non-health semantics.
Applied managed race12.449s and client/access/collector races, vet and skill
validation pass. This supersedes the pending application/checks for the earlier
status draft, not runtime qualification or public mutation routes. Schema40 is
unchanged. The next guarded VM includes actual CLI reads after three interrupted
continuations; its owner UID and complete Go source manifest are explicitly bound.
Evidence: diagnostics/2026-10-24-project-saved-status.md.

## M181 — Schema40 continuation appliance frozen and running (2026-10-24)

Independent checks bind all 833 source inputs and embedded binaries to retained
appliance33; private host refusal passes. The no-NIC VM is admitted with isolated
rootless Docker and resource limits. It exercises three lost replies, three store
reopens, explicit observations/continuations and actual saved-status CLI reads.
This supersedes only M180's pending preparation; runtime and independent stopped
disk verification remain pending. Store reopen does not certify OS service restart.
Evidence: diagnostics/2026-10-24-anchor-continuation-preparation.md.

## M182 — Public project reservation and lookup-only CLI pass source gates (2026-10-24)

The operator-only route reconstructs a reviewed fresh-fake plan and atomically
reserves its membership/ports without runtime effects. The pinned CLI consumes
the offline JSON and preserves actor/request identity for lookup-only recovery.
Applied managed race14.868s, client1.045s, collector1.028s, vet and skill validation
pass. Corrected refusal tests reach their intended checks; the earlier content-type
short-circuit limitation is retained in the diagnostic. Schema40 and historical
store fingerprints are unchanged. This supersedes prior "registration unexposed"
statements only. Public lifecycle actions and combined real CLI/runtime dispatch
remain pending; frozen VM33 does not include this later source.
Evidence: diagnostics/2026-10-24-public-project-reservation.md.

## M183 — Initial public project actions and real-CLI lost-reply gate pass (2026-10-24)

Five separately capability-gated actions now delegate to the established queues:
prepare, anchor Start, member storage, fresh launch and initial readiness. Every
action requires exact reviewed inputs; lookup-only starts no worker. The actual
collector/private Unix server/inert store gate loses an accepted reply, retrieves
its original receipt and completes one worker without resending the mutation.
Applied managed race17.223s, client1.057s, collector1.046s, vet and skill checks
pass. Schema40/fingerprints are unchanged. This supersedes only M182's pending
initial public actions and inert combined CLI/runner dispatch. Public member
Stop/resume/recovery, actual Docker public-command qualification and QML remain.
Frozen VM33 predates this source. Evidence:
diagnostics/2026-10-24-public-initial-project-actions.md.

## M184 — VM33 failed on fixture misuse; correction passes source gates (2026-10-24)

A fresh reopened drain observer correctly refused AwaitDrainExit because it had
no in-memory acknowledged Stop. The fixture must poll InspectDrain on its saved
uncertain intent, then submit an explicit durable observation. Do not weaken the
adapter's acknowledgement requirement to accommodate this fixture mistake.
The corrected bounded, read-only wait passes race1.111s and existing concrete
adapter/lost-reply tests pass30.679s. VM33 is failed, shut down and its original
disk retained; extraction and rerun are pending. This supersedes only M181's
running status; M179 remains the prior uninterrupted recovery pass. Evidence:
diagnostics/2026-10-24-anchor-continuation-trial33.md.

## M185 — VM33 failed-run disk audit and retention complete (2026-10-24)

Independent stopped-disk checks preserve the single unacknowledged Stop and exact
interrupted checkpoint across one store reopen; no recovery observation or
continuation was queued. SQLite mirrors and19 extracted files match. Twelve
acknowledged writes and three earlier subscription cycles passed with two original
logins, but coordinated recovery/saved-status remained unreached. Both final pier
states were independently hashed. Original disk and44 metadata files are retained.
This supersedes M184's pending extraction only, not its failed runtime status.
Evidence: diagnostics/2026-10-24-anchor-continuation-trial33.md.

## M186 — Public member lifecycle and prerequisite refusals pass (2026-10-24)

Six explicit member actions now reuse the durable queues with exact prerequisite
receipt IDs and separately advertised capabilities. Replay-only never queues work.
Five formerly private admissions misclassified unknown user receipts as storage
corruption; fresh-admission checks now distinguish these while retaining genuine
corruption refusal and historical replay. Initial launch/readiness missing inputs
also no longer disable a healthy runner. Schema40/fingerprints remain unchanged.
Expanded race321.622s and applied focused race24.172s pass with client/protocol/CLI,
vet, diff and skill checks. This supersedes M183's pending public Stop/member-resume
source gate only. Coordinated recovery APIs, public-command Docker qualification
and QML remain. Evidence: diagnostics/2026-10-24-public-member-actions.md.

## M187 — Corrected continuation appliance34 verified and running (2026-10-24)

853 source hashes and embedded artifacts match the retained appliance; host refusal
passes. The no-NIC VM repeats the interrupted recovery and actual saved-status
checks with the corrected read-only uncertain Stop wait. M186 source is frozen
into the image, but public mutation CLI dispatch is not tested by this fixture.
This supersedes M184/M185's pending rerun preparation only. VM33 remains failed;
VM34 runtime and independent disk checks are pending. Evidence:
diagnostics/2026-10-24-anchor-continuation-rerun-preparation.md.

## M188 — Public saved recovery checkpoint read passes (2026-10-24)

Observer-readable checkpoint summaries expose exact phase/sequence/history/checkpoint
digests without opening Docker or exposing private journals. Hints grant no action
permission; unknown-ID Create remains review-required. Managed race98.549s verifies
zero store/effect changes and preserved uncertainty; client/access/CLI, vet, diff
and skill checks pass. This supplements M186 with the saved recovery read surface,
not recovery effects. Schema40 is unchanged and frozen VM34 predates this route.
Evidence: diagnostics/2026-10-24-public-recovery-checkpoint.md.

## M189 — Saved recovery source and exact member references pass (2026-10-24)

The observer-readable recovery-source command supplies a validated saved history
digest and creator/original/current readiness references without runtime effects.
Missing initial prerequisites remain refusals; corrupt stored history fails closed.
Managed race9.988s and client/access/CLI, vet, diff and skill checks pass. This
supersedes only M188's pending initial saved-source selection. Existing interrupted
work must retain its own checkpoint source. Public recovery actions and runtime
qualification remain; frozen VM34 predates this source. Evidence:
diagnostics/2026-10-24-public-recovery-source.md.

## M190 — Public recovery actions and actual collector reconciliation pass source gates (2026-10-24)

Explicit start/observe/continue now share existing durable queues with pinned
identity, operator/capability checks and exact reviewed source/checkpoint inputs.
Unknown-ID Create stays review-required. Missing user selections refuse ordinarily;
corrupt journals fail closed. Schema40 and historical fingerprints are unchanged.
Managed154.260s/refusal27.069s, related regression378.406s and actual-collector
inert lost-reply62.226s pass, plus client/protocol/CLI, vet, diff and skill checks.
This supersedes only M188/M189's pending public recovery wiring. VM34 predates these
commands. Real public Docker and UI qualification remain; no installation changed.
Evidence: diagnostics/2026-10-24-public-recovery-actions.md.

## M191 — Actual public recovery VM fixture passes source/admission gates (2026-10-24)

The new opt-in fixture routes recovery start/observe/continue through the actual
collector, guest-private enrollment and public source/checkpoint reads, retaining
exact lookup inputs and receipts. Existing continuation faults/application tests
remain. Compilation, focused inert race1.130s, two entry refusals, actual host
refusal, vet and diff pass. This advances M190's next fixture; no runtime pass is
claimed and VM34 predates it. Full lifecycle/UI and OS service restart remain.
Evidence: diagnostics/2026-10-24-public-recovery-vm-source.md.

## M192 — Interrupted recovery passes with persistent application state (2026-10-24)

VM34/schema40 passes2505.19s, outer0/clean shutdown and independent stopped-disk
verification. Three lost replies and three store reopens lead through two explicit
observations/three continuations with exact once-only effects. Sixteen writes,
four subscription/state cycles and the same two original Eyre sessions pass;
both final piers and actual public saved-status projection are verified. All
audit attempts are retained, including a correction requiring the exact legitimate
parent-message transition. This supersedes M174/M175/M187's pending private runtime
gate and M184/M185's pending corrected rerun; VM33 remains failed. Public mutation
CLI, OS service restart and UI are separate. Evidence:
diagnostics/2026-10-24-anchor-continuation-runtime.md.

## M193 — Actual public recovery appliance35 verified and running (2026-10-24)

883 source hashes and embedded binaries match the retained no-NIC appliance;
host refusal passes. A fresh dedicated12GiB/no-swap/one-core VM runs the M191
actual collector source/checkpoint/recovery path after M192's private reference
completed and was retained. This supersedes only M191's pending preparation.
Runtime and independent verification remain pending; no initial/member public
lifecycle, OS service restart or UI claim is made. Evidence:
diagnostics/2026-10-24-public-recovery35-preparation.md.

## M194 — DMs profile routing passes private QML gate (2026-10-24)

Host-injected IDs now reach overlay close/layer/labels and nested popup launch,
including late injection. Three source roots keep separate singleton state and
handlers. Four behavioral tests pass with native controls/inert I/O, zero warnings;
unchanged baseline fails. Explicit Commons.Color resolves an existing ambiguity;
instanceLabel avoids a draft collision with contact displayName. All trials are
retained. This supersedes only M177's pending QML routing; native compositor/IPC,
hooks and collection switching remain. Installed plugins are unchanged. Evidence:
diagnostics/2026-10-24-dms-profile-ui.md.

## M195 — Theme profile UI and real IPC pass privately (2026-10-24)

The native host injects service identity after construction: leave IPC disabled
until that identity is valid, then derive its own address. Three profile UI cases,
eighteen existing UI cases and four real IPC calls pass in a networkless namespace
with inert helpers; cleared dev identity removes only dev's address. Baseline and
fixture failures are retained. This qualifies Theme panel/service routing only;
hooks, appearance ownership, native desktop and collection switching remain. No
installed plugin changed. Evidence: diagnostics/2026-10-24-theme-profile-ipc.md.

## M196 — Theme profile hooks and owned files pass; visible desktop remains shared (2026-10-24)

A real metadata-selected dev notification formerly deleted stable's appearance
file; the unchanged-source private regression reproduces it. Instance paths now
preserve stable, including generated themes/backups. Hook filenames, helper paths,
IPC targets and ownership receipts are separate. Applied36 file/sync tests,21 QML
cases and actual private hook/IPC pass. This supersedes M195's pending hook/file
routing only. Applying a theme still targets shared desktop state and multiple
includes can affect one compositor; collection activation must be exclusive and
restore the prior collection. Names alone are not a sandbox. Evidence:
diagnostics/2026-10-24-theme-profile-files.md.

## M197 — Public recovery VM35 retains uncertainty but fails checkpoint timeout (2026-10-24)

Independent failed-run audit passes: clean VM shutdown, schema40, three lost
replies/store reopens, exact partial effects, four new/three lookup public requests,
twelve writes/three state-subscription cycles and two original sessions. The
public checkpoint GET times out at replacing/sequence22; cause is unconfirmed.
Original disk, databases, final pier hashes and all evidence are retained. This
supersedes M193's running status only; M192 remains a separate private pass. No
completed public recovery or final saved-status gate is claimed. Diagnose the
read before retrying; public-initial draft remains parked. Evidence:
diagnostics/2026-10-24-public-recovery35-failure.md.

## M198 — Recovery read timeout reproduced and redundant validation removed (2026-10-24)

Immutable VM35/VM34 database probes in fresh no-disk/no-NIC VMs reproduce5-second
read failures with unchanged source (6.3s interrupted,11–12s complete). The applied
change validates embedded ancestry once and resolves each member history once
per context selection; all ownership, proof and sequence checks remain. Revised
reads meet the same deadline in2.2–3.6s with identical projections and unchanged
databases. Full project-network race and vet pass; the broader managed run hit
its aggregate10-minute test limit and a longer bounded rerun remains pending.
Both drafts/negative trials are retained. This supersedes M197's unconfirmed cause
with a reproduced validation-cost failure; VM35 remains failed. Fresh full public
Docker recovery is still required. Evidence:
diagnostics/2026-10-24-recovery-read-validation.md.

## M199 — Fixed public recovery appliance36 verified and running (2026-10-24)

885 source hashes and embedded artifacts match the retained no-NIC appliance;
host refusal passes. A fresh12GiB/no-swap/one-core VM exercises M198's read-cost
fix through the complete public recovery/application scenario. This advances
M198's pending fresh trial to running, not passed. Its broader managed race rerun
is also pending. VM35 remains failed and M192 remains the private reference.
The public-initial draft passes compilation/admission checks but is still unapplied.
Evidence: diagnostics/2026-10-24-public-recovery36-preparation.md.

## M200 — Public initial-lifecycle fixture passes source/admission gates (2026-10-24)

The actual collector now has a separately guarded VM fixture for offline plan,
reviewed registration, bridge/anchor preparation, anchor Start and two empty
member parents, with exact lookup receipts and original durability/reopen checks.
It boots no ship. Applied compilation/admission race1.025s, CLI race1.064s,
three entry refusals, actual host refusal, vet and diff pass. Cancellation precedes
worker waits and the shared storage proof waits for both members. This supersedes
M199's unapplied draft status only; it has not run in a VM and VM36 excludes it.
Evidence: diagnostics/2026-10-24-public-initial-vm-source.md.

## M201 — Applied recovery validation broader race passes (2026-10-24)

The selected managed recovery/checkpoint/source/public-action/read-view/continuation/
observation race tests pass696.430s under a20-minute aggregate limit. Preserve the
earlier10-minute timeout and all873 source hashes; M200 fixture/help edits postdate
this binary. This supersedes M198/M199's pending broader rerun only, not a whole
managed-suite or public Docker runtime gate. Evidence:
diagnostics/2026-10-24-recovery-read-validation.md.

## M202 — VM36 interrupted; disk and WAL retained, no runtime result (2026-10-24)

Outer143 left no driver or project result; the exact scope is inactive and cause
is unclassified. Read-only extraction plus a private WAL-recovery copy passes
SQLite checks at schema40: one project/two members, five successful setup operations
and one interrupted launch. Original disk/database/WAL/SHM hashes are retained.
No clean shutdown, full application cycle or public recovery pass is claimed.
This supersedes M199's running status; M198/M201 source checks and M192 private
runtime evidence stand. Initial public lifecycle is next, then a fresh full public
recovery retry. Evidence: diagnostics/2026-10-24-public-recovery36-interrupted.md.

## M203 — Public initial-lifecycle appliance37 verified and running (2026-10-24)

All884 source hashes, embedded artifacts and the opted-in host refusal pass. A
fresh no-NIC8GiB/no-swap/one-core VM exercises the actual collector's reservation,
preparation, anchor Start and two empty member parents. No ship is booted. The
independent verifier is prepared, not run. This supersedes M200's pending VM
preparation only; runtime qualification remains. VM36 is still incomplete.
Evidence: diagnostics/2026-10-24-public-initial37-preparation.md.

## M204 — Backend install patterns verified against four source snapshots (2026-10-24)

The developer skill now contains kiln-install/revive shapes and source-preserving
missing/pending/suspended/readiness decisions. Omart's complete status flow is
useful; Talon's acknowledgement/readiness split is useful but false-on-exception
probes cannot authorize installation. Pikes absence alone is insufficient and a
client-specific groups publisher is not universal. Grubbery framework/code/instance/
route/permits remain separate. Lattice's historical blanket restart-cookie note
does not supersede M005/M192's measured original-session continuity. Exact source
hashes, skill validation and diff checks are retained. This extends M003; actual
plugin installer/compatibility runtime qualification remains, not a new Omabit
catalog. Evidence: diagnostics/2026-10-24-backend-setup-references.md.

## M205 — Public initial lifecycle passes actual CLI and Docker gate (2026-10-24)

VM37 passes293.44s with clean shutdown and first-attempt independent disk audit.
Five new/five lookup requests and three duplicate-new refusals preserve pinned
runner/actor, once-only effects, exact schema40 mirrors and empty0700 member
parents. Only the namespace anchor starts and stops; no ship boots. All originals
are retained. This supersedes M200/M203's pending initial runtime gate only. Public
member lifecycle, full public recovery, OS service restart and UI remain separate;
VM35 is failed and VM36 incomplete. Evidence:
diagnostics/2026-10-24-public-initial-lifecycle-runtime.md.

## M206 — Fresh public recovery retry38 running on verified frozen appliance (2026-10-24)

The new no-NIC12GiB/no-swap/one-core VM uses an untouched fresh overlay and exactly
the885 source/artifact inputs from appliance36, including M198. M200 and the
member CLI draft are excluded. This advances M202's pending retry to running,
not passed; the interrupted36 originals remain preserved and M205's separate
initial lifecycle pass stands. Evidence:
diagnostics/2026-10-24-public-recovery38-preparation.md.

## M207 — Public member lifecycle fixture passes applied source gates (2026-10-24)

A separate guarded VM entry routes26 existing member queue sites through actual
collector commands, retaining exact prerequisites, lookup-only receipts, duplicate
refusals and direct application state/subscription checks. Private paths remain.
Applied race24.518s, four Python entry tests, vet/diff and identical-draft actual
host refusal pass. The non-executable evidence-copy test failure and corrected
private-copy rerun remain recorded. This advances M205's pending member fixture
only; runtime qualification has not run and frozen VM38 excludes it.
Evidence: diagnostics/2026-10-24-public-members-vm-source.md.

## M208 — Read-only project history UI passes private QML test (2026-10-24)

Actual QML/Quickshell Process reads only pinned inventory/history and shows saved
status, observer access, profile identity, interruptions and truncation. Sixteen
inert read calls pass, including malformed/oversized/timed-out replies and a
completed late old-profile response discarded after switching. Source and screenshots
are retained; installed plugins are unchanged. This advances M205's UI gap only
for read-only pane/controller behavior, not full shell/native compositor, real-runner
UI or mutating project controls. Evidence: diagnostics/2026-10-24-project-history-ui.md.

## M209 — VM38 preserves interrupted first launch; full recovery not reached (2026-10-24)

The713.68-second retry shuts down cleanly but fails post-Start ownership observation
for its first fake member. Its exact `started` journal/receipt remains interrupted;
specific ownership versus freshness cause is unknown. Schema40 mirrors and five
successful setup operations pass independent audit. Member cleanup escalates after
300seconds and exits137 without reported OOM; anchor cleanup passes. Original disk,
results and failed/corrected audits are retained. Storage admission was preserved
by deduplicating only verified copies of completed probe artifacts. This supersedes
M206's running status; M198's recovery read fix was not reached, and M192/M205
remain separate passes. Next is precise observation diagnostics and a fresh run.
Evidence: diagnostics/2026-10-24-public-recovery38-failure.md.

## M210 — Member observation diagnostics distinguish freshness from ownership (2026-10-24)

Specific refusal categories and elapsed-time freshness diagnostics preserve all
ownership predicates and the30-second bound. Elapsed time now retains the monotonic
clock; persisted timestamps stay UTC. Project-adapter race262.543s, applied focused
race1.335s and vet pass; failed draft/sandbox/application-check attempts and the
separate baseline run remain recorded. This advances M209's diagnostic work only;
VM38's precise cause is still unknown and a new runtime trial is required.
Evidence: diagnostics/2026-10-24-member-observation-diagnostics.md.

## M211 — Public member-lifecycle VM39 verified and running (2026-10-24)

All887 source inputs and embedded artifacts match, and opted-in host refusal
passes. A fresh no-NIC12GiB/no-swap/one-core VM runs initial/member public commands
with direct application state/subscription checks. M207 and M210 are included;
the independent verifier is prepared, not executed. This supersedes M207's pending
member runtime preparation only. VM38 remains failed; full public anchor recovery,
OS service restart and mutating UI remain. Evidence:
diagnostics/2026-10-24-public-members39-preparation.md.

## M212 — Project history reuses the existing compact time display (2026-10-24)

Visual review found verbose locale-dependent timestamps in M208's pane. It now
uses Omabit's existing `Model.shortTime` formatter. All16 private QML read cases
still pass with zero mutations; the new screenshot was inspected and retained.
This supersedes only M208's timestamp rendering. Full-shell/native-desktop and
real-runner UI qualification remain separate. Evidence:
`/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-project-history-time`.

## M213 — Native loader overlap and retained generations reproduced privately (2026-10-24)

Four exact native service-loader functions create incoming QML before destroying
outgoing QML; `keepLoaded` retains original code/helpers under refreshed manifest
metadata. Eight synthetic cases pass with real private Quickshell/Process, including
explicit child-exit waits, distinct generation replacement and stable/dev/edge
roundtrips. Host refusal and all trial/native-source evidence are retained. This
advances M196's collection lifecycle research; it does not qualify full shell or
real plugin watcher/appearance/credential switching. Developer guidance now requires
separate quiesce/disable/verify/enable and actual-generation readiness. This remains
cooperative lifecycle isolation, not a wallet-key or same-user security boundary.
Evidence: diagnostics/2026-10-24-collection-loader.md.

## M214 — Public member VM39 retains interrupted Start after engine read failure (2026-10-24)

The333.96-second clean VM run fails before application checks. Actual CLI initial
actions pass; first member Start is recorded, then engine identity API evidence
is unavailable. Docker records canceled `/version`; exact deadline/cancellation
origin is unconfirmed. Six new/five lookup requests, three refusals and schema40
mirrors pass independent audit. One member Create/Start only; cleanup exits143,
not the clean Urbit gate. Original disk/evidence are retained. This supersedes
M211's running status; M210's later observation is not reached. Test bounded
two-core emulation next without changing production checks; resource sensitivity
remains a hypothesis. Evidence: diagnostics/2026-10-24-public-members39-failure.md.

## M215 — Public member retry40 runs with explicitly bounded two-core emulation (2026-10-24)

Fresh no-NIC overlay40 reuses exactly appliance39's887 guest inputs. Scope admission
confirms200% CPU/multi-thread TCG, unchanged12GiB/no-swap and two guest CPUs. The
opt-in host-driver setting passes three CPU/three entry tests; one core remains
default and non-runtime/unbounded increases refuse. This tests M214's resource
sensitivity hypothesis; production timeouts/ownership/freshness stay unchanged.
Independent verifier is prepared, not run. This supersedes M214's pending retry
only; VM39 remains failed. Evidence: diagnostics/2026-10-24-public-members40-preparation.md.

## M216 — Private chain contracts, admission and database reopen pass (2026-10-24)

Pinned public Bridge/Azimuth and Solidity0.4.24 reproduce the six local addresses.
The isolated fixture deploys nine contracts/13 receipts and preserves bytecode,
ownership/genesis/receipts on reopen; wrong chain/genesis/absent code refuse before
writes. Confirmed4GiB/no-swap/one-core, no network or ships. Native uWS/asm.js fallback
warnings and all prototypes are retained. This supersedes M026's research-only
state for the chain foundation, not its pending ship-devnet gates. Vere dawn source
uses getPoint/getDns gateway methods: raw Ganache RPC is insufficient for keyed
boot; actual pinned binary capture and private gateway remain. Public deployment
seed is development-only, not wallet storage or fresh ship network keys. Evidence:
diagnostics/2026-10-24-private-chain-foundation.md.

## M217 — Pinned Vere confirms the private boot gateway requirement (2026-10-24)

Actual isolated `vere eval` generates getPoint/getDns and256 distinct galaxy
requests from its embedded ivory. Independent decoding/host refusal/scope checks
pass. No keys, pier, HTTP call or ship boot occurs. This supersedes M216's source-
inference-only status for request generation, not its pending gateway/ship gates.
A raw chain RPC endpoint is insufficient; keep the private gateway chain-bound
without public fallback. Evidence: diagnostics/2026-10-24-private-boot-requests.md.

## M218 — Fresh private-chain keys and dawn responses pass pinned validation (2026-10-24)

Two fresh Suite B network keys registered on a copied admitted chain pass the
pinned Vere dawn validator; wrong public keys/revisions refuse. Chain-backed
point/256-galaxy/DNS responses parse, with exactly two keyed galaxies and two
new receipts. All private trial scopes stopped; failed formatting/type/interface
probes and the final pass are retained, excluding secrets. This advances M217
from request generation to response/key interoperability, not HTTP or ship boot.
Pinned ivory uses crub and lacks current-source cric; never assume snapshot/API
parity. Private boot still needs local discovery, gateway, pill observation and
peer/restart checks. Evidence: diagnostics/2026-10-24-private-key-validation.md.

## M219 — Native keyed boot reaches identity; shutdown gate fails (2026-10-24)

Fresh private chain-backed HTTP boot verifies keys and the second probe reaches
public identity. Both native parent/worker processes ignore SIGTERM and exceed
shutdown bounds; only exact owned processes are force-stopped and all evidence
retained. Probe01 also had a readiness URL/port mistake;02 corrects it. Namespace
entry for a Lens exit is OS-refused, not approval-rejected. Cause remains unknown;
compare launcher/session signal handling before attributing a runtime defect.
This advances M218's HTTP/bootstrap evidence only, not a clean boot/stop or devnet
pass. Draft remains unapplied; scopes inactive and secret files excluded. Evidence:
diagnostics/2026-10-24-private-keyed-boot-shutdown.md.

## M220 — Public member lifecycle and application recovery pass independently (2026-10-24)

VM40 passes2093s under the admitted two-core/no-NIC cap. Thirty actual CLI requests
and30 exact lookups match schema40;16 duplicates refuse. Three retire/resume cycles
preserve12 writes, six subscriptions/18 snapshots and two original sessions, with
no write replay. Final piers/database/history are independently verified and
retained. The outdated audit transport-field assertion is corrected to require
exact routing; both attempts remain. Storage reserve refusals remain unchanged;
only verified completed copies are deduplicated. This supersedes M215's running
member gate, not M214's failed trial or M219's native shutdown failure. Full public
interrupted recovery, OS service restart and UI remain. Evidence:
diagnostics/2026-10-24-public-member-lifecycle-runtime.md.

## M221 — Fresh public interrupted-recovery VM41 admitted (2026-10-24)

Appliance41 freezes887 inputs and independently verifies artifacts, embedded
binaries and host refusal. The fresh recovery-only no-NIC VM has admitted
12GiB/no-swap/two-core limits; all production deadlines remain unchanged.
Its six-request/six-lookup recovery and application audit is prepared, unrun.
This advances M220's next-gate statement to running, not a recovery pass.
OS service restart/UI and M219 native shutdown diagnosis remain separate.
Evidence: diagnostics/2026-10-24-public-recovery41-preparation.md.

## M222 — Keyed SIGTERM failure traced to duplicate runtime cleanup (2026-10-24)

The same Node launcher cleanly stops a fresh fake. A separate private keyed
pre-worker trace shows pinned5a7614f keyfile parsing calls `_cm_signal_done`
twice: the second leaves SIGTERM ignored after libuv has resumed. Signal masks
confirm it before worker creation. This supersedes M219's unknown-cause status
only; initial keyed shutdown remains failed and no runtime fix is deployed.
Filtered stacks exclude values and secret bootstrap output. Lens exit/reopen
is the next separate qualification. Evidence:
diagnostics/2026-10-24-keyed-signal-diagnosis.md.

## M223 — Lens exit does not bypass the pinned keyed shutdown failure (2026-10-24)

Two stopped private trials are retained. The first has a Lens identity-newline
fixture error; the second validates identity but `|exit` times out and does not
stop the process. Both require exact-child forced cleanup; reopening never runs.
This supersedes M222's pending Lens workaround, not its established runtime cause.
A separately labeled source-build correction and behavioral regression are next;
no deployed runtime or user ship changes. Evidence:
diagnostics/2026-10-24-private-lens-shutdown.md.

## M224 — Experimental signal fix passes real keyed stop and reopen (2026-10-24)

The original pinned code fails SIGTERM delivery regression; the minimal source
correction passes seven deliveries plus pinned boot/noun tests. An explicitly
labeled, hash-pinned build then boots fresh private keyed~zod, exits0 on SIGTERM,
reopens the same pier without bootstrap requests and exits0 again. Source hashes,
final databases and inactive scope are independently verified. This advances
M223's pending experimental build, not the official runtime's failure. No binary
is installed or released; chain watcher, peers and Docker devnet remain open.
Evidence: diagnostics/2026-10-24-private-fixed-keyed.md.

## M225 — Canonical seed literals prevent intermittent fixture syntax failures (2026-10-24)

Leading zero nibbles in a fresh random seed can produce noncanonical Hoon hex.
The reusable key validator now normalizes representation without changing the
256-bit input value. Actual evaluator rejection/acceptance regression and a
fresh full chain/key-validation run pass; the failed pre-boot attempt is retained
and created no chain or pier. This supersedes M218's current formatter, not its
valid earlier receipts. No production key/storage behavior changes. Evidence:
diagnostics/2026-10-24-private-chain-hex.md.

## M226 — Lens watcher setup acknowledged; confirmation fixture corrected next (2026-10-24)

Three failed private watcher trials are retained and cleanly stopped. Lens needs
generator syntax and a separate app sink; all five commands acknowledge in04.
Its successful RPC log request stops at head minus30, before the new keys.
Fresh05 mines64 empty blocks and remains pending. This supersedes assumptions
behind earlier setup drafts, not the pending watcher gate. M223's incorrect
`|exit` source cannot rule out a correctly encoded Lens exit; M222's independent
signal diagnosis stands. Verified stopped pier duplicates are deduplicated only
after exact hash/mode checks. Evidence: diagnostics/2026-10-24-private-watcher-setup.md.

## M227 — Public recovery runtime audited; final replay actor fixture corrected (2026-10-24)

VM41 remains failed at its final hard-coded-actor check. Independent stopped-disk
audit passes six requests/six lookups, three interrupted continuations and16
app writes/eight subscriptions across four cycles using original sessions. A copy
of its real database reproduces the wrong-actor failure and passes the corrected
saved-actor, lookup-only helper without changing36 receipts. Only that fixture
changes in source. This supersedes M221's running status, not its outstanding
overall clean-run/OS-service gates. Original evidence and storage refusal retained.
Evidence: diagnostics/2026-10-24-public-recovery41-actor-audit.md.

## M228 — Actual owner-service restart fixture applied with host refusal (2026-10-24)

The VM-only fixture transfers the same installed empty managed store between
fixture and actual owner service at three recovery cuts. It requires changed
PID/invocation, the frozen executable, unchanged pinned collector checkpoint/
history and identical project containers, then cleanly stops before SQLite
reopen. Source checks, vet, five entry and three storage checks pass; opted-in
host execution refuses before effects. This advances M227's next-gate draft
to applied source only. Runtime remains pending; production code is unchanged.
Evidence: diagnostics/2026-10-24-service-restart-source.md.

## M229 — Private-chain key revision and Jael persist across keyed restart (2026-10-24)

Experimental trial07 observes actual fresh-key life1→2 in Azimuth and Jael,
reopens the same pier retaining2, and exits cleanly twice. Independent copied-
chain audit verifies unchanged genesis/code/head, four receipts and three key
events without transactions; final databases are hashed. Failed05/06 show that
identical keys are a contract no-op; timer configuration is separately corrected.
This supersedes M226's pending watcher gate, not M224's official-runtime limit.
Two keyed peers and Docker devnet remain unqualified. Evidence:
diagnostics/2026-10-24-private-watcher-recovery.md.

## M230 — Actual service-restart VM42 admitted with frozen source (2026-10-24)

Appliance42 independently verifies888 inputs, schema40, embedded artifacts and
host refusal. Its fresh no-NIC VM runs at12GiB/no swap/two cores with unchanged
production deadlines. Three real owner-service restart cycles and full public
recovery/application evidence are the pending gate; the independent process/
checkpoint/container audit is prepared, unrun. This advances M228 from source
to running, not a runtime pass. Evidence:
diagnostics/2026-10-24-service-restart42-preparation.md.

## M231 — Two-keyed-peer fixture needs more than the single-ship memory cap (2026-10-24)

Trial01 hits its4GiB cgroup cap booting peer2; exact unit reports oom-kill and
later no remaining cgroup. Immediate incomplete cleanup metadata is retained
unchanged with later evidence. No messages or clean shutdown pass. Piers/chain
are retained and hashed. Fresh02 has admitted6GiB/no swap/one core, requires
8GiB available before launch, and remains running. This advances M229's next
peer gate only; official runtime/Docker remain unqualified. Evidence:
diagnostics/2026-10-24-private-peers-oom.md.

## M232 — Private keyed peer messaging passes; creation flag blocks fixture reopen (2026-10-24)

Trial02 passes two local/two cross-ship receipts, intentional Gall rejection,
private peer key observation and cross-ship cookie refusal. Its initial receiver
and sender stop cleanly; the new child refuses creation-only `-H` on reopen.
The overall failed evidence and exact inactive scope are retained. Fresh03 uses
the tested boot/reopen command builder in a new isolated6GiB scope. This supersedes
M231's running status, not the unqualified restart/official-runtime/Docker gates.
Evidence: diagnostics/2026-10-24-private-peers-reopen.md.

## M233 — Private keyed peers pass messaging and same-pier receiver restart (2026-10-24)

Fresh03 passes six exact receipts, one intentional Gall rejection, separate-session
isolation and both directions after receiver restart using original sessions.
All three stops are clean. Independent retained-pier and copied-chain audits
verify four databases, both life1 public keys, three receipts/two key events and
unchanged genesis/code/head without transactions. This supersedes M232's pending
native restart gate only; the experimental runtime is not installed, and Docker
or keyed application-state recovery remains unqualified. Correct official Lens
exit is running separately. Evidence: diagnostics/2026-10-24-private-peers-recovery.md.

## M234 — Actual Theme/DMs empty-account helper teardown passes collection roundtrips (2026-10-24)

Two private offscreen runs pass five stable/dev/edge selections each with exact
old-process exit before new enable, unmodified Python helpers and preserved,
distinct profile device IDs. DMs requires overlay-loader teardown in addition to
services. Applied portable harness and host refusal pass. This advances M213's
synthetic-helper limit for Theme/DMs empty accounts only. Connected subscriptions,
pending/uncertain writes, Omabit service, appearance and durable rollback remain.
Evidence: diagnostics/2026-10-24-real-plugin-lifecycle.md.

## M235 — Correct Lens exit and same-pier SIGTERM reopen pass on official Vere (2026-10-24)

Fresh official trial03 still observes ignored SIGTERM, but `+hood/exit` with
sink.app hood acknowledges and exits0 in4805ms. Same-pier reopen retains identity,
needs no new bootstrap requests and exits0 on SIGTERM in4277ms. No forced cleanup;
independent inactive-scope/two-database audit passes. This supersedes M223's Lens
workaround conclusion (qualified by M226), not M222's signal defect. No installed
binary changes. Official two-peer/Docker lifecycle remains to qualify. Evidence:
diagnostics/2026-10-24-official-lens-recovery.md.

## M236 — Official-runtime private keyed peer messaging/restart passes (2026-10-24)

Fresh04 passes six receipts, intentional rejection, original-session restart and
three clean stops on the official binary. Two initial stops use acknowledged
Lens exit; reopened receiver uses SIGTERM. Independent inactive-scope/four-database
and copied-chain public-key/receipt audits pass. This advances M233's experimental-
runtime-only peer gate using M235's explicit workaround, not a SIGTERM defect fix.
Docker/runner integration remains open. Fresh05 is running persistent Gall state
and subscription recovery. Evidence: diagnostics/2026-10-24-official-private-peers.md.

## M237 — Official keyed persistent state and subscription recovery passes (2026-10-24)

Fresh05 passes four unique remote writes, seven scries and six subscription
snapshots across two channels. The original session/channel resumes its cursor
and queued update after receiver restart; a fresh channel reconciles, and both
are deleted. Three installation receipts, installed source, four databases and
copied-chain identity/public keys/receipts pass independent audit. All stops are
clean using M235's official-runtime workaround. This supersedes M236's pending
native application gate only; Docker/runner and profile lifecycle remain open.
Evidence: diagnostics/2026-10-24-keyed-application-recovery.md.

## M238 — Reproducible native keyed recovery stage applied and admitted (2026-10-24)

The official keyed recovery fixture is now `private-chains/run.py --stage
native-peers`, with explicit inputs, helper hashes,8GiB available admission and
6GiB/no-swap/one-core scope. Existing foundation limits stay unchanged. Two
host-refusal/argv tests and syntax checks pass. Fresh06 independently matches
applied source and is running. This advances M237's integration plan only; the
applied-stage runtime result and Docker/runner integration remain pending.
Evidence: diagnostics/2026-10-24-native-keyed-source.md.

## M239 — Applied native keyed stage passes fresh application/reconnect runtime (2026-10-24)

Run06 independently matches applied source and passes the complete keyed peer/
application gate: seven attempts, four writes, seven scries, six snapshots/two
channels, original sessions/cursor and three clean stops. Installed Gall source,
four databases and copied-chain public keys/receipts/genesis/code pass audit;
all evidence is retained. This supersedes M238's pending applied-stage result.
Docker/runner integration and the first-boot SIGTERM defect remain separate.
Evidence: diagnostics/2026-10-24-native-keyed-recovery-runtime.md.

## M240 — Tend missing-backend experiment inputs preserve generated desk compatibility (2026-10-24)

Frozen25-file Tend overlay excludes static Kelvin409 and preserves the four
ship-generated minimal marks. Its client readiness contract is snapshot protocol2,
separate from Gall state11. Source records and hashes are retained without running
its global installer, modifying Tend or touching a ship. This refines M204's next
backend runtime gate, not an installation pass. Use only fresh private publisher/
receiver ships; keep the eventual button and direct data flow in the plugin and
discovery in Omart. Evidence: diagnostics/2026-10-24-tend-backend-inputs.md.

## M241 — Fresh private Tend backend distribution trial admitted (2026-10-24)

Trial07 now tests M240's frozen overlay between fresh private keyed publisher and
receiver ships, with absence/source checks, generated-file preservation, direct
kiln-install JSON, protocol2 readiness and receiver restart. Unapplied fixture
syntax/host refusal pass and6GiB/no-swap/one-core scope is admitted. This advances
the source-only next step to running; it is not an install/UI pass. No global
installer or existing ship/plugin is used. Evidence:
diagnostics/2026-10-24-tend-trial-admission.md.

## M242 — Tend trial07 retains uncertain publisher setup; no install pass (2026-10-24)

Trial07 fails its new-desk acknowledgement before overlay/backend installation.
Both private ships stop cleanly and exact inactive scope/data-copy audits pass.
The response was not captured; do not infer no effect or repeat on this publisher.
An ask-generator dialog effect is a source-supported hypothesis. Fresh08 uses its
noninteractive equivalent and records the response. This supersedes M241's running
status only; protocol readiness and plugin UI remain unqualified. Evidence:
diagnostics/2026-10-24-tend-publisher07-failure.md.

## M243 — Real owner-service restart and public recovery pass VM42 (2026-10-24)

Three real service cycles preserve runner/store identity, checkpoints/history and
exact containers across seven distinct process generations. Six public requests/
six lookups, three lost replies and explicit continuation pass with exact effects.
All36 receipts,16 app writes/eight subscriptions/24 snapshots/four cycles survive
using original sessions without uncertain replay. Independent stopped-disk audit
and retention pass. This supersedes M230's running state and M227's pending clean
overall rerun, not VM41's original failure. Production/UI/Docker-devnet/release
gates remain distinct. Evidence: diagnostics/2026-10-24-service-restart42-runtime.md.

## M244 — Tend08 publisher type failure reproduced and corrected offline (2026-10-24)

Captured Lens nest-fail identifies an inferred fixed-length path list in fixture
setup. Both ships stop cleanly; original data and inactive-scope audit are retained.
A no-NIC evaluator reproduces failure and compiles the explicit `(list path)` gate
without executing it; evaluator exit0 alone is not success. Fresh09 is running.
This refines M242's fixture work, not its unrecorded07 response or the outstanding
real install/UI gates. Evidence: diagnostics/2026-10-24-tend-publisher08-type-failure.md.

## M245 — Tend09 identifies generic Lens mark; explicit marked source is next (2026-10-24)

The typed publisher expression compiles, then hood rejects mark noun. Both ships
stop cleanly and failure data/copies verify. Lens's explicit as/mark/next source
wrapper and inner helm-pass value pass serialization/host-refusal checks; fresh10
is running. This supersedes M244's running status, preserving its passing compile
gate and both earlier failures. Backend install/readiness/UI remain unqualified.
Evidence: diagnostics/2026-10-24-tend-publisher09-mark-failure.md.

## M246 — Connected helper/stream teardown passes; remote channels remain (2026-10-24)

Actual Theme/DMs helpers with private real keyring and synthetic Eyre pass two
five-selection collection runs;10 exact helpers and old streams stop before each
replacement. Default empty mode also passes the unified harness. Both source/helper
and inactive-scope audits pass. Ten synthetic channels remain undeleted per connected
run; process exit does not prove channel cleanup or pending-write completion. This
advances M234's empty-account limit only. Graceful busy cleanup, Omabit, appearance,
durable activation/rollback and real-Gall collection runtime remain. Evidence:
diagnostics/2026-10-24-connected-plugin-lifecycle.md.

## M247 — Tend10 publisher ready; acknowledged receiver install is not readiness (2026-10-24)

Marked setup and exact25-file overlay pass; publisher yields protocol2. Receiver
kiln-install acknowledges but readiness stays404 through deadline. Both ships stop
cleanly; source/generated-file and stopped-tree audits pass. Accepted install stays
pending, not replayed. Fresh11 adds explicit publication and permission observation
because new desks default private. This supersedes M245's running status, not the
receiver/UI gate. Evidence: diagnostics/2026-10-24-tend-receiver10-readiness.md.

## M248 — Fresh publisher read permission change verified; display-only fixture fails (2026-10-24)

Trial11 acknowledges public %tend and its raw permission atoms prove private→public
read with unchanged private write. A symbolic-string assertion wrongly rejects
numeric noun display before receiver installation. Publisher/source/clean-stop and
retained-tree audits pass; original overall failure remains. Fresh12 uses a typed
boolean comparison. This confirms M247's new-desk privacy premise only, not its
unobserved10 permissions or receiver/UI gates. Evidence:
diagnostics/2026-10-24-tend-publisher11-permissions.md.

## M249 — DM busy=false does not resolve a lost acknowledgement (2026-10-24)

One actual-bridge synthetic poke receives no Gall receipt. DMs correctly reports
unknown and does not resend, but clears busy/pendingSend while retaining text.
Helpers/streams stay unchanged during observation and teardown is clean. Source,
inactive-scope and retained-copy audits pass. This refines M246's pending-write
gate: collection activation needs explicit durable unresolved-operation handoff,
not a busy-only predicate. Fixture remains unapplied; no real ship/UI rollout.
Evidence: diagnostics/2026-10-24-uncertain-plugin-lifecycle.md.

## M250 — Tend direct backend install, readiness and receiver restart pass (2026-10-24)

Fresh12 preserves25 frozen files/five generated prerequisites, explicitly publishes
only the new test desk, and installs Tend through each ship's own Eyre. Separate
acknowledgements/readiness, exact receiver publisher, protocol2 and same-state/
original-session restart pass. Seven transport attempts/three clean stops, four
DB hashes, exact source and copied-chain audits verify. This supersedes M247/M248's
pending receiver gate, preserving their failures. Standalone plugin button/durable
intent, Grubbery and Docker devnet remain. Evidence:
diagnostics/2026-10-24-tend-backend-runtime.md.

## M251 — Standalone Tend install transport/card pass; outer13 counting fixture fails (2026-10-24)

Separate Tend worktree preserves canonical IDs/publisher and standalone direct
Eyre. Backend-free plugin login, authoritative desk/source inventory, locked durable
install intent, no uncertain replay, existing-source protection and separate protocol2
readiness are implemented.13 added tests/full138 Python+JS/QML checks pass before
card extraction; real extracted card/service passes private synthetic-Eyre UI.
Commons.Color needs explicit aliasing against QtQuick.Color. Native13 actual-client
install/session/restart subgate passes with three clean stops, but old outer fixture
counts only one direct install instead of its expected two; original failure retained.
Fresh14 will qualify unified receipts/trace. Extends M250, superseding its wholly
unimplemented plugin-flow status only; release/full desktop remain. Evidence:
diagnostics/2026-10-24-tend-client13-ui.md.

## M252 — Standalone Tend client install/restart and private release checks pass (2026-10-24)

Fresh14 actual client inventories absence, emits exactly poke→ack0→delete,
persists acknowledgement, does not replay on a second call, and returns protocol2
ready with the original session after restart. Source/pier/chain audits verify;
three clean stops. Final source passes138 Python+JS/QML/manifest and deterministic
release/temporary-target install checks in isolation. Separate actual setup-card/
service offscreen test passes. Supersedes M251's pending overall/release checks,
retaining its original failed trials. Canonical source/installed plugins and ships
remain untouched. Full desktop/public publisher/Grubbery remain distinct. Next:
durable unresolved application writes and graceful profile transitions. Evidence:
diagnostics/2026-10-24-tend-client14-runtime.md.

## M253 — DMs durable unknown-send recovery passes profile/helper roundtrip (2026-10-24)

Instance-bound private send intent is fsynced before PUT and retained without
replay after lost/unknown receipts. Exact same-origin history can resolve it;
absence/different source cannot.86 Python+JS/inert QML checks pass. Actual private
keyring/helper test preserves one unknown send and unchanged journal through
stable→dev→stable; no second poke, exact old helpers/streams exit, final shell0.
Supersedes M249's in-memory-only send limitation; M246 channel cleanup plus
invitations/read markers, real Gall reconciliation and a full coordinator remain.
No installed resources changed. Evidence:
diagnostics/2026-10-24-dms-outbound-recovery.md.

## M254 — DMs explicit transport drain deletes exact channels before replacement (2026-10-24)

92 Python tests and real private QML/helper/keyring roundtrip pass. Opt-in drain
pauses admission, waits existing work, deletes exact channel and joins listeners;
failures retain cleanup binding or report blocked. Three DMs channels deleted,
unknown send retained stable→dev→stable without replay, session preserved.
collectionReady is explicitly false. Supersedes M246 only for this DMs transport
path; Theme channels, other mutable effects and full coordinator remain. Evidence:
diagnostics/2026-10-24-dms-transport-drain.md.

## M255 — Theme drain closes both plugins' channels before replacement (2026-10-24)

199 Theme Python tests, model/manifest/native QML/SDK/hook and separate profile IPC
checks pass after preserving the helper status envelope and separating QML fixture
entry points. Actual private Theme+DMs helpers delete all six channels before
stable/dev/stable replacement; Theme saved state and DMs unknown send remain,
with no resend. Failed test logs retained. Supersedes M246's undeleted channels
only for the opt-in drain path and extends M254; collectionReady=false, all mutable
effects/appearance/Omabit/coordinator remain. See diagnostics/2026-10-24-theme-transport-drain.md.

## M256 — Applied three-profile drains and Omabit helper-exit gate (2026-10-24)

The applied --connected --drain fixture passes stable/dev/stable/edge/stable with
ten channels deleted, one unknown send retained and no replay; original zero-write
baseline remains passing. Extends M255 draft coverage to repeatable source/edge.
Omabit's internal transport stop pauses admission, waits explicit inventory exit
and other owned reads, and prevents late callbacks from restarting helpers. Actual
service with inert tools/monitor stub passes foreground refusal, delayed exit,
late-read and five-profile tests. Bootstrap behavior and real collector/monitor
transport are not yet covered. collectionReady=false throughout. Evidence:
diagnostics/2026-10-24-collection-transport-drains.md.

## M257 — Invitation and read-marker uncertainty survives profile replacement (2026-10-24)

DMs persists invitation/read intent before PUT, resolves only exact receipts and
keeps unreads unchanged on uncertain/rejected read markers. Stale-connection
receipts cannot change new UI.101 Python+6 profile QML tests pass. Actual private
helpers retain message+invitation+read unknowns across all three profiles with
three total pokes/no replay and ten deleted channels. Initial store-error assertion
and synthetic-history-shape failures retained. Extends M253/M256 beyond sends;
observation reconciliation, launcher effects, appearance and complete activation
remain. Evidence: diagnostics/2026-10-24-dms-conversation-effects.md.

## M258 — Fresh authenticated state resolves conversation effects without replay (2026-10-24)

DMs distinguishes desired-state confirmation from original operation ACK. Active
chat membership and explicit unread=null main-stream state resolve only the exact
pre-read intent cohort/current origin/ship/generation; absent/malformed/stale data
cannot. Source semantics checked against Tlon. Journalv1→v2 preserves identities;
future formats refused.112 tests pass. Actual private helper refresh and another
restart resolve two effects while preserving the unknown send, with three total
pokes, six drains and twelve deleted channels. Supersedes M257's pending effect
reconciliation, not real Gall/compositor/full activation gates. Evidence:
diagnostics/2026-10-24-dms-effect-reconciliation.md.

## M259 — Cooperative exclusive appearance and generation-bound hooks (2026-10-24)

Theme uses one HOME-level helper lock, persistent active selection IDs and applied
owner-guarded Lua. Actual Lua stable/dev/edge/stable executes only one override;
actual helpers refuse overlap and permit takeover after exit. Hooks queue intent
without the separate UI publication bypass; delayed old selections cannot publish.
Generated files have atomic content receipts; unknown/edited/unreceipted files
are preserved/refused.213 Python tests, native/profile QML and connected6-drain/
12-channel lifecycle pass. Supersedes M258's pending appearance ownership and the
old unconditional override repair only for this development contract. Legacy
migration, partial-apply rollback, actual compositor and full activation remain.
Evidence: diagnostics/2026-10-24-theme-appearance-owner.md.

## M260 — Legacy appearance adoption requires exact retained application evidence (2026-10-24)

Theme startup reconstructs adapter2 bytes from retained observed/applying profiles,
backs up an exact match and atomically writes owner-guarded adapter3. No old code is
evaluated and migration issues no desktop command/ownership claim. Edited, unknown,
linked/special and mismatched-backup files refuse intact. Interrupted replacement
retries with the original backup.54 focused/218 total Python checks pass, including
actual connected helper migration/restart without stale publication. Connected
Theme/DMs6-drain/12-channel regression also passes. Supersedes M259's blanket legacy
refusal only for these exact evidence-backed files; rollback/compositor/power-loss
and full activation remain. Evidence: diagnostics/2026-10-24-theme-legacy-adoption.md.

## M261 — Real-service partial appearance failures expose and fix rollback stall (2026-10-24)

A failed native working theme left a temporary name that blocked the returning
profile before capture/reconcile. Recovery now requires the HOME lock, intact
owned generated-directory receipt and an existing authenticated hub. It preserves
uncertain intent, persists applying before effects and verifies readback; empty/
offline/unknown/edited cases refuse. Errors no longer report Following. Actual
private Service/helpers pass four post-effect failures, exact drains/previous-owner
restoration and failed-restore retry:12 transitions,28 deleted channels,4 Talon
pokes and0 appearance pokes.222 Python/58 focused checks and original zero-write
mode pass. Supersedes M260's untested online partial-restoration gap, not offline
snapshot, native compositor/fanout or automatic collection rollback. Evidence:
diagnostics/2026-10-24-theme-partial-rollback.md.

## M262 — Offline appearance restore survives explicit failure and helper kill (2026-10-24)

Theme has private version1 content-addressed checkpoints, frozen local assets and
request-bound applying/verified receipts. Native read locking avoids mixed theme
generations. Unknown/edited state refuses; sync/publication journals are preserved.
237 Python/73 focused checks pass. Actual private helpers restore after the test
hub/keyring stop and source assets change; failed effects, blocked sync startup,
exact retries, effect-free completed replay and an exact helper SIGKILL pass.
Online four-stage rollback still passes. Supersedes M261's missing offline recovery
prerequisite only. Wallpaper, actual native fanout/compositor/shell restart,
cross-profile transition fencing and power loss remain separate. Native source
shows font changes restart the owning shell; the coordinator must survive that
boundary outside plugin QML. Evidence: diagnostics/2026-10-24-theme-offline-checkpoint.md.

## M263 — Copied native headless theme staging preserves frozen rollback files (2026-10-24)

The actual Omarchy setter/template/palette scripts and built-in templates are
copied, hashed and run only in the private headless fixture. Stable checkpoint,
partial candidate, offline failed restore/retry, read-only success replay and
exact helper kill/retry pass with every restored file matching the snapshot.
Both scopes exit cleanly. Applied --native-theme-staging is explicit and requires
the offline appearance mode. Supersedes M262's simulated theme-file staging only;
compositor, background, app-retint, font-triggered shell restart and durable
collection activation remain. Native template generation preserves existing frozen
files on this source snapshot. Evidence: diagnostics/2026-10-24-native-theme-staging.md.

## M264 — Native font restart kills shell-owned work; exit0 hides retint failures (2026-10-24)

Copied native scripts with real private Quickshell confirm font-set restarts the
owning shell. Its child helper loses its result; the same Theme command runner
outside the shell survives. Native theme-set returns with background cache alive,
while Theme's runner kills the command group after return. An early app-retint
failure still yields native exit0. The initial overstrict cache-start test failed
and is preserved; corrected draft/applied tests pass and all scopes are inactive.
Extends M263's headless-only native lifecycle evidence; no real plugin services or
compositor in this narrow test. Actual offline Theme restore through native shell
restart is next; full collection activation remains. Evidence:
diagnostics/2026-10-24-native-appearance-effects.md.

## M265 — Actual offline Theme restore survives native font/shell replacement (2026-10-24)

The unchanged restore CLI runs independently while copied native font-set/restart
scripts replace a real private Quickshell. An injected failure after the native
font effect returns Needs attention and retains applying; a fresh exact retry
verifies, and success replay adds no effects. Sync pending/attempted/applying state
stays byte-identical. Draft/applied tests pass with exits1,0,0; all scopes stop.
Supersedes M264's missing actual CLI integration, not full plugin-service collection,
font rendering, app-retint/compositor or durable activation qualification. Local
verified state is seeded; no ship/keyring is contacted. Remaining Omabit monitor,
launcher and auth-helper quiescence is next. Evidence:
diagnostics/2026-10-24-native-font-restore.md.

## M266 — Omabit waits for actual monitor/setup reader exits (2026-10-24)

Shared registration covers runtime logs, project history and SSH suggestions even
outside Service's tree. Exit signals clear pending state; losing a live owner
leaves reader-owner-lost rather than claiming closure. Actual private QML catches
and fixes late SSH suggestions accepted after drain. Draft/applied five-profile
cycles pass delayed exits, admission refusal, stale results, failed start and
removed-owner uncertainty; baseline/project UI regressions pass. All scopes stop.
Supersedes M256's monitor-stub-only coverage and narrows M265's monitor gap; real
collector/auth/clipboard descendants and durable activation remain unqualified.
collectionReady stays false. Evidence: diagnostics/2026-10-24-omabit-reader-drain.md.

## M267 — Signal-killed helper owners cannot certify descendant cleanup (2026-10-24)

Actual receiver adapter normal/cancel paths reap its group, but SIGKILL of its
qualification owner leaves receiver and child alive. Private exact cleanup is
verified. Omabit now retains helper-descendants-unconfirmed for signaled service,
auth and auxiliary helpers. Real-QML inventory/log/auth crash cases pass, including
an orphan after the shell's child list empties; active auth refuses drain. Normal
five-profile reader and baseline paths still pass. Source/guards/scopes verified.
Supersedes M266's possible clean receipt after abnormal helper exit, not production
orphan cleanup, real collector/SSH/clipboard, combined plugins or durable activation.
The receiver proof uses the actual adapter via a small owner, not the full CLI.
Evidence: diagnostics/2026-10-24-helper-exit-fence.md.

## M268 — Omabit/Theme/DMs pass combined native-loader profile transitions (2026-10-24)

Actual QML services/overlay share copied native loader functions with private
profile IDs/state, real Theme/DMs helpers/keyring, synthetic Eyre and inert Omabit
tools. Six drains each delete12 channels; unknown send never replays, two effects
confirm by authenticated state and survive reload, with three total pokes. Device
IDs persist. Immediate drain and asynchronous panel assumptions caused retained
fixture failures; corrected busy/readiness waits and applied runs pass, along with
the zero-write baseline. All scopes stop. Supersedes M256/M267's separate-service
limit, not actual runner/compositor/appearance or persistent activation. Keep M267
uncertainty across future coordinator restarts. Evidence:
diagnostics/2026-10-24-supported-collection.md.

## M269 — Collection journal survives restart and preserves uncertainty (2026-10-24)

New internal format1 intent journal records exact supported specifications,
selected versus loaded identities, deterministic phase/effect IDs, per-member
drain receipts, appearance checkpoint and last verified collection. Private
24-test repeat includes real coordinator SIGKILL/resume, every transition boundary,
partial rollback, stale callbacks and uncertain writes. Four original-writer
fixtures and source hashes are retained; test scope is inactive. Missing/unknown
state refuses intact; helper uncertainty cannot be reset through reload. Supersedes
M268's missing persistent intent only at the model layer: adapter observations
are not independently verified and no runtime effects or switch command exist.
Existing bootstrap/runner state is unchanged. Evidence:
diagnostics/2026-10-24-collection-journal.md.

## M270 — Journal coordinates real private plugin effects and lost-reply recovery (2026-10-24)

Internal one-step coordinator and qualification adapter pass stable/dev/stable,
partial edge appearance failure and rollback. Six actual coordinator SIGKILLs
after effects/before receipts resume; enable reuses exact instances and verified
restore replay adds no effects. All16 channels close. Thirty private model/driver
checks and the original zero-write lifecycle baseline pass. Loader completion
precedes property assignment; private identity capture now waits and freezes once.
Initial failed trial retained, source/journal/scopes independently verified.
Supersedes M269's model-only limit for these isolated paths, not full shell-loss,
production loaded-identity API, native compositor or actual Omabit tool effects.
The adapter is private instrumentation, no general switch command. Teardown leaves
an explicit pending checkpoint after final drain. Evidence:
diagnostics/2026-10-24-collection-coordinator.md.

## M271 — Shell replacement resumes, unclean owner loss retains a durable fence (2026-10-24)

Two controlled private Quickshell replacements between effects/receipts resume
activation and rollback. Six coordinator kills still pass. A final unexpected
shell SIGKILL leaves two recorded helpers and an undeleted DMs channel; the
adapter journals reader-owner-lost. A new empty shell and exact fixture cleanup
cannot erase uncertainty or activate the candidate. Final scope is inactive;
source hashes and journal replay verified. Supersedes M270's missing private
shell replacement evidence, not production cleanup/native-shell/compositor support.
Recovery needs plugin-owned durable channel/intent evidence plus independently
verified cleanup; never infer it from reload or empty process lists. Preserve
standalone clients and direct Eyre. Evidence:
diagnostics/2026-10-24-collection-shell-recovery.md.

## M272 — DMs EOF now drains its channel without clearing collection uncertainty (2026-10-24)

Bridge main finally drains on EOF/broken output, skips duplicate successful close,
and exits nonzero when cleanup remains unknown. All118 private DMs checks pass.
Actual private shell loss now deletes all16 channels; controlled replacements,
six coordinator kills and rollback remain passing. reader-owner-lost still persists
across new-shell creation and exact fixture cleanup. Surviving helpers in M271/M272
map to inert Omabit inventory/log readers, not actual collector processes.
Supersedes M271's undeleted DMs channel for EOF-driven parent loss only. Helper
SIGKILL/reconnect loss still needs durable plugin-owned transport evidence and
explicit verified recovery. No installed plugin changes. Evidence:
diagnostics/2026-10-24-dms-eof-drain.md.

## M273 — DMs owns durable channels and recovers killed helpers without app replay (2026-10-24)

Format1 channel journal, lifetime lease and PID/start/boot/namespace evidence
persist exact creator/session/channel and stable cleanup requests before HTTP.
147 private tests pass. Actual helpers with a private keyring/synthetic Eyre recover
two SIGKILLs, an accepted delete with a lost reply and later stream reconnect;
three channels close, uncertain app journals stay byte-identical, three original
pokes never replay, and a concurrent helper refuses. Six original-writer fixtures
are retained. Final-source QML collection regression also passes all16 deletions,
rollback and controlled shell replacements while preserving reader-owner-lost.
Supersedes M272's missing helper-crash/reconnect evidence only for this isolated
synthetic-peer scenario. Actual Urbit, expired/missing credentials, supported archive
handling (1024 receipts), old-helper adoption/downgrade and coordinator recovery
remain gates. This is cooperative same-user crash coordination, not wallet secrecy.
Installed plugins/ships remain unchanged. Evidence:
diagnostics/2026-10-24-dms-durable-transport.md.

## M274 — Real Eyre recovery passes after explicitly cancelling owned stream reads (2026-10-24)

Actual fresh-fake trial02 recovered both helper kills and a lost204 reply but
failed drain with stream-still-open: remote deletion did not wake the local read.
A duplicated exact stream-socket descriptor now permits shutdown before join;
late opens and stale reader exits cannot affect replacements. 150 private tests,
actual fresh-fake trial03 (eight subscription ACKs, two channel404s, same cleanup
request/session, private logout, clean SIGTERM) and full16-channel collection
regression pass. No application pokes occur in the real-Urbit fixture; M273 remains
the uncertain-app no-replay evidence. Both failed trials and four actual writer
snapshots retained; scopes/source hashes independently checked. Supersedes M273's
synthetic-only transport limit for this fake, not hosted/native/coordinator recovery
or expired credentials/archive/adoption gates. Existing ships/plugins untouched.
Evidence: diagnostics/2026-10-24-dms-urbit-transport.md.

## M275 — Theme transport ownership is a remaining recovery prerequisite (2026-10-24)

Source inspection after M274 finds Theme Eyre.pending_watch_channels is an
in-memory set. Watch reconnect retains failed deletes only within the live Eyre
object; poke channels attempt deletion in finally without durable channel evidence.
SIGKILL can therefore lose both kinds of cleanup identity. Watch, engine writes,
bridge destinations and standalone CLI can use multiple Eyre instances, unlike
DMs' single helper-owned channel. Do not copy its exclusive single-channel lease
in a way that breaks legitimate Theme operations. Establish Theme-owned durable
channel/mutation evidence and exact dead-owner cleanup before adding coordinator
recovery authority. Keep direct Eyre/standalone operation and separate sessions;
Omabit must not become the application data proxy. This is a source finding, not
a new executed Theme failure or supersession of M274's DMs pass. No Theme source
or installed state changed. Source fingerprints are retained in
diagnostics/2026-10-24-theme-recovery-prerequisite.json.

## M276 — Theme concurrent transport and uncertain settings survive client loss (2026-10-24)

Theme format1 uses per-operation leases and short journal transactions, permitting
live watch/publisher concurrency. Exact creator/session/channel/cleanup intent
survives death; only dead/released owners can be recovered. Unknown settings writes
remain separate from deletion and block re-publication until exact authenticated
readback (state-confirmed, not ACK) or a known Gall receipt. 268 private Python
checks, actual client publisher/cleaner SIGKILL with a surviving watch and one
unreplayed poke, and the full16-channel collection regression pass. Six original
writer fixtures retained; sources/scopes independently verified. Declaration and
installer include the new module and recognize the previous checkpoint inventory.
Supersedes M275's in-memory-only finding for these paths. Actual Urbit Theme, safe
disconnect/credential preservation with live clients, expiry, archive/adoption and
independent coordinator recovery remain. Limits2048 channels/4MiB/8192 owner files
refuse intact. No installed changes; DMs M274 unchanged. Evidence:
diagnostics/2026-10-24-theme-durable-transport.md.

## M277 — Theme disconnect preserves credentials until transport quiesces (2026-10-24)

Format2 persists retired session fingerprints and admits reads/channels under
shared origin locks. Disconnect pauses the row, fences new work and waits for
exclusive access plus verified channel owners/deletes before local removal. Failed
cleanup keeps the account/credential for retry; receiver cancellation uses the same
barrier. 284 private tests pass, including the actual M276 writer/fixture migration
and downgrade refusal. Actual CLI + private keyring/synthetic peer survives two
kills and failed/lost deletion replies, blocks new HTTP work, closes two channels
before logout/clear and never replays one uncertain settings write. Five snapshots
retained; full16-channel QML regression preserves the coordinator uncertainty fence.
Supersedes M276's unsafe-disconnect finding for cooperating current clients only.
Actual Urbit, remote expiry, archive/pre-journal adoption/native identity and
independent coordinator recovery remain. Existing installed state untouched.
Evidence: diagnostics/2026-10-24-theme-safe-disconnect.md.

## M278 — Theme actual-Eyre recovery and credential-preserving disconnect pass (2026-10-24)

A fresh pinned fake/private keyring validates actual Theme login, two positive
settings subscription receipts, two exact client kills, lost successful deletion
reply and same-session/request recovery. Actual Disconnect waits for the client;
both channels independently return404 before logout/clear. No app writes occur.
The disposable ship exits0 on SIGTERM, scope inactive; four original writer
snapshots/source/runtime hashes verified, secrets/piers/raw Vere logs excluded.
Supersedes M276/M277's synthetic-only transport limitation for this fake only;
M277 still supplies failed-cleanup and uncertain-write no-replay evidence.
Next independently bind old instance transport and application intent into
collection recovery. Hosted/remote expiry, archive/adoption/native identity remain.
Evidence: diagnostics/2026-10-24-theme-urbit-transport.md.

## M279 — A surviving external supervisor can prove its owned descendants exited (2026-10-24)

Known-helper snapshots cannot close reader-owner-lost after shell death. A private
subreaper prototype now passes double-fork/setsid, late fork during TERM, controller
EOF and guardian-loss cases. PIDfd-bound owned-child signaling plus waitpid ECHILD
produces closure only for the same still-authoritative guardian. Guardian SIGKILL
leaves its record unconfirmed; a new empty guardian cannot replace that proof.
The initial test's all-descendants-direct-parent assumption was false and is
superseded by the corrected orphan-root assertion; both trials retained/inactive.
This does not yet supervise actual QML or resolve the collection journal fence.
Bind exact activation/guardian and plugin-owned state next; preserve uncertainty
if the supervisor itself is lost. Existing installed state remains untouched.
Evidence: diagnostics/2026-10-24-collection-supervisor.md.

## M280 — Actual private QML collection has independently bound process closure (2026-10-24)

Optional supervised-shell qualification passes stable/dev/stable, edge rollback,
six coordinator kills and four shell guardians. Loaded activation records bind the
original guardian nonce/PID/start/boot/namespace. After shell SIGKILL that guardian
reaps both surviving inert Omabit readers; a replacement guardian is distinct.
All16 Theme/DMs channels close; final-source four-case supervisor regression passes.
Sources/receipts verified, both scopes inactive. Supersedes M279's missing actual
QML supervision only. Format1 still retains reader-owner-lost; no recovery authority
has been added. Next bind exact old plugin transport/application state and a durable
explicit recovery transaction. Native loader/external-service accounting and lost
guardian recovery remain separate. Production ships/plugins/credentials untouched.
Evidence: diagnostics/2026-10-24-supervised-collection.md.

## M281 — Unresolved application writes survive supervised collection loss (2026-10-24)

The strengthened actual-QML fixture leaves one DMs send and one Theme setting
accepted without acknowledgement before shell SIGKILL. Their retained intent IDs,
DMs journal bytes and Theme value/session digests survive original-guardian cleanup
unchanged; neither write replays. All17 channels close, four guardians certify
closure and six coordinator crash cases pass. Sources/receipts verify; failed
trial01 and passing02 retained, scopes inactive. M280 lacked unresolved writes;
this supplies that missing scope, not recovery authority. Format1 retains its
reader-owner-lost fence. Next explicit versioned recovery must bind this old
activation, original guardian and unchanged app/credential cohort. New empty UI,
new guardian or successful channel deletion cannot erase uncertain app work.
Evidence: diagnostics/2026-10-24-collection-recovery-cohort.md.

## M282 — Collection recovery has a versioned prepared/verified model (2026-10-24)

Format2 binds explicit recovery to the old activation, original supervisor/root,
closure request, exact fences and unchanged member app/pending/credential/channel
cohorts. Preparation persists before cleanup; retries re-sync it before effects.
Verified closure preserves old problems and advances only to disable-previous;
it never acknowledges app writes. 48 private checks pass, including two actual
model coordinator kills, original M281 writer reproduction/migration and downgrade
refusal. Failed discovery/stale-test-store trials retained. Closed Store now refuses
before relative open. Supersedes M280/M281's missing recovery model only; actual
plugin/runtime adapter and competing-admission proof remain unqualified. Only old
drain-previous is supported; no guardian replacement or changed-cohort reset.
Evidence: diagnostics/2026-10-24-collection-recovery-model.md.

## M283 — Exact private plugin recovery survives two coordinator losses (2026-10-24)

Actual QML/helper journals retain an unknown DM and Theme setting while old-channel
deletes fail. Original-guardian closure plus four exclusive plugin admission locks
bind preparation; replacement guardian is refused. Kill after prepare, failed
delete, cleanup then kill before receipt, and no-HTTP retry/replay all pass. Two
pending app intents and credentials remain unchanged; all17 channels close. Six
ordinary crash/rollback cases and four guardian closures remain verified. Scope
inactive; hashes checked. Supersedes M282's model-only limit for this private
synthetic-Eyre cohort. Final phase is checkpoint-appearance; candidate activation
and return after recovery are next. Native durable admission, external services,
conversation effects/multi-account, lost guardian and production rollout remain.
Evidence: diagnostics/2026-10-24-explicit-collection-recovery.md.

## M284 — Passive Theme recovery re-publishes and fixture success was false (2026-10-24)

Post-recovery return-to-stable trial02 reports pass but independent audit rejects it:
a worker-thread assertion caught an identical Theme poke after authenticated
readback state-confirmed the old intent. The actual journal contains a new uncertain
intent with the same digest. publish intentionally re-notifies for Publish Now;
Helper's automatic pending retry improperly shares that behavior. Preserve explicit
renotification but deduplicate passive matching state. Collect HTTP worker failures
as fatal fixture evidence. Trial01 exposed private keyring auto-activation racing
manual unlock; wait for exact launched daemon bus ownership. Both failed trials
retained/inactive. M283 remains valid only through checkpoint-appearance; return
is unqualified. Evidence: diagnostics/2026-10-24-recovery-resume-finding.md.

## M285 — Passive Theme repair and recovered profile roundtrip pass (2026-10-24)

Passive publication now skips authenticated matching entries; explicit Publish Now
still re-notifies. Unknown mismatching writes remain fenced. 287 private Theme tests
pass; historical migration explicitly tests unknown ownership refusal and a modeled
later boot rather than relying on namespace reuse. Original fixture bytes unchanged.
The corrected fatal-HTTP-error/keyring-ownership harness rejects frozen old Theme
and passes with only main.py/eyre.py changed. Recovered candidate activation and
stable return restore10 positive subscriptions with original sessions; unknown DM
bytes remain visible/unchanged and the original Theme intent becomes state-confirmed
without replay. All21 channels close, eight coordinator kills and four guardians
verify, scopes inactive. Supersedes M284's passive replay/false pass and M283's
missing return evidence only. Native loader/admission, external services, broader
cohorts and production rollout remain. Evidence: diagnostics/2026-10-24-recovery-resume.md.

## M286 — Immutable artifacts bind native loaded identity (2026-10-24)

Explicit pinned preparation and exact loaded-URL observation are implemented for
supported non-default collection runtimes; no manifest is installed/enabled.62
private tests pass including process death before publication and exact retry.
Complete copied native registry and native enable IPC plus service/panel loaders
prove ten generations, selected2/loaded1 separation, explicit replacement and five
profile roundtrips with15 distinct instances. Plugin bodies in this new fixture are
synthetic; M285 remains the actual application recovery evidence. All eight scopes
inactive, final hashes verified. Theme/DMs use ambient XDG plus profile namespaces;
only Omabit accepts per-instance roots. Same-user hashes/modes are not wallet secrecy.
Supersedes M285's missing native artifact identity prerequisite only. Durable
admission, real wrapper/widget integration, scoped Omabit monitor/setup IPC and
external effects remain. An external coordinator can use stock shell enable IPC;
no collection-wide ordinary-plugin API is required. Evidence:
diagnostics/2026-10-24-native-artifacts.md.

## M287 — Actual Omabit monitor/widget/setup IPC follows profile ID (2026-10-24)

Service monitor and Panel widget/setup targets now derive from injected identity;
status reports module/profile. Actual native-control Panel + Service and readers
pass five private profile cycles, scoped open/close, canonical target absence after
initial injection, unload disappearance and all prior helper/reader-loss fences.
An ambiguous unqualified Color lookup is fixed with Commons.Color. Old canonical
source fails the new test. All six scopes inactive and final hashes verified.
Quickshell unknown IPC target returns exit0 with `Target not found.`: validate the
response and target inventory, not exit status alone. Failed exit-code/color/fixture
trials retained. Supersedes M286's steady-state Omabit IPC gap only; initial native
creation-before-injection/coexistence, gated payloads and durable admission remain.
Evidence: diagnostics/2026-10-24-scoped-omabit-ipc.md.

## M288 — One-use native admission fences replacement shells (2026-10-24)

Explicit format1 sealed permits bind exact shell and three immutable members.
One-use guard claims survive guard death; releasing a lock never grants replacement.
Inert QML hosts create no application code without a grant and drain/fence on loss.
78 private tests and five copied-native-loader cases pass: zero unadmitted apps,
three admitted apps, killed-guard replacement refusal, unused stale permit refusal
in a second shell and three deliberately admitted new apps. Six guards exit; four
scopes inactive and sources verified. Failed import/settings fixture trials retained.
Supersedes M286/M287's missing durable admission prerequisite only. Host/helper
artifact packaging, actual supported widgets/apps, cross-root coordinator ownership,
constructor/drain accounting, recovery integration and production rollout remain.
Closing claims is not transport closure; same-user controls are not wallet secrecy.
Evidence: diagnostics/2026-10-24-native-admission.md.

## M289 — Immutable admission hosts and widget gates are packaged (2026-10-24)

Format2 pins application and admission inventories, packages guard/host/registry
inside each immutable generation and redirects primary/widget entries to inert
loaders. Widgets wait for their exact admitted primary and receive fixed ID/service
before construction. Original M286 bytes reproduce; old-reader refusal and unchanged
format1 readability pass. Legacy reads cannot authorize guarded observations.
All88 private tests and five packaged native admission cases pass; sources verified,
three scopes inactive. Trial01 global widget counting failed because the fixture
recreated other still-admitted surfaces; corrected per-Theme checks pass.
Supersedes M288's packaging gap only. Actual bodies, native bar API/IPC, cross-root
coordinator/recovery and constructor/drain accounting remain unqualified.
Evidence: diagnostics/2026-10-24-guarded-artifacts.md.

## M290 — Actual Omabit Service/Panel work through guarded native hosts (2026-10-24)

Copied actual Service/Panel plus native Ui controls pass all five admission cases
with inert runner/SSH/focus tools. Stable/dev constructors see independent IDs;
the widget receives the actual service payload. Scoped monitor/setup/widget IPC,
setup-open/close and actual pre-unload transport drain pass. Six logged fixture
helpers end with exit records; admission owners exit, source hashes match and scope
is inactive. Extends M289's synthetic Omabit and M287's directly injected component
limits. Theme/DMs still synthetic; native bar slots/compositor, canonical coexistence
transients, connected accounts, coordinator recovery and broader effects remain.
Evidence: diagnostics/2026-10-24-guarded-omabit.md.

## M291 — Actual supported set and widgets pass guarded native admission (2026-10-24)

All actual Omabit/Theme/DMs primary/widget bodies pass five admission cases and
stable/dev startup with empty private profiles. Six primary constructions, exact
artifact identity, service/DMs singleton access, scoped Omabit IPC and actual
transport drain pass; six recorded guards exit. Theme/DMs Python remains unchanged.
Portable harness uses explicit source paths and copied native controls. Four scopes
inactive, source hashes verified. Trial01 had fixture newline escaping errors;
trial02 incorrectly disabled Theme's transport owner and correctly failed closure.
Keeping the real helper with inert desktop commands passes trials03/04.
Supersedes M290's synthetic pair gap only. M285 connected recovery still needs
native admission integration; closing-time UI/IPC, real bar/compositor, cross-root
coordinator recovery, broader effects and descendant closure remain.
Evidence: diagnostics/2026-10-24-guarded-supported-set.md.

## M292 — Widget/monitor admission closes before busy transport drain (2026-10-24)

Negative control confirms old Theme widgets retain open IPC after guard loss.
Live manageIpc binding now withdraws Theme/Omabit widget/setup handlers. Host closes
and fences overlay opening; settings stop forwarding while non-admitted. Omabit's
separate uiAdmissionOpen closes/refuses monitor/setup while an existing operation
finishes, without conflating UI closure with transportStopping. Actual supported-set
native tests pass held-operation close/release/drain and DMs overlay refusal;
all88 private tests pass, sources verified and four scopes inactive. Failed fixture
newline trial and genuine prior-code negative retained. Supersedes M291's closing
UI gap only; connected native recovery, full bar/compositor and external effects remain.
Evidence: diagnostics/2026-10-24-closing-ui-admission.md.

## M293 — Connected native profiles preserve sessions and unresolved DM (2026-10-24)

Actual supported bodies/widgets in immutable guarded generations pass stable/dev/
stable: nine admitted primaries, fifteen positive subscriptions, ten deleted channels,
unchanged six saved sessions and zero new logins. The real helper's single uncertain
DM remains byte-identical without replay. Failed trials retained: missing private
appearance opt-in and incorrect cross-plugin uncertain-write counting. Explicit
starting/failed markers and readiness snapshots prevent false positive reports.
Four scopes inactive; sources verified. Supersedes M291/M292's connected-account
limit for clean native switch/return only. Native coordinator/original-guardian
recovery, full bar/compositor and production commands remain unqualified.
Evidence: diagnostics/2026-10-24-native-connected.md.

## M294 — Native guarded profiles recover through original guardian (2026-10-24)

Actual native hosts remain inert under stale enabled startup after shell SIGKILL.
The durable coordinator refuses progress until explicit original-guardian recovery.
Wrong guardian and503 cleanup refusal preserve the fence; kills after prepare and
after cleanup safely retry without duplicate HTTP. Killing after candidate native
activation also retries without duplicate construction. Dev then stable commit:
15 subscriptions,10 deleted channels,9 primary constructions, unchanged six sessions
and byte-identical unresolved DM. Two original guardian ECHILD receipts and nine
immutable generations independently verified; three scopes inactive. Failed partial
loader observation retained and corrected. Supersedes M293's private native
coordinator/recovery integration gap only. Cross-root ownership, full bar/plugin
API/compositor, broader effects and public profile commands remain.
Evidence: diagnostics/2026-10-24-native-recovery.md.

## M295 — Guarded widgets translate native bar ownership (2026-10-24)

Negative control shows actual Ui.Panel navigation cannot identify its outer guarded
slot. WidgetHost now translates its own payload/host popout and navigation identity,
mirrors native presentation and unwraps matching peer widgets. Closing admission
refuses new navigation/popouts/commands while allowing cleanup. Three actual widget
profile cycles pass copied native bar methods/PluginBarApi; ten closed-payload
probes and one inert host pass. All88 tests and connected native recovery regression
pass; nine retained M294 generations still verify unchanged. Six scopes inactive.
Failed empty-payload assumption retained. Narrows M294's bar-ownership gap only;
full bar/compositor, native shell facades, cross-root ownership and public profile
commands remain. This forwarding contract is not a same-user security boundary.
Evidence: diagnostics/2026-10-24-native-bar-ownership.md.

## M296 — Actual supported payloads use native scoped shell APIs (2026-10-24)

Copied unchanged native shell/bar-state APIs and27 exact factory/routing functions
pass nine guarded payload activations and three DMs summon/open/dismiss cycles.
Own service lookup, foreign-ID refusal, generic-config refusal and public manifest
trust-marker removal pass; bar ownership/closing and stale-admission checks remain
passing. Scope inactive, sources verified. Narrows M295's facade-injection gap.
Full shell/bar/compositor, API cache revocation/pruning, cross-root ownership and
public profile commands remain unqualified; this is not a QML security sandbox.
Evidence: diagnostics/2026-10-24-native-shell-api.md.

## M297 — Real native bar surfaces reveal and fix two UI lifecycle defects (2026-10-24)

Private signed Sway/wlroots pixman runtime, no host display/GPU/ships. Complete
native bar/registry/KeyboardPanel plus actual supported guarded apps pass stable/
dev/stable: nine primaries/widgets, six popup openings, three real DMs overlays,
Enter-control/Escape cycles and closure of mapped popups during guard loss/drain.
DMs Escape moves from non-Item PanelWindow to the card; WidgetHost now closes on
admitted changing false, avoiding phase-signal binding order. Controlled negatives,
failed harness trials and twelve screenshots retained.88 tests and native connected
recovery pass;15 subscriptions,10 deleted channels, unchanged sessions/unresolved
DM; nine older generations verify and18 scopes inactive. Narrows M292's closure
claim and supersedes M295/M296's simplified-bar/inert-popup limit only. Explicit
bar API pruning and stable selected layout are fixture actions; full shell/API
lifecycle, Hyprland/IME, connected compositor, cross-root ownership and public
profiles remain. Keyboard04/05 alone inconclusive; dedicated negative proves
Enter arrives while Escape fails. No host package/plugin installation.
Evidence: diagnostics/2026-10-24-native-wayland.md.

## M298 — Native shell APIs are destroyed across real-bar profile unloads (2026-10-24)

Exact native API sync/prune callbacks plus diagnostic lifetime markers pass three
real-Wayland stable/dev/stable unloads. Nine admitted API instances differ; all13
created APIs (including inert/recreated hosts) have destruction records. Scoped
caches/descriptors/bar-state and owned bar UI clear. Five scopes inactive; sources
verified. Earlier failures were assertions about configured empty-slot placeholders
and Qt's non-null wrapper for a destroyed QObject; final check requires no usable
shell API and no owned UI objects. Narrows M296/M297's supported-set cleanup gap.
Explicit bar pruning remains; full shell, captured-JS-callback revocation,
Hyprland/IME, cross-root ownership and public profiles are not qualified.
Evidence: diagnostics/2026-10-24-native-api-lifecycle.md.

## M299 — Connected state and subscriptions survive real native Wayland switching (2026-10-24)

One private real-bar/compositor/API-lifecycle run passes stable/dev/stable with15
subscriptions,10 deleted channels, six unchanged sessions and no login/replayed
DM. Unknown DM stays byte-identical; visible Theme palette and DMs empty-inbox
refresh are healthy. Nine primaries/widgets and13 native API lifetimes close.
Screenshots retained/checked. Fixture heads/palette read gaps and a mistaken blank
status assertion are corrected with prior trials retained. Native original-guardian
recovery regression still passes; four scopes inactive, source/receipt hashes verify.
Supersedes M297/M298's separate connected/compositor limit for clean transitions.
Compositor plus crash recovery, real groups message contents, full shell,
Hyprland/IME, captured callbacks, cross-root ownership and public profiles remain.
Evidence: diagnostics/2026-10-24-connected-native-wayland.md.

## M300 — Different coordinator roots bypass per-root admission exclusion (2026-10-24)

Private characterization: two synthetic shell parents each hold their own real
coordinator lease and admit three actual guards for the identical collection/spec.
All six coexist; same-root competing lease refuses, while closing one root leaves
the other admitted. All guards then end; scope inactive and sources verified. No
application code/credentials/ships started. This confirms M288/M294's documented
cross-root limit, not a safety pass or a new external blocker. One explicit shared
runtime authority, canonical journal binding and journal/guardian-backed handoff
must precede public profiles. Preserve old cleanup/readability and version new
admission rather than rewriting old seals. See collection-activation.md and
diagnostics/2026-10-24-admission-roots.md.

## M301 — Shared initial reservation and format2 guard admission (2026-10-24)

Qualified:124 private tests and a real private-compositor demonstration bind three
actual guards to one physical runtime/canonical journal; another shell/journal is
refused. Authority loss fences all guards while preserving local cleanup. Controller
SIGKILL, lost replies, replaced objects and strict old-reader refusal pass. Seven
scopes are inactive; sources and original-writer snapshots retained. Legacy
admission/journal/artifact sources remain byte-identical. This narrows M300 only
for the new initial-admission API; existing format1/native hosts retain its limit.
No handoff/release exists yet, and no app/ship was started. Next: versioned native
packaging and journal/original-guardian-backed handoff, followed by connected native
recovery. Persistent bootstrap/provisioning recovery remain separate gates. See
diagnostics/2026-10-24-shared-authority.md. Do not infer cleanup from ended guards,
free locks or a vanished compositor; do not silently recreate missing authority.

## M302 — Shared-authority format3 native generation (2026-10-25)

Qualified:133 private tests, six native loader/compositor cases and the existing
connected recovery regression. Format3 pins SharedHost plus the format2 guard;
missing authority/legacy permits load zero apps, valid admission loads three
primaries/widgets, a second shell refuses, and authority loss fences/drains all.
The exact frozen format2 writer and refactored writer reproduce original bytes;
old-reader refusal and nine M294 generation reads pass. Fifteen subscriptions,
ten closed channels, six unchanged sessions and the unresolved DM survive the
legacy recovery regression. Four scopes inactive; sources/guardian receipts verify.
Supersedes M301's missing native-packaging limit only. New native bodies remain
synthetic; shared handoff and connected recovery are not yet supported. Next:
journal/original-guardian-backed authority handoff, preserving uncertainty and
historical contracts. See diagnostics/2026-10-25-shared-native-artifacts.md.

## M303 — Shared handoff preserves connected state through crashes (2026-10-25)

Reservation2 pins the original guardian before construction and retains canonical
cleanup/history for handoff.158 tests,25 handoff cases, original-writer refusal,
actual connected native Wayland recovery and legacy regression pass. Five shared
controller kills preserve exact handoff, avoid duplicate enable/unload, restore15
subscriptions and close10 channels with six unchanged sessions and byte-identical
unknown DM. Two original guardians and canonical journal independently verify.
Actual QML unload killed guards before durable end; explicit post-drain EOF release
fixes clean handoff without weakening lost-owner recovery. Trial04 retains the
failure; trial06 passes. Eleven scopes inactive; no production resources touched.
Supersedes M301/M302's shared-handoff/connected-recovery gap only. Partial candidate
rollback has model/actual-process coverage with synthetic app receipts; native
partial rollback, initial adoption, cross-session bootstrap, full shell/Hyprland
and public profiles remain. See diagnostics/2026-10-25-authority-handoff.md.

## M304 — Native partial startup rolls back with measured cleanup (2026-10-25)

Actual copied plugins/private compositor pass two partial cases: controller killed
after reservation before issue, and after only DMs has loaded/four subscriptions.
Both retain the unbound candidate and original preconstruction guardian; rollback
requires measured transport closure plus original ECHILD. Stable resumes with10/14
subscriptions, six unchanged sessions and byte-identical unknown DM without replay.
Candidate gate evidence is unchanged; four guardian receipts and two rolled-back
journals independently verify, both scopes inactive. No persistent contract change.
Narrows M303's native partial rollback gap for these cases only. Failed partial
transport cleanup, Theme/all-member partial startup, initial adoption/cross-session
recovery and public profile commands remain. See
diagnostics/2026-10-25-native-partial-rollback.md.

## M305 — Failed candidate cleanup resumes without losing state (2026-10-25)

Journal3 retains fenced rollback and reservation/original-guardian-bound candidate
recovery alongside outgoing recovery.179 private tests and actual DMs-only native
recovery pass: rejected deletion keeps the fence, wrong guardian refuses, two
recovery controller kills do not repeat cleanup, and stable resumes14 subscriptions
with seven closed channels, six unchanged sessions and byte-identical unknown DM.
Full shared regression passes five kills/15 subscriptions/ten closed channels.
Original format2 writer/history and nine old generations verify; eight scopes are
inactive. Earlier absent-journal and nested-lock fixture failures are retained.
Supersedes M304's failed partial cleanup gap only for DMs-only startup; Theme/all
members, initial/cross-session bootstrap and public profiles remain. No production
resources touched. See diagnostics/2026-10-25-candidate-cleanup-recovery.md.

## M306 — Theme and full-set candidate recovery preserve appearance (2026-10-25)

Actual private native Theme-only/all-member/DMs-only cases pass rejected cleanup,
original-guardian verification and two recovery crashes apiece. Stable returns with
11/15/14 subscriptions and9/10/7 closed channels; all six sessions and unknown DM
persist without login/replay. Theme candidate appearance is observed, then the
retained stable checkpoint is verified/restored. Six guardian receipts and journals
independently verify; four scopes inactive. Core/179 M305 test sources unchanged.
The first Theme trial's cohort-loop variable bug is retained and corrected. This
supersedes M305's Theme/all-member candidate gap only; initial adoption/cross-session
bootstrap, corrupt admission and public profiles remain. No production changes.
See diagnostics/2026-10-25-candidate-startup-cohorts.md.

## M307 — Initial activation replays with the original live shell (2026-10-25)

Four actual private controller kills cover reservation-before-issue, construction-
before-adoption, adoption-before-binding and binding-before-response. One exact
initial reservation and only three primaries/five subscriptions survive; changed
collection/epoch refuse before construction. Completed retries are byte-identical
and issue no HTTP; six sessions and the unresolved DM persist. Original guardian,
canonical four-event journal, binding and ended guards independently verify. Both
scopes inactive; core/179 test sources unchanged. Trial01's process-shape comparison
failure is retained and corrected. Narrows M306 initial adoption only for the
original live owner; interrupted provisioning, owner/compositor loss, cross-session
bootstrap and public profiles remain. No production change. Evidence:
diagnostics/2026-10-25-initial-activation-replay.md.

## M308 — Initial owner loss recovers before canonical adoption (2026-10-25)

Version1 prepared/verified initial cleanup binds original reservation/guardian,
canonical registration history and all application/session/channel cohorts.
Reservation3 restart retains the old ownership and immutable cleanup receipt while
pinning a fresh epoch/guardian before construction.205 tests/26 startup cases pass.
Actual initial shell loss plus five controller kills and rejected deletion recover
ten subscriptions/six closed channels with six unchanged sessions and byte-identical
unknown DM. Ordinary shared recovery regression passes15 subscriptions/ten channels;
four scopes inactive and receipts/history independently verify. Original reader2
refuses3; original source/histories and nine older native generations are retained.
Older generation readability is not reservation3 admission; compatible reprepare
and public upgrade preflight remain. Supersedes M307 owner-loss gap only before
adoption in the same physical compositor. Adoption-before-binding loss, partial
initial cohorts, provisioning/cross-session bootstrap and public profiles remain.
No production changes. See diagnostics/2026-10-25-initial-owner-recovery.md.

## M309 — Original-owner recovery closes the adoption/binding gap (2026-10-25)

Canonical verified cleanup can now reconcile a missing initial historical binding
from the adopted runtime, original guardian closure and dead original guard/helper
cohort before unload. Exact retries preserve bytes; reservation shapes stay2/3.
217 tests and actual private stable/dev/stable recovery across seven controller
kills pass:15 subscriptions, ten closed channels, six unchanged sessions and the
byte-identical unknown DM. Pre-adoption regression also passes five kills/ten
subscriptions. Eight scopes inactive; source/history/closure hashes independently
verified. A demonstrated atomic-publication read race is fixed with bounded reopen;
missing/linked/malformed records still refuse. The first guard-setup failure did
not recur in100 diagnostic trials; its exact cause remains unconfirmed. Supersedes
M308's post-adoption/pre-binding owner-loss gap only in the same live compositor.
Partial early startup, provisioning/cross-session bootstrap, compatible-generation
preflight and public profiles remain. No production change. See
diagnostics/2026-10-25-closed-initial-binding.md.

## M310 — Early and partial initial startup recover with original cohorts (2026-10-25)

Actual private before-issue, after-issue, Omabit-only, Theme-only and DMs-only
startup recover across five controller kills each; complete-set and seven-crash
adopted recovery regress. Missing unissued records stay missing; empty/partial
claim cohorts stay unchanged. Original cleanup verifies before one replacement
construction per plugin. Six sessions persist; DMs-only/full cases preserve an
actual unknown outgoing message, while absent DMs state is distinguished from
message preservation. Independent records/history/closures verify;37 kills,
14 guardian receipts, eight inactive scopes. Core/217 M309 test sources unchanged.
The first fixture's cohort-variable shadowing failure is retained/corrected.
Supersedes M308/M309 early-startup limits only for characterized cohorts in the
same live compositor. General corruption repair, interrupted provisioning,
cross-session bootstrap, compatible-generation preflight and public profiles
remain. No production changes. See diagnostics/2026-10-25-initial-startup-cohorts.md.

## M311 — Prepared generations declare runtime protocols before effects (2026-10-25)

Artifact4 pins an explicit protocol declaration checked statically against bundled
reader constants. Formats1–3 stay unchanged/readable with undeclared capability;
original writer3 reproduces and refuses4.233 tests pass, including16 compatibility,
publication and CLI cases. Actual native mixed-generation enable/forward-drain
refuse without changing the live collection; protocol4 post-adoption/initial-owner/
candidate recovery and artifact3 regression pass. Sessions and unknown DM persist;
source/artifact/history/closure evidence verifies and seven scopes are inactive.
Preflight never gates required candidate cleanup. The new public command is
read-only; metadata is a versioned bundle contract, not a code audit or sandbox.
Supersedes M308–M310's explicit capability-check gap for qualified preparation and
private runtime integration only. Public activation/legacy live upgrade,
interrupted provisioning and cross-session bootstrap remain. No production change.
See diagnostics/2026-10-25-runtime-protocol-preflight.md.

## M312 — Explicit authority setup survives controller death (2026-10-25)

A separate format1 intent binds the original compositor/coordinator and staged
inodes. Atomic no-replace publication retains unbound partial objects and refuses
changed/missing bound evidence; legacy partial setup is never guessed. Fresh empty
journals persist before intent so a new controller can reopen them.27 new tests and
all260 collection tests pass. Two native runs each survive13 setup-controller kills,
then seven post-adoption or five initial-owner recovery kills. Original authority,
six sessions, unknown DM and current histories persist; subscriptions return and
channels close. Completed setup replay after reservation2/3 recovery is unchanged.
Five scopes inactive; both compositors exit zero; original sources/populated intent
and initial three test failures retained. Supersedes M307–M311's provisioning limit
only for new explicit same-compositor attempts. Cross-session bootstrap, public
activation/upgrade and full shell remain. No production changes. See
 diagnostics/2026-10-25-authority-provisioning.md.

## M313 — A new compositor cannot inherit the old authority (2026-10-25)

Private native characterization closes the original compositor/shell/owned tree,
then starts another compositor at the same socket pathname. Two remote channels
(five subscriptions) remain undeleted despite original guardian closure. Existing
and fresh-coordinator admission paths refuse unchanged; saved enabled selections
produce fenced hosts and zero new apps. Six sessions and unknown DM persist. Both
compositors exit zero, scope inactive; artifacts/source/history/closure independently
verified. Core and260-test sources remain M312. This refines its cross-session gap;
it does not implement recovery. Retained-evidence inspection, explicit original
transport cleanup, persistent evidence outside XDG_RUNTIME_DIR and linked successor
ownership remain. Runtime-directory removal, reboot/keyring unlock and full shell
are unqualified. No production changes. See diagnostics/2026-10-25-compositor-loss.md.

## M314 — Inspect retained authority without reopening admission (2026-10-25)

Read-only inspection validates original objects/history/bindings and guardian
closure after compositor death; the explicit CLI gives summary/details without
loading an active Authority. Other PID namespaces are unobservable; closure never
implies transport cleanup.20 focused/280 total tests pass. Native CLI/internal
snapshots agree before/after compositor replacement and on populated reservation3,
without state changes. Four scopes inactive; three compositor exits zero. Formats
and active admission code unchanged. Supersedes M313's inspection gap only; explicit
old-cohort transport cleanup, durable evidence outside runtime directories and
successor ownership/bootstrap remain. Original missing/replaced objects refuse;
archives do not become authority. No production change. See
 diagnostics/2026-10-25-retained-authority-inspection.md.

## M315 — Explicit original-desktop transport cleanup survives controller loss (2026-10-25)

Separate format1 prepared/verified receipts bind original authority/history,
guardian and all app/session/channel cohorts. Compositor death plus original
closure permits only explicit cleanup of a fully bound committed collection;
no pending switch or missing original evidence is inferred away.17 focused/297
total tests pass. Two native runs each survive two controller kills and rejected
deletion, close four channels/five subscriptions, and preserve six sessions and
unknown DM. Verified retry has no HTTP; completed replay unchanged; final old
Authority still refuses. Four scopes inactive/four compositor exits zero. Original
writer/records retained; runtime/journal formats unchanged. Supersedes M313/M314's
transport gap only for this state. Lost receipts/runtime directories, pending or
unbound activation, durable cross-session successor/bootstrap and public activation
remain. No production changes. See diagnostics/2026-10-25-desktop-transport-cleanup.md.

## M316 — Verified cleanup evidence survives original runtime-path loss (2026-10-25)

Immutable checkpoint1 copies exact original authority/history/cleanup metadata
outside the disposable runtime. It preserves the canonical coordinator/lock and
journal prefix, never reopens admission.17 focused/314 total tests pass. The private
native run survives two checkpoint kills after13 setup/two cleanup kills; a new
controller reads/retries unchanged after runtime retirement. Four channels/five
subscriptions close; six sessions and unknown DM persist; original activation
refuses. Three scopes inactive/two compositors exit zero. Writer and populated
records retained; existing parser extraction verified equivalent. Supersedes
M315's runtime-path dependency only for evidence captured after verified cleanup.
Lost original cleanup, capture after loss, pending/unbound state, linked successor,
reboot/keyring and public activation remain. No production changes. See
 diagnostics/2026-10-25-desktop-runtime-checkpoint.md.

## M317 — A preserved checkpoint can reserve one linked successor (2026-10-25)

Immutable successor intent1 binds original checkpoint/coordinator/history/closed
cohort and an explicit new live display. It records the staged claim inode before
no-replace publication; retries preserve it and competitors/changed or missing
bound evidence refuse.22 focused/336 full tests pass, including three real controller
kills in each final run. Four scopes inactive; all trials and original populated
records retained. First trial's reader-only maximum field comparison bug corrected;
negative tests strengthened. M316 core sources unchanged. Supersedes its missing
successor-intent/exclusive-claim contract only. Cohort adapter remains synthetic;
native plugin verification, successor admission/versioned generations, second runtime
loss and public bootstrap remain. No production changes. See
 diagnostics/2026-10-25-desktop-successor-ownership.md.

## M318 — Actual Theme/DMs cohorts reserve a successor through controller loss (2026-10-25)

Private copied plugin helpers verify original generations, closed transport and
saved app/session/pending state while holding their admission locks. Three controller
kills preserve one successor intent/claim; wrong pending observation refuses before
staging. No HTTP/login/message replay or new application; six disposable sessions
and unresolved DM persist. Three compositor exits zero/scope inactive. Original
history/checkpoint preserved and runtime-retirement read still passes.336-test core
unchanged; sources/artifacts/receipts independently verified. Supersedes M317's
synthetic-only cohort qualification for this state. Successor authority/admission,
second runtime loss, reboot/keyring/public bootstrap remain unqualified. No production
changes. See diagnostics/2026-10-25-native-successor-ownership.md.

## M319 — Linked authority2 admits guards through a preserved canonical transition (2026-10-25)

New descriptor2/publication1 preserves original mirror/checkpoint/history and binds
staged objects before promotion. Initial reservation needs the exact canonical
replacement effect, original closed cohort and new guardian; no journal reset or
initial re-adoption.19 focused/355 total tests pass; each final run has four actual
publication-controller kills and three admitted/ended guards.48 journal appends
preserve live guards; missing checkpoint fences them while allowing end. Lost bound
publication records now refuse reconstruction; unbound retries require their original
temporary inode. Six scopes
inactive; original writer/populated records and both fixture-failure trials retained.
Old readers refuse2; prior336-test sources and old formats unchanged. Supersedes
M318's absent descriptor/admission contract only at the private core level. New native
application/appearance observations remain synthetic. Compatible immutable generations,
native activation, second session loss, public/installed bootstrap remain. No production
changes. See diagnostics/2026-10-25-linked-successor-authority.md.

## M320 — Complete immutable bundles declare authority1/2 compatibility (2026-10-25)

Artifact5 pins the successor guard and complete local import set; protocol2 explicitly
declares authority readers1/2 while preserving old metadata/writers/state.18 focused /
373 total tests pass, including imports without checkout/site packages, static
non-execution, mixed4/5 refusal for2, original4 reproduction/refusal5, tampering and
lost publication reply. New preflight defaults2; public read-only CLI keeps1 and adds
an explicit flag.43 frozen original4 files verify. Initial missing deferred import
and an accidental repeated focused run are retained; four scopes inactive. Supersedes
M319's missing package compatibility only; native successor activation/subscription
recovery, second session loss, public/installed workflows remain. No production changes.
See diagnostics/2026-10-25-successor-runtime-bundles.md.

## M321 — Actual native apps reconnect on linked authority2 through controller loss (2026-10-25)

Artifact5 Theme/DMs/Omabit resume the same profile on a fresh owned software desktop.
Four publication and three activation controller kills preserve one reservation,
three loaded instances, five fresh subscriptions, six saved sessions and the unknown
DM without login/replay. Canonical history extends and commits; old mirror/checkpoint
remain. All six channels close and all three guards end at teardown; three compositors
exit0/scope inactive. Earlier20 kills and original-runtime retirement still pass.
Ten artifacts/receipts/source independently verify;373-test core unchanged. Supersedes
M319/M320's missing native successor activation only for the cleaned committed,
same-profile/same-generation path with unlocked private keyring and synthetic Eyre.
Different-generation upgrades, repeated desktop loss, public/installed profiles,
full-shell/logout/reboot/keyring and real backend release gates remain. No production
changes. See diagnostics/2026-10-25-native-successor-activation.md.

## M322 — Native package4-to5 continuation preserves profile state and sessions (2026-10-25)

Old artifact4/native authority1 advances through verified desktop cleanup to three
new artifact5 hashes under linked authority2. Old-set admission refuses; canonical
history retains the predecessor and commits the new collection. Seven new controller
kills preserve one three-app activation, five subscriptions, six sessions and the
unresolved DM without replay. Ten old/nine new preparation records verify, with
identical application inputs/payloads. Earlier20 kills and runtime retirement pass;
three compositors exit0/scope inactive;373-test core unchanged. Supersedes M321's
same-generation-only limit for this private packaging/guard upgrade. App-data format
migration, repeated desktop loss, public/installed workflows, reboot/keyring and real
backend release qualification remain. No production changes. See
 diagnostics/2026-10-25-native-successor-upgrade.md.

## M323 — Linked authority2 remains inspectable after desktop and ancestor-runtime loss (2026-10-25)

A separate read-only inspector validates authority2 publication, lineage/checkpoint,
canonical prefix and current guard/guardian bindings. Original reader/persistent
formats unchanged; output2 explicitly grants no admission or transport cleanup.
18 focused/391 total tests pass; initial synthetic append fixture corrected and
retained. Actual native upgrade passes five CLI/internal snapshots and27 controller
kills, including closed successor compositor and retired predecessor runtime.
Final metadata/artifacts verify; four scopes inactive/three compositors exit0.
Supersedes M322's missing linked retained inspection only. Repeated-session cleanup,
checkpoint/admission, public/installed profiles, reboot/keyring remain. No production
changes. See diagnostics/2026-10-25-linked-authority-inspection.md.

## M324 — Linked desktop transport cleanup survives a second compositor loss (2026-10-25)

Separate cleanup2 binds authority2 lineage, current committed activation, original
guardian closure and locked plugin cohort; original writers remain unchanged and
old reader refuses2.19 focused/410 total tests and actual native upgrade/loss pass.
Two channels remain after process closure; rejected DELETE and two cleanup controller
kills preserve one intent and recover with all six channels closed. Six sessions and
unknown DM remain unchanged without login/replay;29 total injected kills, three
compositors exit0/four scopes inactive. Expected QML display-loss exit255 is recorded;
historical admitted claims are not rewritten as process-closure proof. Initial
ancestor-path fixture error retained. Supersedes M323's missing linked cleanup only.
Linked checkpoints/repeated admission, pending/unbound cases, public/installed
profiles and reboot/keyring remain. No production changes. See
 diagnostics/2026-10-25-linked-desktop-cleanup.md.

## M325 — Linked checkpoint survives retirement of the replacement runtime (2026-10-25)

Separate checkpoint2 captures cleaned authority2 metadata and references checkpoint1
by inode/hash without embedding its body. Canonical immutable links/cleanup/ancestor
remain required; historical writers unchanged and original reader refuses2.
20 focused/430 total tests and actual native capture/retirement pass on first runs.
Two publication controller kills preserve one checkpoint through lost replies; after
both runtime directories are retained elsewhere, a new controller reads/retries with
empty replacement directories and no reconstructed authority.31 total injected kills,
three compositors exit0/three scopes inactive. Six sessions and unknown DM were
preserved before private-keyring shutdown; no login/replay or production changes.
Supersedes M324's missing linked durable checkpoint only. Checkpoint2 admission,
public/installed profiles, pending/unbound cases and reboot/keyring remain. See
 diagnostics/2026-10-25-linked-desktop-checkpoint.md.

## M326 — A fresh desktop claims checkpoint2 with the actual closed plugin cohort (2026-10-25)

Separate continuation-intent2 checks checkpoint2 ancestry, its current mirror, canonical
committed history and closed cohort before staging. Distinct desktop-continuation
prefix avoids conflating intent, linked cleanup and checkpoint files. Original writers
unchanged; original intent reader refuses2.24 focused/454 total tests and actual native
ownership pass on first runs. Three controller kills preserve one intent/claim and
abandoned stage; no HTTP/login/replay/new apps, six sessions/unknown DM unchanged.
Prior checkpoint/runtime retirement still passes;34 total native kills, four compositor
exits0/three scopes inactive. Unit fixture retires both prior runtimes before claiming;
native verifies actual keyring/plugin locks before its private keyring closes.
Supersedes M325's missing checkpoint2 ownership only; authority/guard admission,
claimed-successor loss, public/installed profiles and reboot/keyring remain. No
production changes. See diagnostics/2026-10-25-linked-session-ownership.md.

## M327 — Authority3 admits owned guards from checkpoint2 without resetting history (2026-10-25)

Separate descriptor3/publication1 and authority1/2/3 reader preserve old writers and
canonical lineage; older descriptors refuse3. Checkpoint ancestry is verified without
ancestor runtime access or a second canonical lock.21 focused/475 total tests pass;
four controller kills/three guard exits0 in each final run,48 journal appends retain
guards, missing parent/grandparent fences them. Initial preservation-fixture errors
retained; corrected fixture keeps prior binding unchanged and final full run uses
three distinct new hashes. Three scopes inactive. Exported sources/metadata checked;
application/appearance observations synthetic. Supersedes M326's missing core
admission contract only. Compatible packages, actual native activation, claimed
successor loss, authority3 cleanup/checkpoint and public/reboot/keyring gates remain.
No production changes. See diagnostics/2026-10-25-checkpoint2-authority.md.

## M328 — Artifact6 explicitly bundles checkpoint2 continuation readers (2026-10-25)

Authority1/2/3, original protocol declarations and transitive dependencies are pinned
and statically verified in artifact6. Original artifact5 writer/generation is frozen,
reproduces exactly and refuses6; historical1–5 readers remain.18 focused/493 full tests
pass first runs. Mixed5/6 accepts2 but refuses3, old preflight refuses6, and public
read-only CLI retains default1. Exported bytes/source verified; two scopes inactive.
Supersedes M327's missing compatible immutable packaging only. Actual native admission,
public installed profiles, repeated cleanup/checkpoints, claimed successor loss and
reboot/keyring remain. No production changes. See
 diagnostics/2026-10-25-continuation-runtime-bundles.md.

## M329 — Native checkpoint2 activation preserves sessions across package5-to6 (2026-10-25)

Actual copied native plugins activate under authority3, with old5 capability refusal
and new6 compatibility. Seven new controller kills preserve one three-app activation,
five subscriptions, six sessions and the unresolved DM without login/replay. All41
injected kills pass; eight channels close, guardian/guards close, four compositors exit0
and scope inactive.28 artifact records/source/metadata verify;493-test core unchanged.
Supersedes M326–M328's missing native checkpoint2 admission only. Private software
compositor/unlocked keyring/synthetic Eyre; public installed/full-shell/reboot/keyring,
application-data migration and real backend gates remain. Repeated-session delivery
must use bounded reusable lineage, not a new persistent schema per desktop restart.
No production changes. See diagnostics/2026-10-25-native-checkpoint2-activation.md.

## M330 — Authority3 is inspectable after compositor and ancestor-runtime loss (2026-10-25)

Separate read-only output3 preserves old1/2 readers and reports complete checkpoint2
ancestry without live Authority construction.19 focused/512 full tests and five native
CLI/internal snapshots pass first runs. Original inode/byte inventories stay unchanged,
including after both ancestor runtimes retire.41 native controller kills, six sessions,
pending DM and five new subscriptions pass; eight channels close, four compositors
exit0 and three scopes inactive.28 immutable bundles match M329. Supersedes M329's
missing retained authority3 inspection only; no activation/cleanup authority is inferred.
Reusable repeated-session lineage, claimed/unbound loss, public profiles/reboot/keyring
and real backend gates remain. No production changes. See
 diagnostics/2026-10-25-checkpoint2-inspection.md.

## M331 — Authority3 transport cleanup preserves state after third-compositor loss (2026-10-25)

Cleanup3 binds full ancestry, committed activation and original guardian closure,
using a distinct desktop-chain-cleanup prefix.20 focused/532 full tests pass; old
cleanup readers/writers remain unchanged. Native display-loss exit255 leaves two
remote channels. Wrong guardian, rejected DELETE and two controller kills preserve
one intent; verified replay sends no HTTP. All eight channels close, six sessions and
unknown DM remain without login/replay.43 injected kills, five inspection snapshots,
four compositor exits0/five inactive scopes;28 unchanged artifact records verify.
Initial native HTTP idle-reset fixture failure retained; narrow pre-parse fix and4
regressions pass before the fresh native pass. Supersedes M330's missing authority3 cleanup only. Reusable bounded lineage is next;
no new persistent schema per further restart. Public/pending/unbound/claimed-successor,
reboot/keyring/full-shell/real-backend gates remain. No production changes. See
 diagnostics/2026-10-25-checkpoint2-cleanup.md and repeated-session-recovery.md.

## M332 — Bounded iterative legacy ancestry reader (2026-10-25)

19 focused private tests pass first run. The reader validates checkpoint1/2 with
their original validators, exact canonical links and current journal prefixes;
it never opens retired runtimes or acquires a second lease.64 nodes/4096 files/
64 MiB including a16 MiB journal reserve bound allocation and traversal. Exact
limits pass; exceeded limits, repeated nodes and altered evidence refuse without
writes. Valid atomic journal appends pass. All previous source files match M331.
This advances M331's reusable-lineage next step only; it does not supersede any
persistent format or qualify repeated runtime activation. Do not create a version
per restart. Next implement the reusable family, then repeated same-package sessions.
See diagnostics/2026-10-25-bounded-checkpoint-ancestry.md.

## M333 — Checkpoint3 excludes all ancestor bodies and survives retirement (2026-10-25)

23 checkpoint tests plus19 ancestry regressions pass. The current envelope captures
cleaned authority3, references checkpoint2/1, checks aggregate capacity before
publication, and preserves exact retry after runtime retirement. Original writers
are unchanged; checkpoint2 refuses3. The reader now traverses3→2→1. This extends
M332's supported reader versions, without changing its limits or admission boundary.
The first harness selected zero tests: retain it as invalid acceptance. Future
launchers must assert the intended test count; corrected42-case run passes.
This is not reusable runtime qualification yet. Keep checkpoint3 for the eventual
reusable authority; do not add a format per restart. Resolve cross-version intent
ownership before admission: old and new writers must not independently claim one
legacy predecessor. No user action is needed. See
diagnostics/2026-10-25-reusable-checkpoint-envelope.md.

## M334 — Reusable intent fences legacy writers before runtime ownership (2026-10-25)

Intent3 imports checkpoint1/2/3, binds one closed cohort/display, then publishes
object-bound refusal records in both legacy intent prefixes before the runtime
claim. Original writers are unchanged and refuse the new fence shapes.20 private
tests pass, including four lost replies and an older writer winning between fences;
the latter preserves both intents and refuses new ownership without applications.
Missing/replaced fences cannot be reconstructed. The initial test-count assertion
caught leaked class aliases before execution; corrected20-case suite passes and
the failed launch is retained. Supersedes M333's unresolved preparation ownership
gap only. Reusable authority/inspection/cleanup still need runtime qualification;
same checkpoint/intent formats must be reused for subsequent sessions. See
diagnostics/2026-10-25-reusable-session-ownership.md.

## M335 — One recovery family survives three successive sessions (2026-10-25)

Authority4/intent3/checkpoint3/cleanup4 now repeat without another format per
restart.14 focused and608 full tests pass first run. Nine actual guards admit and
close across three sessions with unchanged synthetic artifact identifiers and
application/credential/pending-DM cohorts. Direct legacy checkpoint1/2 imports pass.
Six copied checkpoints and the canonical journal independently validate; ancestor
bodies remain excluded. Four lost publication replies, missing fences/mirrors,
cohort changes, failed cleanup and24 live-guard journal appends pass. Both scopes
inactive. Supersedes M334's missing reusable authority core only; real copied
plugins and immutable package execution are still pending. Preserve the same
family for further sessions. See diagnostics/2026-10-25-reusable-runtime-core.md.

## M336 — Artifact7 packages one reusable runtime family (2026-10-25)

18 package and2 unchanged historical CLI tests pass. Artifact7 explicitly supports
authority1/2/3/4, imports without checkout/site packages and preserves original
writers. Frozen artifact6 (83 files, ed20c91e931071c489f72d252bdc615f902778e62b489595dcaf5c8f987d9030)
reproduces exactly; old reader refuses7 and mixed6/7 allows3/refuses4. Never rerun
its freeze builder. The first package run exposed the missing CLI reader update;
retain that failure. Updated preflight defaults to authority1 and supports explicit4.
Three scopes inactive. Supersedes M335's packaging gap only; actual repeated native
Theme/DMs still need qualification with one unchanged artifact7 generation set.
See diagnostics/2026-10-25-reusable-runtime-packages.md.

## M337 — Three native sessions reuse one package and recovery family (2026-10-25)

Actual copied Theme/DMs pass checkpoint1→3→3→3 with identical artifact7 hashes.
45 repeated-session controller kills preserve one three-app activation per session,
six credentials,15 fresh subscriptions and the unknown DM without login or replay.
Wrong guardian/rejected DELETE preserve intent; all remote fixture channels close.
Four checkpoint bodies/canonical history/immutable bundles independently verify;
five compositor exits0 and expected display-loss roots255 have original guardian
closure. Two setup failures (fresh path prefix and missing package-reader imports)
are retained. Supersedes M336's missing same-package native proof only. Reusable
versions stay fixed; no schema per restart. Next address claimed-but-unpublished
loss and pending/unbound states, then public profile integration. Full-shell,
reboot/keyring, real backends and Docker private-chain gates remain independent.
See diagnostics/2026-10-25-native-repeated-sessions.md.

## M338 — Keyed Docker shutdown must prove both acknowledgment and exit (2026-10-25)

The guest-only lifecycle source passes14 inert boundary checks. Creation mounts
fresh keys/pill read-only with named nested piers; reopen omits boot-only flags.
Exact container/image/anchor/command/mount/resource identity gates every effect.
First keyed shutdown uses Lens; reopen uses TERM. Persist before dispatch, never
resend an unknown stop, and require acknowledged exit0 without OOM. No forced
Docker stop fallback. These are source checks only: no actual Docker or ship ran.
Two pre-test refusals are retained (sandbox namespace restriction and corrected
14-case count). M337's repeated native proof remains unchanged. Immediate work now
advances Docker private chains independently; this supersedes only M337's proposed
next-task order, not its outstanding profile recovery requirements. Public/installed
profiles and claimed/unpublished/unbound states remain open. See
 diagnostics/2026-10-25-keyed-docker-lifecycle-source.md.

## M339 — Docker's pinned identity may be the manifest digest (2026-10-25)

The first actual private-chain VM refuses gateway admission before Start: Docker's
containerd store reports the pinned manifest digest, not only the config digest.
No gateway/Urbit process or fresh ship key ran. Exact pinned reference plus either
verified digest now passes15 inert lifecycle tests. Supersedes M338's single-digest
assumption only; Docker messaging/recovery is still pending. The371.99s failed run
shuts down cleanly. A second read-only no-NIC guest confirms the serial receipt,
never-started gateway and unchanged disk hashes; use this bounded metadata reader
when full raw conversion cannot satisfy quota, without relaxing its reserve.
Retain the failed disk in verified lossless gzip privately, never export keys/piers.
Code-only caches/public runtime artifacts relocate with verified copies and stable
symlink paths; preserve the relocation receipt. An earlier preboot hash-path error,
EXDEV base-link refusal and read-only cache cleanup refusal remain recorded.
Next rerun with a fresh overlay/keys; no user action needed. See
 diagnostics/2026-10-25-keyed-docker-image-pin.md.

## M340 — A gateway using the Vere image must override its declared volume (2026-10-25)

After M339's pin check passes, the image's automatic anonymous `/urbit` volume is
refused before gateway Start. Read-only audit proves the declaration, zero start
timestamp and unchanged disks. No ship/pier/ship-key was created. The draft now
uses explicit empty read-only `/urbit` tmpfs and `/lab` cwd; actual runtime remains
pending. Preserve the refusal, do not relax filesystem admission to accept volumes.
Both diagnosed empty VM disks are now retired with source/receipts/audits/hashes
and retirement records retained. This explicitly supersedes M339's raw empty-VM
archive retention status only. Actual ship piers remain excluded from cleanup.
Pristine code-image relocation kept its hash/path but did not free temporary quota;
do not reduce the8GiB admission or2GiB running reserve. Next fresh appliance03/run04
uses the mount fix. No user action is needed. See
 diagnostics/2026-10-25-keyed-docker-gateway-mount.md.

## M341 — Stream package archives within the existing verification limits (2026-10-25)

Fresh Docker run04 stops before host installation or any ship: engine evidence is
missing although installed bytes match the signed package. Original verifier
reproduces twice; a full-runner trace proves native tar timeout while buffering
100.8MB. A smaller diagnostic pass was insufficient. Streaming authenticated tar
members fixes the full runner on the same read-only disk:8 roles verified in27.25s,
unchanged limits/receipts/disks, RAM-only writes.4 focused/60 full top-level tests
and vet pass; preserve trailing-budget and fixture-runtime test failures.
Supersedes the unresolved bootstrap diagnosis, not M340's pending gateway/ship
runtime gates. Empty pre-install disk retired after retaining source/audits/hashes;
actual piers remain excluded. Next fresh VM uses this candidate plus explicit
single-attempt offline verification before installation. No production changes.
See diagnostics/2026-10-25-package-payload-stream.md.

## M342 — Match native addons to the supplied runtime, not the base image (2026-10-25)

Fresh Docker VM05 passes M341 verification/install and M340 explicit mounts, then
gateway startup fails before keys or peers. Its Alpine marker selects a musl
LevelDB addon for glibc Node. Pin LIBC/PREBUILDS_ONLY, actual glibc runtime, addon
path and hash; a private native-module/database-reopen probe passes.17 inert helper
tests pass for bounded private stderr capture; stdout-only diagnostics had hidden
the error. Read-only audit leaves disk hashes unchanged. Zero-ship disk retired
with code/audits retained; immutable appliances move to separate private lab
storage without weakening8GiB/2GiB admission/reserve. Supersedes M340's pending
mount proof and M341's pending fresh-install verification only. Actual addon-in-
Docker, messaging/recovery and all broader gates remain pending. For namespace
checks, avoid mistaking a host sysfs bind mount for the private network view.
See diagnostics/2026-10-25-keyed-docker-gateway-libc.md.

## M343 — Qualify the packaged runtime without host-library fallback (2026-10-25)

Run06 selects the correct glibc addon, then fails before keys/peers because Node's
own dependency list omitted the addon's libpthread.so.0. M342's host-library probe
proved selection/loading, not package completeness; this supersedes only that
assumption. Inspect14 pinned Linux x64 addons, retain their full library closure,
and test the exact loader/path with no host /usr/lib, /lib, /lib64 or loader cache.
The minimal baseline reproduces; adding the pinned14,360-byte library passes
Ganache load and private database reopen. Next appliance packages that exact set.
Three auditor tests plus4 negative cases pass; private Hoon error text is hashed,
with only fixed codes/source locations exported after keys exist. Read-only audit
preserves disk hashes. Zero-ship failed VM retired with source/proof retained.
Actual Docker keyed messaging and all broader gates remain open. See
 diagnostics/2026-10-25-keyed-docker-node-closure.md.

## M344 — Measure public batch evaluation inside the VM (2026-10-25)

Run07 loads the complete native runtime and validates fresh point keys, then fails
at its256-galaxy evaluator before any ship. A read-only public-data reproduction
fails at30s and passes in71.45s with120s; the original generic error lacks exact
exit detail, so distinguish reproduction from direct observation. Extend only that
batch allowance, retain other limits, and save bounded nonsecret evaluator metadata.
Five inert checks pass; new gateway runtime remains pending. Preserve the first
diagnostic's missing-tmpfs-target failure. Key-bearing run07 disk stays private with
verified relocation/hash and stable old path; do not retire it as an empty VM.
All scopes inactive and disks unchanged. Supersedes M343's pending actual native
addon loading only. No Docker keyed messaging or broader completion claim. See
 diagnostics/2026-10-25-keyed-docker-evaluator-budget.md.

## M345 — Preserve the guest-only Docker sources in the repository (2026-10-25)

Exact frozen gateway/peer/case sources are integrated with21 inert lifecycle and
host-entry checks. They refuse before engine/process construction on the host.
The expanded read-only inspector passes four top-level tests and10 negative cases,
exporting only fixed scalar evaluator metadata or opaque hashes. Earlier readers
stay unchanged. This supersedes temporary-only source ownership, not any runtime
acceptance gate. VM08 has reached gateway readiness/first boot; its full keyed
Docker result is still pending. The reusable outer launcher, runner integration
and broader roadmap remain open. No production or installed-plugin changes. See
 diagnostics/2026-10-25-keyed-docker-entry.md.

## M346 — Official keyed Docker peers preserve state and subscriptions (2026-10-25)

Fresh no-NIC VM08 passes in1918.39s:6 acknowledgments/1 intentional nack,
4 unreplayed writes,7 scries,6 snapshots/2 deleted channels, original sessions/cursor
and same-pier receiver replacement. Two acknowledged initial Lens stops and one
reopened TERM stop all exit0; gateway/VM stop cleanly. Read-only audit verifies
receipt/state/cursor coherence and unchanged disk hashes; both scopes inactive.
Exact repository sources match the frozen appliance. Five auditor tests plus10
negative cases now prevent raw application values escaping through peer error text.
Keys/piers/chain/logs remain private; only code/public metadata/hashes are exported.
Supersedes the pending keyed Docker messaging/recovery gate in M338–M345 only.
Reusable launcher, runner integration, private-project isolation and broader goal
remain open. See diagnostics/2026-10-25-keyed-docker-recovery.md.

## M347 — Reuse the qualified VM boot boundary in source (2026-10-25)

The common launcher accepts explicit private-chain artifacts with M346's read-only
backing/no-NIC/resource/deadline settings. Eleven inert tests pass; malformed/mixed
cases and changed backing refuse,16GiB host-memory admission remains, and existing
import/fake1GiB reserves are preserved alongside the private2GiB reserve. Apart from
those guards, repository-path resolution and reserve selection, normalization
matches the qualified frozen driver byte-for-byte. New end-to-end source launch
and reusable preparation remain pending. M346's key-bearing disk is verified in
private home storage with its old lab path preserved; this supersedes only its
physical temporary-storage location, not its successful evidence. No raw state
export or production change. See diagnostics/2026-10-25-private-chain-vm-launcher.md.

## M348 — Reusable verified private Docker preparation (2026-10-25)

The source command prepares the explicit pristine/toolchain/unkeyed-chain/runner
inputs without starting runtimes.32 inert Python checks and18 Go payload tests
including subcases pass. The successful sealed12,357-file/642,437,372-byte inventory
matches M346 exactly. Old/candidate runner and guest hashes bind init replacement.
Both precreation refusals are preserved: low temporary quota, then unavailable home
quota reporting. Four stopped code appliances were hash-preserved in private home
storage; a fresh quota-verifiable temporary output passes without weaker limits.
Supersedes M347's pending reusable-preparation status only. Source-entry VM09 is
running; its runtime result, runner classification and public devnet lifecycle
remain separate. No raw state export or production change. See
diagnostics/2026-10-25-private-docker-preparation.md.

## M349 — Distinct private-chain offline plan and project binding (2026-10-25)

A separate private-local-keyed-v1 model/CLI binds chain1337 genesis and the nine
qualified contract addresses/code hashes, exactly fresh ~zod/~nec members and
acknowledged first Lens Stop. Cloneable genesis and ship names do not identify a
project: runner/project/generation also bind runtime observations.188 top-level
regressions/1,392 including subcases and vet pass; the built CLI produces JSON/text
without registry access. Fake registration refuses the private plan. This advances
M348's pending classification work only at pure review/CLI level. Existing fake
formats, stored schemas and runtime capabilities remain unchanged; persistent
private-project lifecycle, publication and isolation still need implementation.
Source-entry VM09 is independently running. See
private-chain-project-contract.md and diagnostics/2026-10-25-private-chain-plan.md.

## M350 — Reusable stopped-disk auditor with fixed metadata output (2026-10-25)

The source auditor has8 inert admission/mount checks,40 Python regressions,
25 reader tests/subcases, vet and host PID1 refusal. It replaces the machine-local
test fixture with synthetic state and exports only explicit public fields/hashes:
raw diagnostic maps/logs and receipt text no longer leave the reader. Wrong project,
key/ID/channel shapes, RPC methods and shutdown state refuse. The no-NIC reader VM
uses read-only/noload disks and requires unchanged hashes, clean exit and inactive
scope. Supersedes one-off audit-source availability only; the new source command's
runtime audit is pending VM09 shutdown. M346's reader/output remain frozen and
independently qualified. No production or installed-plugin effects. See
diagnostics/2026-10-25-private-docker-auditor-source.md.

## M351 — Route private-chain review separately in agent skills (2026-10-25)

The operator/developer guidance now identifies M349's implemented offline command,
public foundation input and distinct project/generation binding. Fake registration
is explicitly separate; private runtime and source qualification are not conflated.
Both skills validate, six relative links resolve and real command help opens no
registry. Existing system Python supplies the validator's YAML dependency; no
installation was needed. Extends the earlier skills for M349, without superseding
M350's pending runtime-audit gate or granting live-operation authority. Installed
skills/plugins remain unchanged. See diagnostics/2026-10-25-private-chain-skill.md.

## M352 — Repository preparation, Docker recovery and read-only audit pass (2026-10-25)

Fresh source-entry VM09 passes in1946.4s and the new source auditor passes
in11.19s. Six acks/one intentional nack, four unreplayed writes, seven
scries, six snapshots/two channel deletions, original sessions/cursor and same-pier
replacement all pass. Three ships, gateway and VM stop cleanly; both exact scopes
are inactive. Fresh project/keys differ from M346 while the unkeyed foundation is
unchanged. The read-only audit preserves both disk hashes and exports only fixed
metadata/hashes. Supersedes pending source preparation/launch/audit runtime status
in M347/M348/M350. Private runner lifecycle, publication, simultaneous project
isolation and release remain open; M349's planner is still offline. See
diagnostics/2026-10-25-private-docker-source-runtime.md.

## M353 — Private-chain ownership persists separately with original-writer coverage (2026-10-25)

Schema41 reserves reviewed private plans and shared member names/HTTP ports without
runtime effects. Actor-scoped replay, restart, rollback, corruption and concurrent
admission pass28 focused checks,279 historical/reservation checks and29 race checks;
vet passes. The original40 source is frozen, its populated continuation fixture
reproduces, and its reader refuses actual populated41 unchanged. Full managed
regression remains in progress. Supersedes M349's absent persistence only; public
API, runtime, publication and simultaneous-project isolation remain open. VM09's
verified private relocation supersedes physical location only; keys/piers remain
unexported. See diagnostics/2026-10-25-private-chain-project-store.md.

## M354 — Public private-chain reservations and saved inventory (2026-10-25)

Separate capability-checked registration/inventory API and CLI now bind reviewed
plans to pinned runners. Observer reads export no private runtime tokens; no chain,
key or ship is created. Initial lost-reply tests caught an omitted lookup header;
the targeted common-transport fix passes replay regressions. Five full affected
packages pass765 checks including subcases, managed API race67 and vet; both skills
and built offline walkthrough pass. Full store regression1,367 also passes, closing
M353's pending gate. Supersedes M353's absent public interface and M351's review-only
guidance, not runtime support. Next are verified inputs and a coherent durable
private lifecycle. No installed/production changes. See
diagnostics/2026-10-25-private-chain-public-reservations.md.

## M355 — Gateway resume must never repeat initialization (2026-10-25)

The qualified gateway copies the unkeyed foundation and generates keys at startup;
it is initializer-only. A separate retained-state source candidate now has36 inert
checks, exact prior-result/input bindings, fresh read-only chain/key observations,
unchanged-head shutdown and host-entry refusal. It opens no key files and contains
no initialization/mutation path. Child setup initially hit sandbox EPERM; scoped
inert execution passes. Existing qualified initializer remains frozen. Actual
Ganache reopen with zero wallet accounts, ownership-aware VM controller, peer
reconnection and public lifecycle remain unqualified. Refines M354's runtime-input
next step with this restart prerequisite; no runtime gate is superseded. See
diagnostics/2026-10-25-private-gateway-resume-source.md.

## M356 — Ownership-bound retained gateway trial is running (2026-10-25)

The separate guest controller validates the exact stopped predecessor, immutable
inputs and chain directory before replacement.49 inert Python checks pass. The
sealed appliance adds only three helpers and the appended guest hook; all other
inputs match VM09. Fresh no-NIC VM10 is admitted under the existing resource limits.
Supersedes M355's absent controller/preparation only, not its pending Ganache
reopen/runtime gate. No public lifecycle is enabled. An independent resume audit
must follow shutdown before a runtime pass. See
diagnostics/2026-10-25-private-gateway-resume-preparation.md.

## M357 — Required resume proof in the independent disk auditor (2026-10-25)

The source reader now binds the explicit resume case to its original proof,
retained chain inode, replacement/operation, unchanged public chain observations,
RPC refusal and clean shutdown.69 Go checks,50 Python tests,vet and host exit80
pass; fixed projections exclude raw secrets/state. A random fixture-path rejection
was corrected without relaxing admission. Supersedes M356's absent resume auditor
source only. VM10 and the actual read-only audit remain pending. Prior qualified
reader evidence stays immutable. See
diagnostics/2026-10-25-private-gateway-resume-auditor.md.

## M358 — Retain client intent independently of the gateway (2026-10-25)

The original fixture keeps cookies in gateway tmpfs, so whole-gateway replacement
needs a separate private client checkpoint. The explicit source variant preserves
original cookies, pier identities and an unread fifth-write subscription update,
reopens peers without boot material and checks revisions5–7 without write replay.
55 Python checks,109 Go audit checks,vet and host refusal pass. Supersedes the
absent peer-reconnect fixture only, not any runtime gate. Preparation refused the
8GiB quota admission before output creation; VM10 continues with frozen inputs.
Keep immutable runtime inputs, project state and client intent separately owned.
See diagnostics/2026-10-25-private-gateway-peer-source.md.

## M359 — Final predecessor hash is a separate retained-chain proof (2026-10-25)

M355–M358's first resume candidate binds original height/keys and compares hashes
within the reopened service, but lacks the original final-head hash. The legacy
top-level blockHash predates gateway setup and must not substitute for it. The
next initializer records gateway.headHash; resume requires an exact match before
readiness and the peer auditor requires explicit coverage.39 JS checks,110 Go
checks and vet pass. Clarifies/supersedes any broader interpretation of the earlier
head-preservation claim, not a runtime result. Frozen VM10 stays unchanged and
will expose its limited coverage; exact predecessor-hash runtime remains pending.
See diagnostics/2026-10-25-private-gateway-original-head.md.

## M360 — VM10 peer pass, retained gateway refused a fresh-copy mode mismatch (2026-10-25)

VM10's original two-ship/state/subscription case passes independent audit, but the
appended resume gate fails before replacement creation: the copied chain is0755,
not required0700. Owner/canonical path are correct. Do not assume directory modes
survive packaging. Next initialization refuses existing destinations and makes only
the fresh copied chain0700; retained checks stay strict.111 reader checks,55 Python
checks and vet pass; actual corrected reopen remains pending. Supersedes M356–M359's
VM10-in-progress status with an overall refusal and a qualified peer subgate. Both
read-only audits preserve hashes; all three scopes are inactive and the original
disk is privately retained. No production/installed state changed. See
diagnostics/2026-10-25-private-gateway-resume10-refusal.md.

## M361 — Preserve stopped labs before retrying storage admission (2026-10-25)

A second fresh preparation refused before output creation at7.50GB quota headroom.
Three exact stopped disposable plugin labs were privately retained with complete
hash/mode verification and original-path symlinks, freeing2.49GB. Neither8GiB
admission nor2GiB running reserve was relaxed. Fresh preparation now succeeds;
VM11 exercises corrected fresh-chain mode, exact predecessor hash and original
client/peer recovery. Supersedes M360's stale preparation status only; runtime and
independent audit remain pending. No installed or production state changed. See
diagnostics/2026-10-25-private-gateway-peer-preparation11.md.

## M362 — Verify inert inputs separately from runtime authority (2026-10-25)

`private-chain-inputs` now verifies a local sealed dependency generation against
separate manifest/foundation review hashes, with no registry, engine, execution or
runtime authority. Actual12,352-file/834MB inspection, nine built CLI cases,468
affected race checks,vet and two skills pass. Original format1 writer/manifest/
reader are retained; unknown formats refuse. The producer's first pre-creation
path refusal is preserved; a fresh embedded-payload preparation passes. Refines
M354's absent input boundary only: guest-only inputs are not supported delivery,
publisher authentication or chain observations. Schema41 and VM11's frozen inputs
are unchanged; whole-gateway runtime remains pending. See
diagnostics/2026-10-25-private-runtime-inputs.md.

## M363 — A same-device bind mount still changes the input boundary (2026-10-25)

M362's unchanged reader passes a real private-namespace pass/refuse/pass sequence:
binding the exact Node file over itself refuses despite unchanged device/content.
All three capped scopes are inactive and no dependency/runtime was executed.
Supplements M362's mount-traversal evidence only; no format, authority or schema
changes. Do not substitute a device-number check for no-mount traversal. VM11's
runtime recovery and independent disk audit remain pending. See
diagnostics/2026-10-25-private-runtime-input-mounts.md.

## M364 — A retired namespace number is not a project generation (2026-10-25)

VM11 passes its original peer/state case and reaches retained-gateway readiness
with a read-only RPC probe, but refuses before peer creation because the namespace
number repeats after predecessor shutdown. Namespace inode strings are not durable
generation IDs. The corrected fixture keeps distinct container ownership, stopped
predecessor, network=none and live guest/peer namespace checks; historical equality
is explicit metadata.117 Go checks,vet and54 Python tests pass; runtime fix remains
pending. All four read-only audits preserve hashes, all scopes are inactive and
the disk is privately retained. Outer wrapper143 remains unexplained, separately
from independently proven clean VM exit. Supersedes M361–M363 pending status and
M358's historical-number inequality, not isolation or full recovery qualification.
See diagnostics/2026-10-25-private-gateway-peer11-refusal.md.

## M365 — Locked credential reads/saves preserve independently stored sessions (2026-10-25)

Unchanged Theme/DMs stores pass16 real private-keyring helper checks across stable/
dev identities: four locked reads and replacement saves refuse; original cookies
and DMs metadata survive an owned daemon restart with the original password.
Fresh source-entry run and host/output refusal checks pass; all scopes inactive.
Two earlier CLI-unlock trials remain failures: exit0 did not unlock the collection.
Narrowly supplements M337's keyring evidence; UI unlock/full desktop reboot and
Eyre recovery remain unqualified. Private keyring data/logs stay out of evidence.
No plugin/runner format or installed state changed. VM12 continues with frozen
namespace-corrected inputs. See diagnostics/2026-10-25-plugin-keyring-lock.md.

## M366 — Prepare inert inputs through a reproducible source entry point (2026-10-25)

The reusable producer verifies an explicitly reviewed appliance and pinned pill,
preserves existing/partial outputs, copies selected inputs and seals them without
execution. Actual12,352-file/834MB output has M362's identical format1 manifest
and passes its independent Go reader. Eight boundary tests pass, including a
test-discovered rejected-directory FD fix. The first240s copy timeout remains
private/incomplete; a fresh900s-budget run passes unchanged validation. Supersedes
M362's one-off-only preparation gap, not runtime delivery/authority or unkeyed-DB
provenance. Schema41 unchanged; VM12/audit continue independently. See
diagnostics/2026-10-25-runtime-input-preparation.md.

## M367 — Whole private gateway, piers and client intent recover (2026-10-25)

VM12 and independent read-only audit26 pass: exact retained chain/head/public
keys; distinct replacement gateway and two same-pier containers; original sessions
and queued cursor; state revisions4→5→6→7 with no write replay; five clean ship
stops, two clean gateway stops and clean VM poweroff. Historical namespace number
reuse recurred and correctly passed distinct-container/current-membership/guest
separation checks. Both scopes are inactive and the disk is privately retained.
Supersedes M364–M366's pending whole-gateway gate; publication, concurrent private
projects and durable public runner lifecycle remain open. See
diagnostics/2026-10-25-private-gateway-peer-recovery.md.

## M368 — Opt-in direct HTTP publication enters an isolated VM (2026-10-25)

The retained gateway can be source-qualified on an exact private bridge with only
two IPv4 loopback Eyre publications. Original sessions feed a direct guest client,
with no relogin/unknown-write replay; RPC/admin/IPv6 refusal and an independent
mandatory publication audit are added.69 Python/40 Node checks and Go tests/vet
pass. The actual published ship type mismatch was caught and corrected before
launch; appliance14 remains unbooted. A quota refusal created no VM; exact private
retention restored headroom without relaxing admission. Fresh VM14/appliance15
is admitted, with audit27 queued after shutdown. This updates M367's next-step
status only: actual publication remains pending; no runner lifecycle activation
or schema/input-format change. See diagnostics/2026-10-25-private-http-publication-source.md.

## M369 — Freeze original41 before private lifecycle persistence (2026-10-25)

Original schema41 source is retained and reproduces the same synthetic fixture
twice:8 legacy instances,46 operations, existing fake recovery history and2 private
projects/4 members. A permanent race-tested preservation check reopens3 times,
retains all original rows/sidecars and actor replay without host-port probes.
The first synthetic port collision correctly refused; fresh fixture ports resolved
it without changing the original writer. This adds the future migration baseline;
it does not supersede M353's original40 evidence or claim a new migration/runtime
schema. Current schema remains41; publication VM14/audit27 continue. See
diagnostics/2026-10-25-original41-private-reservations.md.

## M370 — Private workspace directories retain identity across interruption (2026-10-25)

The plan/parent-bound workspace controller and Linux adapter pass216 package
race checks,85 workspace checks included, and vet. Exclusive0700 project/member
parents are created only after committed dispatch. Known root identity survives
sync failure; lost layout acknowledgement can be resolved by inspection/sync of
the complete retained tree without another mkdir. Partial trees, replacement
objects, symlinks and unexpected contents refuse without repair/deletion. A
changed pier-parent halts remaining creation. Tests use new temporary directories.
This adds the filesystem prerequisite for M369's future persistence migration;
it does not supersede original41 preservation or claim durable runner dispatch.
Schema41 stays unchanged; publication VM14/audit27 remain independent. See
diagnostics/2026-10-25-private-workspace.md.

## M371 — Schema42 retains private workspace intent and refuses old readers (2026-10-25)

Workspace/operation mirrors now commit atomically; actor replay, every interrupted
phase, lost replies, rollback and corruption checks pass. Startup marks unfinished
intent interrupted without repeating mkdir or changing its original binding.
The affected managed suite passes89 race checks and vet; full regression is running.
Original41 reproduces the pinned fixture and refuses actual populated42 state
without modification. M369's archive/fixture remain unchanged. This supersedes
M370's pending persistence and unpersisted timestamp spelling (now existing
seconds:nanoseconds); public queues/observation receipts remain unwired. No live
database/runtime is used. See diagnostics/2026-10-25-private-workspace-store.md.

## M372 — Published private clients survive whole-gateway recovery (2026-10-25)

VM14 and read-only audit29 pass exact loopback publication, original sessions,
queued state/subscriptions and two new direct writes with no replay. RPC/admin/
IPv6 probes refuse; chain/head/keys and piers stay bound; all shutdowns are clean.
Audit27's coverage refusal was a verifier error: four SSE acknowledgements make
seven channel PUTs, not three. Audit28's fixed counters and the Recovery source
established this;146 reader checks pass and audit29 verifies the unchanged disk.
All earlier audits/source remain; disk privately retained, all scopes inactive.
Supersedes M368–M371 pending publication and M368's three-PUT assumption, not
concurrent-project or public runner qualification. See
diagnostics/2026-10-25-private-http-publication.md.

## M373 — Concurrent private projects have an audited source fixture (2026-10-25)

Two nested fresh project cases keep four same-name keyed ships live with separate
chains, keys, namespaces and original four-write recovery receipts. Crossed cookies,
independent subscribed writes and A surviving B shutdown are mandatory. The reader
requires actual live snapshots and both clean peer/gateway proofs.90 Python suite
checks,11 focused audit-entry checks,10 launch/CPU checks and161 Go race checks plus
vet pass; compiled host entry refuses. Appliance16's12,358 entries/artifacts are
verified and frozen. It remains unbooted: available memory is below32GiB admission;
no resource limit was relaxed. This updates M372's next-step status only, not a
runtime isolation claim. Full schema42 managed regression continues independently.
See diagnostics/2026-10-25-private-project-isolation-source.md.

## M374 — Workspace scheduling shares the runner fence; full42 passes (2026-10-25)

Internal workspace dispatch now uses the shared host gate and management-update
fence, rechecks ownership, closes driver authority and preserves uncertain intent.
Replay never reads/recreates a missing parent; replacement parents refuse. Actual
new-directory integration, failure/cancellation and original41 preservation pass
139 focused race checks plus vet. Review found and fixed a typed-nil factory
failure; a negative control proves the regression test detects it. No public or
startup dispatch exists; shared storage provisioning and durable observation
recovery remain. Schema42 is unchanged. The earlier full managed run passes1416
checks in2083.390s and exits0, superseding M371–M373's pending status; that run
predates the new queue, which has its own focused evidence. M373's concurrent VM
remains unbooted awaiting memory admission. See diagnostics/2026-10-25-private-workspace-queue.md.

## M375 — First gateway effects have explicit uncertainty boundaries (2026-10-25)

The pure initializer binds project/workspace/input/engine/image and commits before
each network/container/Start effect. Known IDs survive an accompanying error;
lost replies never retry or regenerate state. Fresh readiness requires exact
loopback publication, retained objects, foundation/head and public key/life/suite
evidence. Observer-only recovery leaves the original attempt unchanged.327 package
race checks (102 gateway) and vet pass with inert drivers. This supersedes M374's
pending initializer-model work only: schema42, persistence, runtime dispatch and
installed state are unchanged. M372 proves replacement-gateway publication, not
first-initialization publication; the initial adapter still needs that evidence.
Appliance16 remains unbooted below memory admission. See
diagnostics/2026-10-25-private-gateway-start-model.md.

## M376 — Original42 preserves completed and interrupted workspace intent (2026-10-25)

Frozen original42 reproduces one synthetic fixture twice:8 legacy instances,
48 operations, existing fake recovery history,2 private projects and2 workspaces
(observed/succeeded and layout-dispatched/interrupted). A permanent race check
reopens3 times, preserves every original row/sidecar, verifies actor replay without
host probes and refuses restarting old operations. Original41/support regressions
also pass (21 checks) and vet passes. This supersedes M375's pending original42
baseline only; schema remains42 and no future migration/downgrade refusal or
runtime is claimed. Register this fixture in the prior-reader inventory only with
a real successor. See diagnostics/2026-10-25-original42-private-workspaces.md.

## M377 — Staged lifecycle adapters preserve original uncertain attempts (2026-10-25)

Three candidate tables retain first gateway initialization and distinct workspace/
gateway observation receipts. Actor replay, atomic mirrors, interruption and fresh
observation checks pass161 focused race checks plus vet, including real temporary
directory recovery after a lost reply. Review added cross-project directory claims;
a negative control proves the regression detects the omission. Original attempts
remain unchanged, and observation interfaces cannot create/start resources.
This supersedes M375/M376's pending adapter source only: the seven candidate files
remain staged, schema42 is unchanged and the combined43 migration/global startup
hooks are not activated. No concrete gateway runtime or public dispatch is claimed.
See diagnostics/2026-10-25-private-lifecycle-store-proposal.md.

## M378 — Schema43 retains separate recovery evidence through upgrades (2026-10-25)

Status: integrated; focused/source and original42 refusal passed; full regression running.
Three new tables now participate in transactional migration, global reservation
validation and normal startup reconciliation. Original42 rows/sidecars remain;
new observation receipts preserve unknown creation/Start attempts. Failed migration
rolls back all tables and the marker; corrupt mirrors refuse unchanged.51 top-level/
280 focused race checks and vet pass. The original42 writer reproduces unchanged
and refuses genuine populated43. A corrected test now permits exactly six new
receipts while comparing every original field; its initial failure is retained.
A negative control proves the cross-domain directory guard is exercised.
This supersedes M377's staged-only/migration/startup gaps and M376's pending
successor qualification only. Full43 race session13388 is running; no public runtime,
installed upgrade or initial HTTP publication is claimed. No protected state accessed.
See diagnostics/2026-10-25-private-lifecycle-schema43.md.

## M379 — Staged lifecycle queues preserve replay and observation boundaries (2026-10-25)

Status: staged outside checkout;63 race checks and vet pass. Three operations use
the common work/update gates, replay before backend reads, validate fresh queued
receipts and close owned handles on every outcome. Actual temporary-directory
recovery after a lost layout reply preserves original intent; observation factories
hide creation methods. Gateway drivers are inert. Failed trial1 used an ineligible
completed workspace, and trial2 had a test syntax error; both are retained and the
corrected fixture passes. This supersedes M378's pending scheduler source only;
full43 session13388 remains running and does not include the new queue files.
Integrate after that baseline, then qualify affected checks. Concrete gateway/runtime
and public orchestration remain. See diagnostics/2026-10-25-private-lifecycle-queues.md.

## M380 — First-start publication source has a mandatory separate audit (2026-10-25)

Status: integrated, source-qualified; runtime pending. An explicit first-start mode
uses two private guest-loopback Eyre ports and exact owned bridge routes. Original
sessions feed direct HTTP; a separate two-write/four-subscription extension preserves
the original recovery transcript. The reader binds head hash, state, receipts,
cursors, both channels and private session files; initial uses eight channel PUTs,
replacement remains seven.4 JavaScript/74 Python/198 Go race checks and vet pass;
host reader exits80. Review caught and corrected the previous loopback-only peer
guard before runtime. Staging-loader/preparation-wrapper failures are retained.
Preparation17 hit the8GiB `/tmp` reserve before output; verified unbooted appliance16
was relocated with all20 files/325,434,805 bytes unchanged and its old path retained
as a symlink. Fresh private-directory preparation19 is running. This supersedes
M375/M379's pending fixture/audit source only; M372 replacement evidence remains,
and first-start/concurrent runtime and public dispatch are still open. See
diagnostics/2026-10-25-private-initial-publication-source.md.

## M381 — Full43 baseline and integrated lifecycle queues pass (2026-10-25)

Status: integrated/qualified source; runtime publication still running. The unchanged
schema43 baseline passes408 top-level/1,839 race checks in2,411.22s. M379's two queue
files then integrate unchanged and pass27/132 affected race checks plus vet. The
full baseline excludes those later queues; do not claim a later full rerun. This
supersedes M378/M379's full-regression pending and M379's staged-only statements.
M380's preparation19 refused unsupported output quota before creating an appliance;
the quota guard remains. Fresh appliance20 passes, with all23 files retained privately
unchanged behind its original path. VM15/session54631 is running under the admitted
no-NIC/12GiB/no-swap/two-core boundary; stopped-disk audit30 is mandatory. No runtime
pass, concrete gateway backend, public dispatch or protected-state change is claimed.
A staged follow-up aligns older workspace replay/typed-nil handling with the newer
queues; tests run separately. See diagnostics/2026-10-25-private-lifecycle-integrated.md.

## M382 — Older workspace queue shares replay and handle checks (2026-10-25)

Status: integrated and source-qualified. Retained workspace receipts now replay
before backend lookup. Fresh work still requires runtime support. Shared worker,
admission, exact receipt and handle checks reject rebound actor/request/revision
or project and typed-nil drivers without creating directories, crashing or retrying.
The new replay test fails against the original backend-first source and passes
with the fix.13 top-level/91 race checks plus vet pass; after byte-identical install,
seven/28 overlapping workspace checks pass. This supersedes M374's replay/typed-nil
gaps and M381's staged follow-up only. Schema/public routes/runtime authority stay
unchanged; first-start VM15/audit30 session54631 remains running. See
diagnostics/2026-10-25-private-workspace-queue-parity.md.

## M383 — Private gateway network ownership has a concrete inspection contract (2026-10-25)

Status: integrated/source-qualified. The runner derives bridge arguments and exact
labels from retained gateway/workspace/input selections. Inspection binds network
and sole gateway endpoint IDs, private canonical IPv4 allocation and all required
network flags/options; foreign participants and duplicate/case-aliased JSON refuse.
Eight top-level/76 race checks plus vet pass; six/74 overlapping checks pass after
byte-identical integration. No engine is contacted and no runtime authorization is
granted. This supersedes M382's pending network-contract source only. Concrete
container/input/engine adapters, shared parent provisioning and public lifecycle
remain. VM15/audit30 session54631 and its frozen sources are unchanged; first-start
publication remains pending. See diagnostics/2026-10-25-private-gateway-network-contract.md.

## M384 — First-start HTTP publication passes runtime and independent audit (2026-10-25)

Fresh no-NIC VM15/rootless engine and read-only audit30 pass cleanly. Original
four-write receiver recovery is preserved; original separate sessions perform23
direct requests, two new writes, four subscription reads and three scries, reaching
revision6 without replay. Both channels close; cross-cookie and five unpublished
RPC/admin/IPv6 probes refuse. The reader binds complete state/cursors/receipts,
initial/final network/head and private session files without exporting their values.
Both scopes are inactive; base/overlay are unchanged. The3,665,690,624-byte disk is
retained privately with verified bytes and its original path preserved as a link.
This supersedes M380–M383's pending first-start runtime/audit only. M372 replacement
proof remains distinct. Concurrent appliance16 still lacks32GiB memory admission;
concrete runner gateway/public lifecycle and production delivery remain open.
See diagnostics/2026-10-25-private-initial-publication-runtime.md.

## M385 — Private gateway container and publication checks integrate (2026-10-25)

Status: source-qualified. Commands and inspection bind exact immutable inputs,
project-only writable storage, resource/namespace limits, command/environment and
two loopback Eyre ports. Created/running states are distinct; unexpected storage,
privileges, process history and endpoints refuse. Network allocations now carry
their resource IDs to prevent cross-resource substitution.14 top-level/168 race
checks plus vet pass;12/166 overlapping checks pass after integration. Trial1's
fixture map alias, not a runtime failure, is corrected and its evidence retained.
This supersedes M383 allocation binding and M384 pending container-contract source
only. No engine/ship ran, schema/public routes stay unchanged and the guest-only
program guard remains. Concrete backend, shared parent and public lifecycle remain.
See diagnostics/2026-10-25-private-gateway-container-contract.md.

## M386 — Current initializer and resume programs have a new sealed generation (2026-10-25)

Generation06 contains12,352 files/834,036,904 bytes from fresh unbooted appliance21.
Independent built-collector inspection in private HOME/XDG passes all bytes,
permissions and object checks and reports runtime_authorized=false. New manifest
44782315f0cf0bc1732b57a76387dde975dd7d23decaf6f5cf7a0258c514d9db binds M384's
initializer and unchanged resume programs; generation04 remains byte-identical.
Review/ordering/path mistakes in wrappers refused before the preparer or collector
effects and are retained; fresh corrected inspector03 passes. No VM/engine/input
program ran and no installed generation changed. This supersedes M366's current
initializer applicability and M385's pending generation inspection only. The
guest-only guards, concrete backend and public-delivery gaps remain. See
diagnostics/2026-10-25-private-gateway-input-generation.md.

## M387 — Completed workspaces have a read-only identity-bound verifier (2026-10-25)

OpenWorkspaceVerifier holds the recorded parent and exposes only VerifyEmpty/Close.
Repeated fresh reads compare every retained directory identity, so same-shaped
replacement piers/member parents refuse. No repair, chmod, mkdir or sync occurs.
62 focused race checks/vet pass; a negative control without the child-identity
comparison fails both replacement cases. After integration, all46 top-level/354
privatechain package race checks pass. This supersedes M386's pending verifier,
not concrete engine/input/dispatch or public lifecycle work. Preserve the empty
workspace until journalled initialization; the guest candidate's home/marker/
namespace metadata must not be created by an unrecorded preflight shortcut.
Schema43 and protected/installed state remain unchanged. See
diagnostics/2026-10-25-private-workspace-verifier.md.

## M388 — Runtime inputs stay bound to the held directory (2026-10-25)

OpenRuntimeInputVerifier checks root/review identifiers; Verify freshly hashes
contents and brackets the scan with pathname/object checks. Identical-byte root
replacement, missing/aliased ancestors, permission changes and closed handles
refuse. The offline inspector now shares this path.65 focused race checks/vet,
two complete generation06 scans,364 integrated privatechain checks and3 collector
checks pass. The first ancestor fixture failed before its assertion because Linux
requires write permission when moving a directory between parents; corrected
setup restores0500 before opening, with the failed evidence retained. No pins,
format or schema changed; no bundle code executed. Supersedes M386 snapshot-only
binding and M387 pending input handle only. Concrete dispatch and journalled
initialization remain. See diagnostics/2026-10-25-private-input-verifier.md.

## M389 — Gateway metadata belongs to recorded Start (2026-10-25)

GatewayBootstrap opens a planned journal and held empty workspace read-only;
Initialize requires the same start-dispatched selection before creating the
private home/marker/current-process namespace metadata. First write is fenced,
files are exclusive and synced, and partial failures remain without retry or
cleanup. No chain, key, pier or engine work occurs.31 focused race checks/vet
include actual filesystem effects under the existing controller with an inert
engine: failed/lost commits do not write, partial bootstrap blocks Start, and
lost post-Start replies never repeat it. All395 privatechain checks pass after
integration. Supersedes M387-M388 missing journal-gated metadata helper only;
concrete runtime callers, observation and fresh VM qualification remain open.
The guest-only guards stay intact; this metadata does not prove a no-NIC VM.
See diagnostics/2026-10-25-private-gateway-bootstrap.md.

## M390 — Initial gateway resource effects now have a concrete adapter (2026-10-25)

The internal adapter binds M388 inputs/M389 bootstrap to the selected rootless
engine/image and M383/M385 network/container contracts. It checks current mounts,
loopback ports and the empty workspace before effects, retains known IDs with
errors, refuses unknown retries and rehashes inputs immediately before startup.
Metadata initialization occurs only inside recorded Start; Close releases both
handles.45 focused race checks/vet pass, including journal order, mount overlap,
lost replies and changed inputs during Docker inspections. A real generation06
constructor/real temporary workspace test passes with an inert engine. All211
private-gateway race checks pass after integration. The initial test fixture's
config-vs-manifest image mismatch refused before effects; failed evidence remains.
Supersedes M389's missing concrete resource methods only. Fresh chain/runtime
observation, managed backend wiring, client/parent provisioning and a new no-NIC
VM run remain. No actual Docker, ship, installed plugin or service access occurred.
See diagnostics/2026-10-25-private-gateway-resources.md.

## M391 — Sealed database inputs need a fresh writable copy (2026-10-25)

The original cpSync preserved0400 database files; an unprivileged synthetic
read/write open reproduced EACCES. The initializer now makes an exclusive private
copy with0700 directories/0600 files and fsyncs files/directories/parent. It never
changes source or existing retained chains and never resumes partial output.
Actual generation06 foundation copy passes:7 files/291,373 bytes, unchanged source,
matching private writable copy, retry refused.58 integrated source checks pass.
A reporter-format assertion initially stopped integration before edits; explicit
TAP/no process isolation gives detailed counts and a failing sentinel confirms
callbacks run. Evidence is retained. New initializer hash is5c1f9da910456ad76c708153380c940b288ab8c0a2334971f9c8b8cc3eb55d66.
Supersedes M386 generation06 suitability for future sealed-bundle boot only;
its manifest/inspection and historical VM evidence stay unchanged. Build a new
input generation after remaining startup/readiness changes; do not repair06.
Readiness must query live RPC/code/public keys at the same explicit head; getPoint
uses an earlier block. Also establish post-initialization durability before
claiming Synced: the new copy fsync does not cover subsequent key/chain writes.
No gateway, engine, ship or installed state was operated. See
diagnostics/2026-10-25-private-gateway-fresh-copy.md.

## M392 — Initial startup now has an explicit durability checkpoint (2026-10-25)

The initializer closes and syncs its private state, reopens without signing
accounts, checks chain/genesis/head/code/public keys again and syncs once more.
Private umask covers new files. Exclusive ready-file publication and parent sync
precede the public in-memory status; a publication failure closes the listener.
All 114 extracted-helper/copy/initial/guarded-resume source checks pass, including
wrong code/key rows, moving heads, lifecycle failures and retained partial output.
No full initializer, chain library, engine or ship ran. Supersedes M391's pending
checkpoint source work only; it does not prove runtime or power-loss durability.
Generation06 stays unchanged; make new sealed inputs after readiness code settles.
Next implement a read-only external observer that combines fresh RPC evidence with
engine/container/namespaces and filesystem identity checks. Never accept a saved
ready file alone or send mutation probes from observation. Managed wiring and a
fresh no-NIC VM qualification remain. See
diagnostics/2026-10-25-private-gateway-startup-checkpoint.md.

## M393 — Fresh gateway RPC evidence has a bounded read-only probe (2026-10-25)

The embedded probe checks startup status, chain/genesis/head, nine code hashes
and both explicit-head getKeys rows, bracketed by status and namespace reads.
Its nineteen requests use a fixed loopback endpoint and only read methods;
body/time/output limits and a fixed exact-container Node command bound execution.
The Go parser validates the selected foundation/member order and rejects malformed
or ambiguous output. Errors discard raw diagnostics without retries. All 46 JS
checks and 40 focused Go race checks/vet pass; 251 private-gateway Go race checks
pass after integration. The initial expected request count of twenty was a fixture
mistake; corrected nineteen and failed evidence are retained. All transports were
inert. Supersedes M392's missing external RPC probe only. Its result grants no
readiness by itself: still bind current engine/container/network/namespaces and
filesystem identities, verify the qualified initializer, compose managed backend,
build new inputs and qualify in a fresh no-NIC VM. See
diagnostics/2026-10-25-private-gateway-rpc-probe.md.

## M394 — Initialized gateway storage has a read-only metadata verifier (2026-10-25)

GatewayStateVerifier preserves the retained workspace/piers/member identities
and binds chain/secrets on first successful opening. Two bounded descriptor-based
scans must agree; private modes, owner, regular single-link files, expected first-
start layout and empty member parents are required. O_PATH inspects regular file
metadata without reading key/database contents. No sync, repair or creation is
exposed. All 27 focused race checks/vet and all 422 privatechain package race
checks pass on synthetic fixtures. Supersedes M393's missing initialized-state
verifier only. Its metadata does not prove database validity or durability, and
reopening alone cannot prove historical chain/secrets identities. Complete live
engine/RPC/namespace attribution and fresh no-NIC VM qualification remain next.
Same-UID isolation and wallet-safe key storage are not claimed. See
diagnostics/2026-10-25-private-gateway-state-verifier.md.

## M395 — Gateway readiness now combines live and filesystem evidence (2026-10-25)

The observer brackets RPC with exact engine/image/container/bridge, namespace,
mount overlap, input and initialized-state checks. A fixed M392 initializer pin
is required before trusting startup flags; no mutation methods are exposed.
All 91 focused race checks/vet, 434 privatechain and 285 private-gateway race
checks pass. Actual generation06 fully rehashes but its old initializer is refused
before engine access. No input program or actual gateway ran. Supersedes M394's
missing observer composition only. Next enforce the same program pin in resource
preflight before any creation/Start; otherwise an obsolete bundle could initialize
and only then fail observation. Also add bounded first-start readiness waiting,
managed/client/parent wiring, new inputs and fresh no-NIC VM qualification.
Historical kernel mount identity and same-UID mutation remain explicit limits.
See diagnostics/2026-10-25-private-gateway-observer.md.

## M396 — Creation preflight and readiness share one initializer pin (2026-10-25)

The resource input precondition now verifies the fixed M392 initializer as well
as the full generation. It covers all creation checks and the late recheck before
bootstrap/Start; the observer reuses that same path. All 82 focused race checks/
vet and 287 integrated private-gateway checks pass. Actual generation06 fully
rehashes but is refused by the controller before any journal dispatch, metadata
initialization or engine command. A late program change also prevents Start.
Supersedes M390's permissive program admission and M395's pre-start gap only;
historical evidence remains unchanged. No input program or actual engine ran.
Next prepare new sealed inputs, add bounded first-start waiting, wire the managed
backend/client/parent and qualify in a fresh no-NIC VM. See
diagnostics/2026-10-25-private-gateway-preflight-pin.md.

## M397 — Generation07 supplies the qualified startup source (2026-10-25)

A fresh collector verifies the new sealed generation:12,352 files/834,044,028
bytes, manifest fd4c0739be071ef2ad88fb10bcd17cbc93c712e6dc7544ca8a2c9f0232f7832a.
The M392 initializer and public foundation are pinned; generation06 is unchanged.
Temporary quota refusal preceded output. Bounded verified retention of unbooted
appliances restored quota reserve without weakening admission. Home Btrfs lacks
the quota query; artifact retention used verified copy/sync and an atomic
compatibility link, not a VM admission fallback. Preparation23's frozen runner is
historical infrastructure, never booted and not current-adapter qualification.
Supersedes M396's missing new input generation only. No bundled program or engine
ran; actual runtime qualification and supported delivery remain. See
diagnostics/2026-10-25-private-gateway-input-generation-07.md.

## M398 — First Start now waits through read-only observation (2026-10-25)

The private driver combines the resource controller and observer with a bounded
wait only after successful Start. It never repeats effects, aborts changed runtime
invariants and retains the same observer handles across retries. Unknown Start
outcomes still require the separate observer, which has no creation methods.
All26 focused race/vet checks pass, including actual generation07 and a genuine
temporary workspace with an inert engine;312 integrated gateway checks pass.
A first overly short caller-deadline fixture expired during setup, safely refusing
before Start; the corrected readiness-specific fixture and failed evidence remain.
Supersedes M395/M396's pending first-start waiting only. Managed backend and owned
client/parent provisioning, supported inputs and fresh no-NIC runtime qualification
remain. No real engine, ship or installed state was accessed. See
diagnostics/2026-10-25-private-gateway-first-start-wait.md.

## M399 — Managed gateway factories own their input and client selection (2026-10-25)

The real backend now composes the internal queues with fixed manifest-named input
selection, read-only preflight, exclusive client metadata and the first-start or
read-only observer adapters. Held private directories, an exact public marker and
an empty Docker config are checked around every CLI command. Changed config or
objects refuse; known creation IDs survive post-command failure. Partial setup
and Close preserve evidence. Recovery exposes no creation methods.42 focused
race/vet checks (including real generation07/workspace with an inert engine),83
queue checks and353 integrated gateway checks pass; runner compilation passes.
Supersedes M398's missing backend/client composition only. No public route,
startup dispatch, schema change or actual engine execution occurred. Shared
private-projects parent provisioning, supported bundle delivery and fresh no-NIC
VM qualification remain. Never silently recreate a missing parent with retained
workspace claims. Same-UID checks are not wallet security. See
diagnostics/2026-10-25-private-gateway-managed-session.md.

## M400 — First-use grouping metadata respects retained workspace claims (2026-10-25)

The internal queue provisions only an empty private shared parent after replay,
fresh review, host-gate and maintenance checks. Any retained workspace requires
its original parent identity; missing/replaced parents are never recreated or
rebound. Project roots/keys/chains/piers remain journal-gated. A failed receipt
leaves only verified empty grouping metadata and no worker; a later valid request
can reuse that same empty object.27 filesystem/50 queue race checks plus vet and
452 full privatechain/95 workspace checks pass. A0755 test-root refusal was fixed
in the fixture, not by relaxing production permissions; failed evidence remains.
Supersedes M399's missing shared-parent provisioning only. Schema43 and public
routes are unchanged; no actual engine, ship or installed state was accessed.
Supported input delivery and a fresh no-NIC VM with the current managed adapter
remain next. See diagnostics/2026-10-25-private-workspace-parent-provision.md.

## M401 — Current managed gateway Start reached; readiness remains failed (2026-10-25)

Fresh current-source no-NIC VM24 created one private workspace/gateway and
reached Start, but no successful readiness observation. Recovery never ran.
Three stopped-disk audits preserve the same disk hash; all scopes inactive.
The container lived 24.710 seconds, exit143/no OOM, with no readiness/failure file.
Its eight stored bind mounts exclude --tmpfs, which Docker stores separately
in HostConfig.Tmpfs. Requiring tmpfs in Mounts is a contract defect; regression
and fresh runtime verification follow. Preserve exact tmpfs config checks.
Supersedes M400 only for the previously unattempted current-source VM gate;
this is failed runtime evidence, not lifecycle qualification. Input delivery
is fixture-only. No protected/installed state access. See diagnostics/2026-10-25-managed-private-gateway-first-runtime.md.

## M402 — Docker --tmpfs is a separate inspection representation (2026-10-25)

Do not require --tmpfs entries in Mounts: Docker keeps them in HostConfig.Tmpfs.
The exact two bounded/private definitions remain mandatory; validate reported
entries too, and continue requiring all exact binds. The old implementation
fails both the new regression and realistic first-start fixture. Corrected
container and readiness inventory checks pass354 race checks, integration and
vet. Supersedes M401 only for the unfixed source defect; fresh runtime remains
pending. Failed disk and historical appliances are retained with verified hashes
and compatibility paths; resource floors unchanged. See diagnostics/2026-10-25-private-gateway-tmpfs-inspection.md.

## M403 — Launcher loss leaves an inconclusive runtime, not a pass (2026-10-25)

Corrected appliance25/run02 entered initialization, then its launch session
ended143 with no final result or clean shutdown. Cause unknown. Exact VM scope
is inactive. Audit04 preserved both disk/base hashes and refused uncheckpointed
SQLite; no journal replay or repair. The disk is privately retained unchanged.
Supersedes run02 running status only; M402 source checks stand, runtime/recovery
remain unqualified. Next fresh disk uses a bounded transient supervisor independent
of the interactive command session, with original VM limits and no automatic
retry. No protected/installed state access. See diagnostics/2026-10-25-managed-private-launch-interruption.md.

## M404 — Initializer ready, managed observation still unqualified (2026-10-26)

Supervised run03 reached GATEWAY_READY and clean exit0, but its manager fixture
hit the operation deadline before readiness/recovery. Audits05/06 preserve disk
and base hashes; the disk is privately retained. Actual sealed-input verification
in the one-CPU/2GiB read-only auditor took40.977s cold and10.953s warm. Runtime
uses two vCPUs/7GiB, so do not claim identical timing or sole causation. Bulk
verification can exceed the25s freshness window. No keys/runtime chain contents
read; no input programs executed. Supersedes corrected-run pending statements
only. All scopes/supervisor terminal. See diagnostics/2026-10-26-managed-gateway-readiness-timing.md.

## M405 — Separate full input preparation from live snapshot age (2026-10-26)

Initial full input verification has its own bounded2min stage; the existing25s
window starts with mutable engine/runtime observation. Final full verification
remains inside that window on the same handle. No caching or permission changes.
Original source rejects the synthetic41s preparation; corrected source passes
and still refuses a26s stale snapshot. All355 gateway race checks plus vet pass.
Supersedes M395 ordering only, not M404 failed runtime evidence. Fresh supervised
VM qualification remains next. See diagnostics/2026-10-26-private-gateway-observation-window.md.


## M406 — Keyed-member commands preserve the boot/reopen boundary (2026-10-26)

The pure private-member command builder matches the four independently qualified
Python PublishedShip invocations. Reopen excludes initial key/pill/identity/gateway
and creation flags; canonical plan and exact member/mode are mandatory. All 21
targeted race checks and package vet pass; integration preserves tested hashes.
No file/engine access or launch authority is supplied. Verified mounts, fresh
gateway binding and durable managed member lifecycle remain. Schema/public routes
are unchanged. This adds source evidence, superseding no runtime qualification.
The current readiness VM still runs frozen M405 binaries. See
diagnostics/2026-10-26-private-member-command.md.


## M407 — M405 still needs a runtime diagnosis (2026-10-26)

Fresh supervised run04 again reaches GATEWAY_READY but its manager fixture hits
the operation deadline. Container exit0/no OOM; recovery never runs. Read-only
audit01 preserves disk/base hashes; the disk is privately retained and all units
are terminal. The auditor's first input scan takes95.963s; the second is canceled
after4.033s at the shared100s limit, so it is not a warm-scan success. Do not infer
the live failing stage from those different-resource timings. Supersedes M405's
pending-runtime status only; readiness remains unqualified. A logging-only overlay
passes355 gateway race checks/vet and changes no conditions or deadlines. Next
fresh diagnostic VM will emit only fixed stages/timings/booleans, no keys or raw
errors. Production observer remains unchanged. See
diagnostics/2026-10-26-managed-gateway-observation-followup.md.

## M408 — Freeze original43 initialization and recovery before member persistence (2026-10-26)

Two original-writer extractions reproduce identical synthetic state: completed
and interrupted workspace/gateway attempts plus separate interrupted/completed
observations. Reopen three times preserves every row/sidecar, source ownership,
readiness attribution and actor replay; old operations cannot restart. Focused
race checks and vet pass, including original41/42 regressions. Schema remains43;
future migration and original-reader refusal require actual successor evidence.
No runtime operations. Supersedes no prior qualification: M376/M407 remain valid.
See diagnostics/2026-10-26-original43-private-lifecycle.md.


## M409 — Runtime trace isolates the RPC probe, not an expired snapshot (2026-10-26)

Diagnostic VM05 shuts down cleanly with initialization failed. Read-only audit
preserves disk/base hashes; disk retained, all units terminal. Final128 of300
fixed events contain eight RPC failures under2s with no expired context; preceding
input/runtime/namespace/state/mount checks pass. The RPC substage remains unknown.
Do not widen freshness deadlines on this evidence. Initializer ready/exit0;
no managed recovery or member launch. Supersedes M407 diagnostic-pending only;
M405 source checks and M408 original43 preservation remain valid. Production
observer unchanged. See diagnostics/2026-10-26-managed-gateway-rpc-diagnosis.md.


## M410 — Foundation-v1 hashes canonical code text, not decoded bytes (2026-10-26)

Read-only audit finds twelve probe attempts stopping after accepted Azimuth code
reads. Original deployment pins use SHA256 of lowercase 0x-prefixed UTF-8 hex;
the probe incorrectly decoded bytes. All nine pinned artifacts confirm the mismatch.
Corrected probe preserves existing foundation/input pins; two old-source regressions
fail,47 JS checks and355 Go race checks plus vet pass. No schema/deadline/permission
change. Supersedes M409 unknown RPC substage and unfixed source only; failed VM
evidence remains valid. Fresh ordinary-source runtime/recovery qualification is next.
See diagnostics/2026-10-26-private-gateway-code-hash.md.


## M411 — A pre-start failure does not exercise the RPC correction (2026-10-26)

Ordinary VM06 fails with the gateway still planned, before network/container
dispatch. The precise driver-opening error is not retained. Clean shutdown,
immutable read-only audit and verified private disk retention pass; all units
terminal. M410's probe was never exercised. A logging-only preflight overlay
passes355 gateway race checks/vet and leaves rules/deadlines unchanged; fresh
diagnostic VM follows. Supersedes M410 pending-runtime only, not its source fix.
No protected state accessed. See diagnostics/2026-10-26-managed-gateway-preflight-failure.md.

## M412 — Keyed member containers bind bootstrap and retained-pier modes (2026-10-26)

The pure member contract binds exact project/workspace/gateway/operation/runtime
and validates private namespace, mounts, image, labels, resources and process
state. Boot mounts only its own key/pill read-only; resume excludes them and
requires distinct retained pier identities. All77 targeted race checks and vet
pass. No dispatch/freshness authority, schema or public API change. Held storage,
member-aware gateway observation and durable lifecycle remain. Extends M406;
supersedes no runtime qualification. See diagnostics/2026-10-26-private-member-container.md.


## M413 — Hold exact member mount objects without touching peer state (2026-10-26)

The read-only member mount verifier holds and rechecks retained parent/root/piers/
member identities. First boot additionally refuses any partial child; refusal
preserves it. Peer piers, keys, chain data and engines are not inspected. All17
targeted race checks/vet pass, including replacement and concurrent close. No
creation, launch, freshness or resume-state authority; no schema/API change.
Extends M412 and supersedes no runtime evidence. VM07 runs frozen earlier binaries.
See diagnostics/2026-10-26-private-member-mount.md.


## M414 — Verify one accepted bootstrap-key file without exporting its contents (2026-10-26)

The read-only member key verifier pins exact whole-file bytes and holds the
recorded root/secrets/file objects. Replacement, unsafe metadata, format or hash
mismatch refuses without repair; diagnostics export no key/path/hash. All34
targeted race checks/vet pass with synthetic files only. Accepted digest capture,
cryptographic key/chain correspondence and fresh launch authority remain separate
driver requirements. No schema/API or actual-key operation. Extends M412/M413;
supersedes no runtime evidence. Same-UID checks are not wallet custody. See
diagnostics/2026-10-26-private-member-key.md.


## M415 — Diagnostic managed gateway initialization and recovery pass (2026-10-26)

VM07 passes all three stages with clean shutdown. Independent read-only audit
preserves disk/base hashes and confirms the original interrupted receipt, separate
actor recovery, two manager reopens, exact filesystem identities and zero ships.
RPC correction succeeds with unchanged observation limits. Disk retained, all
units terminal. Logging-only overlays remain distinct from ordinary-binary
qualification, which is next. Supersedes M411 pending diagnostic status; its
earlier pre-start cause remains unknown. M412–M414 are separate source evidence.
See diagnostics/2026-10-26-managed-gateway-recovery-diagnostic.md.


## M416 — Hash the pinned boot pill once and hold its exact objects (2026-10-26)

Private first-boot pill verification is integrated:32 targeted/109 combined
member race checks, vet and diff pass with synthetic files. Production pins
remain unchanged; held identities/metadata detect replacement after full hashing.
Resume, mutable/public/linked artifacts and extra workspace contents refuse.
The initial fixture permission failure and second wrapper's stale-log count are
retained with explicit corrections; neither weakened verifier rules. Extends
M412–M414 and supersedes only their pending pill-verifier source gate. Staging,
retained-pier/state, key proof, gateway membership and runtime remain separate.
See diagnostics/2026-10-26-private-member-pill.md.


## M417 — Ordinary managed gateway recovery passes independent audit (2026-10-26)

VM08 passes without diagnostic overlays; unchanged stopped disk/base audit
confirms interrupted-source preservation, separate actor recovery, two manager
reopens and zero ships. Verified disk retained and all owned units terminal.
Supersedes M415's pending ordinary-source gate, not M411's unexplained historical
failure. Teardown container exit143/no OOM is recorded explicitly: clean VM
shutdown is not acknowledged gateway Stop or whole-gateway restart evidence.
Managed member lifecycle/public runtime remain. See
diagnostics/2026-10-26-managed-gateway-recovery.md.


## M418 — Retained member pier identity and stopped-state metadata (2026-10-26)

Read-only held pier/.urb verification integrates with36 targeted/87 combined
race checks, vet and diff. Exact selected objects, lock absence and bounded
log/checkpoint metadata are checked without payload/peer reads or repair.
Replacement and unsafe state refuse and remain preserved. Lock absence is not
process exit, acknowledged Lens Stop or durability; the driver must establish
those separately. Extends M413 and supersedes only its pending retained-pier
metadata helper. No schema/public/runtime change. See
diagnostics/2026-10-26-private-member-pier.md.


## M419 — Bind key proof to a fresh chain observation without exporting the key (2026-10-26)

The source primitive holds one key, uses stdin-only veri:dawn evaluation and
freshly rechecks the full chain afterward. Strict Go parsing binds its hash to
the exact boot contract, public keys/head and namespace; resume refuses.25
targeted/174 combined Go race checks,46 new/47 existing JS checks, pinned ABI
synthetic decoding, host refusal and vet pass. Real key/evaluator/chain runtime,
complete admission and persistence remain. Extends M414 and supersedes only
the missing source primitive, not runtime evidence. The failed positional-ship
fixture was corrected without weakening rules. Node --test's file-only pass was
insufficient; direct --test-reporter=tap case counts are the accepted evidence.
See diagnostics/2026-10-26-private-member-key-proof.md.


## M420 — Guest-only key proof and independent retained-key audit (2026-10-26)

The tagged fixture brackets both real member proofs and four explicit %.n
controls with full observations, then uses the host key verifier and preserves
history. The tagged stopped-disk reader independently binds six canonical
receipts to retained keys and public chain evidence, exporting no key bytes or
individual key hashes. Four bridge/32 audit checks and170 integrated race checks,
three ordinary-reader checks, vet and host refusals pass. Two synthetic fixture
mistakes were corrected with failed evidence retained; rules were unchanged.
Extends M419; supersedes fixture/auditor preparation only. Actual VM proof and
member lifecycle remain. See diagnostics/2026-10-26-private-member-key-fixture.md.


## M421 — Observe gateway metadata around explicitly retained member piers (2026-10-26)

The new constructor pins ready-gateway filesystem identities and exact retained
member pier/.urb objects, leaving live payload opaque. Omitted members remain
empty; initial-start admission stays unchanged.31 new/145 combined filesystem
and633 integrated affected race checks, vet and diff pass using synthetic data.
No schema, actual pier access or dispatch. Extends M418; supersedes only the
missing filesystem helper. Engine membership and durable lifecycle remain.
Frozen VM09 excludes it and failed earlier at fresh host installation; cause is
under read-only investigation. See diagnostics/2026-10-26-private-member-gateway-state.md.


## M422 — Successful systemctl exit can still leave output pipes open (2026-10-26)

VM09 stopped cleanly during fresh-host management activation before gateway/key
proof. Two unchanged-disk audits identify Go ErrWaitDelay: successful systemctl
exit, incomplete output drain. The installer preserved partial state; no retry.
The bounded error remains private. Disk retained; all owned units terminal.
Fix acknowledgement only for explicit service effects followed by existing
activation verification; incomplete JSON/probe output must still refuse. Keep
deadlines and output bounds. Supersedes M420 pending VM09 and M421 unknown cause,
not M417/M421 evidence. Key-proof runtime remains unexercised. See
diagnostics/2026-10-26-managed-key-proof-install-failure.md.


## M423 — Acknowledge successful management starts without accepting incomplete probe output (2026-10-26)

A private typed marker distinguishes exact ErrWaitDelay with confirmed exit0,
live context and bounded buffers. Ordinary commands still refuse. Only explicit
management-service starts can acknowledge it, then require the unchanged full
activation verification; no retry or deadline relaxation. Disposable inherited-
pipe children reproduce the condition.16 targeted/400 full installer race checks,
vet and diff pass with no skips or live service access. Supersedes M422's pending
source fix, not its failed runtime or the pending key proof. Fresh VM follows.
See diagnostics/2026-10-26-management-command-drain.md.


## M424 — Bind recorded member containers around mount and sharing inventories (2026-10-26)

The read-only engine preflight requires exact cloned contracts, unique member/
runtime/container IDs and recorded process starts. It brackets full inspections,
mount objects and all-container namespace-sharing projections. Foreign project,
key/pill aliases and direct/indirect namespace joins refuse; unrelated resolved
projects remain possible.72 new/839 affected race checks, vet and diff pass.
The absent-container mock correction retains failed evidence and changes no
admission rule. Extends M412/M421; supersedes the missing engine preflight only.
Actual kernel namespace/held pier integration, observer wiring and durable
lifecycle remain. No schema/runtime/production change; frozen VM10 excludes it.
See diagnostics/2026-10-26-private-member-engine.md.


## M425 — Compose member-aware gateway observation without lifecycle effects (2026-10-26)

The private ready-gateway observer combines M421 held piers and M424 exact engine
membership with fresh running-member kernel namespace checks. Process snapshots
span RPC/filesystem observation; created/exited members receive no exec. Exact
saved gateway, cloned contracts and resume pier bindings are required. Initial
empty-member mode and both freshness budgets remain unchanged.35 composition/
874 affected race checks, vet and diff pass with synthetic evidence. Supersedes
M421/M424's pending internal composition only. Durable lifecycle/first-boot
identity capture, schema44 and actual member runtime remain; no public/schema
change. Frozen VM10 excludes this source. See
diagnostics/2026-10-26-private-member-observer.md.


## M426 — Real member key proof and independent retained-key audit pass (2026-10-26)

Fresh VM10 passes installation, gateway initialize/recover/verify and all six
key cases, then cleanly stops. The read-only audit independently confirms two
positive/four explicit negative verdicts, unchanged retained keys/chain, host
verification and preserved original recovery history across two manager reopens.
Disk/base unchanged, zero member boots, all owned units inactive, disk/audit
retained. Supersedes M419/M420 pending actual proof and M423 pending fresh-install
VM only; the ErrWaitDelay branch itself remains evidenced by M423's regression.
Extends M417. Frozen VM10 excludes M424/M425; durable member lifecycle, actual
member observer, first-boot capture and schema44 remain. See
diagnostics/2026-10-26-managed-member-key-proof.md.


## M427 — Retain initial member launch intent across uncertain effects (2026-10-26)

The model binds pinned boot inputs, private key acceptance, peer references and
exact artifact/container/process evidence. Dispatch precedes each effect; known
returned objects survive errors, and interrupted/completed state cannot repeat
boot.91 new/965 affected race checks, vet and diff pass using inert drivers.
Extends M412/M425 and supersedes only the missing launch state machine. SQL,
schema44 original-writer migration/refusal, first-boot capture, durable readiness
and actual runtime remain. No schema/public API change. See
diagnostics/2026-10-26-private-member-launch.md.


## M428 — Persist the first private member launch without restart dispatch (2026-10-26)

Schema44 retains original/recovered gateway attribution, reserved runtime identity,
actor/request replay and canonical journal/operation mirrors. Transactional phase
commits preserve known objects; restart only interrupts unfinished work. One first
member per project is admitted; durable readiness is required before second-member
support. 34 new store/two migration checks and 536 integrated race checks,
vet and diff pass. The original43 writer reproduces its fixture and refuses real
populated schema44 state unchanged. The initial test-only six-versus-two receipt
count failure and the combined-suite timeout are retained; exact prior-row
comparisons remain. The remaining tests passed in a separate partition without
repeating completed top-level cases. Supersedes M427's
pending SQL integration and M408's pending successor migration/refusal only.
Concrete driver/queue, first-boot capture, readiness and real runtime remain. The
next VM auditor must explicitly handle schema44; no public route or installed change.
See diagnostics/2026-10-26-private-member-store.md.


## M429 — Hold the empty mount across first boot before capturing pier identity (2026-10-26)

The read-only handle opens before boot, rechecks emptiness after durable Start
dispatch and accepts one matching acknowledged Start. Fresh exact process
observations bracket held pier/.urb birth identities. Replacement, links, stale
processes and incomplete capture refuse without repair or partial authority.
It reads no payload, peer or key contents. 49 new/1,014 affected race checks,
vet and diff pass using synthetic files and inert observers. Extends M413/M418/
M427/M428; supersedes only the missing held first-boot metadata primitive.
Durable capture binding, full gateway/HTTP readiness, concrete adapter/queue and
actual runtime remain. No schema/public API change. Freeze original44 before
the next persistent format. See diagnostics/2026-10-26-private-member-birth.md.


## M430 — Compose the concrete first private member driver (2026-10-26)

The private driver now connects exclusive held-root artifact publication, repeated
key/chain proof, full member-aware gateway checks, exact Create/Start, held birth
capture and final process evidence. Known partial objects survive failure; lost
Create/Start acknowledgements cannot advance, and effects/capture cannot replay.
Foreign artifact mounts include ancestor and file-inode aliases. 90 new/1,104
affected race checks, vet and diff pass using synthetic files/mocked engines.
Retained failures document a corrected synthetic boot-ID mismatch and the fixed
pill-alias gap. Supersedes only M427/M429's pending concrete first-member adapter.
Schema44, public routes and installed runtime are unchanged. Durable capture,
scheduler factory/queue, readiness, second-member admission and actual VM runtime
remain. Freeze original44 before the next format. See
diagnostics/2026-10-26-private-member-driver.md.


## M431 — Freeze original44 before durable member birth/readiness (2026-10-26)

Two independent runs of the pinned original44 writer reproduce the same synthetic
fixture with completed and Start-dispatched/interrupted member launches. 23
selected race checks, vet and diff pass. Three reopens preserve every original
cell, sidecar, runner identity, gateway attribution and actor replay; original
launches cannot restart. Supersedes M429/M430's pending original44 freeze only.
The frozen archive's inherited purpose wording is corrected here: initial member
launch persistence is present; durable birth/readiness is the next format.
Successor migration and old-reader refusal remain unqualified until implemented.
No runtime operation or protected data access. See
diagnostics/2026-10-26-original44-member-launch.md.


## M432 — Persist first-boot capture without rewriting original44 history (2026-10-26)

Schema45/version2 member launches require captured pier/.urb identities bound to
the original process, namespace and post-capture gateway evidence. Capture and
operation mirrors commit together; known private object overlaps refuse. Original
version1 rows remain unchanged and cannot execute as version2. 1,216 affected race
checks, vet and diff pass. Original44 migration preserves old rows/sidecars alongside
a new captured project through three opens; corrupt history refuses unchanged.
The original44 writer reproduces its fixture and refuses real schema45 state
unchanged. A terminated overlay run (exit143, cause unknown, no assertion failure)
is retained; separate migration and complete integrated runs pass. Supersedes
M429/M430's pending durable binding and M431's successor migration/refusal only.
Readiness, second member, scheduler/VM lifecycle and acknowledged Stop remain.
Freeze original45 before the next format. No runtime/installed changes. See
diagnostics/2026-10-26-private-member-durable-birth.md.


## M433 — Wire first-member launch into the shared runner queue (2026-10-26)

The internal queue performs actor replay before backend lookup, pure contract
binding inside acceptance, and driver execution only after durable acceptance
under the common work/update fence. The read-only session reuses the retained
gateway client and pinned inputs; owned handles close once even on failure.
95 targeted/213 integrated race checks, vet and diff pass with mocked backends
and synthetic files. Queue coverage includes cancellation, corruption, maintenance,
update fences, typed nil and partial setup; startup never dispatches. Supersedes
M430/M432's pending concrete session factory/queue only. Schema45 is unchanged.
Readiness, second member, actual VM lifecycle, acknowledged Stop and public routes
remain. Freeze original45 before the next persistent format. See
diagnostics/2026-10-26-private-member-queue.md.


## M434 — First managed-member VM fixture and schema45 auditor (2026-10-26)

The actual internal queue now has a fresh no-NIC VM fixture for first keyed boot,
captured pier identity, manager reopen/replay, HTTP/Lens identity and one-shot
acknowledged Lens shutdown. The VM-only bridge is excluded from ordinary builds;
fixture Stop is not managed Stop or application readiness. 107 queue/lifecycle
checks, vet and host refusals pass. The schema45 stopped-disk auditor adds exact
launch/operation mirrors, reopen/receipt order, pier/pill/key identity and retained
clean-exit checks; 13 audit checks and prior key-audit tests pass. Packaging first
refused the unchanged 8 GiB quota floor. Twelve completed artifact directories
were durably retained with hashes and compatibility symlinks, then a fresh build
passed. VM dispatched; runtime and independent audit are pending. No runtime
limitation is superseded. See diagnostics/2026-10-26-managed-member-boot-fixture.md.


## M435 — Freeze original45 before managed member readiness (2026-10-26)

Two independent original45 writer runs reproduce one synthetic fixture with three
private projects: two version2 captured running launches and one interrupted
Start dispatch, plus earlier gateway/fake-project histories. Three store opens
preserve all table cells, sidecars, runner identity, capture evidence and exact
actor replay; wrong actors and repeat begin refuse. The preservation race test,
existing schema45 migration/corruption checks, vet and diff pass. Two retained
new-test failures were selector mistakes (shared label, then assumed zod-first);
using the actual sorted workspace-nec instance passes with writer/fixture unchanged.
Supersedes M432–M434's pending original45 freeze only. Successor migration and
old-reader refusal remain future gates. No runtime operations. See
diagnostics/2026-10-26-original45-member-birth.md.


## M436 — Real first-member attempt exposes ready-client phase mismatch (2026-10-26)

The M434 VM passes gateway initialization/recovery/verification but its member
operation becomes interrupted while the launch remains planned. Independent
read-only inspection preserves gateway history and confirms no member launch
qualification. The session factory supplied a validated ready gateway to a client
helper that allowed reuse only at started/start-dispatched: a deterministic refusal
before member effects. The regression reproduces it; ready-phase reuse now passes
without metadata writes, engine calls, repair or weakened selection checks. 198
affected race checks, vet and diff pass. Failed VM disk/base remain unchanged;
all three units are terminal and the disk is durably retained. A new VM is needed.
Resolves M434's first-run outcome and M433's client integration gap; no runtime
readiness/second-member/Stop claim. The final auditor hash supersedes M434's prior
hash and explicitly projects private key reading when performed. Schema45 stays.
See diagnostics/2026-10-26-managed-member-ready-client.md.


## M437 — Read-only first-member HTTP readiness model (2026-10-26)

A first captured version2 member can now produce an internal HTTP identity result
bound to its launch digest, fixed endpoint, ship, pier/.urb identity, original
process and unchanged gateway generation. Fresh full observations bracket one
HTTP read; held verification and cancellation guard each step. Callback inputs
and returned snapshots are cloned, and any failure returns no partial proof.
54 targeted/159 integrated race checks, vet and diff pass with inert drivers.
This is HTTP evidence, not Gall readiness, credentials, Lens authority, persistence,
second-member admission or managed Stop. No runtime limitation is superseded.
The already-running fixed VM predates this helper and cannot qualify it. Concrete
observer and schema-backed queue remain; original45 is frozen for that future
change. See diagnostics/2026-10-26-private-member-http-readiness.md.


## M438 — Concrete read-only first-member HTTP adapter (2026-10-26)

The M437 collector now has a concrete held-object runtime adapter, restricted to
the captured first member and original gateway/process generation. Complete
gateway snapshots are bracketed by member process observations; exactly one
observed fixed-loopback HTTP read is allowed. Handles close on partial creation
and synchronize Close. 64 targeted/225 affected race checks, vet and diff pass.
This supersedes only M437's missing concrete implementation. Schema45 is unchanged;
persistence/queue, second-member admission, managed Stop and actual readiness
runtime remain open. The running fixed VM predates this source and cannot qualify
it. Original45 must qualify the next format. See
diagnostics/2026-10-26-private-member-http-driver.md.


## M439 — Fixed client reaches Start; observation failure remains (2026-10-26)

Fresh VM12 passes the three gateway stages and acknowledges member Create/Start,
then interrupts at started without a running/birth receipt. Four read-only audits
preserve the disk/base and original history. The fixture deadline did not expire;
member exit1/no OOM is recorded. A parse-pill log initially suggested an input
problem, but the retained pill/hash/mount match exactly and Vere reports `%term`,
which maps to SIGTERM. This supersedes that root-cause hypothesis: shutdown may
explain the parse message, while the first observation failure remains unresolved.
No runtime budget/check is relaxed. A fresh logging-only observation run is next;
never replay this interrupted launch. M436's client fix is proven through Start,
not a completed boot/readiness. Disk is privately retained and every owned unit
is terminal. Diagnostic input-budget details and hashes are in
diagnostics/2026-10-26-managed-member-started-diagnosis.md. Schema45 is unchanged.


## M440 — Fresh logging-only post-start observation fixture (2026-10-26)

Temporary overlays emit fixed return locations/timings/cancellation only, with
unchanged original acceptance code.579 affected race checks/vet/diff and armed
host refusal pass. A separate bounded trace auditor passes20 checks and host
refusal, and cannot claim normal runtime qualification. The new isolated VM is
running with the same resources, pinned inputs and storage reserve. Previous
failed disks are retained. This completes M439's diagnostic preparation, not its
unresolved observation cause; ordinary runtime remains unqualified. Schema45
and original45 baseline stay unchanged. See
diagnostics/2026-10-26-managed-member-observation-trace.md.


## M441 — Capture is not yet durable before full running observation (2026-10-26)

Source review confirms the first-boot pier/.urb capture stays in the driver until
ObserveRunning finishes all later gateway/process checks and the controller saves
running. An intervening failure/restart can leave only started, losing that binding.
This clarifies M432's narrower completed-running guarantee; it does not prove which
check failed in M439. Future persistence should commit validated capture first,
then support separately attributed read-only observation without repeating Start
or adopting an unbound pier. Original45 migration/refusal and interruption/reply-
loss tests are required. No format or runtime behavior is changed now. M440's
diagnostic VM remains active. See
diagnostics/2026-10-26-private-member-capture-gap.md.


## M442 — Trace identifies post-capture gateway deadline (2026-10-26)

M440's fresh diagnostic run completes birth capture, then expires the25-second
gateway observation context during its second member namespace read. The outer
launch deadline remains live. Audit16 retains1,290 valid fixed return records,
unchanged disk/base and original history. This resolves M440's pending outcome,
narrows M439's first-failure question, and demonstrates M441's capture persistence
gap in this run. No complete boot/readiness/Stop qualification or timeout change.
All owned units are terminal and the disk is retained. Next reduce redundant
read-only inventories with equivalent bracketing, then test a fresh VM. Schema45
stays unchanged; original45 remains the future persistence migration baseline.
See diagnostics/2026-10-26-member-observation-deadline.md.


## M443 — Combined inventories and overlapping final verification (2026-10-26)

The complete member gateway observation now uses45 instead of57 inert Docker
commands by combining mount/sharing snapshots. Full post-RPC input hashing runs
alongside independent final runtime checks, with cancellation/join on every exit
and serialized Close. Deadlines/freshness, contracts and schema45 stay unchanged.
791 affected race checks, vet/diff, ordinary builds and host refusal pass. Storage
preflight first refuses; verified retention recovers headroom without changing
the reserve. Fresh ordinary appliance11/run11 is dispatched; audit17 and actual
launch/recovery remain pending. This implements M442's optimization follow-up,
not a runtime fix qualification. M441 capture persistence still requires its own
original45 migration and recovery work. See
diagnostics/2026-10-26-member-observer-inventory.md.


## M444 — Staged capture-before-observation journal model (2026-10-26)

An overlay-only version3 model commits validated first-boot capture before later
running observation, retaining it on subsequent failure and refusing first-boot
replay.237 affected race checks/vet pass, including commit/reply loss, cancellation,
substitution and cloning. The old unknown-version test needed3→4; its initial
failure is retained. Repository/schema45 and active M443 VM stay unchanged.
No persistence/recovery authority is implemented yet. Next qualify schema46
canonical mirrors/global claims, original45 preservation/refusal, concrete adapter
and a fresh runtime. This advances M441's design without superseding its runtime
limitation. See diagnostics/2026-10-26-member-capture-model.md.


## M445 — Staged capture persistence preserves original45 (2026-10-26)

The schema46 overlay atomically records version3 capture and operation mirrors,
including global object claims and interruption-only reopen.47 unique selected
race checks/vet pass after splitting an initially timed-out combined harness.
The actual frozen original45 reader refuses a genuinely migrated/populated46
database unchanged. Original rows, sidecars, identities and actor replay survive.
This advances M444 persistence; repository/schema45 and M443 VM stay unchanged.
Concrete adapter, admission, support-window integration and runtime recovery are
still pending, so M441's runtime limitation remains. See
diagnostics/2026-10-26-member-capture-store.md.


## M446 — Capture adapter passes; ordinary member run remains failed (2026-10-26)

M443 run11 and audit17 are terminal: gateway setup/recovery passed, member launch
is interrupted at started, and no durable capture/readiness is present. Exact
failure location is unproven in this ordinary run. Disk/base/history are preserved
and the disk is retained. This supersedes M443's pending outcome, not M442's
separate diagnostic finding. The staged concrete capture/session adapter passes
156 race checks/vet, including held filesystem capture, commit-reply loss and
post-capture failure without repeated Start. Repository/schema45 remain unchanged;
new admission, historical tooling, integration and fresh runtime are next.
M441's runtime limitation still applies. See
diagnostics/2026-10-26-member-capture-adapter.md.


## M447 — User pause, bounded retention and current integration state (2026-10-26)

The user requested a reasonable stopping point, storage cleanup and a manual
review. The goal is PAUSED; no test batch or experiment VM remains running.
Schema46 capture-before-observation is now integrated (27 files), superseding
M446's repository/schema45 and pending-integration status only.197 focused
admission/queue/HTTP race checks pass. The broader integration batch recorded
2,878 passing test/subtest events before a300-second runtime harness timeout;
remaining integrated store/history/auditor/build checks did not execute. No full
regression or schema46 runtime pass is claimed. The ordinary managed-member
failure and M441/M442's runtime limitations remain. Prepared VM12 wrappers were
never invoked; do not resume them without revalidating prerequisites and paths.

At the user's request,382 exact superseded generated files were removed from the
two task evidence/lab roots:378.94→30.38 GiB reported allocation. Keep M352's
passing disk, M442's trace disk and M443/M446's ordinary-failure disk, their required
inputs/base, shared cache, historical fixtures and compact source/results. Their
inode/size/mtime records remain unchanged. Pruned bundles carry explicit tombstones
and are no longer runnable archives. Exact plan, removal log and results live at
`~/Documents/ChatGPT/omabit-qualification-evidence/2026-10-26-storage-review/`.
This supersedes prior instructions/notes implying indefinite retention of every
full pier, disk or generated bundle; historical test outcomes are unchanged, but
old full-data availability claims must be read against this pruning inventory.

Adopt40 GiB maximum retained experiment data across those two roots, including
shared caches; one active runtime experiment with a separately declared scratch
budget, and at most three retained diagnostic/reference VM disks by default.
Measure before/after, share immutable inputs, and prune superseded payloads while
keeping compact provenance. This is a recorded operating rule; launcher-level
automatic enforcement is not implemented. No new experiments while paused.

The next user-facing deliverable after resumption should be a separately labeled,
size-bounded manual-review build, following narrow completion of current checks,
not further unchecked backend expansion. See
[the review, architecture, counts and manual steps](development-review-2026-10-26.md).
Protected ships, tmux,`~/temp-ur`, installed plugins and production services remain
excluded.


## M448 — Corrected archive dependency claim; manual review VM authorized (2026-10-26)

While preparing the user's requested labeled manual-review VM, strict resolution
of the old shared base revealed a cleanup error. `/tmp/omabit-import-appliance-20261024-42/artifacts/base.raw`
was a symlink chain into `omabit-qualification-evidence/2026-10-25-private-docker-pristine-input/base.raw`.
M447's exact-file cleanup deleted that target despite preserving the alias. The
three retained qcow2 overlays are therefore not currently bootable complete
archives. Their reports, source records and overlay bytes remain. No intact
retained raw-base copy was found in the inventoried retained roots, and no recovery
is claimed. This supersedes M447 and the review's claim that all required shared
bases/inputs were retained. The user was informed immediately upon discovery.

Before future retention/deletion, resolve every symlink and qcow backing dependency
to actual filesystem objects and verify the complete retained dependency closure.
A preserved alias or top-level directory is not proof of preserved data. Prefer a
self-contained image for the human review deliverable. Do not delete the remaining
overlays incidentally or reuse their obsolete launch instructions.

The user explicitly authorized a neatly prepared, labeled manual-review VM and
step-by-step desktop/process walkthrough. This is the active requested deliverable;
the broader automatic roadmap remains paused. A fresh self-contained16 GiB sparse
raw disk (no backing files) is prepared at
`~/Documents/ChatGPT/omabit-private-docker-labs/manual-review-20261026-01/`.
Its copied Omabit/Theme/DMs plugins have `.review` IDs and REVIEW labels. Sourcebox
has no accessible `/dev/kvm`, so QEMU uses TCG software emulation. A single new user
unit `omabit-manual-review-20261026.service` runs the guest with10 GiB RAM, a12 GiB
no-swap process limit, three host CPUs maximum, no guest NIC and no host shares.
First-boot package/runner setup is in progress; desktop readiness and manual
walkthrough acceptance are not yet claimed. The localhost noVNC viewer connected
successfully. Preparation uses396 signed packages, mostly existing public cache;
no host installation/configuration change. After deleting the verified staging
copy, the review directory occupies4.28 GiB and the two retained roots total34.68
GiB, below the40 GiB budget. Continue this one disk, not another test-run ladder.


## M449 — Labeled manual VM ready; same-disk restart verified (2026-10-26)

The user-requested manual handoff is ready in the single M448 directory
`~/Documents/ChatGPT/omabit-private-docker-labs/manual-review-20261026-01/`.
This supersedes M448's setup-in-progress/readiness status only. The broader goal
remains paused, and M448's historical archive-base correction remains in force.
Do not start another qualification run or clone this disk as incidental work.

The actual Omarchy/Hyprland desktop runs copied Omabit, Theme and DMs with review
IDs/labels. Omabit displays `0.2.0-review.20261026 / UNQUALIFIED DEVELOPMENT`.
F2 opens the guest terminal; F3 the local walkthrough; F4 Omabit; F6 maximizes;
F7 the full DMs review panel. Theme and DMs own sign-in panels were visually
accessible; application authentication/consent is left for the user's manual pass.
The guide includes architecture, expected outcomes, scope limits, per-section
notes/status and guest-local feedback export. Host README includes start/status/
stop, viewer credentials and optional SSH forwarding. No public viewer binding.

A real guest-managed ~zod is healthy. A separately booted ~nec was cleanly stopped
and retained at `/home/review/import-source/unclassified-pier` for manual import.
Clean preview identified ~nec and destination `fakeships/nec/nec`; adding the one
intentional `review-link` symlink yielded the precise unsupported-symbolic-link
error. Leave this fixture unimported for the user. The pinned core pill is NOT a
prepared %groups fixture: message delivery, theme exchange and real backend
installation are deferred, as are Ethereum devnet startup and public profile
rollout. Approval alone is not evidence of backend data correctness.

Clean stop/restart passed (QEMU exit 0). Omarchy's guest power-button-ignore default
initially made host stop ineffective; the guest-only 90-omabit-review logind drop-in
now makes that virtual button power off. The same runner identity, ~zod identity,
pier path and config revision survived; ~zod resumed healthy automatically.
The guide, marker and intentional import link survived. Host evidence is
`shutdown-check.json`, `restart-check.json`, `storage-check.json`, and
`review-healthy.png`; no full-copy checkpoint was made. Browser restore-pages notices
can occur on OS shutdown; close guest Chromium after saving notes to avoid them.

Sourcebox has neither KVM nor a GPU. The actual desktop required a guest-only
Aquamarine 0.15.1 patch selecting its existing GBM allocator renderer overload at
two DRM creation sites. Patch, upstream source hash/archive and library hash are
retained beside the disk. QT Quick uses RHI with software Mesa; this renders the
bar icons. Earlier nested Weston/Cage approaches were unsuccessful and are not
the active desktop. No host graphics packages/configuration were installed or
changed. Cold VM/ship startup took roughly five minutes; do not mistake TCG timing
for native product performance. Hyprland configerrors was empty.

At handoff the review directory allocated 5.36 GiB; the two retained roots totaled
35.76 GiB, below 40 GiB. One self-contained 16 GiB sparse raw disk, no backing files,
no guest NIC and no host shares. The controller checks the 40 GiB total at launch
and every minute and requests shutdown on overflow; logs have a 64 MiB cap check
and one previous generation. This supersedes M447's unenforced-launcher statement
for THIS manual controller only, not for all old harnesses. Keep existing roots,
cache and source; do not revive pruned-bundle launchers.

Automatic review rejected an attempted unnecessary plaintext login-code file
before execution. No credential file was created; the guide uses the existing
Copy +code control, and no credentials belong in evidence or feedback. Production
~lismyl-lismyl, tmux ships, ~/temp-ur, installed plugins and host Docker remain
outside this work. See [manual instructions](manual-review-vm-2026-10-26.md).


## M450 — Portable fresh-onboarding review, one persistent session per host (2026-10-26)

The user asked to run the manual review locally on any Omarchy machine, especially
sourceport while away from sourcebox, and confirmed WiFi/tailnet SSH access. This
explicitly authorizes the additional portable copy for that deliverable; it does
not resume the broad qualification roadmap or permit another full-copy test ladder.
M449's populated review VM is preserved separately and stopped. The new handoff is
`manual-review-20261026-01/portable/`; source lives in `scripts/manual-review/`.
See [portable instructions](portable-onboarding-vm-2026-10-26.md).

A pull script uses the user's normal SSH alias and rsync, resumes interrupted
transfers with an identity marker, validates all immutable files/symlinks, excludes
the source verification overlay, and refuses to replace an existing review session's
base. Each machine has one relative-backed persistent overlay over a self-contained
compressed base; keep the directory and backing dependency together. No reset/prune
command. User systemd bounds memory, CPU, swap and logs; the launcher monitors 12 GiB
per portable bundle, or the combined 40 GiB retained-root budget on sourcebox. The
original protected piers, tmux, ~/temp-ur, installed plugins and host Docker/services
remain excluded. No host installation or desktop configuration was changed.

“Fresh” means no Omabit owner/runner/enrollment/managed ship/application session.
Omarchy, native prerequisites and review-labeled development plugins are already
installed. This is not proof of cold native-package installation or published-plugin
acquisition. Preserve OS-assigned subordinate IDs: clearing those along with Omabit
state made fresh setup unavailable on the guest's NSS configuration. Restoring only
the disposable guest's collision-checked OS delegation fixed readiness. The sealed
base retains that prerequisite, but no owner config or runner. No user setup was
pre-run. The core fake pill is not a prepared %groups fixture; full messaging/theme
and public-publisher backend installation remain explicitly unavailable here.

This supersedes M448/M449's blanket “no accessible KVM” statement. A fresh sourcebox
probe opened /dev/kvm, returned API 12, and the portable guest actually ran with KVM.
Availability must be probed per host; the launcher falls back to TCG without changing
permissions/modules. Preserve prior TCG timing as historical only. QEMU's sandbox
also needs read-only /etc/resolv.conf for its user-network DNS proxy. Guest private,
LAN and tailnet egress are blocked with a DNS-only exception; public HTTPS downloads
work. No forwarded guest ports or host home/control sockets. This trusted-review
network policy is not a hostile-guest-root security claim.

Validation: six small contract tests, image check, sealed-input verification, real
fresh-overlay boot, can_setup=true / ready=false / host_present=false, exact pinned
rootless helper HTTPS hash, registry HTTPS 401, empty fleet/unchecked setup consent,
and clean shutdown. A same-overlay restart retained the corrected OS prerequisite.
One preparation overlay was consolidated into its stopped base with a removal record,
then removed only after commit; a new verification overlay was booted and preserved.
No full per-test disk copies. Sourceport was unreachable during preparation; its
specific hardware launch remains unverified. F6 was intercepted in the browser
check, so the guide includes closing the walkthrough for full-width Omabit and F3
to reopen it. Setup navigation was initially rejected by automatic approval review;
reading its handler confirmed navigation only, and a freshly grounded click succeeded.
No actual setup consent was submitted.

Final portable allocation: 2.05 GiB; sealed logical transfer inputs: 2.31 GiB. The two retained roots total 37.68 GiB, below 40 GiB. Compact proof: `portable-check.json`, `portable-preparation.json`, and `portable-onboarding.png`. Both review VMs are stopped at handoff.


## M451 — Human review opened on sourcebox (2026-10-27)

The user returned to sourcebox and requested the review desktop. Started the existing
M450 portable session with KVM, connected its localhost viewer, and left the first-use
walkthrough visible for the human. No reset, copied disk, setup approval or ship
operation was performed. Unit: `omabit-review-099521816333.service`; disk remains
`manual-review-20261026-01/portable/state/session.qcow2`. Treat this overlay and its
base as human review progress from now on; do not consolidate, reset or prune it as
agent-only scratch. The original populated review VM remains stopped. Launch
preflight measured 37.68 GiB of the 40 GiB retained-root budget. M450's “both VMs
stopped at handoff” is superseded for this active human session only. The broader
automatic qualification roadmap remains paused.


## M452 — Human review redirects the resumed goal (2026-10-27)

Status: active product decisions; implementation acceptance pending. Supersedes
M451's paused-roadmap/active-VM status: the user resumed work; their review VM is
now cleanly stopped and preserved, including its base. Its human-added live ship
credentials prohibit reusing that overlay as fresh-test input. Use the clean base.

[Original feedback](diagnostics/2026-10-27-human-review.json) records successful
setup/first boot and failures in project fake networking, Theme sign-in removal,
and restart resource admission. The user separately requests compact fleet-backed
Application Connections with plugin-to-Omabit navigation, automatic backend-only
managed authentication, and hosted enrollment in its own Setup pane. Prefer the
existing explicit application-connection declaration to a plugin-name heuristic.
Per-plugin Eyre sessions and direct plugin-to-ship data flow remain required.

New and initialized ships need distinct configurable minimum/recommended resources;
recommendations are advice, not hard admission. Keep maintenance guards and late
capacity rechecks. Same-project fake networking remains in scope and must appear
in ordinary creation/import, not only a qualification CLI. See the
[acceptance list](review-pass-2026-10-27.md). Storage is 38.48 GiB / 40 GiB before
this pass; no full repeated test bundles. This memory records decisions and
observations, not claims that the requested changes already work.


## M453 — Review fixes, source evidence and cache retirement (2026-10-27)

Status: implemented source with focused tests; fresh-VM acceptance pending.
M452 remains the product direction. Resource start admission now reads an additive
owner-private policy: new minimum 5120 MiB RAM / 8192 MiB disk, recommended
8192 / 16384; initialized minimum 1024 / 2048, recommended 4096 / 8192.
`urbit-runner resource-policy show|set --phase new|existing` configures all four
values. Recommendations never gate starts. Existing maintenance/copy guards
remain separate. Tests cover late capacity loss, initialized fake/comet restart,
invalid-policy Stop, persistence, concurrent phase edits and CLI validation.
See [the CLI contract](resource-policy.md). These replace hardcoded start thresholds
only; no production policy or service was changed.

Application Connections now has one ship selector and compact declared-plugin
rows. Hosted enrollment/login moved to Remotely Hosted Urbits. Fleet candidates
use observed endpoints and explicit runner browser-host preferences; a remote
loopback address is not guessed as local. The new managed connection CLI verifies
runner, instance revision, container and HTTP identity, reuses a valid parent
session or obtains code privately, then uses the existing independent-session
handoff. No code/cookie enters QML/argv/stdout. Private mock-runner/HTTP tests cover
reuse, expiry, stale pins and wrong HTTP identity. Theme/DMs have profile-specific
Connect via Omabit navigation and retain direct login. Actual shell navigation
and live managed handoff remain VM gates. Narrow 440px offscreen QML flow passes
explicit approval, separate hosted login and interrupted disconnect recovery.

The Theme removal failure is reproduced with a revoked session and an abandoned
watch: channel DELETE receives an authentication failure after broker logout.
Eyre's local source `pkg/arvo/sys/vane/eyre.hoon` +close-session closes associated
channels. Theme now confirms session-only logout on that specific failure and
records its 303/401 receipt separately from 200/204 channel-deletion receipts;
original poke uncertainty is unchanged. Network failure preserves retry state.
30 connection/recovery and 30 transport/journal tests passed; 85 Theme backend
and 26 DMs connection/recovery tests passed after endpoint-policy updates.
Older retained receiver artifacts do not acquire this fix automatically. The
human-reviewed overlay remains preserved, not migrated or used as a fixture.

Explicit HTTP origins now support LAN/VPN/public hostnames and unicast addresses
in collector and receiver validation. This supersedes the earlier loopback-only
HTTP restriction; HTTPS verification, exact origin/identity checks, no proxy and
no redirect behavior remain. HTTP offers no transport encryption; use HTTPS on
untrusted paths. No host or tailnet ship was contacted in these tests.

Removed only the obsolete `2026-10-25-private-docker-code-cache/omabit-go-build`
cache: 19,671 reproducible files, 6,193,258,496 allocated bytes. Its adjacent
BUILD-CACHE-REMOVAL-2026-10-27.json records the classified metadata inventory and
completion. All reviewed disks/bases and compact evidence remain. The retained
roots are now 35,114,692,608 bytes (32.70 GiB), leaving 7.30 GiB under the 40 GiB
cap. The next single fresh VM is budgeted at at most 6 GiB additional allocation.
Do not reinstate per-test cache/disk copies.

## M454 — Fresh guest acceptance and guided project creation (2026-10-27)

Status: managed authentication/restart and narrow layout passed in a new disposable
VM; guided project creation is implemented in source, awaiting the same VM's
runtime check. Extends M453 without claiming project/import integration complete.

`manual-review-20261027-01/portable` uses a hardlink to the verified immutable,
credential-free 20261026 base, never the human-reviewed overlay. **Never commit
this experiment's overlay into that shared base.** Seal any successor by converting
to a new base file. Only this VM is active (unit omabit-review-2600be0b4f9c), with
10 GiB RAM, KVM, the existing internet-only egress boundaries and a temporary
read-only source artifact disk. It is agent acceptance state, not yet the fresh
human handoff. All production exclusions remain in force.

Both Theme and DMs entry buttons open Application Connections with their requested
row highlighted. Setup cards wrap at the narrow tiled desktop width. The new
managed auth CLI connected both labeled plugins to fresh `pass-zod / ~zod` with
independent sessions and no code field. Theme's one-click Disconnect completed
server logout, local cleanup and archival; DMs remained connected. The updated
Theme popup now also closes after navigation (source polish pending deployment).
A single URL appears above compact approval rows. This core-pill fixture does not
prove %groups application data or full Theme/DMs subscription behavior.

The initialized fake restarted successfully with **new minimum RAM 16384 MiB**,
**existing minimum 1024 MiB**, and **existing recommendation 16384 MiB** inside the
10 GiB guest. The new-ship threshold and recommendation did not gate restart.
Guest test values were restored to defaults afterward. Source and the small guest
JSON records preserve the result; no full per-test VM copies were created.

The new `project-create` coordinator uses existing runner reservation/preparation/
launch/readiness operations. It accepts 2–16 complete fresh fake memberships,
emits bounded JSON progress, keeps deterministic per-step request IDs, stops on
ambiguous/failed effects, and provides explicit same-attempt resume. Tests cover
lost replies, interrupted jobs, cancellation and unrelated history. First readiness
now waits up to three minutes using read-only observations before collecting its
normal fresh proof. Source UI links ordinary fake creation to a project form.
Joining/importing existing piers and ordinary fleet integration remain unfinished;
a folder change is never represented as joining a network. The compact project
history and application panels pass offscreen QML checks after these changes.

Storage after the fresh guest's first boot was 36,824,162,304 allocated bytes
(34.29 GiB), within the 40 GiB cap and the declared 6 GiB incremental budget.
Continue reusing this one guest and immutable inputs; retain compact diagnostics.

## M455 — Human-path project creation exposed an initial-anchor failure (2026-10-27)

Status: unresolved runtime defect under investigation; supersedes any implication
that M454's new guided project form has passed live creation. No project member
was booted in this attempt. Existing managed auth and restart results still stand.

The actual narrow QML form reserved `review-net` (~zod/~nec, HTTP18381/18382) in
manual-review-20261027-01. Preparation succeeded; initial anchor Start received its
acknowledgement and saved phase `started`, then its operation became interrupted.
The anchor was running. A later guarded read-only invocation of the same runtime
adapter passed ownership, ports, bridge, boot-bound namespace and inventory without
restarting it. This does not establish why the initial observation failed and does
not repair or qualify that interrupted receipt. Preserve the original operation.
Compact evidence is `docs/diagnostics/2026-10-27-initial-anchor-review-failure.json`
(3.8 KB), rather than another full disk copy. The same acceptance VM was then
cleanly shut down and restarted to activate bounded internal diagnostics.

A temporary binary transfer over serial initially stalled. Its exact loader was
interrupted and the console restored; a raw transfer with an adequate transport
deadline later succeeded and its SHA256 matched. USB hotplug is not useful in this
minimal guest because usb-storage/sd_mod modules are absent. A verified read-only
QMP node switch was technically refused by QEMU and did not change the block graph.
Do not repeat these experiments for routine deployment: update the existing source
artifact disk while this VM is cleanly stopped. Temporary diagnostic tooling is
not a new supported product interface or a reason to duplicate the VM.

The pending human walkthrough now covers automatic fleet selection, separate
Theme/DMs rows, both plugin entry points, disconnect completion, hosted-ship pane,
project creation and new/existing resource thresholds. Its versioned feedback key
keeps this pass separate. It still needs deployment and visual review in the fresh,
credential-free human handoff image; the populated acceptance overlay is not that
image. The history resume UI and targeted coordinator tests pass; this is source
validation, not live project qualification.


## M456 — Guided pair messaging and persistent recovery passed (2026-10-27)

Status: bounded live acceptance passed for the second fresh project in the same
isolated review VM. M455's original anchor interruption remains unresolved; a
successful second creation does not explain or repair that first receipt.

`review-pair` launched fresh ~nec/~zod through the public project-create workflow.
Its first readiness observer exited before HTTP was ready; explicit read-only
checks later succeeded without repeating Start. The observer-open step was outside
the existing wait loop. Source now retries both observer-open and proof collection
read-only within a ten-minute deadline, with cancellation/permanent-refusal tests.
This supersedes M454's three-minute wait description; this fix still needs deployment
and a first-project acceptance run.

Both peer directions and local controls received Gall acknowledgements; a deliberate
negative poke was rejected. Public stop/observe-exit/retire/resume/observe-resumed
operations then replaced only ~nec's container while preserving its pier inode.
Bidirectional messages passed afterward. The tiny application fixture retained its
state and queued update, resumed the same subscription/cursor, and reconciled a
fresh subscription. No writes were replayed. Both test channels were deleted and
both test sessions were revoked. Compact evidence is
`docs/diagnostics/2026-10-27-review-pair-recovery.json`; no full disk copy was made.
An initial test-only guard failure occurred before credentials or effects because
project-wide operations omit instance_id; its original result is preserved.

Fresh-first project onboarding still needs explicit preparation of the pinned
image/pill: this VM already had them from its ordinary fake boot. Normal fleet
projection, project controls and fresh human handoff remain pending.


## M457 — Explicit project inputs and clean successor preparation (2026-10-27)

Status: source tests pass; first-project live acceptance pending. The guided flow
now performs an explicit consented dependency step before the first project effect.
It verifies/reuses the pinned Vere image and core pill, downloads absent inputs,
and refuses mismatched cache contents unchanged. It does not create a ship or
network. Operator/capability/runner pins, the resource gate and update fence apply;
shutdown cancellation is checked directly as well as through the child context.
Download failure retains the reservation for explicit same-intent continuation.
Resume after project effects does not repeat dependency preparation. The command
budget is now 25 minutes; source/CLI/UI descriptions supersede the prior 15-minute
guided budget. The 10-minute initial readiness check remains read-only.

Focused and race tests passed for dependency consent, pins, cache reuse, refusal,
cancellation and same-intent continuation. Project fresh Create/Start and initialized
resume Create/Start also now use the configured new/existing resource policy, with
late rechecks. Adapter tests prove high new minima do not block modest initialized
restarts and late capacity loss prevents dispatch. No production settings changed.

Retired only the completed agent-only acceptance overlay and unused read-only
diagnostic disk after exporting useful records. Manifest:
`manual-review-20261027-01/AGENT-ACCEPTANCE-DISK-REMOVAL-20261027.json`; reclaimed
3,264,192,512 allocated bytes. This supersedes M454's reference to retaining that
populated agent overlay. The original human-reviewed disks and their shared base
remain preserved. A new small overlay from the same clean base now contains updated
.review artifacts and the walkthrough, without native setup or enrolled ships.
The guest login creates empty keyring collections automatically: verify their item
counts, not absence of the login.keyring file, before sealing. Both counts were zero.

Source transfer finished despite its console reader timing out; its marker, hashes
and subsequent shell reload verified completion. Do not repeat that mutation. The
temporary console helper now uses an exclusive lock so a second reader cannot
consume another command's output. The clean guest terminal no longer includes a
nonexistent theme file. Sealing and first-project acceptance remain the next gates.

## M458 — Clean review seal and fresh-first project failure (2026-10-27)

Status: clean image verified; project-first acceptance still failed and is under
investigation. M457's sealing step is complete. The new standalone review base is
SHA256 `d2c259d4effc62c2b2e54613a934b13e4b3f2fce0d5337556079efcdf175af33`.
It contains the updated .review plugins and walkthrough, zero enrollments, no
runner installation and two empty Secret Service collections. The earlier human
review base and credential-bearing overlay remain unchanged. The seal manifest
is `manual-review-20261027-01/CLEAN-REVIEW-SEAL-20261027.json`.

An acceptance overlay from that base passed native setup and began with zero
ordinary ships, projects or Docker images. Explicit project dependency preparation
succeeded, as did network/anchor preparation and the first member (~nec) launch.
The ~zod launch was interrupted before its launch journal was initialized. A later
read-only invocation of the exact adapter passed. This does not prove the original
cause or authorize replay. Preserve the original failure in
`diagnostics/2026-10-27-first-project-launch-failure.json` (8.5 KB); do not claim a
fresh-project pass. Added bounded launch-failure diagnostics for the next controlled
attempt. Focused launch tests and actual concurrent download/update/resource-fence
race tests pass. No host service, plugin, protected pier or credential was accessed.

The read-only diagnostic transfer unnecessarily repeated M455's known refused
PCI hotplug and block-node switch approaches; both were refused and the root disk
was unchanged. The subsequent exclusive serial transfer verified SHA256 and worked,
but is slow. For future routine deployment, stop the sole test VM and replace its
existing read-only artifacts disk. Do not repeat unsupported hotplug experiments.
M456's messaging/persistent recovery pass remains valid for its earlier pair; it
is not evidence that this fresh-first onboarding attempt passed.

## M459 — Project fleet navigation and diagnostic narrowing (2026-10-27)

Status: fleet navigation source-qualified; live first-project creation still
unqualified. Collector polling now independently reads identity-checked saved
project reservations, without allowing a failed project read to hide ordinary
fleet observations. Project members appear under named project groups and open
the matching project/history view. Their reservations are not treated as live
health, code retrieval or application-login authority. Ordinary instance mutation
buttons remain unavailable for project members. Reserved project HTTP ports also
participate in local form suggestions. Go race, actual Model.js regressions and
networkless offscreen project-navigation tests pass. Live UI deployment is next.

A second fresh-first attempt reproduced a second-member interruption, now narrowed
to storage/artifact preflight after the workspace was created and before container
creation. The pure controller discarded the underlying driver error; added bounded
internal stage diagnostics rather than guessing a fix or repeating the interrupted
action. Its original compact evidence is
`diagnostics/2026-10-27-first-project-preflight-failure.json`. M458's earlier nil
launch journal and this workspace-created journal are distinct observations, not
proof of the same cause. The second VM is retained for the next diagnostic update;
no extra full copy was created. The first attempt's classified disk was retired
with `FIRST-PROJECT-DISK-REMOVAL-20261027.json` after compact evidence export and
clean shutdown, reclaiming 1,284,816,896 allocated bytes. This supersedes retaining
that first attempt's generated disk; its original failure evidence remains.

## M460 — Engine probe latency and reactive project navigation (2026-10-27)

Status: focused source checks pass; successor first-use VM acceptance pending.
The diagnostic pair retained M459's failure and reached both acknowledged Starts,
but its final engine read interrupted the second launch. The driver log identifies
`private engine API evidence unavailable`; Docker recorded canceled `/version`
requests while containerd was busy. Capacity sampling stayed above 7.27 GB RAM
and 10.28 GB disk. Later read-only version/info samples were valid and fast; these
are not evidence about latency at the original failure. Exact compact evidence:
`diagnostics/2026-10-27-project-engine-probe-failure.json` (15.8 KB). Do not retry
Start or relabel the interrupted operation as complete. This narrows the latest
failure; it does not prove M458's earlier failure had the same cause.

The probe had a separate two-second response-header deadline inside a five-second
read budget. It now uses one 15-second total budget, bounded by any shorter caller
deadline. Socket/peer ownership, engine/API/rootless/cgroup/data-root checks are
unchanged. Fixed timeout/cancellation identities survive diagnostic sanitization.
An actual Unix-socket delayed-header test passes, as do bounded-header/body deadline,
unsafe response, peer-boundary and runtime-admission regressions. Runtime acceptance
must still establish whether this resolves first-project onboarding under boot load.

The actual VM sidebar exposed a QML update defect that Node/model and project-pane
tests did not cover. The new actual-monitor fixture reproduced a stale one-group
Repeater while the current model function returned two groups. Computing the group
list before QML converts delegate data into model roles makes the project row
appear with positive geometry. Both fleet views now use that pattern. A project
selected by the generic monitor launcher routes into its project controls rather
than displaying disabled ordinary-ship controls. Full lifecycle fixture and live
VM recheck follow; M459's source-only navigation claim was insufficient UI evidence.

## M461 — Fresh-first project onboarding passes (2026-10-28)

Status: isolated native first-use path passed. M460's source changes are now in
clean review base SHA256
`e778801a4d1366004bbb0700b3920c2bc9cf4d1b0c07137357adfbabb1b7ec32`.
`CLEAN-REVIEW-SEAL-PASS2-20261027.json` records zero enrollments and two empty
keyring collections. It supersedes the generated d2c259... review base; the
original human base/credential-bearing overlay remain unchanged. The retired
second-failure/diagnostic disk is recorded in
`ENGINE-PROBE-DISK-REMOVAL-20261027.json`, reclaiming 3,310,686,208 allocated bytes.
Original failure records remain; none was rewritten as a success.

A fresh overlay passed native setup and began with zero ordinary ships, projects
and Docker images. One `project-create` downloaded/verified dependencies, prepared
the shared namespace and storage, launched ~nec and ~zod, then passed both identity
and initialized-pier readiness checks. Exit 0 in 347.97 seconds; no action replay.
Both public HTTP identity reads then passed. Compact evidence:
`diagnostics/2026-10-28-fresh-first-project-pass.json` (10.5 KB).
Runner `runner-0ff8830d4889dcc4921646d4b05b1617`, project
`project-fdacc1d66c60db37d69cc1e1dc63e6f6`, instance names first-project-nec/zod,
HTTP ports 18482/18481. Keep this pair for subsequent auth/lifecycle integration;
do not create another full acceptance environment without a new reason/budget.

The actual guest sidebar now shows both project members at narrow width, and the
generic monitor launcher routes to Project networks. Screenshot:
`/tmp/omabit-review-pass-20261027/project-fleet-live.png`. The real panel/monitor
profile/drain fixture also passed after the fix. One cosmetic issue remains for
final handoff: Chromium's clean base restores a crash bubble. Close the guest
browser normally before sealing the final setup-only seed; do not touch the host
browser or the preserved human-review profile.

Startup logs include a %groups desk in this pinned pill; this corrects any inference
that these fresh peers necessarily lack that backend. Presence/download messages
are not proof of app readiness. Verify actual Theme/DMs behavior before claiming
application acceptance. The remaining feedback gaps are project-member assisted
login, straightforward lifecycle controls, and existing-fake project import.

## M462 — Second human review handoff and pause (2026-10-28)

Status: goal paused at the user's request; one fresh manual VM remains running.
This supersedes M461's instruction to retain its automated pair for auth work.
The user requested a fresh review now. The agent pair shut down cleanly, and its
2,146,242,560-byte allocated overlay was retired only after the passed compact
M461 evidence and console logs were preserved. Exact identity/hash/removal record:
`manual-review-20261027-01/ACCEPTED-PROJECT-DISK-REMOVAL-20261028.json` in the lab.
The original human review, its credentials/notes, and required base remain untouched.
Storage after handoff preparation: 37,371,564,032 bytes (34.81 GiB), below 40 GiB.
No full disk copy was made; the new overlay reuses M461's sealed clean base.

Current manual overlay: `manual-review-20261027-01/portable/state/session.qcow2`;
unit `omabit-review-2600be0b4f9c`; viewer localhost:6081. Label:
**OMABIT REVIEW 2 · OCT 28**. Build remains `20261027-feedback-pass-2` with
M461's exact runner/collector hashes. Fresh guest checks found no runner or
Docker daemon and two empty Secret Service collections. The guide has a separate
feedback key/export filename, changed-feature list, test order and explicit gaps.
The visible desktop is at Setup beside `/opt/omabit-review/START-HERE.html`.

Project-member assisted authentication was investigated but not implemented.
The just-started shared credential-helper extraction was withdrawn exactly; the
existing login-code and port-inventory tests pass. No incomplete auth change is
in this handoff. Ordinary fleet approval, compact rows, hosted setup, Theme cleanup,
resource policies and fresh project creation are in the review. Project assisted
login/code retrieval, simple lifecycle controls and existing-fake project import
remain unfinished. The guide tests ordinary ship auth first and projects last;
it does not ask the reviewer to bypass missing project controls with raw Docker.

Do not resume automatic experiments or operate on this new human-review disk
until the user requests it. After human sign-in it may contain real credentials;
treat it like the original human-review disk, never as an automated-test seed.

## M463 — Native main-desktop DEV review replaces the VM workflow (2026-10-28)

Status: authorized by the user's latest review feedback and installed on sourcebox.
Supersedes M462's running-VM handoff and the VM/private-compositor-only requirement
for this manual review. All production exclusions remain: no operations on
`~lismyl-lismyl`, tmux ships or `~/temp-ur`; no production plugin/session/service
replacement. The user explicitly accepted reusing host dependencies with separate
plugins/binaries and experimental storage under `/srv/dev/urbit`.

Installed three `.dev` plugin IDs, separate registry/XDG paths, Secret Service
profiles, rootless engine and runner units. `--development-root /srv/dev/urbit`
requires a separate trusted root-owned owner record; validates the actual Unix
identity before substituting its private runtime home; selects the DEV Docker
socket; excludes production SSH grants and installer/update commands. Host-report
probing also required a socket fix; merely changing HOME was insufficient.
The native CLI wrappers pin the DEV registry. Production service PIDs/start times
remained 2166/3050 from October 21 before/after setup. The second review VM exited
QEMU 0 with `forced:false`; its human-review disk is retained. DEV is approximately
55 MiB, with no new live ship booted. Details: [native review](native-dev-review.md).

Validation: ownerhost/runtimequalify/runner tests passed; managed host-compatibility
checks passed; real DEV inventory has one fresh runner, no ships, the DEV pier
root, separate engine ID and compatible rootless/systemd cgroup evidence. This is
not a new full-stack ship/restart acceptance claim. The native unit is enabled but
does not automatically resume ships. Do not silently add that behavior in a review.

## M464 — Compact ship onboarding and incremental-project gap (2026-10-28)

Status: empty-fleet CTA and compact recommended comet path implemented and installed.
Actual native popup visually fits without vertical scrolling; an inert QML test
checks the submitted name/port/consent/pill/loom and custom-setting toggle. A single
runner is implicit only in the compact popup; full-window detail remains. The
existing real Service/Panel/Monitor drain fixture also passed.

Fake creation now has None/New Project +/saved-project choices in the ordinary
form. Bulk creation has been removed from the GUI. New Project + coordinates one
member; focused Go tests and QML submission pass. Existing-project joins remain
explicitly refused. The immutable complete-roster plan, anchor publications and
historical receipts need a deliberate additive-membership extension; do not edit
stored JSON to bypass their identity checks. This replaces the bulk project GUI
as the desired UX, but does not claim the new join feature is finished.

Native loading exposed cached QML at reused URLs. The final UI uses a distinct
entry-point directory, with actual IPC and screenshot verification. Omabit now
waits for its manifest before registering service IPC, avoiding a transient
production target during DEV construction. Namespaces isolate state/selection,
not malicious same-UID code: this remains relevant to wallet/key-storage research.


## M465 — Native key input and independent sign-in windows (2026-10-28)

Status: implemented and installed in the DEV review. Permanent boot supports
masked paste/stdin and file input; the explicit file workflow securely chmods the
verified owned regular descriptor to 0600. The reported keyfile was 0644; no key
contents were printed and no real key/identity boot was attempted. This replaces
the generic private-file error as the only UI guidance. Browse uses external
Zenity, isolating chooser failure from the shared Quickshell process; no original
shell core was available, so its precise crash cause remains unproved.
Theme/DMs now match Sign In → assisted/direct choices. Source and actual isolated
QML/helper tests passed. Final installed labels/profile paths stay separate.

## M466 — Incremental project membership and live fleet (2026-10-28)

Status: source-qualified and native DEV tested. Supersedes M464's refused join
and M462's missing project authentication/simple controls. Schema47 retains the
original canonical plan, anchor and all lifecycle history. A separate expansion
ledger/claim table records additional galaxies, stable internal HTTP ports,
container creators and exact RootlessKit loopback-publication receipts. Empty
plans create networks without a ship. Same fake identity in different projects
is allowed; duplicate galaxies within one namespace remain refused.

RootlessKit's builtin driver can forward a new host-loopback port to the already
inspected anchor's private bridge IP without replacing/restarting the anchor.
The API directory may be 0755 under a private owned runtime directory; requiring
0700 on that child incorrectly refused the first live attempt before dispatch.
The corrected check retains ownership, type, non-writability and private-parent
checks. Explicit member-continue only permits an exact interrupted reserved
attempt before any storage/container effect. Unknown effects remain interrupted.
Old lifecycle/recovery commands are fenced after expansion to preserve history.
Expanded anchor/engine loss recovery is unfinished, not silently inherited from
fixed-roster qualification.

Project fleet rows now use fresh exact runtime observations and identity checks;
project grouping no longer substitutes placeholders for running ships. Browser,
code, logs and clean start/stop work. Project storage claims participate in
ordinary allocation/import fences and catalogue enrollment. Metrics/maintenance
are still not integrated. The form suppresses only its own pending reservation's
false duplicate message. Regression tests cover join, replay, original history,
reservation corruption and pre-dispatch continuation versus unknown dispatch.

## M467 — Native DEV review acceptance and bounded retention (2026-10-28)

Status: ready for the user's next manual pass; broader roadmap is not complete.
Build label 0.2.0-dev.20261028.2, Omabit UI feedback-3, Theme/DMs UI feedback-2.
Existing demo-zod (HTTP18080) and its anchor retained exact container IDs, PIDs and
start times. Added demo-nec (HTTP18082), stopped/restarted that new member from
its same pier, and received matching remote Gall acknowledgements in both
directions. Both project code/log read paths passed without retaining secrets.
Temporary test Eyre sessions were logged out. review-empty demonstrates a network
with no ships. Production process baselines2166/3050 remained October21; tmux and
~/temp-ur were excluded. No VM or full evidence bundle was generated.

A preliminary invalid Lens `|hi` evaluator probe left its own job pending, making
later reads fail500. Canceled exactly those two DEV test jobs and used the known
typed Khan/Gall acknowledgement probe instead; credential reads recovered. This
reinforces M226/M235: shell/dojo generators are not arbitrary Lens expressions.
Do not add automatic Lens cancellation to normal credential access.

Affected managed/runtimequalify/projectnetwork/protocol/client/collector suites
passed. Later focused continuation tests and actual QML checks passed (Theme20,
DMs6, native Service/Panel/Monitor fixture, compact/new/existing-project forms).
Review guide now includes a current roadmap implemented/open matrix and explicit
expanded-recovery/metrics limits. This supersedes prior native guide statements
that DEV was empty and joining existing projects was not implemented.


## M468 — Native review reliability and security follow-up (2026-10-28)

Status: active reliability pass; supersedes M467's broad clean-restart claim.
The user confirmed project fake networking with browser |hi and successful
independent Theme/DMs application connections. Preserve the auth architecture:
Omabit obtains a managed ship's code privately, creates independent origin-bound
plugin sessions, transfers through the declared private receiver, and leaves
application traffic directly between plugin and ship. Standalone login remains.
A later security/privacy audit must verify keyring locking/reboot, session
revocation, same-UID plugin access, retained metadata/logs, and upgrade boundaries;
working UX is not security qualification. No wallet security claim is made.

User authorizes action testing on the existing DEV project ships and exact review
moons ~lagbet-bisful-dozzod-sarlev and ~marrun-dalmeg-dozzod-sarlev. Parent ~sarlev,
production ~lismyl-lismyl, tmux ships and ~/temp-ur remain excluded. Reuse piers,
never resend personal DMs or erase unknown outbound outcomes. Log display should
preserve @p and +code; continue filtering boot keys, cookies and control escapes.

Observed failures: project metrics are missing, empty namespace observation
rejects its zero member list, and a consumed one-shot moon key's missing host
file blocks unrelated project restart mount preflight. demo-nec is retained at
retired-before-create, empty-zod at reserved-before-storage. Both user moon Stop
requests timed out and Docker exited137; no agent stop caused those failures.


## M469 — Native reliability pass and live recovery (2026-10-28)

Status: implemented and tested; supersedes M466/M467 metrics and narrow
continuation limits, and resolves M468's reported review failures.
Build 0.2.0-dev.20261028.3. Omabit/Theme generation reliability-4, DMs reliability-6.
Project fleet observations now expose fresh bounded in-memory metrics and release
the mutation lock before HTTP/stats reads; fresh inventory took about 2.2 seconds.
The exact `demo-nec` retired-before-create and `empty-zod` reserved attempts were
continued without pier replacement. Missing consumed boot-key files are accepted
only for the exact owned private one-shot key mount; arbitrary missing mounts
remain refused. The zero-member inspect case now allows first addition.

All three DEV fakes completed clean Stop/Start and code/log reads. Both review
moons completed same-pier restart and clean Stop, exit 0; both are left stopped.
Their previous forced shutdown required replay; lagbet took several minutes before
HTTP readiness. Ordinary Stop now signals and observes, never Docker's automatic
SIGKILL escalation. Fresh keyed shutdown uses the earlier qualified identity-bound
hood exit path; unit coverage includes wrong identity, lost response and timeout.
No new keyed identity was booted solely for a shutdown test. `%lens-cancel` from
M467 was that test's explicit cleanup, not normal log watching.

Theme/DMs each connected independently to demo-zod. Revocation overlapping its
restart was recovered using the same saved attempt, then local state reconciled.
DMs cleared its runtime account/channel while Theme stayed connected. Both test
attempts were ultimately archived, both plugins left signed out. Personal outbound
history remained byte-identical; no personal send was replayed. DMs preserves an
exact 401/403 denial as session invalidation, never inventing a successful deletion
receipt. Session replacement returns to login before adopting another account;
late old-session display events are ignored. M468's security/privacy audit remains
open; this functional evidence does not resolve its broader security questions.

Hosted UI now filters managed fleet enrollments and uses unique @p naming.
Application approval is a right-aligned toggle. New project labels are unique
case-insensitively; preexisting duplicate records remain and are distinguished by
ID suffix. Logs preserve @p/+code at the user's request; boot keys/cookies and
terminal control escapes remain filtered. Monitor selection has a loading state.

Full runtime/managed Go suites, relevant collector/client/collection regressions,
a fleet-lock race regression, actual Omabit QML, Theme20 checks, DMs152 Python and
7 QML checks pass. Bidirectional remote Gall acknowledgements passed after member
restarts. Expanded anchor/engine recovery and project maintenance/import remain
open; do not generalize member restart evidence to those contracts.

No VM or full evidence bundle was generated. Existing moon replay/snapshot growth
briefly raised the combined footprint to 40.65 GiB. Removed 1,138,491,392 allocated
bytes of classified historical compiler/download/dependency caches with a manifest;
kept every pier, VM backing chain, source/lockfile and the verified signal-probe
binary. Current allocation is about 39.59 GiB. Measure and make headroom before
another large run. Production baselines2166/3050 still date to October21; no tmux
or ~/temp-ur inspection/operation occurred. See the native review guide and compact
diagnostic receipts for exact results and retention limits.


## M470 — 0.3.0 release preparation, bounded reliability and plugin cohesion (2026-10-28)

Status: active preparation, not a release claim. User explicitly authorizes public
0.3.0 after remaining DEV reliability, matching Theme/DMs UI, bundled skills,
complete CLI documentation and non-destructive upgrade qualification. GitHub
confirms current stable v0.2.0. Worktree is now on release/0.3.0; master and published
pins remain untouched. Source manifests and loaded-QML literal version are 0.3.0.
A manifest-derived displayed version failed the existing native release test and
could mislabel an older loaded component; restoring a compiled literal fixes it.

Exact authorized review moon ~lagbet-bisful-dozzod-sarlev completed inspect-epochs,
pack, meld, roll, another inspection, same-pier start and clean stop. All seven
receipts succeeded and identity/configuration were preserved. Left stopped; no
history pruning or personal messages. Full Go suite and vet pass. CLI simulator
passes57 commands across6 tasks. A new networkless CLI audit checks130 help entries,
verifies no HOME writes and generates docs/cli for all three binaries. Fixed Omabit
help opening stores and collector prototype wording/missing project-add listing.

Theme/DMs now vendor identical small native header/sign-in-choice components,
440-unit popup width,720-unit height cap,12-unit section/8-unit control spacing,
matching native semantic colors/icons/actions and signed-out guidance. Paired
actual Qt renders compare dark/light/narrow/long connected identity states with
inert I/O; Theme sign-in render includes the secondary direct-credential action.
The developer skill records these conventions; both skills validate and packaging
now includes skills plus CLI docs in the binary bundle/flattened export, alongside
the separate provenance-bound skills archive. Five packaging tests pass. Native
DEV installation and broader upgrade gates are still pending at this checkpoint.

Supersedes M448's decision to retain its three unusable historical overlays: their
shared namespace base maps to the confirmed missing pristine-input/base.raw;
no QEMU or child overlay depended on them. Removed only those three obsolete test
root.qcow2 files, retaining SHA256, qemu metadata and per-directory tombstones.
Freed10,927,276,032 allocated bytes (10.18GiB); retained all manual-review disks and
DEV piers. See diagnostics/2026-10-28-release-retention.json. Existing production
PIDs2166/3050 still have their October21 start times. No tmux or ~/temp-ur access.

## M471 — Release gates and final reference styling (2026-10-29)

Status: active 0.3.0 preparation; publication still gated. Supersedes M470's
pending DEV styling note: generation ui-20261028-cohesion-7 is installed only in
Theme/DMs DEV. Paired actual Qt screenshots cover dark/light/narrow/long identity
popups and both sign-in dialogs. DMs now sizes sign-in to content, matching Theme.
Theme's separate installer now packages both shared presentation files and
recognizes its prior transport inventory;20 installer regressions pass. Styles
share only vendored source conventions, not a runtime dependency.

The full race gate exposed two issues. Numeric fixture IDs could coincidentally
appear in public digest/timestamp strings, so the host-inspection privacy test now
checks complete numeric tokens. Project validation repeatedly reread predecessor
journals. A memo now lasts only one synchronous read-only transaction traversal;
an ordinary DB starts its own read transaction, nested recovery views retain that
same scope, and fresh calls after writes always revalidate. Regression checks
cover mutation in the same transaction, rollback, invalid dependencies and cycles.
The focused original45→46/reopen case passed; final broad race tests remain running.
No schema, journal body, safety check or failure is removed by this optimization.

Actual published0.1.0/0.2.0 tools were downloaded and hash-verified as inputs.
A guarded offline guest harness is prepared for final artifact qualification,
using the retained pristine onboarding base read-only, no network or host runtime
mounts and a4GiB scratch cap. This is a release-machine test, not a return to VM
manual UX review. Protected ships, personal outbound journals and production
services remain excluded. The new stable upgrade guide separates desktop update
from explicit management activation and refuses incompatible downgrade shortcuts.

## M472 — Published-binary upgrade rehearsal and final release freeze (2026-10-29)

Status: active release qualification; publication remains gated. Supersedes the
preparation guides’ mistaken schema46 reference: the shipped target is schema47.
M471’s functional and privacy limits remain unchanged.

The offline release-upgrade-030-02 guest used genuine published0.1.0/0.2.0 tools
and candidate source7ea77ba. It created two fresh fakes, kept one running and one
stopped, and passed sequential upgrade, identical-target reinstall, old-client
observation and old-runner refusal. Identity/config/history and the engine plus
running container process were preserved. No user review disk, host service or
network interface was exposed. The initial harness attempt failed before boot
because an unmapped backing path needed an explicit virtual size; its generated
staging was classified and retired with a receipt. The correction and compact
phase reporting are included before the next source freeze. Final-source artifact
qualification remains required.

The managed race suite’s20-minute package timeout expired during an active test
that had run for one second, with no preceding assertion or race failure. This is
a cumulative suite-budget failure, not a passed result. A45-minute diagnostic run
with JSON progress is running; CI allows60minutes for this suite and90 for the
whole source job. Other race packages passed. Do not relabel timeouts as success.

Theme’s290 strict Python tests (ResourceWarning as error), DMs’152 Python tests,
Node model/search/time and paired real-Qt layout checks passed. Both DEV plugins
now use ui-20261029-cohesion-8; only a normal plugin rescan was requested. Protected
production PIDs2166/3050 retained their Oct21 start times. The personal DMs journal
retained SHA2563942f405aea57d3879c6583d38681894e753f97ff418f23a34b1a521983edb6a.
Companion PRs are Theme#1 (0.4.0) and DMs#1 (0.2.0); they remain unpublished.
Retained test roots measured33.52GiB including the completed upgrade guest, below
the40GiB ceiling.

## M473 — Correct release pill identity before publication (2026-10-29)

Status: active; supersedes any preparation claim that the217MB test pill is
Recommended Apps. That file is the pinned standard Vere4.6 bootstrap. Current
collector Recommended Apps pins still intentionally resolve the59,351,751-byte
RC.11 asset, also published unchanged in0.1.0/0.2.0, SHA25685635c0b17a138a6473580ca362475faadf281e09cc9fa3746cfc698cc054f62.
Candidate01/02 packaging mistakenly attached the standard pill under the
Recommended Apps name. Neither was published. Replace them with a new frozen
source/package that retains the existing Recommended Apps bytes and corrects
the bundled installation guide. The upgrade harness now takes its pill from
the exact candidate asset, avoiding this implicit lab-default mismatch. The
previous passing upgrade result remains standard-bootstrap evidence only.

## M474 — UTC recovery portability and bounded complete race gates (2026-10-29)

Status: active release qualification; supersedes M472's single45-minute managed
race plan. That run completed276 top-level tests without a reported assertion or
race, then hit its cumulative deadline. It is not a passing complete gate. CI now
partitions the executable managed test inventory by stable name hash into four
non-overlapping groups; the required source check depends on every group and the
other source checks. Every discovered test is assigned exactly once. Normal tests
explicitly run in UTC with a30-minute package budget.

Public CI identified recovery journal failures that local Pacific-time tests did
not show. `TZ=UTC` reproduced them. Structural time.Time equality incorrectly gave
Go timezone/monotonic metadata authority over persisted timestamps. The project
recovery comparisons now compare instants while preserving exact other fields,
including nil/empty distinctions. Direct process start comparisons use time.Equal.
Regression tests cover UTC/east/west JSON reopen, exact historical checkpoints,
nanosecond changes, ownership, acknowledgement and slice differences. No stored
bytes, schema or journals are rewritten. A source-archive CLI replay fixture also
stops requiring Git VCS stamping; release provenance remains separately checked.

Candidate03/source49ada1 passed the actual Recommended Apps upgrade guest with
published0.1.0/0.2.0 tools, schema21→47, running/stopped intent and runtime process
continuity. It remains unpublished and is superseded for release by these fixes;
new frozen artifacts must pass the final gates. The older standard-pill overlays
were retired only after success, inactive scope and open-file checks, retaining
compact results/hash receipts. User review disks, DEV piers and production remain
untouched. Theme/DMs PRs merged after passing CI; release drafts remain unpublished.
Authentication functionality is qualified separately from the still-open broader
security/privacy audit and wallet key custody work (M468).

## M475 — Frozen0.3.0 candidate passes upgrade and native DEV smoke (2026-10-29)

Status: final qualification/distribution; publication still requires protected PR
checks. Supersedes M474's pending replacement-artifact note. Source dc420f133277abc6108477dd8f0b5ddca35e44b2
is frozen. Candidate04 and its eight unpublished draft downloads match exactly.
The actual Recommended Apps guest passes published0.1.0→0.2.0→0.3.0, schema21→47,
repeat activation, old-client observation, old-writer refusal, manager restart
and clean exit. The support guide explicitly requires0.1.0 users to take0.2.0
first; direct skip and other distributions are not newly live-qualified.

All36 test-bearing Go packages pass in UTC (managed292.569s), vet passes, and the
project-network race suite passes (182.745s). All130 documented help entries were
checked against the actual artifacts offline without state changes; five skill
packaging tests pass. Native bundle startup/activation failure/restart checks pass;
govulncheck1.9.0 finds no known vulnerabilities in all3 binaries. Complete managed
race coverage remains a required four-group CI gate, not inferred from partial
20/45-minute runs. The qualification record links its exact CI run.

Only Omabit DEV received candidate binaries and ui-20261029-release-030, displayed
as0.3.0-dev.review1. Its manager restart preserved all6 DEV container processes.
Demo-nec clean Stop/Start and exact receipt replay succeeded; all3 fake HTTP host
identities and fresh metrics pass, and both moons remain stopped. The ad-hoc probe
was corrected from unauthenticated /~/name (visitor identity) to /~/host; collection
inventory uses no runner selector and checks its sole returned runner ID. These
were test-script corrections, not product defects. Theme/DMs retain cohesion-8.
Production and personal-message baselines remain protected. Broader security/privacy
qualification stays open as recorded in M468; this is not wallet custody assurance.

## M476 — Deadline fixture accounts for handles actually opened (2026-10-29)

Status: release source reopened for a test-only correction; supersedes M475's
final-distribution status, not its completed runtime evidence. CI's runtime race
stage reported TestPrivateGatewayDriverDeadlineAndFailedObservationPreserveState.
Its15ms budget can expire during invariant inspection before any observer opens;
the fixture incorrectly required one Close even then. Cleanup now balances the
actual number of opened handles, including partial-open errors. Error-only cases
retain a normal budget and must exercise their selected failure. A deterministic
before-open deadline case proves the zero-handle path; real caller-deadline coverage
is retained. Fifty race repetitions of the driver group and fifty of the explicit
boundary pass. No production runtime behavior changes in this correction.

Candidate04's real guest and native DEV results remain true for its exact source.
Do not publish it under a newer source label. Restore preparation pins while the
corrected source passes CI, then rebuild/verify and attach fresh qualification to
the final immutable artifacts. The private draft and release PR remain unpublished
and unmerged. Three of four managed race groups have passed at this checkpoint;
all groups remain required. Generated completed upgrade disks were retired with
hashes/results; retained visible allocation is30.78GiB (rootless work directories
have restricted ordinary-user visibility), below the40GiB ceiling with ample
headroom. Production, user review disks and all DEV piers remain preserved.

## M477 — Final0.3.0 artifacts and exact upgrade evidence (2026-10-29)

Status: final distribution prepared; protected PR checks still gate publication.
Supersedes M476's pending rebuild and M475's artifact source. Frozen source
`d859ebfa33116b753eb4e1d103221e60b09c4597` produced candidate05. Its correct
Recommended Apps pill, package/provenance/install checks, actual native startup
and recovery,130 offline help entries,57 workflow commands across six simulated
tasks and all three govulncheck scans pass. Eight unpublished draft downloads
match exactly. Final runtimequalify race passed476.425seconds; repeated deadline
fixtures passed. All four managed race groups passed on the preceding runtime
source; the corrected source and distribution must still pass every required gate.

The final isolated guest passes the actual published0.1.0→0.2.0→0.3.0 path,
schema21→47, repeat activation, old-client observation, old-reader refusal,
manager restart, running process continuity and stopped intent. Its ships stopped
cleanly and the guest powered off. Host receipt records31.91GiB visible retained
allocation after the run, below40GiB; the run stayed inside its4GiB scratch cap.
Compact evidence and hashes are retained; completed generated disks are eligible
for retirement after final publication. User review disks and DEV piers remain.

Only DEV managers received final binaries. All6 existing container process
identities survived activation. Candidate04's lifecycle/metrics/identity evidence
remains explicitly bound to its source; production implementation is identical
to candidate05 (only a deadline test and documentation differ). UI remains
0.3.0-dev.review1 with the cohesive Theme/DMs DEV generation. Production manager
and engine retain October21 start times, and the personal outbound journal digest
is unchanged. Broader security/privacy qualification remains open under M468;
functional authentication tests do not establish wallet custody or malicious
same-UID plugin isolation.

## M478 — Offline guest fixtures are independent of the CI owner (2026-10-29)

Status: test-only source correction; supersedes M477's final-distribution status,
not its successful upgrade/artifact evidence. Frozen-source CI completed all Go
normal/race groups and QML successfully, then failed the offline probe fixtures.
A networkless UID1001 reproduction identified two tests that treated CI-created
files as if their owner were the explicitly admitted guest UID1000. The tests now
model only ownership metadata in their private temporary tree, preserving actual
file type, mode, size and link-count checks. Explicit foreign-owner tests refuse
both cookies and runtime directories before any client call or directory creation.
Real probe admission remains fixed to its guarded offline guest.

All25 probe tests pass under UID1000 and1001, including strict ResourceWarning
handling locally. All15 remaining offline CI fixture groups also pass in a
networkless UID1001 namespace. No production runtime implementation changed.
Rebuild from the corrected clean source and preserve exact-source qualification;
do not relabel candidate05, and do not bypass the final protected checks.

## M479 — Corrected-source0.3.0 qualification and baseline installer recovery (2026-10-29)

Status: final distribution prepared; protected PR checks still gate publication.
Supersedes M478's pending rebuild and M477's artifact source. Frozen source
`8cce14b9afaa1ce45dd98c31b679f76a6017fc03` produced candidate06. Its package/provenance checks, reproducible
builds, actual native startup/recovery,130 offline help entries,57 workflow
commands across six simulated tasks and all three govulncheck scans pass.
Eight unpublished draft downloads match exactly. All25 probe tests pass under
UID1000/1001 and all15 offline fixture groups pass in a networkless UID1001
namespace. Full Go/race coverage passed on the preceding production-identical
source; every required gate remains mandatory on the final distribution.

The first candidate06 guest stopped during published0.1.0 baseline installation,
before0.3.0 ran. Read-only inspection found exec.WaitDelay after systemd had
started the service. The0.3.0 command-drain fix already covers this case. The
original failure and installer receipt remain recorded. Repeating the identical
0.1.0 installer in that same guest resumed its fence/state, then the full
0.1.0→0.2.0→0.3.0 upgrade, repeat activation, old-client observation, schema47
old-writer refusal, manager restart and running/stopped preservation passed.
Both fresh ships stopped cleanly and the guest powered off. No failed result
was relabeled as a clean first attempt. The small resume driver is hash-bound
in the qualification evidence and never touched host services or user piers.

Visible retained allocation after this run was34.09GiB
against40GiB. All6 DEV container process identities survived exact final binary
activation. Earlier lifecycle/metrics/HTTP evidence retains its original source
binding, with unchanged production implementation. UI stays0.3.0-dev.review1;
Theme/DMs use the matching cohesion generation. Production and personal-message
baselines remain unchanged. Broader security/privacy and wallet work remain open
under M468; functional sessions are not malicious same-UID plugin isolation.
