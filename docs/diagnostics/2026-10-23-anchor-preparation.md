# Actual-ship anchor recovery — prepared 2026-10-23

The separate `--network-anchor-recovery` published-fake case extends normal
receiver replacement with a coordinated anchor restart. It queues a fifth state
write, stops/restarts the same anchor and requires a changed namespace. Old live
members must remain in the old namespace; public endpoints must refuse connection.
Both old members are explicitly stopped before either replacement is created.
New containers must retain the same piers and join the new namespace. The existing
subscription must recover its queued event; a sixth write uses original sessions.
Cleanup deletes that channel, revokes sessions and checks clean container stops.

Four new inert checks cover ordering, unchanged namespace, unexpected member drift
and a member that did not stop. All41 network tests pass. Appliance23-05 and its
source/binaries are hash-verified; no runtime has executed for this case yet.
Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-23-anchor-preparation`.
This prepares M033's HTTP-only anchor result for actual Urbit testing. It neither
qualifies the runtime nor implements runner-owned resource journals. All operations
remain within a new no-NIC VM and its own fresh fakes; production stays excluded.
