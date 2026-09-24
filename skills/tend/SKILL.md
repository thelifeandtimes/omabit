---
name: tend
description: Read and manage Tend reminders, lists, schedules, recurrence, sections, assignments, and shared-list invitations through the local Omabit CLI. Use when a user asks an agent to inspect or change their Tend data; do not use for unrelated todo applications.
---

# Tend

Use the dependency-free `omabit` CLI. Prefer machine-readable output:

```bash
omabit --json tend status
omabit --json tend lists
omabit --json tend list
```

Before changing data, read the relevant list and reminder so you have current
IDs and revisions. Lists may be addressed by exact title or local numeric ID.
Reminder and section arguments are numeric IDs.

## Read

```bash
omabit --json tend today
omabit --json tend show "Personal" 42
omabit --json tend access "Shared chores"
omabit --json tend sections "Shared chores"
omabit --json tend invitations
```

`show` includes the reminder's `ancestors` and recursive `descendants`.
`access` identifies the host, connection state, accepted members and pending
invitees. The data model has an owner and editor members; `can-invite` is an
extra member capability, not a separate role.

## Mutate

Use the smallest command matching the request:

```bash
omabit --json tend add "Buy oats" --list Personal --tag groceries
omabit --json tend edit Personal 42 --notes "Get steel cut" --priority high --assignee ~sampel-palnet
omabit --json tend flag Personal 42
omabit --json tend complete 1 42
omabit --json tend move-list Personal 42 "Shared chores"
omabit --json tend section-add "Shared chores" "This week"
omabit --json tend schedule Personal 42 --due 2026-09-25T09:00 --timezone America/Los_Angeles
omabit --json tend schedule Personal 42 --due 2026-09-25T09:00 --timezone America/Los_Angeles --repeat weekly --weekday 5 --count 8
omabit --json tend unschedule Personal 42
```

For an existing reminder, `edit --tag` replaces its whole tag set; repeat the
flag to keep multiple tags. Use `--clear-tags`, `--clear-url`, or `--unassign`
for explicit removal. Cross-list movement moves the selected reminder and its
entire descendant subtree.

For collaboration:

```bash
omabit --json tend share "Shared chores" ~sampel-palnet
omabit --json tend share "Shared chores" ~sampel-palnet --can-invite
omabit --json tend accept 'omabit://tend/invite/...'
omabit --json tend unshare "Shared chores" ~sampel-palnet
omabit --json tend leave "Someone else’s list"
```

Accepted members can edit list metadata, sections and reminders while the host
is online. Pending invitees may be assigned reminders but cannot edit until
they accept. When the host is offline, treat its replica as read-only and
report that condition rather than retrying writes.

## Destructive actions

Do not infer permission to delete. Ask for confirmation immediately before
running a destructive command, then pass `--yes`:

```bash
omabit --json tend delete 1 42 --yes
omabit --json tend section-delete Personal 8 --yes
omabit --json tend list-delete Personal --yes
```

If a mutation fails with a stale revision, read state again and reconsider the
requested change. Do not blindly replay mutations. Never request, print or
store an Urbit `+code`; `connect` reads it interactively or from standard input
only when the user explicitly asks to authenticate.
