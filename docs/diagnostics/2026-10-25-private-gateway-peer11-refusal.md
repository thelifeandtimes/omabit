# VM11 gateway readiness and peer checkpoint refusal

The fresh no-NIC VM cleanly shut down after its original two-ship messaging,
persistent state and receiver-restart case passed. The appended whole-gateway
case failed. Four independent read-only audits preserve both disk hashes.

The retained gateway was created, reached readiness and passed read-only RPC
checks. Peer recovery then refused before creating either replacement ship:
its observed namespace string was `net:[4026532247]`, identical to the historical
string saved before the original gateway stopped. The exact hashed exception
maps to the fixture's historical namespace inequality check. Cleanup stopped the
replacement gateway with exit0 and no OOM. This is not a full successful recovery
proof; the final controller result was not recorded and the peer gate never ran.

Namespace inode strings describe live namespace objects. They are not persistent
project generations. Namespace lifetime depends on live references, as described
in [Linux namespaces(7)](https://man7.org/linux/man-pages/man7/namespaces.7.html).
We infer number reuse from the observed equality, clean predecessor shutdown and
fresh exact Docker container with network=none; no retained namespace descriptor
was available to prove historical object identity directly.

The corrected guest fixture permits a repeated historical number while recording
that fact explicitly. It still requires distinct exact gateway container IDs,
clean predecessor shutdown, new network=none ownership, separation from the live
guest namespace, and each peer's membership in the replacement namespace. The
audit binds the guest namespace to the retained selection receipt. Tests cover
recycled numbers, host sharing, incorrect generation evidence and reused gateway
IDs.21 top-level Go tests /117 including subcases, vet and54 Python tests pass.
One initial test command named a nonexistent Python module; its failed log is
retained and the corrected command passes. Fresh runtime qualification follows.

The outer boot wrapper returned143 without its own final receipt. Its cause is
unresolved. The inner VM result and independent disk audits establish QEMU exit0,
clean shutdown and inactive scopes; no bootstrap was retried on this disk.
The disk is privately retained with full hash verification and its original path
preserved. No raw keys, sessions, piers, databases or application values enter
this evidence directory. No production or installed plugin state was operated on.

M364 supersedes M361–M363's pending VM11 status, and corrects the initial diagnostic
interpretation that resume had failed before readiness. It also supersedes the
historical-number inequality assumption in M358's fixture, not the host-network
isolation requirement or the pending complete gateway/peer qualification.
