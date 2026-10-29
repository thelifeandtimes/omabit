# Private lifecycle queue source

M379, 2026-10-25. Two new files are staged outside the checkout and tested through
a Go overlay. Schema43 persistence from M378 remains integrated; its separate full
managed/privatechain regression is still running. It does not include these queues.

Gateway initialization, workspace observation and gateway observation share the
runner's work gate and management-update worker count. Receipt replay precedes
backend availability or input inspection, so a retained result can be read without
reopening a driver. Fresh workers revalidate their exact queued operation before
opening backend-selected resources. Startup does not schedule these workers.

Cancellation, unavailable/nil/typed-nil handles, partial setup, driver failure,
cleanup errors, corrupt mirrors and retained maintenance all have explicit tests.
Every returned owned handle closes once. Observation interfaces expose no creation
methods. The real workspace factory uses held directory objects; a synthetic lost
layout acknowledgement over actual temporary directories now recovers through the
queue, preserving the original interrupted workspace/operation bytes and empty
pier parents.

Six top-level/63 including-subcase race checks and vet pass. Trial1 correctly
refused a test's already-observed workspace; the test now uses an actual interrupted
layout. Trial2 caught a missing test brace before execution. Both failures and their
source snapshots are retained. The gateway driver remains inert in these tests;
no concrete engine/input adapter or public route was introduced.

This supersedes M378's pending scheduler source only. Integrate after the unchanged
full43 regression completes, then check affected source. Concrete first gateway
startup/publication, shared storage provisioning, concurrent project runtime and
public orchestration remain. No Docker, services, installed plugins or protected
ship state was accessed.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-25-private-lifecycle-queues`.

Update: [M381](2026-10-25-private-lifecycle-integrated.md) supersedes the pending full-regression/queue-integration statements above. The original checkpoint remains retained for provenance.
