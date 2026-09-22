# Omabit Repository Model

## Target layout

```text
omabit/
├── apps/
│   ├── urbit-host/        Independent Go CLI and host service
│   ├── tend/              Gall desk, Omarchy plugin, and Tend tests
│   ├── pals/              Omarchy-user discovery integration
│   └── tlon-messenger/    Omarchy-native Tlon Messenger client
├── packages/              Shared code only after two apps need it
├── docs/                  Cross-workstream architecture and decisions
├── AGENTS.md              Repository coordination rules
└── Makefile               Narrow and aggregate developer commands
```

Only `apps/urbit-host/` is in its final location today. Tend remains at the root
while its active implementation task uses the existing paths. `apps/pals/` and
`apps/tlon-messenger/` currently hold briefs rather than implementations.

## Ownership rules

- Each application owns its build, test, packaging, runtime, and operational
  documentation.
- Root tooling delegates to applications and should not make them one coupled
  executable.
- Shared packages are extracted only when at least two workstreams have a real,
  compatible need. Avoid a speculative framework layer.
- Urbit identities, piers, keys, `+code`, cookies, runtime databases, logs, and
  other private state never belong in this repository.
- Cross-workstream interfaces should be versioned before one app depends on
  another app's internal files or implementation details.

## Worktree workflow

The primary checkout uses `master` as the integration branch. After the baseline
commit, create one worktree per concurrently active workstream:

```bash
mkdir -p ../omabit-worktrees
git worktree add ../omabit-worktrees/tend -b codex/tend master
git worktree add ../omabit-worktrees/urbit-host -b codex/urbit-host master
git worktree add ../omabit-worktrees/pals -b codex/pals master
git worktree add ../omabit-worktrees/tlon-messenger -b codex/tlon-messenger master
```

Use distinct branch names when branches already exist. Do not point two coding
tasks at the same worktree. Repository-wide integration changes should land in
the primary checkout after workstream changes are reviewed.

## Tend migration

Move Tend only when its active task is stopped or explicitly coordinated. The
migration should move these paths together and update references in the same
change:

```text
desk/                         -> apps/tend/desk/
omarchy-plugin/               -> apps/tend/omarchy-plugin/
tests/                        -> apps/tend/tests/
PRODUCT_ARCHITECTURE_PLAN.md  -> apps/tend/docs/PRODUCT_ARCHITECTURE_PLAN.md
```

The root Makefile should then delegate Tend commands into `apps/tend/`, and the
root `make check` can become the aggregate monorepo check.

## Checkout path

The canonical on-disk checkout is `omabit`. A temporary `scratch-omabit`
compatibility symlink may exist while an older Codex task or terminal is still
open. New tasks and saved-project registrations should use the canonical path;
the symlink can be removed after no process depends on it.
