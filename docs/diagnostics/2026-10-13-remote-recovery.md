# Remote recovery and PR #1 — 2026-10-13

**Resolved:** sourcebox's `remote-smoke` and `remote-comet` are running/healthy again,
using their existing piers and identities. The tested PR #1 runner hotfix is installed
on sourcebox. The development UI now distinguishes current container state from the
last historical operation; that wording change awaits the next desktop deployment.

## What the screenshots meant

The displayed “Start — succeeded” / “Resume after host startup — succeeded” receipts
were accurate historical operations from October 10. Their containers subsequently
became exited, with exit code 255 and no Docker OOM flag. Omabit correctly withheld
automatic Start because its startup policy requires a clean exit-0 result.

The UI wording was confusing: it placed the old successful operation above the
current failure and suggested starting the ship to retrieve +code while Start was
disabled. Source now labels that receipt **Last operation**, puts the current exit
first, explains the chronology, and describes explicit Stop/reconciliation for
this particular exited-after-success case. Other provisioning/maintenance recovery
states retain their separate review requirements.

## Evidence and limits

- Both stored pier, `.urb` and parent identities matched the current filesystem,
  including device **31**, inode and nanosecond birth times. These were exited
  runtimes, not the false “missing/replaced directory” state fixed by PR #1.
- On October 10 at **04:00:42 UTC**, systemd aborted `systemd-logind` after its
  three-minute watchdog expired. `coredumpctl info` confirms SIGABRT, PID 1676,
  command `/usr/lib/systemd/systemd-logind`, in boot
  `f293e1a1d22e4746b13ee4e827d25f0b`. Its captured stack was in `ioctl`; the remaining
  logind frames were unsymbolized. This does not establish why the ioctl stalled.
- That boot's journal ended at **04:02:45 UTC**. No dedicated-runner shutdown entries
  appeared in the inspected 04:00–04:12 interval, and no matching kernel OOM/panic
  evidence was found. The exact trigger for the subsequent host restart is unknown.
- The next boot, `30119f1f758b40c79b61a0aab7c807f3`, began at **04:06:22 UTC**.
  Docker began restoring containers at **04:06:25.566 UTC**. The two recorded exit
  times, **04:06:25.604/.608**, fall in that restoration window. Docker logged
  “layer not mounted” and removal of their stale network sandboxes. This is evidence
  of stale runtime state after an unclean host restart, not two new application
  crashes on the day the screenshots were taken.
- During recovery the host had about **64 GiB available RAM** and **467 GB free disk**.
  Current Docker/runner services were active with zero automatic restarts. Current
  logind was also active with zero restarts since its October 10 boot.

The preceding logind watchdog failure is relevant host evidence, not proof that
Omabit caused the host problem. No systemd/logind configuration was changed, no core
memory was exported, and no native/unmanaged ships were operated on.

## Recovery performed

After reviewing the retained logs, filesystem identities and available resources,
each instance received one explicit **Stop**, followed by initialized **Start**.
Stop reconciled/removed its exited container; it did not delete the pier. Start
used initialized resume, not a creation command or a new pill/key.

| Instance | Operation | Receipt | Completed UTC |
| --- | --- | --- | --- |
| `remote-smoke` | Stop | `op-f4f79071f0bdcd890eb2c11afd2d27f2` | 18:08:46 |
| `remote-smoke` | Start | `op-59010da6c30b22ffd33738ec19957387` | 18:08:56 |
| `remote-comet` | Stop | `op-1ac0cb698a49182414e450773abc0218` | 18:08:59 |
| `remote-comet` | Start | `op-97b766d02b4efb5e28b985df06528d7d` | 18:09:08 |

Both operations succeeded for each ship. Revision/applied revision remained **2/2**,
snapshot interval **60 minutes**, and the existing ship identities/pier paths were
preserved. The replacement kinds were `http-restart` and `comet-resume`.

- `remote-smoke`: `~zod`, HTTP 18081; container
  `2d22a600b6e93ed7260ad043d37b276238c488c1ec0bcf582723759c7b0ba1ce`.
- `remote-comet`: `~sapbyt-lignum-widnec-mirbud--faslec-hobnet-pinlup-wanzod`, HTTP
  18082; container `ee714d6b60faf5b5a536cb140329905bb560009635e81f43cd7e03a0bf55f9bc`.

Tailnet `/~/host` checks returned the expected identities. Both remained healthy
during a 90-second observation and were checked again after the hotfix at
**18:58:52 UTC**, still using those same containers. Other local/remote managed
states, containers, policies, revisions and the moved-source fence were preserved.
`panel-check` and `rc1-pill-check` remained intentionally stopped on sourcebox.

## PR review, merge and deployment

[PR #1](https://github.com/thelifeandtimes/omabit/pull/1), by **wundrellama
(~dinnyt-divsud)**, fixes a separate real btrfs reboot regression. Per-boot `st_dev`
values are not durable object identity. `SameObject` compares inode plus nanosecond
birth timestamps (for the pier and `.urb`); records without birth timestamps retain
strict comparison. Same-boot review fingerprints still include device numbers.

The exact PR head **`91d26f3f18b74d665ca763c488bb1fa6cd9db383`** passed Go 1.25.0
`go test -p 1 ./...` and `go vet ./...` in an isolated checkout. It was approved and
merged as **`bf854ad5a789b7f30e20e6ed3fe6fe1f4c1cd60e`**. The
[merged CI run](https://github.com/thelifeandtimes/omabit/actions/runs/38444925129)
passed.

Sourcebox received a static runner built from that tested PR head, preserving
**schema 13** and the existing runner/SSH/operator identities. Installed SHA-256:

```text
9fd3226f4060b253379adc6391fd72a61a5f06c928e1f4f308931dcceaf41eac
```

The first authorization attempt expired at sudo before installation. The explicitly
authorized retry completed, returned `ready: true`, and passed the installed hash
check. Post-update comparison preserved every managed instance's state and container
ID, including both recovered ships. Published rc.1 release assets were not replaced.

The in-progress schema-14 storage work was preserved during integration and extended
to use the same durable comparison for configured roots, import identity claims and
deletion continuation. Tests cover renumbered configured roots and interrupted
deletion after renumbering. Fresh source reviews/content fingerprints remain strict.
Targeted race checks, model tests and mapped QML checks passed.

The PR's separate poweroff-ordering concern and interrupted-prune device audit remain
follow-ups. These incident logs do not establish that an orderly poweroff dependency
bug caused this restart, so no speculative service-ordering change was deployed.
