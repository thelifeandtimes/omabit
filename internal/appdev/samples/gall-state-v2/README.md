# Gall state sample v2

This separate version adds the bounded JSON text mark `%omabit-entry` for direct
Eyre read/poke/watch clients. The original v1 sample and its noun wire remain
unchanged. State stays tagged version0; the two APIs are distinct wire contracts.

Read the complete frozen definition and exact eight-file source through
`omabit call app.sample.v1 --input-json '{"name":"gall-state-v2"}'` or the matching
MCP tool. Save only source files to a fresh staging directory; retain notices
with redistributed copies. No full repository or Unix mount is needed. The
[separate native report](native-qualification.json) binds the exact definition,
source and runtime inputs to a passing single-peer loop. The definition retains
its authoring-time pending status; the CLI/MCP result includes the later evidence
separately. Managed lifecycle/installed upgrades and multi-peer behavior remain
unqualified by this report.

Only fresh disposable ~zod/~nec are admitted by the app; each poke carries JSON
text, at most16 distinct entries are retained, and duplicates/non-text/other
senders refuse without changing state. Authenticated scry and /state watch retain
the v1 revision/entries shape. These are app permits, separate from Omabit's
creation/Commit/Publish gates and Eyre session authority. Unknown saved-state tags
refuse; resetting a pier is never migration or rollback.

The source includes JSON and MIME marks. The minimal desk scaffold does not
provide those conversions: a live agent alone cannot prove its JSON scry works.

Use selected-runner preflight and the normal staged commit/local activation
workflow. Verify native compilation, active code, source hashes, scry, poke Gall
acknowledgement, watch facts, reload, same-pier restart and session/channel cleanup
separately. A source commit or HTTP acceptance alone is not application readiness.
