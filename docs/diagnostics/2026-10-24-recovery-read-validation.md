# Recovery checkpoint validation cost

VM35's failed saved-checkpoint read is reproducible without a running ship,
Docker, a writable database or competing recovery worker. Two no-NIC/no-disk
QEMU diagnostics use static binaries and immutable copies of VM35's interrupted
and VM34's completed schema40 databases, inside separate3GiB/no-swap/one-core
scopes. Both shut down cleanly and leave the database bytes unchanged.

In the final comparison, unchanged reads take6.26–6.31s for interrupted history
and11.31–12.18s for completed history; both exceed a5-second transaction context
in both repetitions. Revised reads finish within that same deadline:2.21–2.28s
interrupted and3.38–3.63s complete. Every successful projection has the same hash
after removing its read timestamp. This establishes excessive validation cost as
sufficient to reproduce the timeout; it is not a full HTTP/Docker qualification.

The journal now validates its deepest payload once; that payload still validates
its entire ancestry. The remaining walk checks every outer ID and exact combined
sequence. Recovery context selection resolves the full causal member history once
and captures requested historical prefixes during that traversal. Only after the
whole history succeeds are those private snapshots used to validate original pier
proofs and readiness. Public selectors still validate their inputs; no cache
survives a call, write or database reopen. Deadlines, fingerprints and schema40
are unchanged.

Added regression cases reject corrupt embedded receipts at every depth, incomplete
or corrupt later history after an earlier valid prefix, foreign bindings and
changed original piers; returned contexts remain private. Existing historical
peer-change cases and the full project-network race suite pass. Vet passes. The broader managed
recovery/source/checkpoint/public-action/observation/continuation/read-view race
run hit its aggregate10-minute test limit after600.027s while processing a fixture,
without an assertion or race report. Its20-minute bounded rerun is pending; it
is not a pass. Exact logs and elapsed times are in verification.json.
The first draft, its slower comparison and baseline failures are all retained.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-recovery-read-validation`.
This advances M197's diagnosis. VM35 remains failed and M192's private runtime
pass remains separate. Run a fresh full public recovery trial before claiming
public runtime success. No installed plugin, protected ship or host engine changed.

## Follow-up: M201

The applied-source managed recovery/checkpoint/source/public-action/read-view/
continuation/observation race rerun passes in696.430s with a20-minute aggregate
deadline. Its873 Go-source hashes and full verbose log are retained alongside
the original10-minute timeout. The five production/model changes match the
applied manifest. M200's later VM-fixture/help edits are outside this compiled
run. This supersedes only the pending broader rerun in M198/M199; it is not a
current full-managed-suite pass or a full public Docker recovery pass.
