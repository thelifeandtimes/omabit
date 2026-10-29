# Project creation uncertainty and fresh-anchor evidence — 2026-10-23

Review found that the Docker wrapper discarded all output on client failure,
preventing the controller from retaining a complete returned resource ID. The new
create-only path preserves just an exact64-character lowercase hex ID, including
when cancellation follows the response. The controller commits that ID and stops;
it does not inspect or replay an uncertain creation. Missing/malformed output stays
unresolved dispatch. Ordinary Docker error output remains discarded. Race tests
exercise successful creation, lost replies, IDs with errors/cancellation and
malformed output, along with affected runtime/redaction regressions; vet passes.

Fresh stopped-anchor inspection also requires Docker's `created` state, zero PID
and never-started/finished timestamps. A previously started then stopped anchor
cannot satisfy initial preparation. The tmpfs decoder refuses unreviewed option
fields instead of ignoring them. Source fixtures cover these refusals.

Appliance23-07 is prepared but unbooted. Its31 source hashes, artifacts and embedded
guarded test binary were verified; only the regenerable staging tree was removed.
It predates the changes above. Its future happy-path receipt must be kept under
its original source identity and can additionally be revalidated by the current
pure inspection test, which makes no engine calls. That optional test compiles
and skips without an explicit retained receipt; no actual-engine pass is claimed.
