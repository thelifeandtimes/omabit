# Private lifecycle persistence proposal

M377, 2026-10-25. The candidate adapters are staged and tested through Go overlays.
They are not installed in the working source or activated in the database format.
Schema remains42; the original42 writer/fixture from M376 remains the upgrade baseline.

The proposal has three tables: separately attributed workspace observations,
first gateway initialization and separately attributed gateway observations. Each
operation and its canonical journal mirror commit together. Actor/request replay
retains the original selection; unknown creation/Start outcomes do not repeat.
Restart reconciliation changes unfinished operation status without changing its
resource journal or dispatching a runtime.

Workspace observation preserves the failed preparation and can resolve only a
complete retained layout. Gateway observation preserves the uncertain initializer
and can resolve only readiness after a recorded Start dispatch. Failed inspections
can receive a new attributed inspection request; completed or active inspection
receipts prevent another concurrent attempt. Saved readiness identifies both the
original resource owner and the observation that established readiness. It is not
fresh runtime authority for member launch.

The corrected source passes25 top-level/161 including-subcase race checks, plus
vet. Tests exercise transaction rollback, interruption at each gateway phase,
actor replay, canonical mirror/source corruption, changed observations and a real
temporary directory tree after a simulated lost layout reply. Original42's two
workspace states also exercise cross-project recovery ownership.

Review found that recovered workspace directory claims needed comparison with
other projects, including completed original workspaces. The added check covers
workspace parents within each project and chain/key directories. A negative
control disables that check and makes the dedicated regression fail as expected.
The corrected implementation passes. Public key claims are also checked across
initial and recovered gateway readiness; historical namespace inode reuse is not
treated as a permanent identity collision.

Next: activate all three tables together under schema43, hook global validation
and startup interruption, and qualify the real migration and original42 refusal.
Shared storage provisioning, concrete gateway adapters/queues, first-start HTTP
publication, concurrent private-project runtime and public orchestration remain.
No Docker, installed service, production ship or existing plugin was accessed.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-25-private-lifecycle-store-proposal`.
