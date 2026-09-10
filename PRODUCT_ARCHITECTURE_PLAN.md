# Omabit Tend: Product and Architecture Plan

Status: initial design specification  
Research baseline: Apple Reminders in iOS/macOS 26, as documented on 2026-09-10  
Product thesis: an Apple Reminders-class application in the Omabit family whose native UI is an Omarchy shell plugin and whose identity, durable state, and multiplayer synchronization live on Urbit.

## 1. Product definition

Omabit is the umbrella project for Omarchy–Urbit applications and tooling. **Tend** is the reminder application, `%tend` is its Gall agent/desk, `io.omabit.tend` is its Omarchy plugin, `omabit tend …` is its umbrella CLI route, and `omabit://tend/…` is its capability-link namespace.

Tend is a self-hosted, replica-backed multiplayer reminder application for Omarchy Linux users.

- The user interacts through a native Quickshell-based Omarchy plugin, not a web application.
- Every user connects the plugin over Eyre/HTTP to an Urbit ship running the `%tend` Gall agent.
- An Urbit `@p` is the user/network identity. Gall and Ames provide authenticated ship-to-ship messaging; there is no Google, Apple, GitHub, or custom account service.
- Planets, moons, and comets are all supported. Authentication, invitations, ACLs, assignment, and collaboration must not impose a ship-rank restriction.
- Reminder data is held by user-controlled Urbit ships. There is no vendor-operated application database.
- Shared lists are replicated to the ships of their participants, with live updates and offline reading.
- The runtime does not require hosted AI, push-notification, analytics, or object-storage services.

The product should match the *in-scope outcomes* of Apple Reminders on Omarchy, not reproduce Apple-only APIs. Siri-style entry becomes keyboard/CLI natural-language capture, and iOS widgets become an Omarchy bar widget. Calendar integration, message/contact triggers, voice entry, and intelligence-generated suggestions are deferred explicitly.

### Success criteria

1. One Omarchy user can install the plugin and Urbit desk, create a reminder, close/restart the shell or Urbit, and recover the same state.
2. Two ships can share a list, edit it while its owner ship is reachable, assign work, receive updates, and continue reading a local replica while the owner is offline without any Omabit-operated server.
3. Every operation that crosses ships is authorized by Urbit identity and the list ACL.
4. The core daily workflow—quick add, today, scheduled, complete, snooze, repeat, search, and shared assignment—feels instant from the keyboard.
5. The data and wire formats are versioned and upgradeable before broader parity work begins.

### Explicit non-goals

- Compatibility with Apple's private Reminders/iCloud protocol.
- Literal Siri, Apple Watch, CarPlay, Apple Calendar, or Messages integration.
- Calendar views/interchange, Tlon Messenger (`%groups`) triggers, voice capture, AI-derived suggestions/auto-categorization, and printing are outside the current roadmap.
- Third-party Reminders account backends such as iCloud or Exchange; the Urbit Gall agent intentionally replaces the account/backend layer.
- A vendor-run relay, notification gateway, database, account directory, or mandatory LLM.
- Anonymous public collaboration in the initial release.
- Byzantine consensus. The initial system assumes list participants are ordinary collaborators and the list owner is trusted to sequence the list.

## 2. What “feature parity” includes

Apple's current feature set is broader than task CRUD. Its documentation includes rich reminder metadata, recurrence, smart lists, sections and columns, grocery categorization, templates, attachments, system notifications, location triggers, calendar and share-sheet entry points, and shared-list permissions and assignment. iOS 26 also added suggested reminders, general auto-categorization, per-reminder time zones, quick capture, and urgent alarms. This research inventory is broader than Tend's immediate scope: calendar, message/contact triggers, voice, printing, generated suggestions, and general AI auto-categorization are deferred.

The tables below define the active target. “Core” is the daily-driver and multiplayer foundation; “Parity” follows after that foundation. Deferred capabilities are recorded separately rather than mixed into the implementation tiers.

### Reminder model and lifecycle

