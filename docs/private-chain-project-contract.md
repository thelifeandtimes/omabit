# Private local chain project contract — development source

The initial private profile is `private-local-keyed-v1`, separate from
`fake-galaxies-v1` and livenet. Offline planning, persistent registration, capability-checked public reservation
and read-only inventory are implemented in development source. Runtime dispatch
and connection/UI classification are not yet implemented. Existing fake project records and digests are unchanged.

## Current implementation boundary (M436)

Public commands still cover reservation and saved inventory. The development
runner now has schema45 internal workspace creation, gateway initialization,
separately attributed read-only recovery queues and first-member launch storage. Its current-source gateway VM
qualification passes with ordinary binaries through two manager reopens and an
independent unchanged-disk audit ([M417](diagnostics/2026-10-26-managed-gateway-recovery.md)).
Member lifecycle and acknowledged gateway Stop/restart remain separate gates. Follow [goal progress](goal-progress.md) rather
than treating older pending entries below as current status.

The initial gateway observer qualifies the initial state with **empty member
mounts**. It rejects pier contents and other workspace mounts. It is not yet an
observer for a project with running members. Extending it must account for exact
journalled member containers and pier identities, while preserving exclusive
chain/key ownership and refusal of unrelated workspace writers. A saved readiness
receipt alone cannot authorize a later member launch.

[M421](diagnostics/2026-10-26-private-member-gateway-state.md) adds the separate
member-aware filesystem constructor with exact retained pier identities and
opaque live payload. M425 now adopts it in a separate private member-aware observer; durable
lifecycle binding and actual member runtime remain required.

[M406](diagnostics/2026-10-26-private-member-command.md) adds the pure keyed-member
command contract, with distinct boot and retained-pier invocations. It performs
no filesystem or engine operations. [M412](diagnostics/2026-10-26-private-member-container.md)
adds pure container/inspection binding, including read-only boot inputs and a
resume mode that excludes them. Held-file verification, fresh member-aware gateway
observation, durable lifecycle and acknowledged shutdown remain separate gates
before public orchestration. Existing successful fixture messaging and application
recovery tests do not by themselves qualify the current managed adapter.

[M413](diagnostics/2026-10-26-private-member-mount.md) provides held, read-only
member-mount verification, including the empty first-boot check. It deliberately
excludes peer pier contents, keys, chain state and engines. Exact resume pier
verification and boot-input verification remain separate requirements.

[M414](diagnostics/2026-10-26-private-member-key.md) verifies one bootstrap key
against an independently accepted file hash, preserving held directory/file
identities. The helper does not establish key/chain correspondence or acquire
launch authority. Member intent must bind that accepted hash, and a driver must
supply fresh gateway/key proof; resume excludes this bootstrap-only access.

[M416](diagnostics/2026-10-26-private-member-pill.md) verifies the pinned first-boot
pill and holds its exact private workspace/file objects. This is read-only source
qualification; staging, retained-pier/stopped-state verification and runtime
admission remain separate. Resume cannot regain a bootstrap-pill mount.

[M418](diagnostics/2026-10-26-private-member-pier.md) holds the retained pier/.urb
objects and separately checks lock absence and bounded log/checkpoint metadata.
It reads no payload or peer state. This does not prove process exit, acknowledged
Lens Stop or durability; the lifecycle driver must establish those separately.

[M419](diagnostics/2026-10-26-private-member-key-proof.md) adds the source-qualified
key-proof primitive: exact held key, stdin-only Vere evaluation and a fresh full
chain observation afterward. M426 qualifies its real evaluator/chain behavior. Complete
member admission is still required; the returned hash alone grants no dispatch.

[M420](diagnostics/2026-10-26-private-member-key-fixture.md) adds a guest-only
qualification bridge and independent stopped-disk key-proof audit. M426 passes its fresh actual evaluator/chain cases and independent retained-key
audit; member launches remain separate.

[M424](diagnostics/2026-10-26-private-member-engine.md) adds the private engine
preflight for exact recorded member containers, mount ownership and namespace-
sharing references. M425 now composes it with held-pier and actual kernel namespace checks in
a separate private observer. It authorizes no lifecycle effects.

