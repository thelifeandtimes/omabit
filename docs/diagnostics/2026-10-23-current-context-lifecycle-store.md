# Current-context lifecycle persistence and scheduling

The private runner queue now carries a fresh current-context readiness receipt
through Stop, exit observation, container retirement, same-pier replacement and
replacement readiness. Historical receipts retain their original context. New
admission requires the current membership and exact observed process start time.
No public mutation API or runtime qualification is claimed by this source gate.

Unreleased schema37 fences earlier readers from this new Stop authority. The
original schema36 writer reproduces its populated fixture, and its original reader
refuses actual schema37 state containing current-context Stop intent without
changing the database. Existing historical migration coverage passes in4.332s.

Both original and replacement targets pass persistence, restart/replay, wrong
process evidence refusal and foreign-member prerequisite rejection (race42.264s).
The private queue passes both targets through successful and lost-reply paths,
reopens their full histories unchanged, and refuses missing backend capability
before admission (race80.716s). A lost Start stays interrupted; it is not replayed.
Vet passes. The broader managed race suite remains running.

The source compile shadowing failure and stale manifest-target test failure are
retained alongside corrections. Product validation was not weakened. All source
and log hashes are in the paired JSON and the durable evidence directory
`/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-23-current-context-lifecycle-store`.

This supersedes M130's pending store/queue integration only. VM25 was frozen at
schema36 and is a separate read-only readiness experiment. Runtime validation of
the new lifecycle requires a new appliance. Production ships, installed services,
user tmux ships and `~/temp-ur` are excluded throughout.

Full regression follow-up: managed race exited0 in972.505s with every recorded source hash unchanged. This supersedes the pending broad-regression status above. VM26-only fixture additions are checked separately; no runtime result is implied.