| Apple capability | Tend behavior | Tier |
|---|---|---|
| Create, edit, complete, uncomplete, move, and delete reminders | Full keyboard and pointer workflow; batch complete/move/delete | Core |
| Title and notes | Unicode plain text initially; preserve line breaks in notes | Core |
| Date, time, and all-day reminders | Separate all-day and timed schedules with a user-defined default all-day alert time | Core |
| Per-reminder time zone | Store an IANA zone plus wall-clock time; render in local or reminder zone | Core |
| Early reminder | Zero or more offsets before the due instant | Core |
| Repeating reminders | Structured hourly/daily/weekly/monthly/yearly rules, intervals, selected weekdays/dates, ordinal weekdays, and optional end date/count | Core |
| Complete a repeating occurrence | Preserve occurrence history and materialize the next occurrence | Core |
| Priority | None, low, medium, high | Core |
| Flag | Independent flag field and Flagged view | Core |
| Tags | Create, suggest, rename, merge, delete, and filter tags; recognize `#tag` in quick entry | Core |
| URL | Validated clickable URL with safe protocol handling | Core |
| Subtasks | Nested reminders, collapse/expand, indent/outdent; parent complete/delete/move cascades | Core |
| Manual ordering | Stable fractional rank keys, drag/drop, and keyboard move commands | Core |
| Search | Search title, notes, tags, assignee, list, and attachment metadata | Core |
| Snooze/defer from notification | Configurable presets plus custom date/time | Core |
| Urgent alarm | Persistent full-screen/overlay alert with sound and explicit complete/snooze/dismiss actions | Parity |
| Attach image/photo/scanned document | File picker, image preview/thumbnail, content-addressed encrypted transfer, quotas, and local cache | Parity |
| Location alert | Optional GeoClue position/geofence plus Wi-Fi/Bluetooth arrival/departure adapters | Parity |
| Natural-language entry | Parse phrases such as “every Wednesday at 5” locally and show the parse before saving | Parity |
| Recently Deleted | Tombstones recoverable for 30 days, followed by compaction/garbage collection | Parity |

Apple's documented recurrence includes hourly through yearly custom schedules, selected weekdays and month dates, ordinal patterns such as “last weekday,” and an end condition. Completing a repeating reminder exposes the next occurrence. Tend should test these as domain semantics, not delegate them to the UI or a platform cron parser.

### Lists, organization, and views

| Apple capability | Tend behavior | Tier |
|---|---|---|
| Standard lists | Create, rename, recolor, choose icon/emoji, reorder, and delete | Core |
| Built-in smart lists | Today (including overdue), Scheduled, All, Flagged, Completed, Assigned to Me, and Groceries | Core |
| Custom smart lists | Any/all predicates over tags, dates/ranges, time, priority, flag, location, list, completion, and assignment | Parity |
| Sections | Create, rename, reorder, delete, and move reminders between sections | Core |
| List and column/Kanban views | Sections become columns; unsectioned items appear under Other | Parity |
| Sort | Manual, due date, creation date, priority, or title, in both directions | Core |
| Per-participant sort for shared lists | Store presentation preferences on each participant's own ship, not in shared list state | Core |
| Groups of lists | Create/rename/delete groups and move private lists among them | Parity |
| Pin lists | Pin/reorder favorite lists and built-in smart views | Core |
| Default list | Configurable destination for capture outside a list | Core |
| Grocery list | Deterministic category taxonomy, manual category corrections, learned local overrides, and reset | Parity |
| Templates | Save immutable list snapshots, instantiate them, export/import, and share a capability link | Parity |

Smart lists are saved queries, not containers. Deleting one must not delete matching reminders. Private smart lists, pinning, grouping, sidebar visibility, sort, collapsed sections, and notification preferences belong to a participant's local state; list title, sections, task ranks, recurrence, and assignments are shared.

### Collaboration

| Apple capability | Tend behavior | Tier |
|---|---|---|
| Invite and accept | Invite any Urbit `@p`—planet, moon, or comet; deliver an in-app invitation and a copyable `omabit://tend/…` capability URI | Core |
| Live shared edits | Initial snapshot followed by ordered deltas over Gall subscriptions | Core |
| Host availability | Show Online, Checking, or Offline for each shared list; enable mutations only while the owner ship is Online | Core |
| Work offline | Keep a full read-only replica; do not accept or queue new shared-list edits while the owner ship is Offline or Checking | Core |
| Assign a reminder | Zero or one assignee from the current participant set; assignment notification | Core |
| Add/remove people and stop sharing | Owner-controlled ACL; removing a ship kicks subscriptions and rejects later actions | Core |
| Allow others to invite | Global default and per-participant `can-invite` permission | Core |
| Notify on item added/completed | Per-participant switches, evaluated by that participant's local agent | Core |
| Activity | Actor-attributed event feed with add/edit/move/complete/assign/member events | Core |
| Connection/pending status | Show host reachability and any operation that was already in flight when connectivity changed | Core |
| Shared templates | Publish an immutable snapshot using a revocable or expiring capability | Parity |

Apple notifies each collaborator independently: sharing a list does not make one person's alert delivery settings everybody's settings. Tend should preserve that separation. Shared task schedule and assignment are list data; early offsets, urgency, sounds, snooze defaults, location details, all-day notification time, and add/complete activity notifications are participant-local.

### Omarchy/Linux-native surfaces

