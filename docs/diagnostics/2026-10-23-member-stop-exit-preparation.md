# Queued Stop exit recovery and guarded fixture — 2026-10-23

The private read-only observation now uses the runner's common work/update gate.
It reconstructs the original launch/readiness/Stop binding, carries completed
peer exits into the adapter, and refuses unresolved other Stops or retained
maintenance. Admission never re-sends the original signal. Cancellation and setup
failure retain the accepted observation as interrupted. Seven queue cases pass,
including exact replay, concurrent update admission, completed peer selection and
unresolved-peer refusal. Affected project/model/adapter race suites and vet pass.

The next disposable fixture stops nec normally, then sends zod's real graceful
signal but deliberately discards its reply after a fixture-only read has verified
exit. That read does not commit an acknowledgement. The original Stop must remain
interrupted at stop-dispatched. A distinct queued read-only observation must then
record the exact clean exit while leaving the original receipt unchanged. Both
operations must retain replay and reopen equality. The injected reply loss is a
controlled test boundary, not a claim of a real network outage.

All four ordinary guest tests skip; explicit host enablement refuses at the guest
marker before effects. Appliance17's127 frozen sources, artifacts and embedded
test match independent verification. Runtime execution remains pending. The
schema29 VM16 result qualifies initial Stop only. Current recovery requires peers
to retain their original lifecycle state; changed peers/namespace, resume and
public integration require separate handling. No production resources changed.
