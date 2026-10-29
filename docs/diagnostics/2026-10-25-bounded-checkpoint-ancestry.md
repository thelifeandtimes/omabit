# Bounded checkpoint ancestry

The shared ancestry reader validates actual legacy checkpoint1/2 records and their
canonical links iteratively. It never opens an ancestor runtime, constructs an active
authority or takes a second canonical lease. Immutable identities and hashes are
rechecked; the atomic journal remains mutable and valid later appends are accepted.

19 focused tests pass on their first run in a private HOME/XDG/PID/network namespace.
The named scope is inactive. Coverage includes retired runtimes, absent/replaced
parents, lost canonical metadata/locks, unknown versions, changed journal prefixes,
late mutation, exact and exceeded file/byte/node budgets, and repeated-node refusal.
The cycle test injects only the traversal boundary: legacy schemas cannot themselves
represent a cycle. No new runtime or persistent format is claimed.

Limits are64 checkpoint nodes,4096 files including the journal, and64 MiB total
including the journal's reserved16 MiB maximum. Callers may narrow, not raise, those
limits. Stat size is charged before JSON allocation; refusal preserves evidence.
Existing source files and retained immutable packages match M331 exactly. This new
module is not yet wired into admission. Full core/native suites were not repeated
for an additive, unused reader after its focused tests passed.

M332 advances the reusable recovery plan from M331 but does not complete it. Next is
a reusable checkpoint/intent/authority family using this traversal, then three later
sessions with the same formats and package hashes. Production ships, services and
installed plugins remain excluded. No credentials or application state are exported.
