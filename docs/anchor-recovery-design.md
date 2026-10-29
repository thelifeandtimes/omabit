# Coordinated project anchor recovery — working design

Status (M175, 2026-10-24): schema39 read-only checkpoint observation and schema40
explicit mutating continuation pass focused store/queue/race and original-reader
gates. Full39 and full40 (M178, race2164.666s plus vet) pass. The guarded continuation fixture is ready
but unrun. Frozen VM32/schema39 now passes corrected cleanup and coordinated repair
with independent retained-disk verification (M179). No public recovery route exists yet. This supersedes the earlier M169
status; dated sections below retain their historical limits.

An anchor restart can retain its Docker container ID while creating a different
network namespace. Existing members remain in the old namespace. A successful
anchor Start therefore does not restore those members' published endpoints. The
runner must retain the old binding and show recovery as incomplete until members
have been deliberately moved through clean exit and same-pier replacement.

The review currently accepts two cases: the exact anchor is present but exited,
or it is running in a new namespace on the same host boot. All current members
must still be running in the recorded old namespace. Missing resources, mixed
namespaces and already stopped members need separate admission rules. A read-only
refusal preserves evidence; it is not a reason to infer ownership or force cleanup.

A proposed recovery operation should retain:

- The complete validated review, current member creators, exact process starts,
  initialized pier identities and original operation IDs.
- One durable dispatch and observed exit per old member. Uncertain replies must
  be resolved by observation of the same process, without replaying its signal.
- The exact anchor process and namespace selected after all old members exit.
  An already restarted anchor should not receive another Start automatically.
- An explicit successor namespace record, distinct from the original anchor-Start
  receipt, plus each member's new creation and readiness evidence.

All old members should exit cleanly before the first replacement is created.
Retirement removes only a recorded stopped container; it preserves the pier.
Each replacement resumes that exact initialized pier without a bootstrap pill.
The project remains in recovery until all members join the selected namespace
and their exact identity endpoints pass fresh readiness checks.

The existing `MemberStop` validator requires a healthy current anchor. Do not
relax it globally to accept a changed namespace. Recovery needs its own authority
bound to the reviewed fault and every current member process. Similarly, do not
manufacture an `AnchorStart` receipt with a new namespace or rewrite an original
launch/resume receipt. Current generation resolution and member contract creation
must explicitly consume a successor namespace record while retaining old history.
Resource generation and namespace succession have different meanings when the
anchor's container and immutable creation contract remain unchanged.

The common project/update fence must exclude pending lifecycle effects. Admission,
replay, restart reconciliation and historical validation need to agree on the same
causal history. The next persistent-format change should cover the complete
review/recovery operation and undergo original-reader and populated-upgrade tests.
No experimental table or lifecycle rule should silently reinterpret an existing
journal.

Runtime acceptance should retain original Eyre sessions outside the anchor,
observe loss of published access after its restart, recover every member through
the runner, and verify persistent state, queued subscription updates and a fresh
reconciliation without repeating writes. M134 prepares the independent client
lifetime. The earlier standalone anchor fixture proves transport feasibility;
it does not qualify this runner operation.

## Follow-up after the M139–M142 source gates

The drain and namespace steps now have pure models and separate concrete adapters.
They are still disconnected from persistence and the runner queue. Their tests
cover one reviewed fault against the supplied current member history; they do not
yet qualify a second anchor recovery after a completed coordinated replacement.

The next bounded implementation should retire the exact recorded exited containers
under the successor namespace, preserving all initialized piers. Each removal needs
its own dispatch receipt and complete absence observation; a lost reply must not
repeat removal. Keep all original process exits available even after their runtime
containers disappear. Do not synthesize ordinary MemberStop/MemberRetirement
receipts to bypass those validators' original healthy-namespace requirement.

A replacement's immutable creation contract must explicitly reference the successor
namespace operation while retaining the original anchor creation/Start evidence.
Its container generation advances; the unchanged anchor resource generation does
not. Do not substitute a new namespace into an old AnchorStart to make an existing
constructor accept it. Every replacement resumes its existing initialized pier
without a bootstrap pill, then gets fresh identity/readiness evidence.

Current-generation reconstruction will need to merge ordinary member resumes with
completed coordinated recoveries in one causal history. Partial recovery must stay
fenced rather than appear as a healthy new binding. Historical validation must use
the exact prefix that admitted each operation, while current admission includes
all completed work and rejects pending effects. This must support subsequent
ordinary member restarts and a second anchor recovery without rewriting earlier
receipts. These are proposed next implementation requirements, not newly qualified
behaviors or a persistent-format change.

## Follow-up after M146

The serial retirement model is source-qualified. It preserves every pier after
old-container absence and resolves lost removal acknowledgements read-only.
Its concrete adapter, persistent coordination and replacement protocol remain
unqualified; the implementation above is not public runtime authority.

M147 qualifies the concrete retirement adapter with inert commands. It separately
proves absence in complete container-ID inventory, retaining all piers. This
supersedes only M146's pending adapter status; no coordinated runtime or persistent
operation has been exposed. Replacement contract construction is the next step.

M148 qualifies explicit successor member contract construction. Original anchor
receipts and ordinary contract digests remain unchanged. A future group intent
must bind all per-member operation IDs uniquely and retain each Create/Start
transition, with all old containers absent before the first replacement. Exact
fresh mount/process checks still belong at dispatch. All replacements running is
not yet application readiness or permission to release the recovery fence.

