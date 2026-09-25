# Tend native mobile client plan

Status: proposed  
Date: 2026-09-25  
Reference implementation inspected: `nisfeb/talon` at commit
`8dccb37424e961a542784232b8e50c5097e0bfb8` (`release: 1.7.11`)

## Executive recommendation

Build the first native Tend mobile client for Android and iOS with Kotlin
Multiplatform and Compose Multiplatform. Share the protocol, sync engine,
database, repositories, application state, and almost all screens. Keep small
native application shells for lifecycle, secure credentials, notifications,
background execution, deep links, biometrics, and other operating-system
integrations.

This is the strongest fit for Tend because:

- Talon demonstrates that an Urbit-backed application can keep a substantial
  Compose UI and Urbit transport in `commonMain` while using native Android and
  iOS shells.
- Tend is a bounded product with unusually high shared-UI potential. Its list,
  reminder, filter, sharing, and settings surfaces are structurally the same on
  both mobile platforms.
- The current `%tend` protocol already has the important foundations for a
  mobile client: authenticated scries, one UI subscription, versioned facts,
  idempotent operations, bounded receipts, host revisions, durable alerts, and
  explicit host availability.
- A shared Room database and deterministic reducer provide offline reading and
  fast startup without making the phone a second authority.

The recommended first beta deliberately excludes a hosted push relay. It can
provide live updates while foregrounded, local due notifications, cached
offline reading, and best-effort reconciliation in platform background time.
True real-time collaboration notifications while iOS has suspended the app
should be a separate, explicit product and privacy decision.

## What to take from Talon

Talon is a premier reference because it is not a thin web wrapper. Its current
repository is organized around a Kotlin Multiplatform core, shared Compose UI,
Room-backed local state, Ktor networking, a SwiftUI iOS shell, and platform
implementations for capabilities that cannot be portable.

Patterns to adopt:

1. **Common by default.** Models, serializers, Urbit authentication, channel
   lifecycle, reducers, repositories, navigation state, and screens belong in
   shared Kotlin until a platform requirement proves otherwise.
2. **Narrow platform contracts.** Define interfaces such as
   `CredentialStore`, `NotificationScheduler`, `BackgroundRefresh`,
   `BiometricLock`, and `ExternalNavigator` in common code. Implement them on
   Android and iOS; use an explicit no-op only for genuinely optional
   capabilities.
3. **A native shell, not two native applications.** Android hosts Compose
   directly. iOS uses a small SwiftUI application that embeds the generated
   Compose framework and owns Apple lifecycle delegates and entitlements.
4. **One transactional local model.** Incoming snapshots and facts reduce into
   Room transactions, then UI observes database-backed flows. Screens do not
   assemble their own competing network caches.
5. **Capability visibility.** Unsupported behavior is disabled or explained,
   never simulated. Background and notification constraints should be visible
   in settings and diagnostics.
6. **Cross-platform CI from the beginning.** Run common/JVM tests on Linux and
   compile the iOS framework on macOS on every relevant change. Add device and
   simulator workflows before beta distribution.

Patterns not to copy blindly:

- Tend does not need Talon's call stack, WebRTC services, or relay complexity.
- Talon's repository covers desktop targets too. Tend already has desktop and
  web interfaces, so mobile v1 should not dilute its scope with another desktop
  client.
- Secure session storage must be a hard requirement. The target design stores
  session material in the iOS Keychain and in Android Keystore-backed encrypted
  storage, regardless of simpler persistence choices in a reference project.
- Tend should keep Gall authoritative and honor its current offline-write
  constraints. A generic optimistic offline queue would violate the sharing
  model rather than improve it.

## Product scope

### Beta must support

- Sign in to a ship by base URL and `+code`, then retain the resulting Eyre
  session securely.
- One active ship, with the storage and navigation model prepared for multiple
  saved ships later.
