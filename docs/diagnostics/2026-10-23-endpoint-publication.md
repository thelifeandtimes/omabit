# Ordinary-bridge endpoint publication — 2026-10-23

**The HTTP fixture passed.** Two project anchors on separate ordinary bridges in
the no-NIC VM published four exact IPv4 loopback endpoints on rootless Docker
29.9.1. All 28 HTTP observations matched their intended fixture identity; four
IPv6 attempts and four attempts to reach unpublished endpoints across bridges
were refused. Requested and active publications were both verified.

Stopping anchor A left its two members running in the old namespace with working
local endpoints, but guest loopback publications became unreachable. Restarting
the same anchor container created a new namespace: `net:[4026532239]` became
`net:[4026532298]`. It did not reconnect the old members. Explicitly stopping and
replacing both members in the new namespace restored A's endpoints; project B
kept its original endpoints. This supports coordinated project maintenance and
rules out treating anchor restart alone as recovery.

Nine clean stops were verified: six HTTP responders exited0; the init-managed
anchors exited143 (A twice, B once). The VM powered off cleanly in269.16 seconds,
without quota, deadline, log-bound or cleanup failure. Original disks/artifacts,
16 extracted files and36 metadata files are independently verified and retained at
`/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-23-endpoints20/`.
Result SHA256: `5f72c7958a470d24eec33f89398c1f90307b2e521e83fec58bd850953439f4c9`.
[Evidence](2026-10-23-endpoint-publication.json) includes exact mappings and stops.
Member liveness/namespace checks are asserted by the frozen source; not every raw
inspect response is duplicated in the receipt.

This supersedes the pending ordinary-bridge experiment after failed internal-
bridge trial19. Both earlier failures remain retained. The result qualifies HTTP
fixtures only: actual Urbit traffic on this bridge, runner ownership/integration,
hostile traffic, external reachability and anchor power loss remain separate gates.
No production engine, existing ship or installed plugin was accessed. An ordinary
bridge is now a candidate for the next fresh-fake publication test, not a shipped
runner capability or a general egress-isolation claim.
