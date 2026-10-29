# Guided restoration through the real runner

Run11 passed using appliance10: actual offscreen restoration QML drove the real
collector and runner inside the fresh disposable VM. The test then verified both
filesystem objects, terminal replay, fresh re-import and explicit
Start/identity/exclusive-parent-mount/Stop. Clean guest shutdown completed in
**865.45 seconds**. No live test job remains.

[Machine-readable evidence](2026-10-22-import-restore-ui-runtime.json) pins the
source, binaries, signed package manifest, images, commands, receipts and database.
This advances the [inert UI checkpoint](2026-10-22-import-restore-ui.md) and the
[preceding CLI/runtime trial](2026-10-22-import-restore-runtime.md).

## What passed

- The fresh fake `~zod` booted and stopped cleanly. The copy was interrupted after
  8,388,608 reported bytes. The complete 7-file / 1,016,823,820-byte source and
  34,078,720-byte partial target survived the runner restart.
- The shipped `StoragePane`, restoration controller/pane and verbatim Service
  callbacks used the collector's real inventory, including its authenticated
  actor. The UI issued exactly one preview and one explicitly approved restore.
  An unchecked button signal was refused, and a duplicate approved signal did
  not issue another command. All four subprocesses (including two inventory
  reads) exited normally with code zero; the Qt log is empty.
- Both reviewed moves, counts and the digest matched an independent CLI preview.
  The terminal receipt remained visible after the failed instance disappeared.
  Screenshots were inspected: paths and approval are readable, and successful
  completion remains outside the retired instance card. The review screenshot's
  warning is the expected result of the deliberately unchecked button signal.
- Independent filesystem checks proved the original source inode/content returned
  and the same partial parent inode/content was archived. Restoration started no
  runtime. Terminal requests replayed unchanged before and after registration
  revision2 replaced the retired revision1.
- Re-import completed stopped. Explicit Start verified `~zod` and the exclusive
  parent mount `/srv/urbit/piers/fakeships/zod` → `/urbit`, with the named child
  `/urbit/zod`. Explicit Stop succeeded; the archive remained unchanged.
- Offline SQLite `quick_check` passed, schema22 remains, and the final database
  contains one stopped revision2 instance, the revision1 tombstone and exactly
  five operations: interrupted import, successful restoration, re-import, Start
  and Stop. An initial offline verifier assumed the schema was in `meta`; it
  stopped before cleanup. The corrected verifier reads `PRAGMA user_version` and
  the stored instance's `public` object, and passed.

## Evidence and isolation

- Appliance: `/tmp/omabit-import-appliance-20261022-10`.
- Completed run: `/tmp/omabit-import-guest-run-20261022-11`.
- Offline extraction: `/tmp/omabit-import-inspection-20261022-11`, with **74 files /
  369,619 bytes** of result evidence and a separate read-only database extraction.
- UI receipts/screenshots: `results/ui-restore/` in that extraction.
- Signed Qt/PySide6 guest cache: `/tmp/omabit-import-guest-packages-20261022-03`
  (210 packages); no host packages were installed.

The VM had no NIC, host shares, production engine, keyring or desktop sockets.
Its admitted scope capped memory at 12 GiB, swap at 0, tasks at 128 and CPU at one
core. Original images/disks and extracted receipts are retained. Only verified,
regenerable packaging trees and the redundant inspection conversion were removed.
Trial10's prior original disks were hash-verified and relocated to the task's
persistent evidence directory, retaining compatibility links; its earlier report
is unchanged. Protected ships, tmux ships, `~/temp-ur`, installed plugins and
production services remained excluded.

## Limits and next checkpoint

The style values and subprocess adapter are private test replacements for native
Quickshell integration. This qualifies actual QML-to-collector/runner behavior,
not native desktop lifecycle/hooks. Empty-target/full-disk cases, process crash,
power loss, device renumbering and release qualification remain open.

The next bounded feature checkpoint is two fresh fake galaxies communicating
inside one private project network namespace, followed by restart and duplicate
identity isolation in a second namespace. Follow the
[networking research and acceptance ladder](../project-fake-networks.md); keep
remaining import failure gates explicit rather than treating this pass as v1.0.
