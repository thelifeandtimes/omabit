# Journal-driven supported collection recovery

The internal `native/collections/coordinator.py` performs one member/phase at a
time under the journal lease, with durable intent already recorded. Busy retains
the current phase; known problems persist. Unconfirmed helper lineage prevents
further adapter calls. Unexpected errors leave the phase pending. Rollback remains
explicit. There is no user-facing activation command or production loader adapter.

The optional `--collection-journal --connected --drain --omabit-source ...` mode
uses real Omabit/Theme/DMs QML and Theme/DMs helpers, copied native loader functions,
a private keyring and synthetic Eyre. Omabit tools and desktop controls are inert.
Private component copies expose captured source URL, profile, creation instance
and activation epoch; these observations are qualification instrumentation, not
an installed plugin API. Artifact trees are hashed and mounted read-only. Exact
helper process identities are retained before draining. Endpoint readback is bound
to the private server and profile/session owner, not a production endpoint claim.

Two applied runtime runs complete stable→dev→stable, then an edge font-apply
failure and explicit rollback to stable. A new coordinator process handles every
step. Six forced SIGKILLs occur after effects but before journal receipts: previous
drain, disable, checkpoint, candidate enable, readiness and successful restoration.
Fresh processes resume each pending phase. Candidate-enable replay retains the same
loaded instances. A deliberately failed restore remains pending; successful retry
uses the exact retained checkpoint, and replay after coordinator death adds no
desktop effects. Actual offline Theme CLI calls forbid network/credential access.
All16 channels, including temporary Theme publication channels, are deleted.
The final collection is explicitly drained for teardown, leaving a pending
checkpoint phase rather than claiming the unloaded desktop is currently ready.

The first trial failed before journal adoption: native loader objects complete
before shell/manifest properties are assigned. The private identity probe now waits
for those assignments and captures once. The failure is retained. The final run
also refuses a stale activation binding. Thirty private journal/coordinator checks
pass, along with the existing connected zero-write lifecycle baseline. Source
hashes, journal replay and inactive scopes were independently verified.

M270 supersedes M269's model-only evidence for these private runtime paths. Whole
shell loss/replacement, partial candidate loading, production loader observation,
real collector/SSH/clipboard effects and native compositor/fanout remain separate.
Individual plugin collectionReady flags still remain false. The private adapter
does not qualify arbitrary plugins or a release-ready collection switch. Existing
unknown DM-send/effect coverage remains M268, not this appearance-focused scenario.
No installed plugin, production credential, ship or desktop was changed.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-collection-coordinator`.
