# Source-entry private Docker recovery — 2026-10-25

The reusable repository preparation, common VM launcher and new stopped-disk
auditor now pass end to end. Fresh VM09 completed in1946.4s, and its separate
read-only audit completed in11.19s. Both exact scopes are inactive.
The sealed12,357-file input inventory matches M346, while the project ID and both
public authentication keys are fresh. The original unkeyed foundation/genesis is
preserved; no prior keyed-chain state or pier was used as a bootstrap input.

The official image passes six acknowledged messages and one intentional Gall nack,
four unreplayed persistent writes, seven scries, six subscription snapshots and two
explicit channel deletions. Same-pier receiver replacement retains original Eyre
sessions and the saved cursor, and a fresh subscription reconciles current state.
Cross-ship sessions refuse. Two initial acknowledged Lens stops and the reopened
receiver's TERM stop all exit0 without OOM; gateway and VM also stop cleanly.

The repository auditor reads only the stopped disk in its own no-NIC VM, with
read-only block bindings and ext4 noload. It independently checks the retained
message/state/lifecycle receipts and leaves both base and overlay hashes unchanged.
Its fixed metadata output omits raw values and diagnostic bodies; receipt text is
hashed. Only curated host outcomes, this public audit, source, input metadata and
hashes are exported. No console, database, key, pier or payload archive is copied.

This closes M347/M348/M350's pending source-entry qualification gates. It does not
implement private-chain runner registration/lifecycle, host endpoint publication,
simultaneous duplicate-project isolation or a supported release. M349's offline
planner remains review-only. Continue with those runner integration gates while
preserving fake-project formats and the user's production exclusions.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-25-private-docker-source-runtime`.
Private run: `/tmp/omabit-import-guest-run-private-chain-20261025-09`; audit: `/tmp/omabit-private-chain-inspection-20261025-19`.
