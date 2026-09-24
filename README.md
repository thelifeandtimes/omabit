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

Checkpoint 1's memory-only runner/collector protocol has passed manual validation.
Checkpoint 2A's dedicated rootless engine setup has also passed manual validation.
Checkpoint 2B's pinned-image fake-ship lifecycle, resource checks, container
replacement, offline pack, and cleanup have passed manual validation. Checkpoint
2C's host HTTP publication and replacement/reconnect have also passed. Checkpoint
3A's persistent fake-only lifecycle and runner restart have passed the exercised
live gate. Checkpoint 4A adds persistent collector enrollment, identity-pinned
selection, and independent multi-runner inventory polling; its manual gate is
pending. Production ship management is not supported yet.

- [Next: collector enrollment and inventory](docs/checkpoints/06-collector-enrollment.md)
- [Accepted workflow: persistent development runner](docs/checkpoints/05-persistent-runner.md)
- [Accepted: host-loopback HTTP checkpoint](docs/checkpoints/04-host-http.md)
- [Accepted: disposable runtime checkpoint](docs/checkpoints/03-disposable-runtime.md)
- [Accepted: rootless setup checkpoint](docs/checkpoints/02-rootless-setup.md)
- [Accepted: checkpoint 1 walkthrough](docs/checkpoints/01-local-protocol.md)
- [Architecture](docs/architecture.md)
- [Implementation plan and validation gates](docs/implementation-plan.md)
- [Implemented experimental protocol](docs/protocol.md)
- [Runtime qualification checklist](docs/runtime-qualification.md)

Development requires Go 1.25 or later. The pure-Go SQLite driver is compiled into
the binaries; no additional database service or host SQLite package is required.

```bash
go test ./...
go build -o ./build/ ./cmd/...
./build/urbit-runner preflight
./build/urbit-runner setup-plan
./build/urbit-runner runtime-plan
./build/urbit-runner http-plan
```

The default is dedicated-account rootless Docker, preserving existing system
Docker and desktop access restrictions. Read the checkpoint before running the
development install helper; it is not an updater or production installer.

The first supported platform will be x86-64 Linux with systemd and Docker.
Laptops are supported with an acknowledgement that sleeping hosts make their
ships unavailable. ARM64 and non-systemd NAS appliances are not initially
qualified.
