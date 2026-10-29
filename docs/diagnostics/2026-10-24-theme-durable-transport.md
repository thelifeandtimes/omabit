# Theme durable concurrent transport and settings intent

Theme now records watch/poke channels in a private version1 transport journal.
Each operation holds its own file lease; short journal transactions serialize
state changes without blocking a live watch behind an independent publisher.
Creator and cleaner process identity include PID/start/boot/namespace. Explicit
local release is recorded only after caller request/stream contexts exit. Live
leases cannot be taken over, even if a release flag has already been saved.
Verified dead or released owners can be cleaned up with the exact saved session.
Missing/unknown/unsafe state refuses intact. No credentials or settings bodies
are retained; settings intent contains desk/bucket/entry and desired-value digest.

Pokes begin as uncertain before dispatch. Exact positive/negative Gall receipts
are distinct from cleanup. Deleting a channel leaves unknown application work
unresolved and prevents another write to that entry. Scry captures candidates
before its authenticated read; exact matching desired state may record
state-confirmed, never an inferred ACK. Absent or different state and a different
session cannot resolve it. Cleanup request identity survives a dead cleaner and
an accepted delete with no reply. Watch readiness now requires positive subscribe
acknowledgement. Direct plugin-to-Eyre and standalone operation are preserved.

All268 Python tests pass in private namespaces, including24 journal model checks,
real SIGKILL/concurrent-process tests, six new HTTP integration checks and private
installer inventory/import compatibility. The receiver declaration and package
inventory include the new module; earlier checkpoint installer receipts remain
recognized. The first full run retained two failing fixture assumptions (sequential
delete ID and an unjournaled manual channel); corrected cases exercise the actual
saved request contract. No installed plugin or account was updated.

The dedicated runtime uses unchanged actual Theme Eyre clients in separate
processes, an actual private Secret Service and synthetic Eyre. One watch stays
live while a publisher is killed after its settings write is accepted without an
ACK. The cleaner is killed while its accepted delete reply is withheld. A new
client preserves the original cleanup request, closes that channel, opens its own
watch and leaves the first watch alone. Re-publication is refused; a fresh read
records matching state without another poke. Both watches then stop cleanly.
Exactly one settings poke, three closed channels and six original-writer snapshots
are retained. The full real-QML private collection regression also passes all16
channel deletions, activation/rollback and shell replacement while preserving the
coordinator's reader-owner-lost uncertainty. Source/scopes independently verified.

M276 supersedes M275's in-memory-only Theme transport finding for these paths.
It is not yet fresh-Urbit Theme recovery, hosted TLS, native desktop rollout or
independent coordinator recovery authority. Disconnect currently removes local
account/keyring state before coordinating live clients; a safe quiesce/preserve
contract remains necessary. Missing/expired credentials, old-helper adoption and
rollback compatibility also remain. History caps at2048 channels/4MiB and8192 owner
leases, refusing intact rather than pruning; supported archival remains required.
Authentication receipts and transport intent are separate journals. Same-user
Omarchy plugins still share OS authority; this is crash coordination, not secure
wallet isolation. M274's DMs evidence remains unchanged.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-theme-durable-transport`.
