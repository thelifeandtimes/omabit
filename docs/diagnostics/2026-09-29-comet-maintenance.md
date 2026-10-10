# Disposable comet maintenance accepted — September 29 UTC

The user supplied a passed 6T `comet-offline-maintenance` report.

- Run: `5418c87ccd3aac4ef8a1597adedeae0c`.
- Public identity: `~ladwel-tobpem-sorfel-fornec--solsep-tirbyr-pantux-wanzod`.
- Completed: `2026-09-29T04:22:37.974225507Z`.
- Fresh bootstrap, post-pack restart, post-meld restart and final initialized-pier
  replacement all verified the same public identity.
- Both utilities used `network=none`, completed with resolved outcomes and had
  identity-preserving restarts. Live limits and pier metadata were observed.
- Pack logical bytes: **1,021,308,940 → 589,230,092**; admission **7.88 GiB RAM /
  404.25 GiB disk**.
- Meld logical bytes: **694,755,340 → 650,174,476**; admission **7.89 GiB RAM /
  404.04 GiB disk**. The two utilities bracket different stopped states, with runtime
  activity between them; logical sizes do not measure loom usage or reclaimed blocks.
- All seven containers and the owned bridge were removed. Provisioning and both
  utility uncertainty flags are false; the private pier and report remain.
- Verified loopback HTTP ports: `32772`, `32773`, `32774`, `32775`.

Report:
`/var/lib/urbit-runner/.local/share/urbit-runner/qualification/5418c87ccd3aac4ef8a1597adedeae0c/report.json`.

An independent identity-pinned management snapshot after the supplied result
confirmed alpha healthy at 2/2, comet-check/import-check/panel-check healthy at
1/1, and ssh-check uninitialized at 1/0. No new qualification run was requested.
Managed comet compaction and scheduling are the next integration gate; production
runtime qualification remains false.
