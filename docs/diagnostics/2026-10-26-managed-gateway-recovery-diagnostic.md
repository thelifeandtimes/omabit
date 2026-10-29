# Managed private gateway recovery passes the diagnostic VM

M415, 2026-10-26. Fresh no-NIC VM07 passes initialization, recovery and independent
manager reopen verification in935.288s, then shuts down cleanly. Its84.797s
read-only audit passes with the same disk/base hashes. All owned units are
terminal, and the exact disk is privately retained with a verified compatibility
path. No protected ship, installed plugin or host Docker resource was accessed.

The fixture deliberately loses the first successful readiness reply. The saved
original operation remains interrupted with the original started network/container
selection. A separate observer actor creates the recovery receipt; two subsequent
manager processes preserve that receipt and the original state, verify actor-bound
replay and refuse another actor. The audit confirms four operations, one recovery
observation, unchanged filesystem identities and zero member instances/boots.

The corrected RPC probe succeeds at2227ms during initialization and2059ms during
recovery. Final full input verification succeeds at9976/10260ms within the existing
observation contract. No deadline, ownership, image, mount or key/chain acceptance
rule was relaxed. The initializer's container later exits0 without OOM. Only fixed
stage/timing/boolean metadata and hashes are exported; key/chain contents remain
private. The separate auditor's57.098/11.397s input scans are not live timings.

This run uses the tested logging-only overlays. It demonstrates the current
managed gateway initialization/lost-reply recovery path, not ordinary-binary
release qualification, member launches, whole-gateway restart or public runtime
orchestration. Fresh ordinary-source VM08 is being prepared without the overlays.

M415 supersedes M411's pending diagnostic-run status and demonstrates M410's
corrected probe in that runtime. M411's earlier pre-start cause remains unknown
and is not reproduced here; its failed evidence remains valid. M412–M414 remain
separate source-only member work.
