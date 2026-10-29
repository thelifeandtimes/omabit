# Guided source-restoration UI

The guided restoration UI is implemented and passes isolated offscreen testing.
This follows the [real CLI/VM checkpoint](2026-10-22-import-restore-runtime.md);
driving the UI against a real runner is the next distinct gate.
[Exact source and test evidence](2026-10-22-import-restore-ui.json) is retained.

The storage page offers **Restore original source…** only for a runner-indicated
candidate. Its review presents the original/held source, partial target, archive,
file/byte counts and whether this continues a recorded restoration. Explicit
consent is required. Submission closes that review and retains its exact request;
an uncertain response cannot enable another restoration. Completion receipts live
outside instance delegates, so retiring the failed registration does not erase
the result. **Inspect restored pier…** starts a fresh guided preview; it does not
import or start anything. Activity also shows retained paths in the runner's
successful restoration history after the UI process restarts.

The collector now includes the authenticated public management `actor` in fleet
observations and restoration preview output. Optional `--expected-actor` on both
restoration commands rejects a credential/actor change before dispatch. The UI
uses it and binds receipts to actor, runner, instance, revision, request and the
reviewed moves. A different fleet operator's receipt cannot resolve its request.
The operator reference documents this binding. No persistent format changed in
this checkpoint; backend/collector changes still require the planned minor release.

Validation:

- Both affected Go packages passed their complete race suites and vet. Tests
  exercise actor refresh, runner replacement and CLI mismatch refusal before
  either the preview or mutation endpoint is called.
- The standard model suite now includes restoration checks for wrong paths,
  bounds, authority, stale revisions, missing consent, duplicate clicks, lost
  acknowledgements, unrelated actors/requests, retired/reused instance names,
  failure/continuation and exact terminal receipts.
- The actual `StoragePane`, restoration controller/review and verbatim service
  callbacks passed in Bubblewrap with private HOME/XDG, no host home/run sockets,
  no network access, offscreen Qt and inert command dispatch. Nine recorded
  calls comprise five restoration previews, three explicitly approved restoration
  requests and one fresh pier preview. There is no automatic import or Start.
- Six rendered states are retained in `/tmp/omabit-restore-ui-20261022-03`.
  The final review and success images were inspected: paths and consent are
  visible, success survives removal of the instance card, and stale global errors
  are cleared. Qt6 parsed all four changed QML files without reformatting source.
- Operator skill validation and `git diff --check` passed.

A strengthened regression exposed a late callback downgrading cached success
after the runner disconnected. The controller now preserves terminal proof even
when no fresh snapshot can repair it. An earlier screenshot also exposed an old
global error below a successful receipt; restoration errors are now scoped to
their own review/request. Earlier evidence is retained. An initial syntax check
selected the system Qt5 formatter and failed; the explicit Qt6 formatter passed.

The fixture uses inert style values and calls QML control signals. It does not
claim native Omarchy interaction, live UI-to-runner transport, or a production
install. All new test processes exited. Existing ships, tmux, `~/temp-ur`, installed
plugins, desktop sockets and production services were untouched.

Next: freeze these UI/collector artifacts and drive preview/approval/restoration
through the actual collector against a fresh fake ship in the private VM. Keep
that guest's source/partial-copy checks and explicit Start/Stop assertions from
the CLI case. Additional import failure modes and the remaining platform roadmap
are still open; the full goal remains active.
