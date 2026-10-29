# Held first-boot pier capture

M429, 2026-10-26. A read-only capture handle opens the selected empty member
mount before first boot and keeps its filesystem objects held. After durable
Start dispatch, Arm verifies that same mount is still empty immediately before
the caller's single Start effect. It binds the exact launch and observation time.
Capture accepts only the matching acknowledged-started journal, once.

Two fresh, identical process/container-contract observations bracket the held
ship-named pier and `.urb` directory identities. The process must have started
after the last empty observation in the saved gateway namespace. Renamed mounts,
replaced piers, links, foreign filesystems, unsafe metadata, changed processes and
stale evidence refuse without returning partial proof. Failed capture cannot
expose a verified handle or repeat capture. Successful capture retains the held
objects for subsequent checks; Close is idempotent. Caller and observer inputs
are copied so callbacks cannot mutate the accepted selection.

The helper reads directory metadata only. It never starts a process, reads state
payload or keys, scans peer contents, removes a lock, syncs or repairs a pier.
It cannot be opened after boot, used for resume, or substituted for attributed
recovery. Its process observer must inspect the complete actual contract and
kernel namespace. The concrete driver must still supply the common work/update
fence, full gateway/project ownership and fresh key/pill preflight.

All 49 new checks and 1,014 affected gateway/member race checks pass without skips;
vet and diff checks pass. Filesystem tests use newly created synthetic directories,
opaque FIFO payloads, retained renamed objects and inert process observations.
No engine, actual ship, production profile, installed plugin or real key was read.

M429 extends M413/M418/M427/M428 and supersedes only the missing continuously-held
first-boot metadata primitive. The evidence still needs a durable journal binding
before peer readiness or resume can consume it. Full gateway/HTTP readiness,
the concrete runtime adapter/queue, recovery and actual VM qualification remain.
Schema44 and public routes are unchanged; this is not a managed ship boot result.
Freeze the original44 writer before advancing persistent readiness/capture formats.