- Today, All, Flagged, Assigned, Scheduled, and Completed views.
- Hosted and shared lists, sections, arbitrary reminder nesting, tags,
  priorities, assignment, due dates, all-day schedules, recurrence, completion,
  flagging, and cross-list moves supported by the current agent.
- Invitations, membership display, host status, and permission-aware editing.
- Per-list presentation and collaboration-notification settings.
- Offline startup and read-only browsing of the last committed local replica.
- Immediate edits only while the applicable host is Online, matching the
  existing Tend contract.
- Local due notifications with Open, Complete, and Snooze actions where the
  platform permits them.
- Deep links to a list, reminder, and invitation.
- Adaptive phone and tablet layouts, dark and light appearance, Dynamic Type,
  screen-reader labels, reduced motion, and adequate touch targets.

### Defer until after beta

- A vendor-operated mandatory account, directory, analytics service, or
  notification gateway.
- General offline mutation queues or CRDT conflict resolution.
- Desktop targets from the mobile codebase.
- Home-screen widgets, watch applications, Siri/App Intents, Android shortcuts,
  and share-sheet quick capture. The architecture should leave clear platform
  extension points for them.
- Location reminders, urgent full-screen alarms, attachments, and features
  already deferred by the Tend product plan.

## Proposed repository shape

The Tend workstream is currently rooted in transitional paths. Do not move it
piecemeal. When the workstream migration is approved, place mobile beside the
other Tend clients atomically. A likely final shape is:

```text
apps/tend/
  desk/                         # Gall agent and marks
  clients/
    web/
    omarchy-plugin/
    mobile/
      settings.gradle.kts
      build.gradle.kts
      gradle/libs.versions.toml
      core/                     # Pure protocol and sync primitives
        src/commonMain/
        src/commonTest/
      composeApp/               # Shared DB, repositories, UI, resources
        src/commonMain/
        src/androidMain/
        src/iosMain/
      androidApp/               # Android entry point and manifest
      iosApp/                   # Xcode project and SwiftUI shell
      test-support/             # Fake Eyre/SSE server and fixtures
```

Until that migration happens, prototype mobile in a separate worktree without
creating a runtime dependency from `%tend` to another Omabit workstream.

## Architecture

```text
┌──────────────── Android shell ───────────────┐   ┌────────────── iOS shell ──────────────┐
│ lifecycle · intents · notifications · keys  │   │ SwiftUI · app delegates · Keychain    │
└──────────────────────┬───────────────────────┘   └────────────────────┬───────────────────┘
                       └──────────────────┬─────────────────────────────┘
                                          │ platform capability interfaces
┌─────────────────────────────────────────▼──────────────────────────────────────────────────┐
│ Shared Compose application                                                                │
│ navigation · screens · adaptive layouts · view models · resources                         │
├─────────────────────────────────────────────────────────────────────────────────────────────┤
│ Repositories and sync                                                                      │
│ Room flows · transactional reducer · pending-operation ledger · alert coordinator          │
├─────────────────────────────────────────────────────────────────────────────────────────────┤
│ Tend core                                                                                  │
│ protocol-2 serializers · auth · scry · poke · SSE channel · revisions · recurrence helpers │
└─────────────────────────────────────────┬──────────────────────────────────────────────────┘
                                          │ HTTPS / Eyre
                                 ┌────────▼────────┐
                                 │ ship / %tend   │
                                 │ Gall authority │
                                 └─────────────────┘
```

### `core`

`core` should have no Compose or platform UI dependency. Its public surface is
small and testable:

- `TendSession`: performs `/~/login`, validates the response, retains the
  `urbauth-~...` cookie through `CredentialStore`, and supports logout.
- `TendHttp`: authenticated GET and scry helpers with size limits, timeouts,
  redacted logs, and protocol-specific error types.
- `TendChannel`: creates and resumes the Eyre channel, subscribes to `%tend`
  `/all`, parses SSE events, sends event acknowledgements, detects server kicks,
  uses bounded exponential backoff with jitter, and distinguishes authentication
  failure from transient connectivity.
