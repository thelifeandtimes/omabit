# Omabit 0.1.0-rc.1 binary bundle

This is an early release candidate for Linux x86-64. It contains prebuilt
`omabit`, `urbit-collector`, `urbit-runner`, the Omarchy plugin and service templates.
Go is not required to use this bundle.

The development checkpoints through 0.23 are consolidated into this first public
release candidate. They are not newer public releases.

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
The runner uses schema 13; upgrades from permanent-layout schema 11/12 retain their
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

The rc.1 pill is a frozen draft supplied as `omabit-brass.pill`, 57,011,409 bytes,
SHA-256 `51f0be723918ab7feaaf28b3c599f36f0f0fca6e68453f84517ba1659f07d57e`.
Its provider-reported desks are base, webterm, groups, trunk, grubbery, omart, pals,
hawk, obelisk, jars, silo and landscape. **MCP and Omachan await the replacement
pill.** The release notes and attached qualification report state what was actually
boot-tested; this candidate does not claim complete application/OTA qualification.

Pill uploads permit 4 GB; stopped-pier moves permit 100 GB of logical contents.
There is no fixed registration-count cap. Resource admission, content verification
and identity/port/storage reservations still apply.

## Release state

This candidate keeps the pinned official Vere 4.6 runtime; it has no automatic
runtime-channel updater. Interrupted/uncertain operations require recorded-state
review. A stopped host move permanently fences its source and leaves its destination
stopped for explicit Start. Historical pier restoration is not an interface action.

See <https://github.com/thelifeandtimes/omabit> for source, current qualification
records and known gaps. Tend is maintained separately at
<https://github.com/thelifeandtimes/tend> and owns the standalone `tend` command.