M149 qualifies the pure serial replacement executor. Each accepted returned ID
survives a Create error; pending Starts reconcile read-only with acknowledgement
status preserved. All earlier replacement processes/piers remain pinned. The
separate Docker adapter is under test. Fresh readiness, repeated causal histories
and durable/public integration remain proposed work, not qualified behavior.

M150 qualifies the private concrete replacement adapter with inert commands. It
revalidates initialized piers, locks/state trees, complete mount inventory, old
container absence and successor/peer metadata. Pending Start reopening is read-only;
uncertain Create stays preserved for review. Next is fresh all-member identity and
readiness against this complete replacement set, followed by causal-history and
persistent integration. No runtime or application-recovery claim follows from the
inert gate.

M151 adds actual Docker metadata evidence: a restarted running anchor retains its
previous FinishedAt, earlier than current StartedAt. Initial-only zero-finish checks
cannot qualify recovery. The source correction accepts this only on explicit
recovery paths, preserves it in observations, and keeps ordinary initial-Start and
member process checks strict. When this operation starts a stopped anchor, a
retained historical finish must equal that observed stop's finish. Runtime rerun
remains required; VM28's full review/reopen failed despite passing application cycles.

M152 qualifies this timestamp correction with exact retained VM28 metadata and pure/adapter race checks. Ordinary initial/member validators remain strict. The full runtime rerun is still pending.

M154 qualifies the pure complete successor readiness collector. Every identity is bracketed by unchanged exact processes/piers, including the last replacement; the resource generation and original anchor history are preserved. Concrete adapter and persistent completion remain separate pending gates.

M155 qualifies the complete readiness adapter with inert engine responses. It proves old-container absence, exact successor contracts/processes and all selected identities without lifecycle effects. Causal history and persisted completion remain next.

M156 qualifies the pure ordinary-replacement suffix after one complete recovery. The next model will retain ordered completed recovery boundaries, each with the ordinary resumes and optional current-readiness proofs that preceded it. The current suffix remains separate. Validate every boundary against its exact prefix and the entire suffix before returning historical contexts. Original launch/anchor receipts stay immutable. Second-recovery source/runtime qualification is still pending.

M157 qualifies explicit multiple recovery boundaries in the pure model, including ordinary replacements between and after two recoveries. Each boundary retains preceding resumes and current-readiness prerequisites. All historical lookup first validates the complete history. Concrete post-recovery ordinary creation must separately name the last recovery, successor namespace and its new ordinary retirement while preserving the original AnchorStart. Existing single-recovery adapter regression does not qualify those future concrete histories.

M158 qualifies ordinary post-recovery creation selection and its separate concrete contract. It names the current ordinary retirement and earlier recovery/namespace, preserving original AnchorStart and older contract bytes. This is side-effect-free construction; generalized history readers and actual lifecycle dispatch are not yet qualified.

M159 qualifies the actual read-only anchor observer on VM29 after direct-client application/member recovery. Healthy refusal, stopped/restarted reviews and retained prior FinishedAt all pass. It does not execute the later coordinated repair protocol; final fault cleanup remains fixture-owned. The VM is stopped and originals are independently retained.

M160 qualifies concrete recovery-history readers, including later ordinary creators and global retired-ID rejection during read-only review. The complete second recovery and equivalent historical-ID checks in every subsequent phase remain next; no persistent authority has been added.

M161 qualifies a complete second recovery through concrete inert adapters and global historical-ID checks before each consequential step. It preserves mixed creators and prior boundaries, with no second Start for an already restarted anchor. Ordinary lifecycle after recovery and persistent coordination remain separate work; this is not a Docker/runtime repair qualification.

M162 qualifies ordinary current-context lifecycle through explicit completed-recovery history, including mixed creators and preserved peers/piers. Historical reads do not grant fresh mutation authority. The next format must persist the complete recovery under the common project fence, preserve creator operation references, and retain uncertainty for observation-only reconciliation.

M164 qualifies the aggregate serial journal and complete-record assembly. Persist source history immutably and verify every embedded receipt against its actual database operation/mirror; pure self-consistency is not ownership authority. Reserve distinct child creator operations under the parent fence before effects. Schema37 remains unchanged at this checkpoint.

M165 adds schema38 immutable source ownership, actor/mirror validation, atomic member creators and common fencing. The private queue commits before every effect and polls only fresh read-only readiness during Eyre warmup. Restart/replay never resumes mutations. Mixed ordinary/recovery history passes focused source gates; full managed regression, explicit interrupted continuation and real coordinated repair remain open.

M167 qualifies a pure observation-only dispatcher for one retained uncertain outcome. It cannot issue runtime effects, skip to a later phase, resolve unknown Create by adoption, or authorize subsequent mutations. An explicit actor/request/sequence-bound persistent observation and continuation protocol remains to be implemented.

M170 qualifies pure continuation classification and an explicit concrete Start-only
path for a retained Create ID. Pending Stop/removal/Start/namespace outcomes still
require observation; unknown Create is never adopted or repeated. The eventual
continuation must own a new actor/request receipt retaining Before, ParentBefore
and Latest, atomically updating parent/member mirrors. Startup and replay remain
inert. This is a proposed persistent contract, not yet implemented authority.
