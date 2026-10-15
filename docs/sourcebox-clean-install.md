# Fresh RC.3 installation on sourcebox

This walkthrough uses **Omarchy's native GitHub plugin installer**, with no manual
repository checkout, Go compiler or development-path override. Shared OS prerequisites remain
installed. The previous app-owned host/desktop state must already be retired; this
document does not perform a teardown or restore old piers.

## First step: install only the desktop plugin

In an ordinary terminal **on sourcebox, inside its logged-in Omarchy desktop**:

```bash
omarchy plugin install https://github.com/thelifeandtimes/omabit.git --enable
```

`omarchy plugin install` is an alias for `add`. Do not use sudo. Omarchy performs
its normal trust confirmation, clone, manifest validation, bar-placement selection
and enable flow. On first use, the plugin downloads the pinned RC.3 `omabit`,
`urbit-collector` and `urbit-runner` executables, checks their hashes and caches them
under `~/.cache/omabit/native-binaries/` (or the XDG cache equivalent).
Allow a few seconds for the initial download. This does not create a runner host
or start a ship. **Pause here to inspect the new-user UI.**

If cloning succeeded but enabling failed, use
`omarchy plugin enable thelifeandtimes.omabit` after inspecting the error. For a
binary-download diagnostic, run
`~/.config/omarchy/plugins/thelifeandtimes.omabit/native/bin/urbit-collector --help`.

## Second step: use the normal first-time host setup

Open the Omabit bar icon → **Setup → Set up this computer**. Step through:

1. Prerequisite checks.
2. Creation of a fresh desktop management key.
3. Download/verification of the rootless host helper.
4. OS authorization for the existing desktop user's separate rootless engine,
   services and user-owned `/srv/urbit/piers/` root.
5. Pinned local SSH enrollment and the Ready view.

The terminal equivalent, when explicitly chosen, is
`~/.config/omarchy/plugins/thelifeandtimes.omabit/native/bin/omabit --format text setup`.
Do not run both setup entry points concurrently. Record the first error rather
than removing additional files to retry.

Acceptance: a new runner ID and management SSH identity, schema 16 capabilities
including `owner-user-runtime` and `human-pier-storage`,
empty ship inventory/history, and enabled/active services with lingering. A later
collector should enroll this new identity; the old runner pin must not be reused.

## Third step: first ship

Use **bare bootstrap** for a disposable fake, then a comet. Registration is stopped;
Start should produce a healthy initialized pier. Test Stop → Start, same identity,
resource display, logs and browser access. The packaged Recommended Apps pill is
still a draft with the documented rc.1 application gaps; test it separately.

Also test `|mount %base`, ordinary user copy/editor saves into the pier's `base/`,
and `|commit %base`. Repeat with a custom storage location under your home. No
service-account chown or ACL preparation is needed. See [desk editing](owner-user-runtime.md).

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

The previously staged binary bundle can remain unused. This native-install test
does not depend on it. `native/bootstrap/release.json` records the frozen tool pins;
the checkout's Git revision records its UI source. See [native distribution](native-plugin.md).
