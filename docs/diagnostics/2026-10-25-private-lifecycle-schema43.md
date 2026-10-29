# Schema43 private lifecycle persistence

M378, 2026-10-25. The staged M377 adapters are now integrated into the isolated
checkout under schema43. Three tables retain gateway initialization and separately
attributed workspace/gateway observations. The normal migration, shared reservation
validation and startup interruption paths all include these records. Public runtime
dispatch is still absent.

Completed and interrupted original42 workspaces retain their original rows,
operation fingerprints, mirror bytes and sidecars. Recovery adds a separate
inspection receipt; it never changes an unknown initializer into a completed
attempt or repeats Create/Start. Startup phase tests now close/reopen the actual
store and run the normal reconciliation path twice. Canonical corruption refuses
open without modifying the database. A failed migration leaves the marker at42
and rolls back all three new tables.

The frozen original42 writer reproduces its fixture exactly. The current exporter
then produces genuine schema43 state with two rows in each new table: completed
and interrupted initialization plus failed/completed observation histories. The
unchanged original42 reader refuses that populated successor without modification.
The development reader inventory includes42; historical RC.12 release claims remain
unchanged. This does not qualify an installed upgrade or a release.

The focused checks pass51 top-level/280 including-subcase race tests and vet.
The first preservation test incorrectly required the operations table to retain
its exact size after six intentional additions. Its corrected helper compares every
original operation field by its primary key and separately checks the new count.
Both the initial failure and corrected evidence are retained. A negative control
removing the cross-domain post-mutation check makes the directory-claim regression
fail as intended; the integrated check passes and rolls the invalid result back.

The full managed/privatechain race regression is running as session13388, with a
frozen source manifest. It is not yet a pass. M374's earlier full42 result cannot
qualify this change. Internal lifecycle queues, concrete initial gateway adapter,
first-start HTTP publication and concurrent private-project runtime remain next.
Appliance16 remains unbooted. No Docker, installed service/plugin, production ship,
tmux ship or excluded pier directory was accessed.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-25-private-lifecycle-schema43`.

Update: [M381](2026-10-25-private-lifecycle-integrated.md) supersedes the pending full-regression/queue-integration statements above. The original checkpoint remains retained for provenance.
