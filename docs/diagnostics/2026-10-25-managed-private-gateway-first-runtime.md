# Current managed private gateway: first runtime attempt

A fresh no-NIC VM ran the current schema43 runner/backend with sealed generation07
inputs and its own rootless Docker engine. The guarded fixture uses the actual
internal queues and adapters. It is intended to lose one successful readiness
reply, reopen the manager, record a separately attributed read-only recovery,
and reopen/replay again. Fixture-only input delivery is not a supported installer.
Six delivery checks, armed host refusals, compilation and tagged vet pass.

This runtime attempt failed. Registration, workspace creation, network/container
creation and Start completed. The source receipt remained interrupted at started;
no successful readiness observation was captured. Recovery and verification were
not attempted. The VM powered down cleanly. Three independent read-only audits
left the disk unchanged (SHA256 9b184754e01064742eb6d725e3a2cd51cfe929ef77545afe129df70145c4c321).
They found schema43, one private project/workspace/gateway, three operations and
zero ship instances or recovery observations. No production ships were involved.

The container ran 24.710 seconds before guest shutdown (exit143, no OOM). Neither
readiness nor evaluator/failure files existed. Its stored Docker MountPoints has
the eight exact bind destinations and no tmpfs entries. The observer currently
requires both tmpfs entries in Mounts while Docker keeps --tmpfs separately in
HostConfig.Tmpfs. This is a confirmed contract defect; a corrected fresh runtime
must still establish whether additional readiness failures remain. See Docker's
[TmpfsMounts and GetMountPoints implementation](https://github.com/moby/moby/blob/v28.3.3/container/container_unix.go#L370-L411).

All original runtime/audit scopes are inactive. Private disks, key material and
raw logs remain private; evidence includes only source, hashes, known diagnostic
codes and bounded metadata projections. Installed plugins, existing Docker,
protected ships, tmux ships and ~/temp-ur were not accessed.

M401 supersedes M400's fresh-current-source-VM-not-yet-attempted statement only.
It does not qualify the managed lifecycle or supersede earlier isolated peer
application recovery evidence. Next: reproduce/fix the tmpfs inspection mismatch,
then test with a new disposable VM and unchanged isolation/resource floors.
