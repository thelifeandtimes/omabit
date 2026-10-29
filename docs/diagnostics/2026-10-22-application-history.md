# Completed application history

The unreleased application connection flow now archives fully closed and
reconciled attempts, freeing active journal slots while retaining permanent
private replay receipts. The CLI exposes `app-archive` and `app-connection`;
the actual approval pane offers **Archive completed** only after cleanup.
Connected, uncertain and unreconciled attempts remain active.

Final frozen runtime: `/tmp/omabit-auth-run-20261021-09`. The
[evidence manifest](2026-10-22-application-history.json) records exact source and
artifact hashes, namespace/sandbox arguments, nonsecret lifecycle events, archived
connection views, logs and the previous collector's refusal. All source/helper
hashes matched when recorded. No release or installation was made.

## Durability and compatibility

The issuer syncs the receipt before removing its active row. It advances only
`applications.json` to version 2 at removal, preventing older issuers from ignoring
the permanent receipt. Same-revision archival is idempotent after a lost result;
stale revisions fail. A real subprocess exits between receipt publication and
active removal; reopening preserves the receipt and safely completes the archive.

Both receivers compact reconciled rows into private receipts before active removal.
Pending/accepted rows remain, and a full legacy journal can be compacted on the next
new approval. Frozen prior helper code proves remaining cleanup still works with
the version-1 active shape. Old helpers do not understand receipts themselves;
the upgraded issuer remains responsible for refusing archived approval IDs.

The previous experimental collector executable from frozen run `08` was run
against only the new lab's version-2 journal. It refused the read with exit 1 and
`application_unavailable`; journal and receipt hashes stayed unchanged. There is
no downgrade-by-reset procedure.

## Evidence

- Go race checks passed for collection, application issuer and collector commands;
  vet passed for the same packages. Issuer fixtures cover 96 completed cycles,
  replay/stale refusal, interruption/retry, unfinished phases and unsafe receipts.
- Theme: 43 receiver/recovery/history/install tests passed. DMs: 35 corresponding
  receiver/recovery/history/account-store tests passed. Each history suite covers
  96 cycles, legacy capacity, old-helper cleanup, crash boundaries and receipt sync.
- The actual offscreen approval pane passed six explicit mutations, ending in
  archival with no active row; it retained private stdin, cleared consent and
  never retried mutations automatically. Evidence is in
  `/tmp/omabit-app-ui-20261021-05`.
- Fresh-fake independent sessions, real helper handoff and durable lost-ack recovery
  passed. Public CLI and real panel profiles each connected both real helpers,
  verified direct reads/SSE, revoked/reconciled independently, then archived both
  attempts. Repeated archive succeeded; stale archive and approval replay failed.
  Both active receiver journals became empty with one reconciled receipt each.
- Public-flow checks took 13.99 seconds after boot. The resource-capped namespace
  run returned test exit 0, clean shutdown and no log truncation.

## Limits and next work

Receipts and per-attempt locks remain permanent; total disk use grows with history.
Retained receiver generations have not been deleted. Future pruning must coordinate
reference checks with approval publication so it cannot remove recovery code.
The 96-cycle tests use fixtures, not 96 real remote logins.

The same isolated lab boundary as prior checkpoints was used: private namespaces,
loopback, HOME/XDG roots and Secret Service, copied artifacts, no host desktop,
keyring, Docker socket or home mount. Theme's desktop adapter and DMs' native
launcher stayed disabled. Managed `~lismyl-lismyl`, tmux ships, `~/temp-ur`, current
plugins and production services were not inspected or changed.

Full desktop/profile behavior, writes/reconnect and hosted compatibility remain.
Core import/layout recovery, CLI skills, fake networks, plugin-owned installation
and private dev chains retain their roadmap gates. This checkpoint completes the
active-journal capacity fix, not the full platform goal.
