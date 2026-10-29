# Official keyed Docker application recovery — 2026-10-25

Fresh no-NIC VM08 passes in1918.39s with the pinned official image and
its own rootless Docker. The private chain gateway generates two new keypairs,
validates correct/wrong key and revision responses, and serves only local read-only
RPC. Both ships observe the peer's key at life1 before application transport.

Six acknowledged messages and one deliberate Gall rejection pass. Four unique
persistent application writes, seven scries and six subscription snapshots across
two channels survive receiver replacement on the same pier. The original Eyre
session and saved cursor are reused; a fresh subscription reconciles full state.
No write is replayed, crossed cookies refuse, and both channels close explicitly.

The two initial keyed containers stop through acknowledged Lens exits, and the
reopened receiver stops through TERM; all three are observed exit0 without OOM.
The gateway also exits0 after its owned shutdown, retaining head0x50 on chain1337.
The VM shuts down cleanly. A separate read-only no-NIC audit verifies counts,
state/cursor coherence, exact receipts and unchanged base/overlay hashes. Both
scopes are inactive.

The peer failure auditor now redacts raw error text because a mismatch exception
can embed application values. Five top-level tests and10 negative subcases pass;
only fixed reasons, public progress counts and hashes are exported. No prior raw
state was exported through this new failure path. Earlier readers remain retained.

The exact source copies integrated in M345 match the executing appliance. Evidence
contains code, public metadata and hashes, not raw logs, chain databases, keys,
sessions or piers. Runtime state stays private at `/tmp/omabit-import-guest-run-private-chain-20261025-08/lab/root.qcow2`.

This supersedes M338–M345's pending keyed Docker peer/recovery checkpoint only.
It is not a public runner devnet implementation or a release. Reusable preparation/
launch tooling, explicit private-chain project classification/lifecycle, project
isolation and broader platform gates remain. No installed plugin or production
ship was changed.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-25-keyed-docker-recovery`.
