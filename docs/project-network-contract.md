# Runner project integration contract — development draft

This is the next implementation slice after the transport/state and endpoint
experiments. It defines required invariants; it does not advertise a shipped CLI,
select an unqualified topology, or migrate existing instances. See
[project research](project-fake-networks.md), [isolation](experiment-isolation.md),
and [upgrade policy](upgrade-policy.md).

## Identity and opt-in membership

A project has a runner-generated immutable ID and a separate editable display
label. A storage folder, fake ship name, instance ID, collector enrollment or
plugin profile is not a project ID. Only explicitly requested fresh fake members
join the first implementation. Existing standalone fake/live instances remain on
their recorded topology and ports; importing or moving storage does not join a
network. Ethereum chains and cross-runner transport have separate contracts.

Persist the project identity, revision, mode, member instance/runtime IDs, and
selected fake identities before dispatching resources. A fake identity is unique
within one project but may repeat in another. The initial qualified member set is
galaxies, matching the proven local discovery path. Reject unsupported identity
classes explicitly rather than inferring that arbitrary fake planets behave alike.

All project members are trusted development participants sharing loopback and
administrative HTTP. Independent piers and process/container ownership do not
provide inter-member administrative isolation. A project description must say so.

## Endpoint ownership

Public Eyre ports are explicitly allocated inside the shared namespace and mapped
only to reserved IPv4 runner-loopback ports. Never publish Lens, conn.sock, Ames,
wildcard addresses or IPv6 implicitly. Plugin traffic goes directly to Eyre;
collector RPC is not a transport proxy. Verify the selected ship at each endpoint.

The pinned Vere image also declares `/urbit` as a data volume and working directory.
A namespace-only anchor must explicitly account for that storage (for example, a
qualified bounded ephemeral mount); it must not acquire unjournaled persistent
volumes as an incidental Docker image default. Inspect its image environment,
mounts and containment independently of member pier mounts. See memory M047.

Anchor publications are immutable Docker creation settings. The first slice
requires the complete initial member/port reservation set before creating an
anchor. Adding/removing bindings is a reviewed maintenance operation with every
member stopped; it cannot silently replace an anchor or disrupt running peers.
Do not invent a hidden capacity limit or reserve an arbitrary host-port range to
avoid this lifecycle. Existing host-port reservations and actual binding conflicts
must participate in the same runner transaction/admission as standalone instances.

Administrative ports are discovered per pier from `.http.ports` and bound to the
current exact container/ship identity before code retrieval. The source discovery
prerequisite is documented separately. A saved port is not durable authority.
Shared-project integration must also replace the remaining internal public8080
assumptions in command construction, inspection, readiness and publication.

## Resource journals and reconciliation

Record exact Docker engine, bridge and anchor identities, immutable labels/options,
publication sets, member attachments and lifecycle generations. Names/labels alone
cannot authorize adoption or removal. Persist create/start/stop uncertainty before
dispatch, with actor-scoped request replay and the expected project/instance
revisions. Observe accepted work by its original receipt; no inverse mutation or
fresh request ID is an automatic response to a lost reply.

A bridge/anchor is project-owned, never owned by the first member's individual
cleanup routine. Stopping or retiring one member cannot remove a peer's namespace.
Start verifies the current anchor, exact namespace membership, complete reserved
endpoints and unchanged pier ownership. Late/repeated completion from an older
lifecycle generation cannot publish a newer generation as ready.

| Observation | Required behavior |
| --- | --- |
| One member cleanly stops | Keep anchor/bridge and other members running; retain that member's pier/slot |
| Member resumes | Create a replacement container from the same pier in the verified current anchor namespace; do not reuse boot flags |
| Anchor unavailable with live members | Mark the project degraded/uncertain; retain old membership evidence; do not start another anchor or move members |
| Anchor returns with a different namespace | Keep the old and new observations; require coordinated member stop/replacement before readiness |
| Unknown attachment or replaced resource | Refuse mutation/adoption and retain evidence |
| Collector/runner restarts or a second operator arrives | Reopen the same durable intent; revision/replay rules prevent duplicate resources |
| Project retirement | Require all members stopped and all lifecycle outcomes resolved; preserve piers unless separately authorized |

Reconciliation needs fresh observations under a runner-wide project mutation
fence, including the engine identity and all known members. Namespace inode numbers
are observations tied to one boot/resource generation, not portable identities.
Do not infer successful cleanup from a single empty inventory after uncertainty.

## Compatibility and acceptance

The schema must advance before project membership/runtime journals are persisted,
so older readers refuse rather than discarding this state. Migration preserves
legacy bodies, signatures, replay records, ports and running/stopped intent.
Opening a store does not create a network, start a ship, or reclassify a directory.
New features use explicit capability negotiation; older runners stay observable.

Implement in bounded slices: transactional project/membership records with replay
and downgrade tests; a driver with exact resource validation and uncertainty
journals; CLI read/preview/explicit action; fresh private-VM integration; then UI.
The first real runner gate must demonstrate two fresh galaxies, direct authenticated
messages, independent stop/resume, durable application/subscription state and exact
public/admin identity. Follow with simultaneous duplicate-identity projects,
anchor failure/recovery, interrupted dispatch and manager restart. Earlier fixture
passes remain scoped evidence, not automatic qualification of this new driver.

