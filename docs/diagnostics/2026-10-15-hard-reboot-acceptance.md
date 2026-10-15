# Sourcebox automatic hard-reboot resume — 2026-10-15 UTC

**Passed for the exercised fleet:** after another user-reported hard reset, all
three previously running sourcebox ships resumed automatically through one
successful `system:startup` operation each. No operator recovery was needed or
submitted during this inspection. The intentionally stopped ship stayed stopped.

## Boot and service evidence

- Prior boot: `13e98cc6-433b-4060-97a3-a147044cf4be`.
- New boot: `ae4dbac8-85fc-4e1e-bde6-eea3157d3131`, first journal entry
  **00:42:14 UTC / 17:42:14 PDT October 14**.
- Dedicated Docker service started **00:42:18 UTC**; runner started
  **00:42:20 UTC**, with its socket startup receipt at **00:42:20.560228**.
- Both services are enabled/active/running with zero service restarts; lingering
  is enabled. The runner retains `--resume-running`.
- Management schema **15**, unchanged runner ID
  `runner-b4043fa2d0aee501aa2cf78322aed911`, installed SHA-256
  `99a2e68198f69726f317c9a9cbb6e77553c244db46c732bc1ef43004d5f8b8cd`.

Docker's restoration window reports the three previous container IDs with
`layer not mounted` and stale sandbox cleanup. It subsequently initializes its
API and the runner replaces those retained containers. This matches the expected
unclean-host-loss recovery path.

## Automatic operations and replay

| Ship | Automatic receipt | Healthy UTC | Replacement container prefix | Replayed events |
| --- | --- | --- | --- | --- |
| `remote-comet` | `op-a43746f51d6703b6e9a868dc5498b560` | 00:42:32.819762 | `8ab4039bae3d` | 40590–40949 |
| `remote-smoke` | `op-92c4c48dd01ddcfc3b16e44edc14f8f3` | 00:42:39.352002 | `661a4e50e0bb` | 52258–52646 |
| `panel-check` | `op-9bf74b9d9d6cb1d1ce713bd97ac356c3` | 00:42:45.850206 | `7a8e55aa484c` | 101320–101720 |

All receipts have kind `startup-resume`, actor `system:startup`, status `succeeded`
and the reboot/replay message. Exactly these three operations were created after
00:15 UTC; there are no later manual operations or duplicate automatic attempts.
Queue times overlap, but actual replacement container starts are serial: comet
00:42:22.456, smoke 00:42:34.095, panel 00:42:40.578. All were healthy about
**25 seconds after management startup**, or about 32 seconds after the first boot
journal entry.

Each Vere log reports **playback complete**, followed by a live pier and HTTP.
Inspected commands use initialized `comet-resume` / `http-restart`, with no
`-c`, `-F`, `-B`, `-w` or `-k` flags. They mount the same exclusive pier parents.
Docker restart policy is still `no` and restart count 0: this was runner-controlled
replacement, not Docker's independent restart mechanism.

## Preserved state and sustained verification

Snapshots at **00:54:22 UTC** and again **00:57:46 UTC**, plus privileged inspection
at **00:55:40 UTC**, confirm:

- All three remain running/healthy in the same new containers, without recovery or
  active-operation flags. Local HTTP identities match; smoke/comet also pass their
  existing tailnet HTTP identity checks.
- Healthy-boot records for all three point to the new boot and their correct new
  container/configuration revision, with no pending resume left over.
- Ship identities, configuration/applied revisions, paths, bootstrap metadata,
  maintenance policies and handoff fences match the pre-reboot fleet.
- `rc1-pill-check` remains stopped. The workstation's seven registrations retain
  their prior state/container IDs, including its permanently moved `panel-check`
  source. All 11 current registrations are accounted for.

## Log qualifications

The logs are not warning-free. Docker emitted stale-network cleanup and rootless
network-namespace warnings during replacement. At startup it also logged DNS
timeouts against `127.0.0.53`; Vere's first external peer-state fetch failed, so its
log reports that the optional czar peer-state double-boot check was unavailable.
Subsequent Ames hostname resolutions produced addresses, and runtime/identity/HTTP
readiness succeeded. Missing optional mDNS/Avahi support also appears in the logs.

These did not prevent lifecycle recovery. They do not establish complete Ames,
application or OTA health; include an outbound DNS/HTTPS check in the next release
qualification pass. The unrelated host crash's cause was outside this inspection.

Private evidence: `/tmp/opencode/reboot-acceptance-20261015-{before.json,host.log,result.json}`.
The final comparison has `passed: true`. This accepts a real unclean host reboot
for the current fake/comet fleet, not interrupted maintenance or incomplete first
boot recovery.
