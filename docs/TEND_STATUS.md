# Tend implementation status

Last verified: 2026-09-11

## Working now

- `%tend` installs and hot-reloads on a live fake ship.
- Existing `%0` through `%9` state migrates to the current `%10`
  schema without losing lists, reminders, schedules, preferences, or snoozes.
- Authenticated Eyre login, identity scry, SSE subscribe/ack, reconnect, and
  poke acknowledgement work through the dependency-free desktop bridge.
- Every complete state snapshot advertises desktop protocol 1. Fresh login
  validates it before writing credentials; status, direct reads, invitation
  mutations, and streamed replacement fail closed on missing or different
  versions. The supported pre-release deployment is now explicitly lockstep
  across each sharing group.
- The desktop identity path is rank-neutral and regression-tested with planet,
  moon, and comet-shaped ship identities; Gall protocol identities remain
  unrestricted `@p` values with no rank authorization branch.
- List CRUD, section CRUD, rich reminder updates, hierarchy/order moves,
  completion, cascading deletion, optimistic revision checks, operation
  deduplication, and JSON snapshot/update conversion work end to end.
- List-local multi-selection supports atomic batch complete/uncomplete,
  move-to-section, and guarded cascading deletion. Batch recurrence completion
  follows the same advancement rules as a single completion, and selections
  are validated and capped server-side.
- Unsafe URL schemes, invalid titles, stale revisions, missing entities, and
  parent cycles are rejected without changing canonical list state.
- Gall now enforces explicit ceilings on note/URL bytes, list/section/reminder
  counts, recurrence and early-offset sets, preference arrays, and batch sizes.
  Desk revision 37 compiled on Vere 4.6; an oversized-note action left list
  revision/content unchanged, while a bounded update and revert both committed.
- The Omarchy service stays read-only in `Checking` and `Offline`, becoming
  writable only after its snapshot arrives.
- The overlay can create lists, sections, and reminders; select a reminder;
  edit its title, notes, URL, priority, flag, and tags; toggle completion; and
  delete the reminder through revision-checked service calls. Saved HTTP(S)
  and mailto links can be opened explicitly through Qt's default application
  handler; the client mirrors Gall's scheme allowlist before enabling it.
- List settings edit title, color, and symbol with a guarded delete action.
  Section settings edit titles, atomically move a section up/down, or safely
  delete it; owner-side placement normalizes tightly packed section ranks. A
  live three-section fixture verified ordering, stale rejection, and cleanup
  on desk revision 41. Reminder organization controls set parent, section, and
  manual rank. The corresponding `%tend` actions compile and round-trip through
  live Eyre.
- Subtasks render in parent/child order with depth cues, can be collapsed or
  expanded by pointer or arrow key, and expose explicit indent/outdent controls
  plus `Ctrl+]` / `Ctrl+[` shortcuts. Manual rows can be reordered with a
  pointer drag handle or keyboard up/down controls. Both paths submit an
  atomic before/after operation; Gall normalizes the sibling group to sparse
  ranks and rejects stale placement without mutation. A live tightly
  packed-rank fixture verified the canonical order.
- Moving a parent between sections updates every descendant's section in the
  same host revision. A live parent/child fixture verified the cascade and
  revision bump, then was removed cleanly.
- Today (including overdue), Scheduled, All, Flagged, and Completed views work
  across lists. The overlay can search reminder text, notes, URLs, tags, and
  list names, then sort by manual rank, due date, creation date, priority, or
  title in either direction.
- Due instants, all-day policy, IANA-zone annotations, early offsets, and
  structured recurrence round-trip through Eyre. The dependency-free bridge
  converts local wall-clock input with system `zoneinfo`, rejects daylight
  saving gaps, selects the earlier fold, and enriches stream data with local
  display values while Gall stores canonical `@da` instants. A live
  Los Angeles wall-time action was verified against its UTC value. Recurring
  completion advances hourly/daily/weekly/monthly/yearly rules with end/count
  enforcement. All-day rows and editors show the calendar date without the
  policy alert time, and Today compares that date rather than the UTC instant.
- The agent persists delivered alert offsets, keeps one generation-tagged
  earliest Behn timer, and emits alert events that the desktop maps to native
  notifications. Live tests verified an immediate overdue wake and recurrence
  advancement.
