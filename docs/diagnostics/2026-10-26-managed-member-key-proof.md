# Managed gateway recovery and member key proof pass

M426, 2026-10-26. Fresh no-NIC VM10 passes ordinary candidate installation,
private gateway initialization, lost-readiness-reply recovery, manager reopen
verification and all six member-key cases. It shuts down cleanly after
1238.719 seconds, with no timeout or storage failure. The independent
read-only audit passes in 73.822 seconds; both overlay and base hashes
remain unchanged. All three owned units are inactive. The disk and audit artifacts
are privately retained with byte-for-byte verification and compatibility paths.

Both fresh member keys pass the actual pinned Vere key/chain evaluation. For
each key, wrong-public-key and wrong-life controls explicitly return the negative
verdict, rather than merely failing a command. The actual keyfiles and chain are
unchanged. The audit binds all six retained receipts to the actual keyfiles,
public chain observations and host held-key verification, and checks that raw
key text is absent from the test log. It exports no key contents or individual
key hashes. There are zero member ship boots and zero saved member instances.

The independent gateway/history audit confirms four operations, the original
interrupted receipt, separately attributed recovery, two manager reopens and
unchanged filesystem identities. Gateway teardown reports exit0 without OOM.
This does not qualify acknowledged gateway Stop or whole-gateway restart.
Immutable-input audit scans take46.079 and11.888 seconds; those are offline
audit timings, not live observer freshness measurements.

The ordinary runner includes the M423 output-drain correction. Fresh installation
passed; this run does not separately prove that the ErrWaitDelay branch occurred.
M423's inherited-pipe regression remains the direct evidence for that branch.
VM09 remains a preserved failed installation; it was not retried or rewritten.

Build SHA256: 15403982af727a6fcf62c58b70d5b2b35435958f17fd908df91c7fae01da6fc3.
Runner SHA256: 0bba5147563141356c8bc8641516ff51b033955f8f9518ec7a68227d9f083c71.
Fixture SHA256: 1f5bf1f7f10bc1bbeabe2092c05fb0e0f47a5d4926483003986c202dc2dbc48b.
Auditor SHA256: 870c2846b71d0e3592545f9eee8078ba8bcbe3a17724d7209f841ebcb47c2e54.
Retained disk SHA256: f7ed721ece115dab3cf7e79b511127005302ae20967757af810af23ec1ad3895.

M426 supersedes M419/M420's pending actual key-proof gate and M423's pending
fresh-install VM gate. It extends M417's gateway/history result with current
ordinary binaries. M424/M425 were integrated after this VM was frozen and are
excluded from its runtime claim. Their member-aware observer still needs actual
member qualification. Durable member lifecycle, first-boot identity capture,
schema44 migration/refusal and public runtime orchestration remain. Schema43,
production ships, tmux ships, ~/temp-ur, host Docker and installed plugins were
not changed. No key or wallet custody guarantee follows from these checks.
