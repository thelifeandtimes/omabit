# Omabit 0.1.0-rc.2 binary bundle

This is an early release candidate for Linux x86-64. It contains prebuilt
`omabit`, `urbit-collector`, `urbit-runner`, the Omarchy plugin and service templates.
Go is not required to use this bundle.

For native GitHub installation through Omarchy instead of extracting this archive,
use `omarchy plugin add https://github.com/thelifeandtimes/omarchy-omabit.git --enable`.
See [native plugin distribution](native-plugin.md). Published RC.2 archive bytes
are unchanged; the generated native distribution uses their UI and binary pins.

RC.2 adds storage locations, reviewed pier import, confirmed deletion, stable btrfs
directory identities, workspace-aware monitor activation and automatic resume after
an unclean host reboot. This remains a development prerelease.

## Verify and install the desktop

After extracting the versioned archive, run from its root:

```bash
sha256sum --check SHA256SUMS
./install.sh --enable
```

For an existing unmodified Omabit installation:

```bash
./install.sh --replace --enable
```

Updates use the cold-replacement installer. Unlock the desktop first and finish
this step before authorizing host changes. Open **~+ → Setup → Set up this computer**
for first-time host setup. Terminal equivalent:

```bash
./build/omabit --format text setup
```

Prerequisites: Omarchy/Quickshell and Python 3 for the desktop; Linux x86-64,
systemd/cgroup v2, Docker 29.7.2, RootlessKit, slirp4netns and OpenSSH for hosting.
The setup flow verifies prerequisites and uses a dedicated rootless account.
It does not install OS packages or replace unrelated Docker installations.

## Update an already configured local runner

Wait for accepted operations to finish, then run from the extracted bundle:

```bash
sudo /usr/bin/bash ./scripts/install-services.sh
```

This updates management/SSH services. Running ship containers continue independently.
The runner uses schema 15; upgrades from permanent-layout schema 11–14 retain their
recorded state. Earlier runner binaries will refuse the upgraded database.

## Install or update a headless runner

Use the prebuilt `build/urbit-runner` (or the standalone runner release asset):

```bash
./build/urbit-runner preflight
./build/urbit-runner prepare-rootless --output "$HOME/omabit-rootless"
sudo ./build/urbit-runner install-host \
  --bundle "$HOME/omabit-rootless" --public-key /path/to/operator.pub \
  --acknowledge-host-changes
```

For an existing host, reuse its reviewed rootless bundle and public operator key.
The install receipt supplies the stable runner ID and management SSH host key;
verify/pin these when enrolling the host in the collector. Management SSH is a
restricted RPC endpoint, not a general shell or SFTP service.

## Draft Recommended Apps pill

New-ship forms offer **bare bootstrap** and **Recommended Apps**. Bare leaves
bootstrap selection to Vere. Recommended Apps downloads the versioned pill asset
configured by this candidate; an operator may override that URL in plugin settings.
The selected bytes are hashed/uploaded before registration and used on first Start.
Initialized piers resume without reapplying a pill.

The RC.2 pill deliberately reuses the frozen rc.1 draft bytes, supplied originally
as `omabit-brass.pill`, 57,011,409 bytes,
SHA-256 `51f0be723918ab7feaaf28b3c599f36f0f0fca6e68453f84517ba1659f07d57e`.
Its provider-reported desks are base, webterm, groups, trunk, grubbery, omart, pals,
hawk, obelisk, jars, silo and landscape. **MCP and Omachan await the replacement
pill.** The release notes and attached qualification report state what was actually
boot-tested; this candidate does not claim complete application/OTA qualification.
The earlier pill qualification found Grubbery compilation errors and Azimuth HTTP
504 snapshot failures. Start clean-install lifecycle checks with bare bootstrap.

Pill uploads permit 4 GB; stopped-pier moves permit 100 GB of logical contents.
There is no fixed registration-count cap. Resource admission, content verification
and identity/port/storage reservations still apply.

## Release state

This candidate keeps the pinned official Vere 4.6 runtime; it has no automatic
runtime-channel updater. Interrupted/uncertain operations require recorded-state
review. A stopped host move permanently fences its source and leaves its destination
stopped for explicit Start. Historical pier restoration is not an interface action.

Healthy initialized ships record their host boot, container and revision. After a
later hard reboot the runner may resume their existing piers once, including after
nonzero exits; Vere performs its normal event replay. Intentional stops, unresolved
host operations and failed resume attempts retain their guards. Legacy failures
already present during an upgrade require reviewed recovery before this policy applies.

See <https://github.com/thelifeandtimes/omabit> for source, current qualification
records and known gaps. Tend is maintained separately at
<https://github.com/thelifeandtimes/tend> and owns the standalone `tend` command.