- `TendActions`: typed builders for `%tend-action-1` pokes. Every mutation owns
  an operation ID; applicable mutations include the known base revision.
- `TendFacts`: strict serializers for `%tend-update-1`, including snapshots,
  list upserts/deletes, access state, invitations, settings, activity, alerts,
  and operation state.
- `TendSnapshot`: scries `/state`, `/accesses`, `/invitations`, `/settings`,
  `/activities/<id>`, `/receipt/<operation-id>`, and `/whoami` with the current
  32 MiB safety ceiling.
- `ProtocolCompatibility`: fails closed on an unsupported protocol version and
  produces an actionable upgrade screen rather than attempting partial writes.

The web reducer's compatibility tolerance is useful for rolling desktop
reloads, but it should not become the mobile compatibility contract. A store
release can remain installed for months; explicit version negotiation and a
documented support window are required before public beta.

### Database and repositories

Use Room for Kotlin Multiplatform with bundled SQLite. Prefer one database per
saved ship so logout, account removal, backup policy, and corruption recovery
have clear boundaries.

Initial tables:

| Table | Purpose |
| --- | --- |
| `account` | Ship, base URL, display metadata, protocol version, last success |
| `sync_state` | Channel identity, snapshot revision/time, backoff, connection state |
| `list` | Canonical and local list IDs, host, title, revision, ownership, status |
| `section` | List section order and revision |
| `reminder` | Hierarchy, content, schedule, recurrence, assignment, state, revision |
| `tag` / `reminder_tag` | Normalized tag catalog and join table |
| `list_access` | Members, roles, pending members, host liveness |
| `invitation` | Local invitation inbox and operation token |
| `preference` | Default list, pins/views, badge policy, all-day policy |
| `list_setting` | Private presentation and collaboration policy |
| `activity` | Bounded local copy of the list activity log |
| `pending_alert` | Durable due/collaboration alerts until local scheduling and Gall ack |
| `pending_operation` | Submitted operations still awaiting durable receipt resolution |

Apply one snapshot or fact in a single database transaction. Persist the event
commit point only after the reducer succeeds. UI reads only database flows; it
does not render a speculative network object graph.

### Sync lifecycle

1. Load the cached database immediately and render it with a clear
   Offline/Checking/Online state.
2. Restore the session. If credentials are absent or rejected, show sign-in
   without deleting the readable cache.
3. Establish the `/all` subscription and consume its snapshot/facts.
4. Reduce facts transactionally and acknowledge the SSE event only after the
   transaction commits.
5. On connection loss, retain the last committed replica and disable writes
   whose host is not Online.
6. On reconnect, resubscribe and reconcile. For submitted operations whose poke
   acknowledgement was ambiguous, use `/receipt/<operation-id>`; never infer
   durable success from an HTTP response alone.
7. If the protocol version is unsupported or a snapshot violates validation,
   preserve the old database, stop writes, and expose diagnostics/export rather
   than partially applying state.

### Write and conflict policy

- `%tend` remains authoritative.
- Personal and hosted-list mutations may be optimistically rendered only after
  the client has captured enough state to revert them deterministically.
- Shared-list mutations are enabled only when the current access state says the
  host is Online and the user's role permits the operation.
- Do not queue a new shared mutation while its host is Checking or Offline.
- Maintain an in-flight ledger only for operations already submitted while
  writes were allowed.
- On a stale entity revision, replace the optimistic entity with the current
  canonical value and offer Copy/Reapply for user-authored text. Do not merge
  silently.
- Cross-list moves follow the existing Tend primitive and expose their pending
  state until the source/destination transaction settles.

## Authentication and security

- Require HTTPS for non-loopback endpoints. Permit HTTP only for an explicit
  local-development mode and label it visibly.
- Exchange `+code` for an Eyre session, then discard the code. Never log or
  retain it.
