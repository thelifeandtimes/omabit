# One-click local setup trial

Candidate: **0.3.7-dev.20261105.5**, branch `feature/one-click-host-setup`, based on
stable 0.3.6. This is a local development build; the public plugin still uses the
published 0.3.6 bundle.

## Manual trial on sourceport

The first candidate (`.1`) installed the missing packages and connected sourceport's
local runner. The next candidate (`.2`) incorporates the user's onboarding feedback:
the welcome button opens local setup directly, ship creation is offered only when
a runner is ready, and technical package names are behind **View details**.
Candidate `.3` organizes those details into system dependencies, Omabit's three
tools, and the Vere runtime used inside Docker containers.
Candidate `.4` fixes key-file Browse on fresh desktops by using Qt's non-native
file chooser instead of requiring a separately installed Zenity executable.
Its manual trial exposed a layering/input defect: the chooser opened underneath
the menu. Candidate `.5` keeps the chooser inside the owning overlay surface.
Real Wayland pointer selection, Escape cancellation, reopening and restored
background input pass in an isolated headless compositor. The user confirmed
the fix on their actual desktop; see M514–M515 in
[development memory](development-memory.md).

Update the existing local-review installation with:

```sh
omarchy plugin update thelifeandtimes.omabit
```

The authorized repeat-trial reset completed on 2026-11-05. The plugin was removed,
the empty runner's services stopped/disabled, and its management state, enrollment,
setup credentials and panel preferences retired with recoverable same-parent
renames. The manifest is
`~/.local/share/omabit-review/reset-20261105T191545Z.json`. The two trial-installed
packages were removed. Pier directories and shared host settings were preserved.
Immediately after that reset, the helper reported `can_setup: true`, `host_present: false`, `ready: false`,
and exactly `rootlesskit`/`slirp4netns` to install.
The user subsequently reinstalled and continued onboarding; no further reset is
part of the picker update.

After confirming `.5`, the user explicitly requested another fresh-install reset.
That reset is complete: Omabit and the two trial-installed packages are removed;
the dedicated services are inactive and management/setup state is recoverably
retired. The new manifest is
`~/.local/share/omabit-review/reset-20261106T002120Z.json`. Post-reset status again
reports `can_setup: true`, `host_present: false`, `ready: false`, and exactly
rootlesskit/slirp4netns to install. The command below starts the next `.5` trial.

The matching UI and prebuilt tools are also packaged as a local Git repository.
Use the standard Omarchy installer as your normal desktop user:

```sh
omarchy plugin install ~/.local/share/omabit-review/one-click-setup --enable
```

This uses the local candidate as its Git origin. The public GitHub URL still
installs stable 0.3.6. The local distribution is a fixed manual-review snapshot;
it does not automatically incorporate further source edits.

To rebuild and install directly from this checkout instead:

```sh
bash ~/repos/omabit/scripts/install.sh
```

The source installer requires Go 1.26+ (already available on sourceport). Once the
tools have been built, the equivalent without rebuilding is:

```sh
python3 -B ~/repos/omabit/scripts/install-panel.py --enable
```

Open **Omabit → Set up this computer**. This goes straight to device setup.
The main view explains that the software needed to run Urbit on Omarchy will be
installed. **View details** lists `rootlesskit, slirp4netns` when they are missing.
Click **Set up this computer** once and
respond to the normal system password prompt. Expect package installation and
verification, rootless engine/services, private SSH enrollment, then **Add a ship**.
Reaching that last step is runner readiness; ship creation is a separate action.

If installation is interrupted, use the same setup action to retry. Completed
packages and the desktop key are reused. Package-manager locks, download errors,
unsupported engine versions and damaged installed packages are reported rather
than concealed by an automatic reset or package replacement.

## Implementation scope

- Native package provisioning supports Arch/Omarchy using configured Pacman
  repositories. Docker, containerd, runc, RootlessKit, slirp4netns, UID/GID mapping,
  iptables and OpenSSH prerequisites are identified by fixed executable checks.
- The package plan is recomputed by the privileged helper, after validating the
  bundle, owner, key and existing installation. No arbitrary package list is
  accepted from the UI. The same authorization covers packages and host setup.
- Read-only setup checks perform no package or service changes. Guided upgrades
  and the offline upgrade bridge keep their existing package behavior.
- Prerequisites and authenticated payloads are checked after installation, before
  service activation. Pacman success alone cannot report a connected runner.
- Subordinate-ID configuration, dedicated rootless engine, owner storage and SSH
  enrollment use the existing tested installer. Existing identities are preserved.

## Verification scope

For `.2`, the setup/details offscreen QML check and the existing onboarding fixture
pass. The latter ran with actual plugin components in networkless Bubblewrap,
private HOME/XDG, offscreen Qt and inert compositor/IPC adapters. It verifies
hidden initial ship creation, direct local navigation and hidden overview cards.
The adapter is `/tmp/opencode/run-onboarding-offscreen.py`; this does not claim
mapped-desktop visual review. The standard Omarchy validator accepts distribution
commit `fcbde29`.

Completed on 2026-11-05:

- Race-enabled Go tests and vet passed for `hostsetup`, `onboarding`, `hostinstall`,
  `rootless`, `hostcompat`, `remotesetup`, the runner CLI and the desktop helper.
- Desktop model/host-compatibility model tests passed, along with the actual
  offscreen setup/compatibility, RC11 pane and service-upgrade QML fixtures.
- Snapshot installation/recovery tests passed. The built plugin was staged outside
  the active desktop and accepted by `omarchy plugin validate` on Omarchy 4.5.0.
- All three static Linux binaries built successfully; the runner still reports
  schema 47. The new helper on sourceport reports `can_setup: true`,
  `host_present: false`, `ready: false`, and exactly
  `packages_to_install: ["rootlesskit", "slirp4netns"]`.

Focused installer/onboarding tests cover missing packages, refusal paths, partial
installation and retry, readiness rechecks, payload-authentication failure,
existing-key compatibility with actual OpenSSH and idempotent owner setup.
The offscreen QML fixture exercises the real setup pane's one-click action,
package list, hard-blocked prerequisites, progress and duplicate-click prevention.

The first manual trial reached a connected local runner; subsequent read-only checks
confirmed both packages installed, all three services running, verified enrollment,
and no registered ships or containers. The `.5` picker correction is available in
local distribution commit `12fd415` for the next manual pass. These trials do not
qualify a stable upgrade from 0.3.6.

The repeat `.5` trial subsequently completed installation, local connection and
Recommended Apps moon boot. A mistyped ship name caused first-boot identity
quarantine. After the user's confirmation and downtime approval, the narrowly
qualified [identity repair](boot-identity-repair.md) corrected registration and
resumed the same pier through ordinary CLI Start. At2026-11-06 01:56:31UTC the moon
was running/healthy as `~somryp-mitfyl-dozzod-sarlev`, with recovery cleared and
fresh resources. Original storage, configured ports, selected pill and operation
history were preserved. See [evidence](diagnostics/2026-11-05-moon-identity-mismatch.md)
and M517. This completes the repeat trial's working-moon objective.
