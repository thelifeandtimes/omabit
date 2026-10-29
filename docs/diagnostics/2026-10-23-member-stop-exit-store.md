# Durable read-only Stop exit observations — 2026-10-23

Schema30 records each read-only exit observation separately from its original
Stop. Planned, observing and observed journals are mirrored atomically in their
operation receipts. Restart interrupts unfinished observations without replaying
effects. Exact actor/request replay preserves the receipt; a new observation keeps
all prior attempts. Completion requires fresh, fully bound process/runtime/storage
evidence. A successful observation can record an unclean exit; it does not change
the original Stop's outcome or release its uncertainty fence.

Focused restart, replay, concurrent admission, invalid completion, corruption,
foreign-key refusal and worker tests pass. The complete managed race suite passed
in173.156 seconds, with vet clean. Original schema29 sources were frozen before
this change; their populated149090-byte fixture reproduces exactly. The original
reader refuses actual schema30 observation state without changing it. Schemas11–29
remain accepted while historical data is preserved. No installed database changed.

This qualifies persistence and the private worker only. Queue integration, real
recovery, resume and public access remain separate gates. VM16 uses the earlier
schema29 initial Stop implementation. Evidence and hashes are in the paired JSON.
