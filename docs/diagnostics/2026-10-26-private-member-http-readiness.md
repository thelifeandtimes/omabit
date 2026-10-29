# First-member HTTP identity evidence

M437, 2026-10-26. `MemberHTTPReadiness` and `CollectMemberHTTPReadiness` define a
read-only internal result for the first version2 member with durable birth capture.
The result binds the exact launch digest, instance, operation, selected ship and
fixed runner-loopback `/~/host` URL. Complete gateway/process observations surround
the identity read; both must retain the captured pier/.urb identity, original
process start and network namespace. Freshness is checked before contacting HTTP
and when completing the result. Gateway state must match the recorded generation.
Historical validation evaluates evidence at capture time; it does not grant live
mutation authority later.

The collector checks continuously held-object verification around each observation
and HTTP read, passes cloned launch values to callbacks, deep-copies gateway
snapshots, and returns no partial result after errors or cancellation. It performs
one attempt with no internal retries. Wrong identities, stale/changed observations,
legacy uncaptured launches, incomplete launch state and already-present peers are
refused. The model intentionally describes only first-member HTTP identity. It
provides no Gall readiness, Eyre credential, private Lens authority, second-member
admission, persistence, managed Stop or resume behavior.

54 targeted and 159 affected integrated race checks pass without skips, as do vet
and diff checks. Tests cover altered pier/urb identity, contract/container/restart/
namespace, incomplete or stopped observations, stale/future timestamps, changed
chain state, fixed endpoint/ship, receipt tampering, callback mutation, all seven
callback failure/cancellation boundaries, clone/serialization and legacy/peer
refusal. All drivers are inert; no runtime or database format changes occur.

This advances the source readiness model only and supersedes no pending runtime
qualification. The fresh M436 fixed VM was already built/dispatched before this
helper was added, so its binary cannot qualify this helper. A concrete held-object
observer and separately attributed persistence/queue remain next steps after the
actual first-member boot gate. Original45 is frozen; a future persistent change
must test original45 migration and downgrade refusal.
