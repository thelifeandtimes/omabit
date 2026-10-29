# Durable current-context readiness — focused checks

The unreleased schema36 adds an independent per-attempt current-readiness journal,
bound to original initialized-pier evidence, the exact current creator and complete
causal membership. Opening/restart preserves old bytes and interrupts pending reads
without runtime effects. Historical reads retain their exact earlier prefix after
later replacements. A current container with retained Stop intent cannot become
running readiness through an unrecorded restart.

Private queue admission shares the update/resource fence and serial worker gate.
Store interruption/replay/rollback/corruption/history tests pass race49.217s; queue
admission/cancellation/maintenance tests pass race12.504s. A separate completed-Stop
refusal passes. Historical migration checks and vet pass. The full managed race
suite is still running; this is not a complete regression or runtime qualification.

Before changing the schema,1438 schema35 source files were frozen and verified.
Its483595-byte populated repeated-replacement fixture reproduces identically in two
independent runs. The original reader refuses actual schema36 state containing a
pending current-readiness observation without modifying it.

The first queue staging script failed before writing source because its search
used a name it had already renamed in memory. A subsequent no-matching-tests command
from that fail-open shell is not counted. The corrected staging and fail-fast run
execute the actual queue tests; the setup failure remains in the evidence.

This advances M123/M125's absent-persistence status for read-only observation only.
M126's new Stop source remains outside store acceptance. Current-context Stop and
later lifecycle, public RPC/UI and actual schema36 runtime remain open. Frozen
running VM24 uses schema35; no installed database was migrated.

## Full regression result

The full managed race suite completed with exit0 in814.392s. All production source
hashes listed in this report still match. The subsequent VM fixture wiring compiled
and passed its skip/host-refusal checks separately; it was not part of that already
running test executable. Runtime qualification remains pending. This supersedes
the pending full-regression status above and in M127.
