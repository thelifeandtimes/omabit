# Offline appearance checkpoints and interrupted restoration

The experimental Theme helper now captures a private, content-addressed local
appearance checkpoint and restores it using an explicit per-attempt request ID.
The version1 snapshot binds the exact instance, verified profile, frozen theme
files and personal configuration hash; credentials and sync journals are separate.
The native theme-set shared lock keeps observation/files on one generation and
refuses active transitions without blocking. Unknown, edited, linked, oversized
or differently bound state refuses intact. Installer inventory includes the module
and recognizes the previous exact owned inventory.

The restore writes and syncs applying before effects, then verifies effective
profile, current theme files and applied owner before recording verified. An
incomplete request blocks that profile's ordinary sync startup and new restore
requests. Retry reuses the same pair; a completed request verifies without effects.
Pending/attempted publication and candidate applying state remain byte-identical.
Wallpaper is excluded with preserve-current and the native skip-background flag.

All237 Python and73 focused tests pass. The actual Service/helper fixture first
adopts stable, drains exact helpers/channels, captures a checkpoint, and fails dev
after a font effect. It then shuts down synthetic Eyre and the private keyring,
removes the source theme and changes the mutable generated source. The actual CLI
restores offline; a command failure retains applying, startup refuses, and exact
retry passes. Replaying success issues no effects. A second intentional restore
is killed after the native-theme substitute effect; a fresh helper retains the
receipt and completes its exact retry. An audit tripwire forbids network and
credential tools throughout local CLI calls. Seven CLI exits are0,1,1,0,0,-9,0.
The original four-stage online partial-effect and failed-restore matrix still
passes. Host-entry and invalid-mode guards pass; all scopes are inactive.

This supersedes M261's missing offline snapshot/restore prerequisite for these
cooperating helpers. It does not qualify automatic collection switching, actual
compositor/native app-retint/background effects, older helpers, cross-profile
transition fencing or VM power loss. Native source review identifies the next
boundary: templates can regenerate theme files, app-retint runs after the native
lock is released, and font changes restart the owning shell. Those scripts were
read and hashed, never executed on the host. collectionReady remains false.
No production ship, installed plugin, host service or desktop setting changed.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-theme-offline-checkpoint`.
