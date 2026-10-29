# Native appearance command and shell-restart lifecycle

The guarded test-native-appearance-effects.py fixture copies and hashes native
Omarchy theme, template, palette, font and shell-restart scripts, and imports the
actual experimental Theme command runner. It uses a no-network private namespace,
private HOME/runtime and real minimal Quickshell instances. App-retint commands,
font availability, systemd/environment and Hyprland dispatch are controlled probes;
there are no actual Theme/DMs services or production desktop connections in this
narrow test. The native font setter really writes the private fontconfig file.

A direct native theme-set control returns while its background cache is still
alive. Through Theme's command runner, process-group cleanup sends SIGKILL after
return; the cache may never start. The first trial incorrectly required its start
and failed; it is retained. The corrected control distinguishes native lifetime
from command-runner cleanup. A deliberately failed early retint command still
produces native exit0. Native success alone therefore proves neither every app's
retint result nor completion of background work.

The copied font setter and restart script terminate the real old private shell,
wait for it, ask the inert Hyprland dispatcher to launch a fresh private shell,
observe readiness and complete the font hook. A helper owned by the old shell
exits without recording its command result. The same command runner launched
independently survives and records exit0. Both replacement shells are explicitly
stopped. The corrected draft and applied source both pass; all three scopes are
inactive, host entry refuses, and source/input hashes match.

This extends M263 beyond headless staging to native process-lifecycle evidence.
The collection coordinator and appearance restoration must have a lifetime outside
plugin QML, keep durable intent before effects, and verify the new shell/appearance
instead of interpreting old-helper death as completion. The next integration must
run the actual offline Theme restore CLI through this native restart boundary.
App-specific retint verification, actual compositor, auth/monitor/launcher drains
and durable full collection switching remain open. No production ships, installed
plugins, host services, keyrings or desktop configuration were changed.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-native-appearance-effects`.
