# Original45 writer with durable member birth

M435, 2026-10-26. Before adding persistent managed readiness or later member
lifecycle, schema45 is frozen as an unreleased source-only writer. The archive
contains 2,762 source files / 31,466,441 bytes, excluding nested historical source
archives. It is 5,962,091 bytes with SHA256
`b366caf7df4c488816cb69b24fbeefb082d6b2618283ab76fe2d721a06496da9`.

Two independent runs of that exact writer produce identical 1,518,255-byte
synthetic fixtures, SHA256
`8908b59a97cad17e10b9cecc3745c207364325c6fab6a3ac5e686a1543f6ed44`.
They contain three private projects and member launches: two completed version2
launches with captured pier/.urb identities and one Start-dispatched interruption,
alongside gateway/workspace recovery and the previous fake-project histories.
There are no engines, live piers, real keys or runtime operations in this fixture.

The preservation race test passes three store opens. Every original table cell,
sidecar and runner identity is unchanged; completed captures and interrupted
intent remain distinct. Exact actor/request replay returns the original receipt,
other actors are refused, and begin cannot repeat a terminal launch. The existing
schema45 migration/corruption tests passed during the initial run; vet and final
diff checks also pass. The initial new test twice selected the wrong synthetic
project/member (shared display label, then assumed zod-first ordering). Those
failed attempts are retained. Inspecting the actual fixture showed the sorted
first member is `workspace-nec`; correcting that test selector passed without
changing the frozen writer, fixture or production code.

M435 supersedes M432–M434's pending original45 freeze only. The next schema still
requires real migration and original45-reader refusal tests when implemented.
The current support window/schema remain unchanged; the new baseline will be
registered in the standard upgrade matrix with its successor. A separate first
managed-member VM is in progress; this fixture grants no runtime qualification.
