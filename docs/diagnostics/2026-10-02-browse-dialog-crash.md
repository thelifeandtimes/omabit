# Browse dialog crash — panel 0.18.1

## Evidence

The user reported Quickshell crashing on Omabit's Browse button. Two retained
SIGABRT dumps show the same native file-chooser path:

- PID 1950841, 2026-10-01 17:40:40 PDT (2026-10-02 00:40:40 UTC).
- PID 2938523, 2026-10-01 22:07:29 PDT (2026-10-02 05:07:29 UTC).
- Command: `quickshell -n -p /usr/share/omarchy/shell`.
- The journal logs `../glib/glib/gmem.c:106: failed to allocate 4 bytes` at each
  crash. No kernel OOM-kill entries appear in the inspected interval. At diagnosis
  the host had approximately 5.9 GiB available RAM and 27 GiB free swap; those are
  current readings, not proof of memory conditions at the crash instant.
- Both stacks enter GTK, `libgvfsdbus`,
  `gvfs_dbus_mount_call_create_directory_monitor_sync`, synchronous GDBus message
  serialization, GVariant construction and `g_malloc`, followed by GLib's fatal
  logging/abort path. GDB's all-thread backtrace of the second retained core exposes
  one thread and agrees with the coredump metadata. Some internal frames have no
  symbols; the reason the four-byte allocation failed is not established.

Installed packages: Quickshell 0.3.1-1, qt6-base 6.11.2-3, qt6-declarative
6.11.2-1, glib2 2.88.3-1, gtk3 3.24.52-1 and gvfs 1.60.2-4.

Omabit's two `FileDialog` objects used the platform-native default. The evidence
implicates that in-process GTK/GVFS chooser path. Browsing chooses a path; key-file
reading and ship registration happen only on a separate explicit submission.

## Fix and regression coverage

Both the private-key and boot-pill pickers now explicitly set
`FileDialog.DontUseNativeDialog` before opening. This uses Qt's QML chooser and
avoids the observed native GTK/GVFS path without changing global desktop settings.
The panel patch version is 0.18.1.

`scripts/test-panel.py --windows` now clicks both Browse actions in a mapped
Quickshell fixture, waits with each real dialog open, cancels without changing the
prior form value, reopens and accepts a synthetic local file. Paths include spaces,
Unicode and `#`, exercising URL decoding into the visible form fields. The existing
CLI audit also confirms that browsing causes no additional registration requests.
The test checks the explicit non-native setting so a regression fails before
entering the known crashing path.

Passed:

```sh
python3 -B scripts/test-panel.py --windows
node scripts/test-panel-model.js
python3 -B scripts/test-panel-install.py
git diff --check
```

## Deployment

The owned plugin snapshot was installed and the unlocked shell reloaded. Monitor
IPC confirms `version: "0.18.1"`. Source and installed `KeyBootPane.qml` both have
SHA256 `642b1daff50fb97bafea4e5b3718f118c25f98fa869585bd7e4d3a33c651964f`.
The previous snapshot is backed up at
`/tmp/opencode/omabit-panel-before-browse-fix.tar`.

No new Quickshell coredumps were recorded during fix verification. A post-update
identity-pinned fleet snapshot still shows seven registrations: alpha, bus-lab and
comet-check running/healthy; import-check and panel-check stopped; rus and ssh-check
uninitialized. Configuration/applied revisions are preserved, with no active
operations, recovery requirements or incidents. No host-service update was needed
for this QML-only fix.
