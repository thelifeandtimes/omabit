# Reviewed source-restoration prototype

This records the earlier fixture-only checkpoint. The later
[runner/CLI integration](2026-10-22-import-restore-integration.md) supersedes its
unwired API/schema status; its original test evidence remains unchanged.

The runtime prototype can restore a held source to its original name after a
partial cross-filesystem import, while retaining the destination under
`archive/import-<runtime-id>`. Both moves use no-replacement renames. The original
failed registration stays fenced throughout this prototype. Nothing is started,
deleted, re-imported or installed. These helpers are not yet wired into a public
command or the runner's persistent state; `recover-import` still supports later
enrollment recovery only.

## Review and move contract

The read-only review requires an unstarted import in `copying` or `verifying`, the
same recorded held source object, a valid source state/epoch layout, the recorded
exclusive destination parent, no Vere locks, and absent original/retained names.
It rejects unrelated staging or runtime evidence, links, overlapping trees and
replaced parents. A digest binds the current paths, parent identities and complete
object inventories. Deletion's subset-tolerant manifest check is insufficient:
restoration checks exact inventories, including missing entries and directory
metadata. The review never claims that an old import journal recorded every
current destination child; approval concerns the current retained contents.

The prototype then performs these phases, requiring a durable callback before
each transition:

1. Record `retaining-destination`, then rename the whole exclusive destination
   parent to the reviewed archive path.
2. Check both trees and sync both rename parents, including when a previous attempt
   already renamed but failed to sync. Record `restoring-source`, then rename the
   held source to its original name.
3. Check both trees and sync the source parents. Record `complete` while keeping
   the now-empty holding directory and the archived destination.

Resume accepts exactly the recorded object at one of the two names appropriate
to its durable phase. An occupied name, changed tree, replaced parent or native
lock stops further work. Completed replay performs no rename. Only a moved root's
ctime may differ; descendant identities, names and metadata stay exact. Full file
hash comparisons in fixtures confirm that both trees retain their bytes.

Directory descriptors bind rename/sync to the reviewed parents, and
`RENAME_NOREPLACE` prevents overwriting an occupied target. These are change
detection measures; Linux provides no rename conditional on inode, and this does
not isolate hostile concurrent same-owner writers. Runner serialization and the
ordinary stopped-pier requirement remain necessary.

## Evidence and limits

[The fixture record](2026-10-22-import-restore-prototype.json) contains source
hashes, test outcomes and the full race-suite log hash. Fixtures cover stale
reviews, changed/removed files, replaced objects, locks, overlapping paths,
interruption before/after both renames and journal publications, cancellation,
failed directory syncs and completed replay. They use fresh temporary files and
inert engine dependencies. The [real interrupted-copy run](2026-10-22-import-interruption.md)
establishes the need for this recovery but predates these helpers.

Current limits are deliberate and visible: only partial copies with an existing
destination child are accepted; device renumbering after a journal is created is
refused; the review binds metadata rather than a filesystem snapshot; no public
API, schema change, registration retirement, real restoration or power-cut
qualification has been completed.

## Next integration checkpoint

Expose a read-only preview and an explicit restore operation with operator
authority, pinned runner identity, current instance revision, the reviewed digest
and actor-scoped request replay. Keep full object inventories private. Persist
the journal in a new runner schema with tested upgrade/downgrade refusal before
exposing any mutation. Preserve the failed-import fence on every interruption.

After verified restoration, retire only the failed registration in a transaction,
preserving operation receipts and the tombstone revision. A fresh import must get
a new revision; old import/restore requests must replay their original receipts.
Do not delete the archived partial tree or silently retry an import.

Then qualify preview, restore, replay and a fresh successful import in the private
VM using real collector/runner binaries. Follow with empty-destination, full-disk,
process-crash, power-loss and UI-to-runtime cases. The current guest initializer
normalizes ownership only for fresh images; correct that first before rebooting a
populated guest for power-loss tests. The platform goal remains active.
