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

The current exporter captures only the authenticated ship's owner-authoritative
lists, sections, reminder metadata, schedules, assignments, preferences, and
active snoozes. It consults authenticated access metadata to exclude every
visible shared-list replica. It also records the source ship and export time.
Authentication material is never part of the envelope, and a participant's
replica can never silently become an owned list through export/restore.

Treat the file as private: reminder notes, URLs, schedules, and assignments may
be sensitive, and mode 0600 does not encrypt the file at rest.

## Restore safety contract

Restore is enabled as an explicit empty-state-only operation:

```sh
omabit tend restore ~/Documents/tend-backup.json --yes
```

It follows these rules:

1. Run exactly the same bounded validation before sending any noun to Gall.
2. Accept a backup only into an empty `%tend` state; never merge or replace a
   non-empty ship implicitly.
3. Show source ship and object counts before requiring `--yes`. Replicas were
   already excluded from the export and are never sent to Gall.
4. Commit all imported owned lists, preferences, snoozes, and next-ID state in
   one Gall transition or commit nothing.
5. Allocate a fresh timer generation, discard expired snoozes, recompute the
   next global ID, and arm the earliest outstanding alert after commit.
6. Preserve canonical content but do not restore Eyre sessions, active
   subscriptions, in-flight peer mutations, or stale operation receipts.

The CLI verifies the durable `/receipt/<operation-id>` result after Eyre accepts
the poke. A second restore is rejected because the target is no longer empty.
Do not post a backup through an ad hoc poke or edit a pier by hand.

Sharing state is deliberately not portable in `tend-backup-1`: hosted ACLs,
pending invitations, replicas, subscriptions, liveness sessions, in-flight
mutations, and old receipts are reset. Assignee values remain reminder metadata,
so an imported reminder may name a ship that is not yet a member; re-establish
sharing deliberately before relying on assignment workflows.