| Surface | Required behavior | Tier |
|---|---|---|
| Full app | Summonable full-screen `overlay` optimized for keyboard navigation, with sidebar, list, details drawer, and list/column modes | Core |
| Quick capture | Small summonable capture mode with natural-language preview and default-list selection | Core |
| Bar widget | Overdue/today count, next item, sync state, and click-to-open | Core |
| Badge and Today policy | Configurable badge count, all-day notification time, and whether all-day items become overdue the next day | Core |
| Headless plugin service | Maintains the Eyre channel, receives updates, dispatches notifications, and recovers after shell reload | Core |
| Desktop notifications | Native Omarchy notifications with Complete, Snooze, and Open actions | Core |
| CLI | `omabit tend add`, `today`, `list`, `complete`, `snooze`, `open`, `share`, and machine-readable JSON output | Core |
| Global shortcut | User-configurable quick-add and full-app bindings | Core |
| Share/open handler | `omabit://tend/…` URI route and commands that accept selected text, a URL, or a file | Parity |
| Accessibility | Complete keyboard control, visible focus, scalable text, contrast-safe theme binding, screen-reader labels where supported, and reduced-motion behavior | Core |

The exact Apple-only surfaces are deliberate substitutions:

- Siri → quick-add shortcut, CLI, and deterministic natural-language parser.
- iOS/macOS widgets → Omarchy bar widget and overview panel.
- Apple share sheet → CLI/stdin, browser helper, file/URI handler, and clipboard capture.

### Deferred capabilities

The following researched Apple features are deliberately outside the current roadmap: calendar views/interchange, printing, voice capture, suggested reminders, and general AI auto-categorization. Message/contact triggers are also deferred; the intended future route is a Tlon Messenger integration through `%groups`, designed only after the reminder and multiplayer protocols are stable.

## 3. Recommended system architecture

```text
┌──────────────────────────── Omarchy session ────────────────────────────┐
│  io.omabit.tend plugin                                                  │
│  ┌────────────┐  ┌─────────────────┐  ┌──────────────────────────────┐  │
│  │ Bar widget │  │ Full UI/overlay │  │ Headless QML service         │  │
│  └──────┬─────┘  └────────┬────────┘  │ Eyre channel + notifications │  │
│         └─────────────────┴───────────┴──────────────┬───────────────┘  │
│                               authenticated HTTP/SSE │                  │
└─────────────────────────────────────────────────────┼──────────────────┘
                                                      │
                                            ┌─────────▼─────────┐
                                            │ User's Urbit ship │
                                            │ %tend Gall agent  │
                                            │ - hosted lists    │
                                            │ - replicas        │
                                            │ - in-flight ops   │
                                            │ - alerts/prefs    │
                                            └─────────┬─────────┘
                                                      │ Gall over Ames
                               authenticated, ordered,│end-to-end encrypted
                 ┌────────────────────────────────────┼────────────────────┐
                 │                                    │                    │
        ┌────────▼────────┐                  ┌────────▼────────┐  ┌────────▼────────┐
        │ Participant ~a  │                  │ List host ~b    │  │ Participant ~c  │
        │ replica/agent   │                  │ canonical state │  │ replica/agent   │
        └─────────────────┘                  └─────────────────┘  └─────────────────┘
```

### 3.1 One Gall agent on every ship

Use a single `%tend` agent with four logical stores:

1. **Hosted lists**: canonical state for lists created by this ship.
2. **Replicas**: the last accepted snapshots/deltas for lists hosted by other ships.
3. **In-flight ledger**: idempotent operations that were accepted while a host was Online but whose acknowledgement is still unresolved. It is not an offline edit queue.
4. **Personal state**: UI preferences, smart lists, alert policies, location definitions, pending notifications, and the invitation inbox.

The frontend always talks to the `%tend` agent on its user's home ship through an authenticated Eyre base URL—normally the user's HTTPS ship domain, but optionally a custom HTTPS endpoint or local loopback—even when displaying a list hosted by another participant. It never needs login credentials for every collaborator and never sends UI-originated traffic directly to another person's ship.

Gall is both the database and state machine. Its state is persistent, while `+on-save`/`+on-load` provide explicitly versioned upgrades. Do not introduce SQLite, Postgres, Firebase, or a filesystem task database. A small `$XDG_CACHE_HOME/omabit/tend` cache may improve shell startup, but it is disposable and never authoritative.

### 3.2 Owner-sequenced shared lists

For the initial protocol, a list ID is `[host=@p id=@uvH]`. The creating ship is the host and the only sequencer for that list.

