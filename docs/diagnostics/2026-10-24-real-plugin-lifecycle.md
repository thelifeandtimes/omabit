# Real Theme and DMs helper teardown across private collections

Two fresh no-NIC offscreen runs pass stable → dev → stable → edge → stable
selection with the actual copied Theme Service/client helpers and DMs overlay/
bridge helpers. Every profile uses an explicit qualification ID/label and private
state. Sources are mounted read-only; host HOME, sessions, keyring, desktop and
engine sockets are absent. Each run starts in a1GiB/no-swap/half-core transient
scope and finishes with a clean shell exit and inactive scope.

The unchanged native service-loader methods and overlay Instantiator run in the
fixture. DMs is an overlay, so service teardown alone would not cover its bridge.
Each transition separately disables both kinds, observes both exact old helper
PID/start-time identities exit, then enables the next collection. Ten distinct
helper instances per run are verified. Theme device IDs survive returning to a
profile and differ across all three profiles. Source hashes independently confirm
all Python helpers are unchanged. No Omabit data relay or session sharing is added.

Only the DMs compositor window is replaced with a Rectangle; native controls use
fixture colors/KeyboardPanel, and the host registry is a private fixture. Accounts
are empty, so this is not connected-subscription, pending-write, keyring, appearance,
Omabit-service, full-shell or rollback qualification. Those remain necessary before
introducing a durable collection coordinator.

`scripts/test-real-plugin-lifecycle-offscreen.py` makes this reproducible using
explicit `--theme-source`, `--dms-source` and fresh `--output` paths. Its inner
host guard refuses before starting any process. The applied portable-source rerun
passes; both complete runs and source/host-refusal evidence are retained.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-real-plugin-lifecycle`.
