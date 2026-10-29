# Private keyed-member container contract

M412, 2026-10-26. The pure command helper now has an internal container contract
that binds the reviewed plan, accepted workspace/gateway, exact member, runtime
token, operation, generation, pinned image and retained pier identity. The
contract clones caller-owned metadata and hashes its selected launch arguments.

Members join the selected gateway namespace, publish no additional ports, use
only their own writable parent mount, and run with a read-only root filesystem,
bounded tmpfs/resources, dropped capabilities and no automatic restart. Initial
boot adds only the selected read-only key and pinned pill. Resume omits those
mounts and all creation inputs; it requires a distinct retained pier and later
container generation. Exact requested/effective mounts, labels, image, process
state and network settings are checked. Docker's optional effective tmpfs entry
is supported while its exact HostConfig remains mandatory.

All77 targeted race-enabled checks pass with package vet and diff checks. These
include command regressions, both members/modes, changed ownership/namespace,
mount/input tampering, malformed inspection, aliasing and process-exit identity.
Nonzero/OOM exit metadata remains failure evidence; it cannot prove a clean Stop.

This is source-only construction and inspection. It grants no engine dispatch,
storage adoption or freshness; the saved gateway receipt is insufficient to
launch a member. Held-file verification, member-aware gateway observation,
durable lifecycle, acknowledged shutdown and new managed VM qualification remain.
Schema43, public routes, installed plugins and production services are unchanged.
M412 extends M406 without superseding prior runtime evidence.
