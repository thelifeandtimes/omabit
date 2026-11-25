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

## M480 — Bound native package evidence to a fleet display response (2026-10-29)

Status: active; supersedes M479's candidate06 release-readiness claim. No release
has been published. Final idle-CPU observation found the DEV manager consuming
about80–90% of one core. A read-only profile of the two existing DEV project
fleets attributed70% of samples to repeated native-package inspection. Copied
SQLite metadata reads took milliseconds; neither pier copying nor history growth
caused this issue.

Fleet display now shares one native-package inspection within each project
response. It retains every engine identity, namespace, container, HTTP identity,
storage and containment check. No report survives a response; login-code and
mutation entry points keep independent admission. Tests cover concurrent reads,
changed engine evidence, new-response invalidation and unchanged mutation refusal.
The same read-only DEV observation fell from2.36 to0.54 sampled CPU seconds
(3.04 to2.41 seconds elapsed, including telemetry waits). This is a bounded
measurement, not a general performance guarantee. Candidate07 must rebuild and
repeat exact-artifact and upgrade gates before publication. Production services,
DEV container processes and personal message history were not changed by diagnosis.

## M481 — Final0.3.0 artifact, upgrade and DEV verification (2026-10-29)

Status: distribution prepared; protected PR checks still gate publication.
Supersedes M480's pending rebuild and M479's candidate source. Candidate07 is
frozen at `ffb3f7c036af3d62220cd2610d77548900568c15`. Actual package/provenance/reproducibility checks,
130 offline CLI entries,57 workflow commands, native startup/activation recovery,
all3 vulnerability scans and focused runtime race regressions pass. Matching
headers and connection-choice components remain byte-identical across Theme/DMs.

The new offline guest passes published0.1.0→0.2.0→frozen0.3.0, repeated activation,
old-client observation, old-writer refusal, manager restart and clean shutdown.
The earlier genuine0.1.0 WaitDelay/recovery history remains in candidate06; it is
not relabeled as this run. Compact host/result files bind this final source.

Exact final DEV activation preserves all6 container processes. The DEV ~nec then
passes clean Stop/Start and exact request replay. All3 fakes report matching HTTP
identities and metrics; both moons remain stopped. The test initially parsed the
plain-text host endpoint as JSON; the read-only probe was corrected without
repeating lifecycle actions. The manager CPU observation drops from80–90% to28.6%
over a20-second window. This is useful measured improvement, not a zero-cost or
fleet-scale performance claim. Production and personal-message baselines remain
unchanged; M468's security/privacy and wallet follow-up is still open.


## M482 — Publication and DMs native-install repair (2026-10-29)

Status: observed and repaired. Supersedes M481's pending-publication status only.
Omabit0.3.0, Theme0.4.0 and DMs0.2.0 are published; Omabit merged as
`2dc56e988ee13925fd12f1303e32840febf823f1`. Candidate07's publication receipt remains
in the compact release evidence; historical qualification boundaries are unchanged.

The stable `urbit.dms` installation was a root symlink into the user's clean
0.1.0 development checkout. Omarchy's validator rejects root and nested symlinks;
its updater rolled back after validation failed. The old README's symlink advice
is superseded by a normal, validated Git checkout and separately isolated DEV
installation. Do not repair this by deleting/resetting the development repository.

Replaced only the installed symlink with a validated0.2.0 checkout at
`7bc6f5ec8b0eb0e38fb864c27b8c71360e4f9156`. The original link and repair receipt
are retained in `~/.local/state/omabit-repairs/dms-native-8_8u6u5n/`; the original
repo, saved application state and layout were preserved. Rescan succeeded and the
normal updater reported current. Production runner/engine PIDs and start times
remain unchanged. No personal message was retried.

## M483 — Guided local service-upgrade buttons (2026-10-29)

Status: implemented and fixture-tested; not yet a released or live-qualified
upgrade path. Supersedes the product expectation that routine local upgrades
require users to paste the install-host command batch. It does not supersede the
published0.3.0 recovery procedure or imply those old artifacts contain this UI.

Setup now has Check services, Update services, progress and retained Retry controls.
The helper binds the current local owner, runner, enrollment and access bytes;
uses the original operator key; retains one exact executable/dependency bundle;
and requires OS authorization. The privileged identity check runs under the host
setup lock. A new desktop cannot retarget an unfinished attempt. No host reset,
key regeneration, implicit OS package update or engine restart is introduced.

Source tests cover interruption, changed inputs, identity refusal, readiness and
real Qt control bindings. DEV local-setup restrictions and reload interlocks are
retained. Final new-artifact disposable-host upgrade qualification remains open;
installer changes belong in the next minor release. See
[the workflow and validation limits](guided-service-upgrades.md). Security/privacy
review remains open as recorded in M468; no wallet-grade assurance is claimed.


## M484 — Requested 0.3.1 and first guided upgrade rehearsal (2026-10-29)

Status: first exact-artifact guest passed; second mixed-desktop scenario pending.
The user's explicit request for 0.3.1 supersedes M483's next-minor expectation
for this one release. Protected PRs and review remain required; no publication
is authorized before the rehearsal review. The trusted-base policy still needs
a reviewed path for the exception before publication; changing head code alone
cannot make that CI gate accept it.

Candidate source `a13ca6787735f155ca2a9abf58c54098e010ac53` includes hash-checked
reuse of the installed upstream launcher/legal bytes, with verified HTTPS fallback
and newly rendered templates. It introduces no cache of untrusted executable
content and does not execute installed scripts during preparation. Six local
reuse/fallback cases, focused race/vet checks, 132 exact-binary offline help entries,
archive/provenance/install checks, cold HTTPS preparation and real Qt/bootstrap
regressions pass.

The fresh offline `guided-upgrade-031-from020-01` guest used published0.2.0
installer/collector and the genuine desktop archive. Its setup receipt was
reconstructed from actual installer evidence and real old-collector SSH enrollment;
the original desktop verified readiness. Actual QML buttons launched the sealed
0.3.1 helper and real pkexec. Guest-only polkit denied once, then granted the exact
retained installer. Retry, schema21→47, unchanged identity/settings/history/access,
running engine/container continuity, stopped intent, current-version no-op, old
reader refusal and final clean Stop passed. Whole-shell activation acknowledgements
are driven by the harness; password entry and Omarchy rescan remain human checks.

Generated disks were retired after poweroff, recovering 2,247,204,864 allocated
bytes. Host/result/serial receipts and exact guest harness inputs remain. No host
production or human DEV ships were used. M468's security/privacy follow-up remains
open. This evidence supersedes M483's lack of live qualification for this first
scenario only; it does not announce a released0.3.1.


## M485 — Both guided upgrade paths passed; review pending (2026-10-29)

Status: rehearsal complete, unpublished candidate. Supersedes M484's pending
second scenario. The separate `guided-upgrade-031-from030-01` guest starts with
published0.3.0 desktop and0.2.0 collector/runner. It passes the same real QML,
pkexec denial/retry, schema21→47, process continuity, stopped intent, preserved
access/settings/history, no-op recheck, old-reader refusal and final clean Stop
assertions. Both cases used frozen source `a13ca6787735f155ca2a9abf58c54098e010ac53`.

A test-only screenshot export read the retry PNG before Qt finished writing it.
The guest's original result was successful; host screenshot postprocessing failed.
The original serial result and offer/completion images were recovered without
rerunning any ship action. The harness now waits for saveToFile acknowledgement
and records upgrade results independently from optional screenshots. A separate
networkless real-Qt capture check passes with zero helper dispatches. This is not
a product upgrade failure and the missing retry image is not invented.

The second guest's generated files were retired after poweroff (2,168,168,448
allocated bytes); receipts, small logs and harness inputs remain. Measured shared
usage after retirement is30.251GiB, with the same18 unreadable Docker overlay work
entries excluded by ordinary-user du. Production service PID/invocation pairs
remain3050/ad17500e2d604ec7841d9f352420739c and2166/b2932cedfac141648ae88a69e8c4c2b3.
No production or human-review ship was used.

Review artifacts: `docs/releases/0.3.1/rehearsal.json` and the compact local
`release-0.3.1/rehearsal-01/REVIEW.html`. Real password entry and stock Omarchy
update/rescan remain manual checks. Remote-runner GUI updates and arbitrary copied
CLI binaries are outside this button. The GitHub master protection still requires
source, qml-offscreen and release-policy, with admin enforcement. Its trusted-base
minor-only code blocks a0.3.1 PR even though the head includes the requested narrow
exception. This needs a separately reviewed policy transition before publication;
no protection was changed and no release was created.


## M486 — Authorized0.3.1 publication from rehearsed artifacts (2026-10-29)

Status: distribution prepared; GitHub checks and policy transition pending.
The user explicitly authorized publishing and chose live rehearsal afterward,
superseding M485's publication-review wait. The release branch starts from the
exact frozen source a13ca6787735f155ca2a9abf58c54098e010ac53; only documentation
and native release pins follow it. The later test-harness improvements remain on
codex/guided-service-upgrades and are not represented as tested runtime changes.
Both small guest results are included with qualification. Existing binaries,
archive and pill remain unchanged; metadata may attach qualification.

This authorization does not itself remove the documented protected-master
requirements. The trusted-base minor-only check still needs a narrow explicit
transition before the release PR can merge. No host update is part of publishing;
production ships, tmux and temp-ur remain outside this task. M468's privacy/security
and wallet work remains open.


## M487 — Qualification replaces minor-only release numbering (2026-10-29)

Status: user requirement; supersedes the version restrictions in M483/M484 and
the narrow exception/approval wait in M485/M486. The user clarified that master
should remain a working native Omarchy plugin, with tested non-destructive upgrade
paths between releases, possibly via intermediate versions. Patch releases and
backend fixes are allowed when qualified. Version digits do not certify safety.

The amended gate retains exact frozen artifacts, UI/tool version agreement, source
identity, passing source/race/QML tests and upgrade evidence. It additionally
checks a declared path from the previous stable version and hashes the committed
successful rehearsal evidence bound to the target. Specific governance files may
follow the freeze without rebuilding runtime artifacts; packaging/bootstrap/CI
implementation changes still require a new freeze. The original0.3.1 CI source
failure was the same minor-only rule rejecting reproducibility packaging after
version pins advanced, not a Go or QML test failure. No test is waived.

The user authorized changing the merge requirement after the legacy gate's
bootstrap problem was explained. Apply the documented one-time transition only
after all other checks pass and immediately restore full protection. Publication
authorization remains active; production ships are untouched.


## M488 — Explicit QML types and visible dependency updates (2026-10-29)

Status: source fix; exact candidate qualification follows. The user's stock-shell
0.3.0→0.3.1 update retained0.3.0 with `ServiceUpgradePane is not a type`. The
published archive contains ServiceUpgrade.qml and ServiceUpgradePane.qml but
omits both from qmldir. Register every shipped component explicitly and enforce
registry completeness in CI. This supersedes any implication in M485–M487 that
the isolated successful load covered every long-lived Omarchy shell update.
Fresh and simple previous→new isolated loads of the old bundle still pass; the
full host-specific resolution failure is not reproduced by that fixture.

The new menubar notice performs a read-only local version check when opened and
offers Update Dependencies. A verified offer starts the existing bound upgrade
and opens local-service progress with OS authorization. Errors require a new
check before retained retry; success hides the notice. Disabled host-setup
profiles and fresh hosts do not offer a production update. No installer, Go,
runner schema or authorization contract changes. Published0.3.1 stays immutable;
the correction is prepared as0.3.2. No production service or ship was operated.

## M489 —0.3.2 candidate desktop recovery qualification (2026-10-29)

Status: frozen candidate742e56ba968c4eb4544fae52c7eb41f31998d6df; publication gates
pending. Extends M488 with exact0.3.0→0.3.2 and0.3.1→0.3.2 packaged desktop tests,
saved-value preservation after a synthetic missing-type failure, hot activation,
cleared failure and restart. See releases/0.3.2/desktop-upgrade.json. The original
host-only resolution failure remains unreplicated by cold/simple hot fixtures.
The published0.3.1 payload is unchanged.

The new menubar readiness check initially triggered a Qt binding loop during
canStart notification; deferring it to the next event turn fixes it. Four button
scenarios pass, including activation becoming available, denial/recheck/retained
retry, double-dispatch protection and hiding after completion. Superseded payloads
were removed; only small build diagnostics remain. The final bounded build peaked
at867540992 allocated bytes. No new VM or ship was created.

A preservation harness initially required identical INI bytes. Qt added unchanged
empty machineUse/runners defaults on object destruction; all prior values were
preserved. The corrected harness checks every old key/value and the actual loaded
selection. Both paths pass. This test-only assertion correction is not a product
migration failure and does not weaken preservation of any existing value.

Go/runtime/installer/bootstrap implementation is identical to the0.3.1 frozen
source; native release pins differ. The broader backend digest differs because
release-governance code changed after that freeze. Earlier live service-upgrade
receipts remain evidence for that unchanged implementation; no new live-host
migration or human password entry is claimed. The original production runner and
engine PID/invocation pairs remain unchanged. Measured combined storage is about
30.4GiB with the same18 unreadable overlay work entries, under the40GiB budget.


## M490 — Production imports remain manual (2026-10-29)

Status: current user constraint. The user is importing their live fleet into
Omabit. Protect `~lismyl-lismyl`, `~sarlev`, `~sarlev-sarsen`, `~tocwex` and all
of their moons, in addition to the prior protected identities. No agent-run live
imports; qualify changes with synthetic fixtures. Existing tmux/temp-ur and DEV
review-state exclusions remain. This supersedes historical live-import/reset
permissions, not the user's ability to import manually through the product.

The current polishing pass covers rooftop dependency verification, large local
pier imports, themed progress, post-import navigation and compact application
connection rows. Rooftop diagnostics are read-only and limited to OS/packages;
no production runtime or package changes are implied.

## M491 — Ubuntu verification and large local import correction (2026-10-29)

Status: tested source on `codex/production-import-polish`; not published or
installed. Extends M490, whose production/manual-import constraints remain current.
See [the report](diagnostics/2026-10-29-production-import-polish.md).

Rooftop's package error was not missing Docker dependencies: Ubuntu's separate
component lines and an unrelated70,841-byte Provides field exceeded assumptions
in our verifier. Merge only equivalent source/key/architecture authority and allow
bounded1 MiB unselected fields; retain selected4096-byte fields and all signature,
archive and executable checks. The fixed read-only native verifier passes all
eight roles on rooftop. Missing-package installation retains its explicit terminal
approval. This is not evidence of a completed remote installation or a native
package transaction. Never replace an unfinished attempt's pinned helper silently.

The old100 GB local import bound was inherited from cross-host handoff. Current
source allows8 TiB locally, including a size-derived copy/verification deadline;
the100 GB network handoff limit remains. No persistent format changed. A synthetic
512 GiB same-filesystem import passes with no runtime call. Large full payload
transfers remain unqualified; do not treat sparse tests as transfer throughput
measurements. This supersedes the local100 GB statements in earlier source plans,
not the behavior of already published0.3.2 or the handoff contract.

Use explicit themed QML components for progress and one-row connection controls.
Post-import navigation requires both the original successful operation and the
fresh exact runner/connection/instance revision/path/ship/source. Keep failures
and uncertain receipts available for recovery. Offscreen UI tests cover these
bindings and narrow-width geometry without accessing production desktop controls.


## M492 — Frozen0.3.3 qualification and authorized publication (2026-10-30)

Status: qualified candidate; protected PR checks and publication pending.
The user explicitly requested “build and publish.” Frozen source
`2e59eca006c6e457d64086ff6df4d069f69806a9` contains M491's fixes.
This supersedes M491's source-only status when publication completes; it never
authorizes a production service update. M490's manual-import and protected-ship
constraints remain active.

Actual0.3.2 desktop/services →0.3.3 passes a fresh offline guest with one running
fake and one stopped fake, real QML button denial/retry, exact target identity,
unchanged runtime processes, fleet/history/access, repeat no-op and manager
restart. Schema47 stays47. Native QML separately proves failed-candidate recovery,
preferences and hot/cold activation. All132 offline CLI help entries match the
shipped references. See [qualification](releases/0.3.3/qualification.json).

The guest reuses the immutable base with a4 GiB scratch cap under the combined
40 GiB budget. No production service, profile, pier or manual-review state changes.
Do not expand cross-host handoff's100 GB cap: the user accepted it for now.


## M493 — QML fixture correction before0.3.3 publication (2026-10-30)

Status: candidate01 not published; supersedes M492's readiness for promotion.
The actual packaged desktop and exact-binary guest passed, but protected CI
found the older RC11 offscreen fixture copied PierImportPane without its new
ProgressMeter dependency. Add the component to that fixture; all three required
QML checks pass locally. No application/runtime change is needed.

Refreeze and rebuild instead of exempting test changes from the release freeze
policy. Repeat exact-artifact qualification for the new source identity; retain
compact failed-candidate evidence and retire only its generated artifacts. The
new freeze also refreshes the archived installation guide's stale release title,
support-path links and local-size limit. Published0.3.2 and production remain
untouched; only our unpublished0.3.3 draft is replaced after qualification.


## M494 — Corrected0.3.3 candidate qualification and publication (2026-10-30)

Status: qualified candidate; protected PR checks and publication pending.
The user explicitly requested “build and publish.” Frozen source
`bcc68e16f8d1100ee0e564972213f6193999a656` contains M491's fixes.
This supersedes M492/M493's first candidate after the fixture correction; it never
authorizes a production service update. M490's manual-import and protected-ship
constraints remain active.

Actual0.3.2 desktop/services →0.3.3 passes a fresh offline guest with one running
fake and one stopped fake, real QML button denial/retry, exact target identity,
unchanged runtime processes, fleet/history/access, repeat no-op and manager
restart. Schema47 stays47. Native QML separately proves failed-candidate recovery,
preferences and hot/cold activation. All132 offline CLI help entries match the
shipped references. See [qualification](releases/0.3.3/qualification.json).

The guest reuses the immutable base with a4 GiB scratch cap under the combined
40 GiB budget. No production service, profile, pier or manual-review state changes.
Do not expand cross-host handoff's100 GB cap: the user accepted it for now.


## M495 — 0.3.3 publication completed (2026-10-30)

Status: published; supersedes M494's pending CI/publication status only. PR #10
merged normally as `8f5fbfd` after all source/race/QML/policy checks passed.
Frozen source `bcc68e16f8d1100ee0e564972213f6193999a656` is tagged v0.3.3.
Public asset bytes and private cold native bootstrap passed, with master
protection preserved. No production service or ship was upgraded by the agent.
Compact publication records are retained in release-0.3.3/candidate-02 evidence.

## M496 — macOS metadata import compatibility (2026-10-30)

Status: source fix and synthetic regression qualification; not yet released.
The user authorized a one-time filenames/types listing that found regular
`._0i391592`, `._data.mdb` and `._lock.mdb` companions. Their names match macOS
AppleDouble conventions; no content or whole-pier health inspection was performed.
The old log-root allowlist rejected these before import. Recognize bounded
regular companions and `.DS_Store`, retain their bytes and metadata fingerprints,
and keep actual epoch/state and filesystem guards. Unexpected names are quoted.
Source tests cover copies/moves, interrupted-copy retention, enrollment recovery,
changed metadata, limits and unsafe files. This is an uncovered compatibility
case in 0.3.3; it does not invalidate that release's stated tested scope.

The user's latest instruction expressly forbids touching `~sarlev`, which will
run elsewhere. This supersedes the one-time listing permission; no further reads
or operations are allowed. Preserve all other production and storage constraints.


## M497 — Superseded 0.3.4 rehearsal fixture (2026-10-30)

Status: unpublished candidate superseded, not release-qualified. Source
`065475cd36da9406fa4e6cf86e7e23afec499e9f` passed synthetic source and desktop
tests and the real QML-button service upgrade with fleet/access preservation.
The final import rehearsal failed because its local `source` path variable was
reused for shell QML text. This is a harness error, not evidence of an import
product failure or a passed full rehearsal. Give the disposable pier its own
`import_source` variable, freeze/rebuild, then repeat in a fresh offline guest.
Retain compact failed-run evidence and retire only that powered-off generated
guest's disks. M496's no-touch production constraints remain unchanged.


## M498 — 0.3.4 qualified macOS import patch (2026-10-30)

Status: qualified frozen candidate; protected CI and publication pending.
Supersedes M497's failed rehearsal and M496's source-only status, preserving its explicit no-touch rule for
`~sarlev` and all other production exclusions. Frozen source is
`bdfe47516bf2c0482cc8450c8c31e2f2db53612e`.
The actual 0.3.3 → 0.3.4 QML-button upgrade preserves running container/engine
processes, stopped intent, fleet history and access; denial/retry, no-op and
manager restart pass. An independently copied guest-only stopped fake reproduces
the old metadata rejection, then identifies and imports stopped with every
synthetic metadata hash retained under the new runner. Native desktop recovery,
preferences, hot/cold activation, archive provenance and all 132 CLI help entries
pass. See [qualification](releases/0.3.4/qualification.json) for exact scope.
Schema 47 remains 47. No installed host service/plugin or protected pier changes.
The guest stays within its 4 GiB scratch allowance; storage accounting adds a
separate 4 GiB reserve for unreadable DEV overlay work directories and records
that limitation rather than claiming a complete measurement.


## M499 — 0.3.4 publication and bounded performance pass (2026-11-01)

0.3.4 was published after normal PR #11 merge (31bdea8), with frozen source
bdfe47516bf2c0482cc8450c8c31e2f2db53612e and distribution 0744fe6. This supersedes
M498's pending publication status; public bytes/cold startup/protection evidence
is retained in release-0.3.4/candidate-02/publication-memory.json. No production
service or ship was upgraded by the agent.

The user now requests CPU/battery optimization through Monday November 2 at
06:00 Pacific (14:00 UTC), with no UI/UX changes and a review candidate at the
end. See diagnostics/2026-11-01-idle-performance.md for ten ranked targets,
measurements, deadline and storage accounting. The earlier broad platform goal
remains paused; this separate task does not authorize production deployment or
publication. Preserve all protected-ship and manual-review-state exclusions.


## M500 — Measured idle-observation candidate, not deployed (2026-11-01)

Status: ongoing performance pass through M499's Monday 06:00 Pacific deadline.
Topic branch `perf/idle-observation-20261101` in `omabit-platform-overhaul`.
First checkpoints 0ba2f58 and 9694785 retain three private authenticated transports,
add 2 s visible/pending-work versus 30 s hidden observation with immediate wakeup,
remove repeated bootstrap interpreter launches and no-op sample-map updates.
Real isolated OpenSSH measurement: 47 -> 3 sessions per 30 s at visible cadence,
with retained-session revocation/identity/role/replay regressions passing.

The third pass batches snapshot epoch-history reads without schema or stored-state
changes; 128 synthetic ships with 2,048 ordinary journal records measured
171.91 -> 11.13 ms per snapshot. See diagnostics/2026-11-01-idle-performance.md
and adjacent JSON evidence for exact scope, CPU/time distinctions, tests and
remaining qualification. The full managed suite passed before the final row-ID
query refinement; final-query focused race checks and vet passed. Complete final
full-suite/paired-artifact qualification before a review installation is proposed.

Production services/plugins, protected ships (especially the express no-touch
~sarlev rule), ~/temp-ur, and human DEV review state remain untouched. The broader
platform goal remains paused. No publication or deployment is authorized by this
performance task. This supplements M499; it does not supersede its constraints.


## M501 — Performance review ready; optimization loop stopped (2026-11-01)

Supersedes M500's ongoing/deadline status. The user permits stopping below 10%
additional improvement per loop and requested a mobile HTML review on the
tailnet. Product source is `4584453`; the final loop added measurement, not a
fourth product optimization. Remaining ideas have no demonstrated additional
>=10% gain. Do not resume optimization or the older paused platform goal.
The heartbeat `omabit-performance-passes-through-monday-review` was deleted
after the private review page was live, stopping further scheduled passes.

Final full Go suite passed on that product source (managed 262.616 s), resolving
M500's final-query full-suite limitation. Actual offscreen QML with an inert
collector measured hidden main-thread CPU over 32 s: empty fleet 0.04598 ->
0.00349 s; 32 stopped ships 1.00338 -> 0.06070 s. Visible observations remained
two-second cadence; visible CPU was effectively unchanged. Fifty open/close
cycles retained one collector and 31 descriptors, returned owners to zero and
drained cleanly; RSS grew about 1.2 MiB. This is not laptop power measurement or
long-duration leak qualification. See diagnostics/2026-11-01-qt-idle.json and
the performance diagnostic for all ten target dispositions and evidence.

The static mobile report is diagnostics/2026-11-01-performance-review.html.
A dedicated copy and hosting verification are in qualification evidence under
performance-2026-11-01/mobile-review. A new private Tailscale path proxies a
loopback-only single-file server; existing routes were compared and preserved.
No Funnel/public exposure, repository directory, keys or fixture logs are served.
Direct static-file serving needed root; noninteractive sudo was unavailable,
so the ordinary-user proxy was used instead. Browser validation at 390 x 844
passed without document overflow, including expanded results. The link is for
review while this machine/report process stays running; it is not an installer.

No product changes followed the final source tests. Fresh paired release builds,
actual installer upgrade rehearsal and real-laptop CPU/battery measurement remain
open before deployment. No release was published or product installed. Protected
ships (especially ~sarlev), ~/temp-ur, production services/plugins and human DEV
review state remain untouched. Storage remains below 40 GiB with the recorded
4 GiB unreadable-DEV reserve; task fixtures use about 231 MiB of 512 MiB.


## M502 — 0.3.5-rc.1 review preparation (2026-11-01)

The user now requests the next version for review. This authorizes frozen local
review artifacts and isolated upgrade qualification; it does not resume the
optimization loop or authorize publishing or upgrading existing services/plugins.
Prepare 0.3.5-rc.1 on the performance branch. The only product change after the
final performance tests is the review version in manifests and Service.qml.

Use exact published 0.3.4 assets and the prior pristine offline guest base, with
a new patch-upgrade-035-rc1-01 overlay. Declared guest scratch cap is 4 GiB, plus
256 MiB retained release assets; previous performance scratch is about 231 MiB.
Current combined visible-root accounting is 31511 MiB plus the recorded 4 GiB
unreadable-DEV reserve; the launcher remeasures exact bytes and refuses over
40 GiB. Build staging must finish and clean up before the guest starts.
No production mounts/network, protected ships, ~/temp-ur or human DEV state.


## M503 — 0.3.5-rc.1 built and qualified for local review (2026-11-01)

Supersedes M501/M502's pending build and installer qualification. Frozen product
source is `cce7cd017bf33b08594a82f0dd8286caed6d8811` on the same performance branch.
Matched UI, omabit, collector, runner, skills and the unchanged pinned pill are
under qualification evidence `release-0.3.5-rc.1/candidate-01/0.3.5-rc.1`.
Qualification was attached without rebuilding the immutable product assets;
final checksums verified. No release publication, push, installed plugin change
or in-place service/DEV upgrade occurred.

Passed native registry (9), bootstrap (12), release policy (15), upgrade-control
QML (4), archive/provenance/install checks, native hot/cold activation and recovery
(9 cases), exact-binary offline CLI reference (132 help entries), and skills (5).
The final product's full Go/race/SSH tests remain as recorded in M501; only version
strings and the version-parameterized qualification harness followed those tests.

Actual offline `patch-upgrade-035-rc1-01` rehearsal passed from published 0.3.4:
new disposable running/stopped fakes, exact QML update button, denied polkit/retry,
access and history preservation, unchanged running container/PID and engine,
completed-update no-op, manager restart and macOS metadata stopped-fake import.
Schema remains47, desktop format1, bootstrap API1. See release-specific rehearsal,
host and qualification JSON. Its powered-off generated disks/boot fixtures were
retired with a manifest (2,506,858,496 bytes recovered); shared base preserved.
Initial setup receipt is reconstructed from genuine installer evidence; human
password input, stock updater, remote upgrade and other distros remain untested.

A separate native preview launcher in candidate-01 opens the real RC with private
HOME/XDG/profile `review035`, no inherited credentials/enrollment and host setup
false. `preview.py --check` passed exact version/source, ready and zero runners/
ships. The first launcher check exposed AF_UNIX path length under the evidence
root; only the wrapper was fixed to use a private temporary /tmp runtime cleaned
on exit. Frozen product artifacts were unchanged. One existing native placeholder
QColor warning is nonfatal; manual appearance review remains appropriate.
The launcher is not installed into Omarchy and does not inherit human DEV state.

The existing private tailnet HTML was updated with results, the one-command
launcher and review steps. Only its owned loopback server was replaced at the
same port; all Tailscale routes compare equal. HTTP/TLS bytes, CSP and refusal to
serve supporting paths verified. Browser review at390x844 has no document
horizontal overflow; viewport restored and tab kept as a deliverable. Report
availability requires this machine and its report process to remain running.

Final charged storage is35.2208GiB/40GiB including4GiB unreadable-DEV reserve and
about231MiB retained performance scratch; RC plus preview cache uses227.3MiB.
No protected ship, especially ~sarlev, ~/temp-ur, production service/plugin or
human DEV review state was used. Original platform goal stays paused; performance
loop stays stopped and heartbeat stays deleted. Do not publish or deploy without
a new user instruction. Local RC readiness is not battery-life qualification.


## M504 — Publish the performance update through the normal updater (2026-11-02)

The user completed the previously instructed stable update step and requests the
next performance-update step. In that context, prepare and publish the complete
0.3.5 stable release so the stock plugin updater and Update Dependencies can
install matching bootstrap/UI/collector/runner artifacts. This supersedes
M503's publication restriction only; no workstation service/plugin upgrade,
production ship operation, human DEV mutation or optimization restart is implied.

Promote the qualified RC product with only its three version strings changed.
Freeze clean source, rebuild stable-version assets, repeat exact 0.3.4 → 0.3.5
rehearsal, and pass protected PR checks before merge/publishing. Initial charged
storage is35.2544GiB including the4GiB unreadable-DEV reserve and231MiB retained
performance scratch. New assets reserve160MiB; disposable guest scratch4GiB:
projected39.4106GiB under40GiB. Finish/clean build and native-test staging before
the single offline guest. Retire only that guest's classified generated disks.


## M505 — Resume stable qualification after workstation reboot (2026-11-02)

The user requests continuation after reboot. Stable 0.3.5 source d3979db and its
immutable built artifacts survive, as do completed archive/native hot-cold/
recovery/CLI checks. The first offline stable rehearsal was interrupted during
its second disposable ship's first boot, before the service-upgrade step; no
result was completed and no successful power-cut behavior is claimed. Its scope
is inactive. Retired only its generated VM disks/boot fixtures with a recorded
manifest, preserving compact logs and the verified immutable base.

The static tailnet review server was restored on the same loopback port and path;
HTTPS bytes match. No production services/plugins or ships were changed. The
user's installed plugin pins and runner were read-only verified at 0.3.4 before
reboot. The next installer review remains 0.3.4 → 0.3.5 through normal update UI.

Resume in fresh patch-upgrade-035-stable-02 with the same frozen assets and 4 GiB
scratch cap. Current charged usage is35.1675GiB including the4GiB unreadable-DEV
reserve; the previous temporary performance scratch was removed by reboot, while
its durable results/logs remain. Projected with guest cap39.1675GiB under40GiB.
No rebuild, protected-state access, deployment or optimization restart is needed.


## M506 — Stable 0.3.5 exact-artifact rehearsal passed (2026-11-02)

Frozen source d3979dbb30efc344ac35687ba2782662981ee478 and its freshly built stable
assets passed the repeated offline 0.3.4 → 0.3.5 QML-button upgrade. Denied
authorization/retry, running container/PID and engine continuity, stopped intent,
access/enrollment/history, repeated no-op, manager restart and synthetic macOS
metadata import all passed. The guest powered off. Its generated disks were
retired with a manifest, recovering2,482,544,640 allocated bytes; the immutable
base remains verified. The interrupted first attempt remains separately recorded.

Native activation/recovery (9 cases), archive/provenance checks,132 CLI entries,
bootstrap (12), native registry (9), skills (5), policy (15), upgrade controls and
observation schedule passed on the stable candidate. Qualification was attached
without rebuilding immutable assets. Schema47, desktop format1 and bootstrap
API1 remain unchanged. See releases/0.3.5/qualification.json and linked records.

Next gate is reviewed protected release PR, matching draft download verification,
merge commit and stable publication. No production or DEV installation is implied.
This supersedes M505's pending-rehearsal status, not its exclusion constraints.


## M507 — 0.3.5 published and verified; manual workstation upgrade remains (2026-11-02)

Supersedes M506's pending-publication state. PR12 passed protected source/full
race/QML/release-policy checks and merged normally as63072b2, preserving frozen
source d3979db and exact qualified product tree. Published v0.3.5 as latest stable
at2026-11-03T01:50:04Z. All nine public downloads match the rehearsed assets;
post-publication isolated native startup/recovery and saved-view restart pass.
Branch protection is unchanged. No workstation plugin/service upgrade or protected
ship operation was performed. The user remains on the manual0.3.4→0.3.5 path.

Stock Omarchy retains keepLoaded service objects across plugin rescans. The
application bundle can activate, but the new bootstrap code needs a shell restart.
After normal plugin update and Update Dependencies finish with local services
matching the bundle, run `omarchy restart shell`. Do not interrupt an active
service update. This is a shell reload, not a ship-container restart. Published
release notes and the private mobile review include this additional step.

Operational correction: cancelling the duplicate current-head push CI left failed
required contexts even after PR merge-ref checks passed. Normal merge refused.
Reran the branch workflow successfully without weakening protection, then merged.
Do not repeat that cancellation pattern; superseded-head runs are different.
Passing runs:41554655173(PR),41554649760 attempt2(branch),41554652642(policy).

The existing tailnet page is refreshed at
https://sourcebox.tail709ccc.ts.net/omabit-performance-review#review. Verified TLS
bytes, CSP, supporting-path404s, unchanged Tailscale routes and390x844 browser
layout without horizontal overflow; viewport restored. It is an ordinary-user
loopback report process and requires the machine/process to stay running.
Final charged storage35.1713GiB/40GiB includes the4GiB unreadable-DEV reserve.
Temporary native test state was removed; no VM is running for this task.

Full publication/check/download/browser/storage evidence and the CI correction
are in qualification evidence release-0.3.5/candidate-01/publication-memory.json
and release-progress.json. These handoff documentation updates are a local topic
commit, not a second same-version product publication. The performance loop stays
stopped, heartbeat stays deleted, and the older platform goal remains paused.
Protected ships, especially ~sarlev, ~/temp-ur and human DEV state remain excluded.


## M508 — 2026-11-02 — Local import preview bound correction (in progress)

The user authorized implementing and publishing a narrow patch after review of
an Inspect pier rejection. Collector guided import and restoration preview used
the 100 GB cross-host bound although the local runner/copy layer permits 8 TiB.
Synthetic RPC tests reproduce rejection above 100 GB in both paths. Align the
client to the existing local bound and distinguish size, identity and destination
diagnostics. Retain every authority, fingerprint, path, stage and entry guard;
cross-host streams remain capped at 100 GB. No schema/runtime migration.

Qualification uses synthetic sparse metadata through the real client/server and
one offline disposable guest, never the user's pier. Exact prior/candidate tools
will demonstrate old rejection and new acceptance above 100 GB, then remove only
the generated sparse fixture before a normal small stopped-fake import. This is
not a claim of a full 100 GB copy rehearsal. ~sarlev remains expressly no-touch,
including reads; ~/temp-ur, production services/plugins and human DEV are excluded.
Publication is authorized; workstation upgrade/import is left to the user.
Declare at most 4 GiB guest scratch plus approximately 160 MiB release assets;
recheck the combined 40 GiB budget including the unreadable-DEV reserve.


## M509 — 2026-11-02 — 0.3.6 frozen and qualified; publication pending

Supersedes M508's in-progress status. Product source is frozen at
b9eb33f2c555a3865c120779a30d3ef4e33672b9. Full Go tests, focused client/managed
race and vet, archive/provenance/install checks, native bootstrap, isolated
QML desktop 0.3.5→0.3.6 activation/recovery and all132 CLI help entries pass.
The real client/server tests admit 100 GB+1,512 GiB and8 TiB logical previews;
negative/over-limit sizes and changed identity/destination plans remain rejected.
Source and destination restoration bounds are tested separately.

The offline patch-upgrade-036-stable-01 guest passes on exact published0.3.5 and
frozen0.3.6 tools. A100,000,000,001-byte sparse fixture uses zero allocated file
blocks; total source logical size is100,399,700,315bytes. The0.3.5 collector
reproduces the generic rejection;0.3.6 accepts with both0.3.5 and0.3.6 runners,
without source/enrollment mutation. Removing only this generated fixture permits
the separate small stopped-fake import, preserving all synthetic macOS companion
bytes. This is not a full100GB copy qualification.

QML upgrade denial/retry, exact target installation, running container/PID/engine,
stopped intent, identity, history, access, no-op repeat and manager restart pass.
Schema47, desktop1, bootstrapAPI1, runtime and pill are unchanged. Guest shutdown
is clean; generated disk/payload/boot scratch is retired with a manifest and the
immutable base verified. Compact results and limits live in docs/releases/0.3.6.
Source, race, QML and release-policy checks must still pass on the protected PR
before merge/publication. No production deployment or protected-pier operation
is authorized by publication; the user installs and imports manually.

## M510 — One-click fresh Omarchy onboarding (2026-11-05, ready for manual trial)

The user confirmed that sourceport's rootlesskit/slirp4netns removal was intentional
fresh-install testing. They require an installed Omabit plugin to take a stock
Omarchy machine to a connected local runner with one setup click and OS password
authorization, including downloading/configuring native prerequisites. This
supersedes the manual-package-preparation UX for Arch/Omarchy, not other host
providers or management-upgrade policy. The user explicitly authorized removing
sourceport's current Omabit plugin for the next manual fresh-install trial; the
stock plugin removal command completed. This does not authorize pier resets.

The `feature/one-click-host-setup` topic branch is based on published 0.3.6 and
labels its candidate `0.3.7-dev.20261105.1`. Root-authorized setup derives a fixed
native package plan, installs missing packages, rechecks prerequisites, authenticates
payloads, then continues owner services and SSH enrollment in the same operation.
The primary button is the installation consent; a second checkbox is removed.
Missing checks retain their names instead of being collapsed into one anonymous
message. Bundled OpenSSH key generation removes the ssh-keygen bootstrap dependency
and preserves existing private keys on retry. The current maintained SSH library
raises the source-only Go minimum to 1.26. No runtime schema change is involved.

Focused Go race/vet, model, offscreen QML and snapshot installation/recovery checks
pass. All three static tools build; the actual sourceport preflight now permits
setup and lists exactly the two intentionally absent packages. Omarchy 4.5.0
validates the staged plugin. The development [trial guide](one-click-setup-trial.md)
records commands and limits. Actual native package download and end-to-end host
installation are the next manual trial; no live success or stable publication is
claimed by these isolated results.

## M511 — First-run navigation and plain-language setup (2026-11-05, ready for repeat trial)

Supersedes M510's UI presentation and pending-first-trial status, not its backend
scope. The user exercised `.1`: read-only inspection on sourceport confirmed both
missing packages were installed, the dedicated user runner/engine and system SSH
service were running, and enrollment matched runner
`runner-8bcf34e969832654f5a2bf9fa3ccd440`. Inventory, operations, incidents and
containers were empty; the standard pier layout directories contained no piers.

The user requested no Add a ship action on the initial frame, direct navigation
from Set up this computer into local device setup, and package names only in
View details. Candidate `0.3.7-dev.20261105.2` implements this in both popup and
monitor entry points. The main setup description explains installing/configuring
software to run Urbit on Omarchy. The local distribution commit is `fcbde29` at
`~/.local/share/omabit-review/one-click-setup`; source changes remain on
`feature/one-click-host-setup` and uncommitted.

The actual host-compatibility/setup QML fixture passes. The existing onboarding
fixture also passes with its real Panel, SetupPane, MonitorWindow and Service in
a networkless Bubblewrap/offscreen run, private HOME/XDG and inert compositor/IPC
adapters. Assertions cover initial hidden ship creation, direct local navigation,
hidden overview cards, setup consent and the existing provisioning checks. The
adapter is `/tmp/opencode/run-onboarding-offscreen.py`; this is not mapped-desktop
visual evidence. Omarchy plugin validation passes.

The user explicitly authorized cleaning up the current trial for another fresh
installation. The clean Git-managed plugin has been removed. The authorized reset
completed after rechecking the exact empty runner. Management files were retired
with same-parent renames; only rootlesskit/slirp4netns were removed from Pacman.
The manifest is `~/.local/share/omabit-review/reset-20261105T191545Z.json`. All
retired inode/device bindings were verified. Pier directories and shared host
settings were preserved. Post-reset preflight reports `can_setup: true`,
`host_present: false`, `ready: false` and exactly those two missing packages. All
three services are inactive with PID 0; active enrollment, setup receipt/key and
panel preferences are absent. Reinstall with the standard command in the trial
guide. This is a manual development reset, not a supported user upgrade path.

## M512 — Explain the three software groups (2026-11-05, ready for review)

Refines M511's details copy at the user's request. Candidate
`0.3.7-dev.20261105.3` describes system dependencies (Docker, RootlessKit,
slirp4netns and OpenSSH), the three Omabit tools and their roles, and Vere as
the Urbit runtime inside Docker containers. This replaces the dense installation
path/version explanation. The existing setup/details QML check and Omarchy plugin
validation pass. Local distribution commit: `3534ac8`; the user can apply it with
`omarchy plugin update thelifeandtimes.omabit`.

## M513 — Key-file Browse on a fresh desktop (2026-11-05, ready for review)

Supersedes M465's external Zenity chooser implementation. Sourceport's Browse
button launched `zenity`, which was absent; missing-executable startup did not
reach the exit-error handler. Candidate `0.3.7-dev.20261105.4` uses the existing
QtQuick.Dialogs FileDialog with explicit DontUseNativeDialog, avoiding both the
extra executable dependency and the historical native GTK/GVFS chooser path.
Accepted selection uses the existing URL-to-local-path conversion; cancellation
preserves the prior filename, and changing it resets boot consent.

The real Qt chooser passed opening, duplicate-click gating, cancellation,
reopening and selection of a synthetic filename containing spaces, Unicode and
`#` in `scripts/test-compact-registration.py`. The fixture is networkless with
private HOME/XDG; it verifies no boot submission. The existing onboarding fixture
also passed in the isolated offscreen adapter, and plugin validation passed.
No actual user keyfile was read. Local distribution commit: `06d7f6a`; apply with
`omarchy plugin update thelifeandtimes.omabit` for manual desktop verification.

## M514 — Key chooser must share the overlay surface (2026-11-05, ready for review)

Supersedes M513's readiness claim: the user's mapped-desktop trial showed the
chooser behind the Omabit menu with blocked interaction. The offscreen test
exercised dialog methods but could not validate compositor stacking or input.
Omarchy's real `Ui.KeyboardPanel` uses the overlay layer; Qt's default non-native
FileDialog uses a separate ordinary window underneath it.

Candidate `.5` explicitly binds the chooser's parent window and selects
`Controls.Popup.Item`, drawing it inside the owning surface. The offscreen check
passes these properties and the prior cancel/reopen/path/no-submission checks.
The new `scripts/test-key-picker-wayland.py` runs the actual KeyboardPanel and
KeyBootPane under privately extracted headless Sway, with Bubblewrap-isolated
HOME/XDG, no host desktop sockets, synthetic files and virtual Wayland input.
The old chooser reproduces as a separate window hidden under the menu, with
pointer selection blocked. The corrected chooser passes actual pointer Browse,
Escape cancellation while retaining the menu and prior path, pointer double-click
selection of a synthetic space/Unicode/`#` filename, reopening/canceling, and
restored keyboard input in the background window. No boot request occurs.
Evidence: `/tmp/opencode/omabit-picker-layer-baseline-10` and
`/tmp/opencode/omabit-picker-layer-fixed-2` (results, screenshots, compositor tree,
source hashes). Sway allows Escape to reach the baseline chooser; this reproduces
its stacking/pointer defect, not every Hyprland keyboard symptom.

Earlier harness runs failed before exercising the chooser: a headless seat needed
both virtual pointer and persistent keyboard capabilities, and its reveal helper
needed bounded scroll positioning. These were harness failures, not product
results. The corrected fixture uses real Wayland input; IPC supplies observations
and menu setup, never programmatic chooser acceptance. Host desktop, user keys,
plugins and services are excluded from the fixture. Actual user Hyprland review
remains the next step through the standard local-candidate update command.
The local distribution is committed as `12fd415`, version
`0.3.7-dev.20261105.5`. Its picker, service and manifest match the tested source
byte-for-byte, and standard Omarchy plugin validation passes. Source changes
remain uncommitted on `feature/one-click-host-setup`; nothing was pushed.

## M515 — Picker confirmed; repeat fresh-install trial requested (2026-11-05)

The user confirmed that candidate `.5` fixed the picker on their actual desktop,
superseding M514's pending-Hyprland-review status. They then explicitly requested
uninstalling Omabit and its dependencies for another end-to-end fresh-install
trial. Current read-only checks bind sourceport's local runner to
`runner-d27c88350efe4c8d298e27c08600cc51`: instances, operations, incidents,
discovered piers, dedicated-engine containers and images are all empty. The known
pier layout contains only its empty standard directories. Pacman history shows
only rootlesskit/slirp4netns were installed by this repeat trial.

The clean installed plugin at distribution commit `12fd415` was removed with
`omarchy plugin remove thelifeandtimes.omabit --yes`. Privileged management-state
retirement and package cleanup are in progress; completion evidence follows.
The first authorization attempt timed out before the script began (no new reset
manifest, services still active, both packages still installed). It was retried
with a ten-minute authorization window.

The retry completed. Recovery manifest:
`~/.local/share/omabit-review/reset-20261106T002120Z.json` (UTC filename; local
date 2026-11-05). All three dedicated services are now inactive/not-found with
PID 0. Management files, enrollment, setup credentials and panel preferences were
retired via same-parent renames; original device/inode bindings were verified.
The pier layout retained its original device/inode bindings. Rootlesskit and
slirp4netns are absent. Read-only setup status reports `can_setup: true`,
`host_present: false`, `ready: false`, and exactly those two packages to install.
The `.5` local distribution remains available for the user's next standard
`omarchy plugin install ~/.local/share/omabit-review/one-click-setup --enable`.

## M516 — First moon boot quarantined by a name mismatch (2026-11-05, diagnosed)

Supersedes M515's empty/reset host status: the user reinstalled `.5`, connected a
new runner and booted a moon with Recommended Apps. CLI snapshot/logs and the exact
running container's public identity endpoint confirm registration
`~somrep-mitfyl-dozzod-sarlev` versus actual `~somryp-mitfyl-dozzod-sarlev`.
Bootstrap completed, but Start failed identity verification and retained durable
provisioning uncertainty. Stop requests are admitted but the graceful keyed-boot
shutdown independently refuses the same name mismatch. Refresh cannot clear it.
The shared network-runtime error incorrectly says "comet" for this owned moon.

No live correction was made. Existing CLI relink/import recovery cannot correct
this first-boot identity. Confirm the intended identity, then qualify an explicit
identity-bound registration correction preserving the existing runtime/pier;
never reset or clear the uncertainty flag blindly. See
[diagnosis and proposed recovery](diagnostics/2026-11-05-moon-identity-mismatch.md)
for exact evidence, code paths and limits.

## M517 — Verified moon identity correction and same-pier resume (2026-11-05, complete)

Supersedes M516's pending-repair status. The user confirmed the intended identity
`~somryp-mitfyl-dozzod-sarlev` and approved a graceful stop/correction/restart.
The new local-owner `repair-boot-identity` CLI binds runner, revision, old/new
identities, exact boot container/process generation, public/private identity and
filesystem objects. A version1 opaque meta journal preserves old launch containment
through shutdown/removal and supports interrupted continuation; final correction
and its audited receipt commit atomically without a schema migration.

Focused and bounded affected-lifecycle race tests pass, including same-pier resume,
refusals, interruption/removal-ack loss, failed final commit, history preservation,
replay and current-manager reopen. Vet and133 network-isolated help entries pass.
The first whole-package race run timed out at ten minutes in unrelated project
recovery/historical-upgrade cases; both tests active at timeout pass in isolation
within the bounded run. Do not describe that aggregate run as passing.

Live repair succeeded at2026-11-06 01:56:16UTC, revision1→2; normal collector Start
succeeded at01:56:31UTC. The existing installed manager now reports the corrected
moon running/healthy, initialized, recovery cleared, applied revision2 and fresh
resources. Exact public identity independently agrees. The pier, its mount and
`.urb` filesystem objects, runtime image, ports, Recommended Apps pill and nine
prior receipts are preserved. The stable instance ID/path keep the original typo;
the displayed label and managed ship identity are corrected. Dedicated engine
PID404613 stayed running throughout. No plugin/runner deployment was needed.

See [repair procedure](boot-identity-repair.md),
[live evidence](diagnostics/2026-11-05-moon-identity-mismatch.md), and compact records
under `~/.local/share/omabit-review/identity-repair-20261106/`. The `.5` local
distribution remains at12fd415. Repair and onboarding sources are uncommitted and
unpushed on `feature/one-click-host-setup`.

## M518 — 0.3.7 release requested; lab unavailable (2026-11-05, preparing)

The user confirmed normal moon operation after M517 and requested a proper stable
release including the one-click setup work. Published0.3.6 remains the baseline.
Preparation moves to `release/0.3.7` with matching root/component/service versions;
master requires reviewed PR, passing source/race/QML/release-policy checks, frozen
artifacts and exact non-destructive previous-stable upgrade evidence.

Sourceport has no retained qualification VM/base. Sourcebox accepts the configured
SSH public key, but this session has no unlocked agent. When asked for lab access,
the user selected **Sourcebox unavailable**, directing source/check preparation
and a report of the remaining qualification blocker. Do not invent successful
upgrade evidence, publish stable or bypass protection. Prepare the release source
and local artifact/desktop checks; finish the VM rehearsal when lab access returns.

## M519 — 0.3.7 frozen local checks pass; service rehearsal blocked (2026-11-05)

Supersedes M518's source/artifact preparation status; its sourcebox blocker still
applies. Source is committed and pushed as
`076f97280960961fc21c88d45fef529185b4fbe2` on `release/0.3.7`, with draft GitHub
PR#14 against protected master. Local matching assets pass checksums, provenance,
isolated installer/replacement checks and all133 networkless CLI help entries.

Local model/native/bootstrap/policy/skill/offscreen/onboarding checks pass. Real
Wayland picker input passes with the corrected extracted-tool path. The exact
published0.3.6 → frozen0.3.7 desktop harness passes hot activation, cold restart,
synthetic failed-candidate recovery and preference preservation inside private
headless Sway/Bubblewrap. Initial tool-path and mountpoint fixture failures happened
before application execution; logs are retained and product source is unchanged.

Evidence and artifact identities: [preparation record](releases/0.3.7/preparation.md),
and sourceport `~/.local/share/omabit-review/release-0.3.7/`. Full source/race CI
run42153140057 is pending. Release-policy correctly rejects the still0.3.6 pins.
Final qualification, exact service-upgrade rehearsal, pins, merge and stable
publication remain pending. The installed personal trial and running moon have
not been upgraded as part of release preparation.


## M520 — 0.3.7 service and desktop upgrade qualified (2026-11-05)

Supersedes the outstanding rehearsal in M518–M519. All six frozen payloads
matched the preparation hashes. Published 0.3.6 upgraded to those artifacts in
the isolated guest, preserving engine and ship processes, stopped intent, runner
identity, enrollment, storage and history through denial, retry and manager
restart. The private Wayland desktop test covered preference retention, failed
candidate recovery and cold restart. See the [release qualification](releases/0.3.7/qualification.json).

## M521 — 0.4.0 scope and first owner-user installer regression (2026-11-06)

The user confirmed 0.3.7 publication. The [0.4.0 plan](platform-roadmap.md#040-release-target-2026-11-06)
prioritizes the rooftop Ubuntu profile, then Debian and Arch/Omarchy on x86-64.
Agent Help follows Omarchy's default-agent crash flow. Runners should share
per-ship origins so that each collector can select one it can use. Public
exposure ends at the ship's web endpoint. This supersedes earlier milestone
ordering. App discovery, optional Ethereum tooling and wallet custody are
outside this release.

Confirmed locally that Omarchy 4.0.6 reads the selection with
`omarchy default agent` and accepts a prompt through `omarchy agent prompt`.
The installed crash wrapper passes a skill path even when the harness lacks
skill discovery. No agent or crash diagnosis was launched.

The owner preflight and installer now agree on two dedicated-account-only
helpers. The owner already exists, and the installer reads subordinate-ID and
NSS evidence directly. A filesystem fixture failed at the old helper gate,
then passed fresh installation and rerun without `useradd` or `getsubids`.
The hostsetup, hostinstall, remotesetup and signed hostcompat suites passed.
This addresses a reproducible portability issue. A fresh Ubuntu guest and
current rooftop onboarding are still unqualified.

## M522 — Approve an app for a ship across verified routes (2026-11-06)

The user's clarification supersedes M521's proposed new approval on a URL
change. For Application Connections, approve the app/profile for the ship
once. The collector chooses and retries verified local, tailnet and clearweb
origins automatically. Users can find the active URL in details or copy a
public URL for a browser. Enabling public exposure remains a separate action.

Implementation is pending. In 0.3.7 the UI matches approvals and active attempts
by origin, the collector's managed enrollment name includes the origin, and
the receiver [handoff contract](application-handoff-v1.md) binds sessions and
cleanup to one origin. Retain those per-session protections beneath a durable
ship grant and qualify automatic reconnection with compatible receivers. See
the [0.4.0 acceptance cases](platform-roadmap.md#one-ship-several-reachable-urls).

## M523 — Clean Jammy remote setup and first ship (2026-11-06)

The [Ubuntu 22.04.5 VM run](diagnostics/2026-11-06-jammy-remote-runner.md)
supersedes M521's pending clean Jammy install. Its packaged RootlessKit needs
attached network namespaces and its default user manager lacks CPU delegation.
The 0.4.0 development installer adapts to both. Native evidence, a default-loom
ship, uninterrupted second-desktop pairing and revocation, and same-generation
recovery from an actual mid-install interruption passed. The collector also
reconnected after a cold boot. The current rooftop installation, stable upgrade
and other distros retain their separate gates.

## M524 — Rooftop trial confirmed and stock Debian 13 package boundary (2026-11-06)

The operator reported that manual rooftop configuration worked with the separate
0.4.0 trial plugin. This supersedes M523's pending rooftop trial. The user did
not specify whether rooftop took the fresh-install or existing-runner branch.

On a clean Debian 13.7 amd64 guest, the configured archive offered Docker
26.1.5, below the 27.2.0 minimum. Guided setup now identifies the package and
version, states the policy range, gives the inspection command, and explains
how to resume. The terminal and saved status supplied the same guidance.
Real same-attempt retry preserved the desktop key and installer. The package
database and administrator SSH grants matched their pre-test hashes. The next
distro check is Arch/Omarchy. [Debian evidence](diagnostics/2026-11-06-debian-remote-runner.md)

## M525 — Arch first ship and remote-install test boundary (2026-11-06)

Supersedes M524's pending Arch check. Two clean overlays of the official
Arch x86-64 cloud image passed native dependency installation, authenticated
package evidence, restricted-key enrollment and a healthy default-loom bare
ship. Stale image metadata first requested a removed Docker package. A full
guest update and reboot allowed the same attempt, key and frozen helper to
resume. The revised helper explains this recovery in terminal and saved status.
Second-desktop pairing and revocation preserved the running ship and services.
[Arch evidence](diagnostics/2026-11-06-arch-remote-runner.md)

The user explicitly ends this round of distro testing after this pass. The
operator skill now distinguishes verified Ubuntu/Arch behavior, the operator's
rooftop report, Debian's package refusal and host-specific agent assistance.
The Arch guest exercises the runner foundation. A complete fresh Omarchy
desktop installation remains untested in this matrix. Agent Help and skill
installation are next. Stable 0.3.7 upgrade qualification remains a release gate.

## M526 — Desk development and Urgit enter the 0.4.0 scope (2026-11-06)

The user expanded M521's 0.4.0 scope to include app development, rollout and
publication through the agent CLI, with per-ship intent and support for fakes
and project networks. The [architecture proposal](app-development-0.4.0.md)
builds on owner-writable mounts, existing private Lens operations and prior
publisher/receiver qualification. General desk deployment APIs still need
implementation and qualification.

Reviewed Urgit at `7834a4e186998265f5175fd527bc17ea8819c44f`. It supports direct
Smart HTTP pushes into a bound Clay desk and reports Git/Clay correspondence.
Its publish-desk action exports Clay into Git. Treaty publication is a further
workflow. A subscribed distributor desk can propagate later mutations, so release
gating belongs before that mutation. Whole-tree projection requires assembling
projects whose deployable desk lives in a subdirectory.

The proposal uses mounted-desk deployment first and optional Urgit integration,
with typed actions, durable receipts, compilation and readiness observations,
and exact publisher/receiver identities. Explicit promotion as the default and
Urbit-desk/frontend scope are proposed while user feedback is pending. Incoming
GitHub notices require explicit pulls in the reviewed upstream, and GitHub import
updates reject Clay-bound destinations. Direct ship pushes are the initial target.
This turn records source research and architecture. Live pipeline qualification
and the new CLI implementation remain future work. M525's end of distro testing
continues to apply, and Agent Help remains the next implementation slice.

## M527 — Default-closed agent gates and Unix/Clay change alerts (2026-11-06)

The user's answers supersede M526's proposed default promotion policy and its
pending scope decisions. Follow each application's existing release configuration,
including Urgit's rules. The 0.4.0 workflow covers Urbit desks and their frontend
assets. Companion Omarchy plugin release coordination is deferred.

Agent CLI desk commit and publication must start disabled. Add **Setup → App
development and publishing**, where the user configures permissions globally,
per runner, per ship and per installed desk. The architecture uses independent
commit and publication permissions with Inherit/Allow/Block and the most specific
explicit value winning. New development fakes and project members also start
blocked. A purpose label conveys intent while the effective policy controls
permission. User-configured gates take precedence over a convenience CLI flow.

Runner enforcement covers equivalent effects, including an Omabit-managed Urgit
push into a bound desk. Updating a published desk requires both permissions.
Agents inspect policy read-only and follow the app's release settings. Human
Dojo and independently configured Urgit clients use their native authority.
Global policy edits need per-runner rollout status, with offline runners showing
their last applied revision. Queued effects recheck policy before execution.

Observe mounted Unix content against Clay and alert on uncommitted additions,
edits and deletions, including direct editor/copy changes. Distinguish divergence
and unavailable observation, and retain later edits when an earlier commit
completes. The [architecture](app-development-0.4.0.md) and bundled agent skills
now record these requirements. This is a design and skill update. Runtime gates,
Setup controls, change alerts and the new desk CLI remain to be implemented.

## M528 — Application policy layer and Setup editor implemented (2026-11-06)

The user prioritized M527's policy layer and Setup controls, explicitly excluding
active/production runners. This supersedes the pending policy/editor portion of
M527; desk mutation adapters, bindings, purpose labels, alerts and Urgit remain
subsequent work. Work used a temporary isolated source checkout, fake backends,
temporary sockets and a private offscreen desktop. No installed plugin, runner,
service or live ship was operated on or upgraded.

**Setup → App development and publishing** now supports Global, Runner, Ship and
Desk permissions with independent commit/publication gates, default Block and
most-specific explicit inheritance. The runner owns schema48 policy storage,
CAS revisions and authenticated audit receipts. Ordinary settings preserve policy.
Instance deletion/restoration atomically retires its scoped grants; project
targets bind the exact creator and generation. The effect guard rechecks current
policy under the mutation lock, blocks stale/unknown evidence and requires both
permissions for commits to a published desk. Runtime desk adapters remain absent.

Global authoring records a private version1 collector rollout before dispatch.
Offline failures retain intent and the last acknowledged global values/revision.
Retries reconcile lost responses and reject stale or unreviewed revisions.
New enrollments are excluded from old rollout intents; pairing an existing runner
preserves its policy. Other collectors' runner revisions remain authoritative.
Agents use read-only `app-policy [--instance ID --desk DESK]`, with effective values
and source scopes. No generic collector setter or desk commit/publish command was
added. The operator skill forbids bypass through authoring helpers or alternate
transports. The owner-process limitation is documented explicitly.

Validation passed: full Go tests for apppolicy, protocol, managed, client, access,
collection, collector CLI and Omabit helper; focused race tests; Go vet; matching
Linux binary builds reporting schema48; panel model, native plugin and snapshot
installer fixtures; skill validation/package tests. The networkless Quickshell
test exercised all four scopes, inherited/default values, offline/observer
restrictions, malformed/mismatched replies and profile-switch late responses.
The rendered policy editor was inspected. Historical schema target metadata now
names unreleased schema48; historical release qualifications remain frozen. The
schema47→48 fixture preserves identity, settings and receipts with no grants.
Full exact-artifact 0.3.7→0.4.0 live upgrade qualification remains a release gate.

See [policy behavior, support window and test commands](app-development-policy.md).

## M529 — Application policy styling aligned with Setup (2026-11-06)

The user requested a visual pass before reviewing M528. Its transparent component
capture and placeholder sans-serif typography were insufficient style evidence.
The policy editor now uses existing Card, CardTitle, StatusBadge, form controls
and theme tokens, with the same margins and spacing as other Setup panes.
Scope, agent permissions and global runner rollout have separate cards. Fields
use two columns where space permits and stack at narrow widths; supporting
authority/revision information is under Policy details. Normal unapplied status
uses a neutral badge. A shadowed message assignment was corrected so successful
saves display their confirmation. Scoped unresolved-identity status no longer
overrides the runner-level summary when that ship field is hidden.

The existing isolated QML harness now captures an opaque app background and uses
the shell's default monospace type tokens, with controlled dark/light palettes,
width and font scaling. Framed global and desk screenshots were inspected at
688px/default scale and 408px/1.25 scale. Both runs passed all four scope edits,
inheritance, offline/observer restrictions, stale-profile/malformed responses,
save confirmation and horizontal control bounds. Native plugin packaging tests
and diff checks passed. Changes are source/UI-only: no installed plugin, runner,
ship, service or user desktop configuration was changed. This supersedes M528's
initial screenshot as the visual-review reference; backend policy behavior and
its qualification remain as recorded there.

## M530 — Persistent isolated application-policy preview (2026-11-06)

The user requested a persistent manual preview while preserving production
plugins and runners. **Omabit 0.4.0 Preview** opens the actual AppDevelopmentPane
and shared controls in a separately titled, scrollable native window. Its inert
Python adapter provides local/remote runners, a publisher and a project member;
both permissions start blocked. Policies persist across window close/reopen.
The preview can simulate offline and observer connections and switch between
controlled dark/light palettes. The connection selector explicitly refreshes
after changing its adapter binding.

The standalone snapshot is installed at
`~/.local/share/omabit-policy-preview-040`, with a separately named application
launcher. It is not an installed Omarchy plugin and does not reload the shell or
start on login. Private Bubblewrap user/PID/mount/IPC/network namespaces expose
only read-only system dependencies, the source snapshot and preview-owned data.
The native window shares only the Wayland rendering socket; host HOME, runtime,
D-Bus, SSH, keyring, Docker and runner sockets are absent. No live ship, existing
production/DEV/trial040 plugin, runner or service was inspected or changed.

Validation: default blocked state, all four fixture scopes, independent helper
reopen/persistence, retained offline acknowledgment, retry, observer refusal and
unsupported-command/state preservation passed. The actual QML connection
selector exercised offline global save → online retry → disabled observer
editing in a fresh offscreen fixture. Namespace inspection found only private
loopback and the declared mounts/environment. Native rendering, duplicate-open
refusal, identity-checked exact-preview shutdown and state-preserving relaunch
passed. A window-only capture was inspected. These are preview/UI checks, not
additional production runner or release qualification. Human review state is
never reset or used for automated mutation tests. See the
[preview guide](app-development-preview.md) for launch paths and review steps.

## M531 — Policy tree, full-panel preview and desk-creation gate (2026-11-06)

The user's review replaces M529's scope-selector presentation and M530's
standalone-pane/palette-picker preview with a selectable **Global → Runner →
Ship → Desk** tree inside the actual MonitorWindow and Setup navigation. The
preview now includes the real fleet sidebar, uses shared native controls and a
fixed palette, and has no theme control. The remote fixture selector remains
explicitly simulated. Wide layouts place the inspector beside the tree; narrower
layouts stack it below. Permission and text filters highlight matches and their
ancestry while dimming unrelated branches. An ancestor on an allowed leaf's path
retains its own effective decision; highlighting never implies a broader grant.
The map represents saved policy, not unapplied edits or pending global intent.

This also extends M528's policy with an independent **Create new desks** gate.
Global, runner and ship scopes can authorize creation; desk scopes cannot grant
sibling creation. New desks inherit commit/publication permissions unless an
explicit desk policy already applies. Allowing creation grants neither of the
other operations. The effect guard requires exact target identity, current
revision and fresh absence evidence. Actual creation/commit/publication adapters
and CLI actions remain unimplemented and refused. The real tree can display
saved and explicitly named desk scopes; complete installed-desk discovery is
still pending. Preview desk inventory is synthetic and labelled as such.

Policy/capability version2 and schema49 add this gate without reopening existing
permissions. The schema48 migration preserves existing commit/publish choices
and audit bytes, adds blocked global creation with inherited scoped values, and
appends one migration receipt/revision. Corrupt history rolls back. Legacy
collector rollout journals retain their bytes/intent until a new version2 write;
creation defaults closed. A no-runner global save now serializes an empty target
array so its journal can be reopened. Bundled agent skills explain the creation
gate, inherited desk policy, release configuration and absent mutation commands.
Exact-artifact stable upgrade qualification remains required before release.

Validation passed: affected Go suites (including the full managed schema and
historical migration suite), focused race checks, Go vet, matching Linux builds
reporting schema49, native plugin/skill packaging tests and skill validation.
The existing QML harness passed wide and narrow/enlarged-font checks. A new full
panel harness passed sidebar/Setup navigation, node selection, the single
`~zod` `%base` allow path, creation inheritance, offline fresh-review rollout,
observer restrictions and responsive bounds. Fresh networkless fixtures were
used for every automated policy mutation. Native and offscreen captures were
visually inspected with no QML warnings or errors in the final runs.

The separately installed `omabit-policy-preview-040` snapshot was updated and
reopened on the main desktop through its exact identity-checked launcher. Its
original code is retained under `versions/before-policy-tree-20261106-205507`.
The original policy bytes are preserved as `policy-v1.before-create-desk.json`;
a structural comparison verified every prior commit/publication choice and
rollout value survived, with only the documented new gate/version/revision
changes. Human state was not used for automated mutations. Production, DEV and
trial040 plugins, runners, ships and services were not inspected or operated on.
See [the updated preview guide](app-development-preview.md).

## M532 — Zoomable policy tree and saved-value reload fix (2026-11-06)

The user requested zoom before reviewing M531. The tree now supports 25–200%
zoom through shared −/+ buttons, a percentage reset button, Fit, Ctrl+wheel at
the pointer and focused-tree Ctrl+plus/minus/0 shortcuts. Only the map scales;
the inspector, selection, filters and unsaved permission choices are retained.
The viewport supports scrolling and dragging. A dedicated drag handler prevents
the containing Setup page from stealing the gesture; scaled nodes remain
clickable. Fit respects the viewport and zoom bounds. This extends M531's tree
without changing policy, storage, runner contracts or runtime artifacts.

Native inspection also exposed an existing reload ordering bug: reopening a
saved nondefault global policy could show Block in the controls while the tree
and saved-value labels correctly showed Allow. The editor now loads after its
policy-view and derived bindings settle. A fresh fixture saves nondefault gates,
closes, and reopens the full panel; this regression failed before the fix and
passes afterward. No saved policy was rewritten to repair the display.

Validation passed: full-panel selection/filter/inheritance/offline/observer
checks plus actual Qt pointer/keyboard zoom, anchor stability, scaled clicks,
ordinary-wheel scrolling, dragging without page movement, bounds/reset/Fit and
preserved edits; close/reopen persistence display; the 408px, 1.25-font-scale
policy fixture; native plugin packaging and diff checks. Captures were inspected.
The separately installed preview was refreshed and reopened with policy bytes
verified unchanged. Native rendering completed; Qt Wayland emitted a text-input
focus warning during window focus changes, with no QML load errors. Its previous
code is retained in preview-owned version backups. Production, DEV and trial040
plugins, runners, ships, services and desktop configuration were untouched.


## M533 — Approved feature goal and Agent Help implementation (2026-11-06)

The user approved continuing Agent Help/skills, the gated desk workflow, fleet
URLs and clearweb assistance as an ongoing goal. Work should continue on every
unblocked path. This supersedes M526's proposed dedicated Urgit adapter: Urgit
now needs a qualified workflow and skill guidance, not a separate Omabit software
integration. Release qualification/publication waits until feature work is ready.
Production exclusions and isolated fixtures remain in force.

Agent Help now uses Omarchy's default-agent query and prompt launcher, with a
release-bound operator skill and exact read-only diagnostics. The monitor and
failed/interrupted remote setup expose the action. Setup provides explicit skill
installation, default/no-default guidance, automatic OpenCode destination and
an explicit directory for other agents. Custom/edited skills are never replaced;
reviewed managed updates are atomic and retain the previous copy. No actual agent
or host skill directory was used during implementation testing.

Two packaging gaps were corrected: development snapshots omitted installed
skills, and native bootstrap discarded bundled skills. New bundles declare
SKILLS.json, and native caches verify those files against their pinned archive.
Historical caches without that declaration retain their previous layout. Agent
Help never silently substitutes guidance from another release/profile.

Helper/installer tests, bootstrap/archive binding tests, skill/native packaging
checks and the networkless full-panel interaction fixture pass. The fixture uses
actual help/installation code with a fake agent, tests complete installation of
both skills, no-default guidance and remote-setup context. This is feature-level
qualification, not the deferred stable upgrade or release rehearsal. Other
approved tracks remain pending; see the current 0.4.0 feature progress record.


## M534 — Agent Help packaged checkpoint (2026-11-06)

The clean development snapshot at `88a07698c289e4776d4df20993672cdcea6f91b9`
built successfully with label `agent-help-040`. Its bundle SHA256 is
`eb4f1f990291cc947703faa50b325ec6a40871526a70675a9531df3a21566003`.
All artifact sizes/hashes, private native import/prepare/re-prepare, activated
helper bundle resolution and both complete skill trees passed. This closes the
packaged-feature check from M533, not stable release qualification. Private
evidence is `/tmp/omabit-agent-snapshot-check-9pnt33l9/result.json`; the generated
snapshot is `/tmp/omabit-040-agent-help-snapshot`.

The final full-panel screenshot in `/tmp/omabit-agent-ui-02` was inspected after
the header-width correction. The policy tree regression fixture also passed.
No installed plugin, real agent, host skill directory, runner or ship was used.


## M535 — Desk transport and source-transfer development (2026-11-06)

The Agent Help checkpoint is integrated at `18f3054` in the isolated 0.4.0
worktree. Desk implementation proceeds in `codex/0.4.0-desk-workflow`. The
new `internal/deskwork` package prepares bounded source manifests, hashes,
file-preserving transfer plans, three-tree comparisons and framed source bundles.
Uploads stage outside a pier; incomplete, changed, linked, oversized or corrupt
source is refused. Identical untracked files are preserved without adopting them
for future deletion. Focused source/bundle/request tests pass. No runner endpoint
uses this package yet, so it does not enable desk mutations.

Disposable native probes use the previously pinned Vere/pill, private network,
PID, mount, IPC, user and UTS namespaces and a 4 GiB/no-swap/one-core scope.
The declared scratch allowance is 2 GiB; prior generated piers/input duplicates
are pruned only after clean shutdown, with removal receipts. Retained classified
lab roots measured about 26 GiB before this round, below the 40 GiB limit.

Probe02 establishes create, mount, five generated prerequisites, uncommitted file
absence, an acknowledged commit and revision1→2. It fails the deliberate raw
byte comparison: a text file ending in newline renders from Clay without that
newline. Raw byte equality is therefore insufficient for uncommitted alerts.
The adapter must use the desk's actual mark conversion, or exact previously
observed commit evidence; it must not suppress real edits by trimming arbitrary
files. A mount naming the current revision can remain a live mount, so a stable
comparison baseline requires captured bytes and a revision fence.

Probes01/03/04/06 retain failed expression/observation evidence under
`/tmp/omabit-desk-probe-NN`, and stopped cleanly. Missing-source scries and malformed
Dojo expressions can leave Lens waiting; check existence first, use typed
requests and never interpret transport submission as successful execution.
Probe05 rejected an empty generated request fixture before starting a runtime.
Hoon numeric literals require grouping for larger decimal/hex values; mixed
JSON fields require an explicit list element type. Probe06 verifies a bounded
JSON-as-hex response; the Go decoder refuses unknown fields and trailing data.
The corrected complete protocol probe is still pending.

Source review also shows that cancelling Kiln autocommit does not clear its
saved timer value. Active Behn timer evidence is required; the saved value alone
cannot establish active/inactive status. Treaty is absent from the tested base
desk; publication capability discovery and the real Treaty/frontend/receiver
workflow remain pending. These findings are not a release qualification claim.


## M536 — Typed desk protocol probe passed (2026-11-06)

Probe12 closes M535's initial protocol spike: the actual Go-generated create,
mount, commit and source-publication requests pass on a private fake. Inventory
reports revision1/private before commit and revision3/published after the
autocommit test and explicit source publication. The source normalizer passes
real file bytes as base64 data through Lens's Hoon transform, uses the selected
desk/revision's MIME→mark converter, accepts semantically equal newline variants
and rejects different content. Vere's Unix driver uses text/plain MIME for its
file input; the adapter follows that representation.

An active `/kiln/autocommit` timer is present before cancellation and absent
after it. The pinned pill lacks the convenience cancellation generator: probe10
retains that failed attempt. Typed `%kiln-cancel-autocommit` succeeds in probe12.
A failed Lens read can persist across process restart; probe08 records that
result. The private harness may clear its own failed read job on a verified
stopped fixture, but production adapters must not cancel another session's Lens
job or replay an uncertain mutation. Probe09/11 preserve intermediate type
errors; probe12 is the first complete pass.

The development operation model now validates reviewed policy/desk baselines,
requires both commit and publish gates for published desks, and separates
dispatch, acknowledgement, observed effect and uncertainty. Reopen recovery
never redispatches an incomplete effect. A later revision alone is insufficient
to claim an interrupted commit: exact frozen-source verification is required.
Focused package/policy race tests pass; runner persistence and transport remain
next. See `diagnostics/2026-11-06-desk-protocol.json` for exact inputs, request
source hashes, results and explicit limits. Production/DEV/trial state is untouched.

## M537 — Private runtime desk adapter and fixture isolation (2026-11-06)

Typed Lens requests now have a runner transport bound to the current exact
container process, registered pier, private HTTP port and ship identity.
Standalone managed ships and current/expanded project members use their existing
containment and generation contracts. Payloads use bounded stdin, not argv;
responses are bounded and transport failures never expose raw source or secrets.
The adapter neither retries uncertain effects nor cancels shared Lens jobs.
The caller must still hold policy/lifecycle admission and journal dispatch.

Standalone and current-project fixture tests cover changed identity, process,
ports, peer containers, transport loss and unsupported inputs. The complete
runtimequalify race suite passes in `/tmp/omabit-desk-private-runtime-race-02`.
The first broad run used the repository rather than package working directory;
its missing-fixture failures are retained in the `-01` evidence directory and
superseded by the corrected pass. `scripts/test-go-private.py` builds one package
then runs it with private user/network/PID/mount/IPC/UTS namespaces, without the
installed owner profile, host HOME/run or Docker socket. No host profile was
changed to make fixture tests pass. This is source/fixture validation, not a
managed Docker end-to-end or release qualification claim.

## M538 — Gated creation CLI and schema50 operation journal (2026-11-06)

Schema50 adds a separate typed desk-operation journal with immutable intent,
authenticated actor and append-only mirrored history. Opening validates the
journal; corrupt/missing receipts fail without repair. Duplicate IDs are accepted
only for the same actor and exact intent. Restart marks planned work interrupted
and dispatched/acknowledged work uncertain; it never queues an effect. Unresolved
work blocks another effect for that exact target. Explicit read-only reconciliation
can record that a creation's desired desk exists on the selected runtime, including
after the creation gate was revoked; it does not attribute an external actor's
creation to Omabit. Missing/changed runtime evidence remains uncertain.

`desk-create-v1` exposes typed observation, asynchronous creation and receipt reads.
The worker rechecks operator authority, target/generation, current process,
configuration, policy revision and actual absence before durable dispatch. Source
commit/publication remain unavailable. Receipt reads remain available during an
effect. `omabit desk-inspect`, `desk-plan-create`, `desk-create`, `desk-operation`
and `desk-reconcile` pin enrollment and preserve the reviewed intent across retries.
Agents cannot author policy. New desks inherit existing commit/publication gates.
Skills now describe this exact path and its limits; installed copies were not changed.

Evidence: focused race suites in `/tmp/omabit-desk-private-store-01`,
`/tmp/omabit-desk-private-api-02`, and `/tmp/omabit-desk-private-collection-01`;
CLI offline help/argument checks and skill-package checks pass. The checked-in
schema49 SQL fixture was written by original source e2f6377, not by stamping an
older version onto current state. `/tmp/omabit-desk-original-migration-01` runs the
original writer, current migration/recovery, then the original reader's unchanged
refusal of actual upgraded state. `scripts/test-desk-upgrade.py` reproduces that
functional compatibility path without running any service or ship.

The broad managed run initially found its explicitly pinned development-schema
assertion and the private harness's missing nested Go compiler. The development
target is now50; the frozen RC12 qualification record and historical fixture
coverage are unchanged. The corrected BootDispatch/G02 race run completed all
preceding cases without failures but reached its cumulative nine-minute limit in
its final migration/refusal test (`/tmp/omabit-desk-private-migration-regression-01`).
That final test passes separately in `/tmp/omabit-desk-private-migration-tail-01`.
The interrupted broad run is retained with cancellation evidence; this is not a
claim that the complete managed race suite or 0.4.0 release qualification passed.
The private test harness now optionally supplies a read-only toolchain/module
cache with private build output for tests that compile their own fixture worker.

No production/DEV/trial service, plugin or ship was changed. The human-reviewed
persistent preview remains at its earlier source and retains its saved policies.
Full managed-runtime desk deployment, source bindings/transfers/alerts, Treaty,
frontends, Urgit workflow, fleet URLs and clearweb assistance remain active work.

## M539 — Live desk discovery and active-autocommit evidence (2026-11-06)

`desk-discovery-v1` adds operator-only live desk names to the runner/collector and
`omabit desk-list`. It pins the current process, configuration, ship/project
generation and enrollment; malformed, stale, duplicate or mismatched responses
fail visibly. Setup's Discover desks button reads only the selected ship, marks
installed names, retains saved scopes and unsaved permission edits, and grants
nothing. Its temporary observations clear on policy refresh/profile change.
The full policy editor passes wide and narrow offscreen fixtures; the wide
screenshot was inspected. The persistent human preview was not updated.

Probe14 passes the actual Go-generated desk list, immutable-revision file paths
and active-autocommit detector (inactive → active → cancelled) on the isolated
native fake. It also repeats the create/mount/commit/mark-conversion/publication
protocol checks and shuts down cleanly. Probe13's desk/file reads passed but its
first autocommit query hit Hoon type inference; that failure remains recorded.
The corrected query reverses the path before refining its shape. Unknown output
is never treated as inactive. Fixture wrappers reject bad paths/inventories and
compiler errors; server/collection race fixtures cover changed selections,
capabilities, generations and failed observations. Evidence and limits are in
`diagnostics/2026-11-06-desk-discovery.json`.

No persistent schema change accompanies discovery. Source bindings/transfers,
canonical Unix/Clay alerts, gated commit/publication, managed-runtime end-to-end,
Urgit, fleet URLs and clearweb assistance remain active work. This checkpoint
supersedes M538 only for its pending live-desk-discovery item.

## M540 — Canonical source comparison and targeted Clay delta (2026-11-06)

The canonical source engine compares SHA256(jam([mark noun])) with each selected
historical desk's MIME conversion. It batches at most128 files/32 MiB, binds reads
to exact source hashes, and rechecks Unix contents/directory identity and Clay's
revision before returning status. It distinguishes shared-baseline changes and
divergence; without a baseline it says different, not an invented direction.
Unknown autocommit remains unknown. Failed/unsupported conversions never produce
a synchronized result. The runtime restricts source reads to ordinary selected
top-level directories inside the registered pier, with the existing process guard.

A typed direct commit constructor sends a bounded MIME delta to the named Clay
desk, guarded by the reviewed revision/hash. It preserves unspecified files and
avoids treating a mutable Unix mount alias as proof of commit targeting. This
supersedes M535/M536's native alias-commit path as the intended agent commit
adapter, while leaving native human |commit available. The runner must still
apply current policy, durable dispatch and exact post-commit source verification;
no commit API/CLI is enabled by this checkpoint. A commit is limited to32 MiB of
changed bytes; larger sources can stage but need a supported bounded deployment.

Probe19 passes canonical comparison and direct add/replace/delete, then source
publication and clean shutdown. Probe16 is the first canonical-batch pass after
probe15's type-inference failure. Probe17's direct delta passed but its verifier
read the old revision; probe18 corrected that, then disproved the assumption that
raw invalid Hoon source necessarily rejects a Clay commit. Clay accepted it as
revision5. Source acceptance, application compilation and Gall readiness must be
reported separately and follow the developer's release checks. Probe19 explicitly
records that behavior rather than relabeling it a compilation pass.

The deskwork race suite and private runtime fixture checks pass. Exact native
inputs, hashes, failures and limits are recorded in
`diagnostics/2026-11-06-desk-canonical.json`. The experiment reuses only its own
clean-stopped fake in capped private namespaces; retained native-probe data was
about2.5 GiB before probe17, with 2 GiB additional scratch declared per run and
about26 GiB in the two shared evidence roots. Production, installed previews,
real agents, runners and ships remain untouched. Persistent source bindings,
agent commit/publication endpoints and the remaining 0.4.0 tracks are still active.

## M541 — Shared ship purpose and desk workflow settings (2026-11-06)

Schema51 adds revisioned ship/desk development settings and append-only author
history, separate from all policy grants and operation receipts. Project targets
include the exact current generation. Defaults are descriptive: standard ship,
Unix source directory equal to desk name, no publication method/checks. The Setup
card records source workflow, directory, optional Urgit repository/branch, release
checks and instructions; it never executes hooks, stores tokens, configures Urgit
or changes permissions. Agents read both ship and desk with `desk-settings` on
`desk-workflow-settings-v1`; Setup owns authoring. A source name does not establish
a native mount mapping. This supersedes the pending descriptive-binding/purpose
portion of M540; synchronization baselines and source operations remain pending.

The actual schema50 writer at9f45b42a produces the retained SQL fixture, including
a dispatched desk receipt. Schema51 preserves original registry/policy/journal,
reopens idempotently and rolls failed migration back. That schema50 reader refuses
actual51 state unchanged. Source proof is in
`diagnostics/2026-11-06-desk-settings.json`; exact-artifact release qualification
is still deferred. The first SQL fixture unit test incorrectly let SQLite create
its database with non-private permissions; the runner refused it correctly. The
fixture now creates its file0600 before import, matching real writer behavior.

Private managed/collector race fixtures pass capability, identity, current project,
concurrent revision, corruption and response-integrity checks. Wide/narrow UI
fixtures pass edits, validation, saves without permission changes, observer refusal,
mismatched targets and a late reply across profile changes. A transient QML warning
on ship-to-desk selection was fixed. Both screenshots were inspected; the full
Setup/sidebar/zoom/reopen suite passes. The packaged skills document settings as
metadata and preserve native Urgit release rules. No installed preview, production,
DEV or trial runner/plugin/ship was changed. Source status, gated commit/publication,
Urgit qualification, fleet routes and clearweb assistance remain active work.

## M542 — Source-status API, canonical baselines and change alerts (2026-11-06)

`desk-source-status-v1` connects the tested canonical source engine to the runner,
collector, `omabit desk-source-status` and the selected Setup desk. The runner reads
only that desk's saved Unix source configuration on a pinned current runtime and
exact project generation. Operator access is required; injected source paths,
changed settings/runtime, stale/malformed responses and Urgit workflow guesses are
refused. Reads perform no source writes, commits or publications and preserve all
policy grants. Autocommit unknown remains unknown; source equality is not app
compilation/readiness. Replies contain counts, hashes and at most256 sorted file
details, not source bytes or absolute paths.

Schema52 stores a canonical baseline only after Unix/Clay match. It is bound to
runtime, configuration, workflow revision, source name and target generation.
Differences and failed reads preserve the prior baseline. A changed binding starts
unclassified rather than falsely attributing direction. Schema51's actual writer
at323ed81 supplies the SQL fixture with workflow settings, policy and a pending
desk receipt. Migration/reopen preserve those records, create no invented baseline,
roll back on failure and refuse corrupt caches. The original51 reader refuses
actual52 state unchanged. See `diagnostics/2026-11-06-desk-source-status.json`.

Core/protocol race tests, private API/collector/migration fixtures and wide/narrow
UI checks pass. Screenshots were inspected. The alert differentiates synchronized,
Unix changes, Clay changes, divergence and unknown-direction differences. Failed
refreshes retain prior details explicitly marked unavailable; deactivation/profile
changes reject late responses. Only the selected saved workflow is refreshed,
90 seconds after completion or manually; no fleet-wide watcher was added. This
supersedes the pending source-status/baseline/selected-alert portion of M540/M541.
The underlying native canonical engine was qualified in M540; this new composition
still needs managed-runtime end-to-end coverage. Staged transfer, gated commits,
Treaty, receivers/frontends, Urgit, fleet origins and clearweb remain ongoing.
Installed preview/production/DEV/trial plugins, services and ships were untouched;
release qualification remains deferred.

## M543 — Frozen source staging, receipts and recovery (2026-11-07)

`desk-source-staging-v1` adds operator-only plan/upload/status/discard commands.
Plans bind runner, ship/desk and project generation, saved workflow revision,
source digest/count/size and a retained random ID. Source streams into a private
versioned runner cache outside all piers, with no runtime call or gate change.
Stopped ships with blocked gates can stage. Exact replays rehash the original
cache and return its receipt; changed source/selection/owner and corrupt content
fail visibly. Limits are8192 files,32MiB/file,256MiB/tree,2MiB manifest and
16 cached stages/1GiB source. SQLite remains52; older clients negotiate capability.

Deterministic pending IDs and retained intents permit cleanup after interrupted
uploads. Explicit discard checks actor/digest, serializes cache writes and holds
the runner mutation lock; any unresolved effect on the target retains the cache.
Source is removed before its intent so interrupted deletion remains recoverable.
An empty pre-intent/post-removal directory can be removed without traversing
unbound files. Unknown, contradictory or malformed metadata stays for review.
Original source directories, mounts and Clay are never discard targets. Client
DELETE requests now also disable implicit transport replay and label lost replies
uncertain. A staged receipt proves transfer only, not commit or compilation.

Private runner/collector/client race fixtures and core/CLI tests pass, including
stopped ships, frozen streams, lost acknowledgements, no implicit replay, exact
receipt binding, ownership, limits, partial upload/discard and unresolved-effect
refusal. Skills and packaging checks pass. See
`diagnostics/2026-11-07-desk-staging.json`. This completes the source-transfer
portion pending in M542; managed-runtime end-to-end, gated commit/publication,
Urgit, fleet routes and clearweb assistance remain active. No production/DEV/trial
plugin, runner, ship or persistent manual preview was changed. Release qualification
remains deferred.


## M544 — Gated staged commits and verified source ownership (2026-11-07)

`desk-commit-v1` connects private stages to read-only review, saved commit plans,
one journaled effect dispatch, source verification and read-only reconciliation.
CLI commands are `desk-review-commit`, `desk-plan-commit` and `desk-commit`; the
latter accepts an unchanged saved plan file or JSON. Review binds the stage,
runtime/project generation, workflow/policy revisions, source digest, pier object
and current Clay revision/hash. Details are bounded with exact counts. No-op and
conflicted plans cannot dispatch. Both commit and publish gates are required for
an already published desk; private desks require commit. All remain closed by
default. Changes between queueing and dispatch block the effect.

The planner pairs previous raw source with its verified canonical Clay ownership.
It preserves unspecified/generated files, shares identical unowned files without
adopting them, and refuses external edits or differing unowned collisions. Only
owned unchanged files can be replaced/deleted. Converter changes are distinguished
from raw byte equality. New marks can enter the delta without pretending an old
converter exists; post-verification normalizes the whole staged source using the
resulting revision and compares the entire Clay tree. One delta is bounded at
32MiB changed source/8192 changes. This proves source acceptance, not compilation,
Gall activation, frontend readiness or publication. Existing-desk ownership/adoption
beyond matching/shared files still needs workflow treatment; it is not silently
inferred from a permission grant.

Schema53 stores immutable deployment reviews and source ownership linked to the
existing desk journal. Verified completion and ownership save atomically; failures
retain uncertainty and prevent another target effect. Replays return the original
receipt without requiring another dispatch or a retained completed stage. Recovery
uses the original runtime and retained stage, never repeats the write. Ownership
survives runtime restart/move of the same pier object, source name and target
generation; changing pier/source starts without ownership. Typed runtime selection
also now uses NativeShip for keyed ships rather than the legacy fake fallback;
this is fixture coverage, not a live keyed-ship test.

Core/protocol/CLI checks, private API/collector tests and all desk regression race
tests pass. Queued policy, workflow, source, pier and stopped-runtime changes block
commits. Lost acknowledgements and failed verification reconcile without repeating
the effect. Original schema52 writer state preserves policy, pending receipts,
workflow and canonical baselines; reopen/rollback pass and that actual older reader
refuses schema53 unchanged. See `diagnostics/2026-11-07-desk-commits.json`. One dispatch
test initially used a desk-level creation Block, which Setup correctly refused;
the corrected fixture inherits creation from Ship and passes.

This supersedes the pending staged-commit adapter portion of M543. Assembled native
workflow qualification, compilation/publication/frontends/receivers, Urgit, fleet
origins and clearweb assistance remain active; no release qualification was run.
Installed production/DEV/trial plugins, services, ships and manual preview stay
unchanged. Before the next native run, a capacity check found44,371,935,232 bytes
across the two evidence roots and Omabit scratch. The obsolete generated
`/tmp/omabit-040-gocache` was classified and its12,620 cache files (4,759,076,864
allocated bytes) removed using `/tmp/omabit-desk-cache-pruning-20261107.json`.
Current Go cache, source, migration fixtures and all piers were retained. Recheck
capacity and declare the next scratch allowance before starting a runtime.

## M545 — Native staged-commit composition and bounded retention (2026-11-07)

Probe21 passes the real disposable kernel + managed API/store + collector/Unix
transport workflow with schema53. Creation and commits are closed by default;
explicit fixture-user grants enable only the requested scopes. Live discovery,
canonical synchronized/Unix-change alerts, source staging, first commit preserving
scaffold, subsequent add/update/delete, deliberately lost acknowledgement and
read-only reconciliation after permission revocation pass. An uncertain stage
cannot be discarded. Exact replay never repeats the commit. A native fixture
publication makes later commits require both Commit and Publish. The ship exits
cleanly. Source acceptance remains distinct from compilation/readiness.

The guard admits only the private namespace, loopback, recorded scope and owned
Vere executable/start time. Effects have retained dispatch/acknowledgement rows;
reuse checks those and terminal managed receipts. Probe20 failed before runner
state or any desk operation because the harness omitted mkdir; corrected in21.
M544's pending native composition is superseded by this evidence. The lifecycle
backend is still synthetic: managed Docker, SSH, activation, Treaty, frontends,
receivers and Urgit are not qualified by this run. See the
`diagnostics/2026-11-07-desk-native-workflow.json` evidence and limits.

Before the run, classified duplicate immutable Vere/pill copies and generated
cache/superseded successful test binaries were pruned with recorded identity/hash
checks. Source, original migration fixtures, logs/results and all piers remain.
Removal manifests are `/tmp/omabit-desk-runtime-inputs-prune-plan.json`,
`/tmp/omabit-desk-generated-pruning-20261107.json` and
`/tmp/omabit-desk-runtime-inputs-prune-19-20.json`. Canonical immutable inputs stay
in runtime-inputs-20261025-07. A read-only DEV disk-size query returned a lower
bound because18 overlay work directories were inaccessible; sudo was unavailable.
No permissions or DEV state changed. Admission21 reserves1GiB for that uncertainty
and2GiB additional scratch, totaling42,542,821,376 bytes under40GiB. Re-measure before
another runtime. Retain at most one active experiment. Production, installed DEV,
trial and manual-preview plugins/runners/ships remain untouched. Publication,
activation, existing-desk adoption, Urgit, fleet URLs and clearweb continue; release
qualification remains deferred.

## M546 — Separate app-stage status and native release workflow (2026-11-07)

`desk-application-status-v1` and `omabit desk-app-status` now expose bounded,
read-only source/installation/Gall/Treaty/Docket observations through the existing
runner/client/collector selection. Runtime/project/configuration and policy
revisions are checked, and unchanged Clay brackets the observation. Unavailable,
stale or malformed data fails visibly. Expected agents and observed live agents
remain separate; source revision is not a compiled-agent revision. No overall
readiness flag, permission grant, persistent format change or effect is added.
Schema remains53. Core/CLI checks, private API/collector race cases and the private
runtime adapter race fixture pass. The first collection test invocation could not
open a sandboxed Unix socket; rerunning with private temporary sockets passed.
The runtime fixture initially refused the host owner profile; the established
private HOME/network/PID/mount harness passed without changing that profile.

Probe32 passes native source publication, local activation, exact Treaty hash and
matching current/realized docket metadata. Probe34 repeats those through the real
app-stage API/client/collector on a fresh isolated fake and additionally uploads a
local Ames glob via native authenticated Docket HTTP. The exact frozen HTML is
served. Upload commits a new docket hash to Clay, advances Treaty and correctly
conflicts with old staged source ownership. An intentionally invalid installed-agent
update returns HTTP500 `crud!`; a subsequent read has the original revision/hash.
The unacknowledged effect stays uncertain. Neither a retry nor a proven rejection
is inferred. Probe34 exits cleanly and its test passes this explicit uncertainty
case, but ordinary mutation-resume must refuse its pier. Probe33 stopped on the
same response before read recovery; its original evidence remains unchanged.

Earlier probes fixed three observation bugs: asking a union mark for only one
variant can stall Lens; read the noun then validate its shape. Hoon's example mold
returns its default rather than validating docket input; use the real docket mold.
Treaty sovereign inspection needs the standard library in its subject. These are
covered by the successful native observations. Malformed fixture docket source29
was corrected before the successful activation runs; its receipt was not cleared.
See `diagnostics/2026-11-07-desk-application-stages.json` for pins, hashes and limits.

Source/Treaty/local-activation constructors and runtime wrappers are implemented,
but their effects still have no public agent API in this checkpoint. Native upload
is qualification only. Gated release receipts/CLI, source adoption, receiver checks,
Urgit, fleet routes and clearweb assistance remain active. The lifecycle backend
is synthetic, so this is not managed Docker/SSH or release qualification. Skills
now distinguish source acceptance, live agents, metadata and actual frontend checks.
Installed production/DEV/trial plugins, runners, ships and manual preview stay unchanged.

Retention removed only classified rebuildable caches/binaries, duplicate immutable
inputs and superseded generated fake piers10/29 with removal manifests. Original
source, failed receipts/runner databases, compact logs and input identities remain;
current failure33 and fresh34 piers are retained. The pruned unresolved29 receipt
still records uncertainty; pruning is not reconciliation. Admission34 measured
42,720,210,944 bytes including2GiB next scratch and1GiB uncertainty reserve for
inaccessible DEV overlay work directories. Re-measure before another runtime.

## M547 — Gated local activation and publication (2026-11-07)

`desk-release-v1` exposes read-only review, saved release plans and a single
journaled dispatch for local activation, Clay source publication and Treaty
publication. CLI commands are `desk-review-release`, `desk-plan-release` and
`desk-release`. Ordinary operation inspection and read-only reconciliation also
cover these receipts. Reviews bind source revision/hash, application observation,
physical pier, runtime/project generation, both workflow revisions and policy.
Source/Treaty publication must match the configured method. Local activation
refuses to replace a remote subscription, including in the native expression.
Commit gates activation, plus Publish when already distributed; publication
requires Publish. Defaults remain blocked. Policy/settings/source changes before
dispatch block the effect. Exact replay returns the original receipt without a
second write. Lost responses remain uncertain and recovery never repeats effects.

Configured publication checks require ordered, named, passing attestations for
the reviewed Clay hash with bounded evidence. These are caller-supplied results,
not commands executed or independently verified by the runner. Activation accepts
no check attestations, because installation can be needed to perform those checks.
Observations distinguish source acceptance, Gall liveness and Treaty/Docket state;
none proves the compiled code version or whole-application health.

Schema54 adds immutable release reviews/checks linked atomically to the existing
journal, with strict reopen validation. The actual original schema53 writer at
3a5a5c5 supplies a preserved fixture; migration retains policy, receipts, workflow,
baselines, deployments and ownership. Reopen and invalid-state rollback pass;
that original reader refuses54 without modifying state. Frozen historical release
fixtures remain unchanged; only the development target advances. Core/CLI,
collector and all managed desk regression race checks pass. One collection fixture
used a stale interval shorter than the four-minute operation bound; correcting its
age passes without product changes.

Fresh isolated native probe35 passes gated source publication, exact replay,
published-desk dual gates, local activation and Treaty publication through the
managed API/store/client/collector. The native frontend fixture still verifies
exact served HTML and the ownership conflict after Docket's Clay update. All35
effects are acknowledged and shutdown is clean. It does not repeat M546's invalid
agent mutation;34 remains uncertain and cannot be resumed for ordinary mutations.
See `diagnostics/2026-11-07-desk-gated-releases.json`. This supersedes M546's pending
public release adapter; frontend upload remains a private test helper. Source
adoption, receivers, Urgit, fleet routes and clearweb assistance remain active.
The lifecycle backend is synthetic; managed Docker/SSH and release qualification
are not claimed. Installed production/DEV/trial/manual preview remain unchanged.

Before35, rebuildable Go cache was pruned with
`/tmp/omabit-desk-release-cache-pruning-02.json`; duplicate immutable35 inputs were
removed with their recorded hash/inode checks. Native35 admission included2GiB
scratch and1GiB unmeasured-DEV reserve at42,838,761,472 bytes. Subsequent regression
admission with1GiB scratch totals42,392,866,816 bytes. Re-measure before another
runtime; retain compact evidence and original migration writers/fixtures.

## M548 — Explicit existing-source ownership adoption (2026-11-07)

`desk-source-adoption-v1` supplies read-only review, saved plans and atomic metadata
adoption through runner/client/collector and CLI: `desk-review-adoption`,
`desk-plan-adoption`, `desk-adopt`. Agents stage only the files they intend to
manage. Every selected file must already match current Clay after mark conversion;
missing, different or unknown conversions conflict. Adoption performs no ship or
Unix write. It needs Commit and, on a distributed desk, Publish. Defaults remain
closed, and an unresolved earlier effect blocks it. Runtime/project generation,
source/pier, stage, workflow/policy revisions and Clay revision/hash are bound.

Only selected paths are newly owned or refreshed. Unselected existing ownership
retains its prior baseline, so later external drift still conflicts. Unspecified
scaffold is never adopted. A subsequent commit must include all owned files it
intends to retain; omission of an owned unchanged file is a reviewed deletion.
Identical ownership needs no adoption. Adoption cannot replace a lost effect's
reconciliation or bypass the application's configured source/release workflow.

Schema55 adds immutable adoption records. Receipt and ownership commit atomically
with no intermediate external dispatch. Replay returns the original observed or
blocked receipt, even after stage discard or policy/source changes. Client code
refuses a partial response and directs inspection of the original operation ID.
Ownership rows preserve their existing encoding and validate against their original
commit or adoption record. Missing dependent evidence is an error, never an empty
baseline. Reopen rejects corrupt, orphaned and rebound records; transaction failure
leaves no partial journal or ownership.

The original schema54 writer at829cfed preserves deployed ownership, settings,
canonical baselines, pending and completed receipts and publication evidence across
migration to55. Reopen and rollback pass; the actual older reader refuses55 without
changing state. Core/CLI, collector race and the full managed desk regression race
suite pass. Early storage fixtures omitted registering the synthetic ship before a
grant, and attempted to insert corruption with foreign keys enabled; corrected
fixtures pass without relaxing product checks. The system Python skill validator
passes; the initially selected Python lacked its YAML dependency.

Native probe36 resumes only the clean private35 pier, with recorded device/inode
transfer. It explicitly selects `sys.kelvin`, verifies the closed gate and grant,
observes unchanged Clay and zero commit calls during adoption, discards its stage,
and replays the exact receipt. Four other scaffold paths remain unowned. Subsequent
staged changes, lost-ack recovery, default publication gates, activation, Treaty,
frontend delivery and the external docket conflict pass. All native effects are
acknowledged and shutdown is clean. See
`diagnostics/2026-11-07-desk-source-adoption.json`. Native36 used the pre-final
client build; the final added rejection of impossible partial adoption responses
has separate collector race coverage. The lifecycle backend remains synthetic.
This supersedes M544/M546/M547's pending source adoption, not receiver, frontend
adapter, Urgit, fleet URL, clearweb or release qualification work.

Retention pruned only the superseded generated33 pier after verifying its exact
scope inactive, rebuildable Go cache, obsolete managed desk fixture binaries and
verified duplicate36 runtime inputs. Failure34/current36 piers, all source,
original migration binaries, failed receipts and compact evidence remain. The33
uncertainty is unchanged, not reconciled. Removal manifests are
`/tmp/omabit-desk-superseded-probe33-pruning.json`,
`/tmp/omabit-desk-adoption-cache-pruning-01.json`,
`/tmp/omabit-desk-superseded-fixture-binaries-20261107.json` and
`/tmp/omabit-desk-runtime-inputs-prune-36.json`. Native36 admission measured
42,773,184,512 bytes with2GiB scratch and1GiB unmeasured-DEV reserve. Re-measure
before another runtime. Production/DEV/trial runners, ships, installed plugins and
the persistent manual preview remain unchanged.


## M549 — Native receiver propagation and bounded launcher (2026-11-07)

Typed receiver primitives select a continuing publisher, inspect pending Kiln
subscriptions even before source arrival, and refuse an occupied Clay desk or
subscription inside the install expression. A separate read distinguishes a
reserved revision-zero desk from received source. Expected hashes are verification
conditions, not version pins; later publisher commits continue to propagate.
No public receiver mutation API or CLI is introduced yet. Schema55 is unchanged.

Native40 passes the assembled publisher workflow and both fresh receiver paths.
Each receiver has the exact initial publisher hash, a live fixture Gall agent,
matching Docket state and exact served HTML. A later publisher commit reaches both
receivers with no second install. Both received desks stay private. All native
effects are acknowledged and all three peers stop cleanly. Core receiver tests,
private runtime adapter race checks and offline recovery/storage tests pass.
See `diagnostics/2026-11-07-desk-native-receivers.json`. This supersedes M548's
pending native receiver verification, not its pending public rollout/frontend
adapters, Urgit, fleet URL, clearweb or release qualification work.

Failures remain recorded:37 hit its4GiB scope memory cap during receiver boot,
before installation.38 was interrupted before its receiver phase; no completion
receipt exists and its later absent transient scope is not proof of clean shutdown.
An external conservative1GiB storage preflight for38 failed by1,728,512 bytes, but
shell fallthrough still started the launcher with its smaller768MiB cap. A later
measurement charged42,956,808,192 bytes including the estimated1GiB DEV reserve,
7,135,232 bytes over budget. The restarted publisher had allocated another epoch.
Do not describe the admission as passed or the shutdown as clean.39 stopped
cleanly after its first acknowledged receiver install: the old observer rejected
revision zero before it could verify arrival.40 fixes that actual transient state.

Admission now executes inside the launcher before copies/transfers/spawn, so shell
error handling cannot bypass it. Known partial DEV measurement is explicit; all
other measurement failures refuse. Offline tests prove refusal has no output,
copy, transfer or spawn. Aggregate periodic monitoring was added after40 and is
not claimed as native40 coverage. Host exceptions now retain an interruption
record. Three-peer capsules use6GiB with8GiB available admission, one CPU and no
swap; all original namespace/production exclusions remain.

Retention removed only verified duplicate38/39 runtime inputs, the superseded
interrupted38 and clean39 generated fake piers, and the old development-loop Go
build cache. Full34 failure and current40 piers, source, binaries, receipts and
original migration fixtures remain. Manifests:
`/tmp/omabit-desk-runtime-inputs-prune-38.json`,
`/tmp/omabit-desk-receiver-failed-bootstrap-retention.json`,
`/tmp/omabit-desk-receiver-superseded39-piers.json`,
`/tmp/omabit-desk-runtime-inputs-prune-39.json`.
Native40 admission charged42,621,595,648 bytes including2GiB additional scratch and
the1GiB DEV estimate. Re-measure before further work. Production/DEV/trial runners,
ships, installed plugins and the persistent manual preview remain unchanged.


## M550 — Fleet route and standing approval foundations (2026-11-07)

The pure shiproute model binds runner, instance/project generation, ship name and
verified pier continuity. It validates bounded unique origin registries and orders
eligible local, tailnet, explicitly selected LAN and HTTPS public routes. Remote
localhost is not a local collector route. Disabled routes are excluded; bounded
unauthenticated identity probes can fall back after failures or wrong identities,
refuse redirects, and never issue sessions or replay writes. Tailscale parsing
uses current local daemon state, actual full DNS name and listener/interface
compatibility. No Tailscale command, firewall change or peer inventory collection
is enabled yet. Official CLI/MagicDNS/status-source research is linked in
`fleet-access-0.4.0.md`; no assumption that installing Tailscale exposes loopback.

Collector format1 `application-ship-approvals.json` records explicit user choices
per ship incarnation and exact plugin/profile/helper, independently of URLs.
Old per-attempt consent does not become a standing grant. A changed helper needs
another explicit choice and replaces the logical grant; the old helper is not
left independently approved. Revision-checked revoke preserves existing attempt
and cleanup receipts. Default/missing state denies, history is contiguous and
checked against current entries, unknown/duplicate/corrupt state refuses unchanged,
and repeated identical choices at the current revision do not create another row.
The existing owner-session authority remains broad, not desk-scoped.

Private route race checks pass at `/tmp/omabit-ship-route-core-03` and approval
storage race checks pass at `/tmp/omabit-application-ship-approval-01`. They cover
network eligibility/fallback, identity mismatch, redirect refusal, listener/DNS
validation, pier/project/helper isolation, reopen/revoke, stale writes, unchanged
legacy attempt bytes and corruption/symlink refusal. The first route harness
attempt could compile but sandbox nesting refused its network namespace; the
same isolated fixture passes when launched with the required namespace permission.

This changes M549's wholly pending fleet work into internal foundations, not a
finished feature. Runner route persistence/API, collector binding verification,
standing-approval UI/CLI, automatic reconnection and receiver compatibility remain
unimplemented. Current internal methods do not authorize a cached approval to be
used later: the new connection path must recheck current grant and ship identity
under its attempt-publication lock. No installed collector/runner/plugin state
was touched, and no automatic login or public exposure is enabled. Schema55 stays
unchanged; release qualification is still deferred.


## M551 — Gated receiver installation and native API verification (2026-11-07)

`desk-receiver-v1` adds read-only review/planning/status and a saved-plan install
through CLI, collector, client and runner. The unused local desk, continuing
publisher/initial expected hash, target/project generation, runtime, pier and
workflow/settings revisions stay bound. Projects require a current publisher
member of the same project. Existing Clay or pending Kiln names refuse; Urgit
workflow bindings are not overwritten. Create desk at ship scope and Commit at
the receiver desk must both allow; Publish is unnecessary because the receiver
stays private. All gates default closed and are checked again before dispatch.

Schema56 stores the immutable review atomically with its operation. Exact replay
returns the same receipt; failed/lost replies remain uncertain, never retried.
Read-only reconciliation requires current operator authority but no effect gate.
It verifies the original publisher/hash/private desk/live subscription and pier.
Status distinguishes pending revision-zero source from arrived Clay and app stages;
none of these facts substitute for application-specific health checks. Existing
subscriptions continue native updates after agent permissions close.

The actual schema55 writer atc912982 produces the checked-in fixture. Migration
preserves registry, policy, workflow, source/deployment ownership, release and
adoption history. Reopen, rollback, new journals and unchanged old-reader refusal
pass. All desk regressions, receiver/schema/store and client fixtures pass race
checks. Core/protocol race, CLI private regression and updated offline help checks
pass; both skills validate. The initial API fixture used the wrong gate scope and
the first migration fixture reused an unresolved desk; corrected fixtures preserve
both intended refusals. Outer-sandbox CLI socket refusal is not a product failure;
the private namespace suite passes. See `diagnostics/2026-11-07-desk-receiver-api.json`.

Native41 passes the assembled publisher workflow and public receiver API path for
~nec and ~bud. Both defaults dispatch zero mutations. ~nec completes normally;
~bud's real acknowledgement is deliberately discarded, producing uncertainty,
then read-only recovery succeeds after revocation. Both receive exact initial
and next-version source, live fixture agent/Docket state and served HTML with
one install each. All three children exit cleanly. This supersedes M549's pending
public receiver adapter, not its pending frontend upload or lifecycle/installed
qualification. No new public frontend upload, Urgit adapter, fleet connection or
clearweb exposure is claimed. Release qualification remains deferred.

Storage correction: migration01's declared300 MiB omitted enough archive/build
allocation. A post-run measurement charged43,034,075,136 bytes including the1GiB
unmeasured-DEV reserve,84,402,176 over40GiB. No native runtime launched at that
point. The rebuildable isolated Go cache was removed immediately; migration now
performs internal1200MiB admission before output/copies/build. Native41 admission
charged42,919,972,864 bytes including2GiB scratch. Retention preserved all originals
and receipts: clean completed40 piers were retired after exact-scope inactivity
verification; its compact proof remains. Failed migration successor source and
an API binary are retained losslessly compressed; duplicate original55 writer
source in migration02 explicitly links the byte-identical retained migration01
source. Do not remove that dependency. Full failure34 and current41 piers remain.
The diagnostic lists exact removal/retention manifests. Production/DEV/trial and
the persistent manual preview stay unchanged. Re-measure before further builds.


## M552 — Native frontend artifacts and empty source files (2026-11-07)

The internal frontend artifact format binds the complete sorted file/MIME map,
with 2048 files/32 MiB limits, required index.html and a reserved desk.js name.
Native Docket's multipart representation loses a terminal NUL; the adapter refuses
such files rather than silently corrupting them. Embedded/leading NULs and empty
files are supported. MIME types use a deterministic table independent of the host.
A pure Lens calculation predicts Docket's exact glob hash before any upload.
Native MIME is a single path atom containing the full header, including its slash.

Native48 passes against the pinned Vere/pill: the predicted and uploaded glob hash
match, all three served assets have exact bytes and MIME types, and Clay advances
from revision6 to7. The generated docket correctly conflicts with older source
ownership. The same capsule verifies empty source normalization, commit and
subsequent comparison; empty base64 now constructs zero-length octs directly in
all three source primitives. All effects are acknowledged and shutdown is clean.

Earlier42–44 hash construction failures occurred before upload. Native45 performed
one acknowledged upload whose bytes were correct but whose predicted hash was
wrong. Read-only diagnostic46/47 transferred that stopped fake, confirmed exact
HTTP bytes and discovered the MIME representation. No upload was repeated; the
original failed verification is retained. Fresh48 validates the correction.

Internal Eyre upload sends one bounded request without redirect/replay. The private
runtime HTTP transport admits only fixed identity/login/logout/upload endpoints,
keeps credentials out of argv/files/logs, and checks runtime identity around each
call. Entire Eyre and focused runtime transport/frontend race suites pass in private
namespaces; core artifact race tests and six capsule Python tests pass. The actual
Docker invocation remains unqualified. See
`diagnostics/2026-11-07-desk-frontend-artifacts.json` for hashes, failures and retention.

This completes the native artifact spike, not the public frontend API/CLI or its
policy/journal integration. Metadata/hash verification is not application health.
Schema56, installed plugins/runners and the manual preview remain unchanged.
Only classified generated inputs, inactive completed piers and rebuildable caches
were retired; original failures/receipts and compressed binary sources remain.
Admission refusals prevented launches before retention made room under40GiB.
Re-measure before the next build/runtime. Release qualification stays deferred.


## M553 — Gated frontend upload API/CLI and durable recovery (2026-11-07)

`desk-frontend-v1` connects the staged build artifact to read-only review/planning
and one saved-plan upload through CLI, collector, client and runner. Reviews bind
stage target/digest, MIME artifact digest, expected native glob hash, source revision,
source excluding desk.docket-0, settings at both levels and exact runtime/pier.
Only current local own-ship Ames globs on the Unix workflow are supported. Urgit,
remote subscriptions, HTTP globs and unchanged advertised hashes refuse.

Both Commit and Publish gates apply even for private Clay: the upload immediately
changes served assets and native Docket HTTP has no atomic expected-revision or
publication precondition. This tightens the typed adapter beyond M546's native
classification. Fresh worker checks run under the policy/lifecycle lock, followed
by an additional source/app/Docket check after ephemeral authentication. External
owner activity can still race the HTTP endpoint; postflight detects changed other
source and requires the expected glob and a newer Clay/Docket revision. It does
not adopt generated source or assert HTTP/application health. Distributed updates
bind every configured check attestation to both source and artifact; private
preparation precedes those release tests. No test command is executed by the runner.

Schema57 stores immutable frontend reviews atomically with operations. Exact replay
returns the original receipt; dispatched/acknowledged interruptions stay uncertain
without another upload. Read-only reconciliation works after permission revocation
and needs matching original pier/source/artifact metadata. Reconciliation records
observation, not causal attribution. Session cleanup remains separate: confirmed,
unconfirmed or not-created. A trusted adapter's pre-upload abort records rejected
with attempted=false; lost upload replies never use that state. Known upload
acknowledgement survives logout failure. Credentials stay ephemeral and absent from
argv, files, logs and journals; an interrupted or lost login can leave cleanup unknown.

Original schema56 writer656e64d produces the checked-in SQL fixture. Actual migration,
reopen, rollback, new journals and original-reader refusal pass while preserving
registry, policy, settings, source/deployment/adoption/release and receiver histories.
Full desk race regressions, collector/client selection/lost-reply tests, transport
session tests and CLI/offline help checks pass. Both skills validate. API03 initially
expected a cache edit after bytes were captured to invalidate those already frozen
bytes; the corrected test edits before capture and verifies refusal. Regression01's
helper did not recognize the new terminal rejected state; regression02 passes after
fixing the helper. Neither failure caused a native upload or touched installed state.
See `diagnostics/2026-11-07-desk-frontend-api.json` for exact evidence and limits.

This supersedes M552's pending public adapter only. The underlying native hash/bytes
were qualified in M552, but the assembled public frontend capsule and actual Docker
transport remain separate checks. Release qualification stays deferred. Installed
production/DEV/trial and persistent preview remain unchanged. Native48's completed
pier and verified duplicate Vere/pill inputs were retired after exact-scope inactivity
and successful proof checks, preserving committed compact proof and all receipts.
Migration admission refused before launch until retention made room. Re-measure
before builds and keep failed/current source and evidence recoverable.


## M554 — Assembled native frontend API and lost-ack recovery (2026-11-07)

Native49 passes M553's public API/store/collector path on a fresh offline fake.
The typed stage/review/plan/upload flow blocks with both gates closed and sends no
frontend request. After granting both, exactly one real Docket upload succeeds;
the fixture discards its acknowledgement. The runner retains uncertainty with
confirmed ephemeral-session cleanup. Closing both gates again still permits
read-only reconciliation to the expected artifact and newer Clay revision.
Exact replay after stage discard sends no additional upload. The native effect
ledger retains the real acknowledgement independently of the lost managed reply.

Predicted/native glob hash and exact HTML, binary and empty-file bytes/MIME agree.
Generated docket changes correctly conflict with older source ownership. The full
publisher/source/release workflow and empty-source regression also pass; the fake
shuts down cleanly. TestNativeDeskWorkflow takes54.78seconds. See
`diagnostics/2026-11-07-desk-frontend-native-api.json` for inputs, receipts and limits.
This supersedes M553's pending assembled native frontend test, not actual Docker/SSH
or installed upgrade qualification. No HTTP/application health beyond the exact
fixture assets is claimed. Schema57 stays unchanged; release work remains deferred.

Native49 admission charged42,941,153,280 bytes including2GiB scratch, below40GiB.
Before launch, completed test binaries were retained as verified lossless gzip,
completed migration source snapshots as archives verified file-by-file (including
modes), and the rebuildable isolated Go cache removed. Original SQL fixtures and
all source/receipt/log evidence remain recoverable; restore the recorded archives
before using historical extracted paths. The original55 writer source dependency
used by migration02 stays intact. The diagnostic links every retention manifest.
Full failure34 and current49 piers remain; native48 proof was already committed.
Production, DEV, trial and the persistent preview remain unchanged.

## M555 — Shared ship URL registry and CLI (2026-11-07)

Implemented `ship-routes-v1`, runner schema58, collector/client and
`omabit ship-routes` / `ship-routes-configure`. URL authoring binds the actual
native ship, verified pier incarnation, current target generation/process and
configuration revision. A replacement starts with fresh settings. Current and
contiguous historical records save atomically; stale edits and corrupt state
refuse. Automatic localhost discovery is verified against the held runtime's
actual published listener. Its host scope is the runner identity; remote SSH
collectors must not treat it as their own localhost. Manual/tunnel URLs and
persistent automatic-route disable choices are separate from dynamic discovery.

Offline core, API/store, runtime adapter, two-collector/client, CLI and help race
checks pass. The actual schema57 writer atbdb81b8 migrates to58 preserving original
policy, source ownership, release, adoption, receiver and observed/dispatched
frontend records; reopen, rollback and actual older-reader refusal pass. The
checked-in original57 SQL is SHA-pinned. See
`diagnostics/2026-11-07-fleet-registry.json`. This supersedes M550's pending registry
API/CLI, not its remaining Tailscale/Serve, approval UI or reconnection work.
Registration does not configure a tunnel, prove public reachability, obtain
credentials or authorize an application. Installed upgrade/release qualification
remains deferred. Operator skills describe this implemented boundary.

The first core capsule could not create its network namespace in the ordinary
sandbox; the identical verified binary passed in the permitted offline capsule.
Runtime cold-cache admission refused before building; completed migration binaries
were retained as verified lossless gzip, then the warm-cache allowance passed.
Native49's committed proofs, receipts and source inputs remain; its cleanly stopped
592,924,672-byte generated fake pier was retired only after verifying the exact
inactive scope. This supersedes M554's temporary full-pier retention. Failure34 and
all production, DEV, trial and persistent preview state remain untouched.

## M556 — Read-only Tailscale Serve/Funnel discovery (2026-11-07)

Connected bounded local daemon inspection to `ship-routes-v1`. The reader invokes
only `status --json --peers=false` and `serve status --json`, without a shell or
ambient socket/debug/proxy overrides. Reads have byte/time limits; a second status
read rejects a profile switch. Only the local node name, suffix and small status
fields escape; peers/users/authentication URLs and raw errors are discarded.

Serve discovery verifies the node's whole-origin HTTPS proxy targets this ship's
exact loopback listener. It supports foreground/background configurations, prefers
443 then the lowest matching port, and classifies Funnel exposure as public. Path
mounts, shadow handlers, ambiguous ports, unsupported fields, service VIPs, raw
TCP and different upstreams do not become automatic ship routes. Disabled choices
survive port changes. Missing/unavailable discovery keeps saved manual URLs usable.
Nothing installs, authenticates, exposes a new port or changes daemon settings.

Core parser/command race fixtures, runner API and two-collector/client checks pass;
byte overflow, cancellation, environment leakage, profile changes and forged or
non-HTTPS metadata are exercised. See `diagnostics/2026-11-07-fleet-tailscale-discovery.json`
for sources, exact evidence and limits. No live Tailscale deployment is qualified.
Persisted schema58 is unchanged; optional discovery metadata is read-only. This
supersedes M555's pending automatic Serve invocation, not its remaining Setup URL
editor, standing-approval UI/CLI, reconnection or clearweb-assistance work. No
installed/production/DEV/trial/preview state was changed; release work stays deferred.


## M557 — Full-panel Ship URLs Setup editor (2026-11-07)

Added Ship URLs to normal Setup, retaining the existing sidebar, typography, cards
and controls. It selects a verified running ship, presents detected addresses,
and edits configured routes and persistent automatic-route disable choices.
Current runtime/configuration binding changes invalidate reads. Saves use the
runner's current revision; failed or lost replies require refresh, with no replay.
Registration does not create a tunnel or change DNS.

Offline full-panel fixture04 passes authoring, persistence, another ship's empty
settings, offline invalidation and a concurrent-edit refusal followed by refresh.
Wide/narrow screenshots were inspected. Initial fixture01 exposed a fixture-only
empty-command assumption and transient QML/anchor warnings; those were corrected,
and fixtures02–04 pass without warnings. Fixture03 adds the save-conflict check;
04 places Save beside the ship selection. See
`diagnostics/2026-11-07-fleet-setup.json`. This supersedes M556's pending URL editor,
not standing approval, automatic reconnection or clearweb assistance. Installed
production/DEV/trial and the persistent review preview remain unchanged.


## M558 — Standing application approval CLI and offline revocation (2026-11-07)

Added collector `app-ship-review`, `app-ship-approvals` and `app-ship-approve`.
Allow requires an explicit owner-session acknowledgement, exact installed helper,
current live runtime selection, verified pier binding and approval revision.
Review obtains no credentials and invokes no plugin. The bounded JSON input rejects
missing/ambiguous choices and duplicate case-alias keys. Revocation uses the exact
saved target/binding and current revision without requiring a live runner or an
installed helper. Existing sessions retain independent disconnect/recovery records.

Core race checks and public CLI fixtures pass grant, stale choice, replacement
pier, changed helper, malformed input, offline help and offline/uninstalled revoke.
No parent enrollment, application attempt or receiver cache is created. Format1
and runner schema58 remain unchanged. See
`diagnostics/2026-11-07-fleet-approval-cli.json`. This supersedes M557's pending CLI
authoring only; automatic issuance/reconnection and UI integration remain open.
Production, DEV, trial and persistent preview state remain unchanged.

The first core test admission refused before building at the40GiB storage bound.
The completed earlier migration-regression fixture's rebuildable Go cache was
inventoried and retired (243,642,368 bytes), retaining its failed result, logs,
original binary, inputs and SQL databases. Admission and the new test then passed.
The diagnostic links the retention manifest. Release work remains deferred.


## M559 — Ship-incarnation-bound credential retrieval (2026-11-07)

Added internal `ship-route-code-v1` for automatic application authentication.
The selected route, revision, runtime and full ship/pier binding are verified under
the runner lifecycle lock before reading a code. Disabled/shadowed/changed routes
refuse; Serve/Funnel configuration is rechecked. The runtime verifies native ship,
pier continuity, process and listener before and after its private Lens read.
Responses bind the exact request. All failure messages and audit records exclude
raw credentials/errors. No session, plugin delivery or public code-printing CLI
is added; standing approval remains the collector's responsibility.

Synthetic runtime race checks pass normal reads, changed ship/pier/process/port,
invalid secrets and read errors. Runner API tests pass current/manual/Serve/Funnel
routes and deny changed/disabled/unavailable routes before credentials. Client
checks refuse unsupported or rebound/malformed responses without exposing secrets.
See `diagnostics/2026-11-07-fleet-route-code.json`. Schema58 and existing state remain
unchanged. Automatic attempt integration, permission UI and reconnection remain
open; actual live transport and release qualification remain deferred. No installed
production/DEV/trial/preview state was accessed or changed.


## M560 — Durable fleet issuance under standing permission (2026-11-07)

Fleet application attempts now retain the exact ship/pier binding, selected route,
runtime selection and approval revision. A fresh grant is checked atomically when
approving, entering issuance and entering delivery, then again before credentials.
Revocation/reapproval cannot revive queued work. Each attempt creates an independent
origin-bound session directly; it never borrows a parent login or saves a ship code.
Lost authentication/delivery replies retain uncertainty and block another URL for
the same integration. Legacy attempts with unknown ship incarnation must finish
recovery before fleet work for that identity. Cleanup remains possible after revoke.

The active journal advances to format3 and fleet archives use format2. Legacy
records/archives retain their original representation. The actual previous writer
at dddb2cb creates pending and archived state; new readers preserve it alongside
fleet attempts, and the original readers refuse new formats without changing bytes.
Offline core and engine race suites pass, including revoke races, corrupt state,
publication failure, independent fakes and lost replies. Native live tests remain
explicitly skipped. See `diagnostics/2026-11-07-fleet-attempts.json`. Runner schema58
is unchanged. This supersedes M559's pending journal/issuer core only; public
automatic selection, permission UI and reconnection remain open. Production, DEV,
trial and preview state remain untouched; release work stays deferred.

Initial migration admission at600MiB refused before output/copy/build. Restricting
the copied source to the collection dependency closure and retaining a warm cache
allowed the declared400MiB capsule to pass. Completed binary evidence is retained
as verified lossless gzip. No original fixture or source input was removed.


## M561 — Automatic fleet URL selection and public connection command (2026-11-07)

Added `urbit-collector app-fleet-connect` for a precise standing ship/helper grant.
The collector checks its own current Tailscale scope, permits localhost only for a
pinned local Unix runner, and uses explicit LAN scopes. A bounded15-second probe
window tries eligible URLs without credentials; the selected runtime, pier, route
and grant are rechecked before publishing an attempt. An existing request cannot
start another selection or login. Public issuance uses Secret Service only, with
no parent session or plaintext fallback. Status exposes nonsecret fleet ancestry
and probe outcomes for the upcoming UI.

Core race fixtures pass default deny, stale/revoked grants, fallback, foreign or
changed tailnets, explicit LAN selection, changed runtime/pier/route/enrollment and
cancellation. The actual public CLI passes approval through selection, independent
login, retained receiver delivery, offline revoke, disconnect/reconcile and archive.
Wrong-ship candidates receive no credentials; lost login/delivery and storage
failure do not trigger another URL/login. The fixture uses a private synthetic
Secret Service command transport; it does not qualify the actual desktop keyring
or live networking. See `diagnostics/2026-11-07-fleet-connect.json`. Persistent
formats are unchanged. This supersedes M560's pending public connection path; UI
integration and automatic reconnection remain open. Installed state stays untouched.

To stay under40GiB, five completed managed-test package archives were inventoried
and removed from the task's rebuildable Go cache (168,030,208 bytes). Exact completed
binaries, logs, source and original migration fixtures remain. Warm-cache smaller
package tests reserve400MiB. Release qualification remains deferred.


## M562 — Standing ship permission in the full application panel (2026-11-07)

Application Connections now discovers capable remote ships even without a local
URL. Verified ship/helper grants drive the saved Allowed switch; Connect uses the
automatic fleet command. Revoke and Disconnect are separate, and offline/removed
integrations retain visible revocation and recovery controls. Runtime, profile or
selection changes invalidate review. Failed choices never become optimistic grants,
and refreshing an uncertain attempt does not replay authentication or delivery.
Legacy hosted-ship sign-in keeps its existing explicit per-attempt flow.

Full-panel fixture06 passes grant persistence, automatic connection, revoke without
disconnect, disconnect/reconciliation/archive, another ship's default deny, offline
revoke, conflict refusal and lost-reply recovery. Wide/narrow screenshots were
visually reviewed with normal sidebar and app styling. Hosted regression and actual
service-drain fixtures pass. Model checks cover remote URLs, independent fakes and
runtime/offline invalidation. See `diagnostics/2026-11-07-fleet-permission-ui.json`.

Earlier fixture runs exposed QML object-order comparisons and an optimistic switch
after a failed save; both are fixed. Fixture-only deadline/profile-simulation and
component-copy issues were corrected. An unrelated historical project-fleet test
expects Project for an observation now marked Stale; the same assertion fails on
unchanged58f10a6. Its expectation was not used to weaken the live-selection guard.
This supersedes M561's pending approval UI. Automatic reconnection/receiver handoff,
clearweb help and Urgit workflow remain open. Installed state and the persistent
manual preview are unchanged; release work remains deferred.


## M563 — Selected-ship remote access assistance (2026-11-07)

Setup → Ship URLs now opens the default agent with the exact nonsecret runner,
ship/pier binding, runtime and saved URL revision, plus provider/action selection.
Help supports private Serve, public Funnel, Cloudflare and custom-domain planning.
The operator skill describes upstream/audience checks, authorized provider changes,
verification, registry updates and targeted removal. Disabling an Omabit route is
explicitly separate from taking its external exposure offline. No provider commands
or automatic account/network changes are added.

Prompt validation and existing dispatch/skill-install tests pass. Full-panel06
passes all provider/action contexts, invalid URLs, selection clearing and offline
invalidation without QML warnings; wide/narrow screenshots were inspected. Run05
passed interactions but exposed missing fields in the inert agent stub; the fixture
was corrected. See `diagnostics/2026-11-07-remote-access-help.json`. This supersedes
M562's pending clearweb Help only. Reconnection/receiver handoff and native Urgit
workflow coverage remain open. Actual provider deployment, installed networking
and release qualification are not claimed; all installed state remains untouched.


## M564 — Non-mutating fleet session health (2026-11-07)

Added a bounded retained-session check and public `app-fleet-health` command.
Current standing permission, exact installed helper, continuing ship/pier identity
and current collector network eligibility precede credential access. Healthy,
expired, rejected and unreachable observations remain distinct from missing/locked
credentials, malformed state, redirects and wrong ship identity. Identity probes
send no cookies; only the verified original origin receives its unexpired cookie.
Checks serialize with disconnect and change no attempt, login, logout or delivery.

Synthetic race fixtures cover21 HTTP/session cases,15 route eligibility cases and
15 engine cases; the actual public CLI passes with private fake runner/Eyre/receiver
and Secret Service command transport. Native live Eyre remains explicitly skipped.
Initial route/expiry fixture mistakes were fixed without weakening the production
guards. See `diagnostics/2026-11-07-fleet-session-health.json`. Persistent formats
are unchanged. This is preparation for automatic reconnection, not a completed
replacement protocol; compatible receiver handoff and Urgit workflow remain open.
Installed state and the persistent manual preview remain untouched.

The CLI test initially refused at the40GiB admission bound before creating output
or building. Four completed, rebuildable managed/runtimequalify test cache archives
were inventoried and retired (114,118,656 bytes); exact evidence binaries, original
fixtures and all source remain. The admitted test then passed. Release work is deferred.


## M565 — Native Urgit development pathway (2026-11-07)

Pinned upstream7834a4e built without source or lockfile changes using Zig0.15.2 and
Node22.23.4. One fresh, offline fake with pinned Vere/pill installed Urgit, served
its exact built index, accepted a stock Git push into Clay, rejected an unknown-mark
conversion without advancing the ref or adding that file, accepted a subsequent
update, and projected an explicit Clay edit back to Git. The final contents and
native bridge status agree. The original fixture stopped on two incorrect Hoon
text assertions; each invocation shut down cleanly and continuation preserved the
same pier without replaying the acknowledged initial push. Native03 passes.

The upstream frontend suite is not green:83 pass and43 fail, including source-pattern
assertions against refactored Hoon arms. Original results are preserved. Build01
needed the resolved certificate bundle mounted into the private capsule; build02
needed the destination directory required by upstream's clearing/copy step;
build03 passes. These were fixture setup errors, not upstream patches.

The operator skill now covers the native binding, permissions, Git credentials,
branch selection, canonical-byte differences, uncertain responses, project checks,
frontend packaging and the distinction between Clay-to-Git and Treaty publication.
No Omabit adapter or new release gate was introduced. See
`diagnostics/2026-11-07-urgit-workflow.json` for hashes and limits. This supersedes
M564's pending Urgit pathway only; automatic fleet reconnection/receiver handoff
remains open. All installed state and the persistent preview are untouched.

Completed task Go and Zig/npm caches were inventoried and retired to admit the
capsules within40GiB. Immutable inputs, dependency source, original fixture Git
history, exact built desk and compressed full test logs remain. This test does not
retain the completed fake pier: its615,211,008 allocated bytes were inventoried
and retired after native03's successful result and clean process exit. This does not
qualify production upgrades, arbitrary app compilation, every frontend asset,
Urgit peer/GitHub/LFS features or Urgit-driven multi-ship releases. Release work
remains deferred.

## M566 — Theme replacement preparation (2026-11-07)

An isolated Theme source branch now has a bound private-stdin preparation path.
It compares the exact accepted old request, account and stored credential fingerprint
before fencing transport admission. It waits for local owners and refuses unknown
settings writes, including those whose channels were deleted. It retains accounts,
credentials, old remote cleanup receipts and the retirement fence after a failed
drain. Preparation sends no network traffic and installs no successor.

The source is8c81ea8 in `/tmp/omabit-receiver-reconnect-040/theme`, branched from
the clean auth source1c239c6. Private namespace tests pass34 transport/admission/
actual-writer compatibility cases and62 connection/recovery/disconnect/admission
cases. See `diagnostics/2026-11-07-theme-handoff-preparation.json`. The declaration
does not advertise complete replacement. Existing journal formats are unchanged.
An activation must revalidate under the account lock; prepared is not a reservation.

This is partial receiver work for the remaining M565 fleet task. DMs cooperative
drain, new-session activation/recovery, issuer lineage and automatic reconnection
remain open. Installed receivers, canonical/auth source checkouts, runners, ships
and the persistent preview are unchanged. Release qualification stays deferred.


## M567 — DMs local retirement and drain (2026-11-07)

An isolated DMs branch now has a bound bridge drain primitive that fences the old
session before waiting for workers and streams. It preserves accounts, credentials,
message/action uncertainty and exact remote cleanup records. Successful local
quiescence is distinct from remote closure; unknown writes prevent handoff. Normal
restore/read admission rejects retired sessions before network IO.

Retirement upgrades the transport journal1→2; ordinary operation remains1 until
then. The unmodified6ba5a02 writer produces the migration fixture, whose channel
and cleanup records survive unchanged; the old reader refuses2 without rewriting.
Interruption and lost-sync cases retain authoritative evidence. Private journal/
cleanup48 and bridge/stream50 cases pass. Earlier run02 duplicated an imported test
class; final03 corrects discovery. Source and hashes are recorded in
`diagnostics/2026-11-07-dms-local-handoff.json`.

The primitive is not yet wired to the receiver/bridge command loop. Cooperative
requests, successor activation/recovery, collector lineage and automatic polling
remain open. This advances M566's pending DMs local drain only; it is not complete
reconnection or installed-upgrade qualification. Canonical/auth source checkouts,
installed state, protected ships and the preview remain untouched. Release work
stays deferred.

## M568 — Theme bound replacement and recovery (2026-11-07)

The isolated Theme receiver now replaces an exact accepted account with a fresh
origin-bound session, preserving its preferences and keeping the old credential
in a separate request-specific Secret Service application namespace. Durable
nonsecret intent precedes credential effects. Partial installation cannot admit
application traffic; a lost accepted reply can be observed without redelivery.
Exact accepted ancestry permits the new session to operate while old remote
cleanup is pending. Old cleanup cannot retire or clear the live successor.

Recovery distinguishes old metadata with a new key, installed but unaccepted
metadata, accepted successors and later standalone changes. A failed partial swap
can restore only its exact retired predecessor slot; it never reopens that session
for traffic. Unknown application writes are preserved and never replayed. Same-URL
renewal, changed URLs, return to prior URLs, multiple generations, active new
channels, locked/altered secrets and lost persistence acknowledgements are covered.

The receiver/transport formats remain1/2. The new bounded active replacement
sidecar has immutable completed receipts. An actual previous receiver journal
writer still finishes an unrelated account; installer upgrades preserve the prior
recognized inventory. Final private run10 passes146 tests, including22 replacement
and16 sidecar cases. Source4365455 and the retention foundation1221aa3 are in the
isolated Theme branch. See `diagnostics/2026-11-07-theme-session-replacement.json`.

This advances M566's pending Theme activation/recovery, not the entire fleet goal.
No complete replacement capability is advertised or consumed by Omabit yet. A
predecessor using an older helper without this protocol must not be silently
promoted. DMs cooperative requests/activation, collector lineage and automatic
polling still remain. Installed state, original source checkouts, protected ships
and the persistent preview are unchanged. Release qualification stays deferred.

## M569 — Cooperative DMs replacement and exact recovery (2026-11-07)

The isolated DMs bridge now serves bound same-user local control from its main
loop while the UI is idle. Preparation runs outside its worker pool and the
receiver releases account.lock while workers persist positive receipts. Original
request/account/fingerprint bindings are rechecked after drain. Missing sockets
and free leases alone do not prove prior worker exit. Exact quiescent generation
receipts now allow cleanup while a paused old bridge process remains alive;
unrelated live or unknown actors retain their existing refusal.

The receiver supports preparation, independent fresh-session replacement and
accepted-status observation. Private nonsecret sidecar intent and request-specific
Secret Service retention precede overwrite; pending installation denies transport.
The paused bridge restores only its exact accepted successor with a fresh pool,
without another login or UI restart. Accepted ancestry keeps original cleanup
pending without blocking new traffic. Bound control can close ancestor channels
while the successor bridge remains active; credentials/account removal still
compare the original binding. Unknown application writes are never replayed.

Partial-key/account/acceptance and lost-clear recovery preserve originals and
later standalone changes. DMs09 found an absent temporary file masking the
original interruption after completed metadata rename; cleanup now suppresses
only that secondary FileNotFoundError. DMs10 passes203 tests; follow-up11 passes29
replacement/control/shutdown cases including saved-session restore retry. Theme
follow-up201fdd5 passes23 replacement cases and preserves independent credentials
changed during partial swap. DMs05 had three unmatched requested patterns; the
harness now refuses unmatched patterns and later full-suite runs supersede it.

Source464076b (control foundation9826d78) remains isolated. See
`diagnostics/2026-11-07-dms-session-replacement.json`. This supersedes M567's pending
DMs cooperative/receiver path at the private source-fixture level only. Receiver
capability advertisement, collector lineage/actual-writer migration, automatic
polling and assembled native qualification remain. Current format/readability
contracts and helper inventory are recorded; older cached helpers must not be
silently promoted. Installed state, protected ships, original source checkouts and
the persistent preview remain untouched. Release qualification stays deferred.


## M570 — Collector replacement ancestry and issuer recovery (2026-11-07)

The collector now recognizes replacement protocol2 only from an exact approved
helper declaration. Old cached1 generations remain1 after an installed update.
Preparation, replacement and status use the existing bounded private launcher.
No installed receiver is promoted or executed merely to discover capability.

Applications journal4 records immutable predecessor intent before preparation or
issuance, and records the accepted successor link atomically with connection. A
shared integration/ship lease serializes every generation, including health and
old cleanup. Original credentials, origins, bindings and pending cleanup remain
separate. Closed ancestors use immutable history3; cycles, missing ancestry,
changed helpers/incarnations and competing active branches are refused. Existing
formats remain readable; actual unmodified collection sources at2552804 create
previous3 state and refuse populated4 without changing it. Reopens preserve the
original evidence. The fixture manifest pins all25 retained source files.

The issuer prepares only with the exact retained predecessor credential, issues
one independent session and invokes bound replacement once. Reused predecessor
cookies are neither delivered nor revoked as new sessions. Lost delivery replies
use bound status observation; unknown replies remain pending, with no repeated
login or offer. Known acceptance survives restart and records lineage before
honoring a concurrent permission revocation. Interrupted preparation closes only
the known-unissued child; it does not pretend the old receiver resumed. Cleanup
remains bound to each original session. No application write is replayed.

Full collector and issuer race suites and focused Eyre validation/health/logout
fixtures pass. See `diagnostics/2026-11-07-fleet-replacement-issuer.json` for exact
logs, binary hashes and previous-writer provenance. This advances M569's pending
collector work at the private source level. Public reconnect/recovery wiring,
automatic polling, capability advertisement in the two receiver branches and
assembled qualification remain. Installed plugins/runners, protected ships and
the persistent preview are unchanged; release qualification remains deferred.


## M571 — Bounded fleet reconnection CLI and desktop polling (2026-11-07)

`app-fleet-reconnect` selects an eligible registered route under the current exact
ship/helper grant, retains the generation and publishes bound successor intent.
It leaves healthy sessions unchanged. Both installed and cached predecessor
generations must support replacement. `app-recover` observes lost replacement
replies instead of revoking an accepted successor by inference. Public views show
predecessor/successor links and discovery reports complete helper capability.

`app-fleet-maintain` performs one bounded health/reconnect or lineage cleanup step.
Known failed children wait five minutes, with three failures requiring review.
Automatic maintenance never archives failures to evade this budget. An explicit
Disconnect durably disables pending automation on the selected attempt and its
unaccepted handoff links; cleanup of an already accepted predecessor does not
stop the live successor. Closed records with already published archive receipts
remain unchanged. A manual explicit retry can resume a retired predecessor only
after all prior child effects are reconciled and current permission still holds.

The desktop schedules at most one eligible step per minute, serially, while
activation is enabled. It refreshes inventory periodically and never implicitly
allows permissions or chooses a LAN scope. Initial uncertain logins stay fenced.
The full Setup panel reports current URLs separately from previous-session cleanup
and exposes recovery/reconnect controls. Wide/narrow screenshots were reviewed;
recovery status now wraps independently from long URLs. The operator skill covers
maintenance, exact lineage, old-helper limits and recovery without replay.

Public CLI fixtures pass healthy no-op, replacement, lost acknowledgement/restart,
old cleanup, bounded retry and explicit cancellation. The first CLI run exposed
a fixture alias overwriting the saved successor revision; the collector correctly
refused it. The fixture was fixed without weakening revision checks. Final CLI03,
full registry/auth race suites, full-panel03, hosted UI regression and skill
validation/package checks pass. The default Python lacked PyYAML; the unchanged
validator passed under system Python. Evidence is in
`diagnostics/2026-11-07-fleet-reconnect-cli-ui.json`.

This advances M570's public/desktop work. Real Theme/DMs declarations remain1
until the assembled receiver/keyring/transport pathway is qualified. Source
fixtures do not qualify a native install, production Tailnet or stable release.
Production/DEV/trial plugins/runners, protected ships, original receiver checkouts
and the persistent manual preview remain untouched.

## M572 — Assembled reconnect and feature implementation closeout (2026-11-07)

The actual collector public command dispatcher, both retained receiver helpers,
real private Secret Service and running Theme/DMs consumers pass the assembled
private test. Each plugin performs an initial connection and three replacements:
same URL, alternate URL with the original stream/route unavailable, and lost
delivery/status acknowledgement. Fresh issuance/code counts, exact helper calls
and strict cookie-origin checks pass. Original cleanup remains pending offline,
then completes and archives without stopping the accepted successor. Both consumers
exit cleanly with their UI input pipes still open.

This test exposed and fixed three Theme gaps: following the selected desktop hub
through accepted handoff ancestry across URL changes; stopping its input reader
without a Python buffered-I/O shutdown abort; and accepting the exact receiver's
durable cleanup receipt for a retained in-memory watch marker. Hub selection does
not enable sync, unpause the user or adopt an unrelated same-named ship. Unknown
records do not prove cleanup. The collector now classifies HTTP502/503/504 health
responses as unavailable routes, without treating them as revocation evidence.

The final assembled run is `/tmp/omabit-real-reconnect-07`. Earlier failed runs
and fixture corrections are recorded, including an archived-flag mistake, omitted
Talon accent behavior and an incorrect synthetic expired-token response. Final
Theme regression14 passes151 tests; focused health/validation tests pass with the
race detector. Skill validation and five packaging tests pass. CLI documentation
was regenerated from172 actual command/topic help entries in a networkless private
HOME; help created no registry/state.

Theme source `821242e` and DMs source `c7b7d15` now advertise declaration2 on their
isolated `codex/0.4.0-session-reconnect` branches. All tested Python runtime bytes
match those commits. Only profile/manifest names, declaration capability and an
inventoried lost-reply wrapper were adapted in the capsule. Theme desktop effects
were inert; its coordinator, watch and Talon bridge were actual code. DMs used its
actual main loop/control. Runner/Eyre peers were synthetic, so this does not claim
a new native Urbit, provider/Tailnet deployment, installed upgrade or release gate.
Older cached version1 helpers retain their original capability and cleanup path.

This supersedes M571's pending assembled receiver/capability work and closes the
approved feature implementation scope. Original migration fixtures and earlier
native desk/Urgit evidence remain authoritative within their recorded limits.
Release qualification/publication remains explicitly deferred. No installed
production/DEV/trial plugin, runner, ship, desktop/keyring or persistent preview
was changed. Exact source identities, tests and limitations are recorded in
`diagnostics/2026-11-07-real-receiver-reconnect.json`.

## M573 — configurable container RAM and CPU limits (2026-11-12, implemented)

The user added per-container resource configuration to0.4.0 and explicitly chose
RAM and CPU limits. Individually managed ships now expose1–16GiB RAM and1–16 whole
CPU cores, retaining4GiB/2CPU defaults. Changes require stopped/uninitialized state
and no active operation, recovery or uncertain provisioning/maintenance. The workflow
is Stop → reload runtime settings → save → Start; its pier is retained. No existing
ship or installed plugin/runner was operated on to implement this request.

Schema59 fences optional desired and immutable per-container limits. Migration
preserves original bodies; absent fields retain historical limits. Old-client edits
preserve omitted limits. Creation records intent before Docker; probes, containment,
maintenance and stats check the recorded ceiling. Requested versus recorded limits
remain distinct, and CPU charts preserve per-core units. Loom and start-admission
policy remain independent. Project members/private gateways keep their own fixed
contracts; this feature does not silently modify their separate lifecycle protocol.

Nondefault handoffs require capability at both endpoints before source fencing and
sign both limits. Explicit defaults normalize to the old settings wire shape, so
restoring4/2 does not break otherwise compatible moves to an older runner. The
actual schema58 writer66c641b and its receipts survive migration/reopen; that same
writer refuses59 unchanged. Original fixture SQL/checksum is retained in source.

Seventy-five focused Go tests pass with the race detector, covering resize/restart,
containment, metrics, concurrency, old clients, signed moves and migration. UI model
checks, real wide/narrow QML controls with inert callbacks, skill validation and82
actual collector help entries pass. See [validation](runtime-settings-validation.md)
and `diagnostics/2026-11-12-container-resources.json` for scope/evidence and initial
fixture corrections. This supersedes fixed4GiB/2CPU assumptions for the capable
standalone runtime only; historical qualification records retain their original
meaning. Release qualification, installed upgrades and a live16GiB workload remain
separate and deferred. Source integration targets only the0.4.0 feature branch;
production/DEV/trial installations, ships and the persistent preview are unchanged.

## M574 — persistent resource-control preview updated (2026-11-12, installed preview only)

The user explicitly authorized updating the isolated preview for manual RAM/CPU
review while preserving production. `prepare-container-resources-preview.py` now
creates the full actual panel with a fixed Python simulation for resource edits,
Start and Stop. It contains no collector, Docker, SSH or runner helper. New choices
persist in a separate `resources.json`; permission state remains independent.
Project members stay outside standalone resizing; running/observer/offline contexts
refuse resource edits. A RAM & CPU shortcut reveals the normal Overview editor.

The full-panel fixture `/tmp/omabit-resources-preview-test-03` passed save/start/stop,
requested versus applied limits, readonly contexts, Setup tree navigation, reopening
saved values and byte-preservation of the original permission file. First test
attempts corrected a nonexistent sidebar test ID and a chart/control name collision
in the test's global selector. Real controls are scoped to their editor. Screenshots
were inspected; no QML warnings/errors occurred in the passing offscreen runs.

Preview source6f9b91a was installed at
`~/.local/share/omabit-policy-preview-040`, preserving its previous source under
`versions/before-container-resources-20261112`. Original permission SHA256 before
and after is recorded in `versions/container-resources-update-20261112.json`.
The native Wayland window loaded and its supervisor was confirmed open from the
host process namespace (a sandboxed PID view cannot establish that status). One
native Qt Wayland text-input warning was nonfatal; the window rendered normally.
The existing separately named desktop launcher is unchanged. This supersedes
M573's unchanged-preview note only; production/DEV/trial plugins, runner services,
ships and the active desktop shell were not inspected or changed. All manual
resource effects in this window are simulated. No installed runner qualification
or release claim follows from this preview update.

## M575 — per-ship controls and persistent real DEV review (2026-11-12, installed)

The user removed the resource header shortcut, requested RAM/CPU in initial boot
advanced settings, enabled Agent Help, per-ship URL configuration, and a real DEV
runner/widget while preserving production. They explicitly authorized local
TRIAL retirement. This supersedes M574's simulated-only review and the old
blanket trial/DEV exclusion only for this fresh namespace and local retirement.

Standalone fake/comet/key boot forms and collector commands now carry validated
1–16GiB/1–16CPU limits. Omission retains4/2 and old-runner compatibility; explicit
limits require capability. The ship's runtime editor remains stopped-only. URLs
now belong to the selected fleet ship's own tab, with binding invalidation on
selection change; bulk CLI workflow remains unchanged. The preview header shortcut
is removed. Agent Help is enabled in the live panel, correctly discovers Opencode,
validates its bundled skills and passes the separate review profile. No real agent
session was launched and no global skills were installed automatically.

A root-owned named development profile now supports `/srv/dev/urbit-NAME`, retaining
the exact old defaults and requiring clean bounded names, its own owner record,
pier root, state and runtime directory. The fresh `/srv/dev/urbit-040` engine and
runner booted a new fake~zod at4/2, stopped, saved16/4 and restarted healthy with
applied revision2 and the original pier. Actual Docker Memory/NanoCpus matched.
This is real configuration/restart evidence, not a16GiB pressure/stress test or
release qualification. Its live URL registry also returned the verified managed
local route. Default publishing policy was not opened.

Native activation exposed three missing qmldir entries; the shipped registry is
fixed and previews now consume it rather than synthesizing registrations. Native
registry tests, CLI bounds/capability/request tests with race detection, scoped
boot-form/resource/URL/Agent Help QML tests, storage admission, skill packaging and
172 networkless CLI help entries pass. Early synthetic CLI fixtures needed private
registry permissions and matching operation receipts; the removed shortcut also
required fixing the resource fixture's navigation. Those failures are retained.

Quickshell cached the first partial external module's failed types and file names.
The working plugin was staged completely, then atomically installed at a fresh
namespace `thelifeandtimes.omabit.review040`; the production shell was not restarted.
The menubar says DEV and the application launcher is Omabit0.4.0 DEV. Local TRIAL
and failed DEV copies/profile records were archived, preserving credentials; no
rooftop action or revocation occurred. The simulated preview data and earlier
human DEV piers are preserved. The new owner directory was provisioned through
Polkit; only the two new user units were enabled.

Generated npm caches were reclaimed; retained identical dependency files were
hard-linked with hashes while preserving paths/content. Completed Go compiler
cache was retired with a manifest after retaining compiled artifacts. Combined
storage including both DEV roots, local plugin/archive and explicit unreadable
rootless-work reserves was39.218GiB, below40GiB. Do not start another experiment
without fresh admission. Full provenance, failures, scripts, receipts and entry
points are in [the live review guide](native-review-0.4.0.md) and
`diagnostics/2026-11-12-live-dev-review.json`. Production plugins, services and
protected ships were not modified or operated on. Release qualification remains
deferred.

## M576 — Agent Help session lifetime fixed (2026-11-12, installed in DEV)

The user reported that Opencode opened from Agent Help then closed after a few
seconds. The DEV controller's actual error confirmed `TimeoutExpired` at 15 seconds.
The helper used `subprocess.run(..., capture_output=True, timeout=15)` for Omarchy's
interactive launcher. Local Omarchy source shows the exec chain through
`omarchy-agent-prompt`, `omarchy-agent`, `omarchy-launch-tui`, `setsid` and `uwsm-app`.
An attached launcher remains alive with its terminal; timing out that subprocess
ends the interactive session. This was Omabit's lifetime bug, not evidence of an
Opencode crash. The old prompt and user conversation were not retained in evidence.

Launch now uses Popen with a separate process session, closed inherited descriptors
and stdin/stdout/stderr on /dev/null. A half-second wait checks immediate startup
failure; timeout there means the interactive session is running and is acknowledged
without terminating it. There is no interactive-session deadline. The QML reader's
own bounded deadline remains unchanged, and skill-installation behavior is unchanged.
Spawn/early exit failures use a generic error without reflecting the prompt.

Eleven helper tests pass. The actual full-panel helper fixture now has a persistent
inert-agent mode: two launches survived beyond 21 seconds, past both the old
15-second launcher deadline and 20-second QML reader deadline, with distinct session
IDs and private stdio. The test also suspended the service and hid the panel.
Two early fixture attempts exited because the simulated preview explicitly quits
when hidden; the corrected fixture removes that test-only hook and keeps a second
inert window, like the surrounding desktop. No real agent, network, production
profile or runner was used by the regression test. Evidence is in
`/tmp/omabit-agent-ui-lifetime-03` and
`diagnostics/2026-11-12-agent-help-lifetime.json`.

Only the DEV installation's `app040/agent-help.py` was atomically replaced, after
checking its original hash. Its previous bytes and patch receipt are preserved in
`/srv/dev/urbit-040/desktop`. No shell or runner restart, global agent setting,
production plugin change or automated real-agent launch occurred. M575's initial
launch-readiness evidence did not cover terminal lifetime; this regression closes
that gap. Release qualification remains deferred.

## M577 —0.4.0 qualification authorized (2026-11-12, in progress)

The user accepted RAM/CPU and Agent Help launch, delegated real app-development
and remote URL/receiver testing, and authorized the release qualification and
isolated upgrade rehearsal. This supersedes M572–M575's deferral of that phase,
not their stated evidence limits. Live provider testing remains with the user.
Production and manual-review state remain excluded. The temporary aggregate
experiment cap is48GiB by explicit user approval, returning below40GiB through
retirement of only new disposable outputs. See releases/0.4.0/preparation.md.

## M578 —0.4.0 qualification and isolated manual upgrade VM (2026-11-12)

M577's authorized qualification now has a frozen candidate at
`a9623329a538ebf83d63365053d2d1011ade091e`. Exact archive, native desktop and
0.3.7→0.4.0 installed-service rehearsal pass. The actual CLI/SSH scenario exposed
a missing source-transfer descriptor/body allowance in the bridge; the bounded
streaming fix, identity/capability checks, cancellation/deadline tests and full
rebuilt VM rerun pass. Candidate02's failure remains recorded. Source/frontend
publication and two receivers pass in a separate native fixture with a synthetic
managed lifecycle backend. The real installed-runner scenario covers source
publication; the separate routed fleet scenario uses actual Theme/DMs helpers,
private Secret Service and standing approvals. It does not certify public Tailnet,
Tunnel/DNS setup or ship content publication on a live provider.

See releases/0.4.0/qualification.json, rehearsal.json and local-validation.json
for exact inputs, source/race coverage, CI and scope limits. The earlier complete
source CI passed. Exact release PR checks remain required before merging; no
qualification record waives them. Companion receiver rollout must coordinate the
compatible Theme/DMs helper versions recorded in the fleet evidence. The tested
private helper branches are not proof that public plugin distributions contain
those versions. Stable publication and production upgrades are not authorized.

At the user's request, a separate persistent manual VM was prepared from the
verified pristine base under `patch-upgrade-040-manual-review`. It has actual
0.3.7 services/schema47, fresh fake~zod running and~nec stopped, and both sealed
desktop bundles staged. F5 loads0.4.0; the service update is left to the user.
Its real Hyprland desktop and stock Omarchy Polkit prompt were inspected; the
probe was canceled without authorizing an update. The guest config was backed up.
No host desktop config or existing plugin was changed. The local viewer is6084,
with its own unit/IPC; no external guest NIC exists. The launcher can only resume
or gracefully stop this disk and enforces40GiB aggregate/4GiB own allocation.

The approved temporary48GiB allowance is no longer needed: new completed test
VMs, native fixture piers, compiler cache, superseded unpublished candidates and
redundant build/download copies were retired with manifests. Compact results and
hashes remain. Charged storage with the new running review VM was39.485GiB before
small verification downloads, which were also retired. Both existing DEV roots,
all historical/manual environments, protected ships and production remain intact.

## M579 — Exact interrupted-stop recovery contract (2026-11-17, source-qualified)

Issues #16/#17 expose a recovery gap after an expanded-member clean Stop. An
inert fixture reproduces the reported observation failure shape when exit occurs
between preliminary/bracketed reads or during a namespace probe. This supports
that race mechanism, not a proven predicate for the historical incident. Fleet
inspection now binds the process and retries only a verified transition, with
all ownership checks repeated. SSH receipt lookup now retains its guard on
project endpoints; unsupported guarded writes refuse before client dispatch.

Schema60 adds pre-signal bindings and independent reconciliation receipts. Saved
diagnosis, live review, exact apply, original receipt lookup and ordinary Start
have distinct contracts. Recovery binds the original typed request/container,
preserves the interrupted operation and clears only its verified fence. Legacy
admission-window evidence remains explicitly distinct from new process capture.
Reopen/admission validate resolution ancestry; corruption and stale review refuse.

Pinned original0.4.0/schema59 writer preservation and actual older-reader refusal
pass. The frozen fixture records the writer, helper and SQL hashes with explicit
supersession. Actual collector lost-reply/replay, focused managed/runtime/bridge
race checks, the full client race suite and affected-package vet pass using inert
fixtures. CLI help generation is networkless and creates no state. See
[the supported source contract](project-member-recovery.md). Disposable real-runtime
recovery and an installed non-destructive upgrade are still required. No installed
deployment or broader issue completion is established by this source checkpoint.

## M580 — Selected-runner preflight and temporary budget (2026-11-17)

Issue21's preflight now distinguishes unavailable policy/API support, missing desk
operation adapters, a missing framework adapter, operator role and exact saved
policy scope. It contacts only the pinned selected enrollment and never boots a
project or changes policy. A future ship grant is not inferred from environment
compatibility. Targeted reads retain project creator/generation; all results limit
their assurance to capability and saved policy. See
[the source contract](app-development-preflight.md). Gall uses existing typed desk
primitives; Grubbery remains unavailable pending its versioned adapter.

Optional managed Hello metadata identifies the executable's compiled provenance,
hash and schema separately from the client. It is an authenticated self-report,
not verified release provenance; absent older-peer metadata stays unknown. Inert
selected-runner/policy-scope/identity/role tests, managed API provenance, offline
help and affected vet pass. This does not identify the historical incident's
runner or establish an installed matching deployment or native app execution.

The user explicitly authorized48GiB aggregate evidence/DEV storage for this goal,
with one runtime experiment active and production/existing ships untouched.
[Experiment isolation](experiment-isolation.md) records the scoped exception; the
normal40GiB default is unchanged. This is separate from M577's expired allowance.

## M581 — Shared CLI/MCP source checkpoint (2026-11-17)

Issue19 now has a typed service facade, a maintained public CLI inventory with
explicit exclusions, canonical inferred input/output schemas and version1 result
envelopes. `omabit call` and the official SDK1.8.0 stdio adapter invoke those same
handlers. Per-call target pins, server allowlists, role/capability/policy admission,
exact original request IDs and bounded results are retained. Existing CLI output
contracts remain separate and unchanged. See [coverage and limits](core-interfaces.md).

Focused fixtures found and corrected a schema-number mismatch without dispatch:
json.Number was classified as string by the inferred-schema library. Exact int64/
uint64 conversion now preserves review/revision/inode values above2^53. Canonical
output validation also runs before returning a successful envelope. Arbitrary
transport/runtime diagnostic text is not reflected into model-visible errors.

Actual CLI dispatch and SDK stdio subprocess fixtures agree on saved policy and
input/capability/role/policy refusals. Four legacy protocol revisions negotiate;
newer2026 negotiation, named-client products and HTTP remain unqualified. Pinned
BUILD/SKILLS resources and ordinary tool reads serve the same rehashed Markdown;
legacy two-skill bundles remain accepted and changed/linked/unlisted resources
refuse. Reading guidance never installs it or grants authority.

The built Omabit MCP adapter submitted a reconciliation, lost its reply, closed,
then the built CLI found the exact receipt after authoritative store/manager
reopen with no observation backend. The old interruption remained unchanged and
lookup performed no additional observation/effect. This is real API/SQLite and an inert runtime
observer. An initial fixture directory mode failure was corrected without
weakening socket checks. A combined rerun also exposed a non-atomic observer
counter in the earlier collector fixture; it now uses an atomic counter. Focused
race, static and networkless-help checks pass;
CI explicitly builds helpers for this cross-interface test. Source staging,
compound workflows, named qualification, live Urbit, installed upgrades and broader
catalog coverage remain later slices. Production and manual-review state were not
changed; this checkpoint does not complete issue19 or authorize publication.

## M582 — Independently usable third skill delivery (2026-11-17)

Issue18's `omabit-app-developer` now routes actual Gall/Grubbery source into
preflight, staged-source/policy and recovery guidance. The entry point is backend
focused and does not require an Omarchy frontend or sibling global skill install.
Its shared app/operations/Urgit references are generated from canonical operator
files, with exact-sync and independent local-link checks. Gall sample/native
execution and the Grubbery adapter remain unfinished; the latter is explicitly
unavailable in preflight and guidance. See [the per-issue status](issues-16-21-progress.md).

`SKILLSET.json` requires the full three-skill inventory in new source packaging;
historical exports without it retain the pair. Agent Help accepts both verified
manifests, lists their actual inventory and routes app-development Help to the
third entry when present. Custom/edited installation preservation is unchanged.
Seven packaging, twelve helper/installation, fifteen native-bootstrap and nine
native-plugin fixtures pass. The system-Python skill validator passes; the other
Python lacked PyYAML, so no global package was installed to run it.

The first full-panel offscreen run exposed the QML reader's old two-skill bound.
The corrected reader validates either exact manifest inventory. A fresh networkless
private-HOME/XDG fixture installs all three complete skills through actual controls,
launches only inert agents and renders their distinct responsibilities. A separate
historical-pair fixture verifies continued UI use. Failed fixture evidence is
retained. No real agent, global skill directory, installed plugin or service was
changed. These delivery tests do not qualify either application runtime workflow.

## M583 — Frozen bounded Gall sample source (2026-11-17)

`app.sample.v1` returns the compiled `gall-state-v1` definition, exact source
manifest, hashes and notices through the same CLI/MCP handler. The4097-byte,
five-file staging payload contains the state fixture, bill mark, desk bill and
only its two required libraries. Source accepted by the canonical staging scanner
is distinct from current Hoon compilation/runtime evidence; the sample explicitly
reports current-workflow-pending. No source read deploys or executes anything.

The three upstream scaffolding files are pinned to Urbit commit35f70d7 and their
bytes match the earlier native fixture hashes. This corrects an implicit dependency
on the base desk's libraries without adding a full repository mount. Upstream
license/provenance is retained separately from the Clay payload. Source/hash and
CLI/stdio parity race fixtures pass; native app state/reload/restart and current
managed-runtime execution remain to qualify. Fresh native inputs were located in
the existing immutable cache and independently rehashed; no new runtime was
started. Private paths and storage measurements remain outside Git.

## M584 — Gall JSON v2 native loop (2026-11-17)

The separate eight-file5528-byte `gall-state-v2` sample passes the named fresh
native loop: scoped policy, staged commit, separate native compilation, activation,
authenticated read/poke/watch, deliberate rejection without state change, changed
code/on-load verification, clean same-pier restart and state/session/subscription
reconciliation. Own channels/session and final runtime shut down cleanly. Native
effect inputs are audited before dispatch; no Unix mount occurs and autocommit
stays inactive. Frozen v1 bytes/status remain unchanged. The separate report binds
the definition/source/runtime/test/proof hashes and is returned with v2 through
the shared CLI/MCP sample handler. Its authoring-time pending field is immutable;
the later qualification report states the actual scope.

Seven sequential capsules exposed and retained two unresolved activation receipts,
a custom mark JSON mold-shadowing error, a new diagnostic's unavailable type alias,
and missing JSON/MIME scaffold conversions before the complete pass. The final
seventh capsule also passed cold activation before the read-only compiler probes.
Later success
does not rewrite uncertain originals or prove their cause. All six stop cleanly.
See the [native diagnostic and limits](diagnostics/2026-11-17-gall-state-v2-native.md).
Managed lifecycle is synthetic; container/installed upgrade,
multi-peer behavior, state migration, clients and Grubbery remain unqualified.
This supersedes M583's pending native scope only for v2's exact reported inputs.
The48GiB temporary aggregate cap, shared-cache accounting, one active capsule,
4GiB/no-swap/one-core boundaries were enforced. Production/manual-review state
and all excluded ships remained untouched.

## M585 — Confined shared staging parity (2026-11-17)

Shared `desk.plan-stage.v1` and `desk.stage.v1` now reuse the existing collector,
manifest/bundle and runner cache contracts. CLI/MCP startup must explicitly open
named local source roots; calls select only relative ordinary subtrees. Pinned
directory handles prevent root-path replacement from rebinding a running server.
Traversal, absolute tool paths, links, unexpected files and existing byte/tree
limits refuse. Planning transfers no source; staging changes cache/journal only,
with no Clay commit or Unix mount. Complete source selection and native plan are
returned together. Typed pre-upload failures establish no upload dispatch; later
transport failures retain uncertainty and the original stage ID/digest.

The actual CLI dispatcher plans, official-SDK stdio MCP uploads with its response
deliberately discarded, and CLI reads the same verified durable receipt without
source authority. Changed source refuses before a second upload. Confinement,
directory replacement, links, bundles, existing collector compatibility and
cross-interface tests pass under the race detector. These inert tests complement
M584's real collector/staging native composition; they do not establish named
product-client, installed upgrade or complete issue19 coverage. See
[current interface coverage](core-interfaces.md). No new persistent format or
policy grant is introduced by pinned local roots.

## M586 — Recording lifecycle prototype (2026-11-17, provisional)

Four sequential private inert capsules compare pipe, controlling PTY, installed
script2.42.4 and tmux3.8. They create no ship/container and cannot access host
HOME, sockets or existing tmux. The final run retains350-line bursts,12288-byte
lines, diagnostic atoms, fake credential strings, redraws and stderr in all
backends. Monotonic timing/offsets cover all raw pipe/PTY/tmux bytes with a UTC
origin. Pipes retain stream identity; PTYs combine streams and transform LF.

Signalling the script wrapper yields actual inner exit7 but wrapper0 and omits an
accepted final write. Stopping the private tmux server delivers HUP, observes
inner129 and an EIO final write; its client0 means submission only. Direct pipe
and PTY TERM preserve observed7 and final output. The first pipe trial timed out;
its cause remains unresolved, while the next three pass with exact owned-target/
signal-mask and later adopted-child exit evidence. Do not erase the failure or
infer a universal behavior from these installed versions/stop paths.

Proceed provisionally with a runner-owned external stream recorder attached before
container Start, preserving the native signal/TTY/lifecycle contract. Actual
Docker buffering/bootstrap, recorder loss, host restart, retention/pressure,
persistent format, APIs/UI/export and installed upgrade remain open. This is an
architecture prototype, not shipped recording support or production permission.
See [recording contract and evidence](runtime-recordings.md). Capsules enforce
256MiB/no swap/one CPU/64 tasks,16MiB declared scratch and the temporary48GiB
aggregate cap. Only known private child/server PIDs are signalled/reaped; no host
process or tmux inventory is taken. All capsule parents exit after collection.

## M587 — Recording frame/transport source foundation (2026-11-17)

`internal/recording` adds bounded original-byte frames with exact session/metadata
and segment anchors, sequence, receive wall/monotonic timing, stream kind and a
chained digest. Failed/short writes poison the encoder; partial/corrupt readers
retain the last complete offset/digest and never rewrite or accept a partial tail
as EOF. Encoded completion is distinct from durable filesystem sync. Wall-clock
reversal is allowed independently of monotonic ordering. Gap/exit payloads remain
opaque here and need typed storage/runtime validation.

A separate non-TTY Docker decoder preserves raw stream labels, bursts, long lines,
diagnostic/fake credential text, controls and invalid UTF-8 without tail/line limits
or redaction. Declared transport payloads stream through64KiB chunks. Interrupted
payload prefixes remain recorded, write failure reports known missing bytes, and
there is no automatic replay. EOF is not proof of process exit. The library opens
no host path or transport and runs no effect. Focused race and static checks cover
partial boundaries, corruption, rebinding/reordering, bounded lengths, injected
disk-pressure/short writes, segment linkage and cancellation.

This source foundation creates no production/DEV recording, persistent migration
or Hello capability. Owner-private filesystem/index/locking/cursors, lifecycle and
gap validation, retention/reserves, sync/reopen, runtime capture, APIs/UI/export and
actual Docker/Vere/installed upgrade remain open. See
[format scope](runtime-recordings.md). M586's backend choice stays provisional;
these unit tests do not broaden its native fixture claim.

## M588 — Shared bounded project creation (2026-11-17)

The legacy progress loop and new CLI/MCP begin/advance/status/wait operations now
share one coordinator and the original deterministic native request IDs. Begin
checks required phase adapters before reservation; optional declared application
preflight refuses before boot, while generic projects remain usable. Advance
submits at most one missing phase. Guarded original reservation lookup, complete
history/actor/revision/digest checks and contiguous receipt phases prevent a new
intent from being inferred after disconnect. Failed/interrupted phases stop at
their original receipt. Bounded saved-receipt waits never advance or cancel work.
Initial completed receipts do not establish current runtime/application health.

Race fixtures reconstruct the coordinator at every lost phase reply and reject
foreign, duplicate, out-of-order and truncated history. Actual CLI/official-SDK
stdio subprocesses exercise app refusal, generic reservation, lost prepare reply,
process exit, CLI lookup, MCP wait/reconnect, subsequent phase and interrupted
refusal without repeating an effect. Legacy progress/resume fixtures pass.
These are inert protocol tests; installed-runtime creation/recovery/upgrade still
requires the separately isolated disposable guest. No new persistent workflow
format or native lifecycle contract is introduced. This narrows M581/M585's
compound-creation coverage gap, without claiming issue19 complete.

## M589 — Guarded installed upgrade/Stop harness (2026-11-17, unqualified)

A new offline guest gate binds actual published0.4.0 tools, matching immutable
runtime/pills and a clean exact development source snapshot. It reuses only the
verified pristine not-onboarded base through a fresh COW overlay, with no NIC or
host state/socket access. One CPU,8GiB guest/10GiB scope RAM, no swap, bounded
scratch/log/time and aggregate shared-cache accounting are explicit admission
inputs. The authorized48GiB ceiling is scoped to this goal. Host/default refusal
tests, syntax, Go compile/skip and static checks pass; **no VM result is claimed**.

The test-only manager delegates to the real backend and blocks before the final
stopped-phase save after a verified native clean exit. Its parent checks exact
child UID/PID/start/executable before killing it. A second helper refuses recovery
observation during exact replay. Both require the precise guest marker, source
pin and fresh two-fake project. There is no production fault switch or database
repair. Published and candidate installed executables remain distinct from this
dependency-injected test helper. Go test lacks automatic VCS metadata, so its
link-time source binding is checked separately before and inside the guest.

The planned gate checks running/stopped upgrade preservation, older-reader
refusal, a real interrupted Stop, supported reviewed resolution, immutable
original/linked receipts, one native SIGTERM, exact guarded SSH replay after
reopen, same-pier resume and clean final shutdown. See the
[gate contract](../scripts/qualification/issues-recovery/README.md). Qualification
must record actual outcomes and failed originals separately; these source guards
do not resolve M579/M580's live/installed deployment gates by themselves.

## M590 — First installed guest failure and bounded inspection (2026-11-17)

The first source-pinned guest at aa71757 installs the actual0.4.0 runner and pinned
image, but the first fresh recommended fake Start fails during pill preparation
with a bounded Docker-command failure. No candidate install, upgrade preservation
or Stop recovery gate is reached. The powered-off original failure disk/receipts
remain evidence. The single virtual CPU may reject the unchanged default two-CPU
contract; this remains a hypothesis. The next capsule uses three virtual CPUs
inside the same one-core host quota; no product resource contract is weakened.

Full raw conversion admission refused before creating a copy because mapped
nonzero ranges exceeded available scratch. A separate512MiB no-NIC reader mounts
the static failure filesystem read-only/noload under a dedicated PID1 and exports
only two fixed bounded saved files. Its first guard rejected a final command-line
token followed by newline, before opening the disk; exact token parsing fixes it.
The next reader succeeds with the original overlay hash unchanged and no services
or ships started. Reproducible inert staging is retired with a private manifest;
the original failure overlay/receipts/results and immutable base stay preserved.
See the [compact failure evidence](diagnostics/2026-11-17-issues-upgrade-first-capsule.json).
This updates M589 with an actual failure, not an installed-support claim.

## M591 — Bounded shareable Stop diagnostic view (2026-11-17)

The shared CLI/API and MCP catalog expose `project.stop-diagnostic-bundle.v1` over
the exact saved operator diagnosis. A pure enumerated projection strips all exact
private bindings, actor/request/member text, endpoints/paths, arbitrary messages,
database bodies and sealed reviews. Known and unknown categories use fixed
messages; saved state never gains live observation fields. A bundle states that
it is not an action review. Reconciliation still requires the private exact fresh
review; exporting a view sends no probe, signal or policy grant and publishes
nothing. This is specific to expanded-member Stop, not universal phase diagnosis.

Focused source/race/static checks and actual CLI/official-SDK stdio parity pass,
including operator refusal and private sentinel text throughout the raw input.
OOM/exit facts stay distinct from historical status and never become Start or
clean-stop authority. No persistent format or installed runner change is needed.

## M592 — Second guest reaches real baseline; correct historical ID use (2026-11-17)

The second fresh guest at d9b536e boots two recommended fake ships using the actual
published0.4.0 runner, then successfully stops one. The three-virtual-CPU/one-core
quota topology passes the first capsule's early Docker failure point. This supports
M590's topology hypothesis without recovering the original suppressed predicate.
Before candidate installation, the harness wrongly inspects the stopped fake's
retained historical container ID; supported Stop has removed that container.
Its bounded Docker diagnostic is “no such object.” This is a harness error, not
upgrade failure. Correct baseline process inspection to running rows only; retain
saved stopped intent, pier/configuration and history comparisons independently.

The failed original is inspected read-only/noload with no old services restarted.
Its saved commands/results and original schema59 SQLite/WAL/SHM bytes are archived
privately before any full-disk compaction. Retire only classified superseded
generated full disks/staging with a manifest; preserve the immutable base, original
store fixture, full compact failure evidence and unrelated/manual-review state.
No candidate install or upgrade/Stop-recovery pass is claimed by this checkpoint.
The [second compact failure record](diagnostics/2026-11-17-issues-upgrade-second-capsule.json)
binds the archived original59 store (two instances/five operations), complete
commands/results and read-only inspection hashes. Integrity and byte preservation
are checked before full-disk retirement. This narrows M589's original process
baseline to live containers without weakening stopped-state preservation.

## M593 — Typed recording attachment/lifecycle metadata (2026-11-17)

Source-only recording frames now validate canonical version1 gap/exit payloads,
rejecting ambiguous/unknown fields, alternate encodings and unsupported metadata
before write. A correct checksum cannot promote semantically invalid lifecycle
data. Unknown missing-byte counts remain unknown and full uint64 counts stay
exact. Stream EOF and failed Start without a launched process cannot become an
observed exit. OOM/dead/native-error facts remain distinct; recorded wall time can
reverse without changing monotonic receive order.

Bounded canonical attachment metadata binds runner/target/configuration, original
operation/request, prepared container, launch/image/runtime/transport and clock
origin, with no argv/environment/secret-content fields. Exact metadata/session
hashes bind a segment; replaced identity/revision/artifact/clock refuses. UTC
normalization is stable. Ownership, live admission, later segment/gap continuity,
pre-container failure diagnostics and every persistent/runtime/API/UI/retention
gate remain separate responsibilities. No path is opened or runtime launched.

Focused race/static tests pass for payload semantics/corruption, unknown coverage,
raw-output preservation, integer precision, identity/revision/artifact rebinding,
clock changes, unsupported metadata and forbidden arbitrary credential fields.
This supersedes M587's opaque lifecycle-payload scope only; there is still no
advertised/shipped recording capability or existing persistent-format migration.
The running installed-upgrade capsule remains frozen at d24c960 and does not
qualify this later source-only library change. See the
[current recording contract](runtime-recordings.md).

## M594 — Exact installed upgrade section passes; staged recovery continuation (2026-11-17)

The third fresh guest qualifies the installed d24c960 snapshot's0.4.0/schema59→60
ordinary-fake path. Running/stopped intent, registrations, configuration, history,
access, engine invocation and running container process survive actual installer
activation and a later manager restart. The actual old writer refuses60 without
changing its dump. Both upgrade witnesses then stop through supported APIs. This
supersedes the installed-upgrade deferral only for these exact artifact/scope
claims; native interrupted-project migration, recovery/app state and desktop/DEV
remain unqualified. The [artifact-bound record](diagnostics/2026-11-17-issues-installed-upgrade-section.json)
keeps the later scenario failure separate from its passed upgrade section.

Before a recovery project, the harness tries a duplicate enrollment of the same
runner and correctly receives the registry's conflict refusal. Switch the exact
existing fixture connection through supported `runner-edit` with runner ID and
revision; do not weaken enrollment protection. Read-only/noload inspection and
private original60-store integrity/byte preservation pass. Original failure
evidence remains unchanged.

The recovery-only continuation uses a new fresh candidate-only guest with3GiB
scratch and the same four frozen artifacts, explicitly linked to the passed
upgrade section. The harness and artifact commits are recorded separately; later
source-library changes are not qualified by this frozen result. Unit guards
refuse changed artifacts/source, failed preservation or old-reader evidence, and
changed enrollment identity before editing. No combined/native recovery pass is
claimed until that separate case completes. Production/manual-review state and
excluded ships remain untouched.

To fit the continuation without removing a selected pier or full reference, the
completed upgrade QCOW is recompressed with its original/base bound read-only.
The first512MiB processing scope hits its memory cap; the original stays unchanged
and only the unverified derivative is discarded. A2GiB/no-swap/one-core retry
passes full QEMU virtual-block comparison and preserves the backing/input hashes
before replacing the archive representation. About1GiB is freed, with original
store/command/result archives retained. This is a content-identical storage
derivation, not a guest restart, filesystem repair or pier mutation.

## M595 — Installed native clean-stop recovery passes (2026-11-17)

The fresh recovery-only case passes on the exact four d24c960 artifacts linked to
M594's passed upgrade section. Restricted access switches with exact connection
ID/revision. Selected installed Gall preflight passes before project creation,
without granting app policy. A fresh original fake and added peer boot; the
original process remains unchanged. A separately hashed test manager delegates
to real native code, blocks after a clean exit before the final phase save, and
only that owned child is killed. The installed manager reopens normal state and
retains the interrupted Stop/fence. No database edit or repair is performed.

Saved diagnosis, fresh exact retained-intent review and ordinary API reconciliation
pass. A new successful linked receipt resolves only that fence; the original stays
interrupted. Reconciliation changes no native process. Exact restricted-SSH replay
works after store/manager reopen with a deliberately refusing observer, then again
under the installed binary. Docker events for the exact owned container show one
SIGTERM. Supported Start resumes the same verified pier and ship with an ordinary
new container, leaving its peer process unchanged. Both fresh ships stop cleanly;
the final installed executable SHA and schema60 match. See the
[artifact-bound native evidence](diagnostics/2026-11-17-issues-native-stop-recovery.json).

This supersedes M579/M589/M594's native recovery deferral only for the clean new
retained-intent path. The interrupted process is the test helper, not the installed
executable; real native backend/API/SQLite/SSH and installed reopen/recovery are
composed explicitly. The historical race remains a hypothesis; native forced read
timing, legacy missing-intent runtime migration, full native refusal/dropped-reply
matrices, managed app-state, later artifacts and desktop/DEV remain outside this
case. Source/inert matrices remain distinct. Recordings/Grubbery/broader interfaces
and containment are still active work; this is not issue-wide goal completion.

## M596 — Bind each bounded creation call to one native phase (2026-11-17)

The initial M588 advance input could mean “next missing phase at call time.” If
the reply was lost and its phase finished before retry, an unchanged call could
select the next phase. Tighten this unpublished shared interface: advance now
requires the exact `native_request_id` returned by begin/status. Repeating a
completed ID returns only that original historical completed handle, without
submission or dependency preparation. Another phase needs its newly selected ID;
foreign/future/out-of-order bindings refuse before dispatch. Legacy progress
orchestration, native request derivation and durable formats remain unchanged.

The regression completes a deliberately lost-reply phase before any read/retry,
then verifies repeated calls cannot move ahead. Actual CLI/official-SDK stdio
tests enforce the same rule and require explicit selection for the next phase;
missing bindings refuse. Existing pending/interrupted/history/wait cases pass
under the race detector. This supersedes M588's ambiguous advance-call semantics.
It is a later client/core source change, not part of the frozen d24c960 native
artifact qualification. Matching final artifacts/interfaces remain required.

## M597 — Private recording files and verified source paging (2026-11-17)

The unwired Linux recording store pins a private owner directory, exclusively
creates unique sessions and syncs immutable metadata/header/directory entries.
Append serializes original frames, checks stream/container and current file/lock
identity, admits explicit byte/free-space ceilings, and fsyncs before success.
Changed, exposed, linked, unexpectedly resized or replaced files refuse; a failed
writer becomes unusable. Existing/incomplete sessions are never reused or repaired.

Cursor pages verify the bounded original prefix and exact metadata/sequence/digest.
Reopen/reconnect and live concurrent paging retain raw bytes without duplication.
Partial/corrupt tails stay visible and unchanged; a lost cursor refuses without
rewind. Writer presence and snapshot EOF are separate from runtime exit. Source
race/static checks pass for privacy/binding, long lines/raw atoms/controls, paging,
replacement/links, quota and reserve refusals. Test temporary roots were initially
0755 and correctly refused; fixtures explicitly become0700 rather than loosening
the store's ownership rule.

This is one bounded segment with a verified scan, not a persistent index/rotation,
writer continuation, per-host/ship retention, incident pins, authorization API or
capture daemon. Before-container errors, runtime lifecycle, pressure recovery,
fsync/power/crash qualification, safe views/exports and UI remain open. Free-space
admission is not an exclusive filesystem reservation. No runtime or installed
state is touched; only synthetic temporary recording files are created in tests.
This narrows M587/M593's filesystem/cursor gap without advertising issue20 support.

## M598 — Freeze Grubbery source before live adapter qualification (2026-11-17)

Review of nisfeb/grubbery7f7701d546ce115eea5f37f66465b362cb121774 finds a
different HTTP/namespace contract from the ff2b66b exercise. Its documents embed
older code references; actual pinned Hoon is authoritative. Independently verified
all1515archive blobs against the exact Git tree, then froze only the1443-file,
8013304-byte Clay desk manifest with the upstream MIT notice. Private downloaded
source stays outside Git and live mounts; the compiled definition contains hashes.

Shared offline framework metadata and confined source verification reject missing,
changed, additional, linked or unsupported files. A caller cannot replace the
compiled definition with its own manifest. Existing stage planning/transfer still
independently bind source bytes. Focused source/race/static checks and actual
CLI/official-SDK stdio parity verify the complete downloaded desk without creating
registry state or connecting to a ship. No framework source is executed or installed.

This narrows issue18's reproducible-input gap, not its native readiness gap.
Build/code namespace, loaded instance/state, shell-owned approvals, actual weirs,
discovery/aliases and served bytes remain distinct. Approval is replacement of the
selected declared subset, with a separate reload; trusted built-ins are not a
substitute for closed desk apps. Grubbery preflight remains unavailable pending
versioned live effects, durable uncertain-write recovery and native qualification.

## M599 — Bound the fresh Grubbery bootstrap gate (2026-11-17)

Add a separately named fresh-only bootstrap capsule for M598's exact complete
framework desk. Host/copy/kernel source checks precede API staging, commit and
cold activation. No Unix mount, app-specific write or permit authoring is allowed
in this gate. Lost activation replies retain the original receipt and permit only
bounded observation/reconciliation. A fixed kernel diagnostic read is independent
of the shell; its owner session is cleaned up. Host/source/admission tests pass;
this records the harness before execution, not a native bootstrap pass.

Storage admission first refuses the proposed archive-processing allowance before
creating a converter. Retire only task-owned inert staging, exact duplicate
immutable inputs and superseded non-frozen source-test executables, with private
per-file hash/inode manifests. Keep all selected piers/VMs and original stores.
The completed recovery-only VM is then recompressed under2GiB/no-swap/one core.
Full virtual-block comparison passes, original/base hashes stay unchanged before
replacement, and the original SQLite archive still matches its recorded SHA.
The retained representation saves1581604864bytes; the public native recovery
record explicitly links old and new overlay hashes. No guest is booted or mounted
during this storage derivation. Native Grubbery results remain pending.

## M600 — Preserve scaffold ownership before framework replacement (2026-11-17)

The first fresh Grubbery gate stages the exact source, then correctly refuses its
commit: mar/hoon.hoon and sys.kelvin differ from unowned fresh-desk scaffold files.
No framework commit or activation is dispatched; shutdown is clean. Preserve that
original failure. Source review separately finds the framework reads a literal
grubbery Clay desk; random names cannot satisfy that pin. The next fresh-only gate
uses the required name and keeps every other native namespace boundary.

Add a typed, pure historical Clay source read with explicit paths and64-file,
256KiB bounds, then API/client/collection/shared-service bindings. Operator role
and exact process/pier/Clay baseline bracket the read. Clients reject changed
identity, paths, revision, bytes/hash or timestamps. The output contains explicit
canonical base64, including empty/binary bytes. An initial []byte wire shape fails
the inferred output schema in actual interface testing; the explicit base64 field
fixes this before native use. Source/race/static checks, actual CLI/stdio parity
and the private API refusal/no-mutation matrix pass.

The native continuation will export only the two original conflicts, stage them,
explicitly adopt their verified equality under the existing policy gate, then
review a new full-framework commit. This does not acquire unspecified ownership,
mount source or replace differing files by relaxing the conflict planner. Source
export is private operator data, not a shareable/redacted export. Its current
fixtures do not establish native conversion or bootstrap success; those results
remain pending. Grubbery preflight is still unavailable.

## M601 — Fit the bootstrap continuation before process admission (2026-11-17)

After M600's source checks, the host refuses the proposed2GiB reservation before
creating a capsule, copying inputs or starting a scope. No second native result
exists yet. Retire only the completed task-owned source-export race fixture
executable with its private hash/inode manifest. Reduce this fresh single-ship
bootstrap's declared additional scratch to1.5GiB; preserve the48GiB aggregate cap,
periodic allocation checks, memory/deadline/one-active-runtime rules and original
failed capsule. Exceeding either scratch or aggregate storage still interrupts
the new owned experiment and retains its uncertain/failure evidence. This is a
smaller admitted experiment, not an unmeasured allowance or a native pass.

## M602 — Pinned native Grubbery bootstrap passes with explicit limits (2026-11-17)

The fresh f9ca8a7 test artifact under the separately pinned a658341 admission
harness passes within1.5GiB declared scratch and48GiB aggregate storage. The Go
artifact inputs are unchanged between those commits. The exact complete framework
desk is reverified. Ordinary API source export reads the two original scaffold
files; narrow explicit staged adoption verifies their equality without changing
Clay. Full framework commit is observed with the frozen source digest and no
ownership conflicts. Autocommit remains inactive; no Unix source mount is created.

Cold activation exceeds its HTTP reply. The original receipt progresses through
planned/dispatched/uncertain/observed with exactly one native dispatch; supported
read-only reconciliation completes it without reinstallation. Both declared Gall
agents are live. An independently authenticated fixed kernel HTTP diagnostic read
reports1335root-code files,0failures (223vases/1112assets), including compiled
shell, desk and nexus-web. The private owner session logs out and the fresh Vere
shuts down cleanly. See the exact-input native bootstrap JSON diagnostic.

The original runtime also reports four failures under /apps/shell/tools/code.
Those are a separate namespace, not disproven by a clean root-code report; the
four individual errors were not queried in this capsule. Preserve that warning
and the original pre-adoption failure. This supersedes M599–M601's bootstrap and
native source-read deferral only. App deployment/state/permits/routes/assets,
publication/discovery/peers, managed lifecycle, installed/DEV qualification and
issue18 completion remain open. Grubbery preflight still reports unavailable.

## M603 — Separate bounded app code and stable state identity (2026-11-17)

The source-only Grubbery adapter contracts now distinguish an app code namespace
from a framework Clay desk. Review requires the exact staged manifest and at most
64files/512KiB (64KiB each), a bounded duplicate-free bill and locally supplied
nexus source for every declared instance. Framework/repository root metadata,
controller-owned version markers, unresolved external nexuses, changed bytes,
nested code bundles and oversized/partial stages refuse. Binary assets remain
lossless. These checks establish input boundaries, not native compilation.

Stable candidate namespace addresses bind runner/mode, exact pier continuity,
project creator/generation/member, framework desk and explicit app ID. Actor,
container, configuration, policy and framework-receipt updates do not rename saved
app state. Immutable source releases are outside the live desk mirror/data and
have distinct original-request addresses. A derived name is never proof of
ownership: persistent ownership, current native observations and refusal of an
existing unowned root are still required before effects. Pure race/static tests
check these boundaries and the exact match to the frozen framework contract.

No app namespace, HTTP write, permission, persistent format or advertised live
capability changes in this slice. Durable step dispatch/reconciliation, runtime
and public interface wiring, the native app sample and its full update/restart/
permit/route/peer qualification remain active work.

## M604 — Construct exact staged app phases without granting dispatch (2026-11-17)

The unwired pinned Grubbery planner now derives an immutable release tree from
the original request, with stable per-step IDs and a complete plan digest. Code
files carry the native hoon blot; binary assets remain mime, and HTML/JS/CSS/JSON
have explicit MIME. Compile observation precedes the controller's version marker,
then the framework source fiber is poked as json rather than overwritten as mime.
An exact followed-source read remains separate from application readiness.

Existing unowned/conflicting roots or changed ownership markers refuse planning.
Updates create new immutable code and leave prior code and saved instance data
untouched. The plan contains no sand/permit/alias/cull effect and never force-
overwrites a source file. Tampered paths, control bodies, source, bindings, epoch,
step IDs/order or digest refuse request construction. Focused source race/static
checks pass. These are pure constructors, not a dispatch authority or a native
app test; derived markers still require actual persistent ownership and fresh
native observation before effects.

Next integrate an authoritative durable coordinator: claim the app namespace,
save dispatch before I/O, admit only the exact original step, preserve uncertain
writes, reconcile by bounded read and replay completed receipts without source
cache/runtime calls. Persistent/migration and runtime/interface wiring remain
required. No namespace is created and no HTTP request, policy or installed
capability changes in this slice. M602 remains bootstrap-only evidence.

## M605 — Guard exact app dispatch and cache-free receipt recovery (2026-11-18)

The source-only app coordinator now defines atomic namespace claiming and
revision-CAS append contracts, immutable creation headers and hash-chained phase
events. Dispatch is allowed only for the exact selected pending original phase,
after exact staged-byte validation and fresh trusted admission. Only the worker
that wins the dispatch CAS may call the native adapter; losing callers return
saved progress. Dispatch carries explicit plan/step context rather than shared
mutable target selection. Unknown native outcomes retain their original events.

Completed/uncertain advance replay, matched Begin replay and saved lookup use no
source cache, probe or native dispatch. Read-only reconciliation needs no cache,
accepts only exact bound after-evidence, and appends one resolution without
replacing the original dispatch/uncertainty. Repeated unproved reads append no
duplicate history. A different unfinished operation cannot claim the app namespace;
only full original phase completion releases its lease. Changed Begin bodies,
targets/actors, future/foreign phases, bytes and malformed history refuse.

Focused race/static fixtures pass for concurrent claims, native reply loss,
coordinator reopen, journal commit/reply loss before native I/O, full completion,
corrupt/reordered/rebound header/event chains, clock correction and cache loss.
The model store tests transaction semantics only: no SQLite/fsync durability,
migration or native app deployment is implied. Source caps are checked before
hashing oversized data, and attested metadata validation stays separate from
new-write byte authority. Next wire the persistent runner store/lease, migration
and exact native transport/observers plus public interfaces, then qualify the app
sample. No persistent format, namespace, policy, runtime or advertised capability
changes in this slice; M602 remains the qualified bootstrap boundary.

## M606 — Derive app ownership from original phase receipts (2026-11-18)

The source-only ownership projection requires observed original marker effects,
exact stable namespace identity and every source/version predecessor in trusted
creation order. A reservation, dispatched marker or uncertain marker creates no
ownership. Read-only marker reconciliation can establish its original proof;
later updates retain that proof and advance only the completed source generation.
Missing, duplicate, reordered or incomplete predecessor history refuses. The
focused race/static tests pass, including partial marker history and updates.

Persist this as immutable operation headers and append-only phase events with a
transactional namespace lease; derive current ownership instead of adding a
second independently mutable ownership row. Schema61 must preserve all existing
schema60/59 history and make older writers refuse. Exact original60-writer fixture,
rollback/reopen/corruption and matching-artifact qualification are still required.
The native adapter must independently check the actual namespace and current role,
framework/process/pier, policy and resource limits; a saved projection alone is
never runtime authority. Safe abandonment of failed compilation before source
activation also remains required before exposing the app workflow. No persistent
format or advertised capability changes in this source-only slice.

## M607 — Audit preactivation abandonment and freeze original60 (2026-11-18)

The source coordinator can abandon a reserved intent without runtime access. Once
writes exist, every submitted write and created-root marker must have its exact
original after-proof, and a trusted bounded read must establish the original live
source baseline unchanged. Unknown writes and any dispatched source activation
refuse abandonment. Failed compile/read uncertainty can remain in immutable
history. An appended terminal audit releases only that namespace lease; it grants
no deletion, marker adoption or native effect. Source ownership origins remain,
while the applied generation does not advance. A replacement uses a new original
request/release and preserves failed staging plus live application data.

Focused race tests cover cache-free terminal replay, changed baseline refusal,
retained marker ancestry, unknown writes/activation, and stale worker dispatch CAS
loss to abandonment. This is source-only; the durable store and native observer
remain required. M605's completion-only lease rule is superseded by this narrow
validated preactivation alternative. Abandonment is distinct from source-followed
completion and cannot become application readiness.

Before schema61 changes, actual frozen d24c960 source with a declared synthetic
helper wrote a populated60 fixture containing original59 history plus a real
linked Stop reconciliation. The isolated writer passed with no host service or
native ship access. Its source, helper, executable, SQL and packed fixture hashes
and row counts are recorded under testdata/grubbery-app-original60. One unexecuted
wrong-working-directory build was classified and replaced before this run;
private provenance records that correction. Preservation, rollback, repeated
reopen and genuine original-reader refusal remain required for the successor.

## M608 — Persist immutable app phases and qualify schema61 ancestry (2026-11-18)

Schema61 adds separate app intent headers, append-only chained events, exact
namespace leases and a transactional creation clock. Header stamps bind creation
order and immutable context. Opening validates the complete original marker and
version ancestry, framework commit receipt/source/pier baseline, clock, events
and leases. Missing/corrupt/mixed state refuses without repair. There is no
mutable current-owner row, queued app replay or native call during opening.
Creation claims the namespace atomically; revision CAS appends an event and
releases only the exact terminal owner's lease in the same transaction. The
SQLite store retains FULL-synchronous acknowledgement and private existing store
ownership requirements. This adapter remains internal and unwired to public app
mutation APIs; live Grubbery preflight remains unavailable.

Focused race/static gates pass for immutable header/history across repeated
reopen, concurrent phase CAS, failed append rollback, namespace conflict,
completed and abandoned lease release, wrong runner/mode/framework/pier/baseline,
missing/rebound evidence, schema/table/event/clock/lease corruption, and an actual
child process exiting immediately after dispatch commit without closing SQLite.
That last case retains the original dispatched fence and cache/backend-free
replay; it does not establish host power-loss behavior. Retained original59 and
actual original60 writer fixtures preserve every original row, linked Stop
resolution and exact replay, with rollback/refusal of unknown successor tables.

The fresh actual60→61 rehearsal passes with a populated uncertain app journal;
the genuine d24c960 reader refuses61 without changing closed database bytes.
Source inputs, helper/artifact hashes and limits are linked in the source
[migration evidence](diagnostics/2026-11-18-grubbery-app-store-migration.json).
Its first capsule correctly refused a reused synthetic test receipt ID, then
received a distinct deterministic fixture ID. Two subsequent sidecar comparisons
found only empty verifier-created WAL/SHM removal, with identical main database
bytes. Immutable reads of fully closed fixtures correct the verifier; no database
or sidecar repair/deletion was used to obtain the passing result. Earlier failures
and before/after hashes remain in private evidence.

The older source inventory now explicitly targets61; RC.12/schema20 qualification
and its release pins remain frozen. Original-writer preservation/replay and
invalid-identity/future-schema byte-preservation regressions pass through the
retained schema45 inventory. Focused synthetic compatibility, legacy runtime
settings and import-completion reopen tests also pass. Those synthetic tests omit
only verified-empty successor app tables before assigning an older test marker;
they never claim original-writer evidence or authorize relabeling a user store.
The first combined regression group exceeded its aggregate four-minute deadline;
separate bounded groups subsequently pass with individual results retained.

This extends development source-reader evidence for retained59/60 state, not
installed/native Grubbery qualification or a stable support promise. Matching
final artifacts, native admission/observers/transport, actual app source and
permit/route/served-byte/state/restart/peer gates remain active. Production and
installed DEV artifacts/state have not changed.

## M609 — Retire superseded task piers before app contract tests (2026-11-18)

After M608's source gates, retire only the completed fresh-fake Gall dependency
failure and first Grubbery precommit ownership-refusal piers. Later passing cold/
warm Gall and exact Grubbery bootstrap references retain their full piers. Every
removed file and directory is bound in a private device/inode/allocation/hash
manifest; original runner stores, receipts, source inputs, logs, guard and shutdown
results remain byte-verified. Both removed piers had confirmed clean shutdown.
A redundant task-local pill and duplicate host workflow executable are verified
against retained exact references before removal. A completed source-test binary
is also retired with its source and passing logs preserved. This frees about1.4GiB;
no installed/manual-review/production data, unrelated experiment or user ship is
part of this retirement. All original failure results remain failures.

The next single-peer app contract case still needs declared storage admission
under the temporary48GiB goal ceiling. M608 establishes source persistence only.
Before public app writes, distinguish a native pre-effect refusal from a lost
reply, and establish exact native ownership after uncertain root creation. An
existing derived namespace or matching content alone must not become adoption.
Native parent creation, root markers, code compilation, followed source and
retained application state need fresh isolated contract tests; these are research
questions, not qualified behavior. The four nested shell-tool failures remain
unclassified. No native app capability or installed update is implied.

## M610 — Prepare a fixed fresh Grubbery app contract case (2026-11-18)

A separately named native case now composes the exact framework bootstrap with
fixed private HTTP contract checks. It records each request hash before one send,
uses an ephemeral owner session, refuses redirects/replay and remains confined to
the fresh guarded kernel. Proposed checks cover atomic plain-parent/marker file
creation, separate nested-tool compilation diagnostics, a tiny local JSON-state
nexus, immutable versioned source, a closed framework desk wrapper, explicit
source following, a known duplicate-create refusal, code-label change and saved
state across source update. No permit/alias/route/public app API is introduced.

Launcher syntax and its11 input/isolation tests pass; the candidate is not yet
native-qualified. Its native operations are a contract experiment with synthetic
runner/lifecycle bindings, not the supported production adapter or public sample.
Original bootstrap-only case identity remains separate. One fresh capsule, the
existing4GiB/no-swap/one-core boundary, original pinned inputs, bounded deadlines
and1.5GiB declared scratch still apply under the temporary48GiB goal cap. Unknown
outcomes preserve original evidence and terminate through the owned supervisor;
there is no mutation retry, source mount or adoption of existing namespaces.

## M611 — Freeze actual61 history before correcting the app plan (2026-11-18)

Pinned source review identifies a concrete draft-plan defect: +version-text in
Grubbery7f7701d accepts an object's `version` string/number, while plan1 writes
only `epoch` and `source_digest`. That object does not signal a release. Native
qualification has not established otherwise, and public app dispatch remains
unavailable. Correct this through a new explicit plan/journal generation; retain
old intent/receipt bytes and refuse a fresh legacy pending dispatch rather than
silently rewriting its payload. Known native pre-effect refusal also needs a
state distinct from an unconfirmed transport outcome before supported dispatch.

Before those changes, freeze the populated actual61 journal1 database from M608's
passing original-writer rehearsal, with original59/60 ancestry and uncertainty.
Every compiled product Go input matches frozen6407cda; exact invoked test helpers,
artifact/build-input/SQL/packed hashes and source scope are recorded alongside it.
The original reader executable remains retained. Successor preservation and real
old61-reader refusal still need qualification; the fixture alone makes no such
claim and contains no installed or native application state.

The fresh app-contract launch first refused its scratch reservation about5MiB
over the temporary cap, before capsule creation. A redundant31MiB historical
source extraction was verified against every original Git blob and declared
helper before retirement, preserving the actual old60 writer, fixture and source
commit. The unchanged frozen app experiment then admitted within48GiB. It is the
only active runtime case. Its result remains pending; preserve the original run
and do not infer app behavior from bootstrap progress or source review.

## M612 — Correct version signals without rewriting legacy app intents (2026-11-18)

The first frozen app-contract capsule ends failed with clean shutdown. Exact
framework bootstrap passes again; one file make creates its plain parent and
marker, and the tiny local state nexus compiles as a vase with four native
scaffold marks. Following remains null with the prototype's missing `version`
field, consistent with pinned +version-text. No saved-state/update/route/permit/
peer or managed adapter claim follows. The original source, effects, failure and
proof are retained; see [the partial native record](diagnostics/2026-11-18-grubbery-app-contract-first-capsule.json).
The separate nested tool report identifies pytz, its rules/calendar dependents
and read-terminal's incompatible extra to-lines arm. These pinned modules remain
unavailable; neither framework bytes nor their failure status is changed.

New app plan/journal2 carries an original-request-bound native `version` value
alongside epoch and source digest. Its first plain-root marker uses the observed
single file/parent call, still requiring refusal of an existing unowned root.
Legacy plan1 is reproduced exactly for history validation; fresh pending dispatch
and native request construction refuse instead of rewriting old source payloads.
Saved uncertainty replay, bound read-only recovery and narrow safe abandonment
remain available. Schema62 fences older writers unconditionally while preserving
the same61 table shape, original headers/events/leases and observed marker ancestry.

A validated pre-effect native rejection is now an immutable `refused` state with
its response digest, distinct from lost/unknown transport. Repeating it loads no
source and performs no native call. Later matching content cannot reconcile it
into ownership. A rejected first write can be audited as no-effect abandonment,
releasing its exact lease while retaining original dispatch/refusal history.
Any dispatched activation still forbids abandonment. A trusted native adapter
must classify only complete supported pre-effect responses; errors/5xx are not
proof of rejection. No public mutation adapter is enabled by this source change.

Focused pure/source race and real SQLite tests pass for native-readable markers,
request binding, atomic-marker planning, known-refusal persistence/reopen, no
later adoption, legacy refusal/replay, lease release, original61 preservation and
corruption rollback. The fresh actual60→62 rehearsal preserves original rows and
uncertainty; genuine60 and61 readers refuse populated62 unchanged. Exact test
artifacts/source hashes and limits are in the [migration record](diagnostics/2026-11-18-grubbery-app-store62-migration.json).
The corrected native contract case, native observers/transport and full app/
permit/route/state/restart/peer/installed/DEV gates remain active. Earlier schema61
and native failure evidence stay bound to their original artifacts.

## M613 — Preserve compact full evidence and prepare corrected native case (2026-11-18)

The original bootstrap-only full pier is superseded by the later app-contract
capsule's independently passing exact bootstrap section. Its original store,
source artifacts, receipt/proof and logs remain byte-verified while only that
fresh fake pier and verified duplicate task inputs are retired. Keep the current
failed app-contract full pier intact. Private before/after manifests bind every
retired object and preserve the original outcomes and source identities.

The completed Gall cold reference is retained as a full sparse lossless archive,
not discarded. A bounded1GiB/no-swap/one-core archive scope verifies all15 file
contents against original SHA256, every directory and ownership/mode, plus original
store/proof hashes before retiring the expanded tree. The archive retains about
193MiB instead of563MiB; original metadata/inodes and exact archive hash are in its
private manifest. It is evidence only: restoration creates new filesystem objects
and never grants runtime ownership or automatic resume authority. Tombstones
prevent older capsule launch instructions from reusing retired/archived trees.
No installed, manual-review, production or unrelated user data is involved.

The separate grubbery-app-contract-v2 case keeps the original failed prototype
available and supplies the explicit native version value/original request in its
new marker. Launcher input/isolation tests pass; native following, saved state
and source-update behavior remain pending for this corrected case. Preserve the
same fresh single-peer boundary and declare full scratch before launching it.
The framework pin and unavailable nested tool modules remain unchanged.

## M614 — Preserve interrupted activation and separate supervisor lifetime (2026-11-18)

The second fresh app-contract capsule's caller exits143 during framework
activation, before app checks. Its exact owned scope is confirmed inactive/dead
with no cgroup; the scope journal records about180seconds wall time and3.3GiB
peak below its4GiB cap. Cause is unestablished. An external command lifetime is a
hypothesis, not proof of a runtime/app defect. No clean-shutdown/final-exit receipt
was saved, so clean shutdown stays unconfirmed. The original activation receipt
is uncertain; no install or app write is replayed. A copied complete SQLite/WAL
snapshot passes quick_check while every original database file remains unchanged.
See [the interruption record](diagnostics/2026-11-18-grubbery-app-contract-launcher-interruption.json).

The source qualification launcher now converts SIGTERM into a recorded exception
and cleanup path. An inert owned-child signal test verifies finally execution;
all12 launcher tests pass. A separate fresh-only background entry submits a
transient user supervisor with512MiB/no swap/25%-CPU and1250second deadline,
retaining source/artifact/command hashes and private logs. Its child still needs
the original independent4GiB/no-swap/one-core namespace/storage admission. This
keeps the long native experiment independent of a short-lived CLI/MCP/tool caller;
it does not install or change an existing service. The next capsule is fresh,
with the same frozen f00f5fb helper and a separately frozen harness.

Storage remains under48GiB without changing the ceiling. The completed first
app-contract failure retains its full1450-file pier as a verified sparse lossless
archive (about253MiB), and remains a failure. Original source-reader executables
are also verified lossless archives; decompress to fresh private executables and
verify original hashes before another actual-reader rehearsal. Preserve the full
unconfirmed second pier and original WAL/receipts. Only verified duplicate task
inputs are retired. These archives/tombstones grant no runtime resume authority;
production, existing user ships and installed DEV remain unchanged.

## M615 — Qualify native source following and retained app state (2026-11-18)

A new fresh capsule under the independent supervisor passes using the same frozen
f00f5fb helper and corrected version marker. Exact source compile, plain-parent
marker creation, two immutable release versions, framework desk following,
initial/saved JSON state, unchanged saved entry after code update, new code label
and known duplicate-create409 with unchanged marker all pass. The workflow makes
28 bounded native calls and the owned ship shuts down cleanly. Pinned framework,
Vere/pill, source/artifact/proof/effect hashes and limits are in the
[native core record](diagnostics/2026-11-18-grubbery-native-app-core.json).
The earlier null-version failure stays failed; the separate interrupted capsule
stays uncertain and was neither resumed nor reinstalled.

This is kernel/API contract evidence with synthetic lifecycle/runner bindings.
It does not qualify the durable managed adapter, public CLI/MCP mutation parity,
permits/aliases/routes/served artifacts, runtime restart, peers, installed upgrades
or DEV. Grubbery preflight remains unavailable. The four nested tool modules
remain failed and do not become healthy because this tiny app works. Next wire
only these typed checked native operations to exact durable phase selection and
fresh runtime/policy/namespace observations, then qualify the remaining workflow.

## M616 — Bind native app calls and observations without advertising dispatch (2026-11-18)

New internal typed reads derive only selected app roots, markers, source/version,
original phase after-reads and the bounded staged manifest. They carry full binding
and original plan/step digests, work without source bytes, and cannot accept caller
URLs/commands or construct writes. The owner-session client constructs exact phase
requests from verified bytes, preserves binary payloads, never retries, separates
complete pinned pre-effect rejections from uncertainty, and treats404 reads as
absence rather than ownership. Private response bodies and compiler errors are
excluded from implicit JSON output; fixed errors do not reflect transport details.

The unwired Linux transport pins one method/path/query/content-type/body hash to
one already verified private container endpoint. The existing desk HTTP routes
are not widened. Cookie and binary payload remain on separate stdin descriptors;
fixed curl disables config/proxy/redirect/retry. Process, pier and framework
baseline are checked around ephemeral owner-session requests. Session cleanup is
reported confirmed/unconfirmed; the future managed adapter must retain its outcome
rather than invent successful cleanup. No public policy or live capability changes.

Bounded native JSON/tree/build decoders reject duplicate/case-alias/unknown fields,
changed namespaces, counts, paths, marks and compile status. Exact numeric spelling
preserves large identity integers. Application source and the kernel's four forced
foundational marks remain distinct; attempts to override those marks refuse before
I/O. A parsed compile report alone is insufficient: staged bytes, namespace
ownership and fresh runtime/policy admission still need independent checks. The
backend must also handle directories proven generated by its own code-namespace
creator, not adopt an arbitrary preexisting path by matching content.

Focused race/static gates pass for constructor authority, changed bytes/target/
headers, known4xx versus5xx/lost reply, no repeated write, cache-free absence reads,
actual fixed-curl binary/query/cookie preservation, source/tree/compile evidence,
large integers and private-error output. These are source/inert/local-HTTP checks,
not real Docker transport or durable managed app workflow qualification. Next wire
the trusted store/policy/native observer coordinator and API/CLI/MCP parity. M615
remains the native kernel-only boundary; Grubbery preflight is still unavailable.

## M617 — Coordinate checked native app evidence with immutable runner history (2026-11-18)

A private native Backend now derives expected source structure and original
ownership from the complete namespace journal in durable creation order. Actual
root/marker, immutable source and pinned foundational bytes, source configuration,
followed version and live-mirror bytes are checked independently. Compilation
requires its own exact native report and cannot hide changed served source.
Unknown paths/marks/nexuses, replaced markers, incomplete reads, mismatched
hashes and unconfirmed owner-session cleanup refuse. Reads have explicit512-call,
1MiB-response/4096-node bounds and require the outer deadline; no history is
pruned to make an observation pass. Original observed code-namespace creation
plus fresh exact foundational source proves its generated mar directory without
another create. This narrowly supersedes the earlier exactly-one-call Dispatch
comment: at most one app call, or a digest of that original derived provenance.
It does not adopt an arbitrary existing directory or change old plans/events.

The Linux runner factory binds this observer to authenticated role/actor, exact
runner/mode/target/config/container/pier, original pinned framework commit,
separate observed activation, live framework agents and fresh Commit policy.
The lifecycle/policy mutex remains mandatory across the Execution call. Creating
the adapter does not grant writes; read-only reconcile/abandon can remain usable
after Commit revocation. New effects still recheck current authority. Private
cache loading checks actor, target, workflow/settings and complete source bytes.
A focused fixture exposed that the frontend reader requires index.html: the
shared confined byte reader now serves both paths while frontend validation
retains its index/MIME rules. No mark conversion or live Unix mount is introduced.

The existing stage-discard API now retains a stage used by any unfinished app
operation, including known refusal until supported abandonment. Terminal replay
remains independent of that cache. SQLite history checks the complete journal
in one transaction and never accepts caller ownership projections or substitutes
request-ID sorting for durable creation order.

Focused race gates pass for complete two-release source following, exact original
ownership, generated-directory provenance, wrong native markers/paths/foundations/
cleanup/authority, compiler/version versus mirror bytes, and uncertain native
reply recovery without source or redispatch. Isolated runner/SQLite gates pass
across normal reopen, unavailable source cache and revoked Commit, preserving the
original uncertain chain and single write. Guard drift/refusal, missing activation,
unresolved framework operations, stage retention/discard and creation-order history
are tested. The underlying native runtime is inert in these tests; this is not
real Docker/native coordinator or public CLI/MCP qualification. No persistent
format/schema, live capability, installed artifact, DEV or production change.
Next qualify the managed coordinator against the pinned fresh native kernel and
then expose exact phase operations with shared interface evidence. M615 stays
kernel-only evidence; Grubbery preflight remains unavailable.

## M618 — Prepare a separately supervised native coordinator gate and compact retained evidence (2026-11-18)

The new explicit grubbery-managed-coordinator-v1 capsule bootstraps the same
frozen framework, then uses the private managed factory/SQLite/cache/policy guard
and exact original phase handles against actual typed native owner HTTP calls.
Each call checks the fresh private fixture and closes its own owner session.
It reuses the M615 JSON-state nexus fixture; the refactored v1 source was compared
byte-for-byte with that qualified capsule. No product API/capability, Docker/curl
transport or installed lifecycle claim follows from this new test. Source compile
and twelve launcher tests pass; the new native case has not run at this decision.
Bounded last-read diagnostic bytes stay private and separate from response JSON.
Unknown phases only get original read-only reconciliation, never write retries.

The background supervisor now captures its actual named unit's properties just
after submission, identifying whether it was still live. Later collected-unit
stubs cannot establish the limits that applied during a run. The new smaller
single-candidate case declares1.25GiB scratch based on the completed app03
allocation (about1.1GiB with copied inputs and pier), retaining the same live
allocation monitor and hard refusal. Other Grubbery cases keep1.5GiB. The scoped
48GiB goal cap, one runtime experiment,4GiB/no-swap/one-core native boundary and
all production/existing-ship exclusions remain unchanged.

To make room, three completed app03 input copies were verified against retained
canonical hashes before retirement. Its complete cleanly stopped pier and the
interrupted app02 pier now have sparse lossless archives, verified across every
original file byte and directory metadata before removing expanded copies.
The latter's failed result, unconfirmed shutdown and original uncertain activation
remain unchanged; archiving grants no restore/resume or filesystem ownership.
All original SQLite/WAL/SHM, proofs, logs and source manifests remain. The two
archives verify1450 and1452 files respectively. The ended owned scopes were
checked explicitly; no existing ship, process inventory or native service was
used. Three old task executables were also losslessly compressed with verified
original hashes and manifests, including the original wrong-marker/bootstrap
helpers. No retained canonical Vere/pill input was removed. Detailed private
manifests bind paths, device/inode, before/after hashes, preserved dependencies
and the bounded archival services. Admission is required again before launch;
temporary headroom is not approval for another experiment or a higher cap.

## M619 — Preserve the first coordinator capsule's fixture refusal and correct its closed guard (2026-11-18)

The dca9e8d native coordinator helper failed immediately in nativeDeskGuard:
its separate fixture-identity case list had not been extended along with the
launcher. Namespace, loopback/no-host-access and scope-admission checks still
ran; the guard then refused the literal framework desk for the new named case.
No framework/app operation or app store was reached. The new fake stopped cleanly.
The [original failed result](diagnostics/2026-11-18-grubbery-coordinator-fixture-refusal.json)
is retained, not relabeled as a native/product pass.

The correction adds only the explicit coordinator case to the existing identity
guard. A direct source regression verifies that case and the existing supported
identities, wrong framework case/desk, traversal and changed Vere pin. It does
not bypass any namespace/process/pier/resource/ownership requirement. This
corrected harness needs a new fresh capsule; M618 is preparation only.

The obsolete pre-app fake and verified duplicate inputs were classified by exact
identity, helper/input hash, clean-shutdown result, ended owned scope and the
absence of an app store before retirement. Original failed results, logs,
source/input hashes and a lossless verified helper remain; private manifests
record each filesystem object and retained canonical inputs. No receipt or
existing ship was changed. The failed capsule is explicitly non-resumable.

## M620 — Native empty code namespaces disconfirm the eager-foundation model (2026-11-18)

The e1e4b5d fresh coordinator capsule passes the closed fixture guard and actual
framework create/adopt/commit/activation observations. The managed private factory
then observes the original source marker and release parent through real native
HTTP and persists their proofs. Its third phase receives the code-namespace
creation acknowledgement but after-observation refuses: the actual native tree
is an empty code neck. M617's inert model/projection expected the foundational
marks at namespace creation. That timing assumption is disconfirmed. The
[failed native result](diagnostics/2026-11-18-grubbery-coordinator-empty-code-tree.json)
retains dispatched/acknowledged/uncertain events; no next phase or repeated write
was issued. The fresh ship stops cleanly. This is not a native coordinator pass.

M617's generated-mark-directory model and its related at-most-one-call exception
remain source/inert evidence only, not a qualified native contract. Before
changing production logic, the new explicit source-layout kernel case observes
trees and raw file bytes before/after each fixed source write, version and live
mirror, with separate bounded foundational-source reads and compiler results.
It uses a fresh unrelated diagnostic namespace and the already exercised v2
state workflow, not a repair/resubmission of the uncertain managed operation.
The new case is included in both launcher and closed fixture guard. No capability
or public source effect is enabled by adding this diagnostic gate.

A later native run's admission first refuses36.5MiB over the48GiB cap after the
helper rebuild. Six obsolete complete framework input copies in ended tombstoned
task capsules are verified against every original input hash and the retained
1443-file canonical pin before retirement, freeing73,900,032allocated bytes.
Original SQL/history/proofs/logs and canonical framework/Vere/pill remain. Private
manifests preserve all removed filesystem identities and dependency hashes.
Original failed results are never rewritten to make qualification pass.

M620 retention follow-through: select the corrected app03 pass and current
managed02 observation failure alongside the Gall07 reference. Retire the now
obsolete wrong-marker app01 and caller-interruption app02 whole-pier archives,
while retaining their original failed results, SQL/WAL/history, logs/proofs/input
hashes and exact helper bytes. Their original archive verification remains a
historical fact; replacement tombstones explicitly link retirement and preserve
the previous tombstone hash/body in the private manifest. No canonical source,
Vere, pill or running input depends on these archives. Current managed02's full
pier is then losslessly archived and verified across1450 files before retiring
its expanded copy; every original app receipt and native diagnostic body remains.
This is compact forensic evidence, not authority to recreate filesystem identity
or resume an archived ship. The next runtime still needs fresh admission.

## M621 — Native foundational sources and materialized trees are separate views (2026-11-18)

The [b4b01c6 fresh layout contract](diagnostics/2026-11-18-grubbery-native-source-layout.json)
passes74 bounded calls and clean shutdown. Immediately after code-namespace
creation the materialized tree is empty, while all four foundational sources are
already directly readable with the exact pinned framework byte counts/hashes.
They remain absent from the materialized tree after bill/nexus/version writes
and in both live code mirrors, while their compiler results are present/clean.
Submitted app Hoon/mime bytes read back exactly. Version following, saved-state
poke/read and preservation on the second source update still pass.

This refines M620: the eager materialized-tree assumption was wrong, rather than
establishing that foundation generation happens later. Keep native source-tree,
virtual derived-source and compiler-cache evidence independent. Correct the
projection/model to verify pinned foundational bytes through separate bound reads;
never manufacture their materialized directory or use matching bytes to adopt an
unowned namespace. Custom-mark directory semantics need their own exact proof
before claiming those phases. M617's inert generated-directory result remains
unqualified. This is a kernel contract, not a managed coordinator, Docker/curl,
public interface, permit/route/restart/peer/installed pass; capability stays closed.

M621 implementation follow-through: the code-namespace projection no longer
invents a materialized mar directory/files. Separate constructor-bound direct
reads verify every virtual foundational grub against the pinned original bytes
on release and live-mirror checks. The source-tree helper and inert model now
represent that same separation; the compiler report still requires all four
foundational artifacts. The unconditional generated-directory skip is removed:
one native call per selected phase, no implicit directory adoption. A new race
regression introduces a materialized directory containing all matching pinned
bytes and confirms that it cannot acquire ownership through those hashes.
Custom-mark directory semantics remain unqualified. The next fresh managed gate
also has a separate fixed kernel diagnostic after the minimal coordinator loop
to observe an explicit custom-mark create, raw source and compile, without
repairing the earlier pending operation or accepting caller proof. No schema,
existing header/event or live capability changes.

## M622 — Make original recording bytes an explicit JSON encoding choice (2026-11-18)

Unwired Frame/Page metadata JSON now omits raw data rather than implicitly
base64-encoding it. A separate bounded ExportOriginalFrame encoding preserves
original octets and frame timing/kind/digest metadata in an independent copy.
The binary format, retained bytes and decoder remain unchanged. Source/race
checks pass for omission of raw/fake-secret bytes, exact explicit export including
controls/NUL/invalid UTF-8, no mutation alias and refusal of invalid/oversize frames.
This is not redaction, authorization, whole-session export or verified capture;
public callers must independently select verified retained data and enforce
current target/operator/explicit raw-export grants. No runtime/API/UI is enabled.

Read-only integration review also identifies the v1 metadata's missing full
project generation/creator, native ship/pier and engine/namespace binding. A
successor needs original-reader preservation/refusal before live capture. Current
pages rescan prefixes; any index must state its actual verified coverage and not
claim a skipped old prefix was reread. Storage reserve refusal cannot guarantee
that a subsequent gap itself persists. Coverage state, retention, safe pressure
recovery and independent exit/drain remain required. Docker CLI29.7.2's pinned
upstream run flow attaches before start and drains output after exit; this is a
primary-source implementation guide, not qualified Omabit recording behavior.
Private notes retain those follow-ups and upstream links. The active native app
helper remains the separately frozen ac3eec2 artifact; this later recorder-only
source change is outside its qualification scope.

M621 retention follow-through keeps the corrected app03 and Gall07 full references.
The completed source-layout01 full diagnostic pier/duplicate inputs are now
obsolete and retired with full file identities/hashes, preserving original
qualified proof/SQL/history/logs and exact helper. The explained managed02
whole-pier forensic archive is also retired; original acknowledged/uncertain SQL,
native response bodies, proofs/logs/input hashes and exact helper remain intact.
Replacement tombstones link superseded archive claims explicitly. No canonical
source/Vere/pill, current experiment, user ship or production service was changed.

## M623 — Corrected private managed native source phases pass (2026-11-18)

The [frozen ac3eec2 coordinator gate](diagnostics/2026-11-18-grubbery-managed-native-source.json)
passes all12 original phases, each dispatched once and observed, with real private
SQLite/cache/policy/ownership/byte/compile/version/live-mirror checks against the
pinned kernel. The fresh ship stops cleanly. Virtual foundational bytes remain
separate from materialized source. No earlier uncertain header or event is
repaired, rewritten or relabeled; those failed capsules remain failed.

A separate fixed kernel namespace confirms the custom-mark question: its mar
fold is absent before explicit creation, the single create returns201, then the
submitted exact custom source is the only materialized mark while four virtual
pinned sources remain compiler inputs. All five compile cleanly. This confirms
removing the unconditional generated-directory skip; it does not by itself
qualify a managed custom-mark source operation. Native per-call owner cleanup
must finish before evidence is accepted. Process-death cleanup remains unqualified.

This gate invokes the private managed coordinator through a native owner HTTP
fixture with synthetic lifecycle and exact typed sources; it does not exercise
product Docker/curl, public CLI/MCP, managed second-version/state/restart, permits,
aliases/routes/assets, peers or installed upgrades. The later recorder-only M622
commit is outside the frozen artifact. Grubbery capability remains unavailable.
Next expose saved app receipts and exact phase interfaces with parity, then
qualify their actual Docker transport and the remaining workflow boundaries.

## M624 — Saved app receipts through shared public interfaces (2026-11-18)

`app.source-operation.v1` now reads the original Grubbery record through the
runner API, pinned client/registry, generic CLI and official-SDK stdio adapter.
Its distinct receipt feature grants no live adapter readiness. Current operator
access suffices for saved inspection; original creator attribution and all
uncertainty remain unchanged. Reads validate the complete SQLite journal in one
snapshot, outside the lifecycle mutex, without source cache or runtime access.
Restricted SSH already allows operator requests and excludes this path for
observers; no broader observer authority was added.

Focused race gates pass for actual CLI/stdio byte parity, startup allowlists,
observer/missing-feature refusals before receipt reads, changed response identity
and corrupt history. An isolated actual SQLite/HTTP gate preserves original
uncertainty and returns while the lifecycle mutex is held, with no runtime or
cache. Catalog/access regression checks pass. This is source/inert interface
evidence, not installed SSH/native app dispatch or final-artifact qualification.
The48GiB admission counted51,095,093,248 bytes before these focused builds; no
runtime experiment or deployment was started.

## M625 — Exact original app phases through typed source APIs (2026-11-18)

Review derives its actor, pinned framework receipt/observed baseline, pier and
policy at the runner; staged bytes remain private and access-bound. Begin writes
the unchanged plan/namespace lease without an app effect. Advance selects exactly
one original phase, with durable dispatch before I/O. Saved replay precedes live
admission. Reconcile never dispatches or loads source; safe abandonment preserves
live/source state and its original journal. Current actor comes from transport,
never from the body. Original actor/process/config remain required; observation
across same-pier restart and grant rotation is still unqualified.

Actual isolated SQLite/HTTP race tests pass for review without journal/native
writes, lost reply and single dispatch, skipped-phase refusal, original digest
and actor binding, missing cache/revoked Commit recovery, original uncertainty
preservation, completed-phase replay and pristine abandonment/replay. Actual CLI
and official-SDK stdio parity pass for all five typed operations; explicit MCP
mutation enablement also gates journal-only calls. Restricted SSH tests pass for
20KiB plans,96KiB overflow refusal, same-connection missing-feature refusal,
observer denial and unchanged ordinary4KiB limits. Catalog/access regressions
and affected Go vet pass. No persistent format changes were made.

`grubbery-app-source-v1` denotes these APIs separately from full workflow readiness;
preflight remains unavailable. These are source/inert gates, not a new native
product or installed qualification. Product Docker/curl, managed state/update/
restart, durable owner cleanup, permits/routes/assets/peers and final-artifact
upgrades/DEV remain outstanding. Frozen M623 remains scoped to ac3eec2.

Storage reached51,464,445,952 charged bytes during focused builds, below48GiB.
Two completed source-only test helpers were losslessly archived with SHA/inode
manifests. An initial atime-inclusive comparison refused before unlink; the
corrected immutable identity comparison verifies both archive inflations.
The known managed03 runtime and supervisor are inactive. Only its exact duplicate
pill/Vere/host helper were retired after canonical hashes were reverified,
freeing270,315,520 allocated bytes; original SQL/pier/proofs/logs remain unchanged.
No user ship, installed service, production plugin or manual-review state changed.

## M626 — Prepare public native update/restart gate within the budget (2026-11-18)

A new explicitly guarded `grubbery-public-workflow-v1` case will exercise the
actual public source API and pinned registry/client for two source versions,
independent fixed native state/code probes, a supervisor-bound clean same-pier
restart, another state write, original SQLite reopen and completed-phase replay.
This is prepared code, not a passed native result. It still uses synthetic
lifecycle and direct owner HTTP, so it cannot qualify product Docker/curl,
recovery of a pending operation on a changed runtime, permits/routes/assets,
peers, installed upgrades or the native CLI combination.

Only this longer case receives a20-minute Go context,21-minute test bound,
1,280-second inner workflow bound,1,600-second outer deadline and1,750-second
independent supervisor cap. It retains one active experiment,4GiB/no-swap/one-CPU
capsule,512MiB/no-swap/25%-CPU supervisor, namespace guards,1.25GiB scratch
admission and live48GiB aggregate enforcement. Existing case deadlines remain
unchanged. The closed Go identity guard and12 launcher tests pass; affected
managed Go vet passes. Build a matching clean helper before admission.

The clean completed managed03 pier is now sparsely archived with all1,450 file
contents verified on streaming decompression: archive SHA256
23fdd4b5b8ea3aebef97c2135e91f5be3f0c0bd736b5a7a14a6b347ee157afc2,
258,569,202 bytes. Retiring its verified expanded files frees520,200,192 allocated
bytes. Original SQL/proofs/logs/helper remain unchanged. The obsolete first Gall
capsule's full pier is separately retired with all file hashes and a tombstone;
its original uncertain activation result/SQL/proofs/source/helper remain failed
and unchanged. The selected successful Gall07 full archive is reverified and
retained. This frees589,733,888 bytes and makes no historical-cause claim.

Three further ended-capsule framework-source duplicates are retired only after
all1,443 files match the exact canonical pin, which remains retained and is
reverified afterward. Their original input/result/proof bytes remain unchanged;
36,950,016 allocated bytes are freed. Completed retirement manifests identify
absent expanded piers; a preliminary tombstone-name assumption refused before
any unlink. No production, installed service, user pier or manual-review state
was selected for these retirements. Final runtime admission is still required.

## M627 — Public native source update/state/restart gate passes (2026-11-18)

The [frozen d2d8cf9 gate](diagnostics/2026-11-18-grubbery-public-native-workflow.json)
passes in1,003.86 seconds. It uses the actual runner API and pinned registry/client
for two exact source plans:12 then9 phases, each dispatched once, acknowledged
and independently observed. Their original final revisions are36 and27. Code
and followed source are checked separately; saved app state survives the source
update and the new code label is observed. The supervisor verifies the exact
owned process/pier before clean stop and same-pier restart. State remains, another
write succeeds, and both original records survive SQLite reopen and old-phase
replay unchanged. The fresh ship stops cleanly at completion; both scope and
independent supervisor are inactive with successful results.

This supersedes M626's prepared-only status for that exact artifact. It still
uses synthetic lifecycle/direct owner HTTP and fixed test state probes, so it
cannot qualify product Docker/curl, pending operation/process rebinding, owner
cleanup on process death, native CLI/stdio integration, permits/routes/assets,
peers, installed upgrade or DEV artifacts. Full Grubbery preflight remains
unavailable. Recorder-only source edits made during this frozen run are outside
its artifact and qualification; they were not compiled during the experiment.

Only the ended capsule's exact duplicate pill/Vere/host helper are now retired
with hashes/inodes and reverified canonical copies, freeing270,422,016 allocated
bytes. Its full pier, source, original SQL/proofs/logs/helper and restart history
remain unchanged. Before subsequent focused recorder builds, charged storage is
50,932,744,192 bytes and128MiB scratch fits under the temporary48GiB cap. No
production, installed service, user pier or manual-review state was changed.

## M628 — Draft capture identity becomes a source-tested format fence (2026-11-18)

Recording metadata2 binds the full target generation/creator, native ship, pier
continuity, daemon identity/endpoint digest, host boot and external recorder's
namespaces; projects additionally bind the verified anchor namespace. These are
attachment identity fields, not evidence that Start occurred or a container
process namespace exists. Future runtime admission must obtain them itself.
Metadata2 shape validation is not runtime observation or caller authorization.

Version1 remains canonical/byte-preserving and never acquires invented identity.
New Store.Create with legacy metadata refuses before creating files. Unknown
versions refuse as invalid. CaptureIdentity also refuses to infer a current
generation from legacy instance/project names. A new writer copies nested identity
values before admission so callers cannot mutate its captured binding. The
binary frame codec remains1 with exact metadata digest binding. There is no
runner database version change, runtime integration or installed mutation.

Full focused recording race checks and Go vet pass, including identity/header
binding, complete project/ordinary distinctions, caller-pointer isolation,
legacy/unknown refusal without filesystem changes and the existing raw codec,
paging, privacy, torn-tail and pressure checks. A separate source rehearsal is
prepared with an actual3218e6f writer/reader and a declared helper: original
arbitrary bytes/gap/exit through new cursor/read/reopen, read-only legacy refusal,
and genuine old-reader refusal of successor bytes unchanged. Its actual gate
is pending at this freeze; unit reconstruction is not original-writer evidence.
Runtime, indexed seeking/rotation/retention, durable coverage/pressure, derived
redaction, API/UI and native/installed capture qualification remain outstanding.

## M629 — Preserve first recording rehearsal and correct helper expectation (2026-11-18)

The first actual original3218e6f rehearsal passes original-byte, cursor and
repeated reopen preservation through the candidate58d0586 reader, including
read-only legacy capture refusal. Its last helper assertion fails: unchanged
Store.ReadPage classifies invalid/unsupported metadata as ErrCorrupt, while the
helper expected the metadata decoder's narrower ErrInvalid. The original store
therefore did refuse metadata2; this is not evidence it accepted it. Inspection
of the exact frozen original source confirms that mapping. The failed command,
original binaries/source/input manifest and candidate report remain retained;
both original and successor file hashes are reverified unchanged after failure.

Correct only the declared helper: independently require ReadMetadata ErrInvalid
and Store.ReadPage ErrCorrupt. Original runtime source remains unchanged. Rerun
in a fresh qualified directory after this freeze; do not relabel the first
failed result or claim the overall gate passed yet. No engine/runtime/installed
service or ship was involved.

## M630 — Actual recording1 writer/reader compatibility passes (2026-11-18)

The [fresh corrected rehearsal](diagnostics/2026-11-18-recording-metadata2-source-upgrade.json)
passes against candidate5041406. The unchanged original3218e6f runtime sources,
original metadata_test fixture and declared helper write real private recording1
files with arbitrary stdout/stderr bytes, gap and independently typed exit.
Repeated candidate reads, exact cursor paging and store reopen preserve every
original file byte and leave absent identity unknown. Legacy capture refuses
without writes. The actual original metadata decoder/store reject metadata2 and
the complete successor file hashes remain unchanged. The failed first directory
is retained as failed; no baseline or result was relabeled.

Original writer SHA256 is0b27ba1be956f24e6ebcfb0de417ede59b84c4b48cd8534f96357f91454ef843;
candidate helper SHA256 is9985e8dd71892518711f863c4fab8f32f6ee09bd24bd5ae59df19031aee70031.
A compact original writer fixture with exact source/helper provenance now runs
in ordinary source tests, including immutable raw octets, legacy identity refusal,
new-capture refusal, cursor resume/reopen and unchanged originals. Full focused
recording race checks pass after retaining it. This qualifies only the frozen
source/filesystem format boundary. No live capture, rotation/index/retention,
durable pressure/coverage, redaction, API/UI, native/installed or stable behavior
is implied, and no runner database or installed service changed.

## M631 — Maintained source samples match the public native fixture (2026-11-18)

`app.sample.v1` now returns named Grubbery state source versions1 and2, exactly
matching the two bill/nexus manifests from M627. Their version2 definitions carry
relative sample contract metadata and separately pinned framework/runtime inputs.
Gall definitions and their exact hashes remain unchanged. Evidence binds exact
qualified source while explicitly declaring that named packaging was added after
the frozen d2d8cf9 artifact. It does not fabricate a prior named-sample test.

Source digests are840fb9fa309325af2ec44087342999dd95595ceacebd04335ba250103e5859b7
and5be1882ece1a6c07f58dce29bace4df9bbf3bd2ab9feae164a3ddc3c874a3a3b.
Source/staging/adapter validation, exact equality with the maintained native Hoon
fixture, Gall regression and actual CLI/stdio sample parity pass under the race
detector. The developer skill distinguishes offline source availability, typed
phase APIs, narrow native evidence and full preflight, which remains unavailable.
Only bill/nexus files belong in staging; definitions/evidence/notices remain out.
No public state read/poke/watch API, product Docker qualification, permits/routes/
assets/peers or installed upgrade is implied. These samples support preparing the
next fresh product-transport VM without copying test-only source generators.

## M632 — Retain exact old VM evidence and retire superseded full copies (2026-11-18)

A [bounded read-only retention utility](diagnostics/2026-11-18-issues-evidence-retention.json)
boots only a fresh kernel/init, mounts the old task VM03 root read-only,noload,
noexec, and copies fixed original SQLite/result/command-log paths. It never runs
source-disk services/code or opens pier paths. Source/base hashes remain unchanged.
The closed original schema60 store has47 tables/10 rows and only the two recorded
disposable fake identities. All original SQLite/SQL/guest commands/result bytes
are retained privately, along with source/helper hashes, code and compressed
utility artifact. The first utility command-line guard refused before mounting
because it assumed a trailing space rather than the kernel's newline; its failed
result/helper and unchanged source hashes remain retained. Field-based matching
corrects only that utility guard for the fresh successful run.

After verified extraction and inactive source/utility scopes, the superseded VM03
whole disk is retired, freeing875,831,296 allocated bytes. Its passed installed
upgrade section and later overall duplicate-enrollment failure remain unchanged.
Both retained VM01/VM04 roots directly depend on the reverified canonical base,
not this retired disk. Tombstones prohibit resuming historical launch commands.
This supersedes older full-VM03 retention claims only; it adds no product gate.

The current completed public01 native pier is losslessly sparsely archived and
all1,450 file contents match on streaming decompression:259,190,089 bytes,
SHA25648b0773c6c251cfe40094618428eb435da0b7678ad7dce5d62b30972a3ea28ff.
Its verified expansion retirement frees495,067,136 bytes, with original source,
SQL, proofs, exact helper and both process/restart logs unchanged. This stronger
full public source/state/update/restart reference supersedes the old app03 kernel
and managed03 private-only full archives; retiring those exact verified archives
frees524,328,960 bytes. Their original SQL/receipts/proofs/logs/exact helpers remain
unchanged. New retirement manifests retain original tombstone bodies and link
replacement status; historical archive hashes are not claims of present files.

Only classified task-created artifacts were selected. The canonical base/runtime/
pill/framework sources, selected Gall07 and current public01 full references,
original writer fixtures, VM04 recovery reference, production, installed services,
existing/manual-review ships and all excluded paths remain unchanged. No runtime
experiment remains active. Recheck2GiB VM scratch after preparing/building matching
product transport artifacts;48GiB remains the hard goal ceiling.

## M633 — Prepare an actual product Docker/CLI/stdio capsule (2026-11-18)

A separate fresh VM harness now targets the actual installed runner and product
rootless Docker/curl source transport, with no backend injection. The helper uses
SDK staging/framework setup plus actual CLI/official-SDK stdio source operations,
exact saved receipt parity, two maintained source versions, fixed independent
state/code probes, supported Stop/Start with changed container ID, manager restart
and completed-phase replay. It saves complete reviewed plans before submission.
Only read-only state observations may repeat while the nexus loads; state writes
and uncertain source effects are never retried. Fixed state probes use the existing
operator code endpoint and exact enabled private local route, keeping secrets
out of proof and process arguments. No new public state API is introduced.

Four no-effect host/guest guard tests pass; the Go helper compiles/skips outside
its explicit scenario/user/QEMU guard, and managed Go vet passes. This is prepared,
not native product qualification. It requires clean matching collector/runner/CLI
and source-pinned helper before launch. The fresh standalone base, complete frozen
framework and runtime/pill must be reverified. Historical bootstrap Kind remains
"recommended" for an explicit uploaded pill; this case selects the standard
f8b08c pill rather than the Apps pill, matching the prior native source gate.

One separate bounded VM uses8GiB guest/10GiB scope memory, no swap/one CPU, no NIC,
no host HOME/run/runner/engine/keyring mounts, and a2GiB storage reservation under
the temporary48GiB goal cap. Admission requires12GiB available memory; allocation,
log and deadline monitors remain active. The independent host supervisor has
512MiB/no-swap/25%-CPU/2700seconds; the guest helper has a30-minute context and
35-minute test bound, with2400-second guest/2500-second host bounds. Only its own
fresh scope is signalled on interruption. No production/DEV service is targeted.
Product transport, workflow failures, pending-phase rebinding, durable auth cleanup,
permits/alias/routes/assets/peers and matching installed upgrade remain pending.

## M634 — Freeze matching product artifacts and separate harness provenance (2026-11-18)

The three product executables and source-pinned helper are built cleanly from
e61d971 with all Go/embedded input hashes checked before/after. This is an
unqualified development snapshot for the fresh product VM, not installed DEV or
stable distribution. The helper source pin and executable VCS identities bind
that exact commit. No prior qualified or installed binary was overwritten.

The launcher now accepts a separate exact ancestor artifact source only when
all Go/go.mod/go.sum/embedded inputs remain unchanged. This permits narrow harness
or documentation fixes without relabeling artifact provenance. Both identities
remain explicit in receipts/scenarios. Host supervisor/init builds explicitly
use cached offline Go dependencies and disable Go telemetry. Four no-effect
guard tests pass after that harness-only change. No native product result is
claimed at this freeze; admission/launch remain the next steps.

## M635 — Bound product input staging after observed supervisor OOM (2026-11-18)

The first product attempt never boots QEMU or creates a ship. Its512MiB host
supervisor reaches exactly that cgroup memory peak while mkfs populates the
immutable input filesystem; the unit journal records oom-kill/status9. Automatic
unit collection makes later default unit properties unsuitable as outcome proof;
the original journal, partial scenario, artifact pins and failed result are
retained. No native/product gate is relabeled. No installed service was targeted.

Correct only the harness: allow a bounded2GiB/no-swap supervisor for filesystem
staging page cache, retain25%-CPU/2700seconds, and require14GiB available memory
before the separate8GiB guest/10GiB scope. Current admission observes45,426,835,456
available bytes. Preparation receipts now distinguish inert staging/payload-built
from guest creation. The48GiB aggregate and2GiB additional storage ceilings are
unchanged; no Go or embedded input changed from frozen e61d971 artifacts.

After the stopped unit is verified, every partial staging file matches its exact
canonical artifact/framework/runtime source or preserved generated scenario.
Only those copies and the incomplete input filesystem are retired, with hashes,
inodes and original failure evidence preserved. This frees519,442,432 allocated
bytes. There was no VM image, pier or guest service to clean up. Retry must use a
fresh output directory with explicit artifact/harness identities; the first
attempt remains failed. No production, DEV manual-review state, protected ship,
tmux session or excluded path was inspected or changed.

## M636 — Preserve product policy-setup failure and correct fixture scope (2026-11-18)

Product VM02 boots the actual installed source-e61d971 runner/schema62 and starts
one fresh fake. The Go helper refuses before desk creation: it incorrectly asked
the desk-level policy update to grant CreateDesk. The existing contract requires
that gate at ship scope and inheritance at desk scope, as the earlier native
fixture already did. This is a fixture mistake, not a product permission change.
Correct the helper's exact selected ship update, then inherit CreateDesk while
allowing Commit/blocking Publish at the selected Grubbery desk. A pure race test
validates both scoped updates and their gates; managed Go vet/guard tests pass.

The failed original image/result/artifact pins are preserved through read-only
retention extraction before retirement. Original SQLite/WAL/SHM, full guest
commands/result and host serial remain unchanged. Queries/dumps use separate
analysis copies because WAL is present; the snapshot has51 tables/schema62 and
only the recorded single disposable fake. No immutable=1 shortcut ignores WAL.
The original full root/input filesystem are now superseded and retired with
hashes/inodes/manifest, freeing2,024,460,288 allocated bytes. The failed result is
not relabeled, and exact e61d971 product artifacts remain retained.

The untrimmed new overlay retained1,699,876,864 allocated bytes even at this early
stage. Enable discard only on future new VM drives, and let the explicitly guarded
guest trim only its own writable root before/after bootstrap and periodically
during its helper. This reclaims deleted bootstrap/checkpoint blocks, without
raising the2GiB scratch or48GiB aggregate ceilings or touching host/reference
filesystems. The earlier staging2GiB-memory supervisor/no-swap limits remain.
After this Go helper change build a fresh matching artifact set; no production
code, saved permission, installed service, user pier or manual-review state changed.

M636 retention also losslessly archives six older ended native helpers and one
completed superseded source test binary, verifying each full inflation/hash and
file identity before retiring its raw duplicate. Exact artifact hashes and sidecars
remain;147,460,096 allocated bytes are freed. Current Gall07/public01 full-reference
helpers, canonical runtime/pill/framework, frozen Stop/product artifacts and all
original SQL/receipts/proofs/logs remain unchanged. Archive sidecars supersede old
raw-path retention claims without rewriting historical input manifests.

## M637 — Wait for an observed container before product desk selection (2026-11-18)

The source60335dd product VM03 completes fresh installed bootstrap/Start, then
refuses before desk creation with current runtime selection unavailable. Its
original command snapshots show running/no active operation but unavailable
resource telemetry; Start completion itself does not supply the container ID
used by desk selection. The preserved later original database has the actual
fresh resource/container observation. This is a fixture readiness race, not
evidence that a native app request failed or that policy should be relaxed.

The helper now polls only bounded snapshot reads for one exact ready ordinary
instance, valid revision and full observed container ID before the first desk
plan and after its supported Stop/Start. It never resubmits a mutation or infers
an ID from an old runtime. Focused race tests cover unavailable telemetry,
recovery/active-operation fences, different project, missing revision and duplicate
targets; managed vet passes. A fresh matching helper/artifact freeze is required.

After all owned units end, a separate no-network read-only utility exports the
original failed database and complete command/result logs without executing any
source-disk code or services. SQL analysis uses copies; original hashes remain
unchanged. Only the explained failed root and copied input filesystem are
retired, freeing1,504,108,544 allocated bytes. The original failed outcome,
60335dd artifact set and full evidence remain retained; no full old pier archive
is claimed. Production, DEV manual review and existing ships remain untouched.

## M638 — Preserve originals and account for derived redacted spans (2026-11-18)

Issue20's unwired source library now has bounded explicit redacted windows over
already verified frames. Narrow credential-field values and owner-selected
literal secrets are omitted with reason/offset/length; broad0w atom removal and
whole-line keyword dropping are not inherited. Matching crosses frame boundaries
per stream without changing original timing/order or raw bytes. Unknown leading
and all trailing partial lines are withheld, and gaps invalidate context. Text
escapes terminal/format controls and invalid UTF-8 for plain-text rendering.
Secret inputs never enter metadata/results. Repeated/overlapping literals use
linear matching; bounded spans refuse without partial output.

All recording race tests and focused vet pass, including every credential split,
interleaved streams, long traces, context gaps, exact byte accounting and original
fixture preservation. This is derived encoding, not authorization, digest-chain
verification, complete export, automatic publication or a guarantee that every
private application value is identified. Capture/integration/API/UI and native
qualification remain pending; the persistent recording format is unchanged.

Storage retention losslessly archives ended frozen60335dd product artifacts,
original60 writer, selected Gall07/public01 and older app helpers, plus frozen
d24c960 tools and failed product02/03 boot helpers. All full inflations match
their original SHA256/size before raw duplicates are retired. Sidecars supersede
historical raw-path claims; sources, original fixtures/SQL/receipts/logs and full
selected runtime references remain unchanged. These two batches free186,970,112
allocated bytes. Explicit inflation and fresh storage admission are required
before reusing an archived executable. No production or manual-review data is
selected or changed.

## M639 — Nonblocking, single-observer desk reconciliation admission (2026-11-18)

The source55331af product VM04 passes exact readiness, real Docker/curl framework
creation, original-scaffold adoption and source commit. Activation retains
planned→dispatched→uncertain with one native install submission. A later
reconciliation poll exceeds the ordinary5-second client timeout while waiting
behind a long observer holding lifecycle admission. The original failed result,
full phase plans, SQLite/commands and serial remain unchanged. This is not a
proved activation failure or permission to install again; app-source phases
never start. See the [scoped failed-case evidence](diagnostics/2026-11-18-grubbery-product-initial-failures.json).

Desk reconciliation now authenticates/validates and reads saved evidence outside
the long admission mutex. `busy` returns that unchanged receipt without queuing
more work; `queued` admits one bounded read-only observer. Pending workers also
return busy, closing the pre-worker scheduling race. Current receipt state is
reloaded under admission; completed replay remains read-only. Failed-store,
closing/update fences and the workers' current actor/ownership/configuration/pier
checks remain. No native effect is retried, timer enlarged or receipt rewritten.
The optional response field is shared by typed clients/catalog/CLI/MCP and is
distinct from original journal state. Unknown admission values refuse. No database
format changes; matching final runner/collector artifacts remain required.

All managed desk source race regressions pass, including creation, commit,
activation/publication, receiver/frontend and their recovery/refusal paths.
Actual SQLite/HTTP held-admission and pending-worker checks retain the original
receipt;30 polling calls admit only one observer and preserve one original native
effect. Actual CLI and official-SDK stdio return byte-identical busy/queued
receipts and reject unknown admission values. Focused client/collection/catalog
checks and vet pass. A new matching-artifact product VM is still required.

After all owned units end, a no-network read-only utility retains VM04's original
SQLite, phase proof and full commands/result logs without executing disk code.
Only the explained root/copied input filesystem is retired, freeing1,667,629,056
allocated bytes. Uncertainty remains unchanged; no full old pier archive is
claimed. A separate lossless archive preserves exact55331af executables and the
older superseded aa71757 failed-Start full image, freeing198,737,920 bytes after
full inflation verification. Sources/base dependencies/original logs remain.

## M640 — Retain exact evidence and reduce the fresh product reservation (2026-11-18)

Freeze new matching development artifacts at5eeae9a after M639's source/race,
actual CLI/stdio parity and vet gates. No installed namespace is updated. Only
harness/documentation follows that freeze; artifact and harness commits remain
separate explicit bindings. Full native app qualification is still pending.

Four ended read-only utilities retain exact init gzip/source, canonical module
bundle and original SQL/proofs/logs. Their generated boot archives and copied
modules are verified member-by-member against those retained bytes before
retirement, freeing54,722,560 allocated bytes. An initial guard uses the wrong
historical manifest filename and refuses before any deletion; the corrected
selection requires the exact existing closed-store manifest. No missing evidence
check is waived. A separate full lossless archive of the ended qualified D24 Stop
image verifies every inflated byte and retains its shared base; only4,337,664
bytes are saved. Original hashes and qualification limits remain authoritative.
Read-only classification finds no shared Go-cache file byte-identical to the
selected archived executables, so no Go-cache files are removed.

The product launcher reduces its declared additional reservation from2GiB to
1920MiB. The explained ended framework case retains1,667,629,056 root/payload
bytes plus a small boot archive; this leaves bounded space for the two small
source versions/state and container replacement. Their final allocation remains
unqualified. Existing five-second scratch/48GiB aggregate refusal and interruption
checks remain, as do one-runtime, memory/no-swap/CPU, guest-only trim and no-network
boundaries. An over-limit new case must remain failed and preserved, never become
permission to exceed the cap. No production, user ships or manual review changes.

## M641 — Preserve acknowledged app uncertainty and scope read authentication (2026-11-18)

The frozen5eeae9a product case with harnessd22733e passes real Docker/curl
framework create/adopt/commit and resolves activation by read-only reconciliation
without another install. Initial app phases0–4 are observed; phase5 is dispatched
and acknowledged, with later phases pending and its exact lease held. The helper
fails after about241seconds in that call; timing fits the four-minute bound, but
the old helper discards its CLI error envelope, so the precise client error is
not proven. The original17-event journal, phase plans, commands/serial and full
VM image are retained unchanged. No unknown phase is redispatched or abandoned.
The scoped [partial product evidence](diagnostics/2026-11-18-grubbery-product-partial-source.json)
does not claim source activation, state, second-version, restart or full CLI/MCP
qualification. The earlier full Stop reference also remains retained.

Each verification read previously created a new owner session. An optional trusted
read scope now authenticates once per bounded read-only capture. Original
sequential/conditional reads, request bounds, exact bindings/process checks,
framework/pier capture brackets, current runner guards and native write dispatch
remain. Provisional scope-active responses cannot produce a proof until logout
is confirmed. Callback/cleanup failure, invalid callback or drift refuse; a
retained reader or independent background context cannot extend the scope.
Ordinary runtimes keep their existing individual-session path; native writes
keep separately authenticated exact dispatch. No persistent format changes.

Native-model race tests produce identical complete plans/read counts/source and
single-write counts in both modes. Cleanup/invalid-callback and conditional-root
refusals remain closed. Inert runtime/HTTP tests prove one code/login/logout,
exact cookies, callback/cleanup failures, foreign actor, parent cancellation,
process/framework drift and retained-reader refusal. The first transport fixture
incorrectly returned plain JSON where Lens requires a hex atom; it fails before
authentication and is corrected to the actual wire format. All focused native,
managed app and HTTP regressions plus vet pass. Native speed/full workflow and
pending-phase container/key rebinding remain unqualified. The next guarded
fixture preserves private request/response JSON and bounded CLI stderr before
reporting a typed failure; no original outcome is relabeled.

## M642 — Prepare read-only private Docker attachment without lifecycle effects (2026-11-18)

Issue20's unexported transport primitive pins one full container ID and an
owner-private Unix socket, checks its unchanged filesystem object and actual
peer UID, and requests only streaming output with no replay/stdin. The pinned
API1.47 path requires explicit101/TCP/non-TTY multiplex acknowledgement; raw TTY,
other upgrade, redirect, oversized headers and invalid layout refuse. Headers
and dialing have bounded timeouts while the stream belongs to its independent
recorder lifetime. Cancellation/close interrupts blocked I/O by closing only
the owned connection. There is no writable console or runtime signal/action.

Inert Unix-peer race tests preserve a350-line burst,12288-byte trace, diagnostic
atoms/fake credentials, controls/NUL/invalid UTF-8 and buffered initial output;
decoder stream identity remains exact. Close/cancel send one attach request only,
with no stdin or lifecycle request. Invalid ID, public/symlink socket, raw TTY,
header/protocol and redirect checks refuse; runtime/HTTP regressions and vet pass.
Primary Moby API source confirms the framing distinction. No real Docker/Vere
capture, launch handshake, process status, independent service, persistent-store
integration, new metadata authority, API/UI or installed behavior is qualified.
The future trusted factory must separately verify exact engine/container/launch
and namespace ownership, complete metadata and durable pre-Start admission.
The existing provisional backend choice remains provisional.

## M643 — Share bounded operation deadlines and preserve accepted app work (2026-11-18)

MCP's blanket90-second timeout could expire before existing four-minute app/desk
RPCs and ten-minute source uploads. Trusted typed catalog descriptors now declare
their bounded timeout; both generic CLI and MCP enforce that same deadline with
15seconds for connection identity/capability handshakes. Earlier caller deadlines
remain earlier, and tool inputs cannot override the bound. Mutation cancellation
still means uncertain outcome, never safe redispatch. Native RPC bounds remain.

Accepted app journal/phase requests use the runner lifetime and existing
four-minute bound rather than the disconnected caller's context. Review remains
caller-cancellable; runner shutdown still cancels phase work. Exact journal IDs,
dispatch/acknowledgement, serialization and saved lookup remain unchanged. No
schema, queue, new native effect or installed artifact is introduced.

Focused catalog/deadline race checks and actual CLI/official-SDK stdio app-source
and reconciliation parity pass. Actual SQLite/HTTP tests prove one observed
write after caller disconnect and no native write for cancelled review or runner
shutdown; existing source recovery checks and vet pass. Initial test-only waits
were too short for race-enabled SQLite setup and are increased to15seconds;
no production timer is enlarged. Native performance and complete app workflow,
pending-operation rebinding and matching installed artifacts remain unqualified.

## M644 — Explicitly retire the old full Stop reference, preserve current uncertainty (2026-11-18)

Supersedes only M640/M641's statement that the older qualified Stop VM's full
image remains retained. Automatic approval review refused irreversible deletion
of that sole full image; no deletion ran at rejection. The user then explicitly
authorized removing that exact archive while retaining its SQLite, receipts,
logs, source and binaries. The verified removal frees1,000,701,952 allocated
bytes. A private retirement manifest/tombstone retains original image/archive
hashes, preserved evidence hashes and dependency checks. No full old Stop pier
or image is now claimed; its original scoped qualification results remain valid.

The newer app VM with an acknowledged pending phase remains intact and unchanged.
Original Stop SQLite and exact losslessly archived binaries remain available.
Shared build-cache growth had taken the measured aggregate above the temporary
48GiB cap by69,615,616bytes; further runtime admission stopped. After the approved
removal, aggregate allocation is50,647,777,280bytes, leaving891,830,272bytes.
Source checks may continue within that margin; a new runtime must separately fit
its declared scratch. No production, existing ships, manual-review state or
unclassified user cache was altered.

## M645 — Preserve recording segments and continue with explicit missing coverage (2026-11-18)

Metadata3 fences the linked-segment layout from older readers, retaining exact
metadata1/2 single-segment reads and original binary encoding. Size/explicit
rotation syncs a new unique header/directory with original metadata, sequence and
previous digest. Same-capture rotation keeps its receive-clock epoch; returned
frame metadata identifies segment and epoch. Header and retained partial bytes
count against the session ceiling, with a1024-segment bound. Host retention and
filesystem-overhead accounting remain separate requirements.

Exact-metadata continuation takes the original exclusive writer lock, validates
its entire bounded prefix and rechecks file identity/size after reserve admission.
It preserves every old byte and opens a new epoch with a durable unknown-loss
restart gap. Crossing an original torn frame requires corrupt-tail evidence,
even through empty successor headers. Corrupt complete frames, missing/foreign
segments, incomplete headers, changed bindings/live writers and observed terminal
processes refuse. Reattachment/runtime authority remains the future factory's
responsibility; a host/identity change requires a distinct session.

Full recording race checks and vet pass, including actual owned writer-process
death, concurrent paging/rotation, exact cursor replay, raw bytes/timing/streams,
new-epoch wall reversal, pressure/quota, links and post-scan replacement. A quota
fixture's first segment bound did not require rotation; its corrected bound
checks the additional header admission. The actual original2 writer/reader
compatibility rehearsal is prepared but not yet run. No capture worker, index,
retention/pins, Docker/Vere, public API/UI or installed qualification is claimed.
No installed schema, namespace, ship or runtime is changed.

## M646 — Qualify the actual original2 reader/writer boundary (2026-11-18)

Supersedes M645's actual-format compatibility deferral for frozen2b5af45.
The declared source-only rehearsal builds unchanged2eb1b67 production sources
with its original metadata/identity fixtures and a separately hashed helper.
Its writer emits populated metadata2 with exact binary/control bytes, stderr,
gap and exit evidence. The candidate repeatedly reads, pages and reopens it
without any file change; segmented continuation refuses the original layout.
The actual old reader refuses the candidate's rotated/continued metadata3 store
unchanged. Source/helper/binary/file pins and [scoped evidence](diagnostics/2026-11-18-recording-segments3-source-upgrade.json)
are retained; an actual original2 fixture now runs in ordinary focused tests.
No Docker/Vere capture, independent worker, retention, API/UI, installed upgrade,
DEV or stable qualification is implied.

## M647 — Bound a fresh peer with a read-only task-only backing (2026-11-18)

VM05's copied input filesystem is classified read-only against all1459 retained
canonical files, including exact frozen binaries/scripts/framework/runtime/pill
and rootless bundle. An initial classifier refuses zero-inode unused lost+found
slots before any deletion; the corrected classifier accepts only that exact empty
slot shape. Exact hashes/source paths and small original scripts/scenario remain
in a private manifest. Retirement frees324,628,480allocated bytes. The full
current-failure root/pier/journal remains unchanged; its original capsule is now
marked nonlaunchable without reconstructed inputs, never relabeled successful.

Prepare a separate fresh COW capsule with that exact ended task-only root and its
pristine base exposed read-only. Before systemd, its private PID1 suppresses
automatic management startup and omits startup resume for the explicitly bound
source service. A new~nec uses separate instance/port/output/request IDs and the
public cached-pill registration API. Original~zod must stay stopped with unchanged
StartedAt; its original pending receipt/rows/lease must stay exact. No old pending
phase is redispatched/reconciled/rebound/abandoned. Only the new peer is operated.

The fixed-case helper's first compile catches a fixture-local boolean colliding
with the existing selected-target name; renaming fixes it. Focused pure helper
race checks, Python refusal guards and isolated PID1 compilation pass. Declare
896MiB additional scratch under48GiB, retaining one active runtime and existing
memory/no-swap/CPU/deadline/no-network boundaries. The allocation and full native
workflow are not yet qualified. This exercises a source service on a derived
fixture, not the installer or a non-destructive installed upgrade. Production,
manual DEV and all user's ships remain excluded. See the [capsule contract](../scripts/qualification/grubbery-app-cow/README.md).

## M648 — Preserve the bounded COW bootstrap failure and restore storage (2026-11-18)

Freeze matching4source artifacts at081b00d after M647's guards. Final admission
is short by6,467,584bytes; lossless archival of the superseded5eeae9a binaries
verifies every inflated original byte and frees46,993,408allocated bytes. Exact
artifacts/source/provenance remain, with archival sidecars for former raw paths.
No unclassified Go cache, installed artifact, namespace or service is changed.

The fresh COW gate boots the intended source service without startup resume and
reads the original pending receipt, then hits its declared896MiB scratch bound
during new~nec Start. The owned scope terminates; no clean runtime exit is
qualified. Allocation after stopping is1,001,316,352bytes. Between five-second
samples aggregate allocation also rises21,491,712bytes above48GiB. This is a
failed resource gate, not permission to exceed either bound or silently raise it.
No framework/app phase or new app journal operation runs. The [scoped failure](diagnostics/2026-11-18-grubbery-cow-bootstrap-budget.json)
does not qualify native source/state/restart or installed behavior.

Restore the cap by retiring only the new byte-verified copied input filesystem,
freeing85,594,112bytes. A separate bounded48MiB no-network kernel utility mounts
the ended COW source read-only/noload/noexec with both backings read-only. It
executes none of that disk's code and preserves exact SQLite/WAL/SHM and commands.
Analysis uses copies; its first query assumes a nonexistent kind column and
refuses, then uses the actual JSON body schema. All original app header/event/
lease rows remain exact: one operation,17events,one lease; no new app operation.
Both source and original backing hashes remain unchanged after inspection.

Retire only this explained obsolete bootstrap overlay after those evidence
checks, freeing897,126,400bytes. Its full new partial fake/pier is not retained;
its exact original store, failure/logs/source/artifacts and retirement hashes are.
The complete original VM05 with acknowledged app uncertainty remains unchanged.
Storage is under48GiB again. Replan native scratch using measurements; do not
repeat the undersized case or turn source checks into a native pass.

## M649 — Keep conditional compiler verification inside its read scope (2026-11-18)

Review for runtime recovery finds a real M641 source integration gap. Ordinary
capture returned before observe-code's additional compiler read; the real scope
had already closed, while the inert model incorrectly left its reader usable.
Tightening that lifetime first deterministically reproduces an uncertain
observe-code phase. This narrows M641's complete scoped model evidence; it does
not explain the earlier5eeae9a timeout, which predates the optimization. The
081b00d COW gate ended before app phases and made no compiler-read pass claim.

All conditional verification reads now occur inside the capture callback before
confirmed cleanup. Callback error is checked independently of a runtime's return
value; a closed capture has no usable reader. Original read ordering, byte checks,
evidence digests and native single-dispatch behavior remain. Invalid compiler
reports, ignored callback errors and unconfirmed logout cannot produce a proof;
exact later reconciliation succeeds only with valid in-scope reads and cleanup,
without another effect. No native timer, journal, schema or API changes.

The full Grubbery race suite, actual SQLite reopen/missing-cache recovery, actual
private HTTP scope lifetime/cleanup and focused vet pass. [Scoped source evidence](diagnostics/2026-11-18-grubbery-scoped-compiler-lifetime.json)
retains reproduction/fix boundaries. Matching artifacts and native qualification
remain pending. Continue with an explicit audited runtime recovery contract;
never edit original plan/events to work around their old container binding.

## M650 — Bind app recovery to an immutable prefix and fresh process facts (2026-11-18)

The first recovery model preserves original plan2/actor/IDs/events and changes
only a separately reviewed effective container selection. It binds exact prefix,
phase/state, previous recovery head, process PID/start/boot/engine/image/runtime
and a trusted launch/namespace context digest. Pending phases need verified
prefix evidence; unknown dispatched phases need their exact native read-after
proof. Metadata validation/digests grant no authority: the future authenticated
server must independently produce and compare a fresh review. A five-minute
window and exact prefix/head CAS cannot silently refresh an old review.
Historical linked receipts remain valid after later legitimate progress.

Focused race checks and vet pass for unchanged plan/events/namespace, stale
prefix/head, expiration, self-resealed immutable-identity changes, changed fresh
process/native facts, missing/wrong proof and historical audit ancestry. This is
only a typed source model. Runtime observation/adapter fencing, schema migration,
transactional storage, HTTP/CLI/MCP, genuine62-writer/refusal and native gates are
still required; no existing process check is relaxed or capability advertised.
See the [implementation contract](app-source-runtime-recovery.md).

Prioritize this path before another fresh bootstrap: the complete retained task
fixture already has meaningful app uncertainty. A new read-only-backed capsule
can use supported Stop/Start and audited recovery without another fresh pier.
Keep original before-evidence unchanged. Native artifacts must include M649's
compiler lifetime fix. Disk-bounded source inputs can be compressed and inflated
into guest tmpfs with exact byte pins; budget memory as well as allocated disk.
No new runtime has been launched for this model; production and manual DEV stay
untouched. Other issue tracks remain active.

## M651 — Persist linked app recovery audits without rewriting uncertainty (2026-11-18)

Schema63 adds separately canonicalized immutable receipts, contiguous per-app
heads and a64-recovery bound. Validate every audit against the original plan/
header/event prefix and exact predecessor; reject missing tails/heads, changed
bodies/digests, triggers and unowned successor tables. Migration creates only
empty heads. The internal metadata-only apply checks current prefix/head/lease
transactionally. Exact saved replay precedes expiration/current phase checks;
commit errors remain uncertain and require lookup. Original app/header/event/
lease/source/access/runtime tables are unchanged. No HTTP exposure or native
binding is granted by this storage adapter.

Build an actual original62 writer/reader at3e780879 with a declared source helper.
Its synthetic plan2 has dispatched/acknowledged uncertainty and its held lease,
with genuine59/60 history. Repeated candidate opening preserves every original
row/DDL. The actual old reader refuses a populated63 store unchanged in the
preliminary rehearsal. Retain a public synthetic SQL fixture with original
source/helper/binary/database pins; never publish the private VM database.
The clean-source reproducible rehearsal script is ready for a frozen candidate.

Focused race checks pass for original62 migration/rollback, actual process exit
after a committed audit without close/reply, exact lookup/expired replay,
stale-prefix/head and concurrent admission, corruption refusal and64-history
overflow. Existing original60/61 migration checks pass after comparisons exclude
only the new successor tables. Synthetic old-version encoder fixtures explicitly
omit only empty new tables; this is not a user downgrade procedure. Focused vet
passes. Runtime process observation/approved adapter, fresh authenticated review,
CLI/MCP and native/installed upgrade gates remain pending.

Remove only46 obsolete regenerable test-package compiler cache objects tied to
this task's exact worktree/unique tests, retaining the newest two objects and all
named frozen binaries/source/evidence. Prevalidated file identity/hashes precede
unlink; the private manifest confirms1,617,776,640allocated bytes freed. A sandbox
read cannot see the DEV PID namespace; the same fixed read-only measurement with
host authority succeeds. Charged storage is49,626,271,744bytes under48GiB, leaving
1,913,335,808bytes. No runtime experiment or user service is operated.

## M652 — Qualify the frozen audit store against the actual62 reader (2026-11-18)

The clean217275f source gate builds unchanged3e780879 Go/embedded inputs/tests
with the declared metadata-only helper and a separately pinned candidate binary.
The actual62 writer creates acknowledged plan2 uncertainty with its exact lease
and original59/60 ancestry. Repeated63 migration/reopen preserves all original
rows/DDL/access/history and that uncertainty. A new metadata-only audit changes
none of those original rows. The actual62 reader refuses the populated63 store;
all closed original/successor files remain byte-identical before/after refusal.
Source/helper/binary/file pins are in the [scoped report](diagnostics/2026-11-18-app-runtime-audit63-source-upgrade.json).

This supersedes M651's frozen-source rehearsal deferral only. Runtime observation,
approved process adapter, authenticated fresh review/apply, CLI/MCP and native/
installed/DEV gates remain pending. No service, engine, runtime or user ship is
opened. Both source fixtures and exact binaries remain retained privately.

## M653 — Fence original app requests on a separately approved process (2026-11-18)

The private managed/project desk constructors now supply a lazy trusted process
reader. Full validated container/launch/network snapshots, fixed namespace stat
and host boot are bracketed with exact engine/process checks. Context hashing
exports no path, argv, environment or credential. Runtime version names the
verified pinned image contract, not a new live executable probe; the project
qualifier explicitly supplies that contract rather than an absent ordinary
ManagedState field. Project evidence includes the unchanged complete fleet.

A read-only recovery view accepts only the original binding and refuses every
GrubberyStep. An audit-backed view validates immutable ancestry, checks a fresh
full process observation and accepts the unchanged original binding/requests on
that approved process. Each later transport boundary rechecks it. No original
plan/payload/ID rewrite or generic URL/command override is introduced. These
trusted adapters require server actor/role/policy/lifecycle and saved-head
admission; they are not registered or advertised and do not grant authority.

Initial observer fixtures refuse because old inert containers omit PID/start;
explicit fixture facts fix that gap while production validation stays strict.
A project namespace test initially changes unused bookkeeping; making the actual
fixed stat response change reproduces/refuses namespace drift. Focused race
checks then pass for ordinary/project PID/start/engine/namespace/launch/port/peer
drift, scoped auth/cleanup/lifetime, sealed original binding and read-only writes.
An exact saved audit permits one unchanged original request on the selected
replacement; namespace drift blocks a later effect. This is inert adapter-level
evidence, not native or authenticated server recovery. Focused vet passes.
Fresh native prefix/unknown-effect review, API/CLI/MCP, matching native artifacts,
installed upgrades and preserved DEV remain pending. No runtime is launched.

## M654 — Observe a retained native prefix without resolving its phase (2026-11-18)

The internal native recovery observer requires the exact unfinished plan2 record
in complete validated ownership history. Pending review captures only retained
observed effects; unknown dispatch/acknowledgement/uncertainty requires the exact
already applied candidate effect. All previous observed compiler phases are
verified again inside authenticated capture before confirmed cleanup. Version
file allowance derives only from its original observed/candidate phase, so a
post-compile prefix is checked with its actual expected inventory. No pending
compile is submitted. Fresh complete history and admission checks bracket the
capture; concurrent prefix progress refuses. No source-cache load, native write,
phase dispatch/resolution or journal edit is permitted by this observer.

The initial stale-prefix fixture tries to advance after disabling its source
cache and correctly refuses; move that earlier fixture action before cache loss
to exercise the intended stale CAS. Focused race checks then pass for absent/
compiled/versioned pending prefixes, compiler/cleanup/unowned/guard refusals,
exact unknown original effect with a missing cache, changed native version,
reverified prior compiler output and prefix progress during capture. Reads do not
change original receipts or write counts; unknown effects are never redispatched.
Focused vet passes. These are inert native-contract source checks. Authenticated
fresh review/apply, selected runtime/head integration, CLI/MCP, matching native
artifacts and installed/DEV qualification remain pending. No runtime is launched.

## M655 — Compose fresh review, audit CAS and later phases privately (2026-11-18)

Private server composition now selects the registered replacement through the
existing target/config/lifecycle/framework/pier/role guards. A trusted runtime
interface supplies fresh process facts and its sealed read-only view. Native
prefix evidence and complete saved record/head checks produce the review. Apply
reobserves independently and compares fresh facts before the audit transaction;
self-resealed caller claims are not authority. Exact saved replay precedes live/
cache/management admission. The private later-phase factory loads the current
saved audit head, fences its approved process on each admission and preserves
original binding/IDs/payloads. Writes still require current policy and source.
Ordinary phase factories remain strict. No public route uses these paths yet.

The extracted stage loader initially references an out-of-scope actor; compilation
catches it and using the already authenticated original binding actor fixes it.
Actual SQLite with inert native/process fixtures passes changed PID claims,
namespace/native/config/actor/reader/fence drift, stale heads and expired reviews.
The first prefix assertion compares decoded pointer identity; comparing complete
event values fixes that fixture error. Fresh audit/app preservation, store reopen,
exact cache-free reconciliation after revoked Commit, denied later writes and
saved replay with unavailable runtime/management fence pass. Existing strict
normal guards and original cache-free lost-reply recovery pass under race.
The scoped transport regression and focused vet pass after the runtime factory
returns only its required native interface. No original private VM or service
is opened. Public HTTP/CLI/MCP, matching native/installed and DEV gates remain.

## M656 — Expose typed runtime recovery with preserved replay boundaries (2026-11-18)

Register typed review/confirmed-apply and saved audit/head HTTP routes, shared
CLI/stdio operations, pinned client/registry methods and exact delegated SSH
feature/body/deadline guards. The separate compiled-API feature supports exact
apply replay even with no live adapter; saved-receipt support is independent.
Neither claims native workflow readiness. Apply requires the original operator
and explicit confirmation, reobserves review facts and writes only its audit.
Original phase routing selects only the validated saved head/approved process.
Saved status validates original history and audit in one SQLite snapshot outside
the lifecycle mutex. A missing original is an explicit absence, not corruption.

Actual HTTP/SQLite source checks pass for empty/missing heads, fresh review,
unconfirmed refusal before native access, confirmed audit without original event
changes, native-free saved lookup, cache-free original reconciliation and saved
apply replay after progress with no native adapter/management admission. Existing
source phase recovery still passes. Actual CLI and official-SDK stdio processes
return byte-identical typed results through the catalog, retaining exact IDs and
no retries. Apply remains a mutation-gated journal write. Declared timeout checks
and delegated SSH source/review/apply role/capability/96KiB bounds pass; focused
vet passes. These remain source/inert transport gates.

A separately gated exact-built-client rehearsal is prepared to drop the apply
reply after commit, replace the test manager without a native adapter, and read
the original audit through CLI. It requires an explicit binary SHA and has not
yet run. Matching native artifacts, retained disposable VM recovery, installed
upgrades, full app workflow and preserved DEV remain pending. Production and all
existing user ships/services remain excluded; no runtime is launched here.

## M657 — Qualify built-client lost apply reply across manager replacement (2026-11-18)

Freeze clean bc047062 Go/embedded/test inputs and build the CLI in a newly admitted
256MiB source capsule. Exact binary970dda1b and source-manifest pins are retained.
Actual official-SDK stdio reviews and applies through the real managed API/SQLite
with an absent source cache and inert process/native observation. Drop the first
apply reply only after successful commit. MCP reports outcome-uncertain and sends
one apply, with no retry. Replace manager/stdio, reopen SQLite without a native
adapter and hold a management fence. CLI saved lookup returns the original exact
audit and still-uncertain original plan/events, with no native/cache access.
The original fixture has one inert marker write; recovery adds none.

The exact gated race test passes in13.033s. [Source evidence](diagnostics/2026-11-18-app-runtime-audit-cli-mcp-source.json)
and private commands/source/binary/storage are retained. This supersedes M656's
built-client rehearsal deferral only. It is not real runtime/installed/DEV
qualification. Continue with a separate read-only-backed disposable VM using its
retained app pier, supported lifecycle actions and explicit audited recovery.
No fresh bootstrap or source DB/receipt rewrite is needed. Production, existing
user ships and manual DEV remain untouched. Other issue tracks remain active.

## M658 — Prepare a bounded copied-pier native recovery case (2026-11-18)

Refresh protected-ship and experiment constraints before preparing a separate
fixed-case [native recovery capsule](../scripts/qualification/grubbery-runtime-cow/README.md).
The exact ended VM05 and pristine base remain read-only dependencies. PID1
validates both predecessor and new scenario before systemd and suppresses startup
resume. Hash/size-verified compressed binaries inflate into guest/run tmpfs.
The guest requires one exact initialized disposable~zod, verifies original62→63
app-row preservation before operations and checks the original process is stopped.
No fresh bootstrap, database repair or original plan/event rewrite is permitted.

The guarded helper uses supported Stop/Start on that copied existing pier, then
actual CLI/stdio review/apply and exact read-after reconciliation. It completes
the original source, checks second-version state/new code, another same-pier
replacement and manager restart historical audit/completed replay. Every original
header/body/stamp and17-event prefix must remain exact; legitimate later source
progress appends and may change its revision/lease. Exact lost phase/apply replies
permit saved lookup only, retaining each original invocation envelope separately.
No mutation is resubmitted merely because its reply is lost.

Declare1280MiB capsule scratch plus64MiB host build reserve under48GiB. Compressed
inputs use a64MiB filesystem. QEMU alone inherits a1152MiB kernel hard file-size
limit; staged inputs/root/log/report allowance must fit before launch. Monitor
allocation/logs each second. Existing guest/supervisor memory/no-swap/CPU/no-NIC/
deadline limits remain. Size/I/O/forced termination is a failed clean-shutdown
gate, preserving evidence; this is not permission to overshoot. Original fixture
and host production/manual DEV/user ships stay excluded from mutation.

Five refusal/input tests pass, including an actual kernel file-size refusal
retaining exact first bytes. The new PID1 compiles and exits80 as an ordinary
host process before mounts. Fixed helper compilation/separation race checks and
focused vet pass. No VM has launched for this case. Freeze matching artifacts,
reclaim only classified task compiler-cache objects as needed and admit storage
before native qualification. Capacity and full native recovery remain unqualified.

## M659 — Freeze matching recovery artifacts and admit the bounded case (2026-11-18)

Freeze clean6676c9ec source for Omabit, collector, runner and the explicitly
source-pinned native helper. The first build uses an incorrect collector cmd
path and refuses before any collector/runtime output; the actual backend/cmd
paths fix it, preserving the verified matching CLI and original failed command.
All4 final hashes/build identities and exact Go/embedded/test/harness input pins
are retained. No installed file or service changes. See the [artifact boundary](diagnostics/2026-11-18-app-runtime-cow-artifact-freeze.json).

Classify76 older regenerable managed/runtimequalify/CLI test-package AR objects
by exact task worktree source file plus task-only test symbols, inode/owner/link/
mtime/content/build identity and exact archive members. Validate every selected
and retained object before unlink; a per-file removal manifest records progress.
Retain the newest2 per package and verify all named frozen binaries unchanged
before/after. This frees867,160,064allocated bytes, touching no source, SQLite,
runtime image or installed artifact. No Go build/test remains active during it.

Read-only admission charges50,089,832,448bytes and declares1,409,286,144bytes
(1280MiB capsule+64MiB host reserve), totaling51,499,118,592 under the approved
48GiB cap. This is admission, not a native pass. Run one guarded offline COW case
with the fixed original read-only, hard QEMU root-file bound and per-second
monitoring. Preserve any failure and report limits; no production/manual DEV or
existing user ship operation is authorized by this checkpoint.

## M660 — Preserve acknowledged native timeout and tighten exact observation (2026-11-18)

The first frozen6676 native runtime-recovery capsule fails its whole workflow at
phase9's four-minute bound. Schema62→63 preserves original app rows; supported
Stop/Start resumes the same initialized copied pier. Actual CLI review and stdio
MCP apply save one linked audit without rewriting the original17 events. Read-only
reconciliation observes original phase6; compile and version-file phases pass.
The next desk-directory phase dispatches once and is acknowledged, but its final
observation is not saved. The runner returns app_journal_uncertain; the shared
facade incorrectly labels this runner_rejected and the helper stops. Neither the
new source nor later state/replacement/restart gates pass. See the [failure evidence](diagnostics/2026-11-18-app-runtime-cow01-failure.json).

After both test units end, a fresh512MiB offline utility mounts the failed disk
readonly,noload,nodev,nosuid,noexec and exports16 bounded SQLite/proof/invocation
files. The closed store has26 events, one audit, exact original header and17-event
prefix. Source/base hashes remain unchanged; retain the complete new failure
root and original VM05. This is independent saved evidence, not a DB repair or
permission to redispatch the acknowledged phase. Host/guest exit cleanly; root
allocation is162,009,088bytes and aggregate50,299,912,192bytes, below48GiB.

Complete process snapshots now each include containment and fresh exact HTTP
ports, alongside all prior container/launch/network/engine/namespace facts.
Remove the redundant surrounding process guards while retaining two complete
snapshots, two namespace/boot observations and both engine brackets. No identity,
ownership, launch, port or peer check is relaxed and no deadline is extended.
Add drift between snapshots for PID, launch, ports and engine plus project ports.
Focused runtime race/vet pass, as do full Grubbery and service race suites and
focused app SQLite/HTTP recovery checks. This is source evidence only; native
latency and complete workflow need fresh matching artifacts and qualification.

On mutation, the shared facade now classifies the runner's explicit
app_journal_uncertain as outcome_uncertain with its original error code and exact
saved-lookup guidance. It does not resubmit effects. Read-only and before-effect
failures retain their existing classifications. Source refusal/privacy tests pass.
This supersedes M659's pending first-case result only; other issue tracks and
installed/DEV/production boundaries remain unchanged.

## M661 — Freeze the optimized candidate within a tighter measured capsule (2026-11-18)

Freeze clean569f39e6 matching CLI, collector, runner and source-pinned native
helper with exact Go/embedded input hashes. Focused actual CLI/stdio app parity
and managed/service vet also pass after M660. The [new artifact/admission pins](diagnostics/2026-11-18-app-runtime-cow02-artifact-freeze.json)
qualify source identity only, not native behavior. Preserve the old6676 executables
and both original/current-failure full roots. No installed files change.

Classify88 obsolete compiler test-package AR objects from this goal's exact
worktree and goal-only test symbols, retaining newest2 per package. Verify owner,
link/inode/mtime/content/build identity, exact AR members and every named frozen
artifact before removal; record each completed unlink. Reclaim350,732,288allocated
bytes without source, SQLite, executable or runtime-state removal. Stop only the
superseded task classifier after its exact process identity is verified; the
replacement uses bounded efficient substring scans and its manifest is retained.

The first copied existing-pier run used162,009,088allocated root bytes, without
fresh bootstrap. Tighten this case's kernel hard root-file cap to896MiB, still
over5x that measured allocation; reserve1024MiB capsule plus64MiB host staging.
All input/log/report bounds must fit before QEMU, and reaching the bound still
fails the gate with evidence retained. Five refusal/input/kernel-bound tests pass.
Admission charges50,297,249,792bytes and reserves1,140,850,688bytes, totaling
51,438,100,480 below48GiB. These are capacity/source gates only. Use a fresh COW
from unchanged read-only VM05, never resume or rewrite the acknowledged failures.
No production, protected/tmux ship or manual DEV operation is involved.

## M662 — Complete original native source; preserve whole-case timeout (2026-11-18)

Frozen569f39e6 completes all12 original source phases on the supported replacement
runtime, each with exactly one durable dispatch. Migration/audit and original
header/17-event preservation pass again. Compiler/version phases take about195s,
desk creation210s and desk owner224s. Poke-source returns explicit journal
uncertainty; follow observation loses its response near the four-minute bound.
The actual CLI now classifies both as outcome-uncertain, reads the saved
acknowledged phase and selects read-only reconciliation. Neither effect is sent
again. The original completes at1713s with36 immutable events. The fixed helper
reads initial app state and receives an acknowledgement for its state update.
Version two reviews and begins through actual stdio MCP, but the whole30minute
fixture context expires during its first advance. Whole qualification fails.

After clean VM shutdown, a fresh offline utility exports35 bounded SQLite/proof/
request/response/stderr/saved-lookup files through readonly,noload,noexec mounts.
It limits aggregate raw exports to16MiB and kernel-bounds serial output to32MiB.
The exact store retains one immutable runtime audit, the completed original,
and version two with zero events/nine pending steps and its lease. No second
source phase dispatch occurred. Source/base/original hashes remain unchanged;
retain the full218,238,976allocated-byte new root and all earlier failure data.
See [scoped native evidence](diagnostics/2026-11-18-app-runtime-cow02-deadline.json).

The measured first workflow leaves insufficient time for the separate9-phase
second version and state/restart checks. Increase only the whole fixture context
to90minutes, its test process to95minutes, guest supervisor to5750s, VM to6100s
and outer supervisor to6300s, leaving cleanup between bounds. Each product RPC
still has its existing four-minute limit and lookup-only uncertainty handling.
Memory/CPU/no-swap/no-NIC, hard896MiB root bound, per-second monitoring and48GiB
aggregate admission are unchanged. Six input/deadline/kernel-bound guards pass;
the helper compiles and its host tests skip the real case outside the guarded VM.
New matching artifacts and a fresh complete native run remain required. No
production, manual DEV, protected/tmux ship or original failure is operated on.

## M663 — Share pinned fleet/private previews and document client setup (2026-11-18)

Add runner.snapshot.v1 over the existing typed snapshot client, checking response
runner/mode in addition to enrollment/connection identity. Its catalog includes
live project/storage probes. Add runtime.log-tail.v1 as an operator-private
bounded sanitized preview, requiring explicit private-data selection, instance,
exact configuration revision and full container ID. Withhold a changed reply.
This is not comprehensive recording access, export or following; legacy log
watch stays separately excluded in discovery rather than claiming stream parity.

A legacy read using POST can return OutcomeUncertainError from its transport.
The facade now follows the declared operation effects: read-only failure remains
read-failed, while mutations retain lookup-first uncertainty. No request is
retried by this change and arbitrary diagnostics remain private. Actual CLI
argument-dispatch and official-SDK stdio subprocesses return byte-identical
snapshots/previews/refusals across observer/capability/response identity, mode,
container/revision, private opt-in and lost-transport cases. Each valid/withheld
pair sends exactly two reads; preflight refusals send none. Revisions above2^53
remain exact. Existing source/runtime recovery parity also passes (24.314s total).
Full service/MCP race suites and focused vet pass. Fixtures are inert private Unix
peers, not production, native UI or named-client connection qualification.

Use the OpenAI Docs skill and primary vendor pages to verify stdio configuration,
protocol and timeout fields for [client recipes](mcp-clients.md). Parse both JSON
examples and TOML. Pin T3's official provider-injection document at c2a7c5cb;
its orchestration service is separate and exact external-server forwarding in a
selected installed provider remains a connection gate. No client profile, T3
thread, plugin or user setting changes. This supersedes the recipe deferral only;
actual named-client connections, remaining operations and HTTP delivery still
need their own checks. The full six-issue goal remains open.

## M664 — Compose durable recording capture without changing lifecycle (2026-11-18)

The source-only CaptureDocker coordinator persists a fresh metadata3 session
before trusted output attachment, then acknowledges durable readiness separately
from Start. One receive-clock origin retains monotonic ordering through wall-clock
reversal. Existing segmented storage fsyncs exact original multiline/long/binary/
credential bytes, without formatting or redaction. The metadata/frame format is
unchanged and actual original1/2 fixtures remain byte-exact in the full source
suite. No installed capture is enabled.

An independent supplied terminal-process proof must match the exact metadata
container; EOF alone produces a gap. Interrupted payloads retain their received
prefix, gap and separately verified exit where available. Cancellation closes only
this output connection, never signals the process, and serializes Close across
cleanup races. Detach immediately after read completion/error before bounded exit
observation or storage diagnostics, avoiding unread-stream backpressure. A failed
Start/attachment never fabricates an exit. Private callback diagnostics do not
escape status. Pressure/write failure remains explicit even when the gap itself
cannot be saved; no reconnect, repair or runtime mutation is retried.

Full recording race tests pass in1.263s and focused vet passes. Fixtures cover
durable readiness, segment rotation/raw/timing preservation, nested metadata
isolation, exact completed-session refusal, partial transport, EOF/running,
wrong exit, storage pressure/write refusal, blocked-read cancellation and legacy/
incomplete/canceled admission before any file or attachment effect. These are
inert streams/private files, not a native process observation. See the
[coordinator boundary](runtime-recordings.md#source-capture-coordinator).
Trusted runtime identity/launch factories, independently supervised workers,
start-failure coordination, retention/pins, indexed seeking, API/UI/export and
real Docker/Vere/installed gates remain. This supersedes only the absence of a
source capture coordinator; comprehensive runtime recording is still unqualified.

## M665 — Close the native artifact boundary over embedded inputs (2026-11-18)

The native launcher previously compared Go/modules and appdev assets after an
artifact freeze. That could miss a different embedded input, notably service's
legacy-inventory.json. No such drift is observed in the previous frozen cases:
their actual source stayed clean and later changes were documented harness/docs.
Still, that extension rule was incomplete and must not become qualification
permission for an unrecorded catalog change.

Allow artifact-source differences only for non-Go docs and the four explicitly
named host/guest/Python-test/README harness files. Reject every other tracked
change, including JSON inventory, source samples, QML, Go/modules, paths escaping
docs and absolute paths, before reading artifacts or storage admission. Eight
host guards pass, including a main-entry reproduction of the formerly missed
embedded-inventory change with zero artifact reads/admission/process dispatch.
The next frozen build records hashes for every tracked ordinary source file,
covering all embedded inputs;3781 tracked files currently have no symlinks.
No installed runtime or native helper behavior changes in this guard checkpoint.

## M666 — Admit complete native qualification with every failure retained (2026-11-18)

Freeze clean6e810bc9 matching CLI, collector, runner and native helper, including
M662's longer whole-case context, tested shared reads and unwired capture library.
Hash all3781 tracked ordinary source files, covering embedded JSON/assets as well
as Go/modules; verify clean source and identical hashes after building. Preserve
exact binary/build/input pins in the [artifact/admission record](diagnostics/2026-11-18-app-runtime-cow03-artifact-freeze.json).
This is source identity/capacity evidence, not a complete native workflow pass.

With no Go build/test or runtime active, classify30 obsolete task test-package AR
objects by exact worktree path and task-only symbols, owner/link/inode/mtime/
content/build identity and exact AR members. Keep newest2 per package and verify
all named binaries before/after. Reclaim109,260,800allocated bytes. Separately
losslessly gzip16 older task executables, verify complete inflated bytes/size/SHA,
then record each raw-path retirement and explicit inflation instructions. Reclaim
208,166,912allocated bytes; all original executable bytes remain available.
Source, SQL, receipts, logs, all three full VM roots and current03 raw binaries
are verified unchanged. No installed file or original migration fixture changes.

The completed first-source case used218,238,976allocated root bytes. Tighten only
this no-bootstrap experiment's hard root-file cap to768MiB, over3x that measured
space; reserve896MiB capsule plus64MiB host staging. Input/log/report worst-case
bounds still fit before QEMU, with per-second aggregate checks. Eight input/
embedded/deadline/kernel-file-limit guards pass. Reaching the file/storage bound
still fails qualification and retains its evidence; no hidden reservation growth.
Admission charges50,487,672,832bytes and reserves1,006,632,960bytes, totaling
51,494,305,792 under48GiB. Run a fresh original-VM05-backed copy, preserving the
completed-version-one/new-pending-version-two failure as separate evidence. No
production, existing ship, manual DEV or tmux operation is involved.

## M667 — Represent bootstrap unknowns without inventing identity (2026-11-18)

Status: implementation/rehearsal prepared; source/native compatibility gates
pending. New comet creation stores ManagedComet{} without a ship name, and
pierstore.Identify requires both the pier and its.urb directory. These source
facts prevent a correct first-Start recorder from requiring a known native name
or initialized pier object. Do not create/alter a pier merely to fill recording
metadata, or substitute a fake identity for mining output.

Prepare metadata4 with explicit native_ship_status registered or
pending-comet-mining, and pier_binding existing-pier or prepared-storage. A
registered label remains expected intent, not an answered native identity. A
prepared-storage digest binds verified exclusive storage intent, not a future
inode. Pending identity has no ship name and is not a project member. Existing
metadata1–3 keeps byte-exact canonical serialization, with omitted basis fields
left unknown. Segments/continuation accept4 without rewriting old files; frame
codec1 is unchanged. New source capture uses current4. Later confirmed identity/
pier/process facts need separately linked evidence and are not implemented by
changing the attachment label.

Prepare source gates for unknown bootstrap bytes, strict mixed/guessed-basis
refusal before storage, exact metadata across rotation/continuation/exit/reopen,
and legacy canonical preservation. Prepare an actual frozen6e810bc9 original3
segmented writer/reader plus declared helper and new4 preservation/refusal
rehearsal. Three Python launch/input guards pass without a source build or
runtime effect. Go race/vet and the actual writer rehearsal remain pending;
do not infer them from preparation. The independent native app case continues
using its frozen6e code, unaffected by these working source changes. No installed
capture, production/manual DEV or user ship is changed. See the [upgrade boundary](upgrade-policy.md#development-explicit-recording-bootstrap-identity-metadata4--gates-pending).

## M668 — Prepare ordinary recording admission from actual native contracts (2026-11-18)

Status: source prepared; Go/native gates pending. The prior source coordinator
accepted trusted callbacks but had no ordinary native factory. Prepare a factory
for a separately supervised worker's newly created ordinary Vere container. The
authenticated manager must supply the durable original job and hold its lifecycle
guard; public target/revision/request names alone cannot prove that authority.
Project contracts and installed launch remain unwired.

Verify the private account/engine and exact unambiguous created full container,
launch/containment/network, exclusive mount and existing-pier identity or empty
prepared storage, with boot and own recorder namespaces. Require a saved create
dispatch without unresolved resources. Recheck the exact created snapshot before
and after attach and immediately before durable readiness. Detach on change;
never Start/Stop/signal/wait/exec or modify storage. Hash private launch/path
context; do not return it. Copy runtime state into a fresh zero value so JSON
decode cannot reuse the caller's nested pointers.

Independent exit reads bracket the same exact terminal container and launch,
daemon/boot/storage context. Missing/out-of-range process times, a live process,
changed evidence or error refuses without inventing an exit. Preserve nonzero,
OOM, daemon-error presence and reversed wall time as distinct evidence; omit raw
daemon diagnostics. Prepare inert native-seam/storage/capture tests for bootstrap
bytes, pointer isolation, pre/during-attach drift, ambiguous admission and terminal
evidence. Only gofmt and diff checks have run. Go race/vet await the single native
app case's completion, avoiding build-cache growth inside its reserved budget.
This supersedes no qualification deferral; installed capture is still disabled.

## M669 — Verify bootstrap metadata and ordinary capture admission in source (2026-11-18)

Full recording race tests pass in1.263s, preserving actual original1/2 fixtures,
metadata3 canonical bytes and segmented continuation, plus explicit4 bootstrap
unknowns through raw capture, gaps, exit and reopen. Targeted ordinary native
factory/attachment/process race tests pass in2.957s with inert engine fixtures
and private temporary files. Native-seam tests preserve bootstrap raw bytes and
independent abnormal exit evidence, refuse changed engine/boot/container/TTY/
launch/network/pill/storage and caller metadata, detach on between-attach drift,
and reject ambiguous/unbound create intent before contacting output transport.
Caller edits cannot rebind copied state or metadata. Focused vet passes.

These gates supersede only M667/M668's pending Go-source checks. Their actual
original3 writer/reader compatibility rehearsal is still pending. The source
factory still requires the authenticated manager's durable job/lifecycle guard;
it is not wired into installed launch and creates no independent worker. No real
container, recording coverage, retention, API/UI or installed upgrade is qualified.
Three Python original-writer launch/input guards also pass. Production, manual
DEV, existing ships and tmux remain untouched.

## M670 — Retain completed app phases and failed manager-readiness gate (2026-11-18)

Frozen6e810bc9/native harness15f766a completes both real Grubbery source workflows
through CLI/official-SDK stdio. Original12 phases have36 events; second9 phases
have27 events. Independently retained SQLite shows every phase observed with
exactly one dispatch, no app leases and one runtime recovery audit. The original
header and17-event prefix remain exact. Initial saved state and new public-v2
code survive the second supported same-pier Stop/Start. Saved storage identity,
instance configuration and revision remain exact. These are scoped native results,
not a whole-case pass; see the [failure record](diagnostics/2026-11-18-app-runtime-cow03-readiness-failure.json).

Immediately after acknowledged systemctl manager restart, the fixture's snapshot
GET returns ENOENT for its Unix socket and the helper fails at2825.69s. It does
not wait for Type=exec's process to listen. A startup window is plausible, but the
eventual manager health is unobserved. Historical audit/completed-source API
replay and final state read-back are skipped. No guest cleanup result arrives for
over five minutes; terminate only the exact task supervisor/VM scope and retain
the full root as unclean/failed. Both cgroups end; original VM05/pristine base and
input payload remain unchanged. A fresh read-only,noload,noexec utility exports53
bounded DB/WAL/SHM/proof/call files; SQLite quick_check passes. Analyze copies,
preserving exact exports. The first utility safely refuses an absent transient
payload mount path; the next reads its separately pinned read-only input disk.

Replace immediate fatal readiness with a one-minute read-only context bound.
Retry only missing/refused sockets and transport EOF; require exact runner/mode,
single instance, ship and configuration plus actual current container evidence.
Protocol, permission, identity and configuration errors fail immediately. No
mutation is retried or deadline extended. Source/race readiness checks pass
in1.018s and vet passes. An initial test fixture used a revision above the desk
contract limit and hung; retain that failed source gate, terminate only its exact
private managed.test, correct the fixture and reject invalid expectations before
probing. The product limit is unchanged. This fixes the harness only; fresh native
manager/replay and whole-case qualification remain pending, with no installed or
production operation.

## M671 — Preserve actual original3 recording bytes across metadata4 (2026-11-18)

The [source compatibility rehearsal](diagnostics/2026-11-18-recording-metadata4-source-upgrade.json)
passes from clean0b28bac. Build unchanged frozen6e810bc9 ordinary Go/modules/
embedded sources plus its declared qualification writer helper, recording every
Git blob/hash and helper SHA. The actual original writer creates six files across
four segments, raw stdout/stderr/control/invalid-UTF8 bytes, recorder-restart gap
and a typed exit fixture. Candidate4 reads/cursors/reopens preserve every
original byte and leave omitted basis fields unknown. The actual original3 reader
refuses the populated4 successor without modifying it. Private commands, exact
source/binary pins and both original/successor byte manifests are retained.

Retain the compact actual original3 files, source-input manifest and declared
helper with writer/source/helper SHA pins. Ordinary regression checks verify
original octets, cross-segment cursors, different receive-clock epoch/gap, terminal
evidence, repeated reopen and unchanged files after refused continuation. Full
recording race tests pass in1.258s with all original1/2/3 fixtures; vet passes.
This supersedes M667/M669's actual original3-writer deferral only. The factory is
still unwired; independent workers, native bootstrap/recording coverage, retention,
API/UI/export and installed qualification remain pending. No user recording,
native runtime, production, manual DEV or existing ship was opened by this gate.

## M672 — Bind an independent recorder's private handoff without private bodies (2026-11-18)

Add an unwired version1 canonical private job codec, bounded64KiB, for an ordinary
manager-owned launch. Bind original runner/mode/target/revision/request/operation,
actor, runtime identity, exact executable SHA and native admission. Whitelist the
state fields required by the factory, excluding signing keys, retained logs,
user endpoints, runtime diagnostics and unrelated histories. Reject original
quarantine instead of sanitizing it into launch authority. Canonical decoding
refuses unknown/duplicate fields, future versions, malformed/sensitive state,
invalid paths/limits and credential-bearing pill URLs. New jobs copy nested
state; exact uint64 bindings remain intact.

Worker admission must be independently observed. Match every native binding,
including daemon/endpoint, host boot, storage/name basis, launch and artifacts;
exclude only fresh session/receive-clock origin and the worker's own namespaces.
No parent namespace is assigned to a future worker. A digest is association only,
never authentication, a fresh observation or Start authority. Targeted job/native
factory/attachment race checks pass in2.395s; vet passes with inert fixtures.
No job file, service, persistent database migration or installed feature is
created by this codec. Claim/readiness durability, actual executable verification,
supervision, lifecycle integration and downgrade/upgrade guards remain required.
The six-issue goal stays open.

## M673 — Persist private recording claims/readiness without replay (2026-11-18)

Add an unwired private job-file source layer, pinning an explicitly authorized
0700 root with0600 owner-only single-link files. Serialize count admission across
processes; enforce explicit max jobs and free-space reserve before creation.
Fsync job/lock and directory entries before acknowledging. Claim once under a
per-job flock, comparing supplied actual-worker executable identity, then fsync
the immutable claim. Active claims refuse duplication; closed/crashed claims
refuse reuse. Existing partial files/directories remain without overwrite,
adoption, repair or deletion.

Readiness requires the same original job/claim, independently admitted metadata
and worker boot/namespaces; it binds session and metadata SHA. Refuse replaced,
linked, exposed, malformed or rebound files/directories and storage pressure.
Saved lookups remain read-only after writer closure: historical readiness is not
live-worker evidence or Start authority. Real supervision/current process checks
and the manager's authenticated lifecycle fence remain mandatory. Source/race
checks pass in2.849s and vet passes with private files/inert native seams,
including distinct parent/worker handles, exact executable mismatch, claim
exclusion/reclaim, no readiness after claim-only, mutation refusal, immutable
partial originals, reserve/quota and concurrent creator limits. This supersedes
M672's absence of source claim/readiness files only. Actual self-executable
observation, independent process/service lifetime, crash/power-loss, integration
and downgrade/upgrade gates remain pending. No installed job root or service is
created, and no ship is operated on.

## M674 — Fit raw capture chunks to configured segments without losing counts (2026-11-18)

The fixed64KiB decoder chunk could exceed a valid smaller segment. Select a
bounded capture chunk after subtracting segment/frame/digest overhead, before
reading the transport. Keep the public decoder's64KiB default. A callback still
acknowledges its entire received chunk or fails: splitting within the callback
would overcount known unrecorded bytes after a partially successful append.
No frame or metadata format changes and no original is rewritten.

Full recording race tests pass in1.275s, including a long binary/multiline trace
across more than16 configured4KiB segments, exact original octets/chunk counts,
and injected pressure after the first acknowledged chunk. Only the next received
unrecorded chunk is counted; the acknowledged prefix stays exact and a gap that
cannot be saved remains explicit. Actual original1/2/3 fixture checks and vet
pass. This qualifies source capture sizing only, with no live engine, installed
worker or comprehensive native-recording claim.

## M675 — Compose exact self-bound worker capture after durable acknowledgement (2026-11-18)

The source worker hashes its kernel-selected self image, reads only own boot/
namespace/process-start facts, verifies the account boundary, claims its original
job and repeats exact native pre-Start admission. Capture syncs metadata/header,
attaches output and syncs readiness before a separate trusted acknowledgement
callback. A complete visible marker can precede successful fsync return; marker
lookup is therefore not that acknowledgement. Failed acknowledgement leaves an
incomplete capture and explicit gap, never successful readiness or exit.

Claim2 binds worker PID/start ticks, boot/namespaces/image and service invocation
when present. Parse Linux comm with embedded spaces/parentheses without weakening
start identity. Legacy claim1 synthetic-shape lookup keeps omitted facts unknown
and cannot claim anew; actual original-writer compatibility remains pending.
Immutable finite result1 records omit raw output/private exception text and
refuse repeated overwrite, invalid success, corruption or unknown fields.
Cancellation interrupts only this attachment, preserving gaps without a ship
signal. Wrong executable/refused admission/storage produces no attachment.

Targeted worker/job/factory/transport race checks pass in3.484s; vet passes.
Fixtures include actual self-executable/proc observation and inert native/data
composition, post-sync callback order, raw long/binary/comet-unknown output,
independent synthetic abnormal exit, stale identity, failed acknowledgement with
a visible marker, EOF/running, blocked cancellation and saved outcomes. No real
worker service or Docker/Vere is operated. Verify primary systemd262 notify/main
process semantics before service work; see the [worker contract](runtime-recordings.md#source-worker-composition-and-acknowledgement).
Independent supervision, fixed-root CLI/configuration, actual crash/lifetime,
authenticated launch and downgrade/upgrade integration remain unwired. No
production/manual DEV/installed state or existing ship changes.

## M676 — Prepare fixed notify and exact supervisor binding (2026-11-18)

Add a bounded fixed READY=1 Unix-datagram notifier, requiring a valid service
invocation and explicit local socket. No arbitrary message, private output,
credentials or runtime effect. Namespace recording units by runner/mode/root/job.
Parse only the exact bounded property response for notify/main/active/running,
full unit, canonical PID, invocation and matching cgroup. Compare independently
obtained live process/image/start/boot/namespaces/invocation against the original
claim; PID reuse, old unknown identity and changed supervisor refuse. These pure
checks are not a fresh probe or Start permission.

Focused supervisor/worker/job/factory/attachment race checks pass in3.537s; vet
passes. The newly compiled gated helper is skipped in ordinary tests. It prepares
a fresh inert service primitive with only self proc/image observations and private
files; actual service acknowledgement/lifetime remains unqualified until run.
No unit, engine, manager, existing process or pier is operated by this checkpoint.
Source CLI/controller, actual old claim writer, active-job upgrade guard and
native recording remain pending. Primary systemd262 sources are linked in the
[worker contract](runtime-recordings.md#source-worker-composition-and-acknowledgement).

## M677 — Qualify the inert notify service primitive (2026-11-18)

The fresh bounded transient Type=notify/NotifyAccess=main helper gate passes
against frozen fb355f8. The exact main process acknowledges readiness, remains
active after its launcher returns, writes a continuation proof, and exits with
success and an empty cgroup. Independent PID/start/image/invocation checks match;
source and executable remain unchanged. Preserve the sanitized
[report and pins](diagnostics/2026-11-18-recording-notify-inert-service.json).

This supersedes M676's pending inert-helper gate only. The helper reads its own
process identity and fresh private evidence; it never opens a job/session store,
manager, engine or pier. Actual recording-worker lifetime, fixed-root CLI,
authenticated launch, active-job upgrade and native capture remain unqualified.
No production, manual DEV, installed configuration or existing ship changes.

## M678 — Observe the exact recorder unit and process (2026-11-18)

Add a fixed, bounded systemctl read for the selected job unit and bracket an
exact-PID proc observation with matching supervisor reads. Bind executable hash,
process start ticks, owner, boot, namespaces, invocation and actual unified cgroup;
changed/dead/stopped processes, ambiguous cgroups, oversized output and changed
unit observations refuse. Only that PID is read, never process arguments or
environment. Invocation comes from the two matching manager observations and is
compared with the original claim. No runtime lifecycle command is sent.

Starting notify/main admission is distinct from active/running readiness and
cannot grant launch authority. Focused source/race checks pass in2.682s with
inert property/process seams and actual self-only proc checks. This is not a new
actual service or native recording gate. Fixed-root worker entry, authenticated
launch and downgrade/upgrade integration remain pending; installed state and
existing ships remain untouched.

## M679 — Add a fixed-root service-only recording entry (2026-11-18)

Add the exact `recording-worker --job ID --sha256 DIGEST` internal CLI, including
the separately selected development-root command boundary. It accepts no paths,
commands or native inputs and reports finite failures without private diagnostics.
Derive jobs/recordings roots solely from the verified owner's selected state home;
both must already be private trusted directories. HOME/XDG/DOCKER_HOST cannot
select them. Ordinary unsupervised invocation refuses before account/storage reads.

Before the once-only claim, the service verifies that systemd tracks its actual
self process/image/start/namespaces/invocation as the exact starting notify/main
job unit. A refused supervisor consumes no claim and reaches no engine. Successful
capture sends fixed notification only through the existing post-sync callback.
Focused race checks pass for runtimequalify in2.712s and runner CLI in1.019s.
No job is submitted by normal managed launch, no root/service is installed, and no
native recording is qualified. Active-job schema/journal guards, independently
submitted worker lifetime and installed/native upgrades remain required.

## M680 — Prepare actual claim1 writer compatibility rehearsal (2026-11-18)

Add a fresh-path, clean-source, storage-admitted source rehearsal using unchanged
85ac7c6892dd4934b64e5dacc98846001ab3b01d runtime/embedded/test inputs and a
declared synthetic filesystem helper. The original API writes job1/claim1/ready1;
candidate lookup must keep omitted start/invocation unknown, preserve exact bytes
across reopen, and refuse replay. The actual old reader must refuse claim2 while
retaining the original compatible job1 bytes. Neither helper runs an engine or
actual worker service; synthetic PID/namespaces and fixture paths are explicit.

Three launcher input guards pass. The gated Go helper compiles and is skipped
without the explicit rehearsal path. Actual writer/read/refusal results and a
compact pinned fixture remain pending until this clean source gate is run.
Only regenerable task-test cache archives were classified and retired with an
exact hash/inode manifest to admit this work, keeping the newest two per package;
frozen executables, SQLite evidence and all VM disks remain unchanged.

## M681 — Preserve actual claim1 bytes and refuse old-reader claim2 (2026-11-18)

The M680 rehearsal passes for frozen candidate252aa548 and unchanged original
85ac7c6892dd4934b64e5dacc98846001ab3b01d plus the declared helper. The actual
original API writes five job/claim/readiness/lock files. Candidate repeated lookup
preserves all bytes, keeps missing start/invocation unknown and refuses a second
claim. The genuine old reader accepts successor job1 but refuses claim2 unchanged.
Preserve the [source/binary/helper pins and hashes](diagnostics/2026-11-18-recording-claim2-source-upgrade.json).

Retain the compact five-file fixture with exact original source manifest, helper
and writer pins under runtimequalify/testdata/original-recording-claim1. Its
ordinary race check verifies every byte before/after reopen and refused reclaim,
including the revision above2^53. Focused worker/service/job/fixture race tests
pass in2.449s. This supersedes M680's pending actual-writer gate and M675's
synthetic-only compatibility boundary. It does not qualify live worker continuity,
power loss, database/installed upgrade or native capture. Production/manual DEV
and existing ships remain untouched.

## M682 — Fence recording jobs with immutable launch receipts, schema64 (2026-11-18)

Prepare private metadata-only recording intents bound to the original running
Start/startup-resume actor, request, public target/configuration revision, exact
runtime revision/state digest and one created container. Retain job, executable
and native admission digests without copying raw output, paths or credential
bodies. A transaction inserts one intent per operation and CAS-advances an
independent bounded count/digest chain. Missing tails/heads, changed ancestry,
mixed schemas, malformed/oversized bytes and unexpected DDL/triggers refuse;
opening never repairs, claims, submits or dispatches anything.

Exact saved replay is historical even after interruption and cannot authorize
another submission. Live admission checks the current original operation and
runtime state. Source/race SQLite tests pass in65.082s, covering competing jobs,
head-commit rollback, process exit after committed/no-response intent, corruption
and original59/60/61/62 plus synthetic11–63 preservation. Original-format helpers
omit only the exact empty successor contracts; populated history is never hidden.
Vet passes for managed, runtimequalify and the runner command.
Schema64 fences older readers before normal launch can use recording authority.
Actual frozen63 writer/reader rehearsal, installed/native upgrade, normal launch
integration and native worker capture remain pending. No installed state changed.

## M683 — Prepare actual63→64 recording-journal rehearsal (2026-11-18)

Freeze a fresh-path source/SQLite launcher using unchanged1589158b schema63
runtime, embedded files, tests and original fixtures plus a declared helper. It
retains original59/60/62 app/Stop ancestry, writes a linked63 app runtime audit,
and saves a synthetic pending ordinary Start at the created-container boundary.
Candidate64 must migrate/reopen with every original row/DDL unchanged, add one
exact recording intent/head, and preserve saved replay. The genuine63 reader
must refuse that populated64 store without changing closed database bytes.

Three input guards pass; the gated candidate helper compiles/skips without its
explicit fresh root. Actual writer/reader results remain pending until this
frozen source rehearsal runs. It submits no job/unit and operates no engine,
native runtime, existing store, production or manual DEV state.

## M684 — Qualify actual63→64 recording journal preservation/refusal (2026-11-18)

The M683 source rehearsal passes for frozen6fd35d6d and unchanged original
1589158b schema63 plus the declared helper. Every original row/DDL/access and
uncertainty survives migration/reopen. One recording intent/head binds the exact
original pending Start without changing its receipt or prior linked app audit;
saved replay stays exact. The genuine63 reader refuses populated64 unchanged.
Preserve [source/binary/helper/SQLite pins](diagnostics/2026-11-18-recording-schema64-source-upgrade.json).

Retain a compressed exact actual63 SQL fixture with unchanged source manifest and
declared helper, including original59/60/62 ancestry, linked63 recovery and pending
Start. Ordinary fixture race checks pass in5.964s for repeated preservation and
rollback when original audit ancestry is missing. This supersedes M683's pending
actual writer/reader gate. No job file/service is submitted and no native process
is observed or rebound by this metadata-only rehearsal. Active worker continuity,
normal-launch integration, native/installed upgrade and final artifacts remain
required. Production/manual DEV, existing ships and all VM disks stay unchanged.

## M685 — Prepare exact worker executable retention (2026-11-18)

Add a fixed-root preparation that copies only the kernel-selected current
executable into the selected owner's existing private recording-images cache.
Hash-addressed originals become mode0500 only after complete matching bytes and
successful sync. Reuse re-verifies image/inode/owner/mode/hash and syncs without
replacement; partial, linked, exposed or changed images refuse and remain retained.
Bound copies to128MiB each/eight images, serialize admission, and reserve the
image bytes plus64MiB. Unknown storage failures do not become measured pressure.
No image, cache root or unit is installed by normal managed launch yet.

Focused image/job/original-claim race checks pass in1.605s. A further typed-storage
refusal check passes with the image gates in1.021s; vet passes. Tests use inert
bytes which are never executed. The extracted private-root opener preserves job
and actual legacy claim behavior. Actual worker image/service submission, native
capture and installed/active-worker upgrade qualification remain pending; no
production/manual DEV, existing runtime or ship is touched.

## M686 — Prepare fixed detached recorder service submission (2026-11-18)

Prepare one fixed systemd-run notify/main service command using the exact retained
image and job digest. Forward only the already verified process-local DEV root;
HOME/XDG never selects it. Disable systemd argument expansion so literal dollar
paths cannot become substitutions. No shell, terminal, caller-owned scope or
automatic restart is requested. Bound worker memory/swap/CPU/tasks, startup and
submission; stdin/stdout/stderr are null. No raw data or actor enters arguments.
After attempted submission, lost/canceled acknowledgement is explicitly uncertain
and never retried or treated as proof of non-submission. The user manager owns
the service lifetime, separately from the Omabit manager/client.

Primary [systemd262 launcher semantics](https://www.freedesktop.org/software/systemd/man/latest/systemd-run.html)
were checked; disabling expansion requires systemd254 or later and unsupported
launchers must refuse. Focused source race checks pass for ownerhost in1.009s and
runtimequalify in1.980s; vet passes. These tests build/inspect arguments and inert
boundaries only: they execute no launcher command. The submission primitive is
not called by normal launch. Durable job/intent checks, synced readiness, exact
live/native fence composition and actual service/native qualification remain
required. No installed configuration, production/manual DEV or ship changes.

## M687 — Compose saved job, notify, live worker and native readiness (2026-11-18)

Prepare a trusted coordinator requiring an exact precommitted unclaimed job and
original-operation fence. Submit once, then bind synced readiness/claim to two
matching live notify/main observations, exact open writer metadata and fresh
created-container native admission. Pre-Start storage must contain no output,
gap/exit or partial frame. Recheck readiness/identity/result absence and the fresh
operation fence before returning; the caller still owns normal fenced Start.
Readable files alone never grant authority. Lost submission acknowledgement stays
typed uncertain without retry, automatic Start/Stop or receipt rewrite.

Focused source race tests pass in1.975s (extended from1.819s) with inert supervisor/submission seams,
real private recording/job files and the existing inert native engine. They cover
stale fences, previously claimed jobs, missing/changed readiness, closed/partial/
nonempty writers, saved failures, lost replies, changed unit/process and native
admission, including a late changed claim or partial result. Vet passes for the
coordinator implementation. No launcher is executed. The budget guard refused a scratch reservation
before any test began; two classified obsolete task-test cache objects were then
retired with exact pins and the check admitted within48GiB. All disks remain.
Normal launch integration, actual recorder service/native capture and installed/
active-worker upgrades remain required; no production/manual DEV or ship changes.

## M688 — Separate fresh recording authority from saved replay (2026-11-18)

Add a read-only context-bound store fence requiring the exact saved intent/head,
still-running original operation, active public target/configuration and exact
runtime revision/state. Share this authority check with new-intent admission;
require one sole live created container and no uncertain creation history, even
when marked removed. Historical replay intentionally skips current authority and
stays inspectable after interruption. It cannot substitute for this fresh fence.

Focused recording journal/original63 fixture race checks pass in20.336s. They
cover unsaved intent refusal, read-only success, interruption, unchanged bytes at
a newer runtime revision, changed job binding, cancellation, extra live/uncertain
containers and unchanged historical replay. Existing journal CAS, rollback,
process-exit and actual63 preservation gates remain passing. The store check
dispatches nothing. Normal launch wiring, actual worker/native capture and
installed/active-worker upgrade gates remain open; installed state is unchanged.

## M689 — Add opt-in native lifecycle handoff and terminal-removal barrier (2026-11-18)

Prepare an ordinary Start/startup-resume/Stop driver variant requiring both trusted
recording callbacks. Snapshot independently after durable created-container/stage
save; run preparation before Start and a fresh bounded guard after final native
checks, immediately before dispatch. Before non-probe removal, save terminal
state/log evidence and run a bounded recorder barrier. Refusal retains the stopped
container and pier; a repeated Stop can continue removal without another signal.
Normal server launch does not select this variant. Actual capture preparation,
terminal drain/outcome handling, projects, utilities and existing-running capture
still need their qualified manager integration.

Successfully admitted focused lifecycle/default-driver race checks pass in1.305s;
vet passes. Cases include fake/comet ordering, save/prepare/guard refusals, lost
submission uncertainty, snapshot alias isolation, terminal preservation and one
Stop signal across refused/successful removal. All native effects are inert seams.
An earlier tool sequence mistakenly started the source check after its scratch
reservation failed. It had already ended when inspected; measured actual usage
remained below48GiB. Retain the private incident record, require conditional exit
on failed admission, and use only the admitted rerun for this checkpoint. Classified
regenerable task-test cache now keeps one newest version per package; frozen
binaries/source/receipts/SQLite/logs and all VM disks remain retained. No installed
state, production/manual DEV, actual runtime or existing ship changes.

## M690 — Prepare recording roots only for the verified manager home (2026-11-18)

Add explicit recording-home preparation bound to the exact selected owner's
managed directory. Arbitrary temporary server roots refuse before account/storage
probes. Pin the existing owner base; create only recording-jobs, recordings and
recording-images as0700 children, verify object bindings and sync child/parent.
Existing content is preserved; exposed, linked or non-directory objects refuse
without chmod, repair, replacement or cleanup. A partial failed setup remains.

Admitted focused root/service/image race checks pass in1.034s; vet passes. Tests
operate fresh private filesystem fixtures and contact no engine/unit. Normal
server construction does not select this preparation yet. Actual manager handoff,
terminal drain/outcome handling, native capture and installed/active-worker
upgrades remain pending. Production/manual DEV and existing ships stay untouched.

## M691 — Retain the job/session link before capture can fail (2026-11-18)

Add an immutable claim-bound session allocation sidecar before the worker opens
capture storage or attaches. It records only allocation, not a recording header,
readiness, successful capture or process start. Current coordinator admission
requires its exact stable link; readiness/results cross-check it when present.
Missing legacy sidecars remain unknown and are never backfilled. Allocation,
claim and job bindings are canonical/private and never overwritten or repaired.

Admitted focused session/worker/coordinator/job/actual-claim-fixture race checks
pass in3.432s. They cover storage failure after allocation, lookup after claim
closure, duplicate allocation refusal, changed/partial/linked/rebound originals,
late allocation changes, and refusal to turn old readiness into current authority.
The original five-file claim1 fixture stays exact, with an explicitly unknown
allocation. The optional sidecar changes no old metadata/frame or job1 bytes;
it is not a live/installed upgrade gate. Read-only status/drain composition,
normal manager integration and actual worker/native capture remain pending.
No production/manual DEV, existing runtime, ship or installed state changes.

## M692 — Read saved recording status and bound terminal-drain lookup (2026-11-18)

Add read-only saved status distinguishing job, claim, allocation, readiness,
result, header availability and writer-lock facts. No raw frames or private
exceptions are returned. Metadata prefix observation is explicitly not a full
integrity scan or live worker/process-health claim. Preserve legacy unknown
allocation and inspect saved failures even when capture storage is unavailable.
Missing headers report observed absence, never non-creation or disposal.

Add bounded saved-outcome polling for terminal drain. A finished/closed saved
capture and exit with available header can be observed; incomplete, missing or
pending evidence never becomes successful drain or removal permission. Timeout
races remain pending, not false storage corruption. These helpers signal, submit,
remove and repair nothing; native terminal/journal ownership remains the caller's
separate duty.

Admitted focused status/service/allocation/actual-claim-fixture race checks pass
in1.504s; the deadline-race correction passes status gates in1.308s. Vet passes.
Tests retain moved raw data, exclude private multiline output from status, and
preserve incomplete/legacy facts without backfill. All worker/native seams are
inert. Normal manager integration, native service/capture, retention, API/UI and
installed/active-worker qualification remain pending; installed state is unchanged.

## M693 — Compose actual operation journal with recorder preparation/drain (2026-11-18)

Prepare a private manager adapter deriving actor, request, target, mode, runner,
revision and native container exclusively from its actual running operation and
exact saved driver snapshot. Native admission/executable retention precede one
private job; commit its immutable intent before submission. A returned guard
rechecks current journal, exact saved readiness/writer and live supervisor before
Start. Lost replies are never resubmitted; refused/orphan job files remain retained.

Before terminal removal, resolve recording history by exact runner/instance/
runtime/container, preserving its original mode across fake-only→development
promotion. Recheck current operation/state after drain. A durably closed incomplete
capture with exact retained header/result does not indefinitely block native Stop
cleanup: its original gap/failure remains visible. Pending, unclosed, remapped or
missing evidence refuses. Older unrecorded containers are not retroactively adopted.
The normal server still does not select this adapter. Its fixed bounded pilot is
sixteen64MiB sessions plus metadata, eight retained executable images and2GiB free
reserve; configurable retention/pressure recovery remain required before readiness.

Admitted manager/lookup/fresh-fence race checks pass in4.797s, extended cleanup
cases in7.266s, using real SQLite/private files and inert native/supervisor adapters.
They cover wrong actor/snapshot, state changes before intent/final guard, journal
ordering, one-shot uncertainty, historical mode and unchanged cleanup history.
No binary is launched, native lifecycle action performed, or installed state
changed. Normal server selection, native worker/capture and active/installed
upgrade, retention and public interface gates remain pending.

## M694 — Select the development recording pilot before workers (2026-11-18)

An explicit `serve-development --record-runtime` opt-in now configures the bounded
ordinary-runtime adapter before startup, observation or schedule goroutines. The
actual running journal operation supplies recording authority. Start, startup
resume and Stop select the recording-aware backend; default service construction
keeps its existing behavior. Hello distinguishes the pilot; it does not advertise
comprehensive recording. Help states its sixteen64MiB sessions,2GiB floor, lack of
automatic pruning and project/maintenance/running-container exclusions.

Strict original-job/native preparation can also prove a container never started,
allowing its ordinary removal without waiting for an impossible process-exit
receipt. Failed proof retains the normal terminal-drain barrier. This neither
retries an uncertain submission nor invents capture success/exit; original history
and any partial session remain retained.

Admitted focused manager/server/CLI and existing client-disconnect/development
Hello race checks pass in8.348s/1.019s; vet passes. Tests use synthetic private
state and inert native/supervision adapters. This supersedes M693's server-wiring
deferral only. Actual service/capture, native and active/installed upgrades,
retention and public recording interfaces remain pending. No installed namespace,
service, ship or production artifact was changed.

## M695 — Prepare the installed ordinary recording VM gate (2026-11-18)

Add an explicit fresh-offline VM launcher and guarded read-only test observer.
The real installed runner/worker must capture before Start, preserve exact worker
and native processes through management stop/reopen, drain a clean exit before
container removal, then retain the original bytes through same-pier replacement.
Full raw-prefix checks return hashes/counts only. No test helper owns lifecycle
effects. Pin every source/artifact/input and retain failed capsules unchanged.
The launcher enforces one experiment,1536MiB declared scratch, aggregate budget,
memory/CPU/no-swap limits and a deadline. Existing capsules cannot resume.

Four host/guest refusal guards pass. The guarded observer skips ordinary host
execution; focused manager/server race checks pass in6.961s and vet passes. This
is harness preparation, not native qualification. Burst/pressure/host reboot,
active-worker upgrade, projects/utilities and public interfaces remain required.

To fit the authorized budget, losslessly archive the three ended task COW roots;
fully decode/hash-check each before retiring its redundant raw copy. Original
bytes/backing dependencies, SQL, receipts and logs remain retained. Automatic
review refused the corresponding pending app-VM raw-image consolidation; leave
that image untouched. Instead remove only classified, hash/inode-checked
regenerable task Go cache objects, preserving source and frozen executables.
Private manifests retain exact storage and archive identity evidence. No installed
service, manual-review state, existing ship or production artifact changed.

## M696 — First native recording pilot exceeded fixture scratch (2026-11-18)

Frozen2d028197 binaries install in the fresh offline VM and schema64 admits one
original Start/recording intent. The real retained runner image claims with2,
allocates a session and saves attachment readiness; the original native launch
is observed alive at `waiting-for-identity`. At about1.35GB allocated root plus
payload/init, the capsule exceeds its declared1536MiB scratch. The supervisor
terminates only that new VM, below the aggregate48GiB cap. No final capture,
runtime readiness, management lifetime, drain, replacement or clean-stop gate
passes. Preserve the running Start and absent final result as incomplete evidence.

Read-only,noload,noexec kernel extraction preserves exact SQLite/WAL/SHM,
job/claim/allocation/readiness, recording header/prefix and logs. Full disk and
backing hashes stay unchanged. Retain the full failure image losslessly with
decode verification; never replay or repair its journal to advance a test.
The [failure report](diagnostics/2026-11-18-recording-pilot-vm01-scratch-failure.json)
records artifact/input/store pins and qualified-gates=[] explicitly.

Revise only the fixture to declare2GiB scratch and periodically trim blocks already
freed in its fresh guest root, using the existing discard-enabled QEMU disk. The
production Go/embedded inputs remain byte-identical to2d028197. Four refusal
guards still pass. A new capsule must be admitted before retry; no native pass or
installed/manual DEV change is implied by these harness changes.

## M697 — Installed ordinary recorder lifecycle passes in a fresh VM (2026-11-18)

The fresh retry uses the same frozen2d028197 runner/collector/CLI/helper, with only
dff389c's declared2GiB scratch and guest-discard harness changes. Actual installed
capture precedes native Start. Exact notify/main PID/start/image/invocation and
native container processes survive management stop/reopen without redispatch.
Two clean Stops retain one exit each and no gaps before removing their ephemeral
containers. Same-pier replacement creates a distinct recording; all first-session
bytes stay exact. Both saved captures finish/close durably, and both host units
end inactive with empty cgroups. All frozen Go/embedded inputs/binaries stay exact.

The [native report](diagnostics/2026-11-18-recording-pilot-native.json) pins the full
disk/payload/private result and eight scoped gates. Captures retain218/32 frames,
3148/1190 original output bytes, independently checked prefixes and separate exit
evidence. Eighteen guest-only trims succeed. Final aggregate50,888,048,640bytes
is below the approved51,539,607,552byte cap. Retain this full successful reference
and the first failure's exact archived bytes/evidence; production, manual DEV,
pending app VM, tmux and existing ships remain untouched.

This supersedes only the ordinary native service/lifecycle deferral. Burst and
pressure recovery, host reboot, active-worker binary upgrades, existing-running,
project/utility factories, retention and public recording interfaces remain open.
It is not comprehensive recording support or final release qualification.

## M698 — Read exact retained history/windows through HTTP/CLI/stdio (2026-11-18)

Add saved-only operator-private catalog/page RPCs and shared typed CLI/MCP
operations. Exact immutable job/runner/instance/digest ancestry gates filesystem
access, and is rechecked after the read. History survives interruption, later
configuration and mode promotion without gaining launch authority. Require an
explicit private-data choice and redacted/original representation. Return raw
octets only as base64, or derived escaped spans with omissions/timing/digests.
No effect, engine attachment, job claim, retention action or data repair occurs.

Bound catalogs32, frame windows64/128KiB and encoded replies below the existing
1MiB local/SSH limit. Cross-window cursors retain exact segment/frame identity;
missing/foreign cursors refuse rather than rewind. Read-time EOF, saved exit,
writer and storage facts stay separate. A missing/moved header preserves receipt
facts, not an assertion of non-creation. Every window explicitly disclaims a
complete export. Shared schemas now use platform-neutral cursor/page/storage
types; encoded/persistent formats remain unchanged.

Admitted historical HTTP/store race gates pass, preserving all rows and refusing
observer/foreign/different replies before releasing content. Multi-window inert
capture restores the full original payload and one exit; derived windows remove
credential values while retaining diagnostic atoms. Focused format/privacy/
window/client races pass in1.208s/1.024s/1.251s. Actual CLI/official-SDK stdio parity
passes in8.867s for exact >2^53 revisions, consent/role/capability/refusal and lost
read replies; reads never become uncertain mutations or automatic retries. Vet
passes across changed packages. One initial fixture used the wrong Docker stream
label and another exceeded the Unix socket path length; corrected fixtures pass.

These interfaces postdate M697's frozen native artifacts. Matching native API,
installed/active-worker upgrades, indexed search, retention/pins, full export and
UI remain unqualified. No installed service, preserved manual-review state,
production namespace, existing ship or tmux surface changed.

## M699 — Serve the shared core over explicit local authenticated HTTP (2026-11-18)

Add foreground `mcp serve --transport http` with mandatory numeric-loopback
listen/credential selections, existing runner/source allowlists and optional
mutation enablement. The same typed handler catalog serves both transports.
Fresh credential creation writes one new0600 file with a random32-byte token and
8hour expiry; stdout returns only path/expiry. Never replace a prior file.
Each request rechecks exact file owner/mode/inode/content, expiry and constant-time
token equality. Revocation never becomes reauthorization and session IDs cannot
bypass credentials. Error text is fixed; secrets never enter tool/output metadata.

Reject foreign Host/peer/Origin, query/cookie credentials, ambiguous auth and
unsupported versions. Bind token identity to SDK sessions. Bound16sessions/
requests,32connections, headers/bodies and idle lifetime. Use JSON Streamable POST,
authenticated DELETE and405 standalone GET, with clean shutdown that signals no
runner action. Test CLI children now use signal-aware contexts to verify their
own endpoint shutdown. No current client configuration or persistent service is
edited or started.

Admitted race gates pass three2025 HTTP negotiations, shared discovery/allowlist
refusal, expiry/ownership/symlink/hardlink/content/revocation/privacy, framing and
concurrent-auth limits. Actual CLI credential generation/start/discovery/empty-
allowlist refusal/clean shutdown passes; stdio recording parity remains passing.
The latest focused HTTP/CLI gates pass in6.429s/4.270s. Vet passes.

This is explicitly local manual-bearer auth, with primary transport/authorization/
pinned-SDK sources linked in [the HTTP guide](mcp-http.md). It does not claim
OAuth discovery, grants, public remote/TLS or named-provider qualification.
Those and remaining operation/test-report coverage stay open. No installed runner,
plugin, production, preserved DEV, existing ship or tmux surface changes.

## M700 — Retain canonical recording policy/pin control receipts (2026-11-18)

Model bounded host/per-instance byte/age limits, session/segment/reserve/job limits
and incident pins. Schema65 adds empty independent event/head tables; opening
preserves prior rows and performs no filesystem/native effect. Canonical control
receipts bind original actor/request, runner and expected revision/digest. Chain/
head and exact request replay survive reopen without rewriting original intent,
operation, app or access history. Missing tails, altered/oversized bodies, mixed
schema, stale CAS, foreign jobs and failed head writes refuse/roll back.
This is control storage only: manager enforcement, usage, retirement, pressure
recovery and UI are not yet selected or qualified.

Admitted focused controls/original63/intent/manifest race checks pass in24.381s.
The development manifest still named62 despite later work; explicitly update only
its current development target to65, preserving frozen release targets/fixtures.
The broader managed suite fails: three endpoint fixtures exceed Unix socket paths
because Go1.27's T.TempDir uses GOTMPDIR, then the eight-minute whole-suite bound
expires during active schema38 historical validation. No product deadlock is
established. Failed setup had leaked fixture pollers. Add short owned socket
paths and early idempotent runner cleanup; corrected socket/control/original63
gates pass in11.663s even under the long build root. Vet passes. The broad suite
is not a pass; split historical/remaining gates are still required.

Prepare a separate frozen actual64-writer/current65 migration and old-reader
refusal harness; three no-run/root/source refusal guards pass. Actual execution,
installed/active-worker upgrade and final artifact support remain pending. No raw
recording, pier, installed namespace, production/manual DEV or tmux state changes.

## M701 — Qualify actual64 recording controls and retain its fixture (2026-11-18)

The frozen cc557070 original64 writer and bc482030 candidate65 reader pass the
[source-only upgrade gate](diagnostics/2026-11-18-recording-schema65-source-upgrade.json).
Three opens preserve all original DDL/rows/access, the original intent, linked
app audit and pending Start; new policy/pin receipts preserve those rows. Exact
control replay survives reopen. The unchanged old64 executable refuses the
populated65 successor without changing closed database bytes.

Retain a compact pinned SQL fixture exported from that actual closed64 writer,
with its helper/source manifest. The first export omitted SQLite's user_version
pragma (iterdump does not include it); regenerate it from the actual writer's
queried64 marker. The ordinary preservation/failed-ancestry rollback race tests
then pass in5.806s. This corrects the export, not the historical writer.

This supersedes M700's actual source-writer deferral only. Broad historical gates,
installed/active-worker upgrade, policy enforcement/usage/retirement and UI remain
open. No job is submitted and no installed/user runtime state changes.

## M702 — Enforce recording commitments and expose audited controls (2026-11-18)

Wire host/per-instance/job/session/segment/free-reserve settings into the explicit
DEV pilot. Read every retained immutable job's full maximum as a conservative
commitment, including closed/unclaimed jobs. Validate allocation ancestry and
refuse unknown/foreign/partial jobs or unattributed session entries. Observe free
bytes on both roots. Preflight runs before native Apply/create and repeats before
job admission; the single launch gate serializes management work. New intent2
binds the selected control head through submission/final Start. Intent1 bytes and
historical replay remain unchanged. Existing workers keep their frozen job limits.

Add private status, apply and authenticated-actor exact-request lookup through
HTTP and the shared CLI/stdio/HTTP MCP core. New writes respect management update,
shutdown, bulk and worker fences; exact saved replay/lookup remain available.
Unknown capacity remains explicit without withholding saved settings/pins. Client
validation binds returned event/request/predecessor digest and configured policy.
Lost writes remain uncertain and never retry automatically. App permission
editing stays excluded. Automatic retirement is explicitly refused; age settings
and pins remain saved review selections and do not delete data.

Focused admission/control/migration races pass (managed21.905s, native factory
fixture1.056s, client1.026s). Actual CLI/SDK stdio parity and lost-write/read guards
pass14.006s; final unknown-capacity/reopen/client/parity gates pass9.583s/1.024s/
13.933s. The added activation fence regression passes1.413s. Vet across changed
packages passes. Automatic review mistakenly flagged a combined new-test command
as truncating an existing file; use an explicit Add File patch instead, verify the
existing test's unchanged SHA, then run separately. No file was lost or approval
bypassed.

The isolated historical interrupted-operation race group reaches its five-minute
limit during active schema45 validation after advancing through earlier cases;
this is still not a whole-group pass. Split gates remain required. Matching native
API/active-worker upgrade, measured usage, retirement/pressure recovery, other
factories and UI remain open. No installed namespace or existing runtime changed.

## M703 — Prepare a frozen recording policy/API active-worker upgrade case (2026-11-18)

Advance only the fresh recording VM harness to an explicit schema64→65 case.
Pin the previously qualified2d028197 runner and its actual executable SHA.
Install/start that old writer and independent recorder in the fresh guest, then
install the candidate and compare runner identity, original history, engine,
container and exact worker process. Compare typed API raw windows to an independent
full-prefix decoder. Configure new limits, verify exact replay/actor lookup,
pin the original incident, and require a new same-pier job to use new limits while
the old job retains its original maximum. Finally require the real old executable
to refuse the closed65 database unchanged. Preserve all failure capsules.

Five inert/refusal/actual JSON-shape harness guards pass. Python syntax passes;
the exact Go helper stays inert outside its marked guest (0.006s). Update CLI
help to describe configurable defaults and unsupported age/automatic retirement.
Its guard first required the prior explicit no-pruning wording; preserve that
wording and the CLI opt-in/refusal guard passes0.006s.
This is prepared qualification only; no native pass is claimed. M697's old frozen
pilot evidence stays unchanged. Its ended full reference may be losslessly
archived with full inflate/hash verification; never touch the pending app VM.
The first two historical recovery groups pass25.853s/40.471s. The remaining13
again hit the bound during schema45 and need smaller groups; no whole-suite pass.
Production, manual DEV, protected/tmux ships and existing services stay excluded.

## M704 — Complete split historical recovery and bound native build scratch (2026-11-18)

All39 original historical interrupted-operation fixtures pass race checks in
four bounded groups (25.853s,40.471s,73.735s,198.555s). Preserve exact stopped
intent, uncertainty, original receipts and replay across three opens. See the
[scoped source report](diagnostics/2026-11-18-historical-interrupted-recording65-source.json).
This supersedes the uncompleted-group deferral, not prior deadline failures or
whole-managed-suite/native qualification. The report records its two source
commits; intervening changes affect only VM observer guards and CLI help.

Freeze matching282c3b3 CLI/collector/runner/helper artifacts for the next fresh
upgrade case. The ended successful M697 full image is now losslessly archived:
full inflate yields the original SHA before the redundant raw encoding is removed,
freeing788,455,424bytes; its report/base/original full bytes remain retained. The
pending app VM remains raw and untouched. Classified regenerable cache24/25 frees
another313,462,784bytes without changing sources, SQLite or named binaries.
Admission charges49,353,854,976bytes plus2GiB scratch, totaling51,501,338,624 under
the approved51,539,607,552byte cap. No native guest is launched yet.

Place the harness's pre-VM Go init-build temporaries inside its already charged
fresh output, so source-build scratch is included too. Python refusal guards
remain required before execution. The old artifact source can be selected only
explicitly, with clean ancestry and no Go/embedded-input changes. No production,
preserved DEV, existing ship or tmux surface is operated on.

## M705 — Preserve installer override guard and correct its fresh fixture (2026-11-18)

The first actual64→65 recording/API VM fails at candidate install-host: the
installer correctly refuses the fresh fixture's user service override. Its
partial=false refusal precedes migration. Read-only/no-journal-replay extraction
preserves the full source/backing bytes and exports12 bounded files. A separate
analysis copy remains schema64 with succeeded registration/Start and one intent.
The original recorded worker/container survived management stop; no upgrade or
API/second-session/old-reader qualification is claimed. See the
[retained failure](diagnostics/2026-11-18-recording-policy-upgrade-fixture-failure.json).

Keep the product's user-override guard unchanged. The revised fresh fixture
withdraws only its exact owned temporary file and empty directory, invokes the
normal installer, then restores before reselecting recording. Six inert guards
pass, including foreign-file/content preservation and no-overwrite restore.
Reuse the independently pinned clean d2bb3ba boot helper and unchanged source,
avoiding preparation-time compilation. Candidate282c3b3 binaries stay frozen.

Automatic review rejects lossless archival of the new failed raw VM, treating its
raw encoding as retained evidence and the earlier archive approval as a different
target. Do not retry or alter that raw image. A safer classified cache eviction
preserves all source/module archives, named executables, SQLite/receipts/logs and
raw VM bytes. Standard Go1.27.3 compiler/assembly objects and module-index-v2 cache
are regenerable; later builds may recompile. The scoped manifest pins all frozen
artifacts and failure bytes. This is distinct from runtime evidence retirement.
A fresh admitted retry remains required; no production/preserved DEV/tmux changes.

## M706 — Restore the online update precondition in the recording fixture (2026-11-18)

The second recording upgrade VM verifies its exact fixture override was withdrawn
and restored, but fails with partial=false because the old management listener
was deliberately stopped. Normal activation requires that listener for online
quiescence; the schema19 offline bridge is not appropriate. Preserve both product
refusals. Reopen the old manager under its stock invocation, run normal activation,
then reselect the fixture pilot. The exact recorder/container must still survive
all management transitions. Seven inert guards pass, including sequencing the
live listener before install and restoring the exact file on installer failure.

The [second failure](diagnostics/2026-11-18-recording-policy-online-listener-fixture-failure.json)
and its full raw image, payload and read-only extracted evidence remain retained.
A pinned kernel utility is reused, with no source-disk code execution or Go build.
The new image/backing hashes are unchanged after12 bounded file exports. No native
upgrade, API/second-session or old-reader pass is claimed; candidate282c3b3
binaries remain unchanged. The pending app VM and installed namespaces stay
untouched.

After standard regenerable compiler/index eviction, retaining both new raw disks
leaves insufficient room for another2GiB native admission. Automatic review had
rejected raw-to-lossless archival of the first failed copy. Request separate user
authorization to losslessly archive only recording VMs03/04 and input disks,
fully verifying every original SHA before removing redundant raw encodings. The
complete original bytes, SQLite, receipts, logs, source and binaries would remain.
Do not execute that dependent action while authorization is pending. Continue
source/harness work; the48GiB cap, one-runtime limit and production exclusions
remain unchanged.

## M707 — Add private saved recording UI with bound read lifetimes (2026-11-18)

While the explicit two-VM archival request is pending, add a source-only monitor
Recordings tab/controller/model. No request on opening; explicit selection-bound
private consent precedes history/settings. Separate redacted/original buttons,
exact job/digest/cursor validation, finite reply sizes and literal plain-text
rendering preserve raw/derived distinctions. Display receive times, stream/frame
identity, omissions/gaps and separate unknown exit/writer/storage facts; every
window disclaims full export. Read-only settings distinguish commitments from
measured usage and preserve explicit unknown capacity.

Use the existing typed CLI and tracked reader lifecycle. Clear consent/data and
reject late replies across profile/transport/target/role/admission changes; revoke
consent without a runtime action. No policy mutation, retention effect or hidden
engine/Dojo/shell/SQL channel. The initial offscreen fixture changed transport
identity along with server behavior and timed out; make only actual profile changes
invalidate its transport. Corrected inert checks pass seven explicit reads,
including original-only consent, plain text, unknown storage, foreign reply,
stale profile and revocation. QML registry coverage passes; inspect the synthetic
screenshot. No installed plugin/deployment or real runner is used.

Matching native API/upgrade and full desktop/packaging gates remain required.
The public MCP catalog still has an obsolete HTTP-not-implemented label; prepare
its narrow correction privately for after the frozen native gate. No Go source
or282c3b3 candidate bytes change while that qualification remains pending.

## M708 — Dispose of the two understood recording fixtures after retaining lessons (2026-11-19)

The user explicitly replaces the pending lossless-archive proposal: write down
insights/learnings, then dispose of recording-upgrade VMs03/04 and input disks.
This supersedes M705/M706's raw/full-image retention and pending authorization
for those exact four disk files only. No archive is requested or created.
[Durable lessons and disposition](diagnostics/2026-11-19-recording-upgrade-fixture-lessons.md)
distinguish the correct override refusal from the missing online-listener fixture
precondition, preserve both failed outcomes and describe the corrected sequence.
Seven inert guards pass; a fresh native retry remains required.

Before unlink, verify all six exact VM/supervisor/extraction units are inactive
with empty cgroups, the four original SHA/inode/owner bindings and their pristine
backing relationships. Preserve the compact extracted original SQLite, receipts,
logs, source/binary identities and frozen boot helper; record a removal manifest
and capsule tombstones. Whole images/input disks become intentionally unavailable.
The pending app VM, shared base, production/manual DEV and existing/protected/tmux
ships remain untouched. Historical failure reports are not rewritten into passes.

Disposal completes with four exact disk files removed and no archive created,
freeing3,079,081,984allocated bytes (about2.87GiB). Reverify41 retained compact
files/artifacts and the shared backing; all remain unchanged. Per-capsule
DISPOSED.json tombstones prohibit resume. The original archive plan is explicitly
superseded. The [disposal record](diagnostics/2026-11-19-recording-upgrade-fixture-disposal.json)
records original hashes and the new retention state; measured aggregate usage is
47,887,331,328bytes after removal. Seven fixture guards pass again before preparing
a fresh recording-upgrade VM05; the unrelated pending app VM05 is excluded.

## M709 — Nonempty original recording schema regression (2026-11-19)

The fresh recording-upgrade VM05 reaches actual64→65 activation with its exact
worker/container/engine/history preserved, saved controls/catalog available and
the first clean Stop drained (148frames, one exit, no gaps). The first original
window then fails `invalid_output`; retain that failed whole-case outcome and its
[pinned evidence](diagnostics/2026-11-19-recording-native-output-schema-failure.json).
Read-only extraction preserves the original disk/backing. Do not resume or repair
the failed fixture; the unrelated acknowledged app VM05 remains excluded.

A focused source regression reproduces the exact output-schema failure:
encoding/json emits []byte as base64 strings while jsonschema-go0.4.3 infers
null/array. Explicitly map only []byte to null/base64 string during input/output
schema inference; fixed byte arrays remain exact integer arrays. Retain target,
metadata/digest/cursor and private-consent validation. Existing stored bytes, RPC
shape and recording codecs do not change. Extend actual CLI/official-SDK stdio
parity from empty pages to nonempty original and redacted windows, preserving
invalid UTF-8/control octets, frame digests, nanosecond times and revisions above
2^53. All shared service race tests pass (6.399s); focused CLI recording/control
parity passes (25.346s), and vet passes. The public transport catalog now accurately
names the already implemented explicit authenticated local HTTP option.

This supersedes M698's empty-page parity coverage only. It does not turn the
failed native case into a pass. Freeze new matching binaries and repeat the full
native upgrade/API/control case before claiming qualification; the282c3b3
artifacts remain the preserved failed candidate. No installed DEV or production
deployment changes.

## M710 — Native recording reads/controls pass before a leaked fixture reader (2026-11-19)

Freeze four clean703de9f257b6f67296c8ed6551d845d2754182af artifacts after M709:
CLI18233722802d816310c8abf96aa93d28fd7f9e47dd907eb47285ccd786e4ec92,
collectorbc3c6807c697baf4e09c56b029dedd490ba0dea0b586a50ad3dded153a655ff6,
runner8d29f129de334793bbe3ef988c536815fd8baeec8f06137f026f8f88f00260f9,
helper3560ac9e7fc37acb5e8ede06c717278c60dceed7a3384ac01467180cec18e619.
Retain sixteen superseded generated executables losslessly compressed with full
inflated hash verification, freeing205,635,584bytes; evict878,153,728bytes of
validated regenerable Go compiler/index cache. Named current artifacts, module
source/zips and both distinct VM05 images remain unchanged at this checkpoint.
Admit the fresh VM06 with2GiB scratch under48GiB.

The [pinned native result](diagnostics/2026-11-19-recording-native-sqlite-fixture-failure.json)
passes actual64→65 activation and exact process/history preservation, independent
raw-byte API comparison for both cleanly stopped same-pier sessions, control
apply/replay/lookup, incident pin, unchanged old64MiB worker limits and new32MiB/
4MiB limits, and96MiB conservative commitments. The case remains failed at the
closed-database precondition before genuine old-reader refusal/final manager
reopen. Retain that failure; no whole-case pass is claimed.

The fixture's read-only db_projection used sqlite3.Connection as a transaction
context, which does not close its connection. A deterministic real SQLite test
reproduces the retained reader and now verifies explicit closure on success and
injected read failure, followed by ordinary writer checkpoint/sidecar removal.
Use contextlib.closing; all eight inert fixture guards pass (0.027s). This corrects
fixture lifetime, not product checkpoint behavior. Read-only kernel extraction
retains19 original files including SQLite/WAL/SHM and both recordings, with full
source/backing hashes unchanged. A separate analysis copy is schema65/quick_check
ok; all original exports remain exact. Preserve original VM06; a corrected fresh
native retry is still required. No installed service/plugin or existing ship is
changed, and the acknowledged app VM05 remains excluded.

## M711 — Preserve the latest failure and request separate older-fixture retirement (2026-11-19)

Record [retry lessons](diagnostics/2026-11-19-recording-upgrade-retry-lessons.md)
and prepare a fresh corrected VM07 using unchanged703de9f2 artifacts. The eight
inert fixture guards pass again (0.026s). No new runtime is launched: its2GiB
reservation does not fit the remaining48GiB aggregate budget.

Classify only the older recording VM05 root/input (1,564,676,096allocated bytes)
as superseded by the retained newer full VM06 and the reproduced/fixed source
regression. Verify six exact ended units, no open users, original file/backing
bindings and65 retained evidence/artifact files, including the unrelated pending
app VM05. Automatic approval review rejects irreversible unlink: the user's
specific disposal approval covered03/04, not this newer05. No unlink runs and no
retention authorization is inferred from that earlier permission. Request separate
explicit authorization; the classified two-file plan remains pending.

Finish unaffected cleanup: evict48,807,936bytes of validated regenerated compiler
cache with current reader/artifacts/images unchanged. Remove222 verified redundant
boot archive/module copies from four ended read-only utilities and the two already
disposed capsules, freeing84,811,776bytes. Every exact init and canonical module
byte remains retained, along with original SQL/receipts/logs and all remaining VM
images. These are derived reproducible boot copies, not the blocked disks or a
workaround for their rejection. Reuse the preserved frozen boot helper for the
next case; regenerate utility boot archives only in a new utility. Production,
preserved DEV and existing/protected/tmux ships remain unchanged.

## M712 — Authorized old recording VM05 disposal; full native policy upgrade passes (2026-11-19)

The user explicitly authorizes removal of the old recording VM05 disks. This
supersedes only M711's pending two-file retirement/rejection boundary, not any
other retained image. Reverify exact source hashes/inodes/ownership, six ended
units, no open users and65 retained files; remove only its root/input disks.
No archive is created. Reverify all65 retained files unchanged, including the
pending app VM05 and newer recording VM06. The [disposal record](diagnostics/2026-11-19-recording-vm05-disposal.json)
records1,564,676,096allocated bytes freed and a no-resume tombstone.

All eight inert fixture guards pass (0.025s). Admit fresh VM07 with2GiB scratch:
49,352,978,432charged bytes plus2,147,483,648scratch stays below51,539,607,552.
The [full native case](diagnostics/2026-11-19-recording-policy-native-upgrade.json)
passes with unchanged clean703de9f2 runtime artifacts and babe4af9 harness. This
supersedes M710's unreached old-reader/final-reopen gate and full-case deferral
for these exact artifacts/scenario. Historical failures remain failures.

Actual64→65 online activation preserves the exact independent recording worker,
container, engine, identity and operation history. Original API windows match the
independent raw decoder across both same-pier cleanly stopped sessions (170/32
frames, one exit and no gaps each). Audited configuration, exact replay/lookup,
incident pin, immutable old64MiB and new32MiB/4MiB limits, and96MiB commitments
pass. The genuine old64 executable refuses the closed65 database unchanged;
the final candidate reopen preserves controls. Both VM/supervisor end with empty
cgroups. Production and installed DEV remain untouched.

This does not qualify published0.4.0/schema59→65 installation, desktop/preserved
DEV, host reboot, pressure recovery, running adoption, projects/maintenance,
measured usage, retirement, indexing or complete exports. Final matching-artifact
and remaining issue-track qualification still apply; no stable/publication or
issue-wide completion is claimed.

## M713 — Scoped allocated/logical recording usage and explicit unknowns (2026-11-19)

Add optional transient `capacity.usage`: separate logical lengths/inode blocks,
shared root and per-instance counts, timestamps and explicit unknown reasons.
Measure only bound private job/session roots, without reading frame bodies,
following links, repairing or deleting files. Validate root/file identity and
ownership/mode/link count; withhold all partial totals on change/unsafe objects,
failure, cancellation, overflow or entry/instance bounds. Existing full-limit
commitments and new-launch admission remain unchanged. No stored encoding or
schema65 change. Status catalog declares fresh filesystem metadata observation;
saved receipt lookup remains a saved read.

[Source evidence](diagnostics/2026-11-19-recording-usage-source.json) passes sparse/
partial-owned-file accounting and unchanged bytes; link/FIFO/permission/root
replacement, limits/cancellation and arithmetic/refusal checks; real SQLite/HTTP
state/replay/fences; actual CLI/official-SDK parity; an actual frozen703de9f2
client's omission of unsupported new fields; focused vet; and ten private UI
reads with consent/profile/foreign-response controls. An initial reply-size test
used128-character IDs and correctly failed the existing64-character guard; fix
the fixture, not the protocol. The first UI namespace setup is sandbox-blocked;
the fresh private run succeeds. Visual inspection catches small allocations
rounded to0.0MiB; switch those values to KiB/B and reverify the final view.

The [synthetic UI](diagnostics/2026-11-19-recording-usage-source.png) distinguishes
24KiB allocated from4MiB logical and selected-ship16KiB. Unknown/older omission
does not become zero, and unknown measurement retains known commitments. Blocks
can include shared/compressed extents; this is a non-atomic scoped census, not
exclusive physical accounting, complete coverage or a reservation. Original
frames/receipts and installed namespaces stay untouched. M712's native pass stays
tied to703de9f2; new matching/native/installed qualification, retirement/pressure,
indexing/full exports and the other issue tracks remain required.

## M714 — Prepare the current-schema Grubbery recovery retry (2026-11-19)

The fixed read-only-backed pending-app case still hardcoded target63 in host,
PID1 and guest checks although the current managed helper requires65. Update
those explicit target markers and post-migration observations to65; preserve the
exact original62/17-event predecessor hash, no-auto-resume boundary, separate
overlay, supported lifecycle/audited recovery and original-row checks. Old63 and
foreign target markers remain refused. This changes the qualification fixture,
not a user database or format; the Go PID1 input requires a new artifact freeze.

Both independent guest SQLite helpers had transaction contexts without explicit
close. Apply deterministic closure, preserving read-only transactions and exact
logical-row comparison. A real SQLite lifetime test checks successful reads,
wrong-schema refusal and initialized-fixture inspection with closed handles.
All nine fixture guards pass (0.045s), including immutable artifact boundaries,
wrong targets, finite decompression and the actual kernel file-size cap.

The [case contract](../scripts/qualification/grubbery-runtime-cow/README.md)
retains896MiB capsule scratch plus64MiB preparation reserve, a768MiB root hard
bound and one offline experiment. The original acknowledged app VM05 is only a
read-only backing and never boots writable; production/manual DEV and existing
ships remain excluded. Matching new binaries and full native execution remain
required. Earlier two-version progress/readiness failure stays historical; no
whole Grubbery workflow pass is claimed.

## M715 — Freeze COW65 boot input and prepare separate VM06 retirement (2026-11-19)

Compile the unexecuted PID1 helper from clean
bf5b6c1c12d7e57715b5faf7cd3b119bac4a7214 under a256MiB build reservation.
Its exact binary SHA is
aa4992d292a925986caa64ea64e0173506119f9d44933a0d00b262b34eab17a9;
the unchanged Go source SHA is
bfa53c26aa0c08266520df8d1b9186be04468df5416efea7769e7ac8a7df9eb8.
Require explicit `--init`, verify both hashes and clean original provenance,
then copy it. COW preparation no longer builds Go; the64MiB reserve remains for
preparation. Ten inert guards pass (0.050s), and actual frozen input verification
passes without executing PID1 on the host. New matching three binaries/managed
helper and full native case remain required.

Evict496,758,784bytes of validated regenerable compiler/index cache after source
checks, with all named artifacts and current VM06/07/app05 images unchanged.
The conservative1GiB matching-build reserve plus960MiB native capsule/preparation
cannot fit current storage. Prepare a separate read-only classified retirement
plan for only obsolete recording VM06 root/input (1,594,851,328allocated bytes).
Its fixture failure is explained and newer VM07 passes the complete native case.
Verify stopped units, original identity/backing/no-open-user bindings and65
retained compact/artifact/reference files. Request explicit authorization because
prior permissions name03/04/05 only; no VM06 unlink or new native launch occurs.
The original SQLite/WAL/SHM, receipts, raw frames, logs/source/binaries remain,
as do full successful VM07 and the separate pending app VM05. All production,
preserved DEV and existing/protected/tmux ship exclusions remain in force.

## M716 — Authorized VM06 disposal, matching artifacts and current65 native recovery starts (2026-11-19)

The user explicitly authorizes the old recording VM06 disks. Supersede only
M715's pending two-file boundary: recheck exact hashes/inodes/ownership, closed
units/no open users and65 retained files, then remove its root/input. Reverify
all65 retained files unchanged, including original SQLite/WAL/SHM, raw frames,
receipts/logs and successful full VM07/pending app VM05. No archive is created;
the [record](diagnostics/2026-11-19-recording-vm06-disposal.json) records
1,594,851,328allocated bytes freed and a no-resume tombstone.

Admit1GiB for matching source builds, with no VM active. Freeze clean
12f789521d7303ed182e43fb3c55e00d7e6dca5f and verify3937 tracked input hashes.
CLI f6dec754e51c03fcff90b5abe3e55d37be46cb3c30ef741288426ee1e161bcf4,
collector bfff81f4e714148fcab8ec68d20997a9ccf78ef032293a5661664dfd5d8eedd9,
runner20af5a66f00339d40800d997e9a24c29b7f1c21e9faed17716f5bb9f996d0feb,
helper62a306cc001b5d5a4c78d9f4178d4a79d0659710d2f5ea29225264933627289d.
The helper source pin is derived programmatically from Git and verified before
launch. The independently frozen COW PID1/source/provenance remains exact.

Ten inert guards pass (0.049s). Admit960MiB capsule/preparation reserve:
49,771,778,048charged plus1,006,632,960 is below51,539,607,552. Start fresh
COW04 under the existing one-CPU/no-NIC/no-swap,8GiB guest/10GiB QEMU and768MiB
root-file boundaries, with no Go builds/tests during native execution. The
[current report](diagnostics/2026-11-19-current65-grubbery-recovery-native.json)
is explicitly running, not passed. It has migrated the copied62 store to65,
performed supported same-pier replacement and saved the CLI-reviewed/MCP-applied
runtime audit; original uncertain source reconciliation and both-version/restart
checks remain in progress. The original acknowledged app VM05 is mounted only
read-only and never resumed writable. Production, preserved DEV, existing ships,
tmux and `~/temp-ur` remain excluded.

## M717 — Current65 native recovery completes its original source workflow (2026-11-19)

COW04 observes the exact retained unknown phase without another dispatch, then
completes the original source workflow at guest1750seconds. Advances that return
without final observation are followed by exact saved-step reconciliation; the
test does not mint a replacement request or resubmit an observed effect. Initial
state and the state-update checks pass before the second source review starts.
The [running checkpoint](diagnostics/2026-11-19-current65-grubbery-recovery-native.json)
stays explicitly unqualified as a whole case. Version two, subsequent same-pier
replacement/state, manager restart/original audit lookup, single-dispatch/prefix
verification and final clean completion remain pending. Frozen12f78952 inputs and
all production/manual DEV/original-backing exclusions remain unchanged.

## M718 — Grubbery helper passes; obsolete outer unit deadline prevents whole completion (2026-11-19)

COW04's actual helper passes in2879.43seconds: both complete source versions,
retained/new application state, two supported same-pier replacements, manager
restart, historical audit/original receipt replay and one durable dispatch per
phase. This supersedes M717's pending helper gates only. The whole case fails:
outer guest logging ends at2401.78seconds and no final guest report/clean Stop is
produced. Inspect the actual frozen PID1 unit: its oneshot
`TimeoutStartSec=2400` was missed by the Python helper/host deadline comparison.
This exact timing is consistent with that obsolete service timeout; retained
journal exit evidence is unavailable, so do not assert a confirmed parent exit
status. A runuser helper can continue in the guest's independent user session.

After preserving milestones, terminate only the exact fresh COW04 scope with
TERM; its host exit is-15. Do not call that a clean runtime shutdown. Preserve
the whole failure root/input, source and all artifacts. Both original backings
remain read-only/unchanged. A new offline kernel utility mounts the ended disks
read-only,noload,nodev,nosuid,noexec and extracts59 bounded files, executing no
disk code. On separate SQLite/WAL/SHM analysis copies, schema62's original header
and all17 events remain exact in schema65's two operations/63events/zero leases;
both integrity checks pass. All raw evidence remains unchanged. Native VM,
supervisor and reader cgroups are empty. See the updated
[failed whole-case report](diagnostics/2026-11-19-current65-grubbery-recovery-native.json).

Add a regression reading the actual generated unit rather than another timeout
constant: it fails at2400 versus the required5870seconds. Correct that standalone
PID1 unit to6000seconds with30seconds for stopping; all11 inert guards pass
(0.052s). New frozen boot input and a fresh corrected native case remain required.
This does not qualify installed59→65, preserved DEV, full permits/routes/assets/
peers or production. No installed service/plugin or existing ship is touched.

## M719 — Independently freeze the corrected standalone boot helper (2026-11-19)

Build only the standalone PID1 from clean
bf3e5f83a7202f369dd6c8cdde80c80121656959 after all native/reader cgroups end,
with256MiB admitted under48GiB. Binary SHA is
04d8cfbeac56e9524355ed4cc4f7e7ec837cb983ff2d9a34f240b7bbfa7f10ee;
source SHA is
9e28db77697f9bd7c870905c9cd86d80d571428815c6205d5a87424259ab3fcc.
Keep the three product executables and managed.test frozen at12f78952. All350
actual product/test dependency entries exclude this standalone package;3931
tracked original inputs are unchanged and six differences are documentation/
harness only. No product Go, embedded data, QML, module or sample input changes.

Explicitly allow only this separately frozen Go path in the native harness
boundary. Its mandatory binary/source/clean-VCS pin check still precedes storage
admission and process creation; other Go files, including siblings, still refuse.
This supersedes M714/M715's blanket Go-harness rebuild requirement only for this
independently compiled PID1. It is not an exemption for runtime or release inputs.
All12 inert guards pass, including wrong boot bytes/source/provenance and the
actual unit's6000s startup/30s stop inside the6100s host bound. A fresh corrected
native case and final clean Stop remain required. COW04 stays a retained failed
whole run with its independently passing helper evidence; original app VM05,
production and manual-review state remain untouched.

## M720 — Fresh corrected Grubbery whole-case retry starts (2026-11-19)

After all prior task units end, admit960MiB scratch/preparation under48GiB:
50,121,469,952charged plus1,006,632,960 is51,128,102,912, below51,539,607,552.
Launch one fresh offline COW05 with harness55d6464 and the independently frozen
bf3e5f83 boot. Verify all four runtime/helper hashes still equal clean12f78952.
The original pending app VM05 and pristine base remain read-only; ended failed
COW04 and successful recording VM07 stay intact. No installed/DEV/production or
existing/protected/tmux ship is changed. The new helper observes the exact
original17-event uncertainty; remaining lifecycle/source/state/restart/final
clean Stop gates are running. The [retry report](diagnostics/2026-11-19-current65-grubbery-recovery-retry.json)
is explicitly unqualified until full completion. No host Go builds/tests run
while this native experiment is active.

## M721 — Draft named checkpoint metadata and explicit historical-report reads (2026-11-19)

While COW05 runs its frozen12f78952 app code, draft later issue19 interface work
without host Go builds/tests. Extract the existing runtime-plan/http-plan
constructors into checkplan; a source-text comparison to12f78952 proves both
bodies unchanged except exported function/call names. Legacy wrappers remain.
Add typed, digest-bound `tests.plans.v1`/`tests.plan.v1` metadata. They execute
nothing and cannot grant local-owner qualifier authority or production readiness.

Add separate startup `--allow-test-report NAME:SHA256:/absolute/report.json`
snapshots, bounded to32 owner-private ordinary single-link files/1MiB each.
Require exact complete bytes, physical path, no links/FIFO/devices and unchanged
bindings; close before serving. `tests.reports.v1`/`tests.report.v1` require
explicit private-data consent and exact name/plan/digest. Source roots, runner
enrollment and tool-supplied paths cannot grant file access. Summaries omit raw
output/errors/names/paths/container IDs and distinguish reported outcomes/pin
matches from actual execution/source attestation. Support only the two existing
legacy runtime/HTTP report profiles; no arbitrary check executor is introduced.
No stored report, runner schema65 or runtime/recording format changes.

Prepare meaningful private-file/format/privacy/immutable snapshot tests, exact
actual frozen12f CLI metadata comparison, shared catalog/refusal checks, actual
CLI/SDK stdio parity and all four previously supported MCP protocol revisions.
These tests are **not run yet**: the native experiment retains exclusive runtime/
Go-test admission. The [draft contract](named-checkpoints.md) and
[prepared source record](diagnostics/2026-11-19-named-checkpoint-interfaces-source.json)
claim no source/packaging/native qualification. Later source edits are absent
from COW05's immutable inputs, and its eventual result cannot qualify them.
Focused tests/vet, generated help/reference, matching final artifacts and the
remaining named executor/client and issue tracks remain required. No installed
client, namespace, service, provider session or production state is changed.

## M722 — Corrected Grubbery recovery/two-version/restart case passes (2026-11-19)

COW05 completes the whole fixed native case at frozen12f78952 with separately
pinned bf3e5f83 PID1. The helper passes in2819.46seconds. Supported lifecycle,
CLI-reviewed/MCP-applied linked runtime audit, read-only original unknown-phase
observation, both completed source versions, retained/new application state,
second same-pier replacement, manager restart, historical audit/original replay
and exactly one durable dispatch per phase all pass. The outer parent remains
alive past the old2400s cutoff and completes the final supported clean Stop.
The guest reports passed/helper0; QEMU exits0 with no interruption. This supersedes
M720's running deferral, not M718's preserved failed whole run.

Retain the full successful root/input and58 read-only extracted files. Root SHA
0c28d526285e5b4e055e209e9021e5c683505ddabb5f0ce661678052949d141c;
input333b83305068a98371537373471c884b52dc0742dbfd9bc9af2a42b212f87135;
proof f1341b7c3dba6623bbc95fc825b5d55261462ef811d3371423b84b8444aa6260.
Independent SQLite/WAL/SHM analysis copies verify schema62→65 preserves the exact
original header/all17 events and original pier storage binding. The current store
has two app operations/63events/zero leases and stopped intent; both integrity
checks pass. Original raw evidence, predecessor and pristine base remain exact.
Native/supervisor/reader cgroups end empty. See the
[scoped passed report](diagnostics/2026-11-19-current65-grubbery-recovery-retry.json).

This qualifies the fixed app recovery/state/restart/clean-Stop scenario only;
full peers/permits/routes/assets, broader cleanup failures, stable59→65 and
preserved DEV/production remain separate. Later M721 plan/report edits are not
inside these binaries. Source tests can resume after the reader ends; no installed
client, plugin, service, existing/protected/tmux ship or production state changed.

## M723 — Named plan/report source and CLI/MCP compatibility pass (2026-11-19)

After M722's native/reader units end, admit768MiB under48GiB and run the prepared
focused checks sequentially in4GiB/no-swap/one-CPU source scopes. Checkplan races
and real frozen12f78952 predecessor runtime-plan/http-plan equality pass(1.091s),
shared offline/catalog/selection/authority guards pass(2.622s), official SDK
four-protocol parity passes(7.201s), actual CLI/stdio parity passes(10.695s),
legacy qualifier plan/help/acknowledgement regressions pass(1.017s), and focused
vet passes on all five packages. No engine, existing runner or client profile is
used. Post-test charge51,052,326,912 remains below51,539,607,552; measured growth
577,810,432bytes stays within the805,306,368-byte reserve.

Retain one initial CLI fixture failure(8.649s): every result already matches, but
the final assertion incorrectly rejects the SDK's normal stderr lifecycle logs.
The stdio contract reserves stdout for protocol and explicitly allows stderr
diagnostics. Correct only that test to allow them while rejecting private report
content/path leakage. The product is unchanged to pass the corrected assertion.
Known reported phase categories remain useful without copying arbitrary names;
snapshot time/basis and missing source/execution provenance remain explicit.

This supersedes M721's pending source-check status only. The
[source record](diagnostics/2026-11-19-named-checkpoint-interfaces-source.json)
separates these later interfaces from M722's native12f app qualification. No
persistent format/schema65 changes or generic/named test executor are introduced.
Generated help/reference, matching final artifacts, actual named-client/provider
connections, the remaining issue tracks and stable-to-candidate/preserved DEV
qualification remain open. No installed namespace, service, existing ship or
production state changes.

## M724 — Prepare bounded native recording allocation observation (2026-11-19)

After source scopes end, classify and evict only299 validated regenerable
Go1.27.3 compiler/index cache files, freeing269,217,792bytes. Full owner/path/
single-link/archive/content-hash checks and before/after hashes preserve137
named artifact, whole-VM and compact evidence files, including successful
recording07/current app COW05, failed COW04 and original pending app05. No source,
module download, recording, SQLite, log or image is retired. This is bounded cache
consolidation for matching builds under48GiB; prior unique-disk permissions are
not reused.

Prepare a separate fixed read-only-backed recording07 allocation case. Its
observer uses actual CLI/official-SDK stdio and independent filesystem stat to
compare logical/allocated/count/shared/per-instance measurements. All recording
bytes, control heads, logical database rows and stopped intent must stay exact.
It creates no runtime/capture, policy write, retirement or full pier copy. Its
pure host/foreign-case Go guard passes(0.004s); nine inert Python guards pass
(0.044s), covering actual unit deadline nesting, bounded inputs/output, exact
fixture/schema and explicitly closed SQLite readers.

The [prepared contract](../scripts/qualification/recording-usage-cow/README.md)
declares256MiB capsule/64MiB preparation,128MiB root hard file bound,2GiB guest/
3GiB QEMU/no swap/one CPU and420s host/600s supervisor deadlines. Boot pins remain
deliberately unfrozen: matching clean artifacts and independently pinned PID1,
fresh storage admission and native execution are still required. No qualification
is claimed yet. This tests stationary closed native captures only; active growth,
pressure/retirement/full exports and installed/preserved DEV remain separate.
All production, existing/protected/tmux ship and original-backing exclusions hold.

## M725 — Matching current artifacts freeze; strengthen inherited override ordering (2026-11-19)

Freeze clean f37b32d4f60d5b2e915ad86186caa12c5b853bc2 and verify3960 tracked
inputs after building all four current tools with Go1.27.3/CGO0/trimpath/stripped.
CLI SHA8b12850184e232241fb800e91520df6f5a5660f5878b2c8aa73f0f47fc5f98b2;
collector705646950713a865d2d8de77faa9494450468c91edd976f01a417ee0583c7726;
runner ea0508ee09d9d31586c25bb91157f9ff0e8d63c01efb7e5c8d030dfd437995cb;
managed.test fb8db26a5921d19a6b2882a87f20d1235c41be1645961b855823ed5db9c4c70a.
Actual buildinfo reports the clean source for three tools; the helper's exact
link-time source binding is exercised and matches. Builds use sequential bounded
source scopes with512MiB initial and256MiB helper-phase admission. No runtime or
installed deployment is updated.

Before native launch, inspect the inherited fixture contract: recording07 retains
recording-pilot.conf. The prepared global99-zz override sorts before that name,
so its ExecStart can lose the merge. Preserve the original file and change only
the fresh-overlay standalone PID1 filename to zzzz-issues-recording-usage.conf.
A new actual-template guard verifies ordering, exact candidate ExecStart reset,
readiness condition, recording selection, no resume and no original removal.
All10 inert Python guards pass(0.042s). No native failure is manufactured; boot
pins remain deliberately unfrozen and admission refuses launch until correction
is separately frozen. The four product/helper inputs stay at f37b32d4; the known
standalone-PID1 exception permits only its separately verified source/binary/
clean-VCS correction. New boot, fresh320MiB admission and full native measurement
remain required. No unique VM, recording, SQL/history or existing ship changes.

## M726 — Corrected allocation boot pins and storage admission pass (2026-11-19)

Build only the corrected standalone PID1 from clean
e0af21ae13dc781b4e299b1afd48476fb5d41a54; binary SHA
ced164ae56c3ee51dd7675af92431dedbf2def7975c1689d38e4522219f66819,
source cbd497447962bf8ca949288bc602c87d5f4eadf6e4812517a2f90878eccccfb3.
Pin and exercise the actual boot hash/source/clean-VCS verification. All10 inert
guards pass(0.043s). Verify every other original f37b32d4 tracked input still
matches or is an explicit documentation/Python harness change; all four runtime/
helper binaries remain frozen unchanged. This uses only the existing separately
pinned standalone-PID1 exception, not an exemption for product or embedded inputs.

The full320MiB reservation fits:51,179,536,384charged plus335,544,320 is
51,515,080,704, below51,539,607,552. Actual launch/worker admission must recheck;
kernel128MiB root and bounded input/log limits retain the full capsule boundary.
No guest has started yet and no native measurement pass is claimed. Current
production, preserved DEV, existing/protected/tmux ships, successful/failed whole
fixtures and original recording07 backing remain unchanged.

## M727 — Native allocation helper passes; isolate overbroad observation-row assertion (2026-11-19)

First usage COW01 runs exact f37b32d4 tools and corrected e0af21ae boot. Its
actual CLI/official-SDK helper passes independent stat allocation/attribution and
unchanged capture/control-head gates:19files/6directories,29,495logical and
94,208allocated bytes, including8,192shared bytes. The whole case fails its final
literal all-table comparison; helper0/QEMU0 are not a whole pass. Preserve root
9d2b724355a1b7391bb44cb85bb157a01626408898071688ec31daeaaa3da8cd and
input1179039efe15e81d22e0b2af74ee52bb146af57efda0517a6ae6942452e7ea5b.

After VM/supervisor end, an offline kernel utility mounts current/original07/
inputs read-only,noload,nodev,nosuid,noexec and exports8 bounded files, executing
no disk code. Separate65 SQLite copies both check ok. The independently exported
original table digest matches the guest's pre-service snapshot. Only instances.
body changes: public.observed_at refreshes and runtime_revision75→76. Every other
table cell and instance value, including stopped intent/storage, is identical.
All raw files, roots/backings remain unchanged. See the
[failed whole-case evidence](diagnostics/2026-11-19-recording-allocation-native-fixture-failure.json).

Correct only the fixture comparator: allow that one later UTC observation and
exactly one revision increment; preserve every other value/row/table. A regression
reproduces the original equality failure, verifies the actual extracted rows,
and refuses changed intent/storage, missing values, extra history/rows/tables,
stale time and additional revisions. All11 inert guards pass(0.047s). Product/
helper/boot binaries stay unchanged. The ended case remains failed; a fresh full
retry/admission is required. No original image, policy, runtime/capture, installed
namespace, existing ship or production state is changed.

## M728 — Preserve failed allocation evidence and admit corrected retry (2026-11-19)

After all task/native/reader/build scopes end, evict only270 newly classified
Go1.27.3 compiler/index cache files(100,954,112bytes). Verify content hash, archive/
index format, owner/physical single-link bindings, and157 preserved files before/
after, including both failed allocation disks, exact original/current SQLite/
proofs, all matching f37b32d4 tools/e0af21ae boot and prior full references. No
unique image, source/module download, recording, SQL, log or retained evidence
is retired. This does not reuse prior user disk-removal permissions.

Admit the fresh corrected320MiB case:51,161,755,648charged plus335,544,320 is
51,497,299,968, below51,539,607,552. The product/helper/boot artifacts remain exact;
only the fixture's exact single-observation comparison and inert regression
change after freeze. Original usage COW01 remains failed and intact. Commit the
correction before launching a fresh COW02; actual worker admission rechecks and
full native success are still required. No guest starts at this checkpoint and
all production/manual DEV/existing/protected/tmux ship exclusions remain.

## M729 — Closed native recording allocation passes with independent preservation (2026-11-19)

The fresh corrected COW02 passes with unchanged frozen f37b32d4 runtime/helper
and e0af21ae standalone boot artifacts. Actual CLI and official-SDK stdio usage
match independent filesystem stat:19files,6directories,29,495logical bytes and
94,208inode allocated bytes, split into86,016instance and8,192shared allocated
bytes. Original capture/job bytes and policy head stay exact. Guest/helper/QEMU
all finish successfully; the exact VM/supervisor scopes are inactive and empty.

A separate64MiB-admitted read-only kernel utility exports8 fixed evidence files,
including original/current SQLite. It executes no source-disk code and leaves
all whole images, inputs and pristine/original backings unchanged. Independent
analysis on separate SQL copies verifies both schema65 quick checks, the exact
original logical-row digest and every original row/value except the one explicitly
allowed later UTC observation timestamp and runtime revision75→76. The retained
raw SQL/export bytes remain unchanged. Native proof SHA256 is
1d070f4f406a4b2ee27d5d3d1efe74d24bb919d48e922b3d2c3ce288fa3e9bbd.

This supersedes M728's pending retry only; M727's earlier whole failure remains
failed and retained. The [scoped evidence](diagnostics/2026-11-19-recording-allocation-native.json)
qualifies stationary closed captures after reboot. Active growth/pressure,
retirement, complete exports, other factories, installed stable upgrade and DEV
are still open. No installed namespace, existing ship or production state changes.

## M730 — Generated help matches the current frozen tools (2026-11-19)

After the reader scope ends, admit64MiB scratch under48GiB and regenerate then
check all179 public help entries from the three frozen f37b32d4 tools. Both
networkless Bubblewrap passes succeed with a private empty HOME and no installed
registry, engine or desktop socket. Refresh the MCP HTTP and exact private report
startup flags in generated omabit/runner references. Collector help is unchanged.
The reference reflects available commands, not native workflow qualification;
named execution/actual provider clients and the final upgrade/DEV gates remain.

## M731 — Exact original recording exports pass source/CLI/MCP checks (2026-11-19)

Add typed owner-private original export manifest/part reads and a shared streaming
ZIP assembler. Closed saved jobs bind exact runner/instance/job/session/metadata;
active writers, missing results, unsafe files and changed hashes refuse.256KiB
parts fit existing RPC replies; complete file hashes are verified before ZIP
completion. All retained encoded originals, including torn/corrupt tails, remain
separate from verified-prefix timing/gap/exit coverage and runtime completeness.
No schema65, job, control, history or original file changes. The CLI writes a fresh
0600 archive in an existing physical0700 directory, syncs and publishes with a
no-overwrite link, and retains failed output as.partial. Archive entries are0600.
MCP receives the same exact reads without model-selected local output paths.

Focused recording/worker/API/client/catalog races pass. Final storage/access/
client/actual CLI+official-SDK stdio checks pass(1.138/1.315/1.011/21.853s).
Retain two initial fixture failures: generated recording-worker help lacked an
explicit internal-only catalog exclusion; the CLI fixture's temporary directory
was not0700. Add that inventory entry and chmod only the new test directory,
preserving product privacy checks. Source review also extends only the exact
export RPC client/server/SSH deadlines; prefix variants retain ordinary bounds.
The [source report](diagnostics/2026-11-19-recording-original-export-source.json)
records results and limits. This is complete original encoded-file packaging,
not a redacted bundle or separate searchable timestamp index. Matching native
export, final stable upgrade/preserved DEV and remaining issue gates stay open.

## M732 — Bound compiler storage; prepare native original-byte checks (2026-11-19)

The first source admission reserves256MiB, but post-test charge is51,724,771,328,
185,163,776bytes above the approved51,539,607,552-byte cap. Do not claim the
source checks stayed within reserve. Pause new builds/native launches; classify
and evict only1168 regenerable compiler/index files in cache37/38, freeing
805,879,808bytes while170 named artifact/whole-VM/compact evidence pins remain
exact. No unique VM or original capture/database/log/source is retired.

A new fast cache/temp/build allocation monitor retains the fixed full aggregate
charge and sets a stop threshold at least64MiB below the cap. Normal-mode source07 compilation
rebuilds older cache entries and crosses its512MiB reservation; the monitor
terminates only that scope at an estimated51,484,127,232bytes, below the cap.
The whole Go invocation remains interrupted. Its completed SHA-pinned test binary
passes the pure host/foreign-scenario guard directly; actual native and unfrozen
build-identity tests skip. Retain that binary unchanged, and remove only its895
classified regenerable compiler work files(269,180,928bytes), preserving older
historical test directories. Focused seven-package vet then passes under a new
256MiB monitored admission. Full post-vet charge51,245,133,824 is below48GiB.

Prepare additive source-bound native export assertions on a fresh read-only-backed
recording07 overlay. A required scenario flag prevents omission. The helper must
compare both closed jobs' actual CLI/SDK manifests and every part with independent
original hashes/files, and real CLI ZIPs with those exact encoded originals.
Private bounded proof retains ZIP bytes/hashes for independent analysis. All
capture/control/history/intent preservation and previous observation-only rules
remain. Eleven inert Python guards and the pure Go VM guard pass; matching clean
builds, admission and actual execution remain pending. The separately frozen
PID1 source is unchanged. No installed namespace, production or existing ship changes.

## M733 — Freeze matching original-export tools and180 help entries (2026-11-19)

The larger cache classification finds insufficient eligible files and refuses
without eviction. Classify the available744MiB subset instead:822 hash/inode/
archive-verified regenerable files free781,938,688bytes; all174 named artifacts,
whole images and compact/source-check evidence pins remain exact. Do not retire
any original VM or use a unique-disk permission for this cache work.

Freeze all four CGO-disabled, trimpath/stripped tools at clean3fc6eab3, checking
3978 tracked input hashes before/after, actual clean product VCS and the helper's
explicit source identity. Its host guard passes; actual native execution skips.
The single consistent build mode uses an896MiB admission and monitored source
scopes. Peak fast estimate50,960,588,800 and full final charge50,843,561,984 stay
below51,539,607,552. No compiler reservation or cap interruption occurs.

Under a separate64MiB admission, build the exact source/kind/required-export-bound
read-only reader and regenerate/check all180 help entries from these tools in
networkless private HOME scopes. Both pass. The actual updated command is now in
the generated reference, with explicit internal-worker exclusion in discovery.
The [artifact record](diagnostics/2026-11-19-recording-original-export-artifacts.json)
keeps frozen hashes and native requirements. The unchanged separately frozen
PID1 may be reused; a fresh native export case, admission and preservation evidence
remain required. No installed namespace, existing ship or production changes.

## M734 — Native original exports match exact retained bytes and original07 chains (2026-11-19)

Admit320MiB and run fresh COW03 with matching3fc6eab3 tools and unchanged e0af21ae
PID1. Actual CLI/official-SDK manifests and every part match both original closed
sessions, and actual private CLI ZIPs match their encoded files. Allocation counts,
original capture/control/history and stopped intent pass. Guest/helper/QEMU all
finish successfully; exact VM/supervisor scopes are inactive/empty. Proof SHA256
is5bb1ee3fdf432ef355b9e25877690c4e3e2ca34799610b91c7f5c40ca5c1ef03.

After those scopes end, a16MiB-admitted reader refinement adds only the four exact
original recording07 files named in the frozen result. The source/kind/required-
export-bound reader mounts source/input/original read-only/noload/noexec; no source
code or services run. Under64MiB admission it exports12 fixed evidence files,
leaving every root/input/pristine/original backing hash unchanged. Independent
Python ZIP/hash analysis compares both archives byte-for-byte with all four raw
original metadata/segment files, verifies manifest/chunk/whole-file hashes and0600
entries, and decodes every frame with independent struct/SHA256 logic. The170/32
frame counts, stdout/stderr sizes/hashes and final timing/digest chains match the
original703de9f2 native07 report exactly. No reconstructed terminal output is run
or printed. Both archives and raw extraction remain private and retained.

Independent schema65 quick checks on separate SQL copies pass, original logical
digest matches the guest and all original rows/values are exact except the one
later observed_at and runtime revision75→76. Raw exported SQL/proof/capture bytes
remain unchanged. The [native report](diagnostics/2026-11-19-recording-original-export-native.json)
supersedes M733's pending native original-export gate only. Complete redacted
bundles, a separate timestamp/search index, active growth/pressure, retirement,
other factories, previous-stable installed upgrades and preserved DEV remain
separate. No production, installed service, existing ship or manual-review state changes.


## M735 — Bound current project allocation and server-held net growth (2026-11-19)

Implement an operator-private, exact revision/plan/ledger-bound read for current
expanded members. Validate creation contracts and saved mount/pier objects without
engine/process/HTTP probes. Metadata-only pinned-root walking separates sparse
logical length from inode blocks, counts internal hardlinks once, inspects pipes/
sockets without reading them, bounds entries/depth, and refuses unsafe/changed or
incomplete trees without partial totals. Current mounts/runtime/project caches
remain distinct from explicit historical/shared/journal/staging exclusions. No
shared allowance or full budget is invented. No measured file contents are read.

A single census releases the manager lock during the scan and brackets the exact
raw saved ledger. Reads remain available with mutation admission fenced and leave
every SQL row, original interrupted receipt and pending fence unchanged. At most32
cloned volatile baselines use monotonic completion times and exact root bindings;
fast polls preserve a comparable baseline, unknown/binding changes invalidate it,
and one-second to five-minute windows emit exact unsigned directions/deltas.
The API/client/restricted SSH and shared CLI/official SDK enforce private consent,
role/capability/identity, exact deadlines, no retry and integers above2^53.

Focused ordinary checks pass. The first combined race build reaches its compiler
reservation; stop only that source scope at sampled51,454,492,672bytes, with full
post-charge51,454,816,256 below the51,539,607,552-byte cap. Only its filesystem
package passed; the combined invocation remains interrupted. Retain that report.
Classify/remove only776 regenerable scratch files(308,461,568bytes), including an
incomplete zero-header linker output; an initial completed-ELF classification
correctly refuses without removal.331+551 verified older compiler/index files
free489,988,096bytes, preserving205/209 named evidence/artifact pins. Larger
classification and a temporary classifier typo refuse without deleting anything.
No unique VM, original capture/database/log/source or frozen artifact is retired.

Smaller final races pass: runtime adapter2.426s, manager9.533s, filesystem1.100s,
client1.013s, SSH1.165s, catalog2.620s and actual CLI/SDK8.284s. Seven-package vet
passes using the same race cache. Final full charge51,145,261,056 is below48GiB;
all final exact source scopes are inactive/empty. See the
[source report](diagnostics/2026-11-19-project-allocation-source.json).
This supersedes M734's pending first resource-read implementation only. Native
project census, original/recovered adapters, complete/shared budget and event
payload evidence, durable incident/reviewed resume/journaled containment, matching
artifacts and installed/preserved DEV gates remain open. Schema65 is unchanged;
there is no new persistent authority, guard, threshold or pruning. Production,
existing ships, installed services/plugins and manual-review state are unchanged.


## M736 — Read complete original project journals without creating expansion (2026-11-19)

Supersede M735's pending original/nonexpanded read adapter only. Add an explicit
`ledger_kind` for exact raw expanded-ledger binding versus a derived original-ready
member/storage projection. Complete each original boot/readiness chain through
existing validated context selectors, preserving initialized pier/mount identity.
Bind exact selected observed readiness bytes and every selected-project operation,
runtime mirror and fingerprint, with1024rows and4MiB per source group admitted
before decoding. Missing proof, any original retirement/replacement/anchor-recovery
history or active original work remains unsupported; no historical creator is
adopted. A reservation never becomes a zero-usage result. No expansion row, member
phase or other persistent data is created/changed by the read.

The actual authenticated API fixture preserves every original row. Independent
raw-readiness whitespace and operation-fingerprint changes with an unchanged
member projection both withhold the census. A deliberately oversized invalid
private-fixture body refuses at byte admission before decode. Legitimate retired
and recovering original journals refuse without backend scans. A real disposable
Linux original-contract filesystem fixture reads without engine commands. Both
binding kinds pass actual CLI/official-SDK transport and strict model validation;
changing source kind invalidates a prior growth baseline.

Focused seven-package races pass: filesystem1.081s, runtime2.446s, manager20.595s,
client1.014s, SSH1.163s, actual CLI/SDK9.672s and catalog2.648s. Updated catalog
wording passes its separate2.213s race check; seven-package vet passes. There is
no monitor interruption: source10's fast sampled peak51,471,007,744 is below its
51,472,498,688 stop threshold, and final full charge51,326,320,640 is below the
approved51,539,607,552-byte cap. Exact final scopes are inactive/empty. No further
cache/scratch/VM retirement occurs. See the
[source report](diagnostics/2026-11-19-project-allocation-original-source.json).

This remains a non-atomic current-root census with incomplete historical/shared
budget and no event-payload or containment authority. Schema65 is unchanged.
Original replacement/recovery histories, matching native measurement/independent
stat/history checks, explicit budgets/shared allowances, durable incident/reviewed
resume/journaled stops and final matching/preserved DEV upgrades stay open.
Production, installed services/plugins, existing ships and manual-review state
remain unchanged. M735's original scope/results and interrupted build remain
retained rather than relabeled as later qualification.


## M737 — Retire obsolete project compiler variants without evicting std/dependency cache (2026-11-19)

A64MiB-admitted dependency inventory uses `go list -export -test`, which can
compile test exports instead of merely reading cache metadata. Its monitored
scope reaches the reservation and is stopped: no completed dependency inventory
or product test result is claimed. The sampled peak51,425,955,840 and retained
post-inventory full measurement51,426,308,096 are below48GiB. Keep the interrupted
report; all implementation races/vet from M735–736 remain passed separately.

The exact scope is inactive/empty. Remove only its21 hash/inode/owner-verified
regenerable compiler archives/importcfg files(49,586,176bytes), preserving210
named pins. Inventory older Omabit archive variants by module-path evidence in
__.PKGDEF. A fresh192MiB classification selects63 complete Go1.27.3 archives
whose content hashes match their cache names and whose mtimes predate source10;
standard-library/dependency and fresh cache entries are excluded. Removal frees
206,110,720bytes, preserving221 VM/SQL/recording/log/source-result/frozen-artifact
pins exactly. No unique image or original evidence is retired. Full post-cleanup
charge51,170,893,824 leaves368,713,728bytes under the51,539,607,552-byte cap.

Prefer this narrow obsolete-variant classification to evicting currently reused
standard/dependency archives. Treat export/dependency discovery as an admitted
compiler action unless demonstrated otherwise. The
[cleanup record](diagnostics/2026-11-19-obsolete-compiler-variants.json) is storage
accounting, not product/native/installed qualification. Product code stays at
M736; all final source scopes end. Production, existing ships and installed/
manual-review state remain unchanged. Recording timestamp/index and complete
derived export implementation remain next; no new capability is claimed here.


## M738 — Verified payload-free timing index, query and optional ZIP entry (2026-11-19)

Supersede M737's pending first timestamp-index/query implementation only. Add a
portable bounded original-part reader verifying metadata, each chunk and complete
file SHA, then decode exact segment/sequence/digest, receive epoch, stream and
exit/gap rules. Index only the complete verified prefix; still drain/hash corrupt
tails and later unparsed files and compare every manifest coverage field before
emitting an end record. Never write or repair originals. JSONL exposes source
bindings, filenames/offsets/lengths, wall/receive timing, stream/payload lengths
and digests, with no payload contents. The stable content digest excludes volatile
read time.1,048,576frames/256MiB total/256KiB line limits fail without completion.
Retained/raw/verified-prefix completeness never establishes runtime completeness.

Typed `runtime.recording-timing.v1` uses the same original export authority with
operator/private consent and no new runner endpoint or filesystem selector. Filter
inclusive wall ranges/stream, order by sequence, require exact source digest for
continuation, and return a separately authorized cursor to surrounding original
context. No matches escape before complete verification. Six-minute trusted
catalog timeout preserves earlier caller deadlines. `recording-export
--timing-index` explicitly adds a0600 payload-free index and re-reads pinned parts;
default export keeps its one-pass entries/behavior. Failed second-pass reads do
not close/publish an archive. No persistent schema/capture/control change occurs.

The first source invocation fails only a test int64-versus-int comparison before
execution; retain its report and correct the explicit cast. Focused races then
pass exact raw-byte offsets/digests, reversed wall time, rotation/restart and
corrupt-tail gaps, incomplete/corrupt/empty/exit/extra-after-exit, source changes,
hash/coverage mismatch, cancellation/fetch/short writes and no payload leakage.
Query continuation/time filters and original context cursor are checked against
actual stored payload. Indexed ZIP retains every raw file/mode/hash and has no
completion directory after an injected second-pass failure.

Final codec/default-export/segment races pass1.609s. Actual CLI/official-SDK query
and real-codec optional ZIP plus existing default/private-history interface races
pass29.863s; catalog2.262s and three-package vet pass. Exact integers above2^53,
operation/session bindings and all private role/capability/consent/target/source/
part/transport refusals remain. See the
[source report](diagnostics/2026-11-19-recording-timing-index-source.json).

Before those checks, remove only28 hash/inode/format-verified pre-index Omabit
compiler archive variants(143,446,016bytes), preserving current index and std/
dependency cache and227 named pins. No unique VM, original capture/database/log/
source or frozen artifact is retired. Final full charge51,136,983,040 stays below
48GiB; all final source scopes end. Matching clean tools/reference/native index,
complete redacted bundles, oldest coverage/retirement/pressure/other factories,
resource containment and final installed/preserved DEV gates remain open.
Production, existing ships and installed/manual-review state remain unchanged.


## M739 — Complete derived redacted ZIP with shared manifest/parts and explicit choice (2026-11-19)

Supersede M738's pending complete redacted export implementation only. Factor
original metadata/chunk/whole-file/codec/coverage verification into one private
bounded scanner shared by timing and redacted output. Never rewrite originals.
Transform at most64frames/128KiB per window through the existing escaped-literal
credential rules, carrying only independently verified newline context per stream
and resetting it at gaps. Withhold leading/trailing partial lines. Optional Go
formatter literals are ephemeral and single-line; the shared default exposes no
secret-file selector. Finite rules still require owner review before sharing.

A private ZIP contains only technical manifest, escaped frame windows with full
timing/digest/omission reasons and final verified coverage. Exact identities,
paths/attachment body, raw metadata and encoded originals are omitted. Exit137/
OOM/status facts remain typed; container ID becomes an explicit alias/omission.
Counters distinguish all verified frame-payload bytes, omitted stream bytes and
aliased exit bindings. Content-size excludes the self-referential coverage record,
which still counts toward its limit. Damaged tail bytes are hashed/reported but
not copied. No final coverage/ZIP directory appears before full verification.
Derived content/ZIP is bounded256MiB, records4MiB and verified frames1,048,576.

Shared `runtime.recording-redacted-export-{manifest,part}.v1` retain exact operator/
job/private consent and explicit representation=redacted. The deterministic
manifest binds source/ZIP/size/256KiB hashes; each part rederives/verifies the whole
source before returning one chunk. This avoids persistent derived cache/retention
authority but can be expensive for large exports; each operation has a trusted
six-minute bound. CLI `--redacted` uses these same handlers and existing private
partial/no-overwrite publication. Ambiguous/no choices, original choice through
the derived API, or the original-only index flag refuse without raw fallback.

Library races first pass1.683s. Source02's interface run remains failed: its
fixture reused c*64 as both container ID and technical image digest. The omission
assertion therefore matched the intentionally retained image hash in manifest.
Use a distinct9*64 container fixture without weakening omission/metadata rules;
also make lifecycle alias/stream omission counters and manifest checks explicit.
Final codec/redaction/index/timing/segment/default-export races pass1.775s, actual
CLI/official-SDK manifest/part/real ZIP plus original/index regressions40.469s,
catalog2.247s and three-package vet. Independent chunk-boundary hashes, whole ZIP
assembly/read-time stability, source/request/choice/consent/hash/writer failures,
split literal/line/gap omissions,0w diagnostics and original byte preservation
pass. See the [source report](diagnostics/2026-11-19-recording-redacted-export-source.json).

No further cache/scratch/image retirement occurs. Final full charge51,297,136,640
stays below51,539,607,552; final exact scopes are inactive/empty. Schema65 and raw
capture/job/control formats are unchanged. Matching clean artifacts/reference,
independent native indexed/redacted exports and preservation, oldest coverage/
retirement/pressure/other factories, resource containment and final installed/
preserved DEV gates remain open. Production, existing ships and installed/
manual-review state remain unchanged; the failed fixture run remains retained.


## M740 — Require complete derived exports in the fixed native helper (2026-11-19)

Prepare, without qualifying, the native assertions for M738–739. Both guest and
Go identity/source guards require the separate derived-export scenario flag.
For each exact closed job, retain default original checks and additionally compare
indexed ZIP originals, every timing row, time/kind/continuation/context results,
redacted CLI/SDK manifests and every part, and actual redacted CLI ZIP bytes.
Private proof preserves archives/queries/manifests for independent decoding.
Eleven inert Python guards pass; matching clean artifacts, Go guard, actual native
case and independent verification remain required. The frozen standalone PID1,
read-only predecessor, one-runtime lock, measured48GiB admission, no source-build
concurrency and all production/manual-review exclusions remain unchanged. See
[prepared assertions](diagnostics/2026-11-19-recording-derived-native-preparation.json).


## M741 — Freeze matching derived-export tools, reader and generated help (2026-11-19)

Build the three products and explicitly source-bound managed helper from clean
4c0b16e9d81311269db57664aa6906187eb23cc1, preserving4018 tracked input hashes
before/after. All products report exact clean VCS identity, CGO0 and trimpath;
helper source/refusal guard passes with actual native case skipped. Build a
separate source/required-flags-bound read-only evidence utility; its host refusal
returns80. Networkless generation/check of180 help entries passes with no HOME
state. Inspection then finds static export help still describes only original
choice; next source checkpoint must add redacted/index choices and regenerate
final help. Do not equate regeneration parity with complete discoverability. See
[artifact pins](diagnostics/2026-11-19-recording-derived-export-artifacts.json).

Remove only114+55 hash/inode/format-verified regenerable Omabit compiler archives,
reclaiming342,069,248+123,293,696bytes with234 then244 named pins unchanged.
Keep standard/dependency caches, every frozen executable, database/capture/log,
source and whole reference/failure VM. The matching build uses a measured512MiB
reserve and monitoring; preparation uses64MiB. All final scopes end. Fresh native
admission/execution and independent verification still gate the new exports.
No production, existing ship, installed DEV or manual-review state changes.


## M742 — Preserve failed derived-export status framing and require compact retry (2026-11-19)

The4c0b16e guest reports every new native assertion passed, but the whole COW04
case fails: copying the entire private proof into serial status exceeds journald
line framing, so the host parser raises unterminated JSON. Do not promote the
fragment's passed flag into native qualification. Both exact scopes end; retain
the whole failed root/input and private proof for independent reading. See
[failure and pins](diagnostics/2026-11-19-recording-derived-serial-failure.json).

The guest now allowlists only required booleans/count and complete-proof-retained
status. Exact complete archives/queries stay in the separate private proof with
SHA-256; no proof bytes are discarded. Twelve inert Python guards pass, including
an oversized private-proof regression and refusal for every missing/false gate or
wrong job count. Product/helper Go and the separately frozen PID1 are unchanged;
a fresh measured native retry plus independent verification is still required.
Production, existing ships and installed/manual-review DEV stay unchanged.


## M743 — Native indexed originals, timing queries and complete redacted exports pass (2026-11-19)

Supersede M738–742's native deferrals only for closed copied recordings at matching
4c0b16e tools, e0af standalone PID1 and d794846 corrected Python harness. Fresh
COW05 passes every helper assertion and host completion without interruption;
all actual CLI/official-SDK timing/manifest/part/ZIP operations pass. Both original
encoded sessions remain byte-exact. Read-only kernel evidence exports fixed paths
without executing source-disk code; independent Python decoding checks170/32 raw
frames against every index offset/length/time/sequence/chain, query filters/pages/
context, all derived ZIP/chunk hashes, exact escaped span bytes/ranges, omitted
counts, technical manifest whitelist, exit aliases and final coverage hashes.
See [native evidence and limits](diagnostics/2026-11-19-recording-derived-export-native.json).

Both separately copied SQL databases quick_check; every original row/control/head,
stopped intent, recording and backing/input remains exact except the one allowed
fresh observed_at and revision75→76. Independently read the failed COW04 proof too:
all data/assertions pass and complete redacted manifests match across both copied
boots. Its original whole-case status remains failed due to serial framing; keep
the full failure root/input/evidence. The fresh retry supplies the whole-case pass.

A further137 older compiler archives alone reclaim67,194,880bytes with252 retained
pins exact. Final full charge51,266,838,528 stays below48GiB. All six exact native/
reader scopes end empty, and all matching artifact/case hashes recheck. Active
capture/pressure/oldest coverage/retirement/other factories, static help choice
visibility and final previous-stable/native-plugin/preserved DEV gates remain.
This is no production, existing-ship or installed/manual-review state change.


## M744 — Classify obsolete failed recording VM01; disposal awaits approval (2026-11-19)

Document this older failed fixture before requesting its unique whole-image
removal. Source2d028197's VM01 ends under a generic storage/log/deadline guard;
the exact limiting predicate is unproven. SQLite64 preserves registration success,
Start still running and one recording intent, so it is not terminal/upgrade
qualification. Exact DB/WAL/SHM, job claim/readiness/session metadata,6528 raw
recording bytes, commands, source and binaries remain under68 verified pins.
Its partial disposable pier exists only in the whole archive. The same-source
successful VM02 whole archive and newer recording07/pending app05/corrected app
COW05 remain intact. See [insights and proposed disposal](diagnostics/2026-11-19-recording-vm01-disposal-proposal.json).

Only the494,424,064-byte allocated gzip and312,774,656-byte allocated input disk
are proposed,807,198,720bytes total. Both ordinary/single-link owner file hashes
match their original manifests. This would remove the only complete image of
that older failed fixture and requires specific authorization; no removal has
occurred. Do not extend earlier recording03–06 permissions to these files. The
48GiB cap, one-runtime rule and production/manual-review exclusions remain.


## M745 — Correct export help choices in source; final generation remains open (2026-11-19)

Fix the static command help missed by M738–739: show exactly one original/redacted
choice, the original-only optional timing index, shared timing-query context,
derived omission/finite-rule privacy limits and private output/partial rules.
This changes only help text; formatting/diff checks pass. The prior4c0b16e native
export evidence remains scoped to its actual frozen tools. A new matching CLI
build and generated reference are still required; do not hand-edit generated
help or claim current parity from the earlier180-entry check. See
[source status](diagnostics/2026-11-19-recording-export-help-source.json).
The proposed old recording VM01 disposal is still pending; no unique image/input
has been removed. Production/manual DEV and existing ships remain untouched.


## M746 — Approved VM01 removal and broader test-archive retention authority (2026-11-19)

The user specifically approved VM01 archive/input removal. Revalidate exact
ordinary single-link file identity/hash and68 retained pins before/after, hold
the one-runtime lock and verify exact scopes ended. Remove only the two approved
files,807,198,720allocated bytes; preserve compact SQL/WAL/SHM/job/raw recording,
commands/source/binaries and all protected full references. A DISPOSED marker
prevents following old restoration instructions. Full charge50,460,041,216 stays
below48GiB. This supersedes M744–745's pending decision only, not historical failure.

The user then expressly permits removing other unneeded VM archives after noting
lessons and retaining needed state/version transition fixtures. Routine classified
obsolete test disks no longer need repeated permission. This does not authorize
production, user ships, installed/manual-review DEV or removal of required test
state. Preserve original pending app05, full successful app COW05, original
recording07 and actual successful schema64 recording02 archive/input.

## M747 — Classify nine superseded COWs with compact transition evidence (2026-11-19)

Before removal, record lessons for failed app COW01–04: acknowledged creation
without final observation; whole-context expiry during version two; missing
read-only startup readiness retry after manager restart; and obsolete generated
PID1 unit deadline despite helper success. Corrected app COW05 supplies the
whole successful same-pier/two-version/state/restart/replay/final Stop evidence.
All original failed statuses remain failed. Exact independently exported stores,
headers/events, compiled source, receipts and logs remain pinned.

For recording usage/export COW01–05, retain the observed_at/revision comparator
failure, corrected allocation pass, original-export pass, oversized-serial-proof
failure and corrected indexed/redacted whole pass. Exact SQL/WAL/SHM when present,
raw original bytes, complete private ZIP/query/proof evidence, source and frozen
binaries stay; recording07 remains the full backing for fresh read cases. The
closed read COWs add no required state transition beyond retained compact stores
and the immutable original; the app failed COWs are superseded by preserved
original62 pending/current65 full app fixtures and compact intermediate63/65 stores.

Classify only18 image/input files(795,537,408allocated bytes), verify1272 retained
pins and all exact task scopes ended. This is classified, not yet removed. See
[consolidation and lessons](diagnostics/2026-11-19-obsolete-test-vm-consolidation.json).
The user's new authority supersedes prior whole-copy retention requirements for
these exact obsolete cases only. Needed transition inputs/backings and production/
manual DEV/user-ship exclusions remain. Do not silently promote any failed case.


M747 execution update: all18 classified obsolete image/input files are removed
under the user's broader authorization, reclaiming795,537,408allocated bytes.
All1272 retained pins match before/after and required full transition fixtures
remain. Each retired capsule has a DISPOSED marker; do not follow old whole-image
retention/restoration instructions. Full charge49,665,126,400 remains below48GiB.
This changes retention only; earlier whole failures remain failed and passing
historical assertions remain bound to their original source/evidence.


## M748 — Separate retained output sequence coverage from wall-clock extrema (2026-11-19)

Add optional version1 `output_coverage` to the completed timing summary. Accumulate
only stdout/stderr/combined output from the same exact verified source scan;
exclude lifecycle/gap timestamps. First/last sequence/time and earliest/latest
wall time stay distinct when the wall clock reverses. Filters and match limits do
not truncate this complete-prefix summary. Empty/exit-only prefixes explicitly
report no verified output; absent older fields are unavailable, never zero. Return
no summary after failed source/hash/output checks. This says nothing about output
before attachment, another session, host-wide oldest coverage or full runtime.

[Focused source evidence](diagnostics/2026-11-19-recording-output-coverage-source.json)
passes full recording races(1.933s), actual CLI/official-SDK timing/derived/indexed
ZIP and existing no-state help tests(19.424s), service races(7.912s) and vet.
Tests preserve original bytes, reverse time/sequence distinction, lifecycle-only
and damaged-tail semantics, exact integers above2^53 and authority/failure
withholding. Twelve inert guards require the new native flag and prevent private
proofs entering serial status. No recording/index file or persisted schema65
encoding, endpoint, capability, admission, policy, retention or lifecycle changes.
Static help describes the new explicit output scope. Full post-charge49,830,883,328
stays below48GiB. Matching clean tools/helper/reference and fresh native/independent
verification remain required; production/manual DEV/user ships stay unchanged.


## M749 — Matching output-coverage tools and corrected generated help pass (2026-11-19)

Freeze all three tools plus explicit-source managed helper at clean
5344c5cd2816623b69450f638ee44966a31b9ee4. All4025 tracked hashes match before/after;
actual product VCS/CGO0/trimpath pins and helper host/source guards pass. Build a
separate required-flags/source-bound read-only utility; it refuses host execution
with80. Networkless generation and independent check of180 help entries pass,
including explicit original/redacted/index options and retained output scope.
This supersedes M745's help-generation deferral only. See
[matching artifact pins](diagnostics/2026-11-19-recording-output-coverage-artifacts.json).

Build/preparation stay below48GiB with measured512/64MiB reserves and monitoring;
full post-preparation charge50,020,380,672. All source/build/preparation scopes
end. Fresh native output-summary/independent-frame verification remains required;
4c0b16e's earlier export evidence is not transferred to these later tools. No
production, existing ship, installed DEV or manual-review state changes.


## M750 — Bound the entire native wire status, preserve full private reports (2026-11-19)

The first5344c5c native output-summary COW06 remains failed as a whole: its full
status is interrupted/truncated at column4088 and the host parser refuses. M742
bounded only the product-proof subsection; input/build metadata still went to
serial. Preserve the ended root/input and independently read complete disk
reports before evaluating assertions. Do not promote any console fragment.

A shared pure Python protocol now emits only exact source/proof hashes, bounded
helper exit and allowlisted boolean gates, with an enforced2048-byte maximum.
One os.write emits the entire marker under4096bytes. Full inputs, private proof,
full report and detailed errors remain on disk; no evidence is discarded. Host
parsing requires the exact kind/source/proof/gates and refuses legacy, foreign,
incomplete, extra-field, duplicate or fragmented success. Actual process/backing
completion is saved before result parsing. Stage/hash the new status module with
the explicit Python harness allowlist; Go products/helper/PID1 remain frozen.

Fifteen inert guards pass, including multiple-MiB private input/error fixtures,
whole-message bounds, false/missing gates, typed values and real log framing.
See [failure and correction](diagnostics/2026-11-19-recording-output-status-failure.json).
Independent COW06 proof reading and a fresh native retry are still required.
All production/manual DEV/user-ship exclusions remain; no installed change.


## M751 — Native output coverage and bounded whole-status retry pass (2026-11-19)

Supersede M748–750's native deferrals only for matching5344c5c closed retained
recordings, e0af PID1 and e2b545e status harness. Fresh COW07 passes actual CLI/
official-SDK output coverage, original/indexed/redacted exports, helper0/QEMU0 and
strict compact-status parsing with no interruption. Independent read-only kernel
exports execute no source-disk code. Python decodes170/32 original frames and
checks stream-only169/31frames(3149/1191bytes), first/last output sequence/time,
earliest/latest wall time, every index row/offset/chain, filters/cursors, ZIP/chunk
hashes, escaped spans and final coverage. New output fields change neither JSONL
index file bytes nor complete redacted manifests compared with prior4c0b16e.
See [native evidence and limits](diagnostics/2026-11-19-recording-output-coverage-native.json).

Both independent SQLite copies check ok; original controls/history/intent/rows,
raw files, roots/inputs/backings remain exact except one fresh observed_at and
revision75→76. COW06's complete retained proof independently passes too; its old
whole status remains failed. The corrected bounded serializer accepts that exact
full report without exposing its inputs. Fresh COW07 supplies whole completion.
All six exact native/reader scopes end empty and frozen artifacts/cases recheck.
Final full charge50,184,650,752 stays below48GiB.

This qualifies only the selected retained closed prefix. Host/session-wide oldest
inventory, active capture/pressure/age/retirement/other factories, remaining issue
tracks and final59/native-plugin/preserved DEV gates remain. Installed/manual DEV,
production, protected/existing ships and tmux remain unchanged. The completed
read-only overlays add no required state/version transition; compact exact SQL/
proof/archives/source/logs suffice, with immutable recording07 and original app/
recording transition fixtures kept for future runs under the user's cleanup rule.


## M752 — Consolidate completed output-read overlays under the user retention rule (2026-11-19)

COW06 remains the whole status-framing failure, with complete independently
verified product/SQL/frame proof and the whole-message correction. COW07 supplies
the corrected whole/native/independent pass. Keep every compact exact database,
recording/query/ZIP proof, source, binary and log. Neither copied closed-read
overlay adds a required state/version transition: immutable original recording07
retains the same source, and original app62, actual recording64 and current app65
full fixtures remain. Earlier failed/passed statuses remain historically unchanged.

Classify only their four image/input files(95,027,200allocated bytes), verify1481
retained pins and exact native/reader/supervisor scopes ended. Removal is not yet
applied. This follows the user's broader cleanup authorization and supersedes
whole-copy retention for these two exact disposable read overlays only. See
[notes and classification](diagnostics/2026-11-19-recording-output-cow-retirement.json).
Production/manual DEV/user/tmux ships remain excluded.


M752 execution update: the four classified output-read image/input files are
removed under the broader user authorization. All1481 retained pins match;
95,027,200allocated bytes are reclaimed. DISPOSED markers prevent old-case reuse.
Full charge50,090,250,240 stays below48GiB. Original app62/recording64 and current
app65/recording65 full transition fixtures, every exact compact store/proof/source/
binary/log, and all production/manual DEV/user-ship exclusions remain preserved.


## M753 — Keep project-action dependencies through both native observation boundaries (2026-11-19)

Prepare the remaining native issue16 transition gate by factoring the existing
public action through a private dependency-taking implementation. The public
entry supplies ordinary defaults; the same dependencies reach initial and
post-Stop snapshots. All existing ownership/process/pier/namespace checks and
manager save callbacks remain. No public override, fixture signal mechanism,
format/schema65 or deployment change is introduced at this checkpoint.

Focused races(4.930s) and vet pass. The test verifies serialized stop-dispatched
intent binds the original operation/full container/StartedAt before the single
signal, save refusal/cancellation dispatch nothing, a changed process after the
signal refuses completion without repeating it, and unselected peer bytes stay
exact. Existing real observer transition/mismatch and read-only retained recovery
source tests also pass. [Evidence](diagnostics/2026-11-19-project-action-dependencies-source.json)
records50,127,073,280-byte full post-charge under48GiB.

A guarded qualification-only helper and matching native injection remain pending.
Do not claim this source refactor proves the actual incident predicate or transfer
5344c5c's older recording qualification to later artifacts. Production/manual DEV,
existing/protected/tmux ships and needed transition fixtures remain untouched.


## M754 — Guarded qualification-only native Stop timing helper source passes (2026-11-20)

Prepare the real-engine injection without a product fault switch. Only the
explicit `omabit_disposable_qualification` build tag includes the guarded runtime
helper and test-manager backend. Ordinary Go file selection proves the runtime
hook absent. The test backend preserves real queued operation/save/terminal
receipt behavior, refusing other projects/members/actions. Complete VM/source/
uid/home/runtime/required-flag guards precede opening a store/socket; only two
fixed newly created fake members in the current65 fixture are accepted.

The test-only interceptor binds the durable exact original Stop intent before
holding the selected process with SIGSTOP and delivering one real SIGTERM. It
captures the real preliminary running response, rechecks process identity before
SIGCONT, waits for exact clean exit, then returns the unchanged earlier response
so the real bracket observes exited. Fresh bounded failure cleanup releases only
the same process; changed processes, other mutations/execs and repeated SIGTERM
refuse. No ownership/pier/container/namespace validation is removed. This is a
fixture breakpoint, not a proposed runtime pause feature.

Tagged runtime races(1.834s), manager refusal/host-skip races(1.015s) and both-package
vet pass with inert engines. Coverage includes lost TERM/CONT replies, inspection
failure, replacement, cancellation, trace failure, journal refusal and peer
preservation. The first read-only file-list check fails its Go template quoting;
retain it and use the corrected range/println form. Corrected ordinary/tagged
selection passes with ambient GOFLAGS cleared. CI adds these inert safety/vet and
ordinary omission gates. [Source evidence](diagnostics/2026-11-20-project-stop-native-timing-helper-source.json)
records50,273,107,968-byte post-charge below48GiB.

No native timing fault has run. A fresh launcher/guest/separately frozen PID1,
matching normal products plus explicitly tagged/source-pinned helper, storage
admission and independent actual native receipt/pier/peer checks remain required.
Do not claim this proves the original incident predicate. Needed transition
fixtures and all production/manual DEV/user/tmux ship exclusions remain intact.


## M755 — Prepare isolated native project-transition capsule (2026-11-20)

Ten inert launcher/guest guards pass for no-run/host/foreign-case refusal, frozen
source boundary, ordinary-versus-tagged products, bounded input inflation and
whole serial status. The fresh offline COW has a distinct current-app backing
name, with exact original pending-app/pristine dependencies read-only. It creates
two new fake members and preserves the inherited stopped app/pier/journals. The
case requires actual Stop terminal success, six exact timing traces, one TERM,
lookup-only receipt replay after reopen, same-pier Start, peer/anchor preservation
and final clean stops. See [capsule scope](../scripts/qualification/project-transition-cow/README.md).

No VM has run and initial boot pins deliberately refuse pending a separate clean
PID1/binary freeze. Matching products/tagged helper, storage admission and actual
native plus independent checks remain pending. The storage check requires the
host's established read-only DEV user-namespace authority; the sandbox-only read
cannot see that PID and correctly refuses. Full measured charge50,273,296,384
leaves insufficient space for the declared2GiB native scratch. Apply the user's
broader obsolete-test retention authority after lessons and needed transition
state are retained. Human-review disks containing credentials remain preserved,
as do all production/installed DEV/protected/tmux exclusions.


## M756 — Resume bounded0.5.0 release completion (2026-11-20)

The user authorized finishing the reviewed release plan. Continue on
release/0.5.0 from4856f15, preserving the original topic worktree and importing
its pending M755 changes through a hash-verified snapshot. The
[release checklist](release-0.5.0-plan.md) identifies finite delivery scope and
explicit remaining issue work; issue22 and production deployment stay excluded.

Review finds M755's predecessor path accidentally names a nonexistent transition
capsule. Correct it to the exact retained successful runtime COW05 and extend the
existing backing-chain regression to pin that input. All ten inert capsule guards
pass. Storage admission now counts existing build directories across registered
worktrees, preventing the handoff from omitting843,911,168bytes of retained tools.
All six admission tests pass, including the new cross-worktree/symlink case.
Full measured charge50,273,591,296bytes leaves insufficient native scratch; no
native transition test or deployment has run at this checkpoint.


## M757 — Preserve recording fixtures losslessly and freeze Stop qualification tools (2026-11-20)

Archive the ended recording07 root/input and recording02 input with full decoded
SHA256 verification before removing their uncompressed copies. Retain all compact
proof, original state and pristine dependencies; restore exact bytes under storage
admission before later read-only/COW use, never resume an ended capsule. This
reclaims1,220,263,936allocated bytes. The initial unsupported fuser option refused
before writes; a256MiB compression scope then failed by OOM with original intact.
Its incomplete output was discarded after rehashing the source, and the bounded
1GiB retry passed. [Archive record](diagnostics/2026-11-20-release050-fixture-archives.json)
preserves these distinctions and exact original/archive identities.

Clean68bf233 builds matching CLI/collector/runner, explicitly tagged test manager
and separately frozen boot helper. All4043 tracked input hashes remain unchanged;
ordinary builds omit the timing hook, source/host-refusal checks pass, and PID1
refuses host execution with80. [Artifact pins](diagnostics/2026-11-20-project-stop-native-artifacts.json)
record49,235,075,072charged bytes after the monitored build, under48GiB. The native
transition run and independent checks remain pending; no final release, installed
upgrade, preserved DEV or production qualification is claimed.


## M758 — Native Stop setup exhausts scratch before the transition (2026-11-20)

The first fresh transition capsule reaches its1920MiB root-file ceiling while
adding the second fake. Guest ext4 reports I/O errors; QEMU remains alive. End
only that owned disposable scope and record native exit-15, never a clean Stop
or transition pass. The first project member was created, but the intended
Stop interleaving was not reached. All original backing hashes remain exact.

A separately pinned host-refusing read-only kernel exports the ended SQLite,
WAL/SHM and eight command records without running any source-disk program.
SQLite integrity passes, and all23 inherited app/recording tables match the
original current65 fixture exactly. Preserve the compact database/log/source/
input evidence and [failed-case record](diagnostics/2026-11-20-project-stop-native-scratch-failure.json).
The failed root and its generated input image add no required transition fixture;
retire just those two images after verifying ended scopes, absent holders and
pins. Original app62/current65/recording fixtures remain intact.

Automatic approval review rejects the proposed lossless archive/removal of older
runtime-input trees02/04/06 as outside clearly authorized test-image retirement.
That script is not executed; all three directories remain untouched. Instead,
clear only the reproducible Go compilation cache, preserving module downloads,
frozen binaries and original inputs. A3GiB fresh scratch reservation supplies one
additional GiB over the failed setup. Add prompt hard-ceiling detection and a
regression so a live QEMU process cannot hide this setup failure until deadline.
All eleven inert guards pass. The fresh retry, exact native transition, final
stable upgrade, preserved DEV and release gates remain pending.


## M759 — Native transition passes; fix receipt-projection assumptions (2026-11-20)

Fresh transition COW02 with unchanged68bf233 tools completes the actual six-step
preliminary-running/bracketed-exited interleaving, one native TERM, clean exit,
terminal Stop success and helper0/QEMU0. Whole qualification fails at the next
harness comparison, so same-pier Start and final clean Stops remain unqualified.

Independent read-only extraction proves the actual post-restart lookup equals
the complete original saved SQLite operation exactly. The harness mistakenly
compares a ProjectOperationSummary with the full protocol.Operation: summary
omits project_id and message. Capture the full original stored receipt before
manager reopen, validate its project/summary binding and compare every replay
field against that saved receipt. Also correct the unfinished harness's expected
ordinary-instance count: project members are composed from project ledgers, and
the original instances table must remain at one, not grow to three. Preserve the
complete inherited projection unchanged. Save partial proof before later gates
so a subsequent failure retains its complete native observations.

All twelve inert guards pass, including actual summary/full-receipt shape and
foreign/missing/changed saved receipt refusals. Independent SQLite integrity,
six ordered operation/container/start bindings and all23 original app/recording
tables pass; the original stopped instance/runtime and backing hashes remain
unchanged. [Evidence and cleanup classification](diagnostics/2026-11-20-project-stop-native-replay-harness.json)
retains exact SQL, logs, traces and source/input pins. Under the existing test-image
retention authorization, retire only COW02's diagnosed images and redundant boot
archives after verifying ended scopes and absent holders. Needed original
app62/current65/recording transition fixtures remain intact. No product/runtime
binary changes or installed/DEV/production actions occur. A fresh complete retry
is still required; the historical incident predicate remains unproven.

The recording guide now leads with the dated current candidate contract and
labels earlier source-only sections as historical checkpoints. It preserves the
explicit pilot/default-off behavior, incomplete-capture limits and remaining
final qualification gates rather than treating older milestones as current claims.


## M760 — Complete exact native Stop transition and independent verification (2026-11-20)

Fresh COW03 with frozen68bf233 tools and92e6880 harness passes all eight native
gates: six exact timing traces, one TERM, terminal Stop success, original full
receipt replay after reopen, same-pier Start with a distinct container, unchanged
peer/anchor processes, inherited stopped app preservation and final clean Stops.
A separately pinned read-only inspector exports complete proof/SQL/traces without
executing source-disk programs. Independent checks confirm original receipt,
mounts, signals, final state and all23 inherited app/recording tables. Only normal
ordinary-instance observed_at/runtime_revision refresh differs; identity, intent,
configuration and runtime binding remain exact. The initial verifier's stronger
whole-row assumption is corrected explicitly, not treated as a product failure.

The first admission attempt refuses before creating an output. Losslessly archive
four superseded generated qualification executables, verifying full decoded hashes
before removing raw copies, to reclaim48,529,408bytes. Preserve original worktree
source/pending changes and all historical executable bytes. The fresh native
admission fits48GiB with3GiB scratch plus64MiB overhead. After independent success,
classify only the ended COW03 root/payload and redundant boot archives for removal;
retain exact compact evidence and original app62/current65/recording fixtures.
[Full evidence and classification](diagnostics/2026-11-20-project-stop-native-complete.json)
records identities, limits and cleanup.

This supersedes M759's whole-case deferral for this exact development Stop case.
It does not prove the historical incident predicate or qualify the final combined
0.5.0 artifacts, stable59 upgrade, preserved DEV or production deployment.

M760 cleanup execution: the five classified ended images/boot copies are removed
after scope/holder/hash checks, reclaiming1,706,594,304allocated bytes. Full charge
48,266,706,944bytes remains below48GiB. Compact evidence and all needed original
transition fixtures remain.

## M761 — Expose bounded Grubbery source/restart compatibility (2026-11-20)

Replace the stale unconditional missing-workflow-adapter result with the explicitly
scoped versioned source/restart contract. Require both Grubbery source/recovery
adapters and their independent receipt APIs alongside desk effects, operator role
and exact current target policy. Old or partial runners still refuse before
app-specific creation. The result includes workflow_scope; this does not claim
peer/permit/route/asset readiness or authorize a native effect. Align packaged
guidance with this finite release scope.

Focused selected-runner race cases and vet pass, including each missing API or
adapter, read-only behavior, current-generation policy denial and Gall independence.
Twelve inert native Stop guards, packaged skill validation and shared-guidance
synchronization also pass. Two initial test invocations fail at Unix socket bind
because their GOTMPDIR makes paths too long; a short fully counted Go test directory
resolves the environment problem. [Evidence](diagnostics/2026-11-20-grubbery-bounded-preflight-source.json)
records48,523,718,656bytes after testing. Matching final artifacts and native/DEV
qualification remain required. Correct the PR template's superseded minor-only
wording to the existing release/X.Y.Z and tested-upgrade policy.


## M762 — Prepare bounded native recording pressure qualification (2026-11-20)

Prepare a separate fresh offline capsule for the remaining recording pilot gates.
Install the matching candidate, create one new fake, and exercise three same-pier
sessions: bounded synthetic binary-line/stderr output through the native Docker
pipes, an encoded1MiB session ceiling, and actual free space below a512MiB reserve
on a640MiB guest-only recording filesystem. Job/state/pier storage remains separate
so incomplete capture can persist its explicit result. Require unchanged native
process identity, saved original prefixes, ordinary Stop/drain, no implicit capture
resume after freeing space, and retained-job capacity refusal without eviction.

Six inert capsule guards and eight existing recording-harness guards pass; focused
capture/refusal races and the default-skipped native observer pass. An independent
bounded raw decoder verifies segment/frame identity, links, hashes and timing,
including corruption/torn-tail refusal. Fix an inert expected-length arithmetic
typo before the clean guard pass. [Source evidence](diagnostics/2026-11-20-recording-pressure-capsule-source.json)
records48,698,540,032charged bytes. No native pressure VM has run.

The launcher reserves2304MiB with a1664MiB hard root-file ceiling, checks complete
aggregate storage, and retains the one-runtime/no-concurrent-build boundary. The
existing read-only observer accepts only this additional exact frozen scenario;
no product fault switch or runtime permission is added. Raw tmpfs evidence is
copied into bounded private output before shutdown, including failed cases.
Matching clean artifacts, native completion and independent extraction are next.


## M763 — Native recording pressure passes selected gates; correct capacity observation check (2026-11-20)

Frozen c88784c tools complete all three native recording sessions: exact1.5MiB
original line and64 stderr messages, encoded1MiB ceiling, and actual separate
filesystem free space below512MiB. Captures close durably; limit failure retains a
gap, free-space refusal explicitly reports an unsaved gap, and each native process
continues until ordinary Stop. Earlier bytes remain immutable. Independent
read-only extraction/raw decoding matches all original segments, metadata, hashes
and saved jobs. The content-free writer.lock is represented by its empty-file
hash in the saved proof rather than exported.

The whole case fails at the final capacity assertion. The original failed Start
receipt correctly reports exhausted recording admission; exactly three jobs
remain. Generic failed-Start completion publishes unknown/recovery_required with
the requested running intent until observation/cleanup. The harness incorrectly
requires its public row to already say stopped as proof of no new process. Compare
actual full container inventory and immutable job inventory, bind the exact saved
capacity refusal, then perform the already planned explicit final Stop. Seven
inert guards pass, including foreign receipt/new-job/new-container refusals.
No product behavior is changed to satisfy the harness.

[Exact failed-case evidence](diagnostics/2026-11-20-recording-pressure-native-first.json)
retains compact SQL/raw/job/command/source proof. Classify only this ended
disposable root/payload and redundant boot copies for retirement under the user's
existing rule; original transition fixtures remain. A fresh whole pass, final
combined release, stable59 upgrade and preserved DEV remain required.

M763 cleanup execution: five classified completed/failed-case images and redundant
boot copies are removed after evidence/scope/holder/pin checks, reclaiming
1,563,996,160allocated bytes. Full charge49,105,166,336 remains below48GiB. Exact
compact recording/job/SQL evidence and every original transition fixture remain.

## M764 — Strengthen final stable upgrade rehearsal before artifact freeze (2026-11-20)

The inherited release-upgrade launcher counted evidence roots but omitted retained
worktree builds/caches. Reuse the complete issues-goal charge, acquire its shared
runtime lock, select a separately frozen host-refusing boot helper, and remove
in-launch compilation. Require clean candidate ancestry and the release-policy
post-freeze allowlist. Keep4GiB scratch with an explicit hard root-file ceiling,
one CPU and no swap. Nothing is launched by these edits.

Add an explicit app-workflow option independent of Theme/DMS. The previous stable
CLI creates a source desk plus policy/settings/receipt history before upgrade.
Verify original table DDL and row digests through actual migration and manager
restart, then use a distinct candidate desk. Close SQLite read handles explicitly,
including errors. Three inert guards pass for no-run, wrong boot-image refusal,
original table selection/digest changes and connection closure. Actual final
0.4.0/schema59 to0.5.0/schema65 qualification remains pending.

## M765 — Native capacity refusal preserves jobs; correct admission-lock counting (2026-11-20)

The second pressure case again passes all three capture/pressure cases but fails
its whole-case inventory assertion. Independent read-only extraction confirms
exact original recordings and three saved jobs, the original failed capacity
receipt, and empty complete Docker inventories before and after that Start.
The harness counts every recording-jobs entry, including the writer's empty
admission.lock, as a job. This is a fixture assertion defect, not evidence of a
fourth job or a new container. No product code changes to satisfy the test.

Inventory the lock separately and require every saved job-file hash to remain
unchanged. Save before/after inventories before asserting, so failures retain the
exact inputs. Eight inert guards pass, including the actual three-directory plus
lock layout, changed job bytes and malformed/symlink lock refusal. The final Stop
and commitment check are still pending a fresh whole pass. The second case remains
failed; [independent evidence](diagnostics/2026-11-20-recording-pressure-native-second.json)
preserves its exact compact proof and classified ended-image retention. Original
transition fixtures and installed/manual-review state remain unchanged.

## M766 — Prepare0.5.0 component versions and finite support claims (2026-11-20)

Set root, component and UI versions to0.5.0. Add candidate release notes with the
explicit intended stable0.4/schema59→65 window and unchanged default-off recording
service behavior. Mark named-client recipes and local HTTP experimental; retain
catalog exclusions and the finite app/source/restart scope. No issue-wide closure,
production deployment or final qualification follows from a version bump.

Native plugin, skill packaging/shared guidance, release-policy, JavaScript models
and snapshot installer fixtures pass. [Preparation record](diagnostics/2026-11-20-release050-scope-preparation.json)
binds the reviewed inputs and limits. Existing0.4 distribution pins remain until
matching frozen assets and upgrade evidence pass. The final source/race/QML,
packaging/native workflows, preserved DEV and reviewed publication gates remain.

## M767 — Preserve fail-closed storage accounting during runtime qualification (2026-11-20)

The third recording pressure attempt is interrupted by complete-storage accounting
when a concurrently run snapshot-installer fixture removes its temporary output
between root enumeration and measurement. The monitor correctly rejects the
incomplete measurement and terminates only the disposable VM. This is an experiment
scheduling error; no product/pressure pass is inferred. Keep all temporary-output
fixture suites, as well as Go builds, idle during a VM. Do not weaken measurement
to bypass disappearing paths.

[Exact interruption](diagnostics/2026-11-20-recording-pressure-interruption.json)
retains host/source/log evidence. Retire only its four classified fresh-fake
images/boot copies after scope/holder/backing checks, reclaiming1,547,980,800bytes.
No original app/recording transition fixture is removed; no successful tmpfs
recording export is claimed. Standard Go compilation-cache cleanup separately
reclaims789,917,696bytes while preserving module sources and every frozen tool.
A new whole run remains required with no concurrent fixture activity.

## M768 — Complete the recording job-layout regression and require measured storage (2026-11-20)

The fourth case reaches the inventory check after all three native capture cases,
but the new strict file allowlist omits each job's empty worker.lock. This is
another harness defect: the independent content export intentionally omitted
empty lock files, so copying its JSON-only inventory was insufficient. Review the
actual job writer and usage inventory; a closed job has five JSON files plus
worker.lock. Require that complete six-file set, an empty regular lock and exact
hashes of every entry. The regression now models the real complete closed layout,
including malformed locks. Nine inert guards pass.

[Fourth-case evidence](diagnostics/2026-11-20-recording-pressure-native-fourth.json)
independently verifies all original recording content, three jobs and the earlier
ordinary Stops. Capacity/final Stop did not run; the whole case remains failed.
Retire only its classified ended images after preserving compact evidence and
checking original fixture hashes. The next native case must complete with final
matching0.5.0 artifacts rather than promoting any partial pass.

The shared issues-goal budget now rejects the older storage helper's estimated
unreadable-directory fallback: every DEV namespace measurement must complete.
This enforces the user's current no-incomplete-measurement rule without altering
the older general launcher's contract. The new regression refuses estimated
admission before passing any data to the budget calculator.

Separately,40 unused public Go module extracts match every byte in their retained
original ZIPs and those ZIPs' h1 content hashes. Removing only these reproducible
extracts reclaims1,893,773,312bytes; current/previous-stable declared dependencies,
original ZIP archives and all frozen executables remain. Exact inventory and
verification are retained in /tmp/omabit-050-handoff/module-cache-retirement-result.json.

## M769 — Freeze0.5.0 assets and pass desktop/interface qualification (2026-11-20)

Freeze product/UI/skill/packaging inputs at bdf43978e1d0ea858c3349b787e2ff8eaa1252ad.
The isolated Go1.27.3 packager builds all six0.5.0 assets, schema65 and the unchanged
pinned Recommended Apps pill. Every asset hash matches its manifest. Plain and
qualification-tagged helpers carry that source identity; three separately pinned
read-only inspectors refuse host execution. No artifact is rebuilt after pinning.

Actual0.4.0→0.5.0 desktop activation in a private compositor preserves preferences
and recovers from a synthetic failed candidate. Private Recordings UI, persistent
Agent Help and old two-skill compatibility pass. Full selected service/MCP/guidance/
CLI source-race suites pass, as do all180 networkless help entries. Separately,
actual packaged CLI/collector and official-SDK stdio pass three lost-reply/exact
saved-lookup/reopen cases against real managed API/SQLite with inert native
observations. These distinct evidence types are recorded explicitly in
[frozen validation](diagnostics/2026-11-20-release050-frozen-validation.json).

Full release-head CI and matching native Stop/app/recording/upgrade cases remain
gates. The final stable59→65 service rehearsal, preserved DEV and protected
review/publication have not occurred. The existing production and DEV services,
plugins, piers, preferences and history remain unchanged.

## M770 — Final0.5.0 native recording pressure and capacity gate passes (2026-11-20)

The fifth whole case uses the actual frozen bdf4397 release binaries and passes
all nine gates. Independently decode all three original session exports and hash
every content file plus all five JSON files per job. SQLite preserves the exact
failed capacity receipt; complete native inventories remain empty, all three jobs
remain byte-identical, and the final ordinary Stop leaves desired/observed stopped.
Commitments remain17MiB for three retained jobs. Empty lock hashes are guest-proof
evidence, explicitly distinguished from independently exported content.

[Final pressure evidence](diagnostics/2026-11-20-recording-pressure-final.json)
qualifies finite synthetic burst/long-line Docker transport, a1MiB session ceiling,
and actual isolated-filesystem free-space refusal with explicit incomplete coverage.
It does not establish automatic retirement, age enforcement, host-wide pressure
recovery, other factories or active-worker upgrades. M763/M765/M768 partial cases
remain failed; this fresh whole pass supersedes their pending retry requirement.

Retire only five classified ended images/boot copies after independent verification,
no-holder/scope checks and unchanged original backing hashes, reclaiming1,611,563,008
allocated bytes. Full aggregate charge46,739,451,904bytes remains below48GiB.
Original app/recording transition fixtures, compact evidence and all installed/DEV
state remain. Final native Stop/app/export and stable59→65/DEV/release gates remain.

## M771 — Final0.5.0 native Stop transition and preservation pass (2026-11-20)

Fresh COW04 uses the actual frozen bdf4397 release binaries and matching tagged
helper. All eight whole-case gates pass, followed by clean guest/host completion.
Independent read-only exports verify six exact ordered timing bindings, one TERM,
the exact saved original Stop receipt after reopen, same-pier Start in a distinct
container, unchanged peer/anchor processes and final clean Stops. All23 inherited
app/recording tables remain exact; the original ordinary instance differs only by
its expected observation timestamp/runtime revision refresh.

[Final Stop evidence](diagnostics/2026-11-20-project-stop-native-final.json)
supersedes M760's final-artifact retry requirement for this exact interleaving.
It does not establish the historical incident's exact predicate or qualify the
stable59 upgrade, preserved DEV or publication. The first invocation refused an
incorrect runtime input path before creating any output; the verified retained
runtime path was then used. No product/harness edits were needed.

Retire five classified ended images/boot copies only after retaining independent
SQL/proof/traces/commands and checking empty scopes, holders and original hashes.
This reclaims1,723,985,920bytes; full charge46,756,741,120bytes is below48GiB.
Original app62/current65 and recording transition fixtures remain. Read-only DEV
inventory separately confirms the expected review runner identity and schema59;
no DEV update or ship action has occurred.

## M772 — Explicit user pause and reboot-safe preservation (2026-11-20)

The user paused to restart/update the machine. Gracefully power off only the
active disposable stable-upgrade guest through its own QMP socket; its scope is
inactive/empty and host exits0. No completed guest result exists, so the upgrade
remains interrupted/unqualified. Retain its whole partial capsule and pause receipt;
never resume that overlay. Production and both manual DEV installations are unchanged.

Because /tmp is volatile tmpfs, move the generated Omabit temporary files into
private durable evidence with a complete pre-move SHA256 manifest and verified
transfers, then independently verify all50,638 retained regular files. Ended
ephemeral sockets/pipes are listed separately. The full post-move charge is
46,809,341,952bytes, below48GiB. The original app/recording fixtures, tools, source
and compact evidence are preserved. [Resume checkpoint](release-0.5.0-resume.md)
records exact paths, final artifacts, completed gates, interrupted upgrade and
remaining work. Goal is paused at the user's request; no release is published.

## M773 — Resume, qualify candidate01 stable upgrade and correct two release inputs (2026-11-20)

The user explicitly resumed after the machine update, superseding M772's paused
status. Verify the preserved manifest and required input hashes, then launch a
fresh offline upgrade guest from the pristine base; never resume the interrupted
overlay. Actual published0.4/schema59 tools create source/policy/settings/receipt
history before candidate01 bdf4397 upgrades to65. The whole guest and host pass:
original app table rows/DDL survive migration and manager restart; running
container/engine, stopped intent, identities, access and history remain preserved;
authorization denial/retry and actual old-reader refusal leave state intact.
A separate candidate desk workflow and final ordinary Stop also pass.

[Exact candidate01 upgrade evidence](diagnostics/2026-11-20-release050-candidate01-upgrade.json)
includes independently extracted report, commands and SQLite. The exported report
matches the complete serial result; schema65, all13 original app table definitions,
all five original runtime-operation receipts and the final successful Stop are
verified independently. All three fresh instances finish with stopped intent.
Retain compact evidence from both capsules; the original pause remains interrupted
and unqualified. Retire only ten classified ended image/boot files after empty
scope, holder and backing checks, reclaiming3,329,003,520bytes. Original app62,
current65, recording fixtures and both manual DEV roots remain unchanged.

Review finds stale packaged guidance claiming the Grubbery adapter is missing.
Correct both synchronized references to the actual bounded source-and-restart
preflight contract; peer/permits/routes/assets remain separate. Seven packaging
tests, shared-reference synchronization and both skill validators pass.

The exact bdf4397 source CI run44792728375 passes every source/race/QML job. Its
duplicate run44792713306 fails only the64-recovery history test at93.05seconds
under a90-second context; the other run passes it in60.83seconds. Timing is
consistent with deadline expiration, but the old diagnostic did not preserve
context.Err(), so do not claim that cause proven. Extend only this test's context
to5minutes and report the iteration/context on failure, retaining all64 recoveries,
overflow refusal and exact row/latest-binding assertions. The focused race test
passes in66.32seconds. [Correction evidence](diagnostics/2026-11-20-release050-resumed-corrections.json)
retains both original logs and the local result.

These packaged skill/test changes require a new clean freeze and matching
artifacts/qualification. M769–M771 and this upgrade qualify bdf4397 only; they do
not qualify the next candidate. No runtime product behavior, installed DEV,
production service or distribution pin changes here. Final app/export/upgrade,
preserved DEV, required PR checks, protected merge and publication remain gates.

## M774 — Freeze candidate02 and pass matching desktop, interfaces and stable upgrade (2026-11-20)

Freeze corrected product/UI/skill/test/packaging source at
59d83aab49384d2f26531320bb5ab68e1257aeb5 and publish that checkpoint to draft PR23.
Build all six immutable0.5.0 assets in an isolated clean Git-blob export with
Go1.27.3, schema65 and the same pinned Recommended Apps pill. Backend source digest
is494115a0837095352b1c27dede8e5c65b63319ec08e76d093f633993308b1a13.
This supersedes M769's source selection for the intended release; retain all
candidate01 artifacts/evidence under their original identity.

[Candidate02 validation](diagnostics/2026-11-20-release050-candidate02-frozen-validation.json)
passes all asset hashes, matching plain/tagged helpers and host-refusing read-only
inspectors. Four full selected interface race suites and all180 offline help
entries pass. Three actual packaged CLI/collector and official-SDK lost-reply,
exact saved-lookup and manager-reopen cases pass with inert native observations.
Actual private desktop cold/hot activation, preferences, failed-candidate recovery,
Recordings UI, persistent Agent Help and legacy two-skill compatibility pass.

Fresh `patch-upgrade-050-final03` then passes the actual published0.4/schema59 to
candidate02/schema65 upgrade. [Final matching upgrade evidence](diagnostics/2026-11-20-release050-final-upgrade.json)
retains original stable app history/DDL through migration and manager restart,
running/stopped intent, original runtime/engine processes, identities, access,
authorization denial/retry, actual old-reader refusal unchanged, the candidate
desk workflow and final ordinary Stop. Independent read-only extraction matches
the full report and verifies schema65, all13 original app table definitions, all
five original runtime-operation receipts, final successful Stop and stopped intent
for all three fresh instances. No source-disk code executes in the inspector.

After complete evidence, empty-scope/holder and original-backing verification,
retire only five classified ended image/boot files, reclaiming2,481,082,368bytes.
Full charge47,193,726,976bytes remains below48GiB; original fixtures and all installed
state remain. Final matching native pressure, Stop, Grubbery and closed exports,
full release-head CI, preserved DEV and reviewed publication are still required.
No production deployment or issue-wide completion is implied.

## M775 — Candidate02 recording pressure and independent preservation pass (2026-11-20)

Fresh pressure06 uses the exact frozen59d83aa release binaries and matching helper.
All nine whole-case gates pass, including exact burst/long-line capture, the1MiB
session bound, actual isolated-filesystem pressure refusal, explicit incomplete
coverage, same-pier sessions, immutable earlier recordings, capacity refusal
without eviction and final clean Stop. Independent read-only extraction verifies
all three original recordings, three saved jobs totaling17MiB of commitments,
the exact saved failed-admission receipt, no new container and stopped intent.

[Candidate02 pressure evidence](diagnostics/2026-11-20-recording-pressure-candidate02.json)
supersedes M770's final-artifact retry requirement for this bounded transport.
It does not qualify automatic retirement, age enforcement, host-wide pressure
recovery, other factories or active-worker upgrades. Preserve these scope limits.

After complete compact evidence, empty-scope/holder and original-backing checks,
retire only five ended image/boot files, reclaiming1,611,763,712bytes. Full aggregate
charge47,209,918,464bytes is below48GiB. Durable original app62/current65/recording
fixtures and both manual DEV roots remain unchanged. Matching Grubbery, Stop,
closed exports, preserved DEV, required checks and reviewed publication remain.

## M776 — Candidate02 Grubbery recovery, two versions and restart pass (2026-11-20)

Fresh offline COW06 uses frozen59d83aa product binaries and its matching helper,
with original app62 as a read-only backing. The whole guest and host pass: original
uncertain-phase observation without redispatch, explicit audited replacement,
completion of original source through CLI, second-version source through MCP,
retained app state and changed code, same-pier replacement, manager restart,
original/audit receipt replay, one dispatch per phase and final clean Stop.

[Candidate02 Grubbery evidence](diagnostics/2026-11-20-grubbery-candidate02.json)
records the exact source and retained proof. The separate read-only reader exports
58 fixed files without executing source-disk code; all original images remain
unchanged. Independent SQLite/proof checks preserve the original header prefix,
all17 original events and pier binding. Current schema65 has two app operations,
63 events and zero leases; the saved runtime audit matches exactly and final
intent/observation are stopped. This resolves the matching final-artifact repeat
for the previously qualified12f case, not broader peers/permits/routes/assets or
process-death cleanup.

After complete evidence and ended-scope/holder/backing checks, retire only five
classified image/boot files, reclaiming345,944,064bytes. Full charge at this point
is48,996,470,784bytes under48GiB. Preserve both physical temporary backings until
the remaining Stop case has finished, and retain their durable originals.

Compiler provenance correction: M774's prose incorrectly names Go1.27.3 for
candidate02. Its immutable release.json, recorded validation and all three binary
build-info records identify Go1.27.4. That compiler statement is superseded here;
source59d83aa and all artifact hashes are unchanged. Older standalone boot helpers
keep their separately pinned source/compiler records. Matching Stop, closed
exports, preserved DEV and protected reviewed publication remain required.

## M777 — Candidate02 exact native Stop and independent preservation pass (2026-11-20)

Fresh transition05 uses frozen59d83aa products and its separately tagged helper.
All eight whole-case gates pass: the guarded native inspection transition, one
SIGTERM, successful terminal Stop, original receipt replay after manager restart,
ordinary same-pier Start, unchanged peer/anchor processes, inherited stopped app
preservation and clean final Stops. The host exits0 with no interruption.

[Candidate02 Stop evidence](diagnostics/2026-11-20-project-stop-candidate02.json)
supersedes M771's matching-artifact repeat requirement. A fixed read-only reader
exports11 files and leaves all source/backing images unchanged. Independent
verification checks six ordered traces with one original operation/container/start
binding, exact original full receipt equality, one SIGTERM, a different resumed
container on the same pier, unchanged peer/anchor state and23 inherited tables.
Only the original stopped app's permitted fresh observation fields change.
All three disposable fakes end stopped. This bounded interleaving does not establish
the historical incident's exact failing predicate.

To fit the declared3GiB plus64MiB reservation, first retire only the previously
authorized standard Go compiler cache,764,129,280bytes across5955 files, verifying
all named release artifacts/helpers unchanged. No Go build runs during the VM.
After independent evidence and ended-scope/holder checks, retire five ended Stop
image/boot files, reclaiming1,734,213,632bytes. Both COW consumers have now finished,
so retire the six verified temporary physical backing/metadata duplicates, freeing
another1,766,367,232bytes while rechecking every exact durable original.
[Retirement evidence](diagnostics/2026-11-20-native-input-retirement.json) retains
the classifications and full-record hashes. Aggregate charge46,487,318,528bytes
remains below48GiB. Final closed exports, preserved DEV, review/checks and release
publication remain; no production or manual-ship operation occurred.

## M778 — Candidate02 closed recording exports independently verified (2026-11-20)

Fresh closed-export08 uses the exact frozen59d83aa products and matching helper.
The whole host/guest case passes all ten product checks: original bytes/policy,
CLI/SDK measurements, original manifest/parts/ZIP, indexed ZIP, timing queries,
selected-prefix coverage and complete redacted manifest/parts/ZIP. Host exit0 has
no interruption. This is the final matching-artifact repeat of M751's bounded case.

[Candidate02 export evidence](diagnostics/2026-11-20-recording-exports-candidate02.json)
retains the proof. A fixed read-only inspector exports12 files without executing
source-disk code and leaves all source/backing/payload hashes unchanged. Independent
Python analysis compares all four retained files byte for byte against the original
recording backing, verifies ZIP manifests/chunks/modes and both complete frame
chains (170 and32 frames), and checks indexed/redacted content against the previous
qualified exports. Both SQLite copies pass; every original logical row remains,
except the one permitted observation refresh. No automatic retention, active-worker
upgrade, other runtime factory or broader incident behavior is inferred.

After proof and ended-scope/holder checks, retire only five ended scratch files,
reclaiming88,170,496bytes. Then remove the three verified temporary recording07
restoration duplicates, reclaiming1,268,256,768bytes, while rechecking the exact
durable compressed original and metadata. The [retirement record](diagnostics/2026-11-20-recording-input-retirement.json)
retains the receipt hash and original pins. Full aggregate charge46,503,624,704bytes
remains under48GiB. All original transition fixtures and manual DEV state remain.
Preserved DEV, final release pins/checks/review and publication are still required.

## M779 — Preserved DEV manager and matching desktop update pass (2026-11-20)

After M774–M778 pass on the exact candidate, update only review040 under the
user's DEV authorization. The original manual `review-zod` already has desired
running, unknown observation and recovery-required with a stopped exit1 container.
Preserve this state; do not repair, start or use that ship as a test fixture.

The [DEV upgrade record](diagnostics/2026-11-20-preserved-dev050-upgrade.json)
binds exact59d83aa helpers and schema59→65. Stop only management, retain complete
offline state, old binaries and desktop preferences, install matching tools and
start only management without resume-running. Verify all45 original non-instance
tables, original configuration/intent/recovery/history, engine/container state,
owner, access registry and pier object identity unchanged. The actual dedicated
engine server reports29.9.2. The old backup remains in
`/srv/dev/urbit-040/state/upgrade-050-20261120`.

Two conservative backup guards exposed harness assumptions: the DEV parent is
root-owned, so the first mkdir refused before mutation; the second attempt found
original empty Docker-config directories after manager stop. Preserve those
folders and continue the exact saved checkpoint with original binaries/runtime
verified, rather than rerunning or resetting state. These are qualification-driver
corrections; frozen product artifacts did not change.

Complete UI staging retains the inner bundle, three helper hashes and skill pins,
with only the isolated review wrapper/DEV button overlay. Normal rescan keeps the
old keepLoaded service. A targeted DEV disable/enable destroys it, but Quickshell
then reports File name case mismatch for newly introduced filenames at the cached
old path. Move the complete candidate atomically to the fresh versioned directory
`~/.config/omarchy/plugins/thelifeandtimes.omabit.review040-v050`, keeping the same
review040 plugin ID, IPC, profile and state. It loads0.5.0 with the exact matching
helper paths and selected original runner. No shell restart occurs; the only
normalization added by the shell toggle (an empty tray key) is reverted under an
exact-value guard, preserving original shell configuration bytes. The original
plugin tree remains outside the watched directory in the upgrade evidence root.

This supersedes M575's installed-version/path statement only. Manual state remains
protected. A new private declared-API client was built after every VM ended under
a2GiB scratch admission; no release artifact was rebuilt. Fresh app creation,
scoped app-policy decisions and workflow evidence are still pending. Production,
protected ships and the earlier DEV namespace remain excluded.

## M780 — Fresh DEV app, closed permissions and preserved restart pass (2026-11-20)

The [fresh DEV workflow](diagnostics/2026-11-20-dev050-app-workflow.json) passes
with exact frozen59d83aa installed tools. Create only `release050-app-zod`, then
use the user's explicit temporary approval for Create on that new ship and Commit
only on its `releasecheck` desk. Publish remains blocked and unrelated policy
unchanged. Observe desk creation, staged sample source, commit and local activation
through the saved original operations; never substitute a retry request.

The declared Gall sample APIs pass authenticated watch/poke, duplicate/nontext
refusal and state checks. A source reload adds an on-load marker and preserves the
first entry and session. Close Commit and Create grants at policy revisions3/4
before ordinary Stop/Start. The new container uses the same pier, the original
Stop receipt replays exactly, and the same session retains the first entry and
accepts a second. Revoke the test session and finish with a clean exit0 Stop.
The fresh pier/history remain; its project anchor is unchanged.

Independent final observations preserve all eight original manual operation
receipts, original manual intent/configuration/recovery, engine invocation,
container state and pier object identity. The existing `review-zod` stays in its
original running-intent/unknown/recovery-required state with its stopped exit1
container; no repair or lifecycle action targets it. Retain427 exact private
evidence files after removing credentials. M779's pending-app statement is now
superseded by this bounded pass; broader app capabilities remain excluded.

Storage accounting correction: the frozen generic enumerator lists the old DEV
plugin path and missed the fresh versioned directory after M779's move. Explicitly
measure and add its71,323,648bytes for all remaining admissions; earlier post-move
totals undercounted that amount. Corrected measurement remains within the original
scratch reservation and48GiB cap, with final charge47,809,650,688bytes. Earlier VM
qualification predates the move. This changes only the private qualification
adapter, not frozen product inputs. Draft verification, final pins/checks/review,
protected merge and publication remain gates. Production is unchanged.

## M781 — Exact0.5.0 draft downloads and distribution pins verified (2026-11-20)

Create draft release453321091 at frozen59d83aa with the six original assets,
original release.json/SHA256SUMS and the completed qualification record. Authenticated
downloads match every local size/hash and GitHub SHA256 digest. No asset is rebuilt
or replaced. The public tag lookup returns404 for the unpublished draft; continue
verification using the unique saved release ID, without repeating creation.

[Draft verification](diagnostics/2026-11-20-release050-draft-downloads.json) binds
all nine assets. Latest stable remains0.4.0. Install only the matching distribution
pins on release/0.5.0; all post-freeze changes are documentation or those pins.
Complete corrected storage charge47,976,513,536bytes stays below48GiB.
Final trusted release-policy/source/QML checks and review, protected merge-commit
publication, public downloads and private cold native startup remain required.
No production deployment or issue-wide completion is claimed.

## M782 — Issue audit and per-ship runtime profile source review (2026-11-20)

The owner requested an issue audit against published0.5.0, issue#22, newcomer
documentation and a later safe manual production review update. No permission
was given to install this candidate or operate on live ships. Work starts from
origin/master ca5f788 on topic codex/issue-22-runtime-profiles; master is unchanged.
Use the0.6.0 minor line for the coordinated runner/collector/schema change.

Close#16 using the bounded native Stop/recovery/receipt/same-pier Start evidence.
Leave#17–20 open for their broader acceptance. Reopen#21 after correcting its
initial overbroad closure: copied pending-Grubbery recovery and a fresh Gall
sample do not establish its fresh Grubbery bootstrap requirement. Keep#22 open.
See [the issue audit](issues-0.5.0-audit.md).

The candidate retains finite immutable per-ship runtime profiles through ordinary
managed imports, starts, recovery, maintenance, recording helpers and signed
handoffs. Omitted legacy metadata remains the exact historical official runtime.
The reviewed#1145 backport is separately staged and verified; no mutable image,
arbitrary command or default fallback is accepted. Schema66 adds an older-reader
fence without rewriting existing ship rows. A change to an already enrolled
ship's runtime is explicitly refused pending a separately qualified transition.

Package the exact approved native binary as a deterministic Docker/OCI archive
without executing it or contacting Docker. Rebuilding from verified cached base
blobs reproduces archive SHA256
c8a412c42d53fa0c9a8c97aa55a580ddf2864bbdf32f80074dfb6bdfd837f510.
The [source evidence](diagnostics/2026-11-20-runtime-profiles-source-review.json)
records source, focused race, preflight boundary, UI and packaging fixtures.
The broad historical race matrix timed out after8minutes; its ordinary test run
passed. Neither result is native runtime or service-upgrade qualification.

The [review guide](runtime-profiles-review.md) records exact provenance and the
owner-operated update/import prerequisites, including preserved state, stopped
native ownership, loom34, independent resource/network settings and replay
hazards. README now explains Urbit, the components, default Agent Help, CLI/MCP,
shipped skills and development entry points. No review installation is offered
as production-ready until real disposable runtime and published0.5.0 upgrade
gates pass. Protected ships, tmux, production services and manual DEV are untouched.

The existing experiment charge was45,098,889,216bytes before this task's build
outputs. The normal limit is40GiB; the earlier48GiB exception covers issues16–21
only. An extension for#22 has been requested but is not yet approved. Do not run
dependent runtime qualification or treat elapsed time as approval. Preserve all
existing manual DEV and retained ship state. Source builds and inert fixtures
are not a substitute for this pending native evidence.

## M783 — Frozen0.6.0-rc.1 package and corrected qualification admission (2026-11-20)

Commit94bc33f4f69d3c257d3beca8ce13f5050e6b3dd4 freezes the issue#22 source review.
Build the coordinated seven-asset review set into
`/tmp/omabit-runtime-profiles-review-assets/0.6.0-rc.1`, including the reviewed
runtime archive. Exact-source build stamps, checksums, offline helper smoke,
isolated installer replacement and customization refusal pass. Retire only this
task's3,548,065,792-byte regenerable Go build cache with a manifest. No candidate
installation, runtime execution, Docker load or ship access occurs. Native pins
remain stable0.5.0; the topic checkout is not a working production update route.
The [frozen package evidence](diagnostics/2026-11-20-runtime-profiles-frozen-package.json)
identifies the exact local outputs and remaining qualification gates.

Correct M782's storage subtotal: full accounting also charges shared Go caches
and build outputs in older T3 worktrees. Add the documented versioned manual DEV
plugin directory explicitly. Total48,533,643,264bytes (about45.2GiB) means a4GiB
upgrade rehearsal requires at least1,289,003,008bytes reclaimed even under48GiB.
Inventory247 files in the two prior-goal build directories with hashes; no files
there are removed. These include generated source/logs as well as binaries, so
never delete the directories wholesale. Preserve source, compact evidence,
required immutable inputs and manual-review state. Approval to extend the
previous48GiB exception to#22 remains pending; no native qualification is admitted.

## M784 — Issue22 budget authorized; complete source CI passed (2026-11-21)

The user explicitly answers the pending request: “Allow 48 GiB for issue #22”. This
supersedes the pending-authorization statements in M782/M783. Keep one disposable
runtime experiment active; remeasure the complete charge, declare scratch, and
preserve all existing ships and manual DEV state. Reclaim only verified redundant
generated outputs with a manifest; never remove the older build directories
wholesale because they also contain source and compact evidence.

PR#24 at f4cd2ee passes core, all four managed-race shards, QML and the aggregate
source gate in GitHub run44882786180. This supplies the full source/race result
that the earlier locally time-limited historical run could not establish.
The protected release-policy check correctly refuses this unqualified rc topic;
do not bypass it. Native runtime and published0.5.0 service-upgrade evidence,
matching distribution pins and owner-operated installation guidance remain gates.

## M785 — Redundant build outputs retired; upgrade scratch admitted (2026-11-21)

Revalidate hashes, regular-file ownership and exact paths before unlinking97
classified obsolete Go executables, compressed executables, one superseded
candidate package and byte-identical published-release download duplicates.
Preserve all directories, scripts, source, logs and manifests, final0.5.0
candidate02/helper02, the separately frozen PID1, immutable OS/runtime/pill
inputs, public0.5.0 and frozen0.6.0-rc.1 assets, and all ship/manual DEV state.
The [complete retirement and admission record](diagnostics/2026-11-21-runtime-profiles-build-retirement.json)
binds every removed path and retained duplicate. No runtime process is touched.

The corrected full charge is47,162,642,432bytes; reserving4GiB produces
51,457,609,728bytes under the approved51,539,607,552-byte cap. Remeasure at launch
and throughout the guest; the earlier incomplete enumerator must additionally
charge the versioned manual DEV plugin. Use one fresh isolated guest and retain
compact evidence before retiring only its newly generated outputs. Admission
does not qualify an artifact or authorize a live installation.

## M786 — Published0.5.0 upgrade passed; console-report failure recovered (2026-11-21)

The frozen candidate completes the native QML service-upgrade rehearsal in one
fresh offline VM. Schema65→66 preserves original ship identities, configuration,
pier paths, operation history, access, engine/container processes and original
app history (19tables,38rows). Manager reopen preserves runtime; the genuine
published0.5.0 writer refuses schema66 without changing state. The guest then
performs app work and stopped disposable-pier import and gracefully stops its
remaining fake. No production, protected ship or manual DEV is inspected.

The VM exits0, but the host's long JSON console line is incomplete and parsing
fails. Preserve this harness failure rather than reporting the launcher passed.
A separately frozen guarded reader mounts the ended VM and input disks read-only,
exports the full report, commands and SQLite database, and verifies unchanged
source/base hashes. Independent database checks confirm schema66, all5original
operation records, both original ship identities/configuration/paths,19table
schemas and all3disposable instances stopped. See the
[upgrade evidence](diagnostics/2026-11-21-runtime-profiles-upgrade.json).

After verifying both transient scopes inactive, retain all compact evidence and
retire77 newly generated VM disk/boot-copy files with checksum/inode guards and
`/tmp/omabit-runtime-profile-build/upgrade-generated-retirement.json`.
Retired allocation is2,398,052,352bytes; complete charge is47,167,893,504bytes,
leaving declared4GiB scratch within the approved48GiB ceiling. Immutable bases,
inputs, originals and all existing ships remain untouched. This supersedes only
the pending upgrade gate in M784, not the pending native patched-runtime gate.

## M787 — Native review exposes Docker image-store lookup defect (2026-11-21)

Frozen0.6.0-rc.1 is not installable for the reviewed custom profile: Docker29's
containerd image store loads the exact archive but cannot inspect it by its config
digest. An independent fresh diagnostic VM confirms both stock and patched image
lookups succeed by manifest digest and fail by config digest. Both container
binary checksums and banners match. Missing archive Start and corrupt archive
installation refuse before runtime execution. See the
[bounded diagnostic](diagnostics/2026-11-21-runtime-profiles-image-store.json).

The first private harness incorrectly required missing-artifact registration to
fail; registering a stopped, uninitialized selection is allowed. Correct that
assertion to require Start refusal and no container/pier execution. Preserve the
failed harness record. Diagnostic eval probes are not decoder evidence: Ivory
lacks the de:json gate, and the guest's default virtual-memory policy refuses a
16GiB mapping. Process exit0 on a Hoon %exit is not success.

Fix lookup to try only the exact reviewed manifest and config IDs, preserving
archive, returned identity, platform and executable verification. Do not retry
successful inspection with mismatched metadata. Add distinct-store and
cancellation regressions. The product change supersedes M783's rc.1 source freeze;
prepare0.6.0-rc.2 from a new clean source and repeat native runtime and upgrade
qualification. The earlier upgrade evidence remains historical, not evidence for
an unbuilt successor. All protected ships and installed services remain untouched.

## M788 — Image-store correction passes source and race checks (2026-11-21)

The full runtimequalify package passes (70.795seconds), including exact-archive
profile routing/staging, after correcting the fixture's private directories and
using a short temporary root for Unix-domain socket tests. Focused runtime-profile
race checks pass (1.615seconds). Native bootstrap9tests and packaging-provenance
7tests pass;2full-rebuild fixtures remain deliberately skipped because the actual
candidate build and smoke are separate gates. Preserve the earlier sandbox and
fixture failures; they are not product qualification results.

Prepare0.6.0-rc.2 with the native image-store correction, matching UI/component
version and current bundle installation documentation. Only exact selected
manifest/config identifiers are tried. Successful malformed image metadata never
falls through to another lookup, and cancellation stops further inspection.
Existing official references and all archive/platform/binary checks are retained.
Native lifecycle and previous-stable upgrade checks must use the new frozen bytes.

## M789 — Native artifact checks pass; shared preflight validation corrected (2026-11-21)

Status: supersedes M788's rc.2 candidate readiness; no published or installed review.
The exact rc.2 native archive installation, idempotent repeat and runner HTTP
preflight passed in fresh offline runtime04. Missing and corrupt archives refused
before pier execution. The shared CLI operation then refused its own catalogue
ID because `RuntimeProfileInput.Validate` used the generic identifier validator,
which excludes dots. Neither stock nor reviewed runtime IDs could pass it.

Validation now requires a nonempty exact member of the finite runtime catalogue;
runner target validation remains unchanged. A shared dispatch regression covers
both catalogue IDs and rejects empty, unknown and path-like inputs before any
registry access. The service, MCP server and CLI packages pass, as does the
focused race check. Native evidence is in
[the diagnostic](diagnostics/2026-11-21-runtime-profiles-shared-preflight.json).
The ended VM's generated disks/boot copies were retired with checksum receipts;
all compact evidence, sources and existing ships/manual DEV state remain intact.

Freeze0.6.0-rc.3 next and repeat native qualification. The rc.1 upgrade result and
rc.2 native image-store checks remain bounded historical evidence, not proof of
full rc.3 qualification or permission to install it. The approved48GiB aggregate
cap and single disposable VM restriction remain in effect.


## M790 — Core-functionality continuation and repository handoff (2026-11-24)

Status: supersedes the decoder-negative-control scope in M787 and the old review
guide, following the user's2026-11-24 instruction to test core functionality.
Do not resume exploit investigation or decoder crash reproduction. Validate the
exact approved artifact and ordinary runtime operations; no new decoder claim.

Recover the frozen rc.3 assets at747dd810, verify all seven hashes and record the
matching passing core/four-race-shard/QML/source CI. Runtime05 passed eight native
checks but stopped when the harness retried Start after an expected failed Start
without clearing recovery state. Preserve this failed scenario; the next harness
uses the supported Stop/reconcile sequence. Runtime06 then ended with host
signal15 before completion; its exact scope was inactive and cause is unknown.
Neither incomplete run qualifies rc.3. Retain compact results and guarded
retirement receipts for both.

The [handoff](runtime-profiles-handoff.md) and repository-owned runtime-profile
qualification scripts replace the dependency on temporary driver source. Reuse
frozen product assets, allow only this private harness/documentation to differ,
and keep the stable release-policy gate unchanged. Checksummed console chunks
refuse truncation/corruption; run long VMs through a bounded transient driver.
The existing48GiB authorization, single VM,4GiB scratch and production/DEV
exclusions remain. Full runtime, exact0.5.0→rc.3 upgrade, activation and immutable
review delivery remain gates; no production installation or stable release.

## M791 — Frozen rc.3 native core runtime scenario passes (2026-11-24)

Status: supersedes the incomplete native lifecycle gate in M789/M790 for the
explicit runtime07 scenario, not release-wide qualification. The unchanged
747dd810 product assets with d332c805's repository harness pass all14 native
checks. Both runtime pins boot; manager restart preserves both containers;
missing/corrupt archives refuse; Stop/reconcile recovers expected failed Start;
replacement, Pack and controlled process recovery retain the reviewed image.
A stopped copy of the fresh fake refuses stock import and boots under its explicit
reviewed profile. The16-container event audit verifies lifecycle/utility pins.
All three disposable instances finish with both desired and observed state stopped.

The [native record](diagnostics/2026-11-24-runtime-profiles-native.json) binds
full report/serial/host hashes and the exact harness. Storage ends at50,541,416,448
bytes below48GiB; guarded retirement reclaims3,369,426,944 allocated bytes from
only that ended VM's generated disks and boot copies. Existing state is untouched.
The same-artifact0.5.0 upgrade, matching review delivery and protected stable
promotion remain separate. No decoder-regression or production-pier claim is made.

## M792 — Matching0.5.0→rc.3 native upgrade passes (2026-11-24)

Status: supersedes the pending same-artifact upgrade gate in M789–M791. The
published0.5.0 binaries and frozen747dd810 target pass the actual offscreen QML
service-update flow, including denial and retry of the same update attempt.
Schema65→66 preserves both original ship identities/configuration,5 original
operation records, access, live processes and all19 original app-history tables
with38 rows. Manager reopen and the genuine0.5.0 writer's unchanged-state refusal
pass; new app operations and stopped fake import pass after the update.

The [upgrade record](diagnostics/2026-11-24-runtime-profiles-upgrade.json) binds
complete checksummed reports, screenshots, source identities and guarded cleanup.
Native sealed-archive loading passes, with activation acknowledgements driven by
the guest harness; full Omarchy shell activation is not claimed. Scope is the
local owner-user0.5.0 starting state, excluding remote updates, active recording
workers, power-loss behavior and every live protected ship.

Both VM scopes are inactive. Upgrade-generated retirement frees2,416,541,696
allocated bytes; final full charge47,172,669,440 bytes is within48GiB. Keep all
compact reports, sources and immutable inputs. Next prepare full isolated shell
activation and immutable review delivery pins/instructions. Do not rebuild rc.3,
repeat these completed scenarios without cause, bypass release protection or
interpret this development pass as production installation authority.

## M793 — Full isolated Omarchy shell activates frozen rc.3 (2026-11-24)

Status: supersedes M792's pending full-shell activation gate for this bounded
scenario. The unmodified Omarchy shell, under a separate headless Sway compositor
and networkless Bubblewrap namespace, cold-loads published0.5.0, hot-activates
exact rc.3 with the same shell/session, preserves the visible page and selection,
retains paired service/widget state across rescan, and cold-restarts on rc.3.
Activation acknowledgements come from the actual native bootstrap.

The [record](diagnostics/2026-11-24-runtime-profiles-full-shell.json) binds sealed
archive hashes, full shell identity, namespace adapters and retained logs. The
private review060 profile has no enrollments and local setup is disabled. No host
HOME, desktop, keyring, Docker or SSH sockets are mounted. The bounded service
finished successfully and is inactive. Host Hyprland integration, remote updates
and wider handoff/recording scenarios are not claimed; socket/teardown warnings
remain recorded. Repository-owned full_shell.py changes only source-root discovery
from the tested temporary driver.

Distribution pins now identify the original747dd810 rc.3 archive and all three
matching helpers. All nine authenticated draft downloads match the qualified
original bytes and GitHub digests; see the [download record](diagnostics/2026-11-24-runtime-profiles-draft-downloads.json).
No same-version artifact was rebuilt. Publish only as a non-latest prerelease with
exact owner-review instructions; master and installed production/DEV state stay
unchanged. Owner acceptance and stable promotion remain separate gates.

## M794 — Complete rc.3 prerelease ready for owner review (2026-11-24)

Status: supersedes the pending immutable delivery gate in M790–M793. Published
[v0.6.0-rc.3](https://github.com/thelifeandtimes/omabit/releases/tag/v0.6.0-rc.3)
is explicitly a prerelease and not latest;0.5.0 remains latest stable. All nine
authenticated draft and unauthenticated public downloads match the original
qualified bytes, sizes and GitHub digests. The tag targets frozen747dd810; the
distribution checkout is664f938ad15f139dbf79e0baef3f4bda145c9d02. Assets were
not rebuilt or replaced. The first public checker reused obsolete temporary
draft URLs; using canonical published metadata passed without an artifact change.

The [publication record](diagnostics/2026-11-24-runtime-profiles-publication.json)
and [manual entry check](diagnostics/2026-11-24-runtime-profiles-manual-entry.json)
bind exact delivery, isolated import/profile preparation and prior non-default
profile/release restoration. The existing0.5.0 native bootstrap/control code is
identical apart from distribution pins. The [manual guide](runtime-profiles-manual-review.md)
starts with a new desktop profile with host setup disabled. An owner-operated
local0.5.0/schema65→rc.3/schema66 update is a separate explicit step; arbitrary
custom DEV layouts, remote upgrades, active recording and protected ships remain
outside that qualification.

All private runtime/upgrade/shell scopes are inactive. Final aggregate storage
47,398,850,560 bytes stays under48GiB. Production, installed plugins/services,
existing manual DEV state, protected identities, tmux and temp-ur remain untouched.
Leave PR24 draft and issue22 open for owner feedback. This ready-for-manual-review
result is not stable promotion, production update or live import authority.

## M795 — Existing review040 DEV owner update command prepared (2026-11-24)

Status: clarifies M794's generic manual-entry instructions for the user's existing
custom DEV environment. The user requested commands to run the update themselves,
not execution by the assistant. The existing review040 manager and widget are
confirmed on published0.5.0/schema65, under `/srv/dev/urbit-040`, profile
`review-040`, plugin ID `thelifeandtimes.omabit.review040`, runner identity
`runner-a9a208a18445179f05df5644ed6d245c`. The earlier normal-plugin profile commands
do not update this custom installation. Use “DEV environment” for the complete
setup, “namespace” for its isolation boundaries and “configuration” for settings.

The private [owner adapter](../scripts/qualification/runtime-profiles/update_review040.py)
is staged with exact rc.3 helpers and the reviewed runtime archive. `--prepare`
only verifies and stages; `--apply` performs the owner's update. It binds old
helper hashes, owner/service/runner identity, stage hashes and storage admission;
backs up management state, tools and preferences; restarts only management without
resume-running; retains container/engine identity and original fleet intent/history;
and loads a versioned DEV widget with the same ID/profile/paths. An existing
attempt is never blindly repeated, and failure never restores an older manager
over migrated state. Offline WAL backup inspection uses a separate analysis copy.

The [preparation evidence](diagnostics/2026-11-24-review040-update-prepared.json)
records passing private full-shell targeted disable/activation/config restoration
and three fault tests, including late activation failure retaining schema66/new
helpers/original backup. The final staged plan binds the final helper digest;
preparation-only staging superseded during guard development is retained or
manifested separately. Exact frozen product bytes remain unchanged. Final charge
47,801,511,936bytes plus1GiB apply scratch fits the48GiB approval.

The live custom DEV update has NOT run and is not claimed as a completed upgrade.
The existing0.5.0 environment, production and earlier DEV environment remain
unchanged. [Owner commands](review040-update.md) use the already staged final work
directory and report success only after a complete receipt. The owner's successful
execution and subsequent manual feedback remain pending.

## M796 — DEV backup-size refusal corrected before any update (2026-11-24)

Status: supersedes M795's readiness claim for its original fixed256MiB management
backup threshold. The owner ran the command and received “management backup
exceeds declared scratch”. The management tree includes two legitimate217,421,172
byte boot pills, making the observed total439,735,368 apparent bytes. No backup,
apply marker, service stop or widget change occurred; DEV remains0.5.0/schema65.
The pasted command was clarified as error output, not permission for the assistant
to perform the live update.

Replace arbitrary per-directory limits with measured management/SQLite-analysis/
desktop/tool backup allocation plus explicit runtime-image and growth reservations.
Preparation and apply use the same estimate and unchanged48GiB aggregate cap.
Add read-only `--check` and an explicit `--refresh-plan` that retains the original
plan and refuses any started attempt, backup, changed identity or altered staged
bytes. The final unstarted plan was refreshed and checks ready with1,137,704,960
bytes of scratch: total48,939,261,952bytes remains below51,539,607,552bytes.
Both boot pills and all other data remain included in the backup; nothing is pruned.

Four fault/measurement tests pass, including large sparse boot pills, aggregate
budget refusal before mutation, symlink refusal, blocked refresh after apply starts,
and late failure preserving migrated state/new tools/original WAL backup. The
[correction evidence](diagnostics/2026-11-24-review040-storage-correction.json)
binds the final script and refreshed plan, retaining the old plan. No release
artifact or installed file changed. The owner can now rerun the same `--apply`
command; live update and manual review remain pending.
