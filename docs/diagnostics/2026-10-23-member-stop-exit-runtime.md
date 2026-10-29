# Real read-only recovery after discarded Stop reply — 2026-10-23

Fresh private VM17 passed the guest runtime gate in1306.48 seconds using frozen
schema30. Nec stopped normally. The fixture sent zod one real graceful signal,
observed its clean exit, and deliberately discarded the reply before acknowledgement
could be committed. This controlled injection leaves the original Stop interrupted
at stop-dispatched. A separate queued read-only observation records the exact
clean exit without changing the original Stop or sending a second signal. Replay
and database reopen preserve both journals and receipts unchanged.

Independent offline verification passed all17 extracted files, original disk and
appliance hashes, SQLite journals/receipt mirrors, process start/finish identities,
namespace contracts and retained filesystem objects. Both ships exited0 without
OOM; the anchor and guest shut down cleanly and the admitted scope is inactive.
All original disks and verified metadata are retained at
`/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-23-member-stop-exit17`.
Only the hash-verified regenerable extraction disk was removed.

The outer execution session returned143 despite a complete driver report with
child exit0 and clean guest shutdown. Its cause is not established. The separate
outer-exec-status.json is retained; the guest gate above relies on independent
disk verification and does not relabel the outer shell outcome as success.

This qualifies the private read-only recovery path after injected reply loss.
Changed peer/namespace recovery, stopped-container retirement, replacement and
public integration remain separate gates. No production resources were accessed.
