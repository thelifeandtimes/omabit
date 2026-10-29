# Queued initial member launch trial14 — 2026-10-23

The fresh schema27 trial failed its HTTP/cleanup gate after 1218.44 seconds.
Independent disk/database verification confirms both members reached committed
running-container receipts, with exact IDs, complete unchanged namespaces, and
no adapter errors. The stale dispatch failures from trial12/13 did not recur.
This is partial evidence, not a complete launch/readiness qualification.

The final fixture HTTP wait expired before a success was recorded. Nec cleanup
exited137 after its grace limit without OOM; zod exited0, the anchor exited143,
and the VM powered off cleanly. There was no outer deadline/quota failure. The
original artifacts, guest disks, receipts and private container logs are retained.
Zod's log announced HTTP; nec's retained log did not. This is consistent with
incomplete cold boot, but does not prove why its stop request failed.

The earlier passing VM08 used serial identity warmup: zod took about405 seconds
and nec585 seconds. Trial14 instead started both before a final480-second HTTP
wait, within a900-second setup/launch context. The next fixture should use the
established serial identity warmup and sufficient cold-boot budget. Keep the
30-second ownership/preflight bounds unchanged. Schema28 readiness source/queue
checks are separate and were not part of this frozen schema27 run.
