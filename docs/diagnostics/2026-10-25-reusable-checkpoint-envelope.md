# Reusable checkpoint envelope: legacy ancestry boundary

Checkpoint3 retains the completed authority3 snapshot and references checkpoint2
and checkpoint1 without copying either ancestor body. Its original cleanup and
canonical history must still verify. Capture checks the complete bounded ancestry
and remaining aggregate capacity before publishing. A lost publication reply
retains the exact object; retry also works after the original runtime is retired.
Earlier checkpoint writers are unchanged, and the original checkpoint2 reader
refuses3. The common reader now explicitly traverses3→2→1.

23 checkpoint tests and19 ancestry regressions pass in a fresh private namespace.
The first harness mistakenly used discovery on hyphenated filenames and selected
zero tests. Its zero exit is not accepted as a pass. The corrected harness loads
the two modules explicitly and asserts42 cases before execution; its full run
passes. Both scopes are inactive, sources match and exported copied evidence
validates independently. Existing source changes are limited to the new ancestry
reader extension; all prior checkpoint/intent/authority/cleanup writers remain.

This is the checkpoint portion of the reusable family, not repeated runtime
activation. The current implementation accepts authority3 snapshots only. The same
envelope must support the reusable authority contract before repeated sessions are
qualified; do not create another checkpoint version for each desktop restart.
The next intent/admission work must also prevent older intent writers from claiming
the same legacy predecessor after the new family claims it. This cross-version
ownership question is an implementation gate, not a request for user approval.

No full core/native suite was repeated for an unused checkpoint module; the42
targeted checks cover it and the changed reader. Real repeated native sessions,
installed/public profiles and the remaining roadmap gates are still outstanding.
Production ships, installed plugins and services remain excluded. Export contains
source and synthetic metadata only, without credentials or raw application state.
