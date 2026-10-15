# RC.2 publication and fresh-install handoff — 2026-10-15 UTC

**Complete:** [v0.1.0-rc.2 is published](https://github.com/thelifeandtimes/omabit/releases/tag/v0.1.0-rc.2)
as a prerelease, not latest. Both old dedicated runner installations are retired.
The verified release is extracted on sourcebox, ready for the user to run its plugin
installer manually. No new sourcebox plugin installation or host setup was run.

## Published artifact

Build/tag commit: **`691eea7f808237458c1bf0c8865a62cd5cee11dc`**.
Go **1.25.0**, Linux/AMD64, static binaries, schema **15**.
[GitHub CI](https://github.com/thelifeandtimes/omabit/actions/runs/38669322929) passed.

Eight assets: binary archive, three standalone binaries, frozen draft pill,
`release.json`, `qualification.json`, and `SHA256SUMS`. Published rc.1 was not changed.

| Artifact | SHA-256 |
| --- | --- |
| Archive | `b59933378d9dc6814b4d03bee6d67b60168ae34e978d1ba82f37a281bf08c7da` |
| Runner | `1c59aaa97f17242d55b283703bea3626a7c5fdfd831841afba319b60fb30660e` |
| Collector | `924b80ce7d8a66d3e2dcab2d81160beb5121c330eb24b20473461d65984b9f0a` |
| Omabit CLI | `e2f42e1c67d308e3a74de3f18d5d2a2c2f93dfb6b4d5f710599c5a95f99f5409` |
| Pill (same frozen draft as rc.1) | `51f0be723918ab7feaaf28b3c599f36f0f0fca6e68453f84517ba1659f07d57e` |
| Outer SHA256SUMS | `7e4babf92a20c4d17abcaa272b90edc9b0e6cccabf9c33247a187a008acc4d6a` |

GitHub-downloaded assets passed full archive/checksum/provenance/executable checks
and isolated installation/replacement/customization-refusal verification. Both
hosts' staged copies were downloaded independently and checked against the pinned
outer checksum file and inner archive manifest.

## Release checks

- Full Go tests/vet and race checks for managed/runtime/client/access/collection/copy.
- Actual OpenSSH with two operators, observer restrictions, attribution, revision
  conflicts, request replay, client loss/reconnect and revocation.
- Desktop model, installer, mapped QML storage/import/delete and real compositor focus.
- Exact packaged runner installed locally for a live disposable storage smoke gate.
  A separate copy of the stopped `import-check` fake was reviewed and moved into
  custom root `rc2smoke`, resumed with the same identity, stopped, and deleted.
  Reusing its instance name advanced revision 1 → 2; a stale delete was rejected,
  replay of the original delete did not affect the replacement, and the uninitialized
  replacement was deleted explicitly. Original source contents and the fleet were
  preserved. The empty custom root was subsequently retired with the local host.
- Sourcebox's existing comet resolved `zod.urbit.org` and reached HTTPS at
  `bootstrap.urbit.org`; the root URL returned HTTP 403. This is connectivity evidence,
  not a successful pill-object download or application qualification.

The attached qualification distinguishes the prior real hard-reboot result from
the exact packaged-candidate checks. Cross-device/forced-interruption and full app
UI/OTA acceptance remain scoped as documented. The user-driven clean-install and
complete independent-collector experience are the next gates.

## Retirement completed

All seven previously running managed ships received one successful explicit Stop
before host retirement. Then only the app-owned management SSH, runner, dedicated
Docker services, lingering and dedicated user-manager state were stopped/disabled.
The old runner accounts/groups and subordinate-ID mappings were removed after their
files were moved out of active paths. No piers were discarded by the host retirement.

Root-only retained installations:

- Sourceport: `/var/lib/omabit-retired/sourceport-rc2-20261015/`
- Sourcebox: `/var/lib/omabit-retired/sourcebox-rc2-20261015/`

Each contains a `receipt.json`, former account/subordinate-ID records, journal
excerpts, prior route configuration, and retired install/config/home paths. These
are diagnostic retirement copies, not automatic restore inputs. Previous root-only
upgrade evidence also remains under `/var/lib/urbit-runner-upgrades/`.

The local owned plugin, collector registry, setup key/profile and app state were
retired to `~/.local/share/omabit-retired/sourceport-rc2-20261015/`. Sourcebox already
had no Omabit plugin, collector enrollment or app setup profile; its per-user receipt
records no moved files. Tend's separate config and legacy CLI were preserved.

Both host retirements verified no old active home/config/binary/service paths or
runner account remained. Unmanaged native/tmux processes and the shared Docker
service/PID were preserved. Only matching Omabit TCP-forward Serve entries were
removed; other Tailscale configuration was compared and preserved.

## Exact user handoff

Both hosts have the extracted package at:

```text
~/omabit-releases/0.1.0-rc.2/omabit-0.1.0-rc.2-linux-amd64/
```

The sourcebox staging receipt completed at **02:19:54 UTC** and confirms verified
checksums, no installed plugin and no runner account. In a normal terminal inside
the **sourcebox Omarchy desktop**, the user should run:

```bash
cd "$HOME/omabit-releases/0.1.0-rc.2/omabit-0.1.0-rc.2-linux-amd64"
./install.sh --enable
```

No sudo is needed for that step. Pause after the plugin appears to review the
first-time UI; host setup is the following deliberate user step. See the
[piece-by-piece walkthrough](../sourcebox-clean-install.md).

Private execution receipts are under `/tmp/opencode/rc2-*`; public qualification is
attached to the GitHub release. Documentation commits following the build/tag commit
do not rebuild or replace its frozen artifacts.
