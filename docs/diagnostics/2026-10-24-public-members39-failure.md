# Public member trial39: engine API unavailable after Start

VM39 shut down cleanly after333.96seconds, QEMU0/outer1, without outer timeout,
log limit or storage failure. Five public initial actions and their lookups pass;
the first public member launch reaches durable `started` before its next engine
identity verification refuses: `private engine API evidence unavailable`.
The guest Docker log records a canceled GET `/version` at that point. The normal
probe has a2-second response-header and5-second overall deadline. Which bound
or parent cancellation caused this exact request is not separately recorded.
No changed engine identity is established. M210's later observation check is not
reached, so this does not determine VM38's specific cause.

Independent stopped-disk audit verifies six new public requests, five lookups,
three duplicate refusals, schema40 and exact journal/receipt mirrors. There is
one project, two members, no ordinary instance and one interrupted exact-ID launch.
Only one member Create and Start occurred. Cleanup exits the early fake process
with143 and OOMKilled false; that is not the fixture's clean Urbit shutdown gate.
No application-state/subscription check was reached. Original disk and all results
are retained; only the hash-verified regenerable raw conversion was removed.

The software-emulated guest advertises two CPUs but used single-thread TCG and a
one-host-core quota. Resource sensitivity during cold boot is a hypothesis, not a
proven production defect. Next is a fresh overlay under an explicit two-core cap
and multi-thread TCG. Production probe deadlines, ownership checks and freshness
limits remain unchanged. This supersedes M211's running status only; no full
public member or recovery pass is claimed.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-public-members39`.