- Revisioned default-list, pinned-list, pinned-view, and snooze-preset
  preferences now live on the user's ship. Smart-view quick entry targets the
  default list, pinned lists and views sort first in the user's chosen order,
  and the overlay can update, reorder, or remove pins and change the default.
- Personal snoozes are stored outside canonical reminder data, share the single
  Behn wakeup scheduler, survive reloads, and are pruned when their reminder is
  no longer actionable. The `%2` to `%3` migration and both new actions were
  verified through live Eyre. Preset durations and explicit future date/times
  are available from the details panel and CLI.
- The schedule editor now exposes recurrence weekdays, month dates, ordinal
  weekday, end date, and occurrence count. Reminder organization includes
  direct up/down placement controls in addition to explicit parent/section/rank.
- Native notifications offer Complete, Snooze, and Open actions. They are
  serialized through a desktop queue; Open deep-selects the reminder and the
  mutation actions remain disabled unless the home connection is Online.
- Quick entry recognizes whitespace-delimited `#tag` tokens. The overlay lists
  all known tags, filters any list or built-in view by exact tag, and can
  rename, merge, or delete one across hosted reminders; server-side bounds and
  set semantics were verified through live Eyre.
- `%4` freezes the complete `%3` schema, adds a canonical optional Urbit
  assignee to reminders, and durably allocates hosted-share, replica,
  invitation, and in-flight stores. A live `%3` ship migrated successfully and
  an assignment round-tripped through Eyre.
- Assignment now uses owner/member choices in the desktop and is checked again
  by Gall. Desk revision 38 rejected a syntactically valid non-member without
  changing state, accepted the owner, and accepted a revert to unassigned.
- `%5` freezes `%4` before collaboration receipts and pending-invitation
  payloads change shape. A live `%4` ship migrated successfully without losing
  its reminders, preferences, schedules, assignments, or snoozes.
- `%6` adds revisioned badge and all-day reminder policy without changing
  canonical reminder/list nouns. A live `%5` ship migrated, accepted a policy
  update, and preserved it across Gall suspend/revive.
- `%7` adds restart-safe owner and peer sessions plus liveness generations.
  `%8` adds durable notification presentation and a newest-4,096 operation
  receipt order. `%9` adds separate hosted and replica activity stores. `%10`
  adds participant-local list order, per-list presentation, collaboration-alert
  policy, and durable collaboration notifications. These migrations compile
  and were exercised on live desks without losing existing lists.
- Gall load and the `+tend!tend-migrations` release gate now call the same
  migration library. The table-driven gate constructs a non-empty noun for
  every `%0` through `%10` schema and verifies the sentinel list, reminder, and
  next-ID survive as `%10`; the complete matrix passes on Vere 4.6.
- The authenticated `/all` stream now emits one access record per visible list,
  including canonical host, owner flag, Online/Checking/Offline state, members,
  and pending invitations. The Omarchy UI visibly marks owner availability and
  disables every shared-list mutation while the host is Checking or Offline.
- Owner-authoritative sharing is active. Owners can invite any valid Urbit
  ship, delegate invite permission, revoke pending or accepted access, and
  distribute complete canonical list replicas. Invitees can accept or decline;
  participants can leave; all authorization is derived from the Gall source
  ship rather than a claimed payload identity.
- Successful invite submission exposes a copyable recipient-bound
  `omabit://tend/invite/<owner>/<token>` reference without list contents. The
  CLI accepts that URI for accept/decline while Gall still authenticates the
  originally invited ship; system URI-handler registration remains deferred.
- A repeatable three-ship harness verified two simultaneous replicas, participant
  editing, convergence, owner-offline readable replicas with blocked writes,
  owner restart/catch-up, duplicate-operation idempotence, stale rejection,
  a 12-submission Ames-held pressure burst, targeted revocation with an
  already-submitted edit still in flight, two participant/owner restart and
  resubscription cycles, and cleanup. The reproducible command and record are
  in `TEND_MULTISHIP_MATRIX.md`.
- Eyre connection files are written atomically with mode 0600, symbolic-link
  credential paths are refused, non-loopback HTTP and URL paths are rejected,
  expired authentication stops the reconnect loop, and users can disconnect or
  switch ships without retaining the reusable `+code`.
- Eyre hardening now disables redirects for authenticated traffic, replaces
  rather than merges cookie jars on a new login, bounds JSON/SSE/action sizes,
  and fails closed on malformed UTF-8, JSON, event IDs, and content lengths.
