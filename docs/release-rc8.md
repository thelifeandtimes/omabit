# Omabit 0.1.0-rc.8 — simpler setup and guided pier imports

## Setup

- The always-on/laptop choice appears only in initial computer setup, as one switch.
- Installation requires an explicit approval checkbox. A compact **View details**
  disclosure retains the complete software, paths, services and source/build links.

## Guided imports and named piers

- Choose the existing stopped pier directory and an already configured storage
  location. Local hosts offer a folder picker. Omabit reads public identity and
  fake/live mode from metadata, derives an unused instance path and HTTP port,
  and presents the plan for confirmation.
- Imports expose a loading bar, real copy-byte counts and durable readable phase
  receipts through the existing inventory connection.
- New imports and known-identity creations use `<instance-name>/<ship-name>`
  directories. Discovery recognizes `.urb` under named and legacy `pier` leaves;
  runtime commands use the recorded basename. Existing paths are not auto-renamed.
- A source already at its destination is enrolled in place. An exclusive parent
  can be reused to rename a legacy leaf. Other moves use no-overwrite rename or
  a content-verified cross-filesystem copy, leaving the ship stopped.
- Testnet is a coming-soon stub for alternate Ethereum/Azimuth RPC endpoints.
  Offline fake/live metadata does not identify that endpoint.

## Existing native-pier compatibility

An actual failed enrollment receipt exposed two additional native-pier cases:

- Retained historical epoch-v1 snapshots (`north.bin`/`south.bin`) mixed with
  epoch-v2 `image.bin` snapshots. Import now validates intact history when the
  newest epoch is already Vere 4.6/v2. Mixed history receives an unqualified
  inventory source; it does not inherit qualified pruning/roll permission.
- Vere's internally hardlinked `.run` and `.bin/...` executables. These inert files
  are preserved only when every link is proven to remain within the reviewed pier.
  Cross-filesystem copies make independent private files. Shared private databases,
  external links, symlinks and special files remain refused; docked binaries are
  never executed.

No stale lock is automatically removed. A lock marker must be explicitly resolved
after proving the source is stopped. Import/recovery never boots or repairs a pier
merely to identify it.

## Qualification and fresh installation

Tests cover metadata bounds/branch traversal/corruption and official @p vectors,
changed identity/network and destination rejection, named-path discovery and
containment, in-place enrollment/rename, cross-filesystem copies and interruption,
mixed epoch history and unchanged pruning guards, closed dock-link sets, and mapped
setup/import/progress UI. The metadata reader also passed a bounded fuzz run.

A read-only review of the relocated protected pier verified its actual identity,
mixed history and complete 130-file / 49,556,773,510-byte source tree without
starting Vere or changing the pier. Fixture moves use disposable temporary data;
no live import/start of that protected pier is claimed.

The existing schema16, qualified Docker/Vere runtime, supplied pill and RC.7
persistent log watcher continue. Install matching runner, collector and QML for
`guided-pier-import` and `named-pier-directories`:

```bash
omarchy plugin install https://github.com/thelifeandtimes/omabit.git --enable
```

Operational cleanup follows the repository's protected-ship guidance and preserves
relocated piers. Native Vere's upstream mDNS patch remains a separate rollout.