1. The participant's home agent maintains the host state as Online, Checking, or Offline. Only Online permits a new mutation.
2. While Online, a participant edits its local replica optimistically; the local agent records the operation as in-flight and pokes the host's `%tend` agent.
3. The host authenticates `src.bowl`, verifies current membership/permission and operation validity, deduplicates by operation ID, assigns the next list revision, commits atomically, and emits a delta.
4. All subscribed participant agents receive and apply the delta, clear acknowledged in-flight entries, and publish a local frontend update.
5. If connectivity changes before acknowledgement, the operation stays visibly Pending and may be retried idempotently after the host returns. No additional shared-list edits are accepted while the state is Checking or Offline.
6. Reconnects request deltas after a known revision; if the history window no longer covers the gap, the host sends a complete snapshot. The list becomes Online/writable only after catch-up completes.

This design is federated rather than centrally hosted: different lists live on different owners' ships, and every participant stores a full replica. When a list's owner ship is offline, other members can read the last confirmed replica but cannot create, edit, complete, reorder, assign, attach, or change membership. The UI explains that the host is offline instead of pretending an edit will synchronize later.

This removes the need for an offline merge protocol, CRDT, anti-entropy between participants, or distributed membership epochs. ACL changes, recurrence, ordering, deletion, and attachment garbage collection remain deterministic and auditable on the owner's ship.

Host status is an application state, not a guess from a stale UI connection:

- **Checking**: initial subscription, reconnect, or snapshot catch-up is in progress. Read-only.
- **Online**: the remote subscription is acknowledged, the replica is current, and a recent application-level liveness probe was acknowledged. Writable.
- **Offline**: the subscription was kicked, Ames reported congestion/unresponsiveness, a probe timed out, or the host could not be reached. Read-only.

The agent retries subscriptions and low-frequency liveness probes with bounded backoff. A successful transport connection alone does not unlock editing; catch-up must finish first.

### 3.3 Wire protocol

Define all shared structures once in `/sur/tend.hoon`. Use separate, versioned marks with JSON conversion:

- `%tend-action-1`: frontend-to-home-agent commands and inter-ship mutations.
- `%tend-update-1`: snapshots, deltas, acknowledgements, conflicts, invitations, and connection status.
- `%tend-blob-1`: bounded attachment chunks and metadata.
- `%tend-export-1`: portable backup/template form.

All protocol envelopes carry:

- protocol version;
- operation/message ID;
- list ID where applicable;
- author ship and source device ID;
- base list/entity revision and client submission time;
- payload kind and data.

The host must derive authority from `src.bowl`, never from the claimed JSON author. Incoming remote data is validated by its mark before application code sees it. Every mutation is idempotent. Keep a bounded deduplication set past the maximum retry window.

Subscriptions:

- `/ui/all`: frontend snapshot and all home-agent changes.
- `/host/list/<id>/after/<revision>`: authorized participant snapshot/deltas for a hosted list.
- `/invite`: incoming invitations and membership changes.
- `/notification`: durable pending notification events for the local desktop.

Remote subscriptions can be kicked by Gall under backpressure or network failure, so subscriber logic must use bounded retry with jitter, then resubscribe from its last committed revision. It must stop retrying after an explicit authorization nack.

### 3.4 Conflict semantics

Use host revisions for canonical ordering and ordinary optimistic concurrency, not a CRDT or field-level clocks.

- Every mutable list, section, and reminder carries a host-assigned entity revision.
- Operations against different current entities can proceed independently even if the enclosing list revision advanced.
- An operation against a stale version of the same entity is rejected with the current canonical value. The UI reverts its optimistic state and offers to copy/reapply the user's text rather than merging silently.
- Complete/uncomplete is an explicit idempotent operation, not a boolean overwrite.
- Delete creates a tombstone; edits based before deletion cannot silently resurrect it.
- Moving a parent moves all descendants atomically.
- Reordering uses fractional rank keys. The host periodically normalizes ranks in one revision.
- ACL and ownership changes never use last-writer-wins; only the owner may commit them.

This gives an Apple-like “changes just appear” experience for concurrent online participants. Offline shared lists never enter this merge path because they are read-only.

### 3.5 Omarchy plugin composition

Use the third-party plugin ID `io.omabit.tend`; `omarchy.*` is reserved. Its manifest declares:

- `overlay` for the full application and quick-capture modes;
- `service` for a headless singleton kept alive with the Omarchy shell;
- `bar-widget` for glanceable state.

The plugin's service owns the connection and model. The overlay consumes that service rather than creating a second Urbit connection. The widget locates the same singleton through the shell service registry.

The plugin communicates with the user's ship through Eyre's authenticated external HTTP API. The onboarding form accepts an Eyre base URL and a current `+code`, regardless of whether the authenticated ship is a planet, moon, or comet:

