# Durable initial anchor Start — 2026-10-23

Schema25 adds a separate initial-Start journal and atomically mirrors it in the
operation receipt. The original completed creation journal remains immutable.
Acceptance requires the reviewed project revision and successful preparation.
A single worker claim precedes inspection; dispatch, acknowledgement and observed
namespace are separate durable commits. Reopening interrupts unfinished work,
retaining its original phase. Neither replay nor a new request starts it again.
Anchor success explicitly says no member was started.

Synthetic SQLite/inert-engine race tests verify every commit boundary, atomic
mirror rollback, concurrent claims, actor-scoped replay, corruption refusal,
lost replies and failed post-Start inspection. The affected project/historical
race suite passes (13.150 seconds); vet passes. These do not qualify real Start.

The frozen original schema24 writer produces a populated completed-preparation
fixture. Byte-for-byte regeneration passes. Its original reader refuses real
schema25 state containing a dispatched anchor Start without modifying it. The
unreleased source snapshot is explicitly separate from release provenance; RC.12
claims and unavailable prototype-writer gaps remain unchanged. No installed
database or production resource was accessed. The actual adapter and private
server scheduling are the next integration work.
