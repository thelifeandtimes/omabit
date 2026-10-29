# Project preparation controller — 2026-10-23

The private controller now sequences bridge and stopped-anchor creation through
the schema24 journal. Every effect requires the prior commit, every inspection
requires a retained returned ID, and every resource/engine check must match the
original binding. A valid returned ID is retained even if the client also reports
an error. A missing reply remains uncertain. Cancellation and failure issue no
inverse action or automatic retry. Nonzero sequences require explicit recovery.

The private managed worker connects these callbacks to SQLite and marks incomplete
work interrupted while preserving its journal. Tests read the real stored record
inside each inert engine call, proving the intended ordering. Repeat workers and
original request replay cannot cause another engine effect. This entry point has
no server/RPC scheduling yet; future dispatch must hold the runner-wide mutation
and management-update fence.

The existing private Unix engine probe now retains Docker's exact daemon ID.
New project work can require that ID independently of a socket/data-root path.
Historical reports without an ID retain their earlier compatibility behavior and
omit the new JSON field. Malformed IDs are refused without echoing provider text.
The private socket probe and affected runtime compatibility race tests pass; an
initial sandbox-limited run and the successful permitted fixture run are retained.
Schema22 reproduction and unchanged refusal of schema24 also pass.

This is source/inert-engine evidence. A real Docker adapter, resource containment,
member launch, readiness collection, recovery, RPC/CLI and UI remain unfinished.
No host engine, installed plugin, existing ship or live database was accessed.