## Current implementation slice

The [offline planner](diagnostics/2026-10-23-project-planner.md) and
[durable reservation store](diagnostics/2026-10-23-project-store.md) now pass source
checks. Schema31 preserves complete planned projects and actor-scoped receipts;
standalone registration/allocation respects their claims. Canonical digests are
review data, not stable project IDs or runtime authorization. [Runtime ownership assessment](diagnostics/2026-10-23-project-observation.md)
now passes source checks. [Durable preparation journals](diagnostics/2026-10-23-project-journal.md)
retain bridge/anchor dispatch and exact returned IDs under schema24. Resource
dispatch now has private preparation and initial Start paths; member lifecycle,
recovery, capabilities and CLI/UI remain to be integrated.
The pure assessment alone is not persistent authority. The original
schema22 snapshot refuses actual schema23 project state; no live store was changed.

The [private Docker adapter](diagnostics/2026-10-23-project-adapter.md) and
[server work/update fencing](diagnostics/2026-10-23-project-fence.md) now pass inert
source tests. [Further review](diagnostics/2026-10-23-project-refinement.md) preserves
complete returned creation IDs on client errors and requires a never-started
anchor. These supersede the missing-adapter/fencing portion of the current slice
above. [Real preparation](diagnostics/2026-10-23-project-preparation-runtime.md) now
passes. [Schema25 Start journaling](diagnostics/2026-10-23-anchor-start-journal.md)
and [the private adapter/queue](diagnostics/2026-10-23-anchor-start-adapter.md) pass
source tests; [fresh VM23-10](diagnostics/2026-10-23-anchor-start-runtime.md) now
passes real initial Start with independent receipt and cleanup verification. Member ownership/piers,
readiness collection, recovery and public routes remain unfinished.
Prepared resources must never be described as communicating ships or as a ready
project merely because the creation journal completed.


[Durable member storage](diagnostics/2026-10-23-member-storage.md) now passes source,
temporary-filesystem and original-reader checks. Storage preparation claims and
observes an empty exclusive parent only. The member container contract is separate;
neither storage completion nor cached anchor evidence authorizes a ship launch.


[Real queued storage preparation](diagnostics/2026-10-23-member-storage-runtime.md)
now passes in a fresh no-NIC VM with independent on-disk identity/ownership checks.
This qualifies empty exclusive parents only. It starts no members and does not
relax the fresh engine, full namespace/mount inventory or readiness requirements.


[Member namespace observation](diagnostics/2026-10-23-member-observation.md) now
passes inert race tests. It reports actual namespace differences and refuses
unknown joins or changes during collection. Its initial-member subset support
does not establish full-project readiness or prove exclusive filesystem use.


[Mount-conflict preflight](diagnostics/2026-10-23-member-mounts.md) now rejects
current source overlaps and unresolved inventory. It is a negative gate only:
saved source paths cannot prove historical active mount identity after replacement.
Keep lifecycle ownership and pier locks separate; unresolved volume sources refuse.


[The initial member launch model](diagnostics/2026-10-23-member-launch-model.md)
passes pure write-ahead/uncertainty tests. Its terminal running-container evidence
is not ship readiness. SQLite wiring and concrete dispatch remain unqualified.


[Schema27 launch persistence](diagnostics/2026-10-23-member-launch-store.md) now
passes SQLite interruption/atomicity tests and original-reader checks. It preserves
original anchor/storage receipts and serializes initial member admission. Its
private worker still needs the concrete runtime adapter and server scheduling.


[The concrete member adapter and private queue](diagnostics/2026-10-23-member-launch-adapter.md)
now pass injected-engine and temporary-filesystem tests. The dependency flow must
stage the pinned pill first; launch does not download it or pull an image. A fresh
private VM must qualify actual Create/Start before public interfaces or readiness.


[The initial readiness evidence model](diagnostics/2026-10-23-member-readiness-model.md)
now passes pure source tests. It brackets expected public ship identity and stable
pier/`.urb` objects with fresh complete project observations. This is historical
evidence only; its concrete collector and durable store/queue remain unfinished.


[Trial12 refinements](diagnostics/2026-10-23-member-launch-refinement.md) keep the
30-second freshness window while checking a fully hash-verified immutable file
snapshot at dispatch. Members now require --init; the fixture uses zod-first boot.
These source/preparation checks do not convert the retained trial12 failure into
a passing result. A separate fresh runtime trial must qualify the new contract.


## Initial readiness receipts (development source)

[Schema28 readiness journals](diagnostics/2026-10-23-member-readiness-store.md)
retain each explicit observation attempt separately from initial launch. A
container-running receipt does not establish initialized storage or the expected
ship identity. The read-only collector brackets a credential-free identity read
with complete unchanged runtime and pier observations. Completed evidence is
historical; a later mutation requires fresh lifecycle-specific evidence.

