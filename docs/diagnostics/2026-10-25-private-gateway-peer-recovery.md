# Whole private gateway and keyed-peer recovery passes

VM12/appliance13 passes the full retained-gateway experiment. The original two
keyed ships exchanged messages and retained four application writes through a
receiver replacement. A fifth write was then deliberately left unread by a
separate subscription. Both ships and the original gateway stopped cleanly.

A distinct gateway container reopened the exact retained chain, with no
initialization, key generation, mining or transaction submission. The genesis,
head hash, contract code and public keys matched the predecessor's final proof.
Both exact piers reopened in new containers with their original Eyre sessions.
The saved channel/cursor delivered revision5; subsequent writes advanced the old
subscription to6 and a fresh subscription to7. Independent observations cover
five scries and five snapshots (4,5,6,6,7), with no replayed writes. Both channels
were deleted, and cross-ship sessions were refused.

The historical network namespace number repeated again, while the exact gateway
container changed and remained separate from the guest namespace. This validates
M364's correction: historical inode inequality was an invalid generation test.
Current peer membership, distinct container ownership and host separation still
passed. Five ship-process shutdowns and both gateway shutdowns returned exit0
without OOM. The no-NIC VM powered off cleanly in1975.13 seconds.

A separately built read-only audit VM passed in11.71 seconds. It checked the
retained receipts and preserved both backing and overlay hashes. Both outer
wrappers returned0, and both exact scopes are inactive. The original overlay is
retained in the task's0700 private Docker lab with a hash-verified compatibility
link; keys, cookies, piers, chain data and raw logs were not exported as public
evidence. The only payload change from VM11 was the namespace correction.

This supersedes M364–M366's pending whole-gateway runtime status. It completes
this transport/state/session recovery gate, not private endpoint publication,
simultaneous private projects, supported runtime distribution, durable runner
initializer/resume journals or public runner dispatch. The schema41 API still
reserves private plans only. HTTP publication in another isolated VM is next;
the currently installed plugins, production ships and host Docker remain excluded.
