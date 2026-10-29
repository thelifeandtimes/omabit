# Private replacement queue — 2026-10-23

The queue reconstructs original launch/readiness/Stop and completed-retirement
bindings under the common work/update gate. Completed peer exits remain exact;
unresolved other Stops and retained maintenance refuse execution. The adapter's
initial generation is checked before it can become a running journal. Cancellation,
setup failure and changed generation preserve the accepted receipt as interrupted.

Six focused race cases pass: shared admission/replay, waiting cancellation, setup
cancellation/failure, changed generation and retained maintenance. Vet passes.
The initial fixture compile mismatch was corrected before execution and its log
is retained. This is private scheduling only; no public route or startup action
can invoke it. Actual replacement and subsequent ship readiness remain pending.
