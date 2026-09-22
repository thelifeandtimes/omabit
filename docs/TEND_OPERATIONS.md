# Tend operations and compatibility

Tend `0.1.0` is a pre-release multiplayer build. Its owner-authoritative peer
protocol is enabled and has passed the documented three-ship live matrix, but
it has not completed the longer release-candidate soak and manual desktop
gates.

## Verified baseline

These are the versions used for the current release gates, not inferred minimum
requirements:

| Component | Verified value |
| --- | --- |
| Architecture | x86_64 |
| Omarchy | 4.0.0-1 |
| Quickshell | 0.3.0, AUR `quickshell-git` revision `28771c7c…` |
| Plugin manifest schema | 1 |
| Urbit runtime | Vere 4.6 |
| Urbit userspace | `%zuse` 409 |
| Tend save state | `%10` (loads `%0` through `%10`) |
| Tend desktop protocol | 1 (exact match required) |
| Tend JSON marks | `%tend-action-1`, `%tend-update-1` |
| Tend peer mark | `%tend-peer-1`, same release on every sharing peer |
| Python / Node used by gates | 3.14.7 / 25.2.1 |
| Time-zone database | System IANA tzdata through Python `zoneinfo` |
| Notification client | `notify-send` 0.8.8 |
| QML validation | `qmllint` against the installed Omarchy shell imports |

Install the desk, plugin, and CLI from the same Tend release, and update all
ships in a sharing group together. A protocol-1 desktop refuses a missing or
different snapshot protocol before becoming Online; mixed pre-release peer
builds are unsupported. The exact matrix and version-bump rules are in
`TEND_COMPATIBILITY.md`.

## Pre-update drill

1. Confirm the current client can reach its ship with `omabit tend status`.
2. Export and validate a backup:

   ```sh
   omabit tend export ~/Documents/tend-before-update.json
   omabit tend validate-backup ~/Documents/tend-before-update.json
   ```

3. Keep a recoverable pier-level backup or host snapshot. A portable export can
   be restored only into an empty Tend state and does not recreate sharing
   relationships; a pier-level backup remains the complete rollback boundary.
4. Build and run `make check-tend` and `make check-tend-release` from the exact
   source revision being installed.
5. On a disposable ship with the candidate desk committed, run the shared-code
   migration matrix:

   ```bash
   scripts/check-tend-migrations.sh /path/to/running/fake-ship-pier
   ```

   The gate must report that `%0` through `%10` passed. It constructs a
   non-empty saved noun for every schema inside the running ship and executes
   the same migration library called by Gall `+on-load`.

6. On three disposable fake ships with authenticated Eyre connection files,
   run the multiplayer fault gate described in `TEND_MULTISHIP_MATRIX.md`.
   The release bundle includes `scripts/check-tend-multiship.py`; the source
   tree also exposes it as `make check-tend-multiship` through nine documented
   path variables. The gate temporarily changes Ames policy and suspends Gall
   agents, so it must never target a production pier.

## Update

Run the release installer against the exact mounted desk. It preflights all
three targets, moves replaced desktop files to timestamped backups, backs up
the complete mounted desk, overlays Tend-owned source, and preserves the
mounted desk's standard marks and `sys.kelvin`:

```sh
./install.sh --force --desk-path /absolute/path/to/your/pier/tend
```

Do not replace a mounted desk with the release's `desk/` directory or copy its
`sys.kelvin` over the mounted value. Commit the installed overlay from the
dojo:

```hoon
|commit %tend
```

Watch the dojo for a successful build and reload. Then reconnect the desktop,
run `omabit tend status`, and verify a read plus one reversible mutation.

## Rollback

- Disable a broken desktop plugin with `omarchy plugin disable io.omabit.tend`.
- Restore the timestamped plugin and CLI backups created by `install.sh
  --force`, or reinstall a known release.
- The installer also creates a timestamped backup beside the mounted `%tend`
  directory before every desk overlay. Restore that complete directory and
  commit it to return to the immediately preceding desk source.
- Replacing mounted desk source with a prior release and committing it is safe
  only when that release understands the currently saved Gall schema. Tend
  rejects unknown saved-state shapes; do not force a code downgrade across a
  state-schema boundary.
- If a schema upgrade itself must be rolled back, restore the whole pier from a
  known-good host snapshot. Do not edit `.urb` state or inject a JSON export by
  hand.

## Troubleshooting

### The overlay asks for a new `+code`

The Eyre session expired or authentication failed. Run `+code` on the intended
ship and reconnect. Tend intentionally stops automatic retries on an explicit
authentication failure. It never saves the reusable code.

### Tend reports an incompatible desk protocol

Install the desk, plugin, and CLI from one release. A fresh login validates the
desk before saving the new cookie or endpoint, while an existing connection is
reported unreachable and never becomes Online. For a shared list, update every
participating ship to the same release before resuming edits. Do not work around
the check by sending raw Eyre actions; use the matrix in
`TEND_COMPATIBILITY.md`.

### HTTPS works in a browser but Tend refuses the URL

Use only the endpoint origin: scheme, host, and optional port. Credentials,
paths, queries, and fragments are rejected. Non-loopback endpoints must use
HTTPS and must pass the operating system's certificate validation.

### The bar says Offline or Checking

`Checking` means the client has not received its authoritative snapshot yet;
`Offline` means the stream ended. The service retries with bounded backoff. Run
`omabit --json tend status` to distinguish saved authentication from actual
reachability. Shared-list writes remain disabled unless their owner is Online.

If only one shared list stays Offline, its owner ship is unavailable or the
peer subscription is catching up. The last confirmed replica remains readable.
Do not repeatedly submit mutations: the Gall agent and CLI both reject new
writes until the owner is Online. After an owner restart, Checking is expected
until the new owner session has sent a fresh snapshot and liveness
acknowledgement.

### The plugin is not discovered

```sh
omarchy plugin validate ~/.config/omarchy/plugins/io.omabit.tend
omarchy plugin list --json
omarchy-shell shell listPlugins
```

The plugin directory must contain `manifest.json` at its root. Runtime and
credential files belong under the XDG runtime/config paths, not inside the
watched plugin directory.

### A mutation is rejected

`stale-list` or `stale-preferences` means another confirmed update advanced the
revision. Wait for the stream update and retry against the current view.
Validation errors leave canonical state unchanged. Use the JSON CLI form when
collecting a reproducible error, but never attach a cookie jar, pier key, or
`+code` to a report.

### A date or time zone is rejected

Tend resolves local wall-clock values with the system IANA time-zone database.
Install or update the distribution's `tzdata` package if a valid IANA zone is
reported as unavailable. A wall time skipped by a daylight-saving transition
is intentionally rejected; choose the first valid time after the gap.

### Notifications do not appear

Confirm `notify-send` exists and that the desktop notification service allows
Tend notifications. Alert presentation is durable: a stable notification is
replayed after a desktop reconnect until the client acknowledges it. If an
alert repeatedly returns, inspect the bridge logs for a failed `ack-alert`
operation rather than deleting Gall state.

### Restore is rejected

`omabit tend restore FILE --yes` works only when Gall has no hosted lists,
replicas, invitations, or in-flight operations. Run `omabit tend list` and
`omabit tend invitations` to inspect the visible blockers. The command verifies
Gall's durable operation receipt and reports its rejection reason; an Eyre HTTP
acknowledgement alone is not success. Restore never recreates ACLs or peer
subscriptions.

The live multiplayer evidence and remaining fault cases are recorded in
`docs/TEND_MULTISHIP_MATRIX.md`.
