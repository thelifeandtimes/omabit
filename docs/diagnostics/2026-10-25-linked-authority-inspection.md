# Read-only linked authority inspection after repeated desktop loss

The new successor_inspection module dispatches authority1 to its unchanged reader
and validates linked authority2 without constructing any active Authority. It checks
publication/root/seal/mirror identities, original successor claim and checkpoint,
verified predecessor cleanup, canonical history prefix and current reservation/
binding/guardian evidence. Original ancestor runtime files may already be retired
because their immutable checkpoint supplies the original proof. Bound canonical
files and the currently inspected authority root must still exist.

Output format2 adds explicit lineage references. Process visibility, current socket
identity, owned closure and remote transport cleanup remain separate. A successful
inspection always reports activation_authorized and transport_cleanup_verified as
false. Missing required links, replacement objects, unknown versions, conflicting
locks or wrong canonical history refuse without creating or repairing files. Missing
optional guardian receipts remain unconfirmed. The public source CLI now dispatches
both formats; its compact output retains the same fields with the explicit version.

18 focused and391 full collection tests pass. Tests use actual owned compositor/
guardian/guard processes with synthetic application observations. The first focused
run's only error was an invalid synthetic new-profile registration sharing existing
plugin IDs; the journal correctly rejected it. The corrected fixture appends a new
generation of the same profile. All trials are retained; original runtime writers,
inspector and persistent formats remain unchanged.

An actual copied native artifact4-to5 continuation also passes its27 controller kills
and five read-only checkpoints: published-empty, committed-live, guards-closed,
compositor-closed and predecessor-runtime-retired. CLI and internal snapshots agree
without changing files. Final inode/hash inventories verify independently, alongside
all original package/continuation evidence. Four scopes are inactive and all three
private compositors exit0. No production resources were accessed; retained exports
exclude private application/session/keyring files and piers.

M323 supersedes M322's missing retained inspection for a linked authority only.
It does not yet implement transport cleanup, a new durable checkpoint or admission
after loss of that second desktop. Those are the next repeated-session gates; public
profiles, installed upgrades, reboot/keyring and real-backend acceptance remain open.

Evidence: /home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-25-linked-authority-inspection
