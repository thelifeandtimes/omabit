# Recovery regression and observation-only checkpoint resolution

The full schema38 managed baseline passes race1307.093s and static checks.
Later queue-readiness changes pass their focused race122.986s; later fixture cleanup
passes10.299s. This supersedes M165/M166's pending full regression. The first
invocation's unexplained exit143 remains recorded and is not reclassified as a pass.
Original schema37 source and fixture hashes remain57ed6461… and1ba8249a… unchanged.

A new pure dispatcher resolves at most one retained Stop-exit, namespace, removal,
Start-running or final-readiness outcome. Its interfaces expose observations only.
It preserves aggregate sequence/ownership input, commits the observation once,
and does not advance into the next mutation. Planned steps, unknown Create outcomes,
completed journals, stale evidence, missing observers, cancellation, changed source
and failed commits are refused without changing the retained checkpoint. Focused
race53.505s and static checks pass. This is a prerequisite for explicit interrupted
continuation, not a persisted observation queue or permission to replay effects.

Frozen image31 excludes this later pure dispatcher. Its corrected coordinated
runtime experiment is ready to boot; no runtime result is claimed here.
