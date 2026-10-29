# Interrupted continuation trial33 and fixture correction

VM33/schema40 failed at the first uncertain Stop boundary after2074.95s total
(1874.89s guest test). The outer driver returned1, while QEMU exited0 and the
no-NIC VM shut down cleanly. Its scope is inactive. The original disk was copied,
hash verified and retained before removing the temporary duplicate. Extraction
and independent partial-result inspection are pending.

The failure was `anchor drain has no acknowledged dispatch`. The fixture reopened
the store after deliberately losing a successful Stop reply, then incorrectly
called `AwaitDrainExit` on a newly opened driver. That method requires the original
in-memory acknowledged dispatch. Its refusal is correct; no production adapter
check has been relaxed.

The guarded VM fixture now waits using only `InspectDrain` on the exact retained
uncertain intent. The wait has a300-second bound, respects cancellation, clones
its input and does not alter acknowledgements or write a checkpoint. A subsequent
explicit recovery observation remains responsible for the durable transition.
The helper's narrow interface cannot issue a Stop. Its inert race test passes
1.111s; existing concrete adapter/lost-reply regressions pass30.679s. A fresh VM
rerun is required before any interrupted-continuation runtime pass is claimed.

This supersedes M181's running status only. M179's uninterrupted recovery pass
remains valid. Public saved-status was not reached in VM33; its runtime gate also
remains open. Production ships, installed plugins and host Docker were untouched.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-anchor-continuation33`.

## Independent failed-trial audit

The stopped-disk audit now passes as an audit of a **failed** run. All19 extracted
result files match recorded hashes. SQLite/schema40 passes quick_check; retained
operation and journal mirrors agree. The first Stop was dispatched exactly once,
its acknowledgement remains false, and the interrupted checkpoint survived one
store reopen unchanged. No recovery observation or continuation was queued.
The actual saved-status CLI stage was not reached.

The prior three application cycles have twelve acknowledged distinct writes,
original two logins, preserved state and subscription recovery, without write
replay. These are partial results, not completed anchor recovery. Both final piers'
log/checkpoint files were independently hashed from the stopped disk and their
directory identities matched. Final fixture cleanup was not reached; this does
not claim the later explicit cleanup gate passed.

The original disk and44 metadata files are retained with hashes. Only the verified,
regenerable raw conversion was removed. The first audit attempt used an incorrect
plural table name; that script error and its correction are retained. The full
continuation/saved-status gate requires a fresh corrected run.
