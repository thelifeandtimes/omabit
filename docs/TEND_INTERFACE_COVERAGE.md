# Tend protocol and interface coverage

Status: pre-release audit, September 24, 2026.

## Roles and permissions

Tend currently has two durable list roles, not a ladder of hidden permission
levels:

| Identity | Authority |
| --- | --- |
| Owner / host | Stores canonical list state; can perform every list and reminder mutation, invite or remove members, delete the list, and publish updates. |
| Member / editor | Can edit list appearance, sections, reminders, schedules, hierarchy and completion while the owner is online. Cannot delete the list, remove another member, or mutate owner-private settings. |

Each member policy also has `can-invite`. This lets that editor invite another
ship, but does not make them an administrator and does not grant removal or
list-deletion authority. Pending invitees can be selected as assignees; they do
not receive list contents or mutation authority until accepting. There is no
read-only/viewer role in desktop protocol version 2.

The host's ship is the decentralized authority for a shared list. A client
replica remains readable when that ship is offline, but editing is intentionally
blocked. Cross-host moves coordinate a durable source reservation, destination
import receipt, finalization and restoration path rather than relying on a
central transaction service.

## Action coverage

The Gall action type remains the authoritative surface. The user interfaces
cover the common interactive workflows; the CLI now covers the core agent
workflow. The remaining gaps are explicit rather than accidental.

| Capability | Web / panel | CLI | Notes |
| --- | --- | --- | --- |
| Read lists, reminders, relationships and access | Yes | Yes | `lists`, `list`, `show`, `access` |
| Create, edit and delete lists | Yes | Yes | New lists receive deterministic-random color and icon in Gall. |
| Create, edit, complete, flag and delete reminders | Yes | Yes | Batch completion/deletion is supported. |
| Parent/child hierarchy and stable placement | Yes | Partial | CLI reads relationships and moves sections/lists; direct parent and sibling placement remain UI-only. |
| Move reminder subtree between lists/hosts | Yes | Yes | `move-list` uses the Gall primitive and checks both hosts. |
| Sections | Yes | Yes | CLI supports list/add/update/delete; drag placement remains UI-only. |
| Due dates, early alerts and recurrence | Yes | Yes | Native, web, and CLI support hourly through yearly recurrence, interval, selected weekdays, month dates or ordinal weekday, end date, and occurrence count. Calendar repeats preserve wall time across DST; hourly repeats use elapsed time. |
| Assignment to owner/member/pending invitee | Yes | Yes | `edit --assignee`; Gall validates list access. |
| Invitations, member removal and leaving | Yes | Yes | Delegated invites use `--can-invite`. |
| Activity and invitations | Yes | Yes | Activity is actor-attributed. |
| Snooze | Yes | Yes | Personal state, not shared list content. |
| Backup, validation and empty restore | No | Yes | Deliberately CLI/operations-only. |
| Personal pins, default list and list order | Yes | No | Deferred CLI surface; protocol actions already exist. |
| Badge/all-day reminder policy | Yes | No | Deferred CLI surface; protocol action already exists. |
| Per-list sort and collaboration notifications | Yes | No | Deferred CLI surface; protocol actions already exist. |
| Global tag rename/merge/delete | Partial | No | Gall `replace-tag` exists; a dedicated management workflow remains to be designed. |
| Notification acknowledgement | Yes | No | Internal UI lifecycle; not normally useful to an agent. |

The agent-facing draft lives at [`skills/tend/SKILL.md`](../skills/tend/SKILL.md).
It uses `omabit --json tend ...` so agents receive stable structured output and
documents the offline-host and destructive-action boundaries.
