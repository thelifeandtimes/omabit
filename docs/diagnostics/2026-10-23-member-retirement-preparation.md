# Guarded stopped-container retirement fixture — 2026-10-23

The new fixture follows the qualified lost-Stop-reply recovery with a separately
queued retirement of zod's exact cleanly exited container. It checks dispatch
before removal, acknowledgement before absence, unchanged original Stop/exit
receipts, replay, refusal of another request and database reopen. It hashes every
regular event-log/checkpoint file before and after retirement. Cleanup checks the
removed target's absence and retains its historical clean exit without attempting
a second Stop or pretending the pre-removal inspection is current.

All four ordinary VM tests skip. Explicit host enablement refuses at the guest
marker before effects. Compilation and vet pass. Appliance18's136 frozen sources,
artifact hashes and embedded test independently match; verified staging is removed.
Its schema31 binary is pinned in the paired JSON. The independent offline verifier
also compares the retained state file set, modes, sizes and hashes with guest-disk
contents, alongside SQLite/receipt, process and complete container inventories.

Actual retirement remains pending. The fixture qualifies neither replacement nor
public project routes. Production ships, installed plugins and host Docker remain
excluded.
