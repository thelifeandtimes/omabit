# Internal-bridge publication trial19 — 2026-10-22

**Failed publication contract, now diagnosed.** The corrected fixture recorded
both desired and observed mappings while the anchor was running. HostConfig
contains exact `127.0.0.1:18281 -> 8081/tcp` and `18282 -> 8082/tcp` requests,
but NetworkSettings lists both ports as null/unpublished. The only attachment is
the project internal bridge with no gateway. No member or HTTP request was made.
This pinned rootless Docker29.9.1 configuration cannot satisfy the chosen direct
publication contract. Requested mappings alone must never be treated as active.

The init-managed sleep anchor stopped with exit143, no OOM, and no cleanup error.
The no-NIC VM powered off cleanly in 217.38 seconds. Original artifacts/overlays,
18 extracted files and exact observations were independently verified and retained
under `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-22-endpoints19/`.
[Evidence](2026-10-22-endpoint-trial19.json) retains the running and stopped states.
Result SHA256: `0d3ba9787e1acd7d0e100551744dd7204c9e32a3cd345734bf8638ae058862b0`.

This resolves trial18's missing diagnostic detail for the corrected source; it
does not rewrite that failed trial or infer its exact exit code. A separate
non-internal bridge fixture is next, still within a private no-NIC guest. That
experiment must demonstrate exact loopback/IPv6 behavior, distinct project
endpoints, cross-bridge refusal and anchor lifecycle. No topology is adopted in
the runner yet, and no real ship or host engine was accessed.
