# RC.8 protected sourcebox reset

The user authorized another fresh-install cleanup after the setup/import fixes
and supplied the [protected-ship policy](../protected-ships.md). The policy was
checked before live work and encoded in the reset's explicit preflight assertions.

## Release

- Published: <https://github.com/thelifeandtimes/omabit/releases/tag/v0.1.0-rc.8>.
- Source commit: `f972dee22d20070de35b6f26a6ce3d4e1d2497de`.
- Source CI: <https://github.com/thelifeandtimes/omabit/actions/runs/38966269442>.
- All eight downloaded assets passed checksums, provenance, executable and
  isolated installer validation. Native pins select the same published tools.

## Reviewed live state

Runner `runner-6921f2202ba2cc8377194f443a8b6c5a` had one healthy running managed
comet, `a-test-comet`, and a stale `lismyl-lismyl` registration with a successful
Stop receipt, no containers, and a missing enrolled path. No operations were active.

The real protected pier was independently located and identified at:

```text
/home/thelifeandtimes/temp-ur/lismyl-lismyl
```

It is a live-mode planet named `~lismyl-lismyl`, with no Vere lock marker. Its
reviewed tree contains **130 files / 49,556,773,510 bytes**. The metadata/tree
fingerprint (not a full content/backup hash) is:

```text
06f6814bf829fa293274b7b838abc804e6db0d0f8a9366059a4402b1ec80f46f
```

The reset asserted that this pier and all twenty reviewed unrelated pier
directories were outside every retirement target. It also pinned the expected
test container's full ID, labels, single data mount and host cgroup/process mapping.
The managed pier root contained only that test comet's `.urb` tree.

## Completed retirement and verification

Retirement completed at **2026-10-16 21:07:17 UTC**:

- Gracefully stopped only `a-test-comet` through the pinned runner.
- Sent no runtime operation to `lismyl-lismyl`; its stale management registration
  was preserved with the retired journal rather than operating on its moved data.
- Disabled Omabit's management SSH, owner runner and private engine services.
- Retired app units, installation, managed state, default pier root, enrollment,
  settings/cache and plugin with same-parent renames. Earlier retirements remain.
- Preserved twenty unrelated pier directory identities and all ten unrelated Urbit
  PID/start-tick/command-hash/cgroup records, including the important native ships.
- Rechecked the protected pier's identity, file/byte totals and full tree fingerprint
  after retirement; they matched exactly. Shared Docker, account/group/subordinate
  IDs and preexisting lingering were also preserved.

The unlocked shell was cold-restarted at **21:17:03 UTC**, new instance
`zphf4r0nt`, PID `22967`. At **21:17:06 UTC** verification confirmed Omabit absent
from the live registry/layout, its active paths and sockets absent, and services
inactive. Other plugins/settings, shared Docker and ten unrelated Urbits were
preserved. The protected pier's fingerprint was checked again after the restart.
Result: **`fresh_install_ready: true`**.

Retirement manifest:

```text
/var/lib/omabit-retired/sourcebox-rc8-reset-20261016T205856Z/receipt.json
```

The actual retired trees remain beside their original paths, including
`/srv/urbit/.piers.retired-20261016T205856Z` and
`~/.local/share/.urbit-runner.retired-20261016T205856Z`. The protected relocated
pier remains at its original `~/temp-ur/lismyl-lismyl` location.

Public reset/verification receipts are in
`/tmp/omabit-rc8-reset-20261016T205856Z/` on sourcebox, copied locally under
`/tmp/opencode/sourcebox-rc8-*.json`. The detailed root receipt stays private.

Next action is the user's fresh installation, as their ordinary desktop account:

```bash
omarchy plugin install https://github.com/thelifeandtimes/omabit.git --enable
```
