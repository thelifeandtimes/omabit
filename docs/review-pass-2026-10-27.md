# Manual review follow-up — 2026-10-27

The user resumed the platform goal and requested another fresh labeled VM for
manual review. Exported feedback is preserved in
[the original JSON](diagnostics/2026-10-27-human-review.json).

## Acceptance list

- Setup cards wrap at narrow panel widths; their height follows content.
- Normal creation/import can join an explicit project fake network. Prove two
  disposable fake peers communicate and survive restart. A Docker bridge alone
  does not satisfy fake Ames routing.
- Application Connections lists fleet ships with usable HTTP endpoints and
  declared installed Urbit plugin integrations. Plugin entry opens this pane
  with the requested plugin selected. Plugins retain standalone direct login.
- Compact rows separate approval from connect/disconnect. Show an endpoint once;
  hide helper digests and request identifiers from the ordinary interaction.
  Retrieve managed ship credentials only inside the collector/runner flow.
- Hosted ship enrollment and first login live in Remotely Hosted Urbits.
- Fix Theme's local sign-in removal after the broker closes its child session;
  verify it for fake and hosted-origin fixtures without using the reviewed live ship.
- Separate configurable minimum/recommended resources for new provisioning and
  initialized starts. Preserve stronger maintenance admission and late rechecks.
- Produce one new fresh review overlay from a credential-free base, label all
  experimental plugins, include a focused walkthrough and exportable feedback.

## Preservation and budget

The human-reviewed portable overlay shut down cleanly (QEMU exit 0, not forced)
after exporting feedback. Preserve it and its backing base. It now contains a
human-added live ship login and must not be cloned into the next fresh review or
used by automated tests. The original populated VM also remains stopped.

Before new VM work the retained roots contain 38.48 GiB of the 40 GiB allocation
cap. Reuse immutable inputs; classify superseded generated artifacts before any
removal. Do not generate repeated full evidence disks. All protected host ships,
tmux, ~/temp-ur, installed plugins and production services remain excluded.

## Follow-up qualification

The narrow Setup layout, ordinary managed-ship assisted login, independent Theme
and DMs sessions, Theme cleanup, and new/existing resource-policy restart behavior
passed isolated guest checks (M454). Pair messaging, persistent application state
and subscription recovery passed for the earlier prepared pair (M456).

Fresh-first project creation exposed an intermittent engine-evidence read refusal
under ship boot load. The bounded-probe change and real QML project navigation fix
passed a fresh-first successor run in M461: both ships initialized and answered
HTTP identity checks in 347.97 seconds, without replaying a mutation. Saved
project reservations now appear in the fleet and route to project history, without
inventing live health. Project-member assisted authentication, simple lifecycle
buttons and importing an existing fake into a project remain unfinished.

The actual panel/monitor fixture now verifies visible project navigation while
switching stable/dev/edge profiles and draining helper processes. This supplements
the earlier model-only and project-pane tests. The original human feedback and
credential-bearing review disk remain untouched.

## Review 2 handoff — October 28

Ready in the existing localhost:6081 viewer, with **OMABIT REVIEW 2 · OCT 28**
in the bar. Goal paused. F3 opens the updated guide, F4 opens Omabit, F6 toggles
window expansion; guest password `omabit-review`, viewer password `review26`.
The guide includes separate feedback fields and export for this pass.

Test in order: first host setup; ordinary fake boot/restart; Theme/DMs approval,
reverse entry and independent disconnect; resource policy CLI; optional import;
then a connected project with ~zod and ~nec. Stop the ordinary/imported ships
before project creation. Project assisted login, simple lifecycle buttons and
existing-pier project import are explicitly unfinished. The desktop and import
fixture are provided; native prerequisites are preinstalled, so this is not a
clean Omarchy OS installation test.

M462 records exact preservation and storage details. The earlier human review
remains intact. The accepted automated pair's generated disk was removed after
compact evidence retention, making room for this fresh human review. The current
review overlay is now reserved for the user; do not use it for automated tests.
