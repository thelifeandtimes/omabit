# Theme instance-owned hooks and appearance files

The actual unchanged notification command, loaded from a copied dev plugin with
its real metadata, deleted the synthetic stable appearance override in a private
HOME. The corrected command clears only dev's override and writes only dev's
notification. Appearance files/backups and generated theme directories now use
instance-specific names. Canonical stable naming is preserved.

Native Git startup uses separate theme/font hook names and ownership receipts.
It preflights both targets, preserves conflicting/user-edited bytes and resumes a
partial install. A staged private filename gives Omarchy's existing hook installer
the right basename without modifying the checkout. The hook validates its suffix
and invokes only its matching helper and IPC target. Remote-application suppression
is preserved. Invalid or misbound names/receipts refuse without affecting stable.

Applied source passes36 file/hook/sync tests in221ms, three profile QML tests355ms,
eighteen existing QML tests3102ms, and real hook-to-Quickshell IPC. Seven targeted
IPC mutations and six theme/font notifications remain bound to their own instance;
the private shell stops cleanly. All tests use a networkless namespace, synthetic
HOME and inert desktop/notification adapters. No existing desktop or ship is used.
The first baseline fixture could not import a newer test helper; the second runs
the black-box CLI regression against unchanged source and reproduces the actual
cross-profile deletion. Both attempts and all qualified source/fixture bytes are
retained.

This supersedes M195's pending hook/file-routing gate only. Separate file ownership
is not separate visible desktops: applying any Theme profile still changes shared
Omarchy/Hyprland state, and multiple appearance includes can act on one compositor.
The collection switcher must coordinate exclusive activation and restoration;
these experiments continue to require private desktop environments. Native
compositor behavior and full stable/dev/edge switching remain open. No installed
hook, plugin, service or desktop configuration was changed.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-theme-profile-files`.