- Store the session cookie in Keychain on iOS and in Keystore-backed encrypted
  storage on Android. Database rows must never contain the cookie.
- Redact cookies, authorization headers, reminder text, notes, URLs, and ship
  endpoints from release logs. Provide an opt-in diagnostic bundle with a
  review screen.
- Support complete account removal: credentials, database, scheduled
  notifications, background jobs, attachments/cache, and logs.
- Offer an optional biometric/local passcode gate without making biometrics an
  Urbit authentication factor.
- Do not use certificate pinning by default. Tend connects to user-controlled
  ships and reverse proxies, so pin rotation and self-hosted certificates would
  create more failure and lockout risk than protection.
- Make lock-screen notification previews configurable: full title, generic
  “Tend reminder”, or hidden.

## Notifications and background work

Mobile notification delivery has two separate jobs and should not blur them:

1. **Scheduled due reminders.** Once the app has committed a Tend schedule, it
   can create an operating-system local notification. Reconcile scheduled
   notifications whenever a snapshot/fact changes a reminder and whenever the
   app resumes. Keep only a bounded horizon scheduled at once, then refill it.
2. **New collaboration or remote-change alerts.** The phone learns these only
   while connected or during a successful background refresh unless a push
   relay wakes it.

For beta:

- Android uses WorkManager for durable reconciliation work and requests exact
  alarm access only if product testing proves exact wall-clock delivery cannot
  be met with ordinary local notifications. Explain any battery or permission
  tradeoff in-product.
- iOS schedules local notifications for known due dates and uses
  `BGAppRefreshTask` only as best-effort reconciliation. Background refresh is
  not a promise of exact or continuous network execution.
- Notification actions open the app or execute a small validated operation.
  Completion remains disabled if a shared host is not Online. Snooze is
  personal state and can use the existing Gall action.
- Ack a Gall notification only after the mobile client has durably recorded it
  and either scheduled/presented the local notification or applied the user's
  action. Crashes before that point must cause replay.

For a later real-time mode, add an optional, independently deployable,
hint-only push relay:

- The ship registers an opaque device token and minimal wake capability.
- Push payloads contain no reminder title, notes, list name, participant name,
  or operation contents—only an opaque account/event hint.
- The app wakes, authenticates directly to the user's ship, and fetches the
  canonical event.
- Device tokens are revocable and expire. Relay compromise must not grant a
  Tend session or reveal reminder contents.
- Self-hosting and who operates the default relay are explicit product choices.

The current durable notification lifecycle is ship-local and acknowledgement
removes an alert from Gall. Before supporting several simultaneous devices,
define whether delivery is “once per user” or “once per registered device”. If
it is per device, protocol state needs device registration and per-device
delivery/ack cursors; otherwise a desktop acknowledgement can legitimately
prevent a phone from seeing the same alert.

## Mobile user experience

### Navigation

- Phone: top app bar plus bottom destinations for Today, Lists, and Search; a
  compact overflow sheet exposes Flagged, Assigned, Scheduled, Completed,
  invitations, and settings.
- Tablet/foldable: persistent list/view rail, central reminder list, optional
  detail pane—the same information architecture as the current responsive web
  client, adapted to touch and platform insets.
- Detail editing uses one scrollable screen or sheet; date/time and recurrence
  use platform-appropriate pickers while preserving Tend semantics.

### Interaction principles

- Completion and selection must remain visually distinct.
- Hierarchy uses consistent left indentation, disclosure controls, and a path
  back to filtered-out ancestors.
- List host, role, and status are visible where they determine editability.
- Offline is not an error modal. Cached content remains useful; controls explain
  why a write is unavailable.
- Destructive actions require a concise undo period where possible; list and
  account removal require confirmation.
- Long titles, large text, landscape, split-screen, notches, keyboards, and
  system bars are first-class layout states.

### Deep links

Reserve and document:

```text
omabit://tend/list/<host>/<list-id>
omabit://tend/reminder/<host>/<list-id>/<reminder-id>
omabit://tend/invite/<owner>/<percent-encoded-token>
```

