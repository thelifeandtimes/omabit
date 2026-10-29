# Initial queued member launch trial 13 — 2026-10-23

The separate fresh trial failed in 725.41 seconds, with clean VM/anchor shutdown,
no outer quota/deadline failure and no reported container OOM. Original disks,
19 extracted receipts and 42 metadata files are independently verified and retained.

Zod launched first with Init=true and committed its running-container receipt.
Nec now passed Create and retained its exact container ID. Start was refused for
expired preflight before the engine effect. Its final inspection is still created,
with zero PID and zero StartedAt/FinishedAt; the interrupted journal remains at
start-dispatched. There was no re-dispatch or bootstrap over an existing pier.

The immutable-artifact refinement therefore advanced the real trial beyond the
previous Create refusal, but did not qualify a complete pair. Full known-container
inspection still makes separate Docker requests for every member in both passes.
Batch those exact IDs while preserving complete inventories, full per-container
contracts and repeated namespace probes. Add bounded freshness ages to the
refusal so future diagnoses distinguish preflight and observation expiry.

Zod's cleanup again exhausted 300 seconds and exited 137 without OOM. Init=true
and zod-first ordering alone did not resolve this early-boot cleanup outcome.
The stored logs show boot activity; no completed HTTP/readiness gate is claimed.
The cause remains unresolved. This does not invalidate the earlier ready-ship
shutdown/recovery passes, and it must not be relabeled as a successful launch.


## Follow-up log and fixture comparison

The retained private `inspection13/member-zod-container.log` announces the public
HTTP listener during boot. This does not establish when it became reachable or
why the earlier stop request failed; the verified exit137 result is unchanged.
The older passing `network_case.py` also awaited each ship's public identity
before booting its peer. The queued launch fixture starts both before its final
HTTP wait. These are diagnostic observations, not a proven cause or a superseding
runtime result.
