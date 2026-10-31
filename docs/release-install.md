# Omabit 0.3.3 binary bundle

This Linux x86-64 bundle contains prebuilt `omabit`, `urbit-collector`,
`urbit-runner`, the Omarchy plugin, service templates, CLI references and agent
skills. Go is not required. Hosting uses the existing pier-owning Unix user and
its separate rootless Docker engine; the default pier root is `/srv/urbit/piers`.

## Install the desktop

For ordinary Omarchy installation:

```sh
omarchy plugin install https://github.com/thelifeandtimes/omabit.git --enable
```

The native plugin verifies its pinned UI/tool bundle and retains profiles,
enrollments and preferences. A desktop update does not automatically update a
local or remote runner.

For this extracted archive, verify and install from its root:

```sh
sha256sum --check SHA256SUMS
./install.sh --enable
```

To replace an existing unmodified archive installation:

```sh
./install.sh --replace --enable
```

Archive installation uses cold desktop replacement. Unlock the desktop first.
Open **Setup → Set up this computer** for fresh host setup, or use
`./build/omabit --format text setup`. A remote-only desktop needs no local Docker.

## Host prerequisites

The desktop requires Omarchy/Quickshell and Python 3. Hosting requires Linux
x86-64, systemd/cgroup v2, supported distribution-native Docker27.2.0–29.x with
API1.47, RootlessKit, slirp4netns and OpenSSH. Native package authenticity and
host identity checks apply. Compatibility does not mean every distro/engine
combination has received live qualification; consult the release record.
The owner installer does not replace unrelated Docker installations or upgrade
OS packages. Guided setup explains any missing dependencies before installation.

## Install or update a runner

For an existing local Omabit Setup installation, choose **Update Dependencies**
in the menubar widget, or **Setup → Local services & updates → Check services →
Update services**, then authorize the OS prompt.
See the [guided upgrade](https://github.com/thelifeandtimes/omabit/blob/master/docs/guided-service-upgrades.md). The commands below remain
for standalone/headless installations.

The qualified previous-stable path is owner-user0.3.2 →0.3.3. Earlier stable
installations use the documented intermediate paths to0.3.2 first.
The target uses **schema47**, management protocol1 and desktop/bootstrap state1.
Finish pending management work first. Prepare a new launcher bundle with the
new executable, reuse the existing owner and public operator key, then install:

```sh
./build/urbit-runner preflight
./build/urbit-runner prepare-rootless --output /absolute/path/to/new/rootless-bundle
sudo ./build/urbit-runner install-host --owner YOUR_EXISTING_OWNER \
  --bundle /absolute/path/to/new/rootless-bundle \
  --public-key /absolute/path/to/existing/operator.pub \
  --acknowledge-host-changes
```

The receipt supplies the stable runner ID and management SSH host key; verify
these when enrolling. Management SSH is restricted RPC, not a general shell.
The activation journal fences new mutations and retains executable generations.
Management upgrades preserve piers, identities, access grants, configuration,
operation history and intended running/stopped state. They preserve the healthy
engine and running containers. No Vere update is introduced by this release.

Retry an interrupted activation with the **same target executable and bundle**.
Once migration commits, use compatible new code for recovery. Older runners
refuse the newer schema; do not lower its marker, erase journals or re-enroll
piers to downgrade. Older RC19 hosts require the separately documented offline
bridge. This bundle does not provide an OS-account or pier-location migration.

Read the [0.3.3 upgrade guide](https://github.com/thelifeandtimes/omabit/blob/master/docs/release-0.3.3.md)
and [qualification record](https://github.com/thelifeandtimes/omabit/blob/master/docs/releases/0.3.3/qualification.json)
for verified behavior and limits. The qualification record may live in the
protected master distribution commit when the tag identifies frozen source.

## Bootstrap pills

New-ship forms offer Bare and Recommended Apps. Recommended Apps uses the existing
stable pill: **59,351,751 bytes**, SHA-256
`85635c0b17a138a6473580ca362475faadf281e09cc9fa3746cfc698cc054f62`.
0.3.3 republishes those unchanged bytes. The configured compatibility alias
`https://urbit-foundation.s3.us-east-2.amazonaws.com/omabit.pill` resolves to the
immutable RC.11 asset and compiled hash/size; changed content fails verification.
No new application composition or OTA inventory is claimed.

Bare uses Vere's standard bootstrap selection. The pinned Vere4.6 bootstrap
used in project/runtime qualification is a different217,421,172-byte artifact;
it is not the Recommended Apps asset. Explicit custom HTTPS/local pill inputs
remain available, optionally with `--pill-sha256` and `--pill-size`. Initialized
piers resume without reapplying a pill. Pill uploads permit4GB; local stopped-pier
imports permit8 TiB, while cross-host handoff remains capped at100GB logical contents. Resource and identity/port/storage checks still apply.

## CLI and agent guidance

Start with `./build/omabit --help`, `./build/urbit-collector help`, or
`./build/urbit-runner --help`. Help and the reference generation are offline.
The bundle includes `docs/cli/` and these skills:

- `skills/omabit-operator/SKILL.md`: runner selection, revisions, request receipts,
  lifecycle, imports, project networks and recovery boundaries.
- `skills/omabit-plugin-developer/SKILL.md`: independent authentication,
  standalone login, missing-backend installation, recovery and shared native UI.

Plugin sessions talk directly to the ship. Omabit assists approval and cleanup;
ordinary Eyre sessions are not desk-scoped permissions. Wallet custody and the
broader security/privacy audit remain separate work.

## Scope

Fake-project networking/member lifecycle, fleet management and application
connections are included. Project maintenance/import, expanded engine/anchor
failure recovery, turnkey private Ethereum chains and collection-switching UI
remain incomplete. Interrupted or uncertain operations retain their records for
review. Production and development profiles need separate runners/storage when
runtime isolation is required; a desktop profile alone does not isolate a ship.

[Release notes](https://github.com/thelifeandtimes/omabit/blob/master/docs/release-0.3.3.md)
cover the support scope. Tend is maintained separately at
[thelifeandtimes/tend](https://github.com/thelifeandtimes/tend).
