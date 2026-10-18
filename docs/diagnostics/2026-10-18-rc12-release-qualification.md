# RC.12 frozen release qualification — 2026-10-18

Release: [v0.1.0-rc.12](https://github.com/thelifeandtimes/omabit/releases/tag/v0.1.0-rc.12).
Source: `1529c1335754d15c6efd03d77ce2af7ede3d48f0`.
Source tree: `c623bda87c95ecfa9a8808945a49520ac7ec93b2`.
Published at 20:22:48 UTC. Schema20, Linux/AMD64, existing-owner-user rootless model.

The immutable assets and their sanitized qualification summary are attached to
the release. The three binary pins refer to that source, not to the later
distribution/documentation commits. Existing RC.11 assets remain unchanged.

## Source, artifacts and CI

- [Final-source CI39150909131](https://github.com/thelifeandtimes/omabit/actions/runs/39150909131)
  passed Go tests/vet, expanded race checks, command builds, model/installer/native
  fixtures, source-isolation/reproducibility and actual Qt6.4 offscreen interactions.
- The first candidate's packaging fixture exposed asynchronous Go telemetry during
  temporary-directory cleanup. Supported `go telemetry off` now runs only in the
  private build configuration; the corrected suite passed Go1.25 and Go1.27 before
  the final source freeze. No application change was required for this CI failure.
- Two independent clean exports of the final source, with Go
  `go1.27.0-X:nodwarf5`, produced byte-identical artifacts. Builds use static/trimpath
  settings and truthful clean VCS/source-tree/epoch stamps.
- All three actual final binaries passed govulncheck v1.9.0 with no vulnerabilities
  found. This is not a whole-host, runtime-image or pill-content security audit.
- The uploaded draft was downloaded independently and passed the complete asset
  and archive inventories, provenance/schema checks, executable smoke tests and
  isolated installer/replacement/customization-preservation checks before publication.

| Asset | Bytes | SHA-256 |
| --- | ---: | --- |
| `omabit-0.1.0-rc.12-linux-amd64.tar.gz` | 29268138 | `d1a636c988ec0104db8de2150f0a8484a027116ac3cdc6b7e0ae98730decfede` |
| `omabit-linux-amd64` | 17480501 | `973565a6aea82a55564393ed6fd4952ba9cb4558ce30201cf8044291c8889134` |
| `urbit-collector-linux-amd64` | 11794340 | `c5a096758712fbe1bdebd3d86f46572327ba3dbe8905c0712571eff8b6403e83` |
| `urbit-runner-linux-amd64` | 21602774 | `cb6256ac0c5337914c8342bc47d8170994eaf10f9cccb16c01bc3ffc5161d823` |
| Supplied pill | 59351751 | `85635c0b17a138a6473580ca362475faadf281e09cc9fa3746cfc698cc054f62` |
| `qualification.json` | 6080 | `c40955ae03cd08ecb132ce37d7b86fdfd4ccf67b7fbd4ff88ee92c8cb672ac96` |

## Controlled guest power loss

**62/62 passed: 186 boots, 62 host SIGKILLs, 62 fresh-boot readbacks.** Actual
production filesystem primitives were interrupted at stopped syscall boundaries
on separate virtual disks, under both ext4 and btrfs. Each filesystem covered18
copy/import/source-removal cases and13 executable/journal-publication cases.
Before destructive source removal, the complete source survived; after the
required barriers the complete destination survived. Installer state remained
recoverable with retained generations and valid journal/publication ordering.

QEMU11.1.1, 768MiB guests, Linux7.2.5-3-omarchy, e2fsprogs1.47.4 and btrfs-progs7.1.
The harness enforced the clean expected commit before and after execution.

Evidence: `build/rc12-powercut/run-20261018T174423.652443Z/`.
Report SHA-256: `5844db5ac6688db5a23475c170b343edfb49b09063189f35e80054682285049d`.
Independent audit: `db215b17a63eee56d59ae56ef8c941ecfad25c3541ae93059267253000b372a1`.

This models lost guest RAM/unflushed guest caches while completed host writes
survive. It does not claim torn-sector, lying-drive-cache or universal host-power
failure tolerance. No host block device, real pier or user host was power-cycled.

## Historical compatibility and live management upgrade

Every published RC.1–11 release has its own reproducible original-writer fixture.
The recovered schema14 and schema18 previews give13 fixtures in total; all original
readers refused real schema20 migrations without modification. Unavailable writers
11/12/17 are explicitly unsupported unpublished prototypes, not invented fixtures.
See the [support matrix](../reviews/2026-10-18-rc12-historical-upgrades.md).

A separate disposable full-system VM ran the exact signed Docker29.7.2 packages,
real rootless/systemd/cgroup-v2 stack and pinned official Vere4.6 image. No engine,
version, admission or process evidence was substituted. The primary frozen run:

1. Booted a fresh fake, verified HTTP and stopped it cleanly.
2. Booted a new comet and kept its runtime running through all updates.
3. Upgraded genuine RC.11/schema19 using the official RC.12 offline installer and
   repeated it without restarting the verified manager.
4. Performed a normal online update to a distinct clean same-source executable
   generation, repeated it, returned to the official release binary and repeated.
5. Replayed original creation receipts and checked old/new collectors over actual
   restricted SSH, including persistent logs and cancellation cleanup.
6. Downloaded the built-in Recommended alias through the actual frozen collector,
   verified its pinned immutable GitHub bytes, uploaded and booted a separate fake.

All six update/repeat boundaries preserved running/stopped intent, configurations,
identities, receipts, access/SSH records, ports, pier/mount objects and ownership,
runtime container/PID/start time, and engine invocation. The final installed
executable matched the official runner hash. All disposable piers were cleanly
stopped and retained afterward; the VM was powered off.

Evidence: `build/rc12-live/frozen/frozen1529-summary.json`, SHA-256
`2847a4b8ce8d0a482b4816857fd3e2c606e716318e6cadb25d154e2892206108`.
The sourceport user's two existing comets retained their original containers,
PIDs and start times throughout the recorded host observations.

## Legacy Ames migration and authenticated connectivity

The separate legacy VM created a new native comet and enrolled it through the
original RC.8 writer. RC.11/schema19 was used as fixture-lineage preparation, then
the official RC.12 offline migration preserved the stopped legacy enrollment and
a running separately generated peer. This is not a general RC.8 host-upgrade claim.

Explicit `publish-ames` configured the legacy ship's UDP34471 while stopped,
preserving its identity, filesystem objects and original enrollment receipt.
Exact replay passed before/after a management restart. Its native-off Start
resumed the existing pier, published the actual UDP binding and became healthy.

Only the two new disposable comets then received explicit native opt-in. Reviewed
active kernel/application sources established the remote Gall/Ames acknowledgement
contract. Fresh nonce-bearing `%helm-hi` pokes passed in both directions:

- Outbound-only peer → migrated/published peer: exact remote success,1.092seconds.
- Migrated/published peer → outbound-only peer: exact remote success,0.841seconds.

Both 106-byte terminal Newt replies were fully drained and independently checked
with the pinned Vere codec. All32 per-ship reviewed source hashes and runtime,
filesystem, configuration and manager identities were unchanged across probes.
Both ships were cleanly stopped and their VM powered off after collecting evidence.

Evidence: `build/rc12-live/legacy/network-qualification-1529/qualification.json`,
SHA-256 `ef908582a287928be97295a837f40196a64a9f68cf3ff3fa37b6ee0ef42bed4c`.
This proves authenticated controlled-peer roundtrips, not public unsolicited
inbound reachability, direct routing or that these packets traversed the new mapping.

## Frozen desktop and real clipboard

All eight mapped cases passed against the exact release archive and matching
canonical native wrappers: direct/native panel, direct/native import recovery,
log watching, onboarding and direct/native monitor focus. Actual fixture IPC
reported RC.12/schema20. Existing windows/workspaces and focus were preserved.
There were no application QML errors/binding warnings; two Quickshell scanner
and eight Qt portal warnings were recorded, with imports/assertions succeeding.

Evidence: `build/rc12-desktop-frozen-evidence/final-report.json`, SHA-256
`805f5892a00d42b944e35381b6c84825e311d812b80047507cc2da9e8a124a77`.

After explicit permission to replace the current clipboard, the frozen `omabit`
binary used synthetic RPC codes and the real sensitive foreground `wl-copy`:

- Expiry passed at30.039seconds, including terminated provider, no bytes available
  from the stale offer, and an empty clipboard for a fresh consumer.
- A replacement installed at5.053seconds remained readable at33.047seconds.
- Sensitive MIME/history filtering passed; actual history-file metadata was
  unchanged. No old clipboard contents were read or logged; both tests ended empty.

Reports: `/tmp/opencode/rc12-clipboard-expiry4/report.json` and
`/tmp/opencode/rc12-clipboard-replacement2/report.json`; SHA-256 respectively
`8e95de34ff6230ef5c910dd563a7685bf8e61eea6d54ed3b59b8dc1a4ffb7d40` and
`a04300a85f2761cd1b6860dc9be23f5c3d23b097a5e1349db2a1414fa6781c13`.

## Post-publication acceptance

All eight public assets were downloaded again after publication and matched both
the local frozen bytes and GitHub's reported digests. Complete package checks,
provenance/schema, executable smoke and isolated installation passed again.
An empty private native cache downloaded all three tools over their public HTTPS
URLs, verified size/SHA-256, and passed each canonical launcher plus schema20.
The pin-file hash is `18ac46b34c96dd53e1214797684055c7a3eb1e1af60977ceeb8cbcf21a55a693`,
identical to the pins used for frozen mapped-desktop qualification.

Evidence: `build/rc12-release/published-verification/report.json`.
The stock Omarchy cold-update result is recorded in a follow-up evidence commit
after these distribution pins become public; the frozen binaries are not rebuilt.

## Scope retained

This is RC.12, not stable0.1.0. The initial installed-host bridge is specifically
owner-user Linux/AMD64 schema19; database-reader coverage is broader. The complete
supplied-pill application inventory, public inbound-network behavior, and the
known upstream native pending-reply disconnect limitation are not newly claimed.
Existing piers and unrelated host resources were preserved. Native plugin update
does not implicitly migrate the user's runner service.
