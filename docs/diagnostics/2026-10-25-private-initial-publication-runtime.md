# First-start private publication runtime — 2026-10-25

M384: fresh VM15 and independent stopped-disk audit30 pass. The VM has no NIC
and uses its own rootless engine, fresh keyed zod/nec and copied immutable inputs.
The run takes 2,006.44 seconds and the reader 11.62 seconds; both exit cleanly.
Actual admission limits, clean shutdown, inactive scopes and unchanged backing
and overlay hashes are independently checked. The wrapper also pins all 17
source files before launch and audit. M383's separate source work changes none.

The original four-write messaging/receiver-restart recovery passes. First-start
publication then uses the original separate sessions through the guest's IPv4
loopback ports: 23 direct requests, two further writes, four SSE reads, three
scries and eight channel PUTs. The application moves from revision4 to revision6
with exact state, acknowledgements, ordered cursors and both channels closed.
No write is replayed and neither client logs in again.

Cross-ship cookies refuse. Five unpublished endpoint probes refuse RPC,
administrative access and IPv6. The audit binds the first and last bridge/port
observations, initial/final chain head, original recovery baseline and private
session-file hashes. It exports fixed metadata and hashes only, with no session
values, key material, application entries or pier/database contents.

Both fresh ships and their gateway shut down cleanly. The 3,665,690,624-byte disk
is retained in the private lab after independent scope checks and byte-for-byte
verification; its original path remains a symlink. Raw logs and state remain
private. Evidence here contains only the public projection and audited source.

This supersedes first-start runtime/audit pending statements in M380–M383.
M372 remains the separate whole-gateway replacement-publication proof. It does
not establish concurrent project isolation, production host egress containment,
the concrete runner gateway backend, public lifecycle or a release. Concurrent
appliance16 remains unbooted because available memory is below its unchanged
32GiB admission floor. Continue container/publication and input/engine adapter
work, shared parent provisioning, then isolated runtime/public lifecycle tests.
