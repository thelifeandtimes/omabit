# First replacement VM preparation

The guarded schema32 fixture now queues one replacement after the original zod
container has cleanly exited and been retired. It preserves the original receipts,
checks dispatch phases, replay and reopen, verifies the replacement identity over
HTTP, and records stopped state hashes after a clean replacement shutdown.

Host refusal, compilation and vet pass. Appliance19 independently verifies145
source files and the embedded test binary
`2521c346bca70e3d96feab8efe3cd5e725b4bbb42011a0fd143dd981ca226599`.
Verified staging was removed. The VM has not yet run. This is the initial generation1
to2 case only; durable readiness, general generations and application recovery
through this runner path remain separate gates.

The runtime uses a fresh no-NIC QEMU guest and private rootless Docker. Production
ships, tmux, temp-ur, installed plugins and the host Docker engine remain excluded.
