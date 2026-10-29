# UI admission closes before transport drain finishes

The new negative test confirms a real M291 gap: after Theme guard loss, its widget
still exposed its open IPC target. Merely hiding/disabling the inert parent did
not disable the application's own IPC handler.

WidgetHost now supplies a live manageIpc binding for Theme/Omabit panels. Closing
or losing primary admission withdraws their widget/setup handlers. Host closes an
existing overlay on leaving loaded and refuses later open calls. Settings no longer
forward into non-admitted payloads. Omabit Service has a separate uiAdmissionOpen
property, default true for ordinary callers; guarded hosts bind it to loaded.
Closing hides its monitor, and openMonitor/openSetup refuse closed UI or transport.
This is separate from transportStopping: an already-running operation may finish
while new UI actions are closed. Helper shutdown/drain semantics are preserved.

The actual supported-set test passes all prior native cases plus widget target
absence after guard loss and closing, exact Target-not-found responses, scoped
monitor reopen refusal, and DMs overlay close/reopen refusal. An actual inert
Omabit fixture operation is held during close: UI withdrawal and monitor refusal
are verified while transport remains not-requested, then the original operation
is released and ordinary transport drain completes before unload. All88 private
collection tests pass with the final pinned hosts. Source hashes match and all four
scopes are inactive. The failed first positive run had malformed generated fixture
IPC newlines; it is retained separately from the genuine negative control.

This supersedes M291's closing-time widget/monitor/overlay admission gap for the
characterized private fixture. It does not qualify the full native bar/compositor,
all external effects, independent process closure or connected cross-root recovery.
Same-user IPC controls are cooperative lifecycle rules, not wallet key isolation.
No installed plugin, production ship/service, tmux or temp-ur data was touched.
Evidence: /home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-closing-ui-admission
