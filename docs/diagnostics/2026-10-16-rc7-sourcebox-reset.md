# RC.7 sourcebox fresh-install handoff

The user requested publication of RC.7 and retirement of the current sourcebox
installation before doing another fresh installation themselves.

## Release

- Published: <https://github.com/thelifeandtimes/omabit/releases/tag/v0.1.0-rc.7>.
- Source: `07e407a4d4c5b4142425547c55094e218bee0697`.
- Source CI: <https://github.com/thelifeandtimes/omabit/actions/runs/38925970558>.
- All eight GitHub assets were downloaded and verified, including archive
  provenance, standalone/archive executable equivalence and isolated installation.
- Native pins select those exact published executables. Actual empty-cache
  bootstrap download, hash/size/build-revision validation and cache reuse passed.
- Before retirement, the frozen collector returned 23 bounded log updates over
  66.53 seconds using one actual OpenSSH process, with successful cancellation
  and child reaping. See the [log-watcher qualification](2026-10-16-dbus-log-watch.md).

## Retirement

Preflight identified runner `runner-1bbdfb345bb67eb0c43d7b42383df115`, one healthy
running comet named `guac-demo`, one expected engine container, the default storage
root only and no active operations. Its configured/applied revision was 1/1.
The root helper pinned the reviewed owner, runner, ship, container ID, labels,
mounts, pier location and exact managed engine inventory before changing state.

At **2026-10-16 15:47:37 UTC**, retirement completed:

- Disabled the desktop plugin, gracefully stopped `guac-demo` through the pinned
  runner and verified the stop operation succeeded and no managed container ran.
- Disabled only Omabit's management SSH service, owner runner and private engine.
- Removed the exact Omabit shell entries and retired its plugin, host units,
  installation/state, collector enrollment, desktop settings/cache and pier root.
- Used same-parent filesystem renames throughout, preserving Docker overlay files
  and the entire pier. Earlier retirement directories were preserved.
- Verified all **10 unrelated Urbit processes** retained their PIDs, start ticks,
  command hashes and cgroups. Shared Docker, account/group/subordinate-ID files
  and preexisting lingering were preserved.

Retirement index (root-private detailed manifest):

```text
/var/lib/omabit-retired/sourcebox-rc7-reset-20261016T154033Z/receipt.json
```

Data remains in the same filesystems under the manifest's `retired_paths`. The
pier root is now `/srv/urbit/.piers.retired-20261016T154033Z`; owner engine/runtime
state is `~/.local/share/.urbit-runner.retired-20261016T154033Z`.

## Fresh-install gate

At **2026-10-16 15:49:47 UTC**, the unlocked shell was cold-restarted and verified:

- New shell instance `og7m1wyb0nt`, PID `2715720`, launch 15:49:44 UTC.
- Omabit absent from the live plugin registry and shell layout.
- Active Omabit paths/sockets absent; its three services inactive.
- Other plugin registry entries and shell settings preserved across the restart,
  including Tend, the Urbit theme, Omachan and Omonitor.
- All ten unrelated Urbit processes and shared Docker still preserved.
- `fresh_install_ready: true`.

Operational receipts are under
`/tmp/omabit-rc7-reset-20261016T154033Z/{log-watch-smoke,reset-result,verification}.json`
on sourcebox, with local copies under `/tmp/opencode/sourcebox-rc7-*.json`.

The next action belongs to the user, as their ordinary desktop account:

```bash
omarchy plugin install https://github.com/thelifeandtimes/omabit.git --enable
```

Then open the widget and choose **Start Setup →**. The new install retrieves the
RC.7 collector/QML pairing. Native Vere processes were preserved; this reset is
not a rollout of the upstream native-Vere mDNS patch.
