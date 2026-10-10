# Desktop refinement — panel 0.14

## Changes

- Per-runner groups ordered Galaxy, Star, Planet, Moon, Comet, Fakeship; running
  instances first within each group. Fake names use italic brackets, such as
  `[~zod]`, with local instance names below them.
- Theme-derived tooltip backgrounds, borders, fonts and wrapping. Full comet names
  wrap in the ship heading; compact navigation elides and offers the full name.
- Activity cards lead with the ship identity, then event/status, with runner/date
  in a quieter metadata row. Identity resolution remains runner-scoped.
- Saved maintenance cards on Overview and Routine maintenance, including disabled
  policies: UTC weekdays/time, next local run, downtime/resume, saved feedback and
  explicit revision-bound enable/disable/edit. One policy per ship remains the
  runner's current contract.
- Confirmed stopped/uninitialized containers show N/A for RAM/tasks. Disconnected,
  stale or uncertain observations remain unknown. Historical charts are labeled.
- History review explains epochs and oldest-consecutive-history removal. Journal
  evidence supplies roll-completion dates; unknown dates remain explicit. Newer
  roll results supersede older prune results in the timeline.
- Inspect epochs now tracks the accepted operation through the observation feed
  and opens the bound completed inventory, without choosing a deletion cutoff.
- Last-roll/prune cards use human-readable local dates near the overview metrics;
  full reports live under Technical details.
- Full-window Setup and prominent Copy +code controls with ship-bound clipboard
  feedback. Secrets stay inside the helper/clipboard path.

## Verification

Passed: full Go tests/vet, targeted race checks (managed store, client/protocol,
collector and desktop helper), native builds and Linux/ARM64 runner cross-build;
panel model, installer, and mapped-window QML fixtures.

The QML fixture exercises policy save/enable/disable, inspection completion and
automatic result review, full-window Setup, +code routing, fake/sidebar identity
labels, stopped resource presentation and comet title bounds at narrow widths.
The CLI audit checks exact mutation counts, unique request IDs, explicit policy
revisions and no secrets in arguments. Backend history tests include successful
rolls outside the activity window and reject foreign-pier, failed and inherited
report evidence without modifying saved runtime state.

After the user unlocked the session, app-window screenshots were reviewed for
comet wrapping, activity hierarchy, policy cards, history review, Setup and themed
tooltips. That review also led to word-aware wrapping and deferred scrolling after
section layout. The screenshot fixture refuses a locked session instead of treating
a lock-screen capture as UI evidence.

## Live deployment

- The user authorized the management/SSH service update. The rootless engine and
  ship containers were preserved. Installed runner/collector hashes match the
  native builds.
- Panel 0.14.0 is installed and loaded; monitor IPC confirmed that version and the
  full-window Setup page. The monitoring window was left open on Overview.
- The live snapshot now supplies alpha's historical roll timestamps and all three
  currently retained comet boundaries (`0i7082`, `0i7847`, `0i26152`). The latest
  comet roll is recorded at `2026-10-01T01:42:07.024389489Z`; its three-entry result
  supersedes the older two-entry pruning summary.
- Fresh running container IDs were unchanged:
  - alpha: `075247b30dca16f9296e12b3d27487f5eb256a8c5e0334985ea2294f47403a73`
  - bus-lab: `493addd69a2c0f818fb4f39e309480432ff32b59ecf4dcb62f60af7b7fbc73ee`
  - comet-check: `41591f6a22fb23a7cfe3bf9c8da25eb3d3938da90cc695ae1d18b49c335c80ce`
- The seven registrations retained their intended running/stopped/uninitialized
  states and policy revisions, with no active work, recovery requirements or
  incidents after deployment.
- A live +code read on `bus-lab` passed runner, instance, revision, container and
  code-format checks. Its secret stayed in the validation subprocess's memory and
  was absent from the subsequent snapshot/audit. Clipboard routing is covered by
  the real QML/helper fixtures; this check did not replace the user's clipboard.

## User state preserved during development

At pre-update inspection, `alpha`, `bus-lab` and `comet-check` were running;
`import-check` and `panel-check` were intentionally stopped; `rus` and `ssh-check`
were uninitialized. The user's newly saved alpha pack policy
`3706427a6c1b5af3ec5027331b6dbb44` remained revision 1, disabled, Sunday/Monday at
02:00 UTC with a 15-minute window and explicit stop/resume settings. No live policy
was enabled and no ship lifecycle or history-removal operation was issued by this
refinement's tests.

Storage locations and the distinction between Omabit's SQLite management journal
and each Urbit pier's own state are documented in [the install guide](../install.md#where-data-lives).