1. Ask for the user's ship URL, for example `https://sampel-palnet.arvo.network`; also accept custom HTTPS Eyre endpoints and explicit local loopback URLs.
2. Ask the user to run `+code` on that ship and enter the current web login code. POST it to `/~/login`, then retain only the resulting session cookie.
3. Create an Eyre channel, subscribe to `%tend`, parse SSE updates, and acknowledge every event.
4. Send actions as channel PUTs/pokes.
5. Reauthenticate with a fresh `+code` when the session expires or the user changes ships.

The first implementation can keep this dependency-light by managing `curl --no-buffer` processes from the QML service and parsing Eyre's line-oriented SSE frames in QML. If that proves fragile under reconnect/backpressure tests, replace only this transport with a small audited bridge; do not change the Gall protocol or UI model.

Secrets belong in the desktop secret service when available. The fallback is a mode-0600 cookie jar with an explicit warning; never retain the reusable Urbit login code. Require HTTPS for a non-loopback URL.

Store runtime files under `$XDG_RUNTIME_DIR/omabit/tend`, preferences under `$XDG_CONFIG_HOME/omabit/tend`, and disposable cache under `$XDG_CACHE_HOME/omabit/tend`. Never write generated state into the installed plugin checkout: Omarchy watches that directory and treats changes as plugin reloads.

### 3.6 Notifications and timers

Each participant's home Gall agent schedules its own next alert through Behn. It should keep one earliest-wakeup timer, not one unbounded timer duct per reminder:

1. Compute the next time-based alert across local and replicated reminders.
2. Ask Behn to wake at that time.
3. On wake, enqueue every due notification, emit local facts, calculate repeat occurrences where appropriate, and schedule the next wake.
4. Keep notifications pending until the desktop service acknowledges delivery or the user acts.
5. On agent/shell restart, recompute missed alerts and apply a configurable grace window.

The Omarchy service presents the desktop notification. Complete and snooze actions poke the home Gall agent. Snooze is personal and can work from a cached shared reminder; completing a shared reminder is a list mutation and is disabled when its host is not Online. Urgent reminders use a persistent overlay and sound policy, but must honor user-level quiet/reduced-interruption settings.

Location and device triggers are intentionally personal. The shell service or a small adapter reports GeoClue, Wi-Fi SSID, Bluetooth-device, and lock/unlock events to the user's home agent. Exact coordinates and named home/work definitions do not enter shared-list state.

### 3.7 Attachments

Keep reminder metadata and blob bytes separate.

- Metadata contains content hash, MIME type, byte size, dimensions, filename, creator, and thumbnail hash.
- Blobs are content-addressed and chunked with strict per-blob/per-list quotas.
- The list host stores canonical bytes; participant ships cache only fetched blobs.
- Transfer occurs over authenticated Gall messages first. Prototype encrypted remote scry as a later performance optimization.
- Validate MIME by content, not extension; sanitize names; never execute or auto-open an attachment.
- Deleting the last reference starts a 30-day garbage-collection window aligned with Recently Deleted.

Large media is a genuine Urbit performance risk. Start with images/PDFs and a conservative configurable size limit, measure on real Ames links, and do not advertise arbitrary-file parity until those tests pass.

## 4. Domain model

The following is conceptual; the Hoon representation should favor explicit tagged unions and maps/sets.

### List

```text
list-id          [host=@p id=@uvH]
kind             standard | grocery
title            text
appearance       color + icon/emoji
sections         ordered map<section-id, section>
reminders        map<reminder-id, reminder>
members          map<@p, member-policy>
owner            @p
revision         @ud
event-window     ordered bounded log
created/modified timestamp + actor
deleted          optional tombstone
```

Smart lists, pinning, list groups, sort direction, column/list view, and collapsed state are not fields here because they are participant-local presentation state.

### Reminder

```text
reminder-id      @uvH
title            text
notes            text
url              optional validated URL
tags             set<tag-id>
priority         none | low | medium | high
flagged          boolean
parent           optional reminder-id
section          optional section-id
rank             fractional order key
assignee         optional @p
schedule         optional all-day/timed wall time + IANA zone
recurrence       optional structured recurrence rule
attachments      ordered set<blob-ref>
created/modified timestamp + actor
revision         host-assigned entity revision
completion       occurrence-aware completion history
deleted          optional tombstone
```

### Personal state

```text
default-list
smart-list definitions
list groups and pins
per-list view/sort/collapse
notification policies and snooze presets
all-day alert time and overdue policy
per-reminder early offsets and urgent policy
private location/device triggers keyed by reminder
pending delivered/undelivered alerts
grocery correction dictionary
```

### Invitation and membership

```text
invitation-id    random 128+ bit value
list-id
inviter/recipient @p
permission       edit + optional can-invite
expires-at
status           pending | accepted | declined | revoked
membership-epoch monotonic owner-controlled revision
```

