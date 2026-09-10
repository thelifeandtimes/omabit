# Tend implementation status

Last verified: 2026-09-10

## Working now

- `%tend` installs and hot-reloads on a live fake ship.
- Existing `%0` through `%5` state migrates to the current `%6`
  schema without losing lists, reminders, schedules, preferences, or snoozes.
- Authenticated Eyre login, identity scry, SSE subscribe/ack, reconnect, and
  poke acknowledgement work through the dependency-free desktop bridge.
- List CRUD, section CRUD, rich reminder updates, hierarchy/order moves,
  completion, cascading deletion, optimistic revision checks, operation
  deduplication, and JSON snapshot/update conversion work end to end.
- Unsafe URL schemes, invalid titles, stale revisions, missing entities, and
  parent cycles are rejected without changing canonical list state.
- The Omarchy service stays read-only in `Checking` and `Offline`, becoming
  writable only after its snapshot arrives.
- The overlay can create lists, sections, and reminders; select a reminder;
  edit its title, notes, URL, priority, flag, and tags; toggle completion; and
  delete the reminder through revision-checked service calls.
- List settings edit title, color, and symbol with a guarded delete action.
  Section settings edit title/rank or safely delete a section, and reminder
  organization controls set parent, section, and manual rank. The corresponding
  `%tend` actions compile and round-trip through live Eyre.
- Today (including overdue), Scheduled, All, Flagged, and Completed views work
  across lists. The overlay can search reminder text, notes, URLs, tags, and
  list names, then sort by manual rank, due date, creation date, priority, or
  title in either direction.
- Due instants, all-day policy, IANA-zone annotations, early offsets, and
  structured recurrence round-trip through Eyre. Recurring completion advances
  hourly/daily/weekly/monthly/yearly rules with end/count enforcement.
- The agent persists delivered alert offsets, keeps one generation-tagged
  earliest Behn timer, and emits alert events that the desktop maps to native
  notifications. Live tests verified an immediate overdue wake and recurrence
  advancement.
- Revisioned default-list, pinned-list, pinned-view, and snooze-preset
  preferences now live on the user's ship. Smart-view quick entry targets the
  default list, pinned lists sort first, and the overlay can update pins and the
  default.
- Personal snoozes are stored outside canonical reminder data, share the single
  Behn wakeup scheduler, survive reloads, and are pruned when their reminder is
  no longer actionable. The `%2` to `%3` migration and both new actions were
  verified through live Eyre.
- The schedule editor now exposes recurrence weekdays, month dates, ordinal
  weekday, end date, and occurrence count. Reminder organization includes
  direct up/down rank controls in addition to explicit parent/section/rank.
- Native notifications offer Complete, Snooze, and Open actions. They are
  serialized through a desktop queue; Open deep-selects the reminder and the
  mutation actions remain disabled unless the home connection is Online.
- Quick entry recognizes whitespace-delimited `#tag` tokens. The overlay lists
  all known tags and can rename, merge, or delete one across hosted reminders;
  server-side bounds and set semantics were verified through live Eyre.
- `%4` freezes the complete `%3` schema, adds a canonical optional Urbit
  assignee to reminders, and durably allocates hosted-share, replica,
  invitation, and in-flight stores. A live `%3` ship migrated successfully and
  an assignment round-tripped through Eyre.
- `%5` freezes `%4` before collaboration receipts and pending-invitation
  payloads change shape. A live `%4` ship migrated successfully without losing
  its reminders, preferences, schedules, assignments, or snoozes.
- `%6` adds revisioned badge and all-day reminder policy without changing
  canonical reminder/list nouns. A live `%5` ship migrated, accepted a policy
  update, and preserved it across Gall suspend/revive.
- The authenticated `/all` stream now emits one access record per visible list,
  including canonical host, owner flag, Online/Checking/Offline state, members,
  and pending invitations. The Omarchy UI visibly marks owner availability and
  disables every shared-list mutation while the host is Checking or Offline.
- Eyre connection files are written atomically with mode 0600, symbolic-link
  credential paths are refused, non-loopback HTTP and URL paths are rejected,
  expired authentication stops the reconnect loop, and users can disconnect or
  switch ships without retaining the reusable `+code`.
- `omabit tend` provides connect/disconnect/status, list/today, add,
  complete/uncomplete, snooze, and overlay-open commands with optional JSON
  output. Its read-only status and list flows were exercised against the live
  desk.
- Assigned to Me is available as a smart view and search includes assignee
  ships. Quick capture supports `#tag` entry and the bar badge can count Today,
  all incomplete, or assigned reminders—or be disabled. Users can choose the
  default time for new all-day reminders and whether stale all-day items stay
  in Today; timed overdue reminders remain visible.
- `omabit tend export` writes a complete `tend-backup-1` JSON snapshot without
  authentication material. Export is atomic, mode 0600, refuses symbolic-link
  targets, and does not replace an existing file without explicit `--force`.
- Deterministic release packaging includes the desk, validated Omarchy plugin,
  CLI, installer, documentation, and per-file checksums. The installer refuses
  symlink targets and preserves forced replacements as timestamped backups.
- Plugin validation, JavaScript reducer tests, and Python Eyre transport tests
  pass.

## Next

1. Owner-hosted invitation delivery, ACL subscriptions, replicas, mutation
   routing, assignment validation, and activity across multiple ships.
2. Batch operations and additional quick-capture refinements.
3. Restore/import design, accessibility audit, fault testing, and final release
   hardening.

The source of truth for scope and exit criteria remains
[`PRODUCT_ARCHITECTURE_PLAN.md`](../PRODUCT_ARCHITECTURE_PLAN.md).
