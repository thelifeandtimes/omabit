# RC.13 update: empty fleet from stale live QML — 2026-10-19

The user ran `omarchy plugin update thelifeandtimes.omabit` on sourceport
(Omarchy4.0.4-1), then saw no previous fleet and could not run fresh host setup.

## Diagnosis

- The installed native checkout was clean at distribution commit `b70f013`, with
  RC.13 manifests and published binary pins.
- The running shell was still PID3509355. Its actual monitor IPC returned
  `version:"0.1.0-rc.12"`, `busy:false`, `first_run:false` despite RC.13 on disk.
- At14:00:30PDT/21:00:30UTC the shell journal recorded repeated local-plugin reload
  notifications followed by null-service bindings in import/storage/key-boot panes.
  No desktop inventory collector remained running when inspected.
- The actual RC.13 collector still loaded the existing `local` SSH enrollment and
  retrieved an available operator snapshot with **two healthy running ships**:
  `bare-comet` and `also-a-comet`. The saved runner identity was unchanged:
  `runner-70a48c60cb4e396c1b9628224f229a1a`.
- The actual setup helper reported `ready:true`, `host_present:true`,
  `can_setup:false`. This correctly identifies an already configured host; the
  disabled fresh-setup action was not evidence of missing host state.

The installed stock update command fetches/fast-forwards, validates and requests
`rescanPlugins`. It does not cold-restart the shell. The evidence establishes stale
RC.12 code and failed desktop observation after hot reload; it does not establish
a migration failure or lost fleet data. The precise internal QML cache/teardown
mechanism was not separately reproduced in an isolated shell.

## Recovery and verification

With the desktop unlocked and the plugin idle, ran the supported
`omarchy restart shell`, then opened the monitor. The new shell PID1148390 reported
**0.1.0-rc.13**, and its RC.13 collector resumed `inventory --watch --interval 2s`.
A cropped monitor-only capture verified both healthy ships, the two-ship/one-runner
count, and the RC.13 header in the actual live UI.

Before and after recovery:

| Resource | Preserved evidence |
| --- | --- |
| Runner service | PID3657443, started2026-10-18 14:27:44PDT |
| Private rootless engine | PID1078779, started2026-10-17 21:58:08PDT |
| `also-a-comet` | Container `f7c90e46054c6875b5553ae72302bba18644717d1ed899fd3a98fbc1fff57ac3`, PID1269096, started2026-10-18T06:30:21.005274482Z |
| `bare-comet` | Container `c08cbe24ed355e7a0980ef3957a0381072eb5b9a5da2d6f142675b8a015aade3`, PID1094395, started2026-10-18T05:06:35.035314298Z |
| Shell configuration SHA-256 | `2ebbc550e93b26a355eeaf9033c085e91cefc3e6705970f138ad638c33a29e21` |
| Collector registry SHA-256 | `e00761805a501c9e0bdcbb0a23f3ba052d3075506ac5ccb470b77d82fc99a03f` |
| Panel preferences SHA-256 | `628da095563d404ede4d1b6b305dfa1b47c52c4095390ffd74e77e513a38fa85` |

The new shell journal showed no Omabit binding errors during this verification.
One unrelated Qt portal-registration warning was recorded. Private visual evidence:
`/tmp/opencode/rc13-fleet-recovery-20261019.png`; it is not included in source.

The installed host remains the existing RC.12/schema20 generation. The RC.13
collector reports the older host's missing `host-compatibility-v1` as diagnostic
guidance while retaining its fleet. This recovery did not perform a host upgrade,
reset, re-enrollment, ship operation or pier modification.

Use the [cold native update sequence](../native-plugin.md#updates) for future desktop
updates, and the separate [schema21 procedure](../host-compatibility-upgrades.md)
when upgrading the existing runner. Fresh-install testing remains a separate task.
