# Started member retained after unresolved observation

M439, 2026-10-26. Fresh fixed VM12 completes gateway initialization, recovery and
verification. Its actual member queue advances through artifact/key acceptance,
Docker Create and acknowledged Start, then interrupts at `started`. The launch
never obtains a durable running/birth result. The guest powers off cleanly after
1,589.80 seconds, without an outer timeout or storage failure.

Independent read-only audit12 preserves the original gateway, recovery receipt,
five operations and disk/base hashes. Its optional input benchmark exceeds the
100-second allowance; this is separate from the failed member gate. Three fresh
read-only diagnostic audits retain the same disk hash and expose only fixed
fields, static Vere diagnostic tokens and hashes. Audits13–15 verify the pinned
12,352-file/834,044,028-byte runtime input generation. Their inner benchmark
allowance is150s; audits13/14 still have a110s parent bound, corrected to160s in
audit15. All complete within their bounds. These changes are temporary diagnostic
sources, not changes to runtime acceptance/freshness budgets or qualification.

The launch operation lasts696.642s and fails before its12-minute fixture deadline.
Storage-failure and deadline messages are absent; the fixture reports an incomplete
launch. Docker records member exit1, no OOM/dead flag and a95.870s process lifetime.
The bounded log contains successful key verification and galaxy-table retrieval,
then `boot: failed: unable to parse pill`, `boot: preparation failed`, and
`mot: %term`. The exact217,421,172-byte pill matches the pinned SHA256 and read-only
mount source. Vere's signal handler maps SIGTERM to `%term`. Therefore the parse
message is not evidence of pill corruption: it is consistent with teardown after
an earlier observation failure. The exact first failing observation check and the
signal's timing relative to that failure remain unestablished.

This supersedes M436's pending fixed-client runtime result: client reuse now
reaches acknowledged Create/Start. It does not qualify a complete member boot,
readiness, application state, managed Stop or restart. The initial hypothesis that
pill parsing identified the root cause is superseded by the intact pill and `%term`
evidence. No runtime deadline or acceptance check has been relaxed. The next
fresh run needs fixed-stage observation diagnostics; do not repeat this launch.

All runtime/supervisor/audit units are terminal. The3,631,087,616-byte disk is
copied, synced and hash-verified privately with its old path retained as a symlink.
SHA256: `4625b487e3cea7d4fd9af58769e090b3ada4334879f1ce7963902deb0e7bfa65`.
No keys, raw logs, chain databases, pier payload or installed-plugin state is
exported. Protected ships, tmux, temp-ur and host Docker remain excluded.
