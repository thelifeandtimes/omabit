# Actual offline Theme restore across native font and shell restart

The --offline-restore mode composes the actual Theme checkpoint/restore CLI with
copied native theme staging, font-set and shell-restart scripts. A real private
Quickshell instance is terminated and replaced. Systemd/Hyprland dispatch, font
availability and compositor/app-specific effects remain controlled private probes.
Local verified appearance and uncertain sync/application intent are seeded directly;
there is no ship or keyring. The plugin helper code is copied unchanged.

The fixture captures stable, applies a candidate, retains pending/attempted and
applying state, and invokes the restore CLI outside the shell. Native font-set
writes the private fontconfig file, replaces the old shell and completes its hook.
A deliberate error after that native effect makes the CLI exit1 with Needs attention;
the durable receipt remains applying. A fresh exact-request CLI retry exits0 and
records verified. Completed replay also exits0 without further appearance effects.
The sync journal remains byte-identical. A Python audit tripwire refuses Eyre/
credential calls from the restore helper. The replacement shell is explicitly
stopped; the original native fanout and shell-owned/external lifetime controls
also pass. The draft and applied integration pass, all scopes are inactive, and
source, snapshot and retained journal hashes are independently checked.

This supersedes M264's missing actual offline Theme CLI/native restart integration.
It proves the independently owned restore command can survive this boundary and
retain truthful intent after an effect. It does not constitute a full collection
coordinator, full plugin-service restart, actual compositor or app-retint verification,
font rendering/availability qualification, or a production release. Monitor, launcher
and authentication helper quiescence and durable collection activation remain next.
No protected ship, installed plugin, host service, keyring or desktop was changed.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-native-font-restore`.
