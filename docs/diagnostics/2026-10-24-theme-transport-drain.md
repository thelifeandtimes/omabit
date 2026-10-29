# Theme and DMs explicit transport closure

Theme now stops and joins its appearance/Talon workers and retries exact retained
subscription-channel deletion before issuing closed=true. Failed cleanup keeps the
helper paused with the original bindings for a repeated shutdown request. Workers
still running cannot produce a closed receipt. Appearance state/session are
preserved and collectionReady remains false. Poke-channel cleanup and arbitrary
process-death recovery are not certified by this watch-channel receipt.

All 199 Theme Python tests pass. Model, manifest, 20 native QML tests, real SDK and
hook IPC checks pass. Separate profile tests pass with private stable/dev/edge IPC.
The initial Python regression caught a missing status envelope on normal EOF;
shutdown now extends that existing envelope and reports Transport stopped. The
initial QML runner discovered profile tests without their separate fixture layout;
it now explicitly runs its own plugin test and the profile runner is checked
separately. Failed logs are retained alongside corrected results.

The actual copied Theme Service and DMs overlay/helpers use synthetic Eyre and a
real private keyring inside a no-NIC namespace. Three stable/dev/stable drains
observe all six exact channels deleted and all old helpers exited before candidate
enablement. Theme desktop state hashes remain unchanged. Stable's unknown send
survives dev isolation and stable return with one total poke and no replay. The
shell exits cleanly; exact inactive scopes and unchanged source hashes verify.

This supersedes M246's undeleted-channel observation only for the opt-in drain
path of these two plugins and extends M254. No general switch command, actual Gall
cleanup, compositor/appearance ownership, all mutable effects or Omabit lifecycle
qualification is implied. No installed plugin or production ship was changed.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-theme-transport-drain`.
