# Gateway filesystem observation with retained member piers

M421, 2026-10-26. The new read-only constructor binds the saved ready gateway's
original root, chain, secrets and member-mount identities. Selected members must
have explicit retained pier/.urb identities, unique within the workspace. Each
selected mount contains exactly its ship-named child. Omitted mounts remain empty.
The original initial-start constructor still requires every member mount empty.

Held member verifiers check layout and identities without entering live database
contents. Gateway scans retain the existing chain/key metadata checks and bounds.
This prevents live pier writes from invalidating an otherwise stable gateway
metadata observation, while replacements, unknown children, changed private
metadata and unselected writes refuse without repair. Closing serializes against
observation and closes all held member objects. Neither constructor creates,
syncs, repairs, adopts or starts anything.

All31 new checks and145 combined filesystem checks pass under the race detector.
After integration,633 affected gateway/member checks, vet and diff checks pass
with no skips. Fixtures are synthetic only, including a FIFO beneath opaque pier
state to show the gateway reader does not walk it. This is not a stopped-state
or pier-integrity pass. The caller still needs exact complete container ownership,
namespace binding and the runner lifecycle fence; accepted durable member intent
must supply identities. No schema or public API change, no real pier access.

M421 extends M418 and supersedes only the missing member-aware filesystem helper.
The runtime gateway observer still uses the initial empty-mount constructor;
member-aware engine integration and durable lifecycle remain. This change is
excluded from frozen VM09. That separate M420 runtime attempt stopped cleanly
during fresh host installation, before gateway/key tests; its cause is under
read-only investigation and is not evidence against or for this helper.
