# Multi-epoch lifecycle acceptance, September 26 UTC

The user supplied the completed `epoch-lifecycle` report for run
`c81cf35049c0b2f6b78b728af5bb9802` from the dedicated-account 6F executable.
Report path: `/var/lib/urbit-runner/.local/share/urbit-runner/qualification/c81cf35049c0b2f6b78b728af5bb9802/report.json`.

All phases passed: engine/image/pill verification, live limits/version probe,
fresh boot, initial epoch inventory, three offline rolls, restart after each,
the final prefix preview, and cleanup. All eight containers were removed with
resolved create outcomes. The four runtime containers recorded private HTTP
reachability. Production runtime qualified remained false.

| Roll | Started (UTC) | Finished (UTC) | Elapsed | Epoch count | Available RAM / disk |
| --- | --- | --- | --- | --- | --- |
| 1 | 00:15:20.686182093 | 00:15:21.471868062 | 0.79 s | 2 | 8.40 / 413.28 GiB |
| 2 | 00:15:27.195503287 | 00:15:27.932435445 | 0.74 s | 3 | 8.48 / 413.10 GiB |
| 3 | 00:15:33.694483461 | 00:15:34.492587711 | 0.80 s | 4 | 8.37 / 412.93 GiB |

Every roll had `uncertain=false` and verified post-roll restart inventory.
Admission values are rounded observations from the console report.

Initial inventory digest: `ed38d2ae9dac709b25fbbc2e2ac2192ffe4667fcd959c15ebe9f09b0815c5366`.
Final inventory digest: `aeb2d24861a34820c55266e5e8c320103413aac5f18c3825836b7c4dee880b2f`.

| Epoch | Logical bytes | Allocated file bytes | Snapshot bytes | Protected |
| --- | --- | --- | --- | --- |
| 0i0 | 212,983,820 | 212,987,904 | 0 | false |
| 0i66 | 803,856,396 | 803,860,480 | 803,799,040 | false |
| 0i101 | 803,856,396 | 803,860,480 | 803,799,040 | true |
| 0i118 | 803,856,396 | 803,860,480 | 803,799,040 | true |

The preview selected **0i0 only**, retaining **0i66, 0i101, 0i118**. Deletion was
unsupported and no epoch was removed. This exercised a three-epoch suffix,
distinct from stock chop's fixed newest-two behavior.

This is operator-reported qualification evidence. The agent independently checked
the restored management API at approximately 00:20:52 UTC: alpha was healthy at
revision 2/2, panel-check healthy at 1/1, both had fresh resource samples, and no
active operation or recovery requirement was shown.

This accepts explicit offline roll, epoch growth, ordinary same-pier restart,
and positive prefix selection on a disposable multi-epoch pier. It does not yet
qualify selected-prefix deletion or recovery after losing the current checkpoint.
