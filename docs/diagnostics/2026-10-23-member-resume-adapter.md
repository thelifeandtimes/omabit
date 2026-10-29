# Concrete first replacement adapter — 2026-10-23

The adapter reconstructs the original generation-one launch/readiness/Stop chain
and its completed retirement. It derives generation2 with resume arguments, the
same initialized pier and exclusive parent, and the same current anchor namespace.
Create omits fake bootstrap flags and the pill mount. Later container generations
need a generalized durable history selector and are refused by this first adapter.

Before Create and Start it repeats full ownership, mount-conflict, initialized
storage, state-file, lock and namespace checks. Previously stopped peers retain
exact start/finish/outcome evidence; raw peer and anchor metadata cannot change
during the new operation. Created-container observations require the exact returned
ID and never-started process. One exact-ID Start requires the committed intent and
fresh unchanged observation. Reply loss remains unresolved; no cleanup or retry
is implied. Container-running evidence is separate from Urbit readiness.

The first injected namespace-change test found Create could dispatch before the
later observation rejected the new boot/namespace identity. Fresh namespace checks
were added to every repeated snapshot, with separate pre-Create and pre-Start
cases. That failed log remains retained. All24 focused cases now pass, including
lost replies, cancellation, returned-ID retention, changed peer/process/storage,
locks, linked state, unknown members, alias protection and replay refusal. Synthetic
state retains its original bytes and inode. Full model/runtime-adapter race suites
and vet pass. No actual Docker runtime was used by these source tests.

Persistence, scheduling, generalized generation selection, real replacement and
post-resume readiness remain. VM18's frozen binary predates this adapter and is
qualifying retirement only. No persistent format or public API changed here.

A later test refinement also attempts Create and Start again while each original
command is in flight at the same committed phase. Both reentries are refused and
only one effect occurs. This strengthens the earlier post-completion duplicate
case; the focused race check passes. Its separate source/log hashes are recorded.
The frozen schema31 reader snapshot predates this test-only refinement.
