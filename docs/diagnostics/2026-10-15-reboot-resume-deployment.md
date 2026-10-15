# Unclean-reboot resume deployment — 2026-10-15 UTC

**Implemented and deployed:** both sourceport and sourcebox runners use **schema 15**
and advertise `unclean-reboot-resume`. The desktop's bundled runner matches them.
The three sourcebox ships interrupted by the October 14 hard reset have resumed
their existing piers and are running/healthy.

## Policy delivered

- Record the host boot ID, container ID and applied configuration when a resolved,
  initialized ship is running/healthy.
- On a later host boot, permit an authenticated terminal container's nonzero/OOM
  exit during automatic resume. Preserve the current pier, runtime pin and identity.
- Persist pending authorization before polling; a management restart before dispatch
  preserves it. Journal one attempt per runtime generation and host boot.
- Recheck container ownership/terminal state, storage identity and RAM/disk admission;
  remove the exited container and create an initialized-pier resume container.
- Keep intentional stops, moved-source fences and unresolved host operations guarded.
  Management-only restart does not authorize unclean-exit recovery. An actual failed
  or interrupted attempt does not create a retry loop.
- Existing schema-14 recovery states are not silently reclassified using invented
  historical boot evidence. This fleet's three legacy failures received the explicit
  recovery below; successful health observations establish their future eligibility.

See [services](../services.md) for readiness limits, migration and exact behavior,
and the [incident analysis](2026-10-14-hard-reset-policy.md) for the rationale.

## Build and deployment

Frozen at **00:07:30 UTC**, Go 1.25.0, Linux/AMD64, static, `-trimpath`, from
`bf854ad5a789b7f30e20e6ed3fe6fe1f4c1cd60e` plus the pending development working tree.
Private `build/reboot-20261015/provenance.json` records source and artifact hashes.
Version remains `0.1.0-rc.2-dev`; this is not a new published release.

Runner SHA-256 on both hosts and in the desktop bundle:

```text
99a2e68198f69726f317c9a9cbb6e77553c244db46c732bc1ef43004d5f8b8cd
```

The collector and Omabit CLI hashes are unchanged from the October 14 full update.
Each runner upgraded **14 → 15** through the host installer with its runner identity,
operator grants, storage/configuration and existing containers preserved. No Docker
restart or host reboot was performed. Both hosts retain root-only diagnostic
database/executable snapshots and receipts in
`/var/lib/urbit-runner-upgrades/reboot-20261015/`. These are not pier restore or
binary-only downgrade instructions; old runners refuse schema 15.

The approved cold-replacement desktop installer completed successfully. The live
plugin reports `0.1.0-rc.2-dev`, installed file/binary hashes match the frozen package,
and its collector sees the new capability on both runners.

## Recovery receipts

One explicit Stop followed by initialized Start was submitted for each affected
ship, with unique request IDs and the existing pinned configuration revision.
No recovery operation was repeated.

| Ship | Revision | Stop receipt | Start receipt | Healthy UTC |
| --- | --- | --- | --- | --- |
| `remote-smoke` | 2/2 | `op-e76831b6940a6b766595d2d3273df4d7` | `op-5d4d128af09466421219a720247f63a8` | 00:09:56 |
| `remote-comet` | 2/2 | `op-b6e63a955fc82e5251a8c9e675a20d9f` | `op-aeb680a321398c4b65fc6d674ea93a72` | 00:10:06 |
| `panel-check` | 1/1 | `op-74d51730632e8ea76c0300b27bea1bdb` | `op-8fcf750455e0955cf6f09069fb6573e6` | 00:10:16 |

| Ship | Replacement container prefix | Vere replayed events |
| --- | --- | --- |
| `remote-smoke` | `35ba9625d42a` | 51563–52257 |
| `remote-comet` | `736246f783d7` | 39983–40589 |
| `panel-check` | `ebe897cc16f1` | 100780–101319 |

All three logged **playback complete**, followed by live HTTP. The fake ships retain
`~zod`; the comet retains
`~sapbyt-lignum-widnec-mirbud--faslec-hobnet-pinlup-wanzod`. The runtime kinds are
`http-restart` / `comet-resume`, using the existing piers and 60-minute checkpoints.
Vere also performed its normal epoch rollover for `panel-check` after replay.

The first verification script stopped after all six lifecycle operations had
succeeded: an additional tailnet HTTP request to `panel-check:18585` was refused.
This was an endpoint-verification failure, not a failed resume. The original partial
receipt is preserved. Follow-up checks were read-only: local host HTTP identity
checks passed for all three ships, tailnet identity checks passed for smoke/comet,
and all three remained healthy in the same containers for a further 90 seconds.
No network-exposure configuration was changed.

## Final fleet and reboot eligibility

All **11 registrations** present at this deployment's baseline are preserved, with
seven running/healthy ships. The user had already deleted `ssh-check` before this
task; it was not recreated.

- Workstation: `alpha`, `bus-lab`, `comet-check`, `dibpen-fogweg-dozzod-sarlev` keep
  their running containers and now have healthy-boot records for
  `6f75f84d-7c19-4783-af44-94286ab79d5f`.
- Sourcebox: `panel-check`, `remote-comet`, `remote-smoke` have verified healthy-boot
  records bound to their replacement containers and current configuration revisions
  in boot `13e98cc6-433b-4060-97a3-a147044cf4be`.
- `import-check` and `rc1-pill-check` remain stopped; `rus` remains uninitialized.
  The workstation's moved `panel-check` source remains fenced.
- Ship identities, configurations, paths, bootstrap metadata, maintenance policies
  and transfer fences match the baseline. No unrelated container was replaced.

## Verification scope

Passed full `GOMAXPROCS=2 GOTOOLCHAIN=go1.25.0 go test -p 1 ./...` and
`go vet -p 1 ./...`, plus focused race checks for the startup/reboot suites in
`internal/managed` and `internal/runtimequalify`.

New coverage includes nonzero 255/137 and OOM terminal exits, boot-bound
authorization, legacy migration, service restart versus reboot, known failures,
persisted pending selection across polling/restart, failed/interrupted attempts
without loops, stale deletion/name-reuse/Stop races, creation-container replacement
without bootstrap replay, and final container-state reinspection before removal.

Live checks establish the deployment, post-hard-reset Vere replay/resume, same
identities, stable health and persisted reboot eligibility. The new automatic path
was initially covered by fixtures. A subsequent user-reported hard reset
[passed the real automatic-resume gate](2026-10-15-hard-reboot-acceptance.md).
No additional hard reset was induced during deployment.
