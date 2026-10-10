# Disposable comet prefix/recovery qualification accepted — September 29 UTC

The user supplied a passed 6X `comet-prefix-recovery` result and confirmed
management was restored. No destructive qualification rerun was requested.

- Run: `198fec0f0e49df2bcb0215789f2b5f1d`.
- Public identity: `~tocmes-mattec-tarsem-podfed--hosduc-misbec-lacryp-litzod`.
- Completed: `2026-09-29T17:28:24.166636086Z`.
- Initial inventory digest:
  `91a18d5f7d7853399dbda1ce95252f469c8c766d553cfaf1b1832969051594e6`.
- Four isolated rolls and identity-preserving restarts prepared five epochs.
- Live limits and pier metadata were observed. All 17 containers and the owned
  bridge were removed, with resolved create and provisioning outcomes.
- Verified loopback HTTP ports: 32781–32790.

| Roll | Started UTC | Finished UTC | Available RAM / free disk |
| --- | --- | --- | --- |
| 1 | 17:27:04.966913648 | 17:27:06.785624333 | 5.62 / 401.03 GiB |
| 2 | 17:27:13.669773450 | 17:27:15.382795141 | 5.60 / 400.85 GiB |
| 3 | 17:27:23.527817170 | 17:27:25.398052740 | 5.69 / 400.69 GiB |
| 4 | 17:27:32.639854907 | 17:27:35.572607671 | 5.86 / 400.52 GiB |

Prepared inventory digest:
`6fe31848ac1960f155961a52fda677eaf25a58dba504523f8384ff757cce5c20`.
Epochs were `0i0`, `0i363`, `0i482`, `0i557`, `0i592`, with the newest two protected.
All four non-genesis snapshots were 803,831,808 logical bytes.

## Repeated deletion and recovery

1. Cutoff `0i363`: removed genesis (4 files) and `0i363` (5 files), then their
   directories. Retained `0i482`, `0i557`, `0i592` with unchanged immediate metadata.
   Ordinary restart preserved the identity and retained epoch set. Controlled
   current-checkpoint loss followed by isolated latest-epoch replay restored an
   **803,831,808-byte checkpoint**, and the recovered restart passed.
2. Cutoff `0i482`: removed its 5 files and directory, retaining `0i557`, `0i592`
   with unchanged immediate metadata. Ordinary restart, controlled checkpoint loss,
   latest-epoch replay and recovered restart passed again, restoring the same-size
   checkpoint. Both rounds completed with uncertainty=false.

A final initialized-pier replacement preserved the same identity and newest-two
epoch set. Final retained inventory digest:
`8f261c59decb7492f808e7a61118b85e7543b501af5764e05b58064888e411eb`.
Both `0i557` and `0i592` remained protected, each with an 803,831,808-byte snapshot.
Metadata can change across runtime activity; the immediate deletion comparisons
were separately verified before each restart. This is latest-epoch recovery
evidence on the disposable comet, not backup-content or older-epoch verification.

Report:
`/var/lib/urbit-runner/.local/share/urbit-runner/qualification/198fec0f0e49df2bcb0215789f2b5f1d/report.json`.

An independent identity-pinned snapshot confirmed alpha healthy at 2/2 and
comet-check/import-check/panel-check healthy at 1/1. No active work, recovery
requirements or incidents remained; both existing test policies were disabled.
Managed comet pruning integration is the next gate. Production qualification
remains false.
