# Omarchy Urbit — 0.1.0-pilot

**Source-only development handoff:** start with [AGENTS.md](AGENTS.md) and
[docs/HANDOFF.md](docs/HANDOFF.md). Runtime behavior is unchanged from the pilot;
prebuilt executables are intentionally omitted.

A JSON-first Go CLI and per-user Linux host service for workstation-hosted Urbit
ships, managed locally or from an Omarchy laptop over SSH on Tailscale.

**This is a disposable-ship engineering pilot, not a production release.**
The code compiles and automated tests exercise lifecycle, persistence, Docker
command construction, Unix RPC, permissions, and local TCP proxying. No actual
Docker daemon, Urbit runtime, SSH host, Tailscale network, or Omarchy desktop was
available in the development environment. Those integration checks are the next
step. See [docs/STATUS.md](docs/STATUS.md) before installation.

There is no QML widget yet. This is the shared backend the widget and agents will
use. Hosted-Eyre connections, Anchor, all backups, runtime replacement, and the
cross-host fake-network relay are not implemented in this cut.

## Install and run the disposable pilot

Prerequisites: existing Linux Docker installation accessible to the host user;
SSH keys and verified host keys; both devices already on the same Tailscale
network; user systemd; `jq` for the pilot script. Browser/clipboard actions run on
the client and use `xdg-open` / `wl-copy`. The original pilot archive contains amd64 and arm64
binaries; this development bundle omits them and builds from source with
Go 1.23+ and no third-party Go modules.

On the **workstation**, from the unpacked project:

```sh
./scripts/preflight.sh
./scripts/install.sh --host workstation
```

On the **laptop**, from another copy of the project:

```sh
./scripts/install.sh
# First verify your existing SSH alias; replace this alias when necessary.
ssh workstation 'true'
urbitctl host add --ssh workstation --default workstation
urbitctl --host workstation doctor
./scripts/pilot.sh workstation
```

The installer does not install/configure Docker or Tailscale, change group
membership, open a firewall, enable lingering, install an SSH key, or touch any
existing pier. The workstation service should have user lingering enabled for
operation after logout; inspect and enable that deliberately as described in
[the pilot guide](docs/PILOT.md).

## Interface

Global flags go **before** the command; command flags go **before** positional
arguments. JSON envelopes and schema version 1 are the default.

```sh
urbitctl --host workstation dev group create demo
urbitctl --host workstation --request-id demo-zod-001 dev create \
  --group demo --fake zod --label demo-zod

# Creation returns an operation ID and an instance ID.
urbitctl --host workstation operation wait OPERATION_ID
urbitctl --host workstation ship list
urbitctl ship list --all-hosts
urbitctl --host workstation ship inspect demo-zod
urbitctl --host workstation ship open demo-zod
urbitctl --host workstation ship code --clipboard demo-zod
urbitctl --host workstation ship dojo demo-zod

# Noninteractive Hoon goes through the loopback control endpoint, not the console.
printf '(add 2 2)' | urbitctl --host workstation --agent ship exec \
  --hoon-stdin demo-zod

urbitctl --host workstation ship stop demo-zod
urbitctl --host workstation ship start demo-zod
urbitctl --host workstation ship archive demo-zod
# A distinct, irreversible operation, only after an archive:
urbitctl --host workstation ship purge --confirm FULL_INSTANCE_ID FULL_INSTANCE_ID
```

A lifecycle response means the operation was **accepted**, not finished.
`operation wait` waits for the container action. Runtime and HTTP readiness are
reported separately by `ship inspect`. If SSH disconnects, reconnect and inspect
the operation; do not invent another boot operation.

## Data layout

The runner honors XDG paths. Defaults:

```text
~/.config/omarchy-urbit/
    host.json                   # Authoritative configuration of THIS execution host
    client.json                 # SSH aliases and client default
~/.local/share/omarchy-urbit/
    piers/real/<instance-id>/pier/
    piers/dev/<group-id>/<instance-id>/pier/
    archive/<instance-id>/pier/
    runtime/0.1.0-pilot/         # Versioned, read-only-mounted runtime scripts
    docs/ and skills/
~/.local/state/omarchy-urbit/
    host.json                   # Atomic persistent registry + operation journal
    host.lock
    inventory/<host-alias>.json # Non-authoritative, explicitly staleable cache
$XDG_RUNTIME_DIR/omarchy-urbit/
    owner.sock and agent.sock
    tunnels/                   # Client-side SSH/proxy state
```

`host init --data-root /absolute/path` selects another persistent filesystem.
Use it before installing the host service, or stop the service before manually
editing configuration. Do not move live piers. This pilot uses atomic JSON plus
single-host locking instead of SQLite; the RPC contract hides that choice.

Every instance bind-mounts only its own directory. The plugin's future checkout
will contain no piers. Archive is a rename within the configured data root,
**not a backup**; the archive must remain on that same filesystem. Purge leaves a
metadata tombstone and removes the archived directory. Empty development group
namespace containers and bridges are retained in this pilot.

## Runtime image policy

The default resolver reads GroundSeg's `https://version.groundseg.app` manifest,
selects `latest` (`edge` / `canary` configurable), and pins the architecture's
image digest. There is no silent channel fallback. An explicitly supplied
`--image` override is resolved to its local immutable Docker image ID before use.
Runtime scripts expect `urbit`, Bash, tmux, curl, and flock in the image. This
combination requires the live pilot; an unsupported image fails visibly.

The default pill is the chosen runtime's normal boot pill. `--pill HTTPS_URL`
overrides it only at first boot; a restart never silently reboots an existing
pier. `--loom` defaults to 31 and accepts the pilot's accepted argument range 30–34
(the Vere behavior itself still needs live verification).

The service checks for changed image digests hourly. `updates check` checks now;
`notices list` retains notices until `notices dismiss ID`. No image is applied
automatically. **Persistent desktop notification delivery and runtime upgrade
application are not wired up yet.** Notices are persistent backend records only.

## Safety boundaries

Real-network creation is disabled by default (`allow_real_ships: false`). Keyed
and comet creation paths exist, but remain behind this pilot gate. Do not enable
it for a valuable ship before the disposable integration checks pass. There are
no backups, rollback automation, or fleet-wide real-identity fencing yet.

The agent API permits development operations and read-only real-ship inventory,
but denies real lifecycle changes, real code/log access, and real Hoon. Enforce
this through a restricted forced-command SSH key, not merely `--agent`; see
[SECURITY.md](docs/SECURITY.md). An agent with your unrestricted SSH shell or
Docker access can bypass the manager and is not contained by these policies.

Shutdown uses SIGTERM and does **not** automatically escalate to SIGKILL. The
restart policy is disabled before an intentional stop. Startup restores the
configured restart behavior. No automatic cross-host failover or replication
exists, and unreachable hosts are never represented as confirmed stopped.

## Development

```sh
go test -race ./...
go vet ./...
go build -o urbitctl ./cmd/urbitctl
bash -n scripts/install.sh scripts/preflight.sh scripts/pilot.sh
```

See [DECISIONS.md](docs/DECISIONS.md), [DEVNET.md](docs/DEVNET.md), and
[SOURCES.md](docs/SOURCES.md) for scope, networking constraints, and upstream
references. The repository includes operator and agent `SKILL.md` files.
