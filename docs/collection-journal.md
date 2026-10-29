# Supported collection journal, formats 1, 2 and 3

`native/collections/journal.py` is an internal intent store and reducer. It does
not operate the loader, verify artifacts, test runtime readiness, or offer a user
switch command. Observations must come from a bound runtime adapter; an input
boolean is not proof that a process exited or a connection works.

The supported set is exactly Omabit, Theme and DMs. Content-addressed specifications
retain plugin IDs, profiles, artifact hashes, four distinct private state roots and
nonsecret endpoint references. Different named collections cannot reuse identities
or overlap roots. Artifact updates within one logical profile may retain its state.
The store validates lexical paths; the future adapter must verify actual filesystem
objects, prepared artifacts and loaded component/helper identities.

The explicit store root is separate from the existing Omabit bootstrap/profile
store and all runner state. Creation requires a new private owned directory;
opening never creates or resets one. A nonblocking coordinator lease serializes
writers. Each accepted event is persisted by fsync, atomic replacement and directory
fsync before return. Failed temporary files remain evidence. Missing journals,
unknown formats, illegal histories and unsafe file objects refuse intact. No
automatic pruning occurs at the 16 MiB/4096-event/128-specification limits.

Initial adoption requires an observed ready collection. A switch records its
request, expected last verified collection, candidate and deterministic effect
identities before effects. Last committed and currently loaded are distinct:
the former is the recovery target while a transition is pending.

Forward order: drain previous members individually; disable previous; checkpoint
appearance; enable candidate; verify candidate; commit. Rollback after attempted
candidate activation: drain candidate, including observed partial absence; disable
candidate; restore the exact checkpoint with verified readback; enable previous;
verify previous. Pending intent IDs and state digests contain no message contents,
sessions or credentials. Callbacks bind phase/effect, artifact/profile, activation
epoch and component instance. Exact replay is idempotent; conflicting or stale
callbacks refuse. Unconfirmed helper lineage persists across restart and cannot
be cleared by later receipts, re-adoption or rollback.

The first 24 tests cover reopening every transition boundary, actual coordinator
SIGKILL/resume, partial rollback, stale callbacks, unsafe storage, uncertain writes
before/after rename, concurrent leases and intact unknown-format refusal. Four
populated original-writer fixtures are retained for future compatibility tests.
They are model observations, not runtime proof. Early-phase cancellation, initial
empty-loader activation, independent orphan recovery and the runtime adapter are
not yet implemented. No existing installation is migrated to this format.

Evidence: [private journal qualification](diagnostics/2026-10-24-collection-journal.md).


## Internal coordinator and private runtime qualification

M270 adds `coordinator.step(store, lease, adapter)`. It accepts no unverified user
JSON receipts: a supplied adapter must bind effects and observations to runtime
identity, reconcile exact retries and preserve uncertainty. Six driver admission
tests extend the private suite to30. The qualification adapter now passes actual
plugin transitions, six coordinator kills and appearance rollback. It uses private
identity instrumentation and inert desktop/Omabit controls; it is not a production
adapter. This supersedes the earlier statement that no runtime integration exists
only for that bounded fixture. See [the evidence](diagnostics/2026-10-24-collection-coordinator.md).

## Explicit recovery format2 (M282)

The later [recovery model](diagnostics/2026-10-24-collection-recovery-model.md)
supersedes format1's permanent fence only for explicitly prepared and independently
verified old-activation recovery. Opening1 is read-only; a new event writes2 while
preserving all historical events. Original readers refuse2. Exact replays do not
rewrite either version. The separate recovery driver persists intent, re-syncs it
before cleanup on every retry, verifies original guardian and member cohorts, and
preserves pending application work/credentials. Its first supported phase is
drain-previous. It is an internal adapter contract, not a production command or
permission to submit self-asserted JSON as proof. Actual runtime integration is next.

## Candidate recovery format3 (M305)

A failed candidate may need cleanup before any complete loaded observation exists.
Explicit fenced rollback preserves the candidate activation and uncertainty; it
does not infer absence from a missing receipt. The separate candidate recovery
plan binds the original reservation, authority, preconstruction guardian and all
three selected namespaces. Verified closure advances only to candidate unload,
followed by retained appearance restoration and the previous collection. Outgoing
and candidate recovery histories coexist; exact completed replay has no effects.

Readers preserve format1/2 histories. New candidate recovery/fenced rollback events
require3, and the frozen original format2 reader refuses the successor. Do not
remove events or lower the format marker for rollback. The private DMs-only native
adapter and179 tests qualify the bounded contract described in
[candidate recovery evidence](diagnostics/2026-10-25-candidate-cleanup-recovery.md).
Initial adoption and public profile commands remain separate work.

M306 extends that private native adapter to Theme-only and all supported members;
no format change. See [cohort evidence](diagnostics/2026-10-25-candidate-startup-cohorts.md).

M308 initial pre-adoption cleanup uses a separate authority-bound prepared/verified
record; it does not invent an adopted prior collection or change this journal
format. See [initial recovery](diagnostics/2026-10-25-initial-owner-recovery.md).

M309 reconciles a missing initial authority binding only from canonical adopted
state and verified outgoing recovery before unload; no journal event or format
change. See [historical binding recovery](diagnostics/2026-10-25-closed-initial-binding.md).
