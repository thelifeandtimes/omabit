# Omabit

Omabit is an Omarchy + Urbit monorepo for self-hosting, identity, discovery,
communication, and multiplayer desktop applications. Urbit supplies durable
personal state and a user-owned network identity; Omarchy supplies the native
Linux desktop experience.

## Workstreams

| Workstream | Purpose | Current location |
| --- | --- | --- |
| Urbit host manager | Run and manage user-owned Urbit ships locally or over SSH/Tailscale | [`apps/urbit-host`](apps/urbit-host) |
| Tend | Apple Reminders-class multiplayer task application backed by `%tend` | Transitional root layout: [`desk`](desk), [`omarchy-plugin`](omarchy-plugin), and [`tests`](tests) |
| Pals discovery | Help Omarchy users discover other participating users through a `%pals` integration | [`apps/pals`](apps/pals) |
| Tlon Messenger | Provide an Omarchy-native client for Tlon Messenger | [`apps/tlon-messenger`](apps/tlon-messenger) |

The Tend implementation remains at the repository root temporarily because an
active development task is using those paths. It should move as one coordinated
change after that work is stable; do not split its desk, plugin, and tests across
old and new locations.

See [`docs/WORKSTREAMS.md`](docs/WORKSTREAMS.md) for product boundaries and
[`docs/REPOSITORY.md`](docs/REPOSITORY.md) for the target layout and worktree
workflow. The existing [`PRODUCT_ARCHITECTURE_PLAN.md`](PRODUCT_ARCHITECTURE_PLAN.md)
is specifically the Tend product and architecture plan. Current protocol and
implementation details live in [`docs/TEND_PROTOCOL.md`](docs/TEND_PROTOCOL.md)
and [`docs/TEND_STATUS.md`](docs/TEND_STATUS.md).

## Development

The compatibility command for the current Tend task remains unchanged:

```bash
make check
```

Monorepo commands:

```bash
make check-tend
make check-tend-release
make check-urbit-host
make check-all
make dist-tend
make build-urbit-host
make urbit-host-version
```

The host manager is an independent Go program. Build and run it directly with:

```bash
make -C apps/urbit-host build
./apps/urbit-host/urbitctl version
```

Read [`apps/urbit-host/AGENTS.md`](apps/urbit-host/AGENTS.md) before changing or
operating the host manager. Its live Docker, SSH, service-installation, and ship
lifecycle steps have stricter authorization requirements than repository tests.

Tend also exposes a dependency-free umbrella CLI:

```bash
bin/omabit tend status
bin/omabit tend today
bin/omabit tend add "Buy milk" --list Inbox --tag groceries
bin/omabit --json tend list
```

See [`docs/TEND_INSTALL.md`](docs/TEND_INSTALL.md) for desk, plugin, CLI, and
release installation instructions.

## Repository State

The canonical checkout is named `omabit`, uses `main` as its integration branch,
and is ready for checkpoint commits. Keep one active coding task per checkout;
use Git worktrees for parallel workstreams after the baseline commit.
