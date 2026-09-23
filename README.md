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

Early implementation. There is no production runner, installer, Docker
integration, or supported real-ship management yet.

- [Architecture](docs/architecture.md)
- [Implementation plan and validation gates](docs/implementation-plan.md)
- [Runtime qualification checklist](docs/runtime-qualification.md)

The first supported platform will be x86-64 Linux with systemd and Docker.
Laptops are supported with an acknowledgement that sleeping hosts make their
ships unavailable. ARM64 and non-systemd NAS appliances are not initially
qualified.