Universal/App Links can be added later if there is a stable HTTPS domain. A
deep link never bypasses Gall authorization and never contains list content.

## Protocol work before public beta

The existing protocol is sufficient for a prototype and most MVP behavior, but
four decisions should be closed before a store release:

1. **Compatibility window.** Mobile releases cannot be lockstep with every desk
   update. Define supported client/agent protocol ranges and an upgrade response
   that fails closed only for unsafe operations.
2. **Multi-device notification semantics.** Decide user-level versus device-level
   delivery and acknowledgement, then make the data model explicit.
3. **Efficient reconnect.** A full snapshot is acceptable initially. Before
   large datasets and constrained mobile links, consider an authenticated
   revision cursor/backfill path with a bounded snapshot fallback.
4. **Device identity and revocation.** If push or per-device settings exist, add
   opaque device records, revocation, last-seen metadata, and strict payload
   validation. Device identity must not become authority for list operations.

None of these requires changing the Gall app merely to start the mobile client.
They should be designed and tested before promising long-lived store-client
compatibility or real-time suspended-app notifications.

## Testing strategy

### Shared tests

- Golden JSON fixtures for every action and fact in protocol 2.
- Snapshot/fact reducer tests, including duplicate events, out-of-order
  connection callbacks, unsupported versions, malformed payloads, and rollback.
- Revision conflicts and `/receipt` recovery.
- Recurrence across DST changes, leap days, month ends, all-day boundaries,
  locale calendars, and timezone changes.
- Hierarchy moves, section moves, cross-list moves, subtree deletion, and
  filtered ancestor rendering.
- Database migration, transaction interruption, corruption recovery, and
  account isolation.

### Integration tests

- A deterministic fake Eyre server for login, scry, channel creation, SSE,
  poke ack/nack, reconnect, kick, delay, truncation, and 401 responses.
- Disposable Urbit ships for contract tests against the real desk.
- Network switching: Wi-Fi/cellular, captive failure, offline startup, long
  suspension, process death, and host Online/Checking/Offline transitions.
- Notification reconciliation after edits from another client, timezone
  changes, reboot, app upgrade, permission denial, and notification action use.

### UI and accessibility tests

- Phone, small tablet, large tablet, portrait, landscape, split-screen, and
  keyboard-visible screenshots.
- Dynamic Type/font scaling through the largest supported accessibility size.
- TalkBack and VoiceOver labels, order, rotor/grouping, and modal focus.
- Light/dark/high-contrast palettes, reduced motion, and color-independent
  status communication.

### CI and release gates

- Linux: formatting, lint, common/JVM tests, Android unit tests, database schema
  verification, and debug APK.
- macOS: iOS framework compile, Swift shell compile, shared tests, and simulator
  smoke tests.
- Nightly: live disposable-ship suite and network-chaos scenarios.
- Release: signed Android/iOS builds, migration from the prior beta, cold-start
  budget, large 10,000-reminder dataset, memory baseline, and notification
  delivery matrix.

## Delivery sequence and rough effort

This is a sequencing estimate for one experienced Kotlin Multiplatform engineer
with periodic iOS/Urbit review, not a commitment.

| Phase | Outcome | Estimate |
| --- | --- | ---: |
| 0. Protocol extraction | Shared models, fixtures, compatibility decision record | 1–2 weeks |
| 1. Foundation | KMP project, auth, secure storage, Room, channel, fake Eyre | 2–3 weeks |
| 2. Core Tend | Adaptive navigation, views/lists, reminder CRUD, offline read | 3–4 weeks |
| 3. Collaboration | Sharing, permissions, nesting, recurrence, moves, activity | 2–3 weeks |
| 4. Mobile integration | Local notifications, background refresh, deep links, lock | 2–4 weeks |
| 5. Beta hardening | Accessibility, migrations, chaos tests, store distribution | 2–4 weeks |