A copied capability URI includes only what is needed to locate and redeem an invitation. It expires, is single-use by default, and is still bound to the intended recipient ship unless the owner explicitly chooses an open link.

## 5. Security and decentralization properties

### What the design guarantees

- Ship-to-ship traffic uses Ames, which authenticates peer identity and encrypts messages end-to-end in transit.
- Gall exposes the authenticated source ship to the application, allowing ACL enforcement without an external identity provider.
- Data lives in user-controlled piers and participant replicas, not an Omabit cloud database.
- The desktop UI talks only to the user's own ship.
- Planets, moons, and comets use the same Eyre login and `@p`-based application authorization path; Tend does not delegate identity to a centralized account provider.
- The service remains useful without hosted AI, telemetry, push, or file storage.

### What it does not guarantee

- Urbit is not infrastructure-free. Planet/star/galaxy identity uses the Azimuth PKI, currently on Ethereum, and Ames peer discovery/relay uses Urbit's network hierarchy.
- A list's owner ship is the initial trusted sequencer and availability point for committing changes. This is not a vendor database, but it is a per-list authority.
- Ames encryption protects data in transit. It does not by itself encrypt a pier, its host filesystem, backups, UI caches, or notification previews at rest.
- Removing a participant prevents future access but cannot erase data that participant already replicated or exported.
- The host can read lists it owns. This design is peer-to-peer authenticated transport, not sender-encrypted content opaque to the host.

These boundaries must be stated plainly in documentation and threat-model tests.

### Required controls

- Check `src.bowl` for every remote action and subscription.
- Reject unknown marks, protocol versions, oversized text/blobs, invalid nesting, cycles, invalid recurrence, unsafe URLs, and unauthorized assignees.
- Rate-limit invite redemption, blob upload, subscription snapshot, and mutation traffic per ship.
- Bound event logs, dedupe tables, tombstones, alert history, and unresolved in-flight retries.
- Use random operation/invitation IDs from Gall-provided entropy.
- Never interpolate reminder text, URLs, or ship input into shell commands.
- Require HTTPS for every non-loopback Eyre URL, reject embedded URL credentials, and scope session cookies to the configured endpoint.
- Make notification preview privacy configurable for locked screens.
- Provide JSON export and documented pier backup/restore testing before calling the product reliable.

## 6. Repository and package shape

```text
/
├── desk/
│   ├── app/tend.hoon
│   ├── sur/tend.hoon
│   ├── mar/tend-action-1.hoon
│   ├── mar/tend-update-1.hoon
│   ├── mar/tend-blob-1.hoon
│   ├── lib/tend/
│   │   ├── reducer.hoon
│   │   ├── auth.hoon
│   │   ├── recurrence.hoon
│   │   ├── ranking.hoon
│   │   └── json.hoon
│   ├── gen/tend-*.hoon
│   ├── desk.bill
│   ├── desk.docket-0
│   └── sys.kelvin
├── omarchy-plugin/
│   ├── manifest.json
│   ├── Service.qml
│   ├── Overlay.qml
│   ├── BarWidget.qml
│   ├── components/
│   ├── models/
│   ├── transport/
│   ├── icons/
│   └── bin/omabit-tend
├── tests/
│   ├── protocol/
│   ├── recurrence/
│   ├── fake-ships/
│   └── qml/
└── docs/
    ├── protocol.md
    ├── threat-model.md
    ├── operations.md
    └── parity-checklist.md
```

The desk and plugin can be released independently but must publish a compatibility range. Protocol negotiation should let an older participant read a snapshot and reject unsupported mutation kinds cleanly rather than crash.

Urbit software distribution can serve the desk from a publisher ship. Omarchy installs a third-party plugin from a reviewed Git repository with `omarchy plugin add`; distribution infrastructure is not a runtime data dependency.

## 7. Delivery plan with exit criteria

### Milestone 0 — architecture spikes

Build throwaway vertical proofs before committing the public schema.

- Two fake ships: host a list, authorize another ship, subscribe, add one item, and receive one delta.
- Disconnect/reconnect a subscriber and recover from a revision gap.
- Take the host offline, show the state transition in the Omarchy UI, and prove every shared mutation path becomes read-only until catch-up finishes.
- Omarchy service: authenticate to a remote HTTPS Eyre endpoint with URL plus `+code` (and to explicit loopback endpoints), subscribe over SSE, acknowledge events, survive shell restart, and render a task count.
- Repeat the authentication and basic CRUD spike with a planet, moon, and comet to prove that no client or Gall authorization path assumes a ship rank.
- Behn: enqueue a durable alert and recover an overdue alert after restart.
- Measure a 100 KB, 1 MB, and 5 MB attachment transfer over Ames.

