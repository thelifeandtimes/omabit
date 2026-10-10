# Disposable comet epoch lifecycle accepted — September 29 UTC

The user supplied a passed 6V `comet-epoch-lifecycle` result and confirmed management
was restored. No qualification rerun was needed.

- Run: `e84ffd83afcfbbe71e6cceca2d69c30f`.
- Public identity: `~nordec-ramnus-tornux-noplyr--tipsel-loswyn-ricdev-litzod`.
- Completed: `2026-09-29T15:33:23.001937287Z`.
- Initial inventory: one `0i0` epoch, digest
  `111fe2c06005010aefd8143d906fd847ddfb4cb11bff5970595b24ca7fc4fb4f`.
- Three network-isolated rolls each added exactly one snapshot-bearing epoch.
  Each post-roll replacement retained the exact public identity and epoch set.
- A final initialized-pier replacement also retained the identity and four epochs.
- All nine containers and the owned bridge were removed; all create/roll/provisioning
  uncertainty flags were false. Live limits and pier metadata were observed.

| Roll | Started UTC | Finished UTC | Available RAM / free disk |
| --- | --- | --- | --- |
| 1 | 15:32:24.646039868 | 15:32:26.722956271 | 7.06 / 402.96 GiB |
| 2 | 15:32:33.563062588 | 15:32:35.339128147 | 6.97 / 402.78 GiB |
| 3 | 15:32:41.870947707 | 15:32:43.390861127 | 6.93 / 402.61 GiB |

Final inventory digest:
`7f34490127f9909b651fb166c703267ff4e9ffb38002e99559c5bfc8b2bd89ea`.

| Epoch | Logical bytes | Allocated bytes | Snapshot bytes | Protected |
| --- | ---: | ---: | ---: | --- |
| `0i0` | 216,633,356 | 216,637,440 | 0 | false |
| `0i78` | 805,343,244 | 805,347,328 | 803,831,808 | false |
| `0i178` | 803,930,124 | 803,934,208 | 803,831,808 | true |
| `0i287` | 817,360,908 | 817,364,992 | 803,831,808 | true |

The preview selected only `0i0`, retaining three epochs. Deletion support was false;
no history was deleted. HTTP identity checks used loopback ports 32776–32780.
These metadata/size observations do not verify backup contents or loom usage.

Report:
`/var/lib/urbit-runner/.local/share/urbit-runner/qualification/e84ffd83afcfbbe71e6cceca2d69c30f/report.json`.

An independent identity-pinned snapshot after restoration confirmed alpha healthy
at 2/2, comet-check/import-check/panel-check healthy at 1/1, and ssh-check uninitialized
at 1/0. No active work, recovery requirements or incidents remained. The comet-check
and import-check test schedules remained disabled. Managed epoch integration is the
next gate; comet deletion/recovery and production runtime remain unqualified.
