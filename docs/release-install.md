# Omabit 0.1.0-rc.15 binary bundle

This is an early release candidate for Linux x86-64. It contains prebuilt
`omabit`, `urbit-collector`, `urbit-runner`, the Omarchy plugin and service templates.
Go is not required to use this bundle.

For native GitHub installation through Omarchy instead of extracting this archive,
use `omarchy plugin install https://github.com/thelifeandtimes/omabit.git --enable`.
Native installation uses the retained bootstrap and a complete checksum-pinned
UI/tool bundle. The extracted archive's installer below remains the separate
snapshot-installation model, with cold replacement for updates.

RC.7 retains management and the separate rootless Docker engine as the existing pier
owner, with `/srv/urbit/piers/` as default storage. Mounted desks are ordinary
user-owned files. Storage/import/deletion and bounded reboot resume continue.
This remains a development prerelease.

RC.15 corrects the first-install subordinate-ID check on stock Omarchy. Existing
root-owned OS grants for the selected local account are reused after validation;
an Omabit owner profile is no longer a prerequisite for consuming those grants.
New ranges still require their normal allocation authority. The setup checklist
and password-authorization flow are retained, with technical diagnostics collapsed.
See [RC.15 notes](https://github.com/thelifeandtimes/omabit/blob/v0.1.0-rc.15/docs/release-rc15.md).

RC.14 adds retained desktop activation, immutable release-specific QML/tool paths,
isolated test profiles and development tooling. It keeps runner **schema21** and
adds a separate version1 desktop activation/profile store. See
[RC.14 notes](https://github.com/thelifeandtimes/omabit/blob/v0.1.0-rc.14/docs/release-rc14.md).
Remote-only workstations can enroll an existing runner without local host setup or
Docker; fresh desktop and fresh host installation are separate checks.

RC.13 introduces **schema21** and distribution-native host compatibility: stable
Docker27.2.0–29.x, API1.47 checks, native Pacman/Debian/Ubuntu authentication,
package-refresh hooks and shared headless/fleet/UI reporting. See
[RC.13 notes](https://github.com/thelifeandtimes/omabit/blob/v0.1.0-rc.13/docs/release-rc13.md).
Fresh-install acceptance on two Omarchy versions is pending user testing; compatible
does not mean every engine/distro stack is live-runtime-qualified.

RC.12 hardens storage/recovery, identity/replay, credentials and release/update
contracts after the adversarial review, retaining the existing desktop layout and
workflows. It uses **schema20**. See [RC.12 notes](https://github.com/thelifeandtimes/omabit/blob/v0.1.0-rc.12/docs/release-rc12.md) and the
[remediation record](https://github.com/thelifeandtimes/omabit/blob/v0.1.0-rc.12/docs/reviews/2026-10-18-rc11-remediation.md).

RC.11 collapses loom and storage choices under **Advanced settings**, limits new
loom selections to30–34 (default31), and makes native control CLI-only. Owned-key
boot includes Get an Urbit ID guidance. Historical runtime settings and existing
CLI native choices are preserved. See [RC.11 notes](https://github.com/thelifeandtimes/omabit/blob/master/docs/release-rc11.md).

RC.10 adds loom20–34 selection and default-off native control for every ship class,
with separate requested and recorded launch settings. Legacy HTTP edits preserve
native opt-in, and cross-host transfers preserve settings under signed receipts.
See [RC.10 notes](https://github.com/thelifeandtimes/omabit/blob/master/docs/release-rc10.md).

RC.9 adds host Ames UDP publication for new live enrollments, universal pre-move
epoch checks and explicit retained-enrollment recovery in the CLI and desktop.
Recovery supports guided, moved, in-place and named-pier flows and leaves ships
stopped. See [RC.9 notes](https://github.com/thelifeandtimes/omabit/blob/master/docs/release-rc9.md)
and the [stable upgrade requirement](https://github.com/thelifeandtimes/omabit/blob/master/docs/upgrade-policy.md).

RC.8 simplifies initial setup to one machine-use toggle and an installation
approval checkbox with expandable details. Guided import needs only source and
configured storage location, derives identity/path/port, and reports copy/verification
progress. Named pier directories and compatible retained historical epochs are
recognized. Update all three tools and the plugin together for the new capabilities.
See [RC.8 notes](https://github.com/thelifeandtimes/omabit/blob/master/docs/release-rc8.md).

RC.7 adds persistent, identity-pinned runtime-log following to remove per-refresh
SSH/PAM/logind sessions. It cancels on pause/hide/minimize/navigation, uses bounded
retry backoff and stops on identity/access/configuration failures. The collector
and plugin are updated together. See [RC.7 notes](https://github.com/thelifeandtimes/omabit/blob/master/docs/release-rc7.md).

RC.6 corrected Recommended Apps default resolution when Omarchy supplies only
explicit widget settings. It includes the RC.5 refinements below and the same
backend/runtime implementation and supplied pill bytes.

RC.5 explains installed binaries and source builds, keeps log/port progress inline
without moving controls, preserves public comet names in runtime output, displays
HTTP ports in the fleet and marks Application Connections as Coming Soon. Confirmed
first boot and healthy-only browser/+code access continue from RC.4.

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
systemd/cgroup v2, stable Docker27.2.0–29.x with API1.47, RootlessKit, slirp4netns
and OpenSSH for hosting. Native package authenticity and host identity checks apply;
see [host compatibility](https://github.com/thelifeandtimes/omabit/blob/v0.1.0-rc.13/docs/host-compatibility.md).
The setup flow verifies prerequisites and uses the selected existing Unix user.
It does not install OS packages or replace unrelated Docker installations.

## Update an already configured local runner

RC.10/RC.11 owner-user hosts first use the new executable's explicit
[one-time offline upgrade](https://github.com/thelifeandtimes/omabit/blob/v0.1.0-rc.13/docs/management-offline-upgrade.md). That procedure stops
only management, preserves running runtime containers, holds the existing state
lock and commits migration/admission fencing transactionally. It does not reset or
re-enroll piers. The ordinary command safely refuses an online historical manager
that lacks quiescence support.

RC.12/schema20 and subsequent quiescence-aware updates use a new rootless bundle
prepared by the target executable and the existing operator public key with the
ordinary owner installer below. Old RC.12 guard/unit templates do not match RC.13.
Updates preserve owner, runner
identity, SSH keys/grants and running containers. Retained generations and the
activation journal provide repeat-command recovery and schema-compatible rollback.
Schema **21** reads historical schemas11–20 without rewriting original bodies or
receipts; older runners refuse it. Database readability does not migrate OS accounts
or pier locations. Source tests cover owner-user schema19 offline and schema20 online
updates; see the [upgrade procedure and scope](https://github.com/thelifeandtimes/omabit/blob/v0.1.0-rc.13/docs/host-compatibility-upgrades.md).
The historical `install-services.sh` refuses owner installations.

## Install or update a headless runner

Use the prebuilt `build/urbit-runner` (or the standalone runner release asset):

```bash
./build/urbit-runner preflight
./build/urbit-runner prepare-rootless --output "$HOME/omabit-rootless"
sudo ./build/urbit-runner install-host --owner "$USER" \
  --bundle "$HOME/omabit-rootless" --public-key /path/to/operator.pub \
  --acknowledge-host-changes
```

Choose a new bundle output directory and reuse the existing host's public operator key.
The install receipt supplies the stable runner ID and management SSH host key;
verify/pin these when enrolling the host in the collector. Management SSH is a
restricted RPC endpoint, not a general shell or SFTP service.

## Recommended Apps pill

New-ship forms offer **bare bootstrap** and **Recommended Apps**. Bare leaves
bootstrap selection to Vere. Recommended Apps now defaults to the supplied URL:

`https://urbit-foundation.s3.us-east-2.amazonaws.com/omabit.pill`

An operator may override that URL in plugin settings.
The built-in URL is a compatibility alias for the immutable RC.11 pill artifact,
whose SHA-256 and size are compiled into the RC.15 collector. RC.15 republishes the
same supplied bytes; no new pill composition or application inventory is claimed.
Network provisioning requires HTTPS. Custom HTTPS/local sources remain explicit
operator inputs and can additionally supply `--pill-sha256` and `--pill-size`.
Verified bytes are uploaded before registration and used on first Start.
Initialized piers resume without reapplying a pill.

This release also attaches the frozen supplied bytes qualified for RC.5: **59,351,751 bytes**,
SHA-256 `85635c0b17a138a6473580ca362475faadf281e09cc9fa3746cfc698cc054f62`.
It passed a disposable fake first boot, healthy HTTP/identity checks and clean stop.
Application inventory and OTA behavior are separate from this runtime smoke gate.
The configured provider URL can change later; each actual registration still binds
the downloaded content hash/size. Release assets preserve the version tested here.

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
