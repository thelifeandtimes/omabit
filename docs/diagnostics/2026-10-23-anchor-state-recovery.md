# Actual-ship anchor state/subscription recovery — 2026-10-23

Fresh private VM23-08 passes the corrected published two-ship recovery case.
Independent verification checked65 direct published Eyre requests, six unique
acknowledged persistent writes,19 state observations and nine subscription facts
at revisions0,1,2,3,3,4,4,5,6. The normal same-pier receiver replacement and both
message directions pass before the extra anchor recovery phase.

Stopping the anchor made both published endpoints unavailable (curl7/status000).
Restarting the same anchor ID produced a different network namespace. Both old
members remained running in the original namespace, and the new publications
returned no HTTP response (curl52/status000, empty bodies). The fixture stopped
both old members before creating either replacement. Both replacements reused
the original pier objects and joined the new namespace. Original Eyre sessions
and the existing subscription recovered the queued fifth write; a sixth write
arrived on that same subscription without replaying previous writes.

All three channels were deleted, both sessions revoked, all five ship-process
stops succeeded, the anchor stopped without OOM, and the VM powered off cleanly
in1349.52 seconds. IPv6 publications and bridge access to administrative ports
remained refused. The16 extracted receipts, original backing/overlay disk hashes,
console and full guest result were independently verified. Original disks and
36 metadata files are retained under the evidence directory in the JSON report;
only the verified regenerable inspection conversion was removed.

This closes the fresh-fake actual-ship anchor recovery experiment. Trial05 remains
failed evidence with its exact curl outcome unknown; trial08 does not rewrite it.
The demonstrated recovery ordering must now be integrated into runner member
lifecycle/recovery. No automatic recovery, production access or public API is
qualified by this experiment. The guest has no external NIC.
