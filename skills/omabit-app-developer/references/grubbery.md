# Grubbery application development

Select this path from actual framework/app/nexus source and repository
instructions. Freeze the framework commit and required components. Historical
exercise revision `ff2b66b58d1ed15f4aecbe7a065559028eb1fb7b` is evidence for that
exercise only. The proposed newer single-release candidate is a separate input;
neither revision establishes a timeless endpoint or compatible installed runtime.

Run `app-preflight --workflow grubbery` before app-specific project boot. The
current development slice reports `missing-workflow-adapter`: the versioned
Grubbery adapter is still pending. This reference describes the intended
qualification contract, not commands that already exist. Author source and tests
while that gap is open; do not substitute a private installer or raw Dojo/Eyre
write for an unavailable policy-enforced Omabit operation.

Current shared offline operations `app.framework.v1` and
`app.verify-framework-source.v1` provide the frozen `grubbery-7f7701d-v1` source
contract. The pin is `7f7701d546ce115eea5f37f66465b362cb121774`; its complete
`desk` subtree has1443files/8013304bytes. Select a source-root handle explicitly
at CLI/MCP startup, then pass that name and relative `desk` subtree for verification.
These operations execute/download nothing and grant no policy. Exact source
equality does not establish framework compilation, live permits or app readiness.
The source must still pass the independently bound staging/commit gates.

Framework Clay desk, app code tree/nexus, installed instance, loaded tree, permits,
HTTP binding, local alias and peer discovery are distinct bindings. Discover the
live framework revision, instance paths and endpoints from supported observation;
do not hardcode a historical `shell.shell`, port sequence or trailing-slash route.
Remote discovery success does not prove the local alias exists. Repair only a
missing alias with one uniquely verified claimant; a conflict must refuse.

The adapter must journal each resumable step: reviewed framework inputs, bounded
source stage/commit, compilation observation, app-tree delta, instance/load,
declared permit state, alias/route and served-artifact verification. Preserve
uncertain writes when the backend has no idempotency key. Bootstrap compilation
may outlive an HTTP response; resume observation instead of reinstalling.
An activation/version marker is the final publication step, after verifying actual
app-owned text/JSON/assets and fresh served bytes. Same-version updates still need
byte-level verification. Preserve state and unowned paths; stale owned-file pruning
requires an explicit bounded reviewed delta.

Prefer existing bounded staging/commit, which creates no Unix mount. Never leave
a full framework repository or dependency cache syncing during application use.
If a qualified framework version truly requires a mount, bind its exact ownership,
close it on success and failure, verify absence, and retain cleanup uncertainty
across process loss. Never unmount an unrelated developer mount. A small source
tree does not bound the event history created by a repeated synchronization loop.

Separate Omabit desk-creation/Commit/Publish policy, Eyre authentication and
Grubbery app permits. Only explicitly declared app permits may be approved through
their supported authoring path. Qualify registry-refresh/instance-reload ordering
for the pinned revision; do not assume permit persistence proves effectiveness.
Check app build/process errors and each required readiness probe separately from
unrelated bundled tools. Do not call the whole framework qualified when the app
works but other components fail.

Completion requires native deploy/update/restart with retained state, exact served
client hashes, two-ship discovery/publication and a three-member project workflow,
plus lost replies, slow compilation, missing/conflicting aliases, partial deploy,
stale client and failed mount-cleanup fixtures. See
[testing and recovery](testing-and-recovery.md). Ordinary app traffic remains
direct to Eyre; the development adapter is not a gameplay proxy.
