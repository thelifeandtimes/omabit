# Managed gateway observation remains unresolved

The fresh supervised VM with M405's observation-window correction failed the
managed initialization fixture after 1,294.8 seconds. The outer VM did not time
out and shut down cleanly. The independent read-only audit preserved both disk
and pristine-base hashes. All owned runtime/audit units are terminal, and the
disposable disk is privately retained with a verified compatibility path.

The initializer published GATEWAY_READY; its container lived 598.986 seconds and
exited 0 without OOM. The manager retained the original `started` gateway, three
operations and no recovery observation. The fixture reports its operation
deadline. Recovery and the third-process replay phase never ran.

The separate one-CPU/2GiB auditor's first full input scan succeeded in 95.963s.
Its second scan was interrupted after 4.033s by the shared 100s benchmark limit;
this is not a successful warm-scan measurement. The runtime uses two vCPUs/7GiB.
These timings show variability, not the actual failing observer stage or proof
that one timeout explains the failure. No key/runtime-chain payload was read.

M407 supersedes M405's pending-runtime statement only. The M405 source regression
still passes; its runtime correction is insufficient to qualify managed readiness.
No retry, timeout increase, cache shortcut or weakened check has been introduced.

A logging-only overlay now wraps 25 existing observer operations without changing
their conditions, return values, order or deadlines. It emits only fixed stage
names, elapsed milliseconds, success and expiration booleans, excluding arguments,
values and error text. All 355 existing gateway race checks and vet pass with the
overlay. Production observer source is unchanged. The next fresh diagnostic VM
must identify the actual refusal; no diagnostic runtime result exists yet.

M406's pure member command helper is independent and was not in this frozen VM.
Managed member lifecycle, supported input delivery and public orchestration remain
open. Protected ships, tmux/temp-ur data and installed plugins remain excluded.
