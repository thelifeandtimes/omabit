# Container CPU and layout refinement — panel 0.15

## Implemented

- Shared inline `host / instance / status` context for ship monitoring and compact
  activity cards. Action/date share a row; routine successful audit messages move
  into Details. Activity/Setup do not highlight a previously selected ship.
- Version under the Omabit title, icon-only refresh and no in-app fullscreen
  button/F11/expand action.
- Routine Save enables the policy and supplies stop permission. The two redundant
  checkboxes are removed; resume remains selectable. Saved policy cards retain
  explicit enable/disable actions and revision checks.
- Container CPU and Container Memory charts with unit/timestamp axes, actual-time
  point placement, bounded session histories and replacement/gap handling.
- Epoch history terminology for the retention section.

CPU telemetry uses the existing private Docker stats transport. Two fresh samples
produce the Docker per-core rate; raw cumulative counters remain exact/private.
Missing/reset evidence is unknown rather than zero and does not poison memory or
ship health. No runtime executable, memory/CPU quota, pier path or schedule was
changed by implementing telemetry.

## Checks

Passed full Go tests/vet, targeted race tests, native and ARM64 runner builds,
panel model/installer/mapped-window tests and screenshot review. CPU tests cover
exact subtraction above JavaScript's integer range, 0/0.5/50/100/200 percent rates,
warm-up, missing fields, resets, replacement, duplicate/stale/reversed samples and
CPU-count changes. QML tests exercise enabled-on-save policy behavior, explicit
disable/enable, inline contexts, inactive fleet markers and labeled chart axes.

## Live acceptance

The first desktop authorization prompt timed out without changing the healthy 0.14
runner. The user then ran the service installer in a terminal. The installed runner
hash matches the native build, Hello advertises `container-cpu`, and panel 0.15.0 is
installed and IPC-confirmed loaded. Its monitoring window was left open.

Two fresh daemon observations ten seconds apart confirmed live CPU values:

| Instance | 20:13:12 UTC CPU | 20:13:22 UTC CPU | Inspected quota |
| --- | --- | --- | --- |
| alpha | 0.9611% | 1.0290% | 2 CPUs / 200% |
| bus-lab | 0.0244% | 0.0166% | 2 CPUs / 200% |
| comet-check | 0.7471% | 0.0283% | 2 CPUs / 200% |

Memory observations remained fresh. All three running container IDs were unchanged:

- alpha: `075247b30dca16f9296e12b3d27487f5eb256a8c5e0334985ea2294f47403a73`
- bus-lab: `493addd69a2c0f818fb4f39e309480432ff32b59ecf4dcb62f60af7b7fbc73ee`
- comet-check: `41591f6a22fb23a7cfe3bf9c8da25eb3d3938da90cc695ae1d18b49c335c80ce`

Pre-update fleet: alpha, bus-lab and comet-check running/healthy with their prior
container IDs; import-check and panel-check stopped; rus and ssh-check uninitialized.
The three saved policies remained disabled at alpha revision 1 and comet/import
revision 2. Post-update checks preserved those states and revisions with no active
operations, recovery requirements or incidents. No live routine was saved/enabled
or ship restarted for this gate. Storage paths were not migrated.
