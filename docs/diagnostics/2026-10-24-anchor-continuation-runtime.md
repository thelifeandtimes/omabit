# Interrupted recovery, application state and subscriptions: VM34

The corrected schema40 no-NIC VM passes in2505.19s with outer exit0, clean
shutdown and independently verified stopped-disk evidence. Three deliberately
lost replies (Stop, Create with its returned ID, and Start) retain their exact
checkpoints across three store closes/reopens. Two explicit observations and
three explicit continuations finish the original recovery. Each of the two
members is stopped, retired, created and started exactly once in that recovery;
uncertain actions are not replayed and old creator attribution is preserved.

Sixteen remote application writes and four state/subscription recovery cycles
pass across member and namespace replacement. Both original Eyre sessions remain
usable, with two initial logins and no later login. Existing and fresh subscriptions
reconcile retained/queued updates. Both final piers' identities, log/checkpoint
files and hashes are independently checked from the stopped disk. Exact-ID
fixture shutdown completes cleanly and original disks are retained.

The actual public saved-status CLI returns text and JSON matching the exact stored
operation projection, without changing the database or claiming fresh health.
SQLite mirrors, checkpoint digests, continuation chains and effect counts match
the extracted receipts. The original VM34 appliance and sources are retained
separately; it predates M188–M191 public recovery commands and their fixture.

Extraction initially refused the temporary quota before effects. Clearing only
the dedicated Go build cache supplied its required reserve. The first audit
incorrectly compared the parent message across a successful observation; the
corrected audit requires the exact documented message change and every other
field unchanged. A retry then refused an existing derived JSON; the successful
pass used a fresh output directory. All attempts and derived files are preserved.
No runtime or original disk was changed to satisfy the audit.

This supersedes M174/M175 and M187's pending private runtime continuation gate
and M184/M185's pending corrected rerun. VM33 remains a failed trial. Store reopen
is not an OS service restart; final Stop is fixture cleanup. Actual public mutation
CLI and UI qualification remain separate. The prepared public recovery appliance35
will test the former next. No production ship, installed plugin or service changed.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-anchor-continuation34`.
