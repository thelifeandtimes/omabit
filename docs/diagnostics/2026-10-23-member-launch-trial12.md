# Initial queued member launch trial 12 — 2026-10-23

This fresh no-NIC trial failed its complete acceptance gate. It ended in 672.63
seconds with a clean VM shutdown, no outer deadline/quota failure and no reported
container OOM. Original appliance/overlay disks, 19 extracted receipts, 42 metadata
files and the bounded private member log are retained. Independent read-only
SQLite/disk verification confirms the exact states below.

The first member, vm-nec, completed the queued initial launch and retained its
running-container/namespace receipt. The second, vm-zod, reached create-dispatched
but the adapter refused stale preflight evidence before Docker Create. No second
container exists. SQLite preserved the interrupted operation and known workspace;
no uncertain mutation was retried. Complete namespace observation plus a repeated
full 217-MB pill hash occurs inside the final 30-second freshness window while the
first ship is booting under TCG. Avoid repeating that hash inside dispatch: bind a
fully verified immutable file to stable metadata, and require unchanged metadata
at the final check. Keep the 30-second runtime freshness bound.

The first member remained running before cleanup and exceeded the 300-second
SIGTERM grace period, exiting 137 without OOM. The anchor stopped cleanly with
143. The saved member log shows bootstrap progress, not a completed HTTP readiness
check. The member contract omitted `--init`, unlike the earlier passing fake-network
cases, and this trial launched nec before zod. Restore the known-working init and
zod-first fixture pattern, then test a fresh pair. Those differences are candidates
for correction; this trial does not isolate the cause of the failed graceful stop.
Do not claim successful two-member launch, readiness or clean ship shutdown here.
