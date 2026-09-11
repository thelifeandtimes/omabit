# Tend Core release checklist

This checklist reconciles the active implementation with the Core scope in
`PRODUCT_ARCHITECTURE_PLAN.md`. “Verified” means an automated or live-desk gate
exists. “Partial” is usable but still misses a stated Core behavior.

Last reconciled: 2026-09-10

| Area | Status | Evidence / remaining gate |
| --- | --- | --- |
| Lists, sections, reminder CRUD, rich metadata | Verified | Gall actions, Eyre round trips, atomic section ordering, reducer and CLI tests |
| Completion, batch actions, subtasks, stable ranks | Verified | Cascades, hierarchy controls, pointer drag/drop, keyboard placement, atomic rank normalization, and stale rejection are tested/live-verified |
| Dates, all-day policy, time zones, early alerts | Verified | `%tend` validation, Behn wake test, `zoneinfo` gap/fold tests, and live local-wall-time-to-`@da` round trip |
| Recurrence and next occurrence | Partial | Hourly through yearly advancement and boundary rules compile/live-test; full occurrence history remains |
| Snooze | Verified | Durable presets, explicit future date/time, and notification action are exposed in both desktop and CLI paths |
| Tags, flag, priority, safe URL, search | Verified | Gall validation and reducer/query tests |
| Built-in views | Partial | Today, Scheduled, All, Flagged, Assigned, Completed verified; grocery behavior is outside the immediate Core build |
| Sorting, pins, default list | Partial | All sort modes, ordered list/view pins, and default list work; per-participant persistence of sort direction and general unpinned list ordering remain |
| Full overlay and quick capture | Partial | Keyboard-first full/capture surfaces and `#tag` capture work; final compact-surface and manual multi-monitor checks remain |
| Bar widget and connection state | Verified | Configurable badge, next reminder, sync state, shared-host state, click-to-open |
| Eyre domain + `+code` authentication | Verified | Redirect/size/UTF-8/cookie hardening and rank-neutral planet/moon/comet-shaped identities |
| Desktop reconnect and stale-event safety | Verified | Bounded reconnect, auth-expiry stop, canonical snapshot recovery, monotonic list/preference reducers |
| Native notifications | Verified | Complete/Snooze/Open actions plus stable-ID replay-until-desktop-ack pass reducer, transport, and live reconnect tests |
| CLI and JSON output | Verified | Collaboration, mutation, snooze, backup/restore, and open commands work; shared mutations fail closed unless owner status is Online |
| Backup/export | Verified | Owner-only private export, offline validation, atomic empty-state restore, durable receipt verification, and second-restore rejection are tested/live-verified |
| Accessibility | Partial | Static labels, keyboard operation, visible state cues, no custom motion; manual Orca/theme/200% scale pass remains |
| Owner host Online/Checking/Offline gating | Verified | Gall, model, UI, and CLI block shared mutation while the owner is not Online; replicas remain readable |
| Assignment | Verified | Desktop choices and Gall enforcement allow only the owner/current members; authorization has no rank branch |
| Invitations, ACLs, replicas, shared edits | Verified | Live three-ship matrix covers convergence, restart catch-up, duplicate/stale edits, revocation, and leave; complete selected-list disclosure is documented |
| Activity and collaboration notifications | Not implemented | Actor-attributed feed and participant-local add/complete/assignment notification preferences remain Core work |
| Packaging/install/rollback | Verified | Deterministic archive/checksum and refusal/backup installer tests |
| State migrations | Partial | `%0`–`%8` paths compile and were exercised incrementally on a live fake ship; automated fixture-per-version gate remains |
| Scale | Verified locally | 10,000-reminder normalization/Today/search and transport ceilings |
| Gall resource retention | Verified | Input and collection ceilings are enforced; the receipt ledger retains the newest 4,096 results |

Calendar integration and Tlon Messenger-specific triggers are intentionally out
of scope. The Parity tier (attachments, location triggers, custom smart lists,
groups, columns, templates, Recently Deleted, natural-language dates, grocery
learning, and urgent alarms) is also deferred under the current directive.

## Release blockers

The first multiplayer release cannot be called complete until:

- activity history and participant-local add/complete/assignment notifications
  satisfy the remaining Core collaboration contract;
- recurrence stores adequate occurrence history, and per-participant sort/list
  ordering preferences are durable;
- every released Gall schema has an automated migration fixture;
- the multi-ship harness covers backpressure, outstanding-edit removal,
  mixed-version behavior, and a longer restart/key-continuity soak;
- the manual Omarchy accessibility and multi-monitor checklist passes on a
  supported release candidate.
