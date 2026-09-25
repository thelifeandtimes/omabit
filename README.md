# Tend by Omabit

Tend is a multiplayer reminders and task application backed by Urbit. This
repository contains:

- the `%tend` Gall desk and browser application;
- the Tend Omarchy panel, service, and menubar widget;
- the `omabit tend` command-line client; and
- release, test, migration, and agent-skill tooling.

Tend keeps its canonical state on your ship. Omabit does not require a hosted
account, database, analytics service, or mandatory notification relay.

> **Status:** Tend is pre-release software. Keep the Gall desk, Omarchy plugin,
> and CLI on compatible versions. Review the plugin source before enabling it;
> Omarchy plugins execute inside the shell process.

## Quick installation

### 1. Install the Gall agent

From your ship's dojo:

```hoon
|install ~dister-dozzod-sarlev %tend
```

This installs the Gall agent, Landscape docket, and browser application. Once
the install completes, Tend is available at:

```text
https://YOUR-SHIP-DOMAIN/apps/tend
```

### 2. Install the Omarchy plugin and CLI

Clone the repository and build the deterministic Tend release:

```sh
git clone https://github.com/thelifeandtimes/omabit.git
cd omabit
make dist-tend
```

Extract and verify it:

```sh
version=$(tr -d '[:space:]' < VERSION)
release_dir=$(mktemp -d)
tar -xzf "dist/omabit-tend-$version.tar.gz" -C "$release_dir"
cd "$release_dir/omabit-tend-$version"
sha256sum --check SHA256SUMS
```

Install and enable the Omarchy plugin:

```sh
./install.sh --enable
```

The installer places the plugin at
`${XDG_CONFIG_HOME:-$HOME/.config}/omarchy/plugins/io.omabit.tend` and the CLI
at `$HOME/.local/bin/omabit`. Existing installations are not overwritten unless
`--force` is supplied; replacements are backed up first.

Connect the desktop to your ship. The `+code` is read from a hidden prompt and
is exchanged for an Eyre session rather than saved:

```sh
omabit tend connect https://YOUR-SHIP-DOMAIN
omabit tend status
```

You can also connect from the Tend panel's settings screen.

## Manual source installation into a ship

Use these steps when you want to build and install the `%tend` desk from this
repository instead of downloading it from `~dister-dozzod-sarlev`.

### 1. Create and mount a desk

In your ship's dojo:

```hoon
|new-desk %tend
|mount %tend
```

The mount appears as a `tend/` directory inside the ship's pier. Keep the ship
running and determine the absolute path to that directory, for example:

```text
/home/alice/urbit/my-ship/tend
```

### 2. Build the release from the repository

In another terminal:

```sh
git clone https://github.com/thelifeandtimes/omabit.git
cd omabit
make dist-tend

version=$(tr -d '[:space:]' < VERSION)
release_dir=$(mktemp -d)
tar -xzf "dist/omabit-tend-$version.tar.gz" -C "$release_dir"
cd "$release_dir/omabit-tend-$version"
sha256sum --check SHA256SUMS
```

### 3. Overlay Tend onto the mounted desk

Run the release installer with the absolute mounted-desk path:

```sh
./install.sh --desk-path /absolute/path/to/your/pier/tend --enable
```

The installer deliberately preserves the ship-generated minimal marks and
`sys.kelvin`, overlays the Tend source and required Urbit libraries, and creates
a timestamped backup before replacing an existing installation. Do not replace
the mounted desk wholesale with the repository's `desk/` directory.

For an intentional source update where the mounted desk already contains older
Tend files, use:

```sh
./install.sh --force --desk-path /absolute/path/to/your/pier/tend --enable
```

### 4. Commit and start the desk

Back in dojo:

```hoon
|commit %tend
|install our %tend
```

`|install our %tend` is needed for the first installation of a newly mounted
desk. On later updates, `|commit %tend` rebuilds and reloads the installed
agent. If the agent is suspended, use `|revive %tend`.

### 5. Optionally publish your desk

To let another ship install your committed desk:

```hoon
:treaty|publish %tend
```

The other ship can then run:

```hoon
|install ~your-ship %tend
```

Do not commit a pier, ship keys, `+code`, Eyre cookie, or any other private
runtime data to this repository.

## Everyday use

Open the Omarchy panel or use the CLI:

```sh
omabit tend today
omabit tend lists
omabit tend add "Buy milk" --list Inbox --tag groceries
omabit tend schedule Inbox 42 --due 2026-09-26T09:00 --timezone America/Los_Angeles --repeat weekly
omabit tend move-list Inbox 42 "Shared chores"
omabit tend complete 1 42
omabit tend invitations
```

Run `omabit tend --help` for the full command list. The same installed desk
serves the browser client at `/apps/tend`.

## Development

Run the complete Tend validation suite with:

```sh
make check-tend
```

Build and validate an installable release with:

```sh
make dist-tend
make check-tend-release
```

The initial Codex-compatible Tend skill is in [`skills/tend`](skills/tend).

## Documentation

- [Complete installation and update guide](docs/TEND_INSTALL.md)
- [Operations and troubleshooting](docs/TEND_OPERATIONS.md)
- [Protocol](docs/TEND_PROTOCOL.md)
- [Compatibility matrix](docs/TEND_COMPATIBILITY.md)
- [Backup and restore](docs/TEND_BACKUP.md)
- [Accessibility](docs/TEND_ACCESSIBILITY.md)
- [Native mobile client plan and open decisions](docs/TEND_MOBILE_CLIENT_PLAN.md)
- [Product architecture](PRODUCT_ARCHITECTURE_PLAN.md)

The repository also contains early Omabit host-manager, Pals, and Tlon
Messenger workstreams. Their boundaries and current state are documented in
[`docs/WORKSTREAMS.md`](docs/WORKSTREAMS.md).
