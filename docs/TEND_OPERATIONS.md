# Tend operations and compatibility

Tend `0.1.0` is a pre-release local daily-driver build. Its peer protocol is
defined but disabled, so this document does not describe it as a multiplayer
release.

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
| Tend save state | `%6` (loads `%0` through `%6`) |
| Tend JSON marks | `%tend-action-1`, `%tend-update-1` |
| Python / Node used by gates | 3.14.7 / 25.2.1 |
| Time-zone database | System IANA tzdata through Python `zoneinfo` |
| Notification client | `notify-send` 0.8.8 |

Until a broader matrix is tested, install the desk, plugin, and CLI from the
same Tend release. An older client can read the frozen Milestone 0 update
shapes, but mixed-version mutation compatibility is not a release promise.

## Pre-update drill

1. Confirm the current client can reach its ship with `omabit tend status`.
2. Export and validate a backup:

   ```sh
   omabit tend export ~/Documents/tend-before-update.json
   omabit tend validate-backup ~/Documents/tend-before-update.json
   ```

3. Keep a recoverable pier-level backup or host snapshot. The portable Tend
   restore transition is not implemented yet.
4. Build and run `make check-tend` and `make check-tend-release` from the exact
   source revision being installed.

## Update

The release installer moves replaced desktop files to timestamped backups:

```sh
./install.sh --force
```

For a mounted desk, copy the new release's `desk/` contents into the exact
mounted `%tend` directory, then commit from the dojo:

```hoon
|commit %tend
```

Watch the dojo for a successful build and reload. Then reconnect the desktop,
run `omabit tend status`, and verify a read plus one reversible mutation.

## Rollback

- Disable a broken desktop plugin with `omarchy plugin disable io.omabit.tend`.
- Restore the timestamped plugin and CLI backups created by `install.sh
  --force`, or reinstall a known release.
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

### HTTPS works in a browser but Tend refuses the URL

Use only the endpoint origin: scheme, host, and optional port. Credentials,
paths, queries, and fragments are rejected. Non-loopback endpoints must use
HTTPS and must pass the operating system's certificate validation.

### The bar says Offline or Checking

`Checking` means the client has not received its authoritative snapshot yet;
`Offline` means the stream ended. The service retries with bounded backoff. Run
`omabit --json tend status` to distinguish saved authentication from actual
reachability. Shared-list writes remain disabled unless their owner is Online.

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
Tend notifications. Alert offsets are recorded durably in Gall, but replay
until explicit desktop acknowledgement is still a release blocker; a client
that is disconnected at the firing instant can currently miss presentation.
