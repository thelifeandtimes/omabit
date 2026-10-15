# RC.3 publication and fresh-install handoff

Published [v0.1.0-rc.3](https://github.com/thelifeandtimes/omabit/releases/tag/v0.1.0-rc.3)
on October 15 at 16:29 UTC as a Linux/AMD64 prerelease, not latest. Build/tag source:
`1b6b79193c6ac58f378ed5c6c689fe9d89045257`, Go1.25.0, schema16, owner-user execution,
default `/srv/urbit/piers/`. RC.1 and RC.2 published bytes are unchanged.

The [source CI run](https://github.com/thelifeandtimes/omabit/actions/runs/38768119425)
passed. All eight GitHub assets were downloaded to a new local directory and passed
the release verifier: exact hashes/sizes, archive provenance, executable smoke,
isolated desktop install/replacement and modified-file refusal.

## Frozen artifact hashes

| Asset | SHA-256 |
| --- | --- |
| archive | `ae064c32c7040c781fc432106a51e80640c296149e0da5dcd841da9773ca4cc5` |
| omabit | `a76ee3f0d90b09772b1dea4442d47bd606059ed19a62f2371b3a5a8b1978dd81` |
| collector | `2b70f2701e18e00dcf39ee353b68cd08ceb143366f09ae85cb9061a442c50ddc` |
| runner | `f1fbca068450c771ad0a698973189aa0fd697c19013092b8e4513fc96d465eb9` |
| draft pill | `51f0be723918ab7feaaf28b3c599f36f0f0fca6e68453f84517ba1659f07d57e` |
| qualification.json | `bb5f5fa6e30c8250897ad10908299386a4ac32060af4c803e55852cb5de9468d` |
| release.json | `0c3ad6ff8160d0a75246864151c6145daeddeaa71ff7d4d067183ab5d4b8ecc4` |
| SHA256SUMS | `004d314de18a5d750541d3f17a6ac29c7779159df7690739c870c82ca58af978` |

Native pins advance separately to these published executables, avoiding a
self-referential build commit.

## Canonical native installation verification

At 16:32 UTC the real `omarchy plugin install` command cloned GitHub commit
`39379661db930aab8feea043981c153050f91141` into an isolated HOME. The install/enable
IPC used a test double, leaving the real desktop untouched. Concurrent first-use
launchers downloaded and verified all three published RC.3 binaries. The helper
reported `execution_model: owner-user` and `/srv/urbit/piers`; the isolated collector
remained unenrolled. Verified cache reuse worked with a deliberately unavailable
HTTPS proxy. The checkout stayed clean and native Git update reported up to date.
No host setup was invoked. Receipt: `/tmp/opencode/rc3-native-install/result.json`.
The [native-pin commit's CI](https://github.com/thelifeandtimes/omabit/actions/runs/38769662373)
also passed.

## Exact-binary live acceptance

The frozen runner was installed with `install-host --owner thelifeandtimes` on the
local qualification host. It retained runner ID, owner profile, passwd/groups,
subIDs, management grants, owner-engine PID and unrelated system-Docker PID. The
running fake kept its container; the comet stayed intentionally stopped.

A controlled restart of only the owner engine/runner then produced exactly one
successful `system:startup` resume for the fake. The comet remained stopped. Both
piers subsequently passed initialized resume and committed Clay content readback;
UID maps still bound container UID0 to host UID1000.

A disposable stopped copy containing normal user-created mode0775 directories and
mode0664 files passed external import into the custom root, same-identity Start,
Stop, Delete, revision-advancing name reuse, stale-delete refusal and original-delete
replay. The replacement registration was deleted. Originals finished stopped.

Full receipt hashes and honest limitations are in attached `qualification.json`.
Initial development desk and multi-key SSH gates are documented in the
[owner qualification record](2026-10-15-owner-user-runtime.md). No owner-user host
reboot/logout, cross-host transfer or full app/OTA qualification is claimed.

## User handoff

Sourcebox remains reserved for the user's own fresh install:

```bash
omarchy plugin install https://github.com/thelifeandtimes/omabit.git --enable
```

Then use **Setup → Set up this computer**. The [walkthrough](../sourcebox-clean-install.md)
describes plugin-only inspection, OS-authorized owner setup and first-ship checks.
Sourceport retains its owner qualification installation with both test piers stopped.
Temporary SSH grants were restored to the exact original file. Prior retirement
copies remain preserved; the real user's login/home/groups and preexisting linger
setting must not be removed as application cleanup.

Final local verification found exactly the two stopped qualification piers, no
active operations or running owner-engine containers. Temporary test SSH keys were
removed after checking grant restoration. Login identity/groups, preexisting
`Linger=yes`, and unrelated system-Docker PID matched the initial baseline. Private
receipt: `build/rc3-qualification/final-state.json`.