Exit: written ADRs confirm hosting model, channel transport, snapshot/delta form, attachment limit, and notification ownership.

### Milestone 1 — single-user daily driver

- Lists and sections.
- Reminder CRUD, notes, URL, priority, flag, tags, subtasks, manual order.
- Date/time/all-day/time zone, early alerts, recurrence, complete/uncomplete, snooze.
- Today, Scheduled, All, Flagged, Completed, search, sort, pin, and default list.
- Full overlay, quick capture, service, bar widget, CLI, and native notifications.
- Versioned save/load migration and JSON backup/export.

Exit: use it for a week on one real ship with restart/upgrade/back-up drills and no lost or duplicated reminders.

### Milestone 2 — multiplayer core

- Invite/accept/decline/revoke.
- Owner ACL and `can-invite` policy.
- Host snapshots/deltas, local replicas, in-flight operation ledger, idempotency, resubscribe/backoff.
- Online/Checking/Offline state, edit gating, pending/conflict state for already-submitted online operations, and activity feed.
- Assignment and Assigned to Me.
- Per-user add/complete/assignment notifications.
- Participant removal and stop-sharing behavior.

Exit: three ships pass online editing, member-offline reading, owner-offline read-only enforcement, ambiguous in-flight delivery, duplicate delivery, stale edit, removal, and resubscription scenarios without unauthorized or lost canonical changes.

### Milestone 3 — organizational parity

- Custom smart lists and tag browser/management.
- List groups and complete per-user presentation state.
- Column/Kanban mode.
- Templates and capability sharing.
- Grocery categorization with local correction learning.
- Recently Deleted retention and compaction.
- Deterministic natural-language quick add for dates and recurrence.

Exit: the parity checklist covers every in-scope Apple organizational behavior with automated reducer/query tests.

### Milestone 4 — rich alerts and attachments

- Urgent overlay/alarm behavior.
- Image/PDF selection, thumbnails, chunking, quotas, caching, and garbage collection.
- Location, Wi-Fi, and Bluetooth trigger providers.
- Browser/clipboard/file share entry points and `omabit://tend/…` handling.

Exit: privacy, lock-screen, malicious attachment, poor-network, disk-pressure, and missed-wakeup tests pass.

### Milestone 5 — hardening and release

- State migration fixtures from every released schema version.
- Fuzz/property tests for recurrence, reducers, JSON marks, malformed remote actions, and ordering.
- Real-network soak tests and performance budgets.
- Reproducible desk/plugin releases, signed tags, update/rollback instructions, and compatibility matrix.
- Accessibility review, keyboard reference, onboarding, privacy model, backup/restore, and troubleshooting.

Exit: release candidate completes the entire in-scope parity checklist and multiplayer fault matrix on supported Omarchy and Urbit versions.

## 8. Test strategy

### Gall/domain tests

- Reducer tests for every action, permission, and tombstone transition.
- Table-driven recurrence tests around leap years, month ends, daylight-saving gaps/folds, time-zone changes, and completion after multiple missed occurrences.
- Property tests: operation deduplication, parent-cycle prevention, rank ordering, snapshot+delta equivalence, and unauthorized actions leave state unchanged.
- Save/load fixtures for forward migrations and downgrade rejection.

### Multi-ship integration tests

Run scripted fake ships for:

- first subscription and delta catch-up;
- duplicate/reordered frontend submissions;
- participant-offline reading and owner-offline edit blocking;
- recovery of an operation submitted while Online whose acknowledgement was lost;
- Gall kick/backpressure and bounded resubscribe;
- invite expiry/replay/wrong recipient;
- member removal during an outstanding edit;
- assignment to a removed member;
- continuity breach/key rotation behavior;
- mixed protocol versions;
- large-list snapshot and attachment transfer.

### Omarchy tests

- `omarchy plugin validate omarchy-plugin` in CI.
- QML linting plus unit tests for query/view models and SSE framing.
- Shell reload, lock/unlock, multi-monitor, theme change, scaling, no-network, ship-down, cookie-expiry, malformed endpoint, certificate failure, and remote HTTPS tests across planet, moon, and comet accounts.
- Notification complete/snooze/open actions, missed-alert recovery, privacy mode, and urgent overlay focus handling.
- Ensure runtime/cache writes never touch the plugin checkout.

### Performance budgets to establish during spikes

- Quick-capture surface visible from hotkey.
- Online optimistic mutation visible immediately without waiting for Ames.
- Warm list switching and search on at least 10,000 reminders.
- Bounded Gall state growth after deletion/compaction.
- Snapshot and delta sizes measured and capped.

Use measurements from the spike to set numeric release gates; do not invent targets before testing Quickshell and Ames on representative hardware/network links.

## 9. Settled implementation decisions

