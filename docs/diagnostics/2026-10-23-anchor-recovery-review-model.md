# Read-only anchor recovery review model

The review retains the old runtime binding and two complete observations of a
present stopped anchor or the same anchor in a changed namespace. Every member
must still be running in the old namespace with its exact current creation and
initialized pier. It rejects healthy projects, missing resources, mixed/migrated
members, changes between reads and invalid process/storage evidence. It never
adopts a namespace or authorizes Start, Stop, removal or HTTP access.

Validated current-readiness receipts can pin process starts. A same-creator
historical receipt remains useful after its peer changes. Original creations may
lack historical start metadata; in that case the review proves only continuity
between fresh inspections, not uninterrupted prior uptime. Replacement starts
remain bound to the retained resume. Callers must supply all known process
receipts and fence pending or uncertain work; persistence/adapters remain separate.

The complete projectnetwork race suite passes in6.752s and vet passes. Fault tests
cover foreign ownership, state/namespace/process/pier changes, timing, driver buffer
aliasing and incomplete history. One intermediate edit assertion failed before
writing source; its log is preserved. Corrected edits and subsequent tests passed.
This is source evidence only. VM26 is an earlier frozen lifecycle fixture.