A credible private beta is approximately 10–14 engineer-weeks if the first
three phases overlap sensibly and no hosted relay is included. Production push
infrastructure, per-device Gall semantics, operations, privacy review, and
abuse controls add roughly 4–8 weeks plus continuing operational ownership.

## Milestone acceptance criteria

### Foundation alpha

- Android and iOS sign in, restore a secure session, subscribe, render a cached
  snapshot, recover from a dropped channel, and pass identical reducer fixtures.

### Internal alpha

- All core list/reminder operations work against hosted and shared lists.
- Host status and permissions gate every write correctly.
- Offline restart renders the last committed replica without data loss.

### Private beta

- Recurrence, nesting, invitations, assignment, moves, preferences, local due
  notifications, deep links, and notification actions work on both platforms.
- Large-data, migration, accessibility, process-death, and network-chaos gates
  pass.
- Unsupported agent versions result in a safe, useful upgrade screen.

### Public release candidate

- Multi-device notification semantics are finalized.
- Privacy policy and diagnostic/logging behavior match the implementation.
- Account removal is complete and verified.
- Store builds, crash reporting policy, support path, and protocol support
  window are documented.

## Decisions needed from the product owner

1. Is v1 Android and iOS only? Recommendation: yes; do not add desktop KMP
   targets while existing Tend desktop clients are active.
2. Should v1 expose one active ship or multiple saved ships? Recommendation: one
   active ship in the UI, but isolate storage so multi-ship can be added without
   migration pain.
3. Are best-effort background refresh and scheduled local due notifications
   sufficient for beta, or is real-time collaboration while suspended a launch
   requirement? The latter implies a push relay and protocol work.
4. If a relay is wanted, who operates it, must it be self-hostable, and are
   Apple/Google push services acceptable as transport for opaque wake hints?
5. Is alert delivery once per Urbit user or once per registered device?
6. Should offline mode remain read-only for all canonical writes, or should a
   later phase permit carefully scoped personal/hosted mutations while offline?
   Recommendation: read-only in v1, especially for shared lists.
7. What minimum versions should the first beta support? Recommendation: iOS 17+
   and Android 10/API 29+ unless the target audience requires older devices.
8. Which distribution channels are required: TestFlight/App Store, Play Store,
   direct APK, and/or F-Droid? These affect signing, update, telemetry, and push
   decisions.
9. Should onboarding remain URL + `+code`, or integrate with the Omabit host
   manager and local ship discovery when available?
10. Is optional biometric app locking required for beta, and what should appear
    in lock-screen notification previews by default?
11. Should the mobile visual language be a native interpretation of the current
    Tend/Omart-inspired design, or pursue strict pixel parity with the web app?
12. Are widgets, share-sheet capture, Siri/App Intents, Android shortcuts, or a
    watch client beta requirements, or explicitly post-v1?

## References

- [Talon repository](https://github.com/nisfeb/talon)
- [Urbit Gall overview](https://docs.urbit.org/build-on-urbit/app-school/intro)
- [Urbit Eyre guide](https://docs.urbit.org/build-on-urbit/runtime/eyre)
- [Kotlin Multiplatform and Compose Multiplatform](https://kotlinlang.org/docs/multiplatform.html)
- [Room for Kotlin Multiplatform](https://developer.android.com/kotlin/multiplatform/room)
- [Android WorkManager](https://developer.android.com/develop/background-work/background-tasks/persistent/getting-started)
- [Android exact alarm guidance](https://developer.android.com/develop/background-work/services/alarms/schedule)
- [Apple local notifications](https://developer.apple.com/documentation/usernotifications/scheduling-and-handling-local-notifications)
- [Apple background app refresh](https://developer.apple.com/documentation/backgroundtasks/bgapprefreshtask)
- [Apple Keychain Services](https://developer.apple.com/documentation/security/keychain-services)
- [Android Keystore](https://developer.android.com/privacy-and-security/keystore)

