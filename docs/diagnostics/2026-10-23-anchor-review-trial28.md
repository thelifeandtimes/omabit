# Anchor review trial28: retained historical Docker exit timestamp

VM28/schema37 completed all six direct published-client application phases, twelve
remote writes and three member replacement/subscription cycles with the original
two sessions, six channels and eighteen subscription snapshots. All three fresh
replacement readiness observations passed. Their measured collection times were
14.887190s,14.239114s and24.623486s; the earlier VM27 readiness interruption was not
reproduced. This advances direct-client application recovery evidence, without
proving VM27's timing hypothesis as its root cause.

Healthy-anchor refusal and stopped-anchor review succeeded. Restarted-anchor review
failed: the pinned Docker runtime retains the prior FinishedAt while the new
StartedAt is later. Frozen recovery validation delegates to initial-Start validation,
which requires zero FinishedAt. A new offline regression using this exact retained
inspection reproduces the rejection before the correction. A separate fixture bug
left final cleanup-only markers unset on review failure, causing a missing-Stop
map assertion to panic during reopen. Full reopen and anchor-review acceptance
therefore remain unpassed; M137 is still the latest full runner runtime pass.

Outer1/QEMU0, clean poweroff in2036.72s and exact inactive scope are verified.
Independent extraction verifies19 result files, database mirrors for three resumes
and readiness receipts, application state/spec and write/subscription sequencing,
both final clean member exits and complete log/checkpoint file hashes, and private
container inventory. Original disks and47 metadata files are retained at `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-23-anchor-review28`.
Only the verified regenerable raw conversion was removed.

The source-only recovery correction permits a historical anchor finish only before
its current start, leaving initial-Start/member validators strict. Coordinated
retirement/replacement readers must explicitly use recovery-aware anchor inspection.
The cleanup-only fixture markers now describe the selected path even on failure.
These corrections pass their narrow metadata regression; broader source gates and
a fresh VM are pending. VM28 contains none of M146–M150 retirement/replacement.
