# Omabit 0.1.0

The initial stable release for the supported Linux/AMD64 owner-user runtime and
stock-Omarchy desktop integration. It incorporates the tested RC.15 setup fix,
retained desktop bootstrap, immutable UI/tool releases and isolated test profiles.
Runner schema21, desktop state/API1 and the existing identity/protocol contracts
are retained.

## Supported contract

- Hosting: Linux/AMD64, systemd/cgroup v2, the selected existing Unix owner and the
  isolated rootless Docker stack. Native Pacman and Debian/Ubuntu verification
  providers are implemented; unresolved providers/configurations remain explicit.
- Stable Docker27.2.0–29.x/API1.47 is the compatibility policy. Actual live-qualified
  stacks and merely compatible stacks remain distinguished in host reports.
- Official content-pinned Vere4.6 and the existing supplied Recommended Apps pill.
- The generic collector/runner do not require Omarchy. A remote-only desktop needs
  no local engine; it connects to an authorized, identity-pinned runner over SSH.
- The retained bootstrap uses stock Omarchy's service lifecycle. The observed
  Omarchy4.0.3/4.0.4 results and remaining platform limits are recorded in qualification.

Stable does not imply that every Linux distribution, native configuration layout,
engine patch or network topology has been exercised. See [host compatibility](host-compatibility.md).
The historical API mode string `development-managed` remains for compatibility;
it is not renamed by changing the release label.

## Installation and desktop updates

```sh
omarchy plugin install https://github.com/thelifeandtimes/omabit.git --enable
```

For an existing native installation:

```sh
omarchy plugin update thelifeandtimes.omabit
omarchy shell thelifeandtimes.omabit.bootstrap status
```

The ordinary default profile follows published pins. Explicit development pins
remain pinned; use `scripts/omabit-dev.py select default --release follow` from the
installed/source checkout to resume following them. Watch the actually loaded
version and activation state while preparation completes.

## Upgrade support

Supported installed owner-user paths are RC.12/schema20 and RC.13–15/schema21 through
the normal journaled installer. RC.10/11/schema19 use the separate documented
offline bridge. Earlier database-reader compatibility is not a claim that legacy
dedicated-account deployments can be converted automatically.

Stable0.1.0 becomes the retained baseline for subsequent minor/major releases.
Each successor must carry explicit reader/migration compatibility, preserved
identities/receipts/settings, interruption recovery and downgrade-refusal evidence.
Management upgrades preserve running ships and stopped intent; a required engine
or Vere restart is a separate coordinated operation.

Frozen0.1.0 forward-activation and successor-schema checks are recorded in
`docs/releases/0.1.0/qualification.json` and the attached release qualification.
Successor fixtures exercise the upgrade machinery and do not invent a released
future version. Future real releases must qualify their own changes. See the
[upgrade policy](upgrade-policy.md) and [host upgrade procedure](host-compatibility-upgrades.md).

## Development after 0.1.0

- `master`: protected stable releases, PRs only, full minor/major `X.Y.0` versions.
- `edge`: opt-in personal/bleeding-edge work, explicitly not stable; immutable
  `-edge.N` prereleases for published iterations.
- `develop`: unqualified development with no stability guarantee.
- Collector/runner, shared runtime/installer or dependency changes require a new
  minor or major release line. UI/QoL patches belong on edge until a reviewed release.

See [development policy](development-policy.md) for channel installation and exact
branch updates. Edge/develop initially share the stable baseline, but their channel
labels and guarantees are different.
