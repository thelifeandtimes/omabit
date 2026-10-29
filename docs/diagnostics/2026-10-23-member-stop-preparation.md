# Concrete graceful Stop and private queue — 2026-10-23

The member observer now returns the complete bracketed container inspections
alongside runtime evidence. An exited process requires an explicitly recorded
start time, exact original container configuration and valid exit metadata.
Original Create/Start readers still reject exited containers. Failed/nonzero/OOM
exits remain observable failure evidence. Running peers and the anchor retain
exact process metadata across a Stop; a completed peer Stop must match its prior
start, finish, exit code, OOM flag and initialized storage.

The concrete adapter reconstructs original launch/readiness bindings, rechecks
complete fresh ownership and pier identities before dispatch, and sends one
explicit SIGTERM to the full recorded container ID. The bounded wait is read-only;
there is no forced termination, removal, download or automatic retry. Lost replies,
timeouts and changed ownership retain uncertainty. Docker's explicit signal option
is documented in its [container kill reference](https://docs.docker.com/reference/cli/docker/container/kill/).
The default signal is never used. A completed receipt is historical evidence,
not authority to restart a pier; namespace recovery remains separate.

The private server queue shares work admission, update-worker accounting and
maintenance fences. Accepted Stop intent and each later phase commit before their
corresponding effect. Completed peer exits accompany the next member's selection.
Actor/request replay does not reopen an adapter or schedule another signal. Setup
and cancellation failures remain interrupted with their accepted binding intact.

Thirty concrete adapter cases, six queue cases, exited-contract/observation cases,
affected lifecycle race regressions and vet pass. Tests cover lost acknowledgements,
delayed exit, no exit, cancellation, changed piers/processes, interrupted commits,
unclean exits and sequential Stop with an already stopped peer. Four ordinary VM
guards skip runtime access; explicit enablement on the host refuses at the guest
marker before any runtime effect.

The guarded fixture now follows both readiness receipts with real queued Stop,
peer first then zod, and requires journal-before-effect callbacks, clean exit,
replay/reopen equality and refusal of a second request for the same container.
Its main operation allows3000 seconds, with Go/guest/service/host bounds of
3750/3810/4200/4500 and separately bounded cleanup. Ownership freshness stays30
seconds. The same no-NIC, one-CPU private VM boundary applies. Appliance16's
sources, artifacts and embedded test are independently verified. Real execution
is pending; VM15's earlier fixture cleanup does not qualify this durable Stop.
