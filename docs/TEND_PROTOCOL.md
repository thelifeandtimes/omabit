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
| `add-section`, `update-section`, `delete-section` | Manage ordered sections; deleting a section keeps and unsections its reminders |
| `add-reminder`, `update-reminder` | Create a reminder or replace its editable metadata |
| `move-reminder` | Change parent, section, and stable rank after validating references and cycles |
| `set-completed` | Complete/uncomplete a reminder tree |
| `delete-reminder` | Delete a reminder and all descendants |

Editable reminder metadata currently includes title, notes, safe HTTP(S) or
`mailto` URL, priority, flag, tags, parent, section, and rank. All writes are
bounded by list revision; titles are non-empty and capped at 1 KiB.

## Updates

`snapshot` contains all visible lists. `list-upserted` replaces one canonical
list after any successful list mutation. `list-deleted` removes one list.
`rejected` never changes canonical list state. Lists and reminders carry their
own revisions and Urbit timestamps.

The Milestone 0 event shapes remain accepted by the JavaScript reducer during a
rolling upgrade, but new agents emit only the canonical update forms.

## Persistence

Gall is authoritative. State schema `%1` contains lists, the next local ID,
operation receipts, and the default list. `+on-load` migrates the original `%0`
list/reminder tuples into `%1`, preserving IDs, titles, completion state, and
revisions while filling new fields with deterministic defaults.

The current ID atoms are scoped to the hosting ship. The multiplayer protocol
will expose them as `[host=@p local-id]` references without requiring a global
database or identity provider.
