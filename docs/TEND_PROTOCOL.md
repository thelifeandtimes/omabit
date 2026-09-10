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

## Implemented action kinds

| Action | Effect |
| --- | --- |
| `create-list` | Create a standard list and select the first list as default |
| `rename-list`, `delete-list` | Rename or remove a list |
| `update-list` | Atomically change a list's title, color, and symbol |
| `add-section`, `update-section`, `delete-section` | Manage ordered sections; deleting a section keeps and unsections its reminders |
| `add-reminder`, `update-reminder` | Create a reminder or replace its editable metadata |
| `move-reminder` | Change parent, section, and stable rank after validating references and cycles |
| `set-schedule` | Set/clear due instant, all-day policy, IANA zone, early offsets, and recurrence |
| `set-completed` | Complete/uncomplete a reminder tree, or advance a repeating reminder |
| `set-preferences` | Atomically update revisioned default-list, pin, and snooze-preset state |
| `snooze-reminder` | Schedule a personal one-shot alert without modifying shared reminder data |
| `delete-reminder` | Delete a reminder and all descendants |

List appearance strings are non-empty and bounded; list deletion requires a
second confirmation in the desktop UI. Editable reminder metadata currently
includes title, notes, safe HTTP(S) or
`mailto` URL, priority, flag, tags, parent, section, and rank. All writes are
bounded by list revision; titles are non-empty and capped at 1 KiB.

Schedules store a canonical Urbit `@da` due instant plus the originating IANA
time-zone name. Recurrence is structured rather than cron text: hourly, daily,
weekly, monthly, or yearly frequency; positive interval; optional weekdays,
month dates, ordinal weekday, end instant, and occurrence limit. Weekly
weekday, monthly date/ordinal, month-end clamping, leap-year, end, and count
semantics are evaluated by the Gall agent. Early offsets are capped at one
year.

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

## Persistence

Gall is authoritative. State schema `%3` contains lists, the next local ID,
operation receipts, revisioned personal preferences, active snoozes, and Behn
timer generation. `+on-load` migrates `%0`, `%1`, and `%2`, preserving IDs,
titles, completion state, revisions, schedules, and the default list while
filling new fields with deterministic defaults.

The agent keeps one earliest Behn wakeup across all outstanding due/early
alerts and snoozes. Generation-tagged wires make replaced timers harmless. Fired offsets
are persisted on the schedule, so wake replay, agent reload, and ship restart
cannot redeliver an already-recorded occurrence; reload re-arms an outstanding
timer and overdue wakeups run immediately. Snoozes are personal state keyed by
list/reminder and are discarded if the reminder is deleted, completed, or loses
its schedule.

The current ID atoms are scoped to the hosting ship. The multiplayer protocol
will expose them as `[host=@p local-id]` references without requiring a global
database or identity provider.
