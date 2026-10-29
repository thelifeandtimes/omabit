# Durable initial readiness receipts — 2026-10-23

Schema28 adds a separate table keyed by observation operation ID. Every explicit
attempt survives, including interrupted attempts. Its immutable launch binding
covers the original complete serial member set, anchor and storage receipts.
Acceptance checks actor/request replay, plan/revision and one active attempt per
member. The worker commits observing before invoking the read-only collector;
only complete fresh evidence commits success. Restart interrupts queued/running
attempts without re-observing or relaunching. A new request may observe again.
Completed receipts remain historical evidence and confer no mutation authority.

Twenty-four focused cases pass, covering phase interruption, exact replay, new
attempts, concurrent admission, stale/mismatched proofs, corrupt mirrors/orphans,
and observation after durable phase commit. Full managed race tests and vet pass.
Schema27 source was frozen before this migration; its original writer generated
populated complete launch receipts. Their fixture reproduces byte-for-byte, and
the unchanged old reader refuses actual schema28 readiness state without changing
it. Earlier published-release qualification remains unchanged.

This is source qualification. Private queue wiring and real readiness collection
remain separate gates. VM14 contains the earlier frozen schema27 binary.
