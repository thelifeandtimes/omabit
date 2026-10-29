# Endpoint fixture trial18 — 2026-10-22

**Failed; retained.** The no-NIC guest powered off cleanly after 219.14 seconds,
but the endpoint gate failed on `unexpected anchor publication` before creating
any member or sending an HTTP request. Cleanup also reported that the sleep
anchor did not stop cleanly. The original fixture did not save the raw mapping
or stop state, so this is not enough evidence to attribute either failure to a
specific Docker behavior. No publication or anchor-recovery claim is qualified.

Independent verification confirmed all original images/overlays, 18 extracted
files, result hashes, scope admission/inactivity and the absence of timeout,
quota or log-limit failures. [Evidence](2026-10-22-endpoint-trial18.json) and originals
are retained under
`/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-22-endpoints18/`.
The endpoint result SHA256 is
`7b13c8f1618612f95db760f6f2658539b82afacb4c4a719b10ae4a6317e76f39`.

The corrected fixture records exact requested/active mappings, network attachments
and stop states before assertions. It uses Docker init to forward SIGTERM to its
sleep anchor rather than assuming PID1 sleep handles it. Three offline fixture
tests pass. A fresh internal-bridge rerun is required; changing to an egress bridge
without understanding this observation would conflate topology and fixture issues.
No real ship, installed plugin, host engine or production configuration was used.
