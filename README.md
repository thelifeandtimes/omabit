# Omabit

An opinionated Urbit experience for agent-enabled Omarchy users, built on reusable
host-management and ship-connection components.

This repository is the source monorepo. Independently installable component
repositories and release artifacts will be generated from it, not maintained as
separate codebases.

## Components

| Component | Responsibility |
| --- | --- |
| `urbit-runner` | Per-host lifecycle, metrics, maintenance, storage, and operation attribution |
| `urbit-collector` | Per-user runner connections, collection, credentials, and generic Eyre transport |
| `urbit-operator-gateway` | Optional personal management-only web interface served by the collector |
| Omabit plugin | Omaship UI, Omarchy integration, notifications, theme publishing, and social utilities |
| `%omarchy-gossip` | Separate, tag-restricted anonymous rumor application |
| Tend | Optional plugin and ship application; application details are outside the initial scope |
| Omabit pill | Versioned, opinionated bootstrap release, normally provisioned by URL |

## Status

Checkpoint 1 is ready for manual validation: a memory-only simulated runner and
collector CLI exercise protocol negotiation, shared snapshots, configuration
conflicts, asynchronous operations, request replay, and incidents over a private
Unix socket. There is no production runner, installer, Docker integration, or
supported real-ship management yet.

- [Start here: checkpoint 1 walkthrough](docs/checkpoints/01-local-protocol.md)
- [Architecture](docs/architecture.md)
- [Implementation plan and validation gates](docs/implementation-plan.md)
- [Implemented experimental protocol](docs/protocol.md)
- [Runtime qualification checklist](docs/runtime-qualification.md)

Development requires Go 1.24 or later. There are no third-party Go dependencies.

```bash
go test ./...
go build -o ./build/ ./cmd/...
```

The first supported platform will be x86-64 Linux with systemd and Docker.
Laptops are supported with an acknowledgement that sleeping hosts make their
ships unavailable. ARM64 and non-systemd NAS appliances are not initially
qualified.
