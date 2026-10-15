# Fresh RC.2 installation on sourcebox

This walkthrough uses the **downloaded GitHub binary bundle**, with no repository
checkout, Go compiler or development-path override. Shared OS prerequisites remain
installed. The previous app-owned host/desktop state must already be retired; this
document does not perform a teardown or restore old piers.

## First step: install only the desktop plugin

In an ordinary terminal **on sourcebox, inside its logged-in Omarchy desktop**:

```bash
cd "$HOME/omabit-releases/0.1.0-rc.2/omabit-0.1.0-rc.2-linux-amd64"
sha256sum --check SHA256SUMS
./install.sh --enable
```

Do not use sudo for the plugin installation. This installs the Omabit plugin and
its bundled `omabit`, `urbit-collector` and `urbit-runner` executables; it does not
create a runner host or start a ship. **Pause here to inspect the new-user UI.**

If installation succeeded but enabling was not confirmed, use the reported
`python3 -B scripts/install-panel.py --enable-only` command rather than reinstalling.

## Second step: use the normal first-time host setup

Open the Omabit bar icon → **Setup → Set up this computer**. Step through:

1. Prerequisite checks.
2. Creation of a fresh desktop management key.
3. Download/verification of the rootless host helper.
4. OS authorization for the dedicated runner account, engine and services.
5. Pinned local SSH enrollment and the Ready view.

The terminal equivalent, when explicitly chosen, is `./build/omabit --format text setup`.
Do not run both setup entry points concurrently. Record the first error rather
than removing additional files to retry.

Acceptance: a new runner ID and management SSH identity, schema 15 capabilities,
empty ship inventory/history, and enabled/active services with lingering. A later
collector should enroll this new identity; the old runner pin must not be reused.

## Third step: first ship

Use **bare bootstrap** for a disposable fake, then a comet. Registration is stopped;
Start should produce a healthy initialized pier. Test Stop → Start, same identity,
resource display, logs and browser access. The packaged Recommended Apps pill is
still a draft with the documented rc.1 application gaps; test it separately.

Local browser access and remote access are separate. The runner binds ship HTTP to
host loopback. Publish only specifically selected Tailscale routes when remote
browser/management access is part of the next test; do not reset unrelated routes.

## Later: multiple collectors

Keep the sourcebox plugin as one independent collector, and install/enroll a second
collector on sourceport using its own key and profile. Add an observer profile.
Check cross-client state, actor attribution, revision conflicts, disconnect/replay,
role restrictions/revocation and reboot convergence as described in
[the next-phase plan](rc2-next-phase.md). Grants are currently administered on the
host; a second collector does not require a second runner process.

The release's `release.json`, `qualification.json` and outer `SHA256SUMS` accompany
the extracted bundle in `$HOME/omabit-releases/0.1.0-rc.2/`.
