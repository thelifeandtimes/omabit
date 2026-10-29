# First persistent collection journal

The internal format1 journal passes 24 focused tests, repeated in a private
no-network HOME/XDG/PID/mount/user namespace. The actual owned coordinator child
is SIGKILLed with a pending transition; a fresh process acquires its released lease
and resumes. Each forward/rollback boundary is also reopened, including partial
candidate absence, exact callback replay and uncertainty that cannot be reset.
Atomic-write failures before replacement and after directory synchronization are
covered without deleting history. Unknown/unsafe formats and missing prior state
refuse intact. See [the contract and limits](../collection-journal.md).

Four populated original-writer fixtures retain committed, rolled-back,
pending-enable and uncertain-helper histories. Source hashes match the tested
files; the exact transient test scope is inactive. No loader, desktop or ship
effects were performed. This is storage/protocol qualification, not an implemented
collection activator or independent verification of supplied observations.

M269 supersedes M268's absence of persistent transition intent at the model layer.
Binding actual runtime observations and effects to this journal remains next.
Existing bootstrap and runner formats are unchanged; no credentials are retained.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-collection-journal`.
