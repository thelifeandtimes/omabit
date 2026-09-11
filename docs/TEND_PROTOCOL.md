# Tend protocol notes

Status: pre-release protocol, implemented by `%tend` and `io.omabit.tend`.

## Desktop transport

The Omarchy plugin authenticates to the user's own Eyre endpoint with the ship
domain and a current `+code`. The bridge stores only Eyre's session cookie and
the normalized endpoint/ship identity. Non-loopback HTTP is rejected.

The client opens one Eyre channel, subscribes to `%tend` at `/all`, acknowledges
every SSE event, and remains in `Checking` until it receives a complete
snapshot. Mutations use `%tend-action-1`; updates use `%tend-update-1`.
Every complete snapshot carries integer `protocol-version` 1. The desktop
requires an exact match during login, direct state reads, and streamed snapshot
replacement; a missing or different value fails closed before editing is
enabled. See `TEND_COMPATIBILITY.md` for the pre-release lockstep policy.

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
| `set-list-order` | Replace this ship's private order for every currently visible list |
| `set-list-presentation` | Save this ship's private sort and direction for one visible list |
| `set-collaboration-policy` | Save this ship's private add/complete/assignment notification choices for one visible list |
| `snooze-reminder` | Schedule a personal one-shot alert without modifying shared reminder data |
| `replace-tag` | Rename, merge, or delete one tag across the user's hosted reminders |
| `delete-reminder` | Delete a reminder and all descendants |
| `invite`, `unshare` | Owner grants or revokes list access, with optional participant invite permission |
| `accept-invite`, `decline-invite` | Accept or discard an authenticated inter-ship invitation |
| `leave-list` | Remove the local replica and ask the owner to remove this participant |
| `restore-empty` | Atomically restore one validated owner-only `tend-backup-1` envelope into empty state |

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
instant, early offset, whether it is a snooze wake, and a stable notification
ID; the desktop maps that event to a native notification. Alerts remain in
Gall's durable pending-notification map and are replayed to a newly subscribed
desktop until `ack-notification` removes them.

The desktop queues native notifications so each alert can expose Complete,
Snooze, and Open actions. Complete uses the current list revision, Snooze uses
the user's first configured preset, and Open routes a list/reminder selection
back into the overlay. Actions that mutate list state are submitted only while
the home connection is Online.

The Milestone 0 event shapes remain accepted by the JavaScript reducer during a
rolling desktop reload, but new agents emit only the canonical update forms.
That reducer tolerance does not make mixed desk/client releases supported.

`accesses` publishes the local alias, authoritative host and host-local list ID,
owner flag, host status, member policies, and pending invitees for every visible
list. `invitations-updated` replaces the local invitation inbox.
`operation-pending` and `operation-settled` expose the remote in-flight ledger
without creating an offline mutation queue. The desktop treats an absent,
Checking, or Offline access record as read-only.

`activities-updated` replaces the bounded activity log for one locally visible
list. Entries identify the authenticated Urbit actor, event kind, event time,
optional reminder, and—for every individually or batch-completed scheduled
occurrence—the exact due instant that was completed. Batch entries derive a
stable event ID from the operation and reminder, so every selected recurrence
keeps its own history. The overlay renders this log in list settings, and
`omabit tend activity LIST` exposes the same data to scripts and terminals. A
log retains the newest 1,000 entries per list.

`local-settings-updated` replaces private list order, per-list presentation,
and collaboration-notification policy for the currently authenticated ship.
Those settings are never included in peer snapshots. `collaboration-alert`
identifies the list, optional reminder, authenticated actor, event category,
and stable notification ID. Each participant derives these alerts from newly
received activity using its local policy; the actor never receives one for its
own operation. Collaboration alerts share the durable replay and
`ack-notification` lifecycle used by due alerts.

The `%tend-peer-1` noun mark carries invite/accept/decline/leave, canonical list
snapshots, mutations, removals, liveness messages, and mutation rejections.
Inviting a ship transmits the complete selected list, its membership metadata,
and its activity log to that ship. The participant stores them as a
non-authoritative replica and receives later owner snapshots in order. Remote
edits are accepted only while the owner's application-level session is Online,
then routed to the owner for authenticated authorization and sequencing. A
restart rotates the owner session and forces every participant through Checking
and catch-up before writes are enabled again.

