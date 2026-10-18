# PR #4 integration for RC.10

Contributor: **wundrellama (~dinnyt-divsud)**.
Original source: `ba4ba805221a6ef3e29829c9bda1bfeff0c0c7c2`, based on RC.9 pins.

The contribution adds bounded loom selection, native control for all ship classes,
immutable recorded-launch settings, private socket creation and desktop/CLI controls.
Its original full suite and GitHub CI passed during review. Additional isolated
tests reproduced two omissions: a completed move reset loom34/native-on to31/off,
and the frozen RC.9 collector cleared native opt-in during an HTTP-only edit.

Integration fixes:

- Presence-aware configuration requests preserve omitted native settings while
  explicit false remains representable. Strict unknown-field decoding is retained.
- Settings-aware transfers bind loom/native choices into a v2 signature, persist
  them at the destination, retain v1 historical signatures and reject altered retries.
  Both client and source preflight prevent legacy callers fencing nondefault ships.
- Schema19 fences older preview runners from losing transfer settings. Supported
  prior versions11–18 migrate transactionally without rewriting instance/operation
  bodies. Existing unpublished Ames and retained-import fences survive reopen.
- Release metadata queries the built binary's schema. Runtime-settings model tests
  are included in CI; the desktop reviews transferred settings and capabilities.

No protected pier is a runtime qualification fixture. Final binary/runtime and
local fresh-install handoff evidence is recorded with the RC.10 release.
