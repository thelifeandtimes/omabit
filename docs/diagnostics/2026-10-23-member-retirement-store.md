# Durable stopped-member retirement — 2026-10-23

Schema31 stores explicit retirement intent separately from the original Stop and
clean exit observation. Each exact member/container can have only one retirement.
Planned, removal-dispatched, removed and absent phases retain their process/pier
bindings and atomic operation mirrors. Restart interrupts unfinished work without
replaying removal or creating a replacement. Only acknowledged complete absence
can finish successfully. Failed transitions preserve their previous journal.
Original Stop and exit receipts remain unchanged, including interrupted Stop status.

Restart at every commit boundary, actor/request replay, duplicate requests,
concurrent admission, invalid/stale transitions, incomplete/unclean/foreign exits,
foreign-key refusal, corrupt mirrors/orphans and worker failures pass. The complete
managed race suite passed in224.008 seconds; vet is clean. The original schema30
writer is pinned from the pre-change source snapshot. Its populated179499-byte
fixture reproduces byte-for-byte and refuses real schema31 state unchanged.
Only this original fixture uses the192KiB limit; older limits are preserved.

This is source persistence/worker qualification. No installed database changed.
Queue admission, concrete runtime retirement, replacement/resume and public routes
remain separate work. VM17 is the earlier frozen schema30 Stop-exit recovery trial.
