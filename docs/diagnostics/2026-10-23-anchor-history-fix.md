# Restarted anchor timestamps remain historical evidence

VM28's exact retained Docker metadata rejects under the old validator and passes
under the recovery-specific correction. A running recovery anchor may retain a
FinishedAt strictly earlier than its current StartedAt. The original value remains
in evidence. Initial-Start and member validators remain strict; malformed, equal
or later finish times refuse. If recovery starts the anchor itself, any retained
finish must equal the previously observed stop. Recovery retirement/replacement
readers explicitly select this recovery-aware snapshot path.

Realistic synthetic fixtures now retain prior exit timestamps. Cleanup-only flags
are set before anchor fault review even if that review fails; reopen also reports
a missing final Stop rather than panicking. Read-only review diagnostics now retain
engine/project call timings, adapter errors and successful observations without
adding engine calls or mutations.

Before/after exact-metadata regression, full projectnetwork race118.399s,
related recovery adapter race164.596s, compile/skip, guarded host refusal and
vet/diff pass. These are source gates, not a new runtime pass. Schema37 and all
production installations remain unchanged. A fresh VM must repeat the full case.