- `omabit tend` provides connect/disconnect/status, list/today/activity, add,
  complete/uncomplete, move/delete, snooze, share/unshare/leave,
  invitations/accept/decline, restore, and overlay-open commands with optional
  JSON output. It fails closed before shared writes when owner access is not
  Online and applies no planet/moon/comet rank restriction.
- Assigned to Me is available as a smart view and search includes assignee
  ships. Quick capture supports `#tag` entry and the bar badge can count Today,
  all incomplete, or assigned reminders—or be disabled. Users can choose the
  default time for new all-day reminders and whether stale all-day items stay
  in Today; timed overdue reminders remain visible.
- `omabit tend export` writes a complete `tend-backup-1` JSON snapshot without
  authentication material. Export is atomic, mode 0600, refuses symbolic-link
  targets, and does not replace an existing file without explicit `--force`.
  `validate-backup` now performs bounded offline shape, schedule, reference,
  URL, snooze, and parent-cycle checks without connecting to a ship.
- Export filters visible replicas using authenticated access metadata. Restore
  revalidates the envelope, requires explicit confirmation, commits atomically
  only into empty Gall state, and verifies a durable operation receipt. Live
  tests restored one list with five reminders and a section, rejected a second
  restore without mutation, and preserved the owner-only boundary.
- Notification presentation is durable across desktop disconnects. Gall keeps
  stable pending IDs until `ack-alert`; a live reconnect replayed one alert,
  acknowledgement removed it, and the next subscription did not replay it.
- Successful owner and remote list mutations append bounded, actor-attributed
  activity records. The log includes reminder targets and preserves the due
  instant of each individually or atomically batch-completed scheduled
  occurrence. Activity survives `%8` to `%9` migration, appears in the
  list-sharing panel and CLI, and was verified across both initial-snapshot and
  live-subscription replication with participant-local list aliases. A live
  two-reminder recurrence batch produced distinct stable IDs and preserved both
  completed due instants before the reminders were removed.
- Each ship now persists its own complete list ordering and per-list
  sort/direction without modifying the shared list. Pinned lists remain first;
  the overlay exposes explicit unpinned-list movement and restores each list's
  view when selection or local settings change. The `/settings` scry and live
  settings updates round-tripped on both owner and participant fake ships.
- Add, complete, and assignment collaboration notifications are generated from
  authenticated activity and filtered by a participant-local policy. They use
  stable durable IDs, replay until acknowledged, never notify an actor about
  their own operation, and clean up with removed lists. A live owner/participant
  run verified category suppression, completion and assignment delivery,
  reconnect replay, acknowledgement, leave cleanup, and restart pruning of an
  older orphaned due alert.
- Deterministic release packaging includes the desk, validated Omarchy plugin,
  CLI, installer, documentation, and per-file checksums. The installer refuses
  symlink targets and preserves forced replacements as timestamped backups.
- Plugin validation, JavaScript reducer tests, and Python Eyre transport/CLI
  tests pass. The desktop reducer ignores stale and duplicate list/preference
  facts so a delayed event cannot roll confirmed state backward.
- The overlay now provides documented keyboard shortcuts, a focusable reminder
  list with arrow/Enter/Space operation, explicit names for otherwise ambiguous
  controls, textual state cues, and no custom motion. A manual Orca/theme/200%
  scaling pass remains a release-candidate gate.
- Bar clicks route the overlay to their output, keyboard/CLI summons use
  Hyprland's focused output, and scaled card limits clamp to that screen. Quick
  capture stacks vertically on narrow outputs. QML syntax/import lint now runs
  on the overlay and service in both source and packaged-release gates; the bar
  entrypoint's filename/type collision is covered by validation and static
  contracts instead.
- A deterministic 10,000-reminder scale fixture now gates snapshot
  normalization, Today, and text search at five seconds each and verifies that
  the fixture remains below the 32 MiB Eyre event ceiling.

## Next

1. Complete manual assistive-technology validation and run a longer real-ship
   release-candidate key-continuity soak using the documented lockstep matrix.
2. Exercise the responsive capture and focused-output routing in the manual
   multi-monitor release-candidate pass.

The current Core reconciliation and honest release blockers are tracked in
[`TEND_RELEASE_CHECKLIST.md`](TEND_RELEASE_CHECKLIST.md).

The source of truth for scope and exit criteria remains
[`PRODUCT_ARCHITECTURE_PLAN.md`](../PRODUCT_ARCHITECTURE_PLAN.md).
