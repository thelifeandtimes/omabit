# Docker tmpfs inspection correction

The gateway now accepts Docker's real inspection shape: eight exact bind mounts
in Mounts, with the two --tmpfs definitions in HostConfig.Tmpfs. Both definitions
remain mandatory, including permissions and size limits. Reported tmpfs entries
are still validated when present; partial/duplicate/unexpected mounts, changed
binds, writable inputs, extra writers and altered tmpfs configuration refuse.
The same correction applies to the overlapping-mount inventory during readiness.

The new regression and actual combined waiting fixture both fail against the
previous source. The corrected source passes all 354 private-gateway race checks,
then all 354 again after integration, plus vet and diff checking. Observer and
first-start fixtures now model Docker's eight-bind layout rather than inventing
tmpfs entries. Runtime permissions, commands, schema43 and public routes are
unchanged. This supersedes M401's unfixed tmpfs-contract defect only; a new VM
must still qualify initialization and manager recovery.

The failed VM01 disk is privately retained with its unchanged audit hash and
compatibility path. Three historical immutable appliances (08,13,15) were also
copied, hashed, fsynced and retained with compatibility links to restore temporary
storage headroom. No resource-admission floor was lowered, and no unrelated files
or installed services were modified. The next prepared appliance is25.

Docker source separates [TmpfsMounts from GetMountPoints](https://github.com/moby/moby/blob/v28.3.3/container/container_unix.go#L370-L411);
this matches the stopped-disk metadata recorded in M401. These checks are source
evidence, not proof of a successful current managed runtime.
