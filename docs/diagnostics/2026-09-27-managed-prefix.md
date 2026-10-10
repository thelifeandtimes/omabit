# Managed prefix pruning acceptance, September 27 UTC

The agent validated 6K through the identity-pinned `workstation` SSH connection
to `runner-f496feed1f7b08f31a5d82b94ac2f5fd`. Preflight confirmed fake alpha healthy
at revision 2/2, no active work, the `epoch-prune-prefix` capability, and RAM
headroom before each stop.

Two explicit roll/resume preparation cycles added epochs **0i33536** and
**0i33561**, producing five epochs. Preparation request IDs were
`checkpoint-6k-{stop,roll,resume}-prepare-{1,2}`; all six operations succeeded.

| Round | Action | Operation | Created (UTC) | Finished (UTC) |
| --- | --- | --- | --- | --- |
| 1 | stop | `op-cfbe8db40876826fd033bdf924b91d78` | 05:09:29.749294957 | 05:09:30.129854606 |
| 1 | inspect | `op-027faf1f9b4954870a2abe2bdc7b6eeb` | 05:09:31.227745344 | 05:09:31.275827806 |
| 1 | prune | `op-e38091b4c9714de6a3504271dfbaf31e` | 05:09:31.611330237 | 05:09:31.705635347 |
| 1 | resume | `op-552beaa063ada5cf267b1791fda18b2c` | 05:09:32.086422117 | 05:09:38.060662341 |
| 2 | stop | `op-1be4e63475e0d2957e64ee277775dd4e` | 05:09:39.579276422 | 05:09:39.964081528 |
| 2 | inspect | `op-86e8ed0ca566237797d585f8d70ffd10` | 05:09:41.034019675 | 05:09:41.074544472 |
| 2 | prune | `op-bc5c6f0b3017ccc4a7a7ac8f77e1d2b1` | 05:09:41.462621511 | 05:09:41.539658027 |
| 2 | resume | `op-5b42ceb7d9b7ed4fd5c8dfddfd52bc71` | 05:09:41.910278291 | 05:09:47.838799168 |

Round request IDs were `checkpoint-6k-{stop,inspect,prune,resume}-alpha-{1,2}`.
All actions succeeded at revision 2.

## Round 1: five to three

Inventory sampled at `05:09:31.273791077Z`, digest
`62cfbc74c2141edff81c84c6a4bf2efda81999a7a7f742986a0fa956fd4b67d9`.
Cutoff **0i22199** selected **0i22185 and 0i22199**, retaining
**0i22213, 0i33536, 0i33561**. Ten of ten files and both directories were removed.
All retained epoch records matched exactly. Selected counts were 775,700,504
logical bytes and 775,708,672 allocated file bytes.

The journaled deletion interval was 05:09:31.650282979–05:09:31.703776441 UTC.
Stopped replay matched the original completed operation. Healthy same-pier
resume took 5.97 seconds, and running-state replay matched again.

## Round 2: three to two

Fresh inventory sampled at `05:09:41.072014254Z`, digest
`2c043ceed4e1f8b27dd2fa72b878e3905afb4e4d239b4a7c0329c1670f2dfc4d`.
Cutoff **0i22213** selected that oldest remaining epoch, retaining
**0i33536 and 0i33561**. Five of five files and the directory were removed;
retained records matched exactly. Selected counts were 389,046,284 logical bytes
and 389,050,368 allocated file bytes.

The journaled deletion interval was 05:09:41.510220372–05:09:41.537873865 UTC.
Stopped replay matched; resume took 5.93 seconds. Both historical pruning requests
were then replayed while running and matched their original full operation results.
The later job did not overwrite the earlier replay result or resurrect inventory.

Final alpha was running/healthy at revision 2/2. Direct proxy-free HTTP at port
18081 returned `~zod`; no incidents or recovery requirements were reported.
Allocated counts are attributed file blocks, not guaranteed reclaimed disk space.
Management-restart and interruption behavior remain fixture-tested in this gate;
the separate 6J run supplied the destructive checkpoint-loss/recovery evidence.

A subsequent read-only, identity-pinned snapshot during 6L preparation reconfirmed
all fourteen 6K operation IDs, request IDs, success statuses and timestamps, the
latest pruning digest/interval, five-of-five file removal, and the retained names.
Alpha and panel-check were still running/healthy. No maintenance was repeated for
that journal check.
