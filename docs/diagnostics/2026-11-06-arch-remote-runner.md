# Arch remote runner installation and first ship

Two independent overlays of the official Arch Linux x86-64 cloud image exercised
remote setup from private desktop profiles. The first run reached a healthy bare
fake ship and exercised second-desktop pairing and revocation. The second run
verified the improved package-failure diagnostic, same-attempt recovery and
healthy first ship using the revised helper. This record uses the operator's
2026-11-06 Pacific date. The guest logs use 2026-11-07 UTC.

The guest covers the Arch runner path underlying Omarchy. A complete fresh
Omarchy desktop install remains outside this matrix. The desktop commands ran
on the developer's Omarchy machine with private HOME and all four XDG roots.

## Inputs

- Official `Arch-Linux-x86_64-cloudimg-20261101.620452.qcow2`, SHA-256
  `794eb4889b39e1f0ed7d5ae998b2a1ed2137db8cf8c3476cf2962c7437ecb003`.
  Verified the signed checksum manifest against the Arch Boxes key, primary
  fingerprint `1B9A16984A4E8CB448712D2AE0B78BF4326C6F8F`, signing subkey
  `656E4C5AC1CC3B86E539D97E343635A6859A9174`.
  [Official image directory](https://geo.mirror.pkgbuild.com/images/latest/)
  and [Arch Boxes key publication](https://github.com/archlinux/arch-boxes).
- KVM/Q35, UEFI, two vCPUs, 7 GiB RAM and a 12 GiB virtual disk. Each overlay
  started with the image's existing `arch` owner and fresh disposable SSH keys.
  A single guest ran at a time. Only its SSH port was forwarded to host loopback.
  Guest routes blocked private, link-local, CGNAT and host-gateway destinations
  while allowing SSH replies and public package/runtime downloads.
- Initial kernel `7.2.11-arch1-1`, updated kernel `7.2.14-arch1-1`, systemd
  `262.1-1`, Arch keyring `20260909-1` and unified cgroup v2. The stock mirrors
  were `fastly.mirror.pkgbuild.com` and `geo.mirror.pkgbuild.com`.
- Baseline source `03fc549cbd179c282cbedbdaf3af6acdea22a5a7`. The repeated run
  used that source plus this change's Pacman recovery diagnostic. Both builds
  used `CGO_ENABLED=0`, `-mod=readonly` and `-trimpath`.
- The source-pinned runtime was Vere `4.6-5a7614f`, image manifest
  `sha256:47fbcf0710ded939931ce3c6408864cee5e8848c5765da0b51a90ca0b1ab1afd`.
  The pinned `urbit-v4.6.pill` was 217421172 bytes, SHA-256
  `f8b08c53dd6d0aa9c4f7654f0a7bcd11e94089222888a3c99c0a10eedcbb377b`.

| Executable | Baseline SHA-256 | Revised SHA-256 |
| --- | --- | --- |
| omabit | `962b96d7c7a42189194e7abf23a9e5d7aabf29b025bacf1d67ded78e0f85e9d9` | `badb56050e278d5ee99cb973beac3a94b6cb631e8070d67b8ff596fb12b6cbe0` |
| urbit-collector | `daae9fadc71d06daf0fba4d0913d19a58d567e6c438164c929f6442b4fd8ce47` | `181d69dc3fe7e24477706423360fd1803496b9dd0cbab52ff6bcf671cbc31c1d` |
| urbit-runner | `1d8c232e5454a226147f01acb8519ace69cf69c5b824fb0b0fc2e162b64c5ead` | `7b2f51ff5f7dfe6ab5ba221be4f4a6bcad99c87eadb876558d9f2e0f535c88dc` |

## Install and recovery

The regular `omabit remote-setup` terminal detected an uninstalled owner and
requested approval to install its missing native dependencies. The image's
cached repository database selected Docker `1:29.9.2-1`. Both rolling mirrors
returned 404 for that package and Pacman aborted before committing packages.

The revised helper explains this specific recovery path in both the terminal
and saved setup status. Review the package-manager error, complete the host's
normal full update (`pacman -Syu` on Arch or `omarchy update` on Omarchy), and
resume. The installer continues to install only its approved missing dependencies.
Full host updates require the user's authority for that host.

In each disposable guest, ran `sudo pacman -Syu --noconfirm`, shut down, and
booted the same disk with its updated kernel. The original setup ID, dedicated
key fingerprint and frozen installer remained bound on resume. The revised
repeat enrolled with zero ships and operator access. Native package signatures
and installed executable payloads authenticated all eight roles.

| Roles | Native package version |
| --- | --- |
| Engine and CLI | docker `1:29.9.3-1` |
| Containerd | containerd `2.4.2-1` |
| OCI runtime | runc `1.5.2-1` |
| RootlessKit | rootlesskit `3.3.0-1` |
| Network | slirp4netns `1.3.7-1` |
| UID and GID maps | shadow `4.20.0.arch1-1` |

## Runtime and access

Each run's collector registered a disposable bare `~nec` and started it
with the default loom of 31. The asynchronous operation succeeded, the instance
reported `running`, `healthy` and `ready`, and HTTP `/~/host` returned `~nec`.
The rootless engine used the owner's `/run/user/1000/urbit-docker.sock`.
Actual container cgroups enforced `cpu.max=200000 100000`,
`memory.max=4294967296` and `pids.max=256`.

A second private desktop paired with the established runner using a distinct
restricted key and then revoked that key. Before pairing, after pairing and
after revocation, the engine and runner PIDs/start timestamps and the container
ID/PID/start timestamp matched exactly. HTTP identity and actual limits matched
as well. Both test ships stopped successfully through their respective
collectors before their guests shut down.

## Source and skill checks

The full `go test ./...` suite passed. Targeted race tests and `go vet` passed
for `internal/remotesetup`, `internal/hostinstall` and `internal/onboarding`.
The five skill packaging tests, skill frontmatter validator, local-link checks
and `git diff --check` passed. Packaging verifies that both the standalone skill
archive and the binary bundle carry every skill reference byte-for-byte.
The operator skill adds a dedicated
[remote-runner reference](../../skills/omabit-operator/references/remote-runners.md)
covering tested tuples, recovery, host-specific agent assistance and remaining
coverage boundaries.

## Lab disposition

The scratch budget was 5 GiB in addition to historical lab/evidence totals of
18 GiB and 8.2 GiB. Actual scratch usage reached 4.0 GiB. The removal manifest
covers only this run's two disposable overlays and their common immutable base,
UEFI variables, cloud-init seed, test keys, private desktop profiles, binaries,
raw logs and temporary source clone.
Both guests shut down cleanly. Both overlay backing chains resolved to the
single immutable base inside the private test root, and the process check found
zero remaining lab VMs. Retire the listed generated inputs after preserving this
record and the implementation in the development worktree.
