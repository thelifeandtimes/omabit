# Read-only project history view

Setup now links a Project networks pane that reads the chosen runner's saved
reservations and bounded operation history. It uses only pinned `projects` and
`project-status` calls, supports observer access and checks inventory/history
capabilities separately. The selected profile is visible. Interrupted receipts
and older-history truncation are explicit; saved success is never presented as
current ship health. Operation labels use readable action names.

The actual pane/controller and Quickshell Process pass with an inert helper in a
fresh networkless namespace, private HOME/XDG and offscreen Qt. Sixteen reads and
zero mutations cover observer access, wrong-runner/format/JSON/member responses,
changed project revision, overflow, timeout,128-row truncation, empty inventory,
unsupported/offline runners and closure. A delayed old-profile helper deliberately
ignores termination, finishes after the profile changes and is discarded. The
screenshot was inspected. No host runner, engine, desktop or keyring socket is
mounted, and no installed plugin changed.

All source/draft/applied trial evidence is retained. The first fixture failed to
register its copied QML dependencies; the corrected module passes. Initial
retention encountered ephemeral Quickshell sockets and namespace-only symlinks;
regular files are now hash-verified and those entries recorded as metadata without
following them. Source application happened only after that retention correction.

This qualifies the pane/controller with an inert backend, not the complete Setup
shell, native compositor, installation or a real runner connected through the UI.
M205 and M192's actual CLI runtime gates remain separate. Creation and member
lifecycle UI remain to be built after their public runtime gates.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-project-history-ui`.
