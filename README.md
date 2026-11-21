# Omabit

**Run your own Urbit ships from Omarchy, and build multiplayer apps on top of them.**

Omabit brings self-hosted Urbit to the Omarchy desktop: set up a host, connect
local or remote ships, start and stop them, inspect their health, and manage
maintenance from a fleet panel. When you need help, **Agent Help opens your
default Omarchy coding agent with the relevant Omabit instructions and context**.

The current stable release is [0.5.0](https://github.com/thelifeandtimes/omabit/releases/tag/v0.5.0).
See its [features, upgrade path and tested limits](docs/release-0.5.0.md).
This development branch is preparing per-ship runtime selection for
[issue #22](https://github.com/thelifeandtimes/omabit/issues/22); that feature is
not included in 0.5.0. The [review and migration guide](docs/runtime-profiles-review.md)
tracks the remaining qualification gates; do not install unqualified development
source on an existing live fleet.

## What is Urbit?

[Urbit](https://urbit.org/) is an open-source personal server and peer-to-peer
network. Your running server is called a **ship**. It has an identity, runs apps,
and keeps their state in a directory called a **pier**. **Vere** is the program
that runs it; **Arvo** is the operating system inside it. You can run Vere on a
Linux computer while using ordinary desktop or web interfaces to your apps.

Urbit combines persistent application state with a cryptographic identity and
[encrypted ship-to-ship messaging](https://docs.urbit.org/urbit-os/kernel/ames).
A durable Urbit ID is controlled through your keys, giving applications a
self-sovereign identity layer that can move with you between hosts. Keeping your
keys and pier backups safe is part of self-hosting. A **comet** is a free,
ephemeral identity for trying the network; a **fake ship** is for isolated local
development and cannot participate in the public network.

For an Omarchy user, this means a desktop plugin can have a persistent backend
on your own ship and exchange messages with other people's ships. Think shared
reminders, chat, collaborative tools or multiplayer games: local interfaces,
user-owned state, and peers identified by their ship names. You can build the
frontend with familiar tools and use Urbit for identity, storage and messaging.
Omabit supplies hosting and development workflows; each app still defines its
own behavior and permissions.

## Get started

Omabit's published hosting builds target **Linux x86-64** with systemd and
cgroup v2. The guided setup checks host dependencies and prepares a separate
rootless Docker engine under the pier-owning user. A desktop managing only a
remote runner does not need a local Docker engine.

On an Omarchy desktop with native plugin support:

```sh
omarchy plugin install https://github.com/thelifeandtimes/omabit.git --enable
```

The plugin verifies and downloads its pinned prebuilt tools. You do not need a
Go compiler. Open **~+ → Setup → Set up this computer** and follow the checklist.
Host installation requests OS authorization. Once ready, choose **Add a ship →
Boot Ephemeral Identity** to try a comet. Use the owned-identity flow if you
already have an Urbit ID, or the stopped-pier import flow for an existing ship.
**Never run two copies of the same live ship.** Importing an existing pier and
booting a new identity are different operations.

For another host, choose **Setup → Manage runners → Connect another runner**.
Use your existing SSH alias or host; setup provisions a separate restricted
management key. Connecting an existing runner preserves its ships and services.
See [installation](docs/install.md), [remote setup](docs/remote-runner-setup.md),
and [storage and imports](docs/storage-management.md).

For an existing stable installation:

```sh
omarchy plugin update thelifeandtimes.omabit
```

Let the verified desktop bundle load, then review any separate **service update
in Setup**. Updating the desktop alone does not update a local or remote runner.
The tested direct 0.5.0 upgrade is from 0.4.0 on Linux/AMD64; older installations
follow their documented intermediate upgrades. Existing enrollments, piers,
settings, history and running/stopped intent must be preserved. Do not reset or
re-create ships to perform an update. See [upgrade guidance](docs/guided-service-upgrades.md)
and the [0.5.0 qualification record](docs/releases/0.5.0/qualification.json).

## Ask your agent for help

Use **Agent Help** in the monitor, or **Setup → Agent Help and skills**. Omabit
launches the default agent you selected in **Omarchy → Setup → Defaults → Agent**
through Omarchy's agent commands. This is your usual `$AGENT` workflow: ask it to
explain a failure, inspect the selected runner, or help build an app. Omabit sends
the matching skill path and bounded context; your agent retains its own sign-in
and execution controls. A help request does not itself authorize a repair.

For example: “Explain why this ship is stopped and show me the supported recovery
steps,” or “Help me build a shared reminders app using a fresh fake project.”
No global skill installation is needed for the button to work. Setup also offers
explicit skill installation while preserving custom or edited copies.

The release includes three complete skill directories, each with a `SKILL.md`
and supporting references:

| Skill | Use it for |
| --- | --- |
| [omabit-operator](skills/omabit-operator/SKILL.md) | Hosting, fleet diagnosis, lifecycle, imports and maintenance |
| [omabit-app-developer](skills/omabit-app-developer/SKILL.md) | Urbit application backends using Gall or Grubbery |
| [omabit-plugin-developer](skills/omabit-plugin-developer/SKILL.md) | Omarchy interfaces and connections to a ship |

They are also available in the verified `omabit-skills-VERSION.tar.gz` release
asset. Keep each skill's references with it, and use guidance matching your
installed tools. See [Agent Help and skill distribution](docs/agent-skills.md).

## How the pieces fit together

| Component | What it does |
| --- | --- |
| **Omabit** | The Omarchy panel, setup and monitoring UI, plus the `omabit` command and agent-facing interfaces |
| **urbit-collector** | Your desktop's connections to runners, identity checks, credentials and access to ships |
| **urbit-runner** | The service on each hosting machine that owns durable ship operations, storage, runtime containers and maintenance |
| **Vere / Urbit** | The per-ship runtime and operating system that actually run your apps and retain their state in the pier |

The desktop asks the collector to contact a selected runner. The runner manages
its ships even after the panel or terminal closes. A runner can be on your
Omarchy machine or another Linux host. The three Omabit components share this
repository and ship as matching artifacts.

Applications can authenticate to a ship through
[Application Connections](docs/application-connections.md), then communicate
with its HTTP interface, **Eyre**, directly. Omabit manages hosting and access;
ordinary application traffic does not have to pass through its management service.
See the [architecture](docs/architecture.md) and [backend entry points](backend/README.md).

## CLI and MCP

The desktop is one interface. The `omabit`, `urbit-collector` and `urbit-runner`
commands also have [offline help and CLI references](docs/cli/README.md).
`omabit operations` lists the shared typed operation catalog, including explicit
exclusions. `omabit call` and the MCP server use the same handlers for supported
fleet, project, recovery and application operations.

For an already enrolled runner, a local MCP client can launch:

```sh
omabit --config-dir /absolute/path/to/your/registry mcp serve --transport stdio \
  --allow-runner NAME:RUNNER_ID
```

Replace the path, enrollment name and expected runner ID with your selected
profile's values. The default exposes read tools; mutations require an explicit
server option and still respect runner role and saved application permissions.
See [the shared CLI/MCP contract](docs/core-interfaces.md) and
[client connection recipes](docs/mcp-clients.md). The tested 0.5.0 support is the
CLI and pinned-SDK stdio contract; named-client recipes and local HTTP remain
experimental. The catalog is not yet a replacement for every legacy CLI command.

## Build your own apps

Start with the [application developer skill](skills/omabit-app-developer/SKILL.md).
**Gall** is Urbit's application framework, normally using Hoon agents.
**Grubbery** is an alternative application framework with its own source and
runtime workflow. Choose from the code you are building; an Omarchy frontend is
optional for either backend.

1. Select a development runner and run `omabit app-preflight` for the intended
   framework. This checks the actual runner's capabilities and permissions before
   creating an application test project.
2. Use a fresh isolated fake project. Follow the maintained
   [Gall sample and workflow](skills/omabit-app-developer/references/gall.md) or
   [versioned Grubbery workflow](skills/omabit-app-developer/references/grubbery.md).
3. Approve only the needed creation/commit permissions in Setup. Stage a bounded
   source tree, review it, commit and verify compilation and application behavior.
4. Test updates and restart with saved state, then add a desktop frontend using
   the plugin developer skill and Application Connections.

0.5.0 qualifies bounded Gall and Grubbery source/recovery/restart workflows.
Fresh Grubbery framework bootstrap remains tracked in
[#21](https://github.com/thelifeandtimes/omabit/issues/21). Grubbery peer discovery,
permits, routes, served assets and broader failure cases also remain separate work.
The [preflight guide](docs/app-development-preflight.md) explains capability and
policy refusals. Testing uses disposable ships, never an existing live identity.

## Explore the ecosystem

| Project | Why look at it |
| --- | --- |
| [Urbit documentation](https://docs.urbit.org/) | Learn the system, self-hosting and application development |
| [urbit/urbit](https://github.com/urbit/urbit) | Urbit OS and Hoon source |
| [urbit/vere](https://github.com/urbit/vere) | The runtime that executes a ship |
| [nisfeb/grubbery](https://github.com/nisfeb/grubbery) | Grubbery's application framework |
| [nisfeb/talon](https://github.com/nisfeb/talon) | A native Urbit chat client |
| [tloncorp/tlon-apps](https://github.com/tloncorp/tlon-apps) | Tlon Messenger's clients and server, from Tlon Corporation |
| [thelifeandtimes/omabit](https://github.com/thelifeandtimes/omabit) | Hosting, desktop integration and development workflows in this repository |
| [thelifeandtimes/tend](https://github.com/thelifeandtimes/tend) | An Urbit reminders application with its own CLI and release lifecycle |

## Contribute and find details

Use [CONTRIBUTING.md](CONTRIBUTING.md) for source builds and checks. `master` is
the protected stable-release branch; `edge` is an explicit prerelease choice;
`develop` is unqualified development. Follow the
[channel update instructions](docs/development-policy.md#edgedevelop-installation-and-updates)
for edge/develop: the stock Omarchy updater fetches the default branch.

Further reading: [release installation](docs/release-install.md),
[services and reboot behavior](docs/services.md), [runtime policy](docs/runtime-policy.md),
[recordings and their limits](docs/runtime-recordings.md),
[development profiles](docs/desktop-development.md), and
[non-destructive upgrades](docs/upgrade-policy.md). Historical milestone notes
remain under `docs/`; the current release's qualification record defines its
actual support window.
