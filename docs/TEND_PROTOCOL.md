# Tend protocol notes

Status: pre-release protocol, implemented by `%tend` and `io.omabit.tend`.

## Desktop transport

The Omarchy plugin authenticates to the user's own Eyre endpoint with the ship
domain and a current `+code`. The bridge stores only Eyre's session cookie and
the normalized endpoint/ship identity. Non-loopback HTTP is rejected.

The client opens one Eyre channel, subscribes to `%tend` at `/all`, acknowledges
every SSE event, and remains in `Checking` until it receives a complete
snapshot. Mutations use `%tend-action-1`; updates use `%tend-update-1`.

Every action is a one-key JSON object. Its body includes a unique
`operation-id`. Mutations against an existing list also include
`base-revision`. The owner either commits exactly one new list revision or
returns a `rejected` event with the current revision. Replaying an operation ID
returns its recorded result without applying it twice.

## Implemented local action kinds

| Action | Effect |
| --- | --- |
| `create-list` | Create a standard list and select the first list as default |
| `rename-list`, `delete-list` | Rename or remove a list |
| `update-list` | Atomically change a list's title, color, and symbol |
| `add-section`, `update-section`, `delete-section` | Manage ordered sections; deleting a section keeps and unsections its reminders |
| `place-section` | Place a section before/after another and atomically normalize sparse ranks |
| `add-reminder`, `update-reminder` | Create a reminder with initial tags or replace its editable metadata |
| `move-reminder` | Change parent, section, and stable rank after validating references and cycles |
| `place-reminder` | Place a reminder before/after a sibling and atomically normalize that sibling group's sparse ranks |
| `batch-set-completed` | Complete/uncomplete up to 500 selected reminders atomically, cascading through subtasks and advancing selected recurrence rules |
| `batch-move-reminders` | Move up to 500 selected reminder trees to a section in one list revision |
| `batch-delete-reminders` | Delete up to 500 selected reminder trees atomically |
| `set-schedule` | Set/clear due instant, all-day policy, IANA zone, early offsets, and recurrence |
| `set-completed` | Complete/uncomplete a reminder tree, or advance a repeating reminder |
| `set-preferences` | Atomically update revisioned default-list, pin, and snooze-preset state |
| `set-reminder-policy` | Atomically update badge mode, all-day alert minute, and all-day-overdue behavior |
| `snooze-reminder` | Schedule a personal one-shot alert without modifying shared reminder data |
| `replace-tag` | Rename, merge, or delete one tag across the user's hosted reminders |
| `delete-reminder` | Delete a reminder and all descendants |

List appearance strings are non-empty and bounded; list deletion requires a
second confirmation in the desktop UI. Editable reminder metadata currently
includes title, notes, safe HTTP(S) or
`mailto` URL, priority, flag, tags, assignee ship, parent, section, and rank. All writes are
bounded by list revision; titles are non-empty and capped at 1 KiB. Tags are
non-empty, bounded strings with at most 100 on one reminder. Quick entry strips
whitespace-delimited `#tag` tokens into the initial tag set; replacing a tag
updates every affected list and merges duplicate target values through set
semantics. Batch selections must be non-empty, contain only reminders in the
addressed list, and are capped at 500 IDs.

Pointer drag/drop and keyboard up/down placement use `place-reminder`. Source
and target must be distinct reminders in the same parent/section sibling
group. Gall derives the complete order from canonical state and rewrites that
group to ranks 1024, 2048, … in the same list revision; it does not trust a
client-computed midpoint. Explicit parent or section moves use
`move-reminder`, and changing a parent's section updates all descendants in
the same atomic revision.

An assignee is valid only when it is the list owner or appears in that list's
current member map. The Gall agent checks this against canonical state; the
desktop presents the same owner/member set rather than accepting free-form
ship text. Rank is irrelevant—the identity is an unrestricted Urbit `@p`.

Schedules store a canonical Urbit `@da` due instant plus the originating IANA
time-zone name. Recurrence is structured rather than cron text: hourly, daily,
weekly, monthly, or yearly frequency; positive interval; optional weekdays,
month dates, ordinal weekday, end instant, and occurrence limit. Weekly
weekday, monthly date/ordinal, month-end clamping, leap-year, end, and count
semantics are evaluated by the Gall agent. Early offsets are capped at one
year.

