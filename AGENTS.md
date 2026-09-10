# Omabit contributor instructions

## Begin here

Read `README.md`, `docs/REPOSITORY.md`, and `docs/WORKSTREAMS.md` before changing
repository-wide structure. Read the nearest workstream documentation before
changing an application. More-specific `AGENTS.md` files take precedence within
their directories.

## Workstream boundaries

Omabit is a monorepo with four primary workstreams: the Urbit host manager,
Tend, Pals discovery, and the Tlon Messenger client. Keep each independently
buildable and testable. Do not create implicit runtime dependencies between
workstreams when a documented interface will do.

The Tend implementation is temporarily rooted at `desk/`, `omarchy-plugin/`,
`tests/`, and `PRODUCT_ARCHITECTURE_PLAN.md`. Another task may be using those
paths. Do not move, rename, or broadly reformat them without explicit
coordination. When Tend moves, move the complete workstream atomically.

The Urbit host manager lives at `apps/urbit-host/` and has its own contributor
instructions. Read `apps/urbit-host/AGENTS.md` and the documents it names before
changing it. Repository edit/test permission does not authorize Docker, SSH,
systemd, live ship, or machine-configuration operations.

`apps/pals/` and `apps/tlon-messenger/` are scoped placeholders until their
product and protocol specifications are approved. Do not invent a production
protocol in either directory as incidental work for another application.

## Concurrent work

Do not run concurrent coding tasks in the same checkout. After the repository
has an initial commit, use a separate Git worktree and branch for each active
workstream. Keep the primary checkout for integration and repository-wide
changes. Check active task context before making cross-workstream moves.

The canonical repository directory is `omabit`. Do not remove any temporary
`scratch-omabit` compatibility symlink while a task or terminal still depends on
the old path. Register new Codex projects and worktrees against the canonical
path.

## Validation

Use the narrowest applicable command first:

```sh
make check-tend
make check-urbit-host
make check-all
```

The root `make check` remains an alias for `make check-tend` until the active Tend
work is migrated into its final application directory.
