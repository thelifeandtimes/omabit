# Private member launch state machine

M427, 2026-10-26. The initial keyed-member launch model binds the exact ready
gateway, selected member/runtime/operation, pinned pill, private accepted-key hash,
peer references and returned artifact/container/process identities. Every effect
has a prior commit; known returned objects are committed even if the adapter also
returns an error. A lost acknowledgement, cancellation or uncertain commit cannot
authorize another boot. Completed or partial state cannot re-enter execution.

All91 new checks and965 affected gateway/member race checks pass without skips;
vet and diff checks pass. Tests inject failures before/after every commit, driver
failure and cancellation, malformed/stale evidence, object aliases and input
mutation. All drivers are inert; no engine, ship or production data was accessed.

This is the model/controller only. SQL persistence, original43-to-successor
migration and downgrade refusal, durable peer readiness, first-boot pier capture,
the concrete driver and actual managed runtime remain. Running process evidence
is not HTTP readiness, clean Stop, retained-pier resume or permission to delete.
Schema43 and public APIs remain unchanged. M427 extends M412/M425; it supersedes
only the missing launch state machine, not their outstanding integration gates.
The earlier M426 VM exercised key proof with zero member boots.