The desktop submits date/time editor values as local wall-clock text. The
Python bridge resolves them with the operating system's IANA `zoneinfo` data
and sends only canonical `@da` instants to Gall. Explicit `Z` or numeric-offset
inputs retain their stated instant. A nonexistent wall time in a daylight
saving gap is rejected; an ambiguous fold selects the earlier instant. For an
all-day reminder, the user's configured all-day alert minute is interpreted in
the selected zone. Stream updates add display-only `local-due` and `local-end`
fields derived from the canonical instant and stored zone. Those fields are
never persisted or sent in `%tend-action-1`. Today evaluates enriched all-day
reminders by that local calendar date rather than by the UTC alert instant;
the user's all-day-overdue preference decides whether earlier dates remain.

The current resource ceilings are 10,000 hosted lists, 10,000 sections per
list, and 100,000 reminders per list. Notes are capped at 64 KiB, URLs at 8
KiB, appearance/tag/time-zone strings at 128 bytes, early-offset sets at 64,
weekday sets at 7, month-date sets at 31, snooze presets at 32, and batch
selections at 500. These are rejection ceilings, not recommended interaction
sizes; the 32 MiB Eyre event ceiling will normally become the tighter snapshot
limit.

## Updates

`snapshot` contains all visible lists, personal preferences, and active
snoozes. `list-upserted` replaces one canonical list after any successful list
mutation. `list-deleted` removes one list and returns cleaned preference state.
`preferences-updated` and `snoozed` affect only the user's own ship.
`rejected` never changes canonical list state. Lists, reminders, and preferences
carry revisions or Urbit timestamps. `alert` identifies the reminder, due
instant, early offset, and whether it is a snooze wake; the desktop maps that
event to a native notification.

The desktop queues native notifications so each alert can expose Complete,
Snooze, and Open actions. Complete uses the current list revision, Snooze uses
the user's first configured preset, and Open routes a list/reminder selection
back into the overlay. Actions that mutate list state are submitted only while
the home connection is Online.

The Milestone 0 event shapes remain accepted by the JavaScript reducer during a
rolling upgrade, but new agents emit only the canonical update forms.

`accesses` publishes the local alias, authoritative host and host-local list ID,
owner flag, host status, member policies, and pending invitees for every visible
list. `invitations-updated` replaces the local invitation inbox.
`operation-pending` and `operation-settled` expose the remote in-flight ledger
without creating an offline mutation queue. The desktop treats an absent,
Checking, or Offline access record as read-only.

The `%tend-peer-1` noun mark and `%6` state reserve versioned peer envelopes for
invite/accept/decline/leave, list snapshots, mutations, removals, and mutation
rejections. The transport handlers that send these envelopes to another ship
are not enabled in the current checkpoint; local clients cannot cause peer data
egress until that trust boundary is explicitly enabled and tested.

## Persistence

Gall is authoritative. State schema `%6` contains lists, the next local ID,
operation receipts, revisioned personal preferences, active snoozes, Behn timer
generation, hosted-share policies, remote replicas, invitations, and in-flight
operations. `+on-load` migrates `%0` through `%3`, preserving IDs, titles,
completion state, revisions, schedules, tags, preferences, and snoozes while
filling assignee and collaboration stores with deterministic defaults. `%4` is
frozen and migrates to `%5`; `%5` is frozen and migrates to `%6`, adding badge
and all-day reminder policy defaults without reinterpreting older receipt,
preference, or pending-invitation nouns.

Personal preferences include the default list, pinned lists/views, snooze
presets, badge basis (`today`, `all`, `assigned`, or `none`), a local-wall-clock
minute for newly scheduled all-day reminders, and whether old all-day reminders
remain visible in Today. Timed overdue reminders always remain in Today.

The agent keeps one earliest Behn wakeup across all outstanding due/early
alerts and snoozes. Generation-tagged wires make replaced timers harmless. Fired offsets
are persisted on the schedule, so wake replay, agent reload, and ship restart
cannot redeliver an already-recorded occurrence; reload re-arms an outstanding
timer and overdue wakeups run immediately. Snoozes are personal state keyed by
list/reminder and are discarded if the reminder is deleted, completed, or loses
its schedule.

IDs are scoped to the hosting ship. A replica receives a collision-free numeric
alias for the desktop while preserving its canonical `[host=@p local-id]`
reference in Gall state, without requiring a global database or identity
provider.

## Backup envelope

`omabit tend export` wraps the current local snapshot in `tend-backup-1` with a
source ship and UTC export timestamp. `omabit tend validate-backup` performs
bounded, offline structural and referential validation before any future
restore operation is allowed. The current snapshot contains locally hosted
lists; the peer engine must add explicit non-authoritative replica metadata
before shared replicas can enter this format. See `TEND_BACKUP.md` for the
empty-state-only atomic restore contract.
