# Coordinated recovery trial31 — retained failure

The no-NIC/rootless-Docker VM ended with guest failure and clean system shutdown.
Its two final cleanup callbacks refused the recovered containers: the fixture
expected ordinary ownership labels, but both actual containers additionally had
correct namespace-operation and retirement-operation labels. The complete receipt,
actual database, Docker metadata and both original piers are retained.

Independent inspection verifies the completed recovery and member creator rows,
unchanged prior history, clean drains, exact old-container retirement, same-pier
successors, fresh readiness, replay/reopen and application succession. Sixteen
unique peer writes and twenty-four subscription snapshots pass across four cycles,
using eight channels, two original sessions, no later login and no uncertain write
replay. Both recovered containers ultimately exited0 without OOM during guest
engine shutdown, but the fixture itself never dispatched their final Stops or
recorded its intended final state hashes. The full trial therefore remains failed.

The first independent-check attempt lacked access to the scope bus. The second
reached the final succession assertion and exposed verifier variable shadowing.
The corrected third check passes all stated partial assertions and records
`passed: false` for the overall trial. None of these changes relax the cleanup gate.

The original disk and 58 metadata/checker files are retained at:
`/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-anchor-recovery31`.
The regenerable raw conversion was removed only after hash-verified retention.
No production ships, host engine, installed plugins, tmux or excluded paths were used.

Next: correct the fixture's complete label comparison, test rejection of altered
ownership, then repeat the full trial in a fresh VM. Interrupted mutation
continuation and public project interfaces remain separate work.