[M425](diagnostics/2026-10-26-private-member-observer.md) source-qualifies that
composition. It requires exact retained gateway/member/pier selections and
brackets running process identities and namespaces without executing stopped
members. Durable lifecycle, first-boot identity capture, migration and actual
member runtime/freshness qualification remain.

[M426](diagnostics/2026-10-26-managed-member-key-proof.md) independently qualifies
two real keys and four explicit negative controls after managed gateway recovery.
No member ships were booted; that frozen VM excludes the M424/M425 observer.

[M427](diagnostics/2026-10-26-private-member-launch.md) adds the initial member
launch state machine with durable-commit callbacks. Its SQL adapter, first-boot
capture, readiness and actual runtime qualification remain; schema43 is unchanged.

[M428](diagnostics/2026-10-26-private-member-store.md) adds transactional first-member
launch storage and verifies original43 migration/downgrade refusal. Startup retains
unfinished evidence and never dispatches it. Second-member admission, first-boot
pier capture, durable readiness and the concrete lifecycle adapter remain. No
public member operation is enabled.

[M429](diagnostics/2026-10-26-private-member-birth.md) provides the held first-boot
metadata primitive: empty-mount Arm before Start, then one process-bracketed
pier/.urb capture. It performs no runtime effect or payload read. Durable capture
binding, full readiness and actual runtime remain; schema44 is unchanged.

[M430](diagnostics/2026-10-26-private-member-driver.md) composes the private first-member
runtime driver, including exclusive artifact publication, fresh key/gateway checks,
exact Create/Start and held birth capture. It refuses peers and uncertain-effect
replay. Durable capture/readiness, scheduler integration and actual managed-member
VM qualification remain; no public runtime route is added.

[M432](diagnostics/2026-10-26-private-member-durable-birth.md) binds first-boot capture
to version2 launch records and their operation mirrors in schema45. Original44
version1 history remains unchanged and cannot execute as version2. Captured pier
identity is still distinct from application readiness, clean Stop and resume
authority. Scheduler integration, second-member admission and actual managed-member
VM lifecycle remain. Original44 migration and old-reader refusal pass.

[M433](diagnostics/2026-10-26-private-member-queue.md) connects the internal first-member
queue and read-only session factory to the concrete driver under the shared work/
update fence. Exact replay precedes backend access. Schema45 and public routes are
unchanged; application readiness, second-member support and actual VM lifecycle
remain unqualified.

[M435](diagnostics/2026-10-26-original45-member-birth.md) freezes the original schema45
writer and captured/interrupted member histories before a future readiness format.
The first member VM then exposed the ready-gateway client reuse mismatch:
[M436](diagnostics/2026-10-26-managed-member-ready-client.md) reproduces and fixes
that source integration error, with 198 affected race checks passing. The failed
launch remains planned/interrupted and its disk is retained. A fresh fixed VM is
running; managed-member runtime success is still pending its independent audit.

[M437](diagnostics/2026-10-26-private-member-http-readiness.md) adds the pure first-member
HTTP readiness model and one-attempt read-only collector. It requires durable
birth capture and complete fresh observations around the fixed identity endpoint.
No concrete readiness driver, persistence, second-member admission or runtime
qualification is added.

[M438](diagnostics/2026-10-26-private-member-http-driver.md) implements the concrete
read-only first-member HTTP driver, superseding M437's missing-adapter statement.
It requires the management fence and retained birth identities; persistent
observation, public dispatch and real readiness qualification remain open.

## Review identity

`internal/privatechain` binds a version1 public foundation: profile
`azimuth-local-1337-v1`, chain ID1337, canonical nonzero genesis and the complete
nine-contract address/code-SHA256 set demonstrated by the pinned foundation and
Docker experiments. A chain URL or numeric chain ID does not identify the chain.
The contract profile is fixed to the qualified deployment; arbitrary deployments
and public testnets refuse. A supplied manifest is review metadata, not proof that
a running endpoint matches it. Runtime admission must freshly verify that match.

Foundation-v1 `code_sha256` hashes the canonical lowercase, `0x`-prefixed code
string as UTF-8, matching the original deployment writer. It does not hash decoded
bytecode. Preserve this encoding and the existing pins (M410).

