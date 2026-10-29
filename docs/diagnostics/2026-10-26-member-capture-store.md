# Staged capture persistence and original-reader refusal

M445, 2026-10-26. The schema46 Go overlay extends M444's version3 launch model
with an atomic canonical capture journal and operation mirror. It is not yet
integrated; repository schema45 and the active M443 VM remain unchanged.

Capture receives a global pier/.urb ownership claim before later observation.
The running record must reuse that capture. Freshness, immutable ancestry and
actor-scoped replay remain enforced; reopening interrupts unfinished work without
dispatching another Start or acquiring new pier identity.

47 unique selected race checks and vet pass. Original45 rows, sidecars, runner
identity and replay survive migration alongside a new version3 project, across
three opens for running, captured and started cases. Tests cover mirror-write
failure rollback, future capture, changed selection, duplicate ownership and
corrupt-original migration rollback. The initial combined test invocation hit
its 300-second harness timeout after completed cases; bounded separate batches
completed the remaining coverage. Runtime deadlines were not changed.

A separate opt-in test exports an actual migrated schema46 database containing
an interrupted captured launch. The frozen original45 source (2,762 verified
files) refuses this populated successor without changing its bytes. Neither its
fixture nor the database header was relabelled to manufacture this result.

This advances M444's pending persistence work, but does not supersede M441's
runtime limitation. Concrete adapter/session forwarding, new-admission wiring,
historical support-window tooling, integrated validation and a fresh VM remain.
Separately attributed recovery must use saved capture without repeating Start.
Private evidence: omabit-qualification-evidence/2026-10-26-member-capture-store.
