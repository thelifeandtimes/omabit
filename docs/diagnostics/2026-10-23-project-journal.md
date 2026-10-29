# Durable project preparation journal — 2026-10-23

Schema24 records project preparation and its actor-scoped operation atomically.
Before dispatch, both resource contracts, the selected engine, reviewed plan,
project revision and generation are retained. Each bridge/anchor creation advances
through planned, dispatched, returned-ID retained, and independently observed.
The anchor cannot advance before the bridge is observed. This is initial resource
preparation, not member launch or whole-project readiness.

Each update compares the current operation and exact sequence, and commits both
the project journal and operation mirror together. A failed mirror write rolls
back intent. Concurrent writers cannot dispatch the same step twice; old callbacks
cannot alter interrupted or completed work. A lost reply retains unresolved intent
and blocks fresh preparation. Restart marks queued/running work interrupted without
changing resource evidence or issuing commands. Original actor/request replay
returns that receipt. Missing or conflicting journal/receipt records refuse opening.

Eight interruption boundaries pass race tests, along with replay, stale/reordered
callbacks, retained IDs, corruption, rollback, concurrent writers and preservation
of member reservations. The full managed race suite passes with private test socket
access; projectnetwork/protocol suites and vet pass. An initial sandbox attempt
could not create the test sockets; that log is retained separately from the pass.

The frozen original schema23 writer reproduces a fixture containing its own
project reservations. Its reader refuses a real schema24 database containing a
pending bridge dispatch without modifying it. Older checked-in migration fixtures
also pass. The previously missing preview18 source object still limits full
original-writer regeneration. Frozen release/host support claims remain unchanged.

Source, test logs, fixture archive and digests are retained at the path in the JSON
report. No live database was upgraded. The store API is private: Docker adaptation,
worker dispatch, member runtime, explicit recovery, capabilities and CLI/UI remain
unfinished. Opening the store never creates a resource or starts a ship.
