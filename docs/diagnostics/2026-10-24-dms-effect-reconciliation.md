# DMs authenticated desired-state reconciliation

A fresh authenticated chat /dm result can confirm local invitation acceptance.
An activity v4 summary explicitly containing unread=null can confirm the main
stream is read even while child-thread counts remain (the original read action is
shallow). These semantics were checked against the local Tlon Gall/JSON source;
exact hashes/commit are retained. A missing entry, absent field, coerced/bogus count
or another ship/origin is not evidence. Only intent IDs captured before the read
began can resolve, and the original connection generation must remain current.

Confirmed desired state uses a distinct state-confirmed phase, never fabricated
acknowledgment. A later exact receipt can supersede that phase. Existing version1
journals remain readable without write-on-read; a mutation writes version2 while
preserving IDs. Future versions are refused intact. Ledger errors leave read-only
inbox/activity display usable and expose unreadable history rather than hiding it.

All112 Python tests pass, including wrong binding, missing/malformed evidence,
late new intent, generation changes, migration and no-dispatch checks. The actual
private helper/keyring/synthetic-Eyre fixture first retains all three unknown
intents through stable/dev/stable/edge/stable. Only then does the server expose
positive conversation/activity state. Refresh resolves the two effects with no
new poke; the send remains unknown. Another stable helper restart preserves both
results. Three total application pokes, six explicit drains, twelve deleted
channels and all helpers gone are independently verified. The exact passing
fixture is applied with optional --reconcile-effects (requires --effects).

This supersedes M257's pending effect-reconciliation limitation. It does not prove
real Gall runtime behavior, all external-launcher effects, exclusive appearance or
complete collection activation. Existing installations and production ships remain
untouched; collectionReady=false remains deliberate.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-dms-effect-reconciliation`.
