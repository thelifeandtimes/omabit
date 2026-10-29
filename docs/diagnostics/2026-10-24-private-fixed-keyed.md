# Experimental Vere correction passes keyed stop/reopen

The original pinned signal code fails the new behavioral regression after its
first successful soft call (one delivery). The minimal corrected success/error
cleanup paths preserve seven deliveries across three success/failure rounds.
Pinned boot and noun tests also pass. This is an actual SIGTERM/libuv behavior
check, not a source-text assertion.

A separate source archive of5a7614f is compiled with Zig0.15.2 as
`4.6-5a7614f-omabit-signal-probe`. The source change moves the second cleanup into
the error branch, preserving each path's existing cleanup point. Build metadata
uses an explicit experimental version and adds the regression target. Two expired
OpenBSD archive URLs use official GNU mirrors with unchanged Zig content hashes.
TLS CA mapping and404 attempts are retained; no verification was disabled.
The candidate SHA256 is
`1b4b5c6a3c3ad8fd06570d0d21a696ce46c37f8a454d00341327bc3038e9680d`.

With this candidate only, a fresh private keyed~zod validates through the
chain-backed gateway, reaches identity, catches SIGTERM and exits0 in4007ms.
The same pier reopens without additional bootstrap requests, preserves identity,
and exits0 on SIGTERM in4286ms. Final databases and all source hashes are
independently verified; no force is used. Both new network keys pass the earlier
positive/negative validator cases. The private scope is inactive.

This supersedes the pending experimental-fix gate, not the official runtime's
failure. The candidate is neither installed nor released; production code, image,
ships and plugin installations remain unchanged. Persistent chain observation,
two-keyed-ship messaging, Docker integration and release qualification remain.
Secret files and raw bootstrap logs stay excluded from copied metadata.

Build evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-vere-signal-fix-build`.
Runtime evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-private-fixed-keyed`.
