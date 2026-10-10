# Prefix deletion and latest-epoch recovery acceptance, September 26 UTC

The user supplied the completed 6G report for run
`e31441cd9345af32611b0153afc0cff7`, profile `epoch-prune-recovery`.
Report: `/var/lib/urbit-runner/.local/share/urbit-runner/qualification/e31441cd9345af32611b0153afc0cff7/report.json`.

Every phase passed, including exact prefix deletion, ordinary restart, controlled
current-checkpoint removal, pinned `play --full` recovery, and post-recovery restart.
Cleanup completed: all eleven containers were removed with resolved create outcomes.
Live limits and pier metadata were observed; production runtime qualified stayed false.

| Roll | Started (UTC) | Finished (UTC) | Elapsed | Available RAM / disk |
| --- | --- | --- | --- | --- |
| 1 | 01:58:29.403996235 | 01:58:30.196720446 | 0.79 s | 8.43 / 411.07 GiB |
| 2 | 01:58:35.896871979 | 01:58:36.645742975 | 0.75 s | 8.40 / 410.90 GiB |
| 3 | 01:58:42.418102426 | 01:58:43.184094921 | 0.77 s | 8.08 / 410.73 GiB |

All three post-roll restarts passed. The pre-deletion inventory digest was
`430b112fb29ac224ae48634878f43e6ca2728c9124429f366f12c146403f2514`.

The selected prefix was **0i0 only**, with 212,987,916 logical file bytes and
212,992,000 allocated file bytes. Its four expected files and directory were
removed. The three retained epochs matched their pre-deletion metadata fingerprints:

- `0i76`: `6f5c7fe3567551d7b9068510260e4707261a8beef8c75d262c0e5557b2a5de03`.
- `0i102`: `88cba4bcbf78325d2698160a8a4ae0e8b4d41a1f4b374b8affc489ebd6f537ed`.
- `0i128`: `9e826b5bbe6e190198e086528bf45019ef31f046f32f6408b2baf693e7ffbf42`.

Each retained epoch had an 803,799,040-byte snapshot. Ordinary post-prune restart
passed. The recovery test confirmed removal of the disposable current checkpoint,
restored **803,799,040 logical bytes** from latest retained epoch **0i128**, and
passed another private-HTTP restart. Both deletion and recovery uncertainty were
false. Allocation and fingerprint measurements do not assert reclaimed space or
full backup integrity.

This is operator-reported live evidence. The agent's subsequent management query
returned an SSH/RPC EOF; no `urbit-runner` management process was found, while the
SSH listener remained available. Separate direct HTTP requests to alpha:18081
and panel-check:18085 both returned HTTP 200 and `~zod`. Management had not yet
been restored; this is not evidence of a ship failure.

The bounded 6G gate is accepted. Managed pruning, non-genesis/multiple-epoch
deletion, older-retained-epoch recovery, and live interrupted-deletion recovery
retain their own gates.
