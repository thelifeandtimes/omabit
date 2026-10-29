# Exact old-container retirement under the successor namespace

The separate adapter reconstructs current old contracts from retained history,
requires their clean exits, and rechecks every initialized pier and state tree.
One committed dispatch permits one exact `container rm ID`, without force or
volume deletion. Both bracketed complete container-ID inventories and project
membership must prove absence; moving an old container to another network cannot
masquerade as removal. Every pier remains required after its container disappears.

Reopened pending work is observation-only. Lost/bad replies, cancellation and
failed acknowledgements never repeat removal or invent an acknowledged outcome.
Inert tests cover serial all-member retirement, preserved data identity, changed
anchor/peer/storage/publications, locks, linked state, unknown containers, failed
commits and copied caller inputs. Initial passing-path tests exposed an incomplete
fixture: its original receiver lacked .urb/log and .urb/chk. The fixture now creates
both state directories for both fresh test piers, and every fault case first proves
valid preflight. Production checks were not relaxed; failed logs are retained.

Focused race36.871s, related recovery/retirement race105.032s and vet/diff pass.
These tests use inert Docker adapters only. No persistent format, public route or
live ship changed. VM28 contains no coordinated retirement. Replacement contracts,
queue integration and real coordinated recovery remain pending.
