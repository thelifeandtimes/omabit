# Protected live ships and pier handling

The user designated the following identities as **maximally important** on
2026-10-16. Their piers must never be deleted, overwritten, started concurrently,
or otherwise mismanaged during development, import, teardown or fresh-install work:

- `~sarlev`
- `~sarlev-sarsen`
- `~lismyl-lismyl`
- `~tocwex`
- `~pondeg`

Moons under these identities are also important. Carefully protect every ship
whose name begins with any of these prefixes:

- `~dister-`
- `~mister-`
- `~master-`
- `~sarlev-`
- `~hoster-`
- `~sogryp-hoster-`

Examples include `~dister-dozzod-sarlev`, `~master-dozzod-sarlev`,
`~sarlev-dozzod-pondeg`, and `~sogryp-hoster-dozzod-sarlev`. A random/disposable
test moon is not an exception merely because it matches a familiar test pattern;
its exact identity and disposition need explicit review.

## Production import constraint — 2026-10-29

The user is now importing production ships manually. `~lismyl-lismyl`, `~sarlev`,
`~sarlev-sarsen`, `~tocwex` and **all of their moons** are protected, regardless of
name prefixes or whether they look like test identities. Preserve the earlier
`~pondeg` protection too. Do not import any user's ship on their behalf. Diagnose
and qualify import changes with synthetic disposable fixtures only; no production
pier reads, copies, moves, starts, stops or repairs. The tmux and `~/temp-ur`
exclusions remain in force. This supersedes any historical reset/import permission.

## Current platform goal exclusions (2026-10-21)

For this goal, `~lismyl-lismyl` is currently managed by Omabit and must remain
untouched. All user's ships running in tmux and all contents of `~/temp-ur` are
excluded from inspection and operations. The historical relocation/reset note
below is not current authorization. See [experiment isolation](experiment-isolation.md)
for the disposable-runtime and separate-plugin requirements.

## Operational requirements

1. Check this list before any operation that could delete or replace pier data.
   Match canonical ship identities, including aliases/paths without a leading `~`.
   Names and directory labels alone are not sufficient evidence of identity.
2. Inventory enrolled locations, actual filesystem objects, runtime/container
   mounts and processes before live changes. A moved or missing enrolled path is
   not authorization to remove another copy or to bootstrap a replacement.
3. Preserve every known copy of protected piers. Prefer same-parent, no-overwrite
   retirement renames for app resets, with a manifest of the exact paths and a
   before/after process check. Never delete an unclassified pier to simplify reset.
4. Do not manipulate a protected pier's lock, data, runtime or location merely to
   make a check pass. Any required intervention needs a specific reviewed plan.
5. Preserve unrelated native Urbits, shared Docker, user accounts, permissions and
   preexisting lingering. Do not revive archived ships or duplicate live identities.

The user reported that `~lismyl-lismyl` was enrolled on sourcebox but its pier had
been moved out of the enrolled location before the requested next reset. Confirm
that situation from current evidence; do not assume the stale registration locates
all copies. Cleanup of runner-owned management state is permitted after the fixes,
while the protected pier and its relocated data must remain intact.

This document is operational guidance for this user's hosts, not a hardcoded
restriction on the generic Urbit application.
