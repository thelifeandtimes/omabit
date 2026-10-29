# Reusable desktop-session recovery

Status: M335 qualifies the reusable core across three sessions with actual guards.
M336 qualifies immutable packaging; M337 qualifies three actual private native sessions with unchanged packages.
The qualified versions remain unchanged. This document narrows the next work to a
reusable restart path; it does not expand Omabit into an application data proxy,
plugin marketplace or wallet implementation.

## Problem and intended behavior

The existing experiments prove two successive recoveries, but their new authority
readers consume a particular predecessor checkpoint version. That is useful migration
evidence, not a viable per-restart interface. A user must be able to close and reopen
the desktop repeatedly using the same installed generation and recovery formats.

Introduce one reusable family only after its contract is tested. It must read the
retained legacy checkpoints explicitly, then use the same current checkpoint,
intent and authority versions for every further session. Old writers, immutable
packages, claims, mirrors and canonical history stay intact. A new package is needed
when supported software contracts change, not because another desktop was started.

## First implementation boundary

Support an already committed, bound collection whose original compositor and all
owned processes are observably gone, with explicit verified plugin transport cleanup.
Preserve the same profile, plugin IDs, paths, endpoint references, application state,
sessions and unresolved writes. Compatible immutable artifact hashes may advance.
Plugins continue to contact Eyre directly; Omabit assists authentication only.

The common ancestry reader must:

- Walk direct checkpoint references iteratively, with exact inode/device/hash and
  coordinator/lock checks. Never open ancestor runtime directories or acquire a
  second canonical lease while checking guard admission.
- Validate each node with its declared version's validator; reject unknown versions,
  repeated nodes/cycles, a different coordinator, changed cleanup/cohort links,
  missing immutable evidence and nonmatching canonical journal prefixes.
- Enforce explicit aggregate byte, file and node limits before unbounded allocation
  or traversal. A limit refusal must preserve evidence and explain the limit.
- Keep checkpoint bodies out of descendant checkpoint payloads. Reference ancestors
  once; do not grow nested JSON or copy an ancestor's runtime into a new authority.
- Treat the canonical journal as an atomic mutable snapshot, while pinning immutable
  evidence. Legitimate appends must not kill admitted guards.

Cleanup, checkpoint publication, intent claim and authority publication remain distinct
operations. Inspection never authorizes any of them. Exact retries preserve existing
objects, bindings and receipts. A lost reply cannot create another application set.

## Acceptance before public profile integration

1. Import retained legacy checkpoint1/2 and a cleaned authority3 into the reusable
   path, without forcing users through artificial intermediate software generations.
2. Exercise at least three subsequent sessions with identical protocol versions and
   the same package hashes. Confirm the ancestry remains reference-based and bounded.
3. Cover absent/changed ancestors, repeated references, unknown versions, limits,
   canonical appends, conflicting ownership and interrupted publication. Retain old
   reader refusal and exact old-writer reproduction where formats change.
4. Run actual private Theme/DMs with repeated compositor loss: six saved sessions,
   fresh subscriptions, one unresolved DM, direct Eyre transport, owned-process
   closure and unchanged prior artifacts/history.
5. Separately address a claimed but unpublished desktop loss, and pending/unbound
   activation loss. Do not label the committed path as covering these states.
6. Connect supported cases to the public profile flow, then qualify full-shell,
   keyring unlock/reboot, installed updates and real backend behavior. Finish the
   remaining fake-project UI and Docker private-chain roadmap gates independently.

## Limits and decisions still open

The existing journal already has retention limits. A generic chain must give an
honest bounded-capacity refusal; automatic deletion or invented checkpoint compaction
is not part of this slice. Any future compaction needs its own data-preserving contract.
The exact reusable persistent schemas are not finalized by this note. Choose them
once the shared validation boundary is concrete, rather than duplicating a format for
each tested restart. Preserve all older readers/writers and record supersession.

All experiments retain the existing isolation: no managed ~lismyl-lismyl, tmux ships,
~/temp-ur, installed plugins, production credentials, Docker daemon or desktop sockets.
Use separate qualification labels and private namespaces/keyrings/compositors. No
commits, publication or deployment are part of this implementation plan.