- The component is **Tend**, with `%tend`, `io.omabit.tend`, `omabit tend …`, and `omabit://tend/…` as its public namespaces.
- The Omarchy client connects directly to the user's own ship over Eyre/HTTP. Primary onboarding is an HTTPS ship domain plus a current `+code`; custom HTTPS and explicit loopback endpoints are also supported.
- Planets, moons, and comets are first-class users. Client validation, invitations, ACLs, assignments, replicas, and test fixtures must accept arbitrary valid Urbit `@p` identities.
- The list owner's ship is the authoritative sequencer. Shared lists are readable from replicas while that host is Offline or Checking, but all shared mutations are blocked until it is Online and caught up.
- Calendar integration, message/contact triggers, voice capture, printing, generated suggestions, and general AI auto-categorization remain outside the current roadmap. A later message-trigger design may integrate with Tlon Messenger through `%groups`.

There are no unresolved product-architecture questions blocking Milestone 0.

## 10. Research references

### Apple Reminders

- [Apple Reminders User Guide for macOS Tahoe](https://support.apple.com/en-lamr/guide/reminders/remn8ac82bcb/mac)
- [Get started with Reminders on Mac](https://support.apple.com/en-ca/guide/reminders/remne4b02adc/mac)
- [Add or change reminders on Mac](https://support.apple.com/en-ie/guide/reminders/remndc729e28/mac)
- [Add dates, locations, recurrence, early reminders, and urgent alarms](https://support.apple.com/en-ca/guide/reminders/remnd4b206fb/mac)
- [Share a reminder list and manage collaboration](https://support.apple.com/en-gb/guide/reminders/remnd509c0d9/mac)
- [Share, collaborate, and assign on iPhone](https://support.apple.com/guide/iphone/share-and-collaborate-iph2a8f9121e/ios)
- [Custom Smart Lists and filter fields](https://support.apple.com/en-ie/guide/reminders/remnfec66479/mac)
- [Sections and auto-categorization](https://support.apple.com/en-gb/guide/reminders/remn14bf0e77/mac)
- [List and column views](https://support.apple.com/en-gb/guide/reminders/remn1d887139/mac)
- [Sort semantics](https://support.apple.com/en-mide/guide/reminders/remn922d0b42/mac)
- [Subtask semantics](https://support.apple.com/en-gb/guide/reminders/remn32a9622b/mac)
- [Tags and tag management](https://support.apple.com/en-gb/guide/reminders/remn45640f4f/mac)
- [Templates](https://support.apple.com/en-euro/guide/reminders/remn29caf6e1/mac)
- [Notifications and notification actions](https://support.apple.com/en-gb/guide/reminders/remn4e53b572/mac)
- [iOS 26 feature summary: suggestions, categorization, quick capture, and time zones](https://www.apple.com/os/pdf/All_New_Features_iOS_26_Sept_2025.pdf)
- [iOS 26.2 urgent reminder alarms](https://support.apple.com/guide/iphone/whats-new-in-ios-26-iphfed2c4091/ios)

### Urbit

- [Gall userspace overview](https://docs.urbit.org/build-on-urbit/userspace)
- [Gall agent structure and state lifecycle](https://docs.urbit.org/build-on-urbit/app-school/2-agent)
- [Gall API: pokes, facts, watches, kicks, and scries](https://docs.urbit.org/urbit-os/kernel/gall/gall-api)
- [Gall subscriptions and reconnect considerations](https://docs.urbit.org/build-on-urbit/app-school/8-subscriptions)
- [Marks and JSON conversion](https://docs.urbit.org/urbit-os/kernel/clay/marks)
- [Eyre external authentication and channel API](https://docs.urbit.org/urbit-os/kernel/eyre/external-api-ref)
- [Ames network guarantees and encryption](https://docs.urbit.org/urbit-os/kernel/ames)
- [Urbit cryptography and end-to-end transport](https://docs.urbit.org/urbit-os/kernel/arvo/cryptography)
- [Urbit ID and decentralized PKI](https://docs.urbit.org/urbit-id/what-is-urbit-id)
- [Behn timers](https://docs.urbit.org/build-on-urbit/app-school/9-vanes)
- [Lick IPC, an alternative future local transport](https://docs.urbit.org/urbit-os/kernel/lick)
- [Software distribution](https://docs.urbit.org/build-on-urbit/userspace/dist)

### Omarchy

- [Omarchy shell plugin architecture and manifest](https://github.com/basecamp/omarchy/blob/quattro/shell/README.md)
- [First-party plugin examples](https://github.com/basecamp/omarchy/blob/quattro/shell/plugins/README.md)
- [Shell development and IPC contract](https://github.com/basecamp/omarchy/blob/quattro/agents/skills/shell-dev.md)