The foundation digest excludes mutable head height. Two projects can deliberately
clone the same original foundation and therefore have the same genesis and ship
names. Runtime selection also needs the immutable runner ID, project ID and chain
lifecycle generation. The pure binding matcher rejects a different project,
runner, generation, foundation, or non-private mode. The eventual driver must
establish ownership/freshness before supplying observations to that matcher.

## Initial supported shape

The plan describes exactly two fresh keyed members, `~zod` and `~nec`, with unique
instance IDs and requested unprivileged runner-loopback HTTP ports. Their internal
HTTP/Ames ports and gateway port follow the qualified Docker fixture. RPC and
Ames publication remain disabled. The plan fixes the two runner-loopback HTTP publications. M372 qualifies direct
clients after gateway replacement; first-initialization publication and the public
runner adapter remain separate runtime gates.

Members share administrative loopback access and explicitly acknowledge that
trust. Every initial member requires a new disposable key for this private chain.
The official image's first keyed shutdown requires acknowledged hood Lens exit
and observed exit0; a sent request is not a completed Stop. Reopening uses the
preserved pier, and must never repeat initial boot/key inputs. Piers, private keys,
chain databases, receipts and uncertain outcomes must survive manager restarts.
A plugin collection change cannot reset or silently repoint any of those resources.

## Offline CLI

```sh
urbit-collector --format text private-chain-plan --label local-apps \
  --foundation /absolute/public-foundation.json \
  --member dev-zod:~zod:18381 --member dev-nec:~nec:18382 \
  --shared-administration
```

The version1 public metadata example is
`internal/privatechain/testdata/qualified-foundation.json`; it contains only
public chain/contract values from the qualified foundation. The parser refuses
unknown/duplicate fields, unsupported versions/profiles, altered contracts,
special files and oversized input. JSON and text output identify the private mode,
foundation/review digests and unperformed reservation check. No registry is opened,
runner contacted, key generated, or resource reserved/started. Do not feed this
plan to `project-register`: the existing fake registration path rejects it.

Schema41 now retains distinct private-project records, transactional shared
reservations and actor-scoped replay (M353), with original40 writer/refusal coverage.
Next implementation gates are a verified immutable runtime input
bundle, project-owned chain/gateway lifecycle, explicit member-key staging and
acknowledged shutdown, then no-NIC VM publication/isolation/recovery tests. Opening
old state must neither reclassify fake projects nor create a chain or ship. Any new
persistent schema needs original-writer/downgrade and non-destructive upgrade
coverage before activation. The offline planner itself never opens or changes stored state.

## Public reservation and saved inventory

The separate `/v1/private-chain-projects` endpoint advertises
`private-chain-registration-v1` for operator POST and
`private-chain-reservations-v1` for observer-compatible GET. Primitive registration
inputs include public foundation metadata, member requests, shared-administration
consent, reservation acknowledgement, review digest and actor-scoped request ID.
Duplicate/case-alias/unknown fields, keys, runtime descriptors and bodies over64KiB
refuse. The runner reconstructs the plan rather than trusting caller runtime data.

`private-chain-register --help` explains pinned selection, the saved JSON review
and `--replay-only`. A lost/malformed response is uncertain; it never automatically
repeats POST. Exact lookup retains the original request and refuses an absent
receipt before probing ports. `private-chain-projects --help` reads saved public
plans/bindings and registration references without exposing member runtime tokens.
The response explicitly grants no runtime authority. Existing fake commands and
saved bodies retain their original contract; private IDs cannot enter that lifecycle.

M354 qualifies synthetic public/client/CLI behavior, strict review-file refusal,
observer inventory and lost-reply handling. The first API test caught a missing
replay-only transport header on the new path; the targeted fix passes existing
creation-replay regressions as well. Installed runners/plugins remain unchanged.

## Retained lifecycle boundaries (M355–M358)

The qualified gateway initializer copies the original unkeyed foundation and
creates fresh network keys. It must never be reused as a restart service. Retained
startup requires a separate path with no fallback to initialization, key generation,
transaction submission or mining. Missing or inconsistent retained state is an
explicit refusal, not permission to recreate it. M355 introduces that source path;
M356/M357's first whole-chain reopen trial and independent audit are still pending.

