# Concrete first-member HTTP observation

M438, 2026-10-26. The read-only runtime factory now composes the M437 identity
collector with continuously held pier/.urb, boot-pill and key verifiers, the
retained gateway Docker client, and complete member-aware gateway observation.
The caller must hold the common management/update fence across Open, collection
and Close. Only the captured version2 first-member launch is supported.

Each runtime observation brackets the complete gateway/chain/engine/mount
inspection with fresh full member-container and actual kernel-namespace checks.
It retains the original process start, contract, captured pier and gateway
generation. One exact loopback identity read is allowed after observation; wrong
endpoints, stale evidence and repeated reads are refused before HTTP. The ordinary
HTTP reader has no proxy, cookies, redirects or credentials. Partial construction
closes retained handles; Close is synchronized, idempotent and preserves errors.
No repair, key proof, ship control or pier adoption is performed.

64 targeted and 225 affected race checks pass without skips, plus vet and diff
checks. Inert tests cover exact collection, changed process/container/namespace/
contract, stale/future observations, changed chain, wrong identity/endpoint,
unobserved reads, held-object failures, cancellation, changed selection, invalid
factory inputs before engine access, no partial/private error result, deep-copy
ownership and concurrent Close. Existing launch, observer and pure readiness
tests are included. These source checks do not certify the real filesystem/engine
composition; the actual readiness runtime gate remains open.

M438 supersedes M437's pending concrete-adapter implementation only. Schema45 is
unchanged. No persistent receipt, scheduler, public route, second-member admission,
managed Stop or runtime qualification is added. The currently running fixed
member-boot VM was built before M437/M438 and cannot qualify either. Original45 is
frozen for the upcoming persistent observation/queue migration and refusal gate.
