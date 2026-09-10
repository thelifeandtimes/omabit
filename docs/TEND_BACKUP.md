# Tend backup contract

Tend exports a portable JSON envelope whose format name is
`tend-backup-1`. Export and validation never include or read an Eyre cookie or
Urbit `+code`.

## Export and validate

```sh
omabit tend export ~/Documents/tend-backup.json
omabit tend validate-backup ~/Documents/tend-backup.json
omabit --json tend validate-backup ~/Documents/tend-backup.json
```

Export writes atomically with mode 0600, refuses a symbolic-link target, and
requires `--force` before replacing an existing regular file. Validation is
offline: it does not require a saved Eyre session and does not contact a ship.
It opens only a regular file without following a final symbolic link and caps
input at 32 MiB.

The validator checks the envelope version and timestamp, list/section/reminder
shape, unique IDs within each collection, title and tag bounds, safe URL
schemes, schedule and recurrence ranges, section and parent references, parent
cycles, and snooze references. A successful result reports only counts and
backup metadata; it does not change Gall state.

## Included data

The current exporter captures the authenticated ship's locally hosted lists,
sections, reminder metadata, schedules, assignments, preferences, and active
snoozes. It also records the source ship and export time. Authentication
material is never part of the envelope.

Shared-list replicas are not exportable until Tend's peer engine is enabled.
When replicas become visible in the snapshot, the backup protocol must label
them as non-authoritative copies and preserve their canonical owner/list
reference. A restore must never turn a participant's replica into an owned
list silently.

Treat the file as private: reminder notes, URLs, schedules, and assignments may
be sensitive, and mode 0600 does not encrypt the file at rest.

## Restore safety contract

Restore is intentionally not enabled in this checkpoint. Its first supported
form will follow these rules:

1. Run exactly the same bounded validation before sending any noun to Gall.
2. Accept a backup only into an empty `%tend` state; never merge or replace a
   non-empty ship implicitly.
3. Show source ship, object counts, and any skipped non-authoritative replica
   before requiring an explicit confirmation.
4. Commit all imported owned lists, preferences, snoozes, and next-ID state in
   one Gall transition or commit nothing.
5. Allocate fresh operation receipts and timer generation, revalidate snoozes,
   and arm the earliest outstanding alert after commit.
6. Preserve canonical content but do not restore Eyre sessions, active
   subscriptions, in-flight peer mutations, or stale operation receipts.

Until that state transition and its migration fixtures exist, do not post a
backup through an ad hoc poke or edit a pier by hand.
