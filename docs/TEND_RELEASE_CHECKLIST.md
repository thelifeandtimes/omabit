# Tend Core release checklist

This checklist reconciles the active implementation with the Core scope in
`PRODUCT_ARCHITECTURE_PLAN.md`. “Verified” means an automated or live-desk gate
exists. “Partial” is usable but still misses a stated Core behavior. “Blocked”
means the implementation is deliberately disabled at a trust boundary.

Last reconciled: 2026-09-10

| Area | Status | Evidence / remaining gate |
| --- | --- | --- |
| Lists, sections, reminder CRUD, rich metadata | Verified | Gall actions, Eyre round trips, reducer and CLI tests |
| Completion, batch actions, subtasks, stable ranks | Partial | Cascades, hierarchical collapse/expand, indent/outdent, and up/down rank controls are verified; pointer drag/drop remains |
| Dates, all-day policy, time zones, early alerts | Verified | `%tend` schedule validation, Behn wake test, policy reducer tests |
| Recurrence and next occurrence | Partial | Hourly through yearly advancement and boundary rules compile/live-test; full occurrence history and DST gap/fold property fixtures remain |
| Snooze | Verified | Durable presets, explicit future date/time, and notification action are exposed in both desktop and CLI paths |
| Tags, flag, priority, safe URL, search | Verified | Gall validation and reducer/query tests |
| Built-in views | Partial | Today, Scheduled, All, Flagged, Assigned, Completed verified; grocery behavior is outside the immediate Core build |
| Sorting, pins, default list | Partial | All sort modes, ordered list/view pins, and default list work; per-participant persistence of sort direction and general unpinned list ordering remain |
| Full overlay and quick capture | Partial | Keyboard-first full/capture surfaces and `#tag` capture work; final compact-surface and manual multi-monitor checks remain |
| Bar widget and connection state | Verified | Configurable badge, next reminder, sync state, shared-host state, click-to-open |
| Eyre domain + `+code` authentication | Verified | Redirect/size/UTF-8/cookie hardening and rank-neutral planet/moon/comet-shaped identities |
| Desktop reconnect and stale-event safety | Verified | Bounded reconnect, auth-expiry stop, canonical snapshot recovery, monotonic list/preference reducers |
| Native notifications | Partial | Complete/Snooze/Open actions work; durable replay-until-desktop-ack requires an approved Gall state migration |
| CLI and JSON output | Partial | Connect/status/list/today/add/complete/snooze/open/export/validate work; `share` depends on the peer engine |
| Backup/export | Partial | Private atomic export and offline structural validation verified; atomic empty-state restore remains |
| Accessibility | Partial | Static labels, keyboard operation, visible state cues, no custom motion; manual Orca/theme/200% scale pass remains |
| Owner host Online/Checking/Offline gating | Verified locally | Model and UI block every shared mutation when a non-owner host is not Online |
| Assignment | Verified locally | Desktop choices and Gall enforcement allow only the owner/current members; owner assignment live-tested and authorization has no rank branch |
| Invitations, ACLs, replicas, shared edits, activity | Blocked | Peer nouns/state boundary exist; outbound data transmission awaits explicit approval and three-ship tests |
| Packaging/install/rollback | Verified | Deterministic archive/checksum and refusal/backup installer tests |
| State migrations | Partial | `%0`–`%6` paths compile and were exercised incrementally on a live fake ship; automated fixture-per-version gate remains |
| Scale | Verified locally | 10,000-reminder normalization/Today/search and transport ceilings |
| Gall resource retention | Partial | Input and collection ceilings are enforced; durable operation receipts still need bounded retry-window retention |

Calendar integration and Tlon Messenger-specific triggers are intentionally out
of scope. The Parity tier (attachments, location triggers, custom smart lists,
groups, columns, templates, Recently Deleted, natural-language dates, grocery
learning, and urgent alarms) is also deferred under the current directive.

## Release blockers

The first multiplayer release cannot be called complete until:

- the owner-hosted peer engine is approved, implemented, and passes its
  authorization/revocation/fault matrix on three ships;
- every released Gall schema has an automated migration fixture;
- alerts are replayed until the desktop acknowledges delivery;
- the operation-receipt ledger has a tested retention bound;
- restore into an empty `%tend` state is atomic and tested;
- the manual Omarchy accessibility and multi-monitor checklist passes on a
  supported release candidate.
