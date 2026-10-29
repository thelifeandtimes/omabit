# Bounded first-start gateway readiness

The private driver now combines journal-gated resource effects with read-only
readiness waiting. Only a successfully returned Start and a committed started
receipt may enter waiting. Its deadline is capped at10 minutes and shortened by
the caller; attempts cannot repeat bootstrap, network/container creation or Start.
Unknown Start outcomes remain for separately attributed observation recovery.

Each attempt checks the held original inputs, fixed initializer, engine/image,
exact running container/bridge and competing mounts. Changed invariants end the
wait. Incomplete initialization may be observed again, but once an observer opens,
its original filesystem/input handles stay held across polling. Success receives
the controller's existing full-contract/freshness validation before ready commit.
Failure and Close never clean up or recreate runtime state. Failed observation
results and raw diagnostics confer no partial readiness.

All26 focused race checks and vet pass, including the actual generation07 with a
real temporary workspace and inert engine. All312 private-gateway race checks
(34 top-level tests) pass after integration; diff checking is clean. The first
caller-deadline fixture incorrectly let its10ms budget expire during setup. It
refused safely before Start; the corrected fixture starts the deadline after the
resource setup and proves cancellation of readiness without repeated effects.
The failed evidence remains retained.

This supersedes M395/M396's missing bounded first-start composition only. The
constructor is private; managed backend/client-parent wiring, supported input
delivery and current-source no-NIC VM qualification remain. No actual Docker,
gateway program, ship, installed plugin, profile or production service was used.