[Private scheduling and its guarded fixture](diagnostics/2026-10-23-member-readiness-queue.md)
share runner work/update admission and preserve cancellation/setup failures.
Restart interrupts unfinished observations without replaying effects. Exact
request replay returns the old attempt; another explicit read-only request may
observe again. These source checks do not qualify runtime readiness, Stop/resume,
recovery or a public project API. The older transport experiments retain their
separate scope.

## Latest private lifecycle qualification

[VM15](diagnostics/2026-10-23-member-readiness-runtime.md) qualifies initial launch
and durable readiness. [VM16](diagnostics/2026-10-23-member-stop-runtime.md) qualifies
queued graceful Stop, exact peer exits and replay/reopen. Schema30 adds separate
read-only recovery observations without rewriting original uncertain Stop receipts.
[The private queue and appliance17](diagnostics/2026-10-23-member-stop-exit-preparation.md)
pass source/preparation checks. [VM17 recovery](diagnostics/2026-10-23-member-stop-exit-runtime.md)
now passes independent runtime verification; its outer-shell143 discrepancy remains recorded. Resume,
changed peer/anchor recovery, capabilities and CLI/UI remain implementation work.


[VM18 retirement](diagnostics/2026-10-23-member-retirement-runtime.md) now preserves
saved state bytes and receipts through actual exact-container removal. The
[first replacement model/adapter](diagnostics/2026-10-23-member-resume-adapter.md),
[schema32 journal](diagnostics/2026-10-23-member-resume-store.md) and
[private queue](diagnostics/2026-10-23-member-resume-queue.md) pass source checks.
They handle an initial generation-one retirement followed by generation2 only;
actual replacement/readiness and generalized lifecycle selection remain.

[Read-only replacement readiness](diagnostics/2026-10-23-member-resumed-readiness-source.md)
now has [schema33 journals and private scheduling](diagnostics/2026-10-23-member-resumed-readiness-store.md).
It preserves the completed replacement and original history, observes the exact new
process/pier/ship, and permits recorded stopped peers. Source checks pass; actual
readiness and general lifecycle selection remain separate gates.

[VM19 independently qualifies the first queued replacement](diagnostics/2026-10-23-member-resume-runtime.md),
including retained-pier identity, correct resume command/mount, replay/reopen and
clean shutdown. Schema33 durable readiness and later generations remain separate.

[Schema34 current-member Stop](diagnostics/2026-10-23-member-current-stop-store.md)
now retains historical bindings while admitting only the latest causal target.
Stopped-peer selection excludes retired historical containers. New readiness after
unrelated peer replacement, later resume persistence and anchor-generation recovery
remain explicit implementation gaps; old evidence never authorizes adoption.


[VM21](diagnostics/2026-10-23-member-current-stop-runtime.md) now qualifies queued
generation2 readiness and current Stop. [Schema35](diagnostics/2026-10-23-member-repeated-resume-store.md)
supersedes the per-member single-resume persistence limit with transactional history
preservation and generations3/4 source checks. [VM22](diagnostics/2026-10-23-member-repeated-preparation.md)
is the pending generation3 runtime test. [The runner app fixture](diagnostics/2026-10-23-runner-application-recovery-source.md)
is separately source-qualified and not included in VM22. Changed-peer/anchor
recovery and public capabilities/CLI/UI still remain; the earlier pending statements
above describe their historical checkpoints.


## Offline CLI review

The development collector now exposes `project-plan --help` and `help projects`.
The [qualified CLI walkthrough](diagnostics/2026-10-23-project-plan-cli.md) covers
JSON/text review, canonical ordering and input refusal without runner contact.
Planning checks no host reservations and grants no runtime authority; public
registration/lifecycle capabilities remain separate. A digest is not a project ID.

The later [VM22/schema35 run](diagnostics/2026-10-23-member-repeated-runtime.md)
now independently verifies two replacements through generation3, readiness, clean
Stop and unchanged history. It supersedes the earlier pending repeated-runtime
status only; the other peer remains stopped and application recovery is tested
separately in appliance23.

## Saved reservation inventory

`project-reservations-v1` and `GET /v1/projects` expose only consistent stored
project/member claims. `urbit-collector projects` uses the selected runner's
identity and validates canonical digests, ordering and distinct claims. Exact
observer GET access is supported; no runtime probing or mutation occurs. The
response reports `basis: stored-reservations` and
`runtime_observation: not-performed`. This public read does not advertise project
registration or lifecycle capabilities. See the
[qualified source/CLI result](diagnostics/2026-10-23-project-reservation-inventory.md).


The [runner application recovery run](diagnostics/2026-10-23-runner-application-recovery-runtime.md)
now qualifies state/subscription recovery across two replacements while the other
peer remains stopped. Running-peer messaging is the separate appliance24 gate.

[Current-context readiness](diagnostics/2026-10-23-member-current-context.md) and
its [read-only adapter](diagnostics/2026-10-23-member-current-readiness-adapter.md)
now pass source checks after unrelated peer replacement. They preserve original
receipts and require fresh complete current ownership. The adapter currently
requires all members running. Persistence, scheduling and new Stop admission are
not yet integrated; do not rewrite old whole-project bindings to bypass that gap.
