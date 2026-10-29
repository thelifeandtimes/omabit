# Public interrupted recovery: runtime evidence passes, final fixture fails

VM41 shuts down cleanly after2459.22s; its overall qualification remains failed.
The final reopened-store test hard-codes actor `qualification`, while the actual
public CLI correctly records `local-uid:1000`. Its attempted replay therefore
misses the actor-owned receipt and is refused against the changed history.

An independent audit of the inactive original disk verifies six actual CLI
requests and six lookup receipts, three lost replies/store reopens and exact
effect counts: two each of drain, retirement, Create and Start. The application
retains16 writes, eight subscriptions/24 snapshots, four recovery cycles and
two original sessions; no uncertain write is replayed. Journal/checkpoint
mirrors, final pier states and retained SQLite history match. These are specific
passing subgates, not a rewrite of the failed overall result.

A copy of this actual database reproduces the old-actor failure. The corrected
fixture uses the saved receipt actor and lookup-only mode, retrieves the exact
succeeded receipt, passes the final reopen helper and preserves all36 operations.
That targeted Go check passes0.587s; only the test helper is changed in source.
The original success verifier remains unchanged; the separate failed-trial audit
requires the exact failure and explicitly marks overall qualification false.

Database SHA256: `6fa6f00ffd0bcdddabd0c611579299ed2f7c3c8efd92c56a4b9bcc6e6cfd5ed8`.
Receipt SHA256: `c72a4a01f65f73bf73fbd4a5a2fc8678a405f771ede445c5641ec31f897168fa`.

The first extraction refuses the unchanged14GiB reserve. Verified duplicate
stopped native piers and the regenerable task-specific Go cache free enough space;
retry passes. The original disk and both extraction attempts remain. Only the
verified regenerable raw conversion is removed. Actual OS service restart is the
next gate and remains unqualified; its draft is separate and not yet applied.
Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-public-recovery41`.
