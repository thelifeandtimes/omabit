# Project member mount-conflict preflight — 2026-10-23

The private negative preflight inventories every exact listed container, including
created/stopped ones, without reading application environments or credentials.
It rejects overlapping current source objects and ancestor/descendant binds,
symlink aliases, missing/noncanonical sources, unsupported volume/plugin sources,
incomplete/duplicate/changing inventories and filesystem replacement during checks.
The only exception is the current dispatch's exact never-started member container,
whose full contract is independently checked before and after inventory. Its
mount order may differ without changing the binding. No name-based adoption occurs.

Twenty-four inert inventory scenarios and temporary-filesystem ancestry/replacement
tests pass, alongside the focused project race suites and vet. An edit-script
assertion was corrected before the final regression run; no runtime test failed.
Original temporary objects in replacement fixtures are retained until test cleanup.

This is a negative gate, not proof that storage is safe to launch. Docker's saved
source path can refer to a different object after a running bind was moved or
replaced. Durable lifecycle ownership and pier locks remain necessary; the helper
does not authorize adoption or settle uncertain active kernel mounts. Unknown
volume sources intentionally refuse instead of treating a Docker mountpoint as
proof of the underlying source. No real Docker daemon, ship, schema or installed
component was changed. Member launch journaling and actual runtime gates remain.
