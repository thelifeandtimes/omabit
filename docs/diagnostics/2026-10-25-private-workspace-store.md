# Durable private workspace operations and original41 refusal

Schema42 adds a distinct workspace journal and mirrored operation receipt. The
original private project, member reservations and actor-scoped registration
receipts remain unchanged. Acceptance, checkpoint updates and receipt mirrors
commit atomically. Shared reservation checks validate workspace ownership and
reject inconsistent or orphaned records before further allocation.

Every filesystem callback sees its committed dispatch phase. A lost creation or
layout reply retains the same operation and partial state. Runner reconciliation
marks queued/running operations interrupted without changing the workspace or
scheduling another mkdir. Exact actor/request replay returns the saved receipt;
a new request cannot replace an unresolved workspace. This internal source entry
is not yet wired to a public route or runner worker queue.

Focused workspace race checks pass. The affected managed/API/support-manifest
suite passes16 tests /89 including subcases with race detection; static checks
pass. Tests cover each interrupted phase, cancellation/lost replies, transaction
rollback, corrupt mirrors/ownership, parent rebinding and preserved original
reservations. The full managed race regression is running separately and is not
claimed here.

The unchanged original41 writer reproduces the pinned fixture SHA256
`5da8b5eb7aa65ddf8bca03eb8233cd5bacfc541b97e86693e55b9337942c7ac3`.
The actual current migrator exports schema42 with an interrupted, parent/root-bound
workspace. The original41 reader refuses this populated successor without changing
it. The reproduction/export/refusal run passes in32.10s. Earlier release support
claims remain unchanged; this adds the unreleased41→42 development gate only.

Before persistence was introduced, the M370 prototype's birth-time string was
aligned with existing pierstore's canonical `seconds:nanoseconds` representation.
The focused filesystem tests pass with that representation. This supersedes only
M370's unpersisted timestamp spelling and pending durable-store statement; its
directory/interruption evidence remains retained. M369's original41 archive and
fixture hashes are unchanged. No installed database, ship, plugin or service was
opened or changed.

Next: finish the full store regression, then wire an explicit queued workspace
action and qualify its actual filesystem/restart path. Observation recovery still
requires a separate durable request/receipt before public exposure. Private chain
initialization, keys, gateway/member launch and simultaneous-project runtime
isolation remain later gates.
