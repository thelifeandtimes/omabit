# Omabit 0.3.1 — guided local service updates

Update an existing Omabit installation from the desktop. After updating the plugin,
open **Setup → Local services & updates → Check services → Update services**,
authorize the OS prompt and wait for **Service update completed. Your fleet is ready.**
The desktop activates matching Omabit and collector tools together; the button
updates the existing local runner using its original owner and access keys.

An interrupted or cancelled attempt retains its verified target. Choose
**Check services → Retry service update** to continue. A completed recheck reports
**Local services match this desktop bundle.**

## What changes

- Visible check, update, progress and retained retry controls.
- Existing runner identity, enrollment and access are verified before installation.
- Pinned launcher files already installed on the host are reused only when their
  hashes match; missing or changed inputs use verified HTTPS downloads.
- Healthy running ships and the rootless engine stay running; stopped ships stay stopped.

## Upgrade and support

1. Update **thelifeandtimes.omabit** using Omarchy's plugin updater.
2. Open the normal Omabit plugin, then **Setup → Local services & updates**.
3. Choose **Check services**, then **Update services**, and approve the OS prompt.
4. Wait for verified completion. Confirm the same fleet appears and check again.

No command batch is needed for the supported local owner-user setup. The source
and binary distribution target Linux/AMD64.0.2.0 desktop/services and0.3.0 desktop
with0.2.0 services both passed real-button disposable-host rehearsals, including
authorization denial, retry and unchanged running container processes. Runner
schema21 migrates to47;47 is unchanged from0.3.0. Older writers refuse47.
Users still on0.1.0 should take0.2.0 first. Preserve any pending update journal;
do not reset state or re-enroll piers to resolve an upgrade error.

Remote runners, unrelated command-line copies, Docker/OS package updates and Vere
upgrades are separate. DEV profiles with host setup disabled cannot update the
production host. Theme0.4.0 and DMs0.2.0 remain the companion releases.

See [the complete upgrade flow](https://github.com/thelifeandtimes/omabit/blob/master/docs/guided-service-upgrades.md)
and [qualification and limits](https://github.com/thelifeandtimes/omabit/blob/master/docs/releases/0.3.1/qualification.json).
Human password entry and stock Omarchy update/rescan remain manual integration
checks. The frozen archive's README retains older0.3.0 manual instructions; this
release's guide supersedes those instructions for local button-based upgrades.
The binaries, archive and pill are the exact rehearsed artifacts, frozen at
`a13ca6787735f155ca2a9abf58c54098e010ac53`.
