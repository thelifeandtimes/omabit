# Stop intent bound to current-context readiness

A new pure constructor admits original or replacement targets only with matching
current-context readiness. Historical validation uses the exact causal prefix and
preserves earlier receipts after later replacements; the same old readiness cannot
authorize a fresh Stop against a changed membership. Captured Stop process identity
and chronology must match its readiness bracket.

The adapter uses the existing committed-dispatch/single-TERM machinery and pins
the target process timestamp from readiness, including original containers whose
launch receipts lack that timestamp. Ten inert cases cover both targets after
generation3, changed intent/context, a restart after readiness, and uncommitted
signals. All pass under race4.584s; the full pure suite passes4.431s and vet is clean.

This extends M123/M125 only at source level. No store admission, queue, RPC or
format has changed. Persistence, replay, later lifecycle adapters and actual runtime
acceptance remain. Frozen appliance24 cannot qualify this new path.
