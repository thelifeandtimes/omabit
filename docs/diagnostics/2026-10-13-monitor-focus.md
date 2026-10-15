# Monitor workspace activation — 2026-10-13

**Fixed and installed on sourceport as `0.22.3+focus.1`.** The same fix is included
in the development plugin source.

## Cause and behavior

The menubar's **Monitor & maintenance** button previously only set the monitor's
`visible` property to true. An already-mapped window on another workspace was
already visible in that sense, so the button neither switched workspaces nor
focused it.

`Service.openMonitor` now calls `MonitorWindow.present`. A shared
`WindowActivation.qml` helper makes the window visible/unminimized, then locates
its compositor toplevel by **this Quickshell process ID plus exact window title**.
It focuses the resulting exact address, using Hyprland's Lua dispatcher when
appropriate and the legacy dispatcher otherwise. Hyprland selects the window's
workspace/monitor, including special workspaces.

Activation waits briefly for the menubar popup to release keyboard focus and for
a newly mapped window's IPC metadata. Lookup retries are bounded. Closing the
window cancels pending activation. Existing windows are not hidden/remapped or
moved to the caller's workspace, so repeat opening preserves their window identity.

## Verification

The real-compositor test `scripts/test-monitor-focus.py` uses isolated collector
fixtures and invokes the actual menu-button handler. It passed:

- First open maps and focuses exactly one monitor.
- Opening from another workspace focuses the same window on its original workspace.
- Repeated activation reuses that window.
- A monitor on a special workspace is revealed and focused.
- Close/reopen finds and activates the newly mapped toplevel.
- Focus actions issue no ship mutations.

The source and the prepared desktop hotfix both passed. The broader mapped QML and
installer suites also passed. Test windows/workspaces were temporary and the user's
prior active view was restored afterward.

## Desktop deployment

The owned installed snapshot was verified and backed up, then the focused change
was applied to a copy of that snapshot using its existing bundled binaries. The
approved cold-replacement installer installed it as `0.22.3+focus.1`. Initial plugin
enablement confirmation timed out during shell startup; once the shell was ready,
`--enable-only` confirmed it without another replacement/restart.

Live IPC reports the hotfix version. A live check moved focus away, reopened the
monitor, and verified that the same window was focused on its original **workspace
2**. Installed snapshot checksums and the activation component's source match were
verified. The larger schema-14 storage update remains development work; this desktop
hotfix changes window activation rather than runner behavior.
