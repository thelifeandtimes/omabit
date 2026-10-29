# Theme profile UI and actual IPC

The experimental Theme panel derives its IPC address, service lookup and visible
label from the host-injected module ID. The service derives its address from the
validated manifest, retaining urbit-theme for canonical stable and adding .dev or
.edge for those copies. Its handler is disabled until identity injection; an
invalid or cleared identity disables it again. The real shell creates a service
before injecting its manifest, so eagerly registering the canonical address would
briefly collide with stable. Explicit Commons.Color also fixes the color-name
ambiguity reproduced by the unchanged-source baseline.

The new networkless Bubblewrap gate passes on applied source: three profile UI
cases359ms, eighteen existing UI cases3112ms, and actual Quickshell IPC via the
installed omarchy-shell wrapper. Four targeted themeChanged calls reach only their
selected service. Dev/edge have no addresses before injection; clearing dev removes
only its address and stable remains usable. The offscreen shell exits cleanly and
all states remain disconnected. Only inert Python fixture helpers are copied,
never the plugin backend. Host HOME, network, D-Bus and compositor sockets are
absent. Direct host execution of the IPC driver refuses before launch.

The first private native trial was a fixture-layout failure: Quickshell refuses
module paths outside its config root. Copying only Service/Model/fixture-helper
under that root fixes it. Earlier incomplete inert bar properties produced warnings
and were not accepted as passes. All source/fixture hashes, baseline/draft failures
and applied results are retained.

This supplements M194 and supersedes only the pending Theme QML/service-IPC routing
gate. Theme hook paths/installation, appearance-file ownership, actual desktop
application, Wayland placement and whole collection switching remain unqualified.
No installed plugin, existing service, desktop configuration or ship changed.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-theme-profile-ipc`.
