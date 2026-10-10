# Repeated-prefix qualification acceptance, September 27 UTC

The user supplied the completed `epoch-prefix-recovery` report for run
`5c72c70d9ed83d9b9f199d48c24c860c`.
Report: `/var/lib/urbit-runner/.local/share/urbit-runner/qualification/5c72c70d9ed83d9b9f199d48c24c860c/report.json`.

All phases passed. Four roll/restart pairs produced five epochs, followed by two
distinct prefix deletions and ordinary/recovery restarts after each. All sixteen
containers were removed with resolved create outcomes. Limits and pier metadata
were observed; production runtime qualified remained false.

| Roll | Started (UTC) | Finished (UTC) | Elapsed | Available RAM / disk |
| --- | --- | --- | --- | --- |
| 1 | 03:03:41.573621040 | 03:03:43.068924604 | 1.50 s | 6.82 / 409.00 GiB |
| 2 | 03:03:49.773484268 | 03:03:51.433097294 | 1.66 s | 6.94 / 408.82 GiB |
| 3 | 03:03:57.970998606 | 03:03:59.386590145 | 1.42 s | 7.12 / 408.66 GiB |
| 4 | 03:04:05.814788678 | 03:04:07.403687145 | 1.59 s | 7.16 / 408.41 GiB |

Prepared inventory digest:
`ee510d0852dfe211d6fcd081125006fb9fe126f3e0c72ccd76f860719c0278dc`.
The epochs were `0i0`, `0i77`, `0i103`, `0i129`, and `0i146`. Each nonzero epoch
had an 803,799,040-byte snapshot.

## Round 1

Cutoff `0i77` selected exactly `0i0` and `0i77`. The four genesis files and five
snapshot-bearing epoch files were removed, along with their directories.
`0i103`, `0i129`, and `0i146` retained their metadata fingerprints:

- `0i103`: `2c49ff0d9f89adc262afdebce22a6e832b39edcfd003bc19afa3677cf95a7bde`.
- `0i129`: `e93c4f604209525fe6b9c96f36459d55a0d8e610859245daeabda911e3f69483`.
- `0i146`: `71f1bd0c05a2d539c49141e275ad56f87ce4f3ed23de20440508b884865ac5ba`.

Ordinary restart passed. The current checkpoint was removed and restored from
the latest retained epoch, yielding 803,799,040 logical bytes; post-recovery
restart passed. Both deletion and recovery uncertainty were false.

## Round 2

A fresh cutoff `0i103` selected only that oldest remaining snapshot-bearing epoch.
All five files and its directory were removed. The newest two remained, matching
their immediately preceding inventory:

- `0i129`: `e93c4f604209525fe6b9c96f36459d55a0d8e610859245daeabda911e3f69483`.
- `0i146`: `3085e939b7329319ec10296526be5cb40883dd973d80348d352285f1a6633765`.

The latest epoch fingerprint differs from round 1 because intervening runtime
and recovery work updated it; comparison was against the fresh round-2 baseline.
Ordinary restart, controlled checkpoint loss/restoration, and post-recovery
restart passed again. Restored checkpoint size was 803,799,040 bytes. No outcome
remained uncertain.

This is operator-reported live qualification evidence. The supplied restored
snapshot, and the agent's independent check at approximately 04:23:37 UTC, showed
alpha healthy at revision 2/2 and panel-check healthy at 1/1, with fresh resource
samples and no active work or recovery requirement.

The bounded 6J gate is accepted. This proves the exercised prefix/repeated-prune
and latest-epoch recovery paths, not every older snapshot's recoverability or
production backup integrity.
