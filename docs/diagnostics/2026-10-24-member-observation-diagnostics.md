# Member observation refusal diagnostics

Ownership refusals now name the failed category: invalid recorded selection,
unmatched/unrecorded namespace membership, inconsistent anchor/inventory, or
metadata/namespace changing during the bracketed inspection. An otherwise matching
observation exceeding30seconds has a distinct elapsed-time message. The existing
30-second limit and all ownership predicates remain. Elapsed time retains Go's
monotonic clock; the persisted observation timestamp remains UTC.

The draft project adapter race tests pass262.543seconds, vet passes, and the
identical applied observation/freshness tests pass1.335seconds. Tests retain
changed-container/namespace and duplicate/unrecorded-selection refusals and check
both sides of the existing freshness boundary. No schema or API changes.

The initial draft had an unused import; the corrected compile is retained.
A restricted-sandbox attempt could not establish the fixture's private owner
profile; normal private-directory ownership allowed the inert tests. A result-log
whitespace assertion initially stopped application, and its following test ran
the old source; that baseline result is explicitly retained separately from the
subsequent successfully applied source test. None was a runtime/approval refusal.

M209's VM38 remains failed with an unconfirmed specific cause. These diagnostics
do not retroactively classify it, authorize retrying mutations, or qualify a new
runtime pass. The next fresh VM includes M207's public member-lifecycle fixture.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-member-observation-diagnostics`.
