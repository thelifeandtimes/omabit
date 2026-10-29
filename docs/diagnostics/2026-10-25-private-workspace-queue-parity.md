# Workspace queue replay and driver consistency

M382, 2026-10-25. A retained workspace receipt now returns before the runtime
backend is selected, matching the three newer private lifecycle queues. Fresh
requests still require a backend. Previously, removing backend support made an
otherwise valid completed receipt unreadable through this queue.

The older queue now uses the same admission/update fence, worker accounting,
exact queued-operation binding and owned-handle checks. A typed-nil driver, with
or without a setup error, is rejected without calling methods on it. The original
intent is interrupted and retained; no directories are created or operations
retried. Workers also reject changed actor/request/revision/project/operation
bindings before opening filesystem authority.

The new replay regression fails against the original source for the expected
missing-backend error. The corrected overlay passes13 top-level/91 including-subcase
race checks across old and new queues plus vet. After byte-for-byte integration,
seven workspace tests/28 including subcases pass against the checkout. These28
overlap the91; they are not additional unique coverage. A stale gateway source
comment was updated to describe the already-qualified schema43 persistence.

This supersedes M374's backend-before-replay and typed-nil gaps, and M381's staged
follow-up status. Schema43, public routes and runtime authority are unchanged.
M381's full baseline remains a prior source baseline. VM15 and mandatory audit30
continue separately in session54631; first-start runtime success is still pending.
No protected ships, installed plugins, host engine or service state was accessed.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-25-private-workspace-queue-parity`.
