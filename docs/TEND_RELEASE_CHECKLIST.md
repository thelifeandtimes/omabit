# Tend Core release checklist

This checklist reconciles the active implementation with the Core scope in
`PRODUCT_ARCHITECTURE_PLAN.md`. “Verified” means an automated or live-desk gate
exists. “Partial” is usable but still misses a stated Core behavior.

Last reconciled: 2026-09-21

| Area | Status | Evidence / remaining gate |
| --- | --- | --- |
| Lists, sections, reminder CRUD, rich metadata | Verified | Gall actions, Eyre round trips, atomic section ordering, reducer and CLI tests |
| Completion, batch actions, subtasks, stable ranks | Verified | Cascades, hierarchy controls, pointer drag/drop, keyboard placement, atomic rank normalization, and stale rejection are tested/live-verified |
| Dates, all-day policy, time zones, early alerts | Verified | `%tend` validation, Behn wake test, `zoneinfo` gap/fold tests, and live local-wall-time-to-`@da` round trip |
| Recurrence and next occurrence | Verified | Hourly through yearly advancement and boundary rules compile/live-test; individual and atomic batch completions retain one completed due instant per recurring reminder in bounded history |
| Snooze | Verified | Durable presets, explicit future date/time, and notification action are exposed in both desktop and CLI paths |
| Tags, flag, priority, safe URL, search | Verified | Gall validation and reducer/query tests |
| Built-in views | Verified | Today, Scheduled, All, Flagged, Assigned, and Completed query behavior is reducer-tested; grocery categorization remains in the deferred Parity tier |
| Sorting, pins, default list | Verified | All sort modes, ordered list/view pins, default list, participant-local per-list sort/direction, and private unpinned-list order are durable and live-verified |
| Full overlay and quick capture | Verified locally | Keyboard-first full/capture surfaces and `#tag` capture work; capture stacks on narrow outputs and opens on the invoking/focused monitor; the physical multi-monitor pass remains a release gate |
| Bar widget and connection state | Verified | Configurable badge, next reminder, sync state, shared-host state, click-to-open |
| Eyre domain + `+code` authentication | Verified | Redirect/size/UTF-8/cookie hardening and rank-neutral planet/moon/comet-shaped identities |
| Desk/client compatibility boundary | Verified | Snapshot protocol 1 is required at login, direct state reads, and live replacement; mismatches fail before credential persistence or Online state |
| Desktop reconnect and stale-event safety | Verified | Bounded reconnect, auth-expiry stop, canonical snapshot recovery, monotonic list/preference reducers |
| Native notifications | Verified | Complete/Snooze/Open actions plus stable-ID replay-until-desktop-ack pass reducer, transport, and live reconnect tests |
| CLI and JSON output | Verified | Collaboration, mutation, snooze, backup/restore, and open commands work; shared mutations fail closed unless owner status is Online |
| Backup/export | Verified | Owner-only private export, offline validation, atomic empty-state restore, durable receipt verification, and second-restore rejection are tested/live-verified |
| Accessibility | Verified locally | Static labels, keyboard operation, visible state cues, scaled/clamped surfaces, responsive capture, no custom motion, and QML lint; manual Orca/theme/200% scale pass remains |
| Owner host Online/Checking/Offline gating | Verified | Gall, model, UI, and CLI block shared mutation while the owner is not Online; replicas remain readable and the overlay labels any previously submitted edit still in flight |
| Assignment | Verified | Desktop choices and Gall enforcement allow only the owner/current members; authorization has no rank branch |
| Invitations, ACLs, replicas, shared edits | Verified | Repeatable three-ship harness covers convergence, 12-operation Ames pressure, restart catch-up, duplicate/stale edits, in-flight revocation, cleanup, complete selected-list disclosure, and recipient-bound capability URI parsing |
| Activity and collaboration notifications | Verified | Bounded actor-attributed activity and occurrence records converge; participant-local add/complete/assignment policy, suppression, durable replay/ack, and cleanup are live-verified |
| Packaging/install/rollback | Verified | Deterministic archive/checksum plus refusal, full-desk backup, base-mark preservation, mounted-Kelvin preservation, and preflight atomicity tests |
| State migrations | Verified | The application and table-driven Hoon fixture generator call the same migration library; non-empty `%0`–`%10` fixtures preserve sentinel list/reminder data and pass on Vere 4.6 |
| Scale | Verified locally | 10,000-reminder normalization/Today/search and transport ceilings |
| Gall resource retention | Verified | Input and collection ceilings are enforced; the receipt ledger retains the newest 4,096 results |

Calendar integration and Tlon Messenger-specific triggers are intentionally out
of scope. The Parity tier (attachments, location triggers, custom smart lists,
groups, columns, templates, Recently Deleted, natural-language dates, grocery
learning, and urgent alarms) is also deferred under the current directive.

## Release blockers

The first multiplayer release cannot be called complete until:

- a longer real-ship restart/key-continuity soak passes using the lockstep
  deployment policy in `TEND_COMPATIBILITY.md`;
- the manual Omarchy accessibility and multi-monitor checklist passes on a
  supported release candidate.