Keep three ownership domains explicit:

- Immutable runtime inputs: pinned executables, dependency/native-library closure,
  gateway source, compiler output and original unkeyed foundation. A supplied
  digest is an identity check, not independent publisher authentication or proof
  that the code is currently qualified for a supported host.
- Project state: the particular runner/project/generation's chain database,
  network keys, piers, runtime container IDs and durable operation receipts.
  Neither a shared genesis nor matching ship names identifies those objects.
- Client intent: each plugin's own session and subscription cursor, separate from
  the gateway process and from other plugins. Gateway tmpfs is insufficient client
  persistence. M358's fixture retains this intent privately and does not log in
  again or replay a write whose outcome is unknown.

Before retained startup, verify the exact predecessor's clean stop, expected image,
command, project binding, mounts and resource contract, plus unchanged selected
chain directory identity and the original proof/input hashes. A saved receipt is
not a fresh engine observation. After opening, observe genesis, head, contract
code and public key/life/suite; clean read-only shutdown must preserve the head
hash and key observations. The current candidate intentionally covers the first
qualified head0x50/life1 profile only; it is not general key-rotation support.

The gateway is also the private network namespace owner in this first fixture.
Replacing it requires stopping the old peers, creating the new namespace, and
reopening their existing piers in that namespace with no first-boot key/pill flags.
Original client sessions and queued subscription updates must survive separately.
M358 has source/inert proof for this coordination; actual peer recovery through a
replacement gateway remains a runtime gate. Host HTTP publication and two
simultaneous private projects remain separate gates as well.

The public schema41 API still reserves plans only. It neither supplies the runtime
bundle nor implements these state transitions. Durable initializer/resume journals
and recovery must gain their own upgrade-qualified persistence contract before
runner dispatch is enabled; do not reinterpret existing planned records as an
already initialized chain. See [resume source](diagnostics/2026-10-25-private-gateway-resume-source.md)
and [retained client source](diagnostics/2026-10-25-private-gateway-peer-source.md).

M359 sharpens the retained-head requirement: the new source records the predecessor's
final `gateway.headHash` and checks it before resumed readiness. A legacy top-level
block hash captured before gateway setup is insufficient. The reader explicitly
distinguishes this coverage; the frozen VM10 precursor does not supply that final
hash.39 JS/110 Go checks pass; the strengthened runtime case remains pending. See
[original-head binding](diagnostics/2026-10-25-private-gateway-original-head.md).

M360/M361 supersede the pending first trial above: VM10 passed its original peer
case but refused before gateway replacement because its fresh chain copy was0755.
The next initializer corrects only its newly created directory to0700; retained
admission remains strict. Fresh VM11 now tests this and the stronger exact-head
plus saved-client recovery contract. Whole-gateway runtime qualification is pending.

The local `private-chain-inputs` command now supplies a read-only input review
boundary. It checks the complete sealed tree against separately reviewed manifest
and foundation hashes; no runner state or runtime authority changes. See
[runtime inputs](private-chain-runtime-inputs.md). Runtime dispatch and a supported
publisher/delivery mechanism are still separate.


M364 supersedes the pending VM11 status: gateway readiness/RPC were observed but
the peer fixture refused a repeated historical namespace number before creating
peers. A namespace inode string must not serve as a durable project-generation
identifier. Distinct exact containers and verified network ownership define the
generation; host separation and current peer membership remain required. Source
correction passes; complete retained-gateway/peer runtime remains pending.

M367 supersedes the pending whole-gateway runtime gate: the exact retained chain
and both piers recover in replacement containers; original sessions and queued
subscription state survive without replay. The independent read-only disk audit
passes, including unchanged source hashes and clean shutdowns. HTTP publication,
concurrent private projects and durable public runner dispatch remain separate.


## Workspace prerequisite (M370)

The private workspace model now derives `private-projects/<project-id>/piers/<instance>`
under an explicitly selected, identity-pinned runner data parent. It prepares
empty exclusive mount parents; it creates no ship-named pier, key, chain or engine
resource. The controller journals before effects and retains unknown outcomes.
The Linux adapter checks exact directory objects, private modes and contents;
recovery only inspects/syncs a complete layout and cannot repair a partial one.
See [source/filesystem evidence](diagnostics/2026-10-25-private-workspace.md).
This source slice is not wired to public requests, queues or persistence, and
does not alter schema41. The original41 fixture in M369 is the baseline for the
next durable lifecycle migration and old-reader refusal gate.


