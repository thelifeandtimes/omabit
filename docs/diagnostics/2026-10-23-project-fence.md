# Project worker admission — 2026-10-23

The private server queue now admits the actor-scoped preparation receipt and worker
under the existing runner mutex, counts it as active management work, and waits for
the shared resource gate before opening an engine session. It rechecks retained
ownership and the existing unresolved-maintenance rule after acquiring the gate.
No public route, capability or automatic startup dispatch has been added.

Inert-engine race tests prove the queue waits behind existing work, prevents
management-update acquisition while waiting/running, dispatches once despite
request replay, and permits update acquisition after completion. Cancellation
while waiting or opening a session, setup failure and retained maintenance all
leave an interrupted receipt without creating resources or retrying. Closing,
failed, fenced and unsupported servers refuse before creating a new job.
The affected project/management-update suite and vet pass.

A separately guarded VM test now compiles. Ordinary host execution skips it;
explicit enablement additionally requires the exact QEMU/systemd guest marker,
no external network devices and the fresh lab owner. It uses a fresh retained
SQLite database and the real engine adapter, checks the journal from inside each
resource call, records exact-ID inspections and before/after volume inventories,
and checks unchanged receipts after database reopen. It starts no anchor or ship.
The appliance is being packaged; no actual-engine result is claimed yet.
