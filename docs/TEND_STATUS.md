# Tend implementation status

Last verified: 2026-09-10

## Working now

- `%tend` installs and hot-reloads on a live fake ship.
- Existing `%0` state migrates to the richer `%1` schema without losing tasks.
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
- Plugin validation, JavaScript reducer tests, and Python Eyre transport tests
  pass.

## Next

1. Scheduling, time zones, early alerts, recurrence, and durable Behn timers.
2. Built-in views, search/sort, pins/default-list preferences, list/section
   editing, and hierarchy/order controls in the overlay.
3. Owner-hosted list references, invitations/ACLs, replicas, presence, and
   read-only owner-offline behavior across multiple ships.
4. Notifications, CLI, packaging, backups, accessibility, fault testing, and
   release hardening.

The source of truth for scope and exit criteria remains
[`PRODUCT_ARCHITECTURE_PLAN.md`](../PRODUCT_ARCHITECTURE_PLAN.md).