Current liveness timing is a five-second heartbeat with an Offline transition
after twelve seconds without a valid acknowledgement. A remote mutation must
have been submitted within ten seconds and may not be more than thirty seconds
in the future. Those constants are protocol behavior in the current pre-release
and may be tuned before a stable wire-version commitment.

Authenticated scries expose `/state`, `/accesses`, `/invitations`,
`/activities/<local-list-id>`, `/settings`, `/receipt/<operation-id>`, and
`/whoami`.
Restore uses the receipt scry to prove that Gall durably accepted or rejected
the operation; an Eyre poke acknowledgement by itself is not reported as
restore success.

An invitation's operation ID is also its single invitation token. Clients may
render the recipient-bound reference as
`omabit://tend/invite/<owner>/<percent-encoded-token>`. The URI carries no list
content and does not bypass Gall: acceptance still reaches the recipient's home
agent, then the owner verifies the authenticated source ship against its pending
invite. `omabit tend accept` and `decline` accept either this URI or the explicit
owner/token pair. System-wide URI-handler registration remains deferred.

## Persistence

Gall is authoritative. State schema `%10` contains lists, the next local ID,
bounded operation receipts, revisioned personal preferences, active snoozes,
Behn timer generation, hosted-share policies, remote replicas, invitations,
in-flight operations, peer sessions/liveness generations, durable pending
notifications, replica alert-delivery state, separate hosted/replica activity
maps, private list/presentation/collaboration policy, and durable collaboration
notifications. `+on-load` migrates `%0` through `%10`, preserving canonical
reminder data while filling newer fields with deterministic defaults. `%7`
adds host and peer sessions plus liveness generation. `%8` adds durable
notifications and bounded receipt ordering. `%9` adds actor-attributed activity
and scheduled-occurrence records without changing canonical list nouns. `%10`
adds participant-local presentation and collaboration-notification state with
deterministic defaults. Loading also prunes pending alert records whose list is
no longer visible.

The migration implementation lives in `lib/tend-migrate.hoon`. Gall `+on-load`
and the `+tend!tend-migrations` table-driven release generator both call that
library, preventing a fixture from silently testing a duplicate implementation.
Every `%0` through `%10` fixture contains a sentinel list and reminder and checks
their IDs and content after migration.

The operation receipt ledger retains the newest 4,096 receipts. Peer retries
are deduplicated while their receipt is retained; clients must not treat an
operation older than that bounded window as safely replayable.

Personal preferences include the default list, pinned lists/views, snooze
presets, badge basis (`today`, `all`, `assigned`, or `none`), a local-wall-clock
minute for newly scheduled all-day reminders, and whether old all-day reminders
remain visible in Today. Timed overdue reminders always remain in Today.

The agent keeps one earliest Behn wakeup across all outstanding due/early
alerts and snoozes. Generation-tagged wires make replaced timers harmless.
Fired offsets are persisted on the schedule, while presentation remains pending
until a desktop acknowledges the stable notification ID. Reload re-arms an
outstanding timer and replays unacknowledged presentation; overdue wakeups run
immediately. Snoozes are personal state keyed by list/reminder and are
discarded if the reminder is deleted, completed, or loses its schedule.

IDs are scoped to the hosting ship. A replica receives a collision-free numeric
alias for the desktop while preserving its canonical `[host=@p local-id]`
reference in Gall state, without requiring a global database or identity
provider.

## Backup envelope

`omabit tend export` wraps owner-authoritative lists from the current local
snapshot in `tend-backup-1` with a source ship and UTC export timestamp. Visible
replicas are deliberately excluded using `/accesses`; restoring one can never
silently convert it into an owned list. `omabit tend validate-backup` performs
bounded, offline structural and referential validation. `omabit tend restore`
revalidates the same envelope, requires `--yes`, sends one `restore-empty`
transition, and verifies the durable Gall receipt. Gall accepts it only when no
hosted lists, replicas, invitations, or in-flight operations exist, and either
commits all imported content or none. Sharing relationships, subscriptions,
transient peer state, old operation receipts, and authentication material are
not restored. See `TEND_BACKUP.md` for the complete contract.
