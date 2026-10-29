# Public coordinated recovery actions

`project-recover PROJECT start|observe|continue` now connects the public CLI and
HTTP routes to the existing recovery queues. Each action requires its own
advertised capability, operator access, a pinned enrolled runner, explicit
confirmation and the reviewed revision/history digests. Observation and continuation
also require the exact retained parent, checkpoint digest and explicit sequence.
The start action can interrupt every member; observation records one uncertain
outcome without repeating its effect; continuation separately authorizes new work.

A lost reply keeps the complete original request for `--replay-only` lookup.
Lookup does not create a missing receipt or schedule a worker. Unknown-ID Create
remains review-required. Missing or mismatched user-selected parent receipts are
ordinary refusals after global stored-history validation; actual corruption still
fails closed. The existing schema40 and historical request fingerprints are unchanged.

Source checks pass: managed initial154.260s, additional refusal27.069s and related
existing journal/replay regression378.406s under the race detector. The first
sandboxed trial could not create fake Unix sockets for client/CLI tests; their
separate temporary-socket rerun passed. Applied client1.055s, protocol1.008s and
CLI1.028s pass with vet, diff and operator-skill validation.

The built development collector also passes62.226s against inert temporary Unix
runners for all three actions. It obtains review inputs from public reads, loses
an accepted reply, retrieves the original queued receipt with lookup-only, completes
one worker, then retrieves the terminal receipt and saved history without repeating
backend effects. Evidence is retained at:
`/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-public-recovery-actions`.

This supersedes M188/M189's pending public recovery source wiring. Frozen VM34
predates these commands, so neither it nor inert CLI tests qualify public Docker
mutations. The next draft adds explicitly guarded actual-collector recovery to a
fresh no-NIC VM. Full public lifecycle runtime coverage, QML, OS runner-service
restart and remaining roadmap work stay open. No installed service or plugin changed.