M371 advances the unreleased runner format to42 and persists that workspace
contract in its own table, mirrored into an actor-bound operation. Shared
reservation validation checks these claims. Startup interrupts unfinished work
without creating directories; exact replay preserves its original receipt.
The internal store runner commits before each filesystem callback. A public
queue and separately attributed observation-recovery request remain unwired.
Original41 preservation and refusal of actual42 state pass; full store regression
is running. See [durable workspace evidence](diagnostics/2026-10-25-private-workspace-store.md).


M372 qualifies direct guest clients over exactly two IPv4-loopback Eyre publications
after whole-gateway replacement. Existing sessions, queued state and old/fresh
subscriptions survive without replay; RPC/admin/IPv6 probes refuse. The audit
requires seven channel PUTs (four acknowledgements, one subscribe, two deletes),
not the earlier three-count assumption. The unchanged stopped disk passes the
corrected independent reader. Concurrent private projects and public runner
dispatch remain open; see [publication evidence](diagnostics/2026-10-25-private-http-publication.md).

M373 adds a separate concurrent-project fixture and mandatory independent audit.
The two projects deliberately reuse ship names/internal ports and the unkeyed
foundation while retaining distinct chain/key/namespace/pier ownership. The new
source gate passes, but appliance16 remains unbooted pending memory admission.
Concurrent private projects are still unqualified; M372's publication pass stands.

M374 adds the internal workspace queue through the common runner work/update
fence. Backend-selected existing parents are inspected read-only; replay performs
no filesystem lookup. Setup/cancellation retain intent, and every opened driver is
closed. Focused tests include real disposable directories and original41 state.
The pre-queue full schema42 managed regression also passes. This supersedes its
pending status only; explicit shared storage provisioning, durable observation
receipts and chain initialization/resume/public orchestration remain required.

## Initial gateway controller (M375)

The source model binds a fresh initialization operation to the observed workspace,
input manifest, engine, image and original plan. Network creation, container
creation and Start each have committed dispatch/observation boundaries. It retains
known IDs even on failure and never repeats an unknown initialization. Readiness
requires fresh complete ownership, private synced objects, the exact foundation,
head/hash/public keys and two planned loopback Eyre publications. An observer-only
recovery interface cannot call Create or Start and preserves the original receipt.

This model is not yet persisted or dispatched by the runner. It requires a concrete
adapter and separately attributed recovery persistence; input inspection or a
cached readiness result grants no runtime authority. M375 source tests do not
extend M372's replacement-gateway publication evidence to initial startup.

M376 freezes the actual schema42 workspace writer and a twice-reproduced fixture
with both succeeded and interrupted workspace journals. Reopen/replay preserves
all earlier data and refuses restarting these operations. This is the baseline for
the next combined persistence migration, not evidence that migration already ran.
The original41 archive and fixture remain unchanged.

M377 qualifies staged database adapters for initialization and distinct workspace/
gateway observation receipts. They preserve original interrupted attempts and
validate cross-project directory/public-key claims. Actual temporary-directory
recovery passes, alongside synthetic database and inert gateway checks. The source
proposal is retained outside the checkout pending one schema43 migration; no
current schema, startup queue, public route or concrete runtime adapter changes.


M378 supersedes M377's staged-only status: schema43 integrates initialization and
separate observation receipts with normal migration/startup and global claims.
Original42 migration, unchanged rows/sidecars and actual old-reader refusal pass.
Workspace recovery cannot commit objects claimed by another project's gateway;
that conflict is checked within the same transaction. Saved recovered readiness
retains both its original resource owner and the new observing operation, and
still grants no fresh runtime authority. Full43 regression is running. Internal
scheduling and concrete first-initialization publication/runtime remain separate.
See [schema43 evidence](diagnostics/2026-10-25-private-lifecycle-schema43.md).


