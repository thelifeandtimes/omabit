# Permanent mount roots and whole-fleet migration — 0.20

## Contract

Every managed pier is now `.../<ship-directory>/pier`. Docker binds only the
exclusive parent at `/urbit`; official Vere creates `/urbit/pier` on first boot
and reuses it for initialized starts and offline utilities. First boot finishes
without stopping, moving or replacing its runtime. The mount parent and pier
directory identities are both checked. Reservations protect the whole parent;
discovery/imports require the fixed child and reject shared/unsafe mount contents.

The runner creates and accepts **schema 11 only**. No old-layout execution path is
retained. Removed code includes `externalPier`, `promoteManaged`, `stopForPromotion`,
the staging-lock recovery helper, `Storage.Ready`, `storage_pending`, the variable
managed pier-basename helper, and incremental feature-specific schema upgrades.
The old storage migration scripts were removed; the completed one-shot schema-10
migration tool was moved out of the application tree to `/tmp/opencode`.
Historical journals and archived backups remain evidence, not runnable compatibility.

## Before/after fleet

All locations below are relative to `/var/lib/urbit-runner/piers`.

| Instance | Current actual pier | Restored state |
| --- | --- | --- |
| alpha | `fakeships/zod/pier` | running, healthy, revision 2/2 |
| bus-lab | `fakeships/bus/pier` | running, healthy, revision 1/1 |
| comet-check | `livenet/comets/comet-check/pier` | running, healthy, revision 1/1 |
| dibpen-fogweg-dozzod-sarlev | `livenet/moons/dibpen-fogweg-dozzod-sarlev/pier` | running, healthy, revision 1/1 |
| import-check | `fakeships/import-check/zod/pier` | stopped, revision 1/1 |
| panel-check | `fakeships/panel-check/zod/pier` | stopped, revision 1/1 |
| rus | reserved `fakeships/rus/zod/pier` | uninitialized, revision 1/0 |
| ssh-check | reserved `fakeships/ssh-check/zod/pier` | uninitialized, revision 1/0 |

The four running ships stopped cleanly before migration. Both uninitialized
registrations remained uninitialized. Stopped epoch inventories were recorded for
all six initialized piers. Migration ran from 19:39:42 to 19:39:58 UTC, preserving
their complete content digests and pier/`.urb` inodes/birth identities while adding
the parent mount directory. Before/after epoch entries, inventory digests and
private pier IDs match. Port configurations, configuration/applied revisions,
labels, identities, SSH enrollment and disabled policy records are preserved.

The running ships were then explicitly resumed from their existing piers. Current
containers after that intentional replacement:

- alpha: `1dc7a5b9cd384c32e88b084b51f0ec4f38908c4fa71960617cd2f6fb2a49fafa`
- bus-lab: `c36ca5830caab30f241848149544d4b1f4cdca3282174259e212cc1eb0e69890`
- comet-check: `12172f52417a1d56a07d0fb81780d5be2117687076be07ea26af949fdc5714b7`
- moon: `194ae9a18544a28ff6970c515c10a48d683c8121159e1db45c2c4ae04316a16e`

Discovery reports exactly six initialized enrolled piers, no unexpected moves or
problems, and excludes the backup/acceptance archives. No recovery flags or incidents
remain. Restoration receipt: `/tmp/opencode/layout20-fleet.json` (`complete: true`).

## Backups and receipts

- Full stopped pier copies: `/var/lib/urbit-runner/piers/archive/layout20-before-20261002T193942Z/`.
- SQLite online backup: `/var/lib/urbit-runner/.local/share/urbit-runner/managed/before-layout20.sqlite`.
- Durable migration receipt: `/var/lib/urbit-runner/.local/share/urbit-runner/managed/layout20-migration.json`.
- Direct-boot acceptance receipt: `/var/lib/urbit-runner/.local/share/urbit-runner/managed/layout20-validation.json`.
- Stopped acceptance pier: `/var/lib/urbit-runner/piers/archive/layout20-acceptance/pier`.
- Previous plugin snapshot: `/tmp/opencode/omabit-panel-before-permanent-piers.tar`.

Backups were content-checked before any live pier rename. Empty obsolete private
`data/` directories were removed; a nonempty unexpected directory would abort the
migration for review. No source pier or archive was deleted.

## Verification

The offline migration first passed an isolated synthetic test, including data
preservation, empty uninitialized registrations and refusal to replay a completed
migration. Before moving real piers, `scripts/validate-permanent-pier.go` booted a
fresh fake through the actual managed driver and verified that its initial
container remained running at the permanent path. An explicit stop/resume preserved
both filesystem identities, then removed the acceptance containers/bridge and
archived the stopped test pier. No temporary ship was added to the eight registrations.

Full Go tests/vet, targeted race tests for pierstore/managed/runtime, native builds,
the unsupported ARM64 stub build, model tests, installer tests and mapped QML tests
passed. Runtime fixtures cover direct first boot for fake/comet/owned-key paths,
partial-bootstrap refusal, foreign-root refusal, exclusive mounts, epoch operations,
imports, moves and initialized resume. The source audit finds no old staging or
promotion execution paths. `git diff --check` passes.

Panel **0.20.0** is installed and loaded after cold replacement. The built,
installed host and plugin-bundled runner all have SHA256:
`d0b7d3489dec4d8151b9506fd8dd6091c1c591cd9ba74240c664d1c394f074f1`.
