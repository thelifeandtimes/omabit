# Integrated private lifecycle regression

M381, 2026-10-25. The full schema43 baseline passed408 top-level/1,839 checks
including subcases with race detection in2,411.22 seconds. All pinned source and
historical fixtures stayed unchanged during the run. Managed passed367/1,512;
privatechain passed41/327. There were no failures.

The two M379 lifecycle queue files were then integrated byte-for-byte. Affected
queue, migration and observation checks passed27 top-level/132 including subcases
with race detection, plus vet, in131.80 seconds. The earlier full baseline did not
include those queues; its result is not relabelled as a later full run. This closes
M378/M379's pending full-regression and queue-integration work. No schema change,
public runtime route or concrete gateway backend was added by this checkpoint.

First-start publication continues separately. Preparation19 refused unsupported
quota reporting on the private output filesystem before creating an appliance.
No quota rule was relaxed. Fresh appliance20 was prepared on admitted temporary
storage, then all23 files/325,478,936 bytes were verified and retained privately
behind the original compatibility path. Its12,361 inputs expand to642,480,519 bytes.
Fresh VM15 is running with no NIC, its own rootless engine,12GiB scope cap/no swap,
7GiB guest and two CPU cores. Independent read-only audit30 is gated on a clean
stop. Runtime success is not yet claimed.

Review found a related older workspace-queue inconsistency: backend lookup preceded
receipt replay, and typed-nil drivers were not handled like the three newer queues.
A small candidate reuses the same admission, replay, queued-receipt, worker and
handle checks. Its negative/positive tests run separately; this is not yet integrated.

No protected ships, tmux, `~/temp-ur`, installed plugins, host Docker or installed
services were accessed. Next: finish the workspace consistency check, inspect VM15
and its mandatory audit, then concrete private runtime adapters and public lifecycle.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-25-private-lifecycle-integrated`.