M379 qualifies staged internal initialization/observation queues with63 race checks
and actual temporary-filesystem recovery. Replay returns before backend lookup;
all workers share the host work/update gates and close opened handles. Observation
factories expose no creation methods. These two source files are not integrated
while the full43 baseline runs, and the gateway backend remains inert in tests.
No concrete gateway runtime or public dispatch is claimed.


M380 adds source-qualified first-start publication with a mandatory independent
audit. Only the two Eyre ports are mapped to guest IPv4 loopback; exact private
bridge routes and initial head hash are checked. Direct original-session state/
subscriptions have a separate transcript. This is not a runtime pass or a concrete
runner gateway adapter. M372 replacement behavior and its seven-PUT audit remain;
the first-start case requires eight PUTs for two new channels.


M381 closes the M378/M379 full-baseline and queue-integration gaps:1,839 unchanged
baseline race checks, then132 affected checks after the two queue files integrate,
plus vet. Startup still interrupts receipts without dispatch. The first-publication
VM is running; concrete gateway/backend delivery and public lifecycle remain open.


M382 brings the existing workspace queue onto the same lifecycle helpers as the
three newer queues. Receipt replay precedes backend lookup; typed-nil handles and
rebound queued receipts refuse before filesystem authority. The original-source
negative control fails as expected and91 combined race checks plus vet pass.

M383 integrates the network portion of the concrete gateway contract: retained
selection labels, exact bridge/gateway endpoint identity, private IPv4 allocation
and ambiguous-response refusal.76 race checks/vet and74 overlapping integrated
checks pass. No runtime driver or public authorization is supplied; first-start
VM15/audit30 remains pending. See [network contract](diagnostics/2026-10-25-private-gateway-network-contract.md).

M384 closes the first-start publication runtime gate: VM15/audit30 verify original
sessions, two more writes/four SSE reads/three scries, ordered state and channel
cleanup, private endpoint refusals, clean shutdown and unchanged stopped disks.
This supersedes M380–M383 pending first-start runtime statements, not concurrent
projects or runner/public dispatch. See [runtime evidence](diagnostics/2026-10-25-private-initial-publication-runtime.md).

M385 adds the concrete container/publication contract, binds allocation observations
to exact network/container IDs, and rejects unexpected mounts, privileges, endpoints
or process history.168 race checks/vet and166 overlapping integrated checks pass.
Runtime engine/filesystem admission and public dispatch remain separate. See
[container contract](diagnostics/2026-10-25-private-gateway-container-contract.md).

M386 supplies a new complete reviewed input generation for the current initializer
and retained resume programs. Independent collector inspection passes12,352 files
with runtime_authorized=false; the old generation remains unchanged. This is
input evidence, not runtime permission. See [generation evidence](diagnostics/2026-10-25-private-gateway-input-generation.md).

M387 provides read-only verification of a completed workspace against retained
parent/root/piers/member identities. It exposes no mutation or sync authority;
same-shaped replacements refuse.62 focused checks/vet, a targeted negative control
and354 full privatechain checks pass. The guest candidate still needs deliberately
journalled home/marker/namespace setup; do not make preflight create it. See
[workspace verifier](diagnostics/2026-10-25-private-workspace-verifier.md).

M389 supplies GatewayBootstrap for journal-gated startup metadata. Open and
VerifyEmpty remain read-only; Initialize belongs after durable start-dispatched
and before the single container Start, after fresh engine/container/input checks.
It creates only home, marker and observed process-namespace metadata, preserving
partial failures without retry. No concrete runtime caller is wired yet.31 focused
race checks/vet and395 full privatechain checks pass; see
[startup metadata evidence](diagnostics/2026-10-25-private-gateway-bootstrap.md).

M390 adds internal concrete gateway resource effects using held inputs/bootstrap
and exact rootless-engine/image/network/container selection. Known IDs survive
errors, uncertain effects never repeat, and input bytes are rechecked before Start.
Current mount overlap is checked without claiming historical kernel identity.
Fresh readiness observation and managed backend wiring remain; no public factory
or runtime acceptance is claimed.45 focused checks/vet, real-bundle constructor
qualification with an inert engine and211 integrated checks pass. See
[resource adapter evidence](diagnostics/2026-10-25-private-gateway-resources.md).
