# Remote runner install on Ubuntu 22.04

On 2026-11-06, a fresh Ubuntu 22.04.5 amd64 cloud guest completed SSH owner
setup from missing native packages through a healthy managed fakeship. A second
desktop paired and revoked its own access while that ship continued serving
HTTP. This exercises the 0.4.0 development installer on the distribution family
recorded for rooftop (22.04.3). The lab used the normal CLI behind the desktop
setup terminal and isolated desktop configuration directories. Rooftop and the
desktop GUI were outside the experiment.

## Reproduction inputs

- Canonical Jammy server cloud image, SHA-256
  `f77bc6d59cb8deaa9657095736b5f5003f7705b47055f01823459715c967de6a`.
  Its checksum list verified against Ubuntu cloud-image signer
  `D2EB44626FDDC30B513D5BB71A5D6C4C7DB87C81`. The release directory moves
  as Ubuntu refreshes images. [Ubuntu cloud images](https://cloud-images.ubuntu.com/releases/jammy/release/)
- KVM, 2 vCPUs, 7 GiB RAM, 12 GiB virtual disk, kernel
  `5.15.0-203-generic`, systemd/cgroup v2. Each fresh attempt used a new overlay
  of the same read-only base. Cloud-init supplied test SSH access and isolated
  the guest from private networks. Only SSH was forwarded to host loopback.
- The standard Ubuntu archive and security repositories included Jammy main and
  universe with updates and security. The fresh plan requested `docker.io`,
  `containerd`, `runc`, `rootlesskit`, `slirp4netns`, `uidmap` and
  `libarchive-tools`. The resulting native package roles were:

| Roles | Ubuntu package | Installed version |
| --- | --- | --- |
| Docker engine and CLI | docker.io | 29.1.3-0ubuntu3~22.04.2 |
| Container runtime | containerd | 2.2.1-0ubuntu1~22.04.2 |
| OCI runtime | runc | 1.3.4-0ubuntu1~22.04.1 |
| User namespaces | rootlesskit | 0.14.6-1ubuntu1.1 |
| Userspace networking | slirp4netns | 1.0.1-2 |
| UID and GID mappers | uidmap | 1:4.8.1-2ubuntu2.2 |

The candidate was built with Go 1.27.3 for Linux/amd64 using `CGO_ENABLED=0`
and `go build -mod=readonly -trimpath` on base `6031878` with the installer
changes accompanying this record. Tested runner SHA-256:
`6dd8367e8ebe7180d5d97399d212d0b15d0879aad0680948005cf0ab972ee12f`.

## Failures and fixes

1. The pinned launcher requested `--detach-netns`. Jammy's RootlessKit 0.14.6
   rejected the flag before the socket appeared. Owner setup now reads the
   authenticated native helper's options and selects the launcher's attached
   network-namespace mode when needed. Both exact generated unit forms are
   recognized by later installer/guard runs. An unrecognized help response or
   locally modified unit still requires review.
2. With that fix, pairing and inventory passed but Docker rejected the first
   ship's `--cpus` setting. The kernel supported CPU quotas, while Jammy's
   `user@.service` delegated only memory and pids. Fresh owner setup now installs
   an additive `Delegate=cpu memory pids` drop-in for the selected UID and reloads
   systemd before the private engine first starts. It checks the running engine's
   CPU quota/period, memory and process-limit support before completing activation.
   Established management updates and access-only pairing retain their existing
   startup policy.
3. A 3 GiB test guest hit the expected host admission limit. In the larger guest
   an explicit loom of 30 (1 GiB) exhausted Vere's heap during bare pill boot.
   With Omabit's default loom of 31, the disposable ship reached healthy HTTP.
   Both observations are separate from the installer fixes.

## Acceptance observations

- Declined the fresh-install prompt, then resumed by setup ID. The attempt
  retained its key fingerprint. The new dedicated connection returned the
  expected runner ID, operator role and empty inventory. The collector's
  `host-check` accepted signed native evidence for all eight package roles and
  reported a compatible rootless engine with permission to mutate. Ubuntu's APT
  post-install also started the distro Docker service. All Omabit container
  checks and operations used the owner's separate socket.
- Started a bare `~nec` with the default loom through the collector. The start
  operation succeeded, the runner reported healthy, and the guest's HTTP
  `/~/host` returned `~nec`. Its live cgroup enforced a 2-CPU quota
  (200000/100000), 4294967296-byte memory limit, and 256-process limit.
- Paired an independent desktop through the existing-runner prompt. It reported
  the same runner and healthy ship under a different key. Before pairing,
  after pairing, and after revoking the second key, both service PIDs and
  activation times, the ship container ID, PID and start time, the cgroup limits,
  and the HTTP identity matched. The first desktop retained access and stopped
  the ship through the normal operation.
- On another fresh overlay, interrupted the approved setup at the engine stage,
  after APT finished and the owner profile was written. The guest retained a
  prepared engine activation and an installing remote journal. After explicit
  approval of the interrupted-install prompt, the same attempt, fingerprint and
  helper SHA-256 completed setup and returned an empty authenticated inventory.
  After a full power-off and cold boot, the same collector verified the runner
  again. Both user services were active and Docker still reported all four
  resource-limit capabilities.

To repeat, point a private SSH configuration and two disposable desktop profiles
at a clean guest, then use the following normal entry points:

1. Run `omabit remote-setup --host ALIAS --ssh-config FILE --owner ubuntu --name jammy`.
   For an interrupted attempt use `omabit remote-setup --id SETUP_ID`. Read the
   setup ID and expected runner ID with `omabit remote-setup-status`.
2. Run `urbit-collector --runner jammy --expected-runner-id RUNNER_ID` with
   `hello`, `snapshot` and `host-check`. With that same prefix, run
   `create-fake jammy-default --http-port 8089 --ship '~nec' --bootstrap bare --request-id UNIQUE_ID`,
   then `operate jammy-default start --revision 1 --request-id NEW_ID`.
   Use fresh 32-character hex request IDs. Poll the snapshot until the start
   operation succeeds and the ship is healthy.
3. Pair from the second profile using the same alias. Compare the live process
   identities and `/~/host` before and after pairing and revocation. Use the
   first profile to stop the ship and wait for the stopped observation.

The full Go suite passed, as did focused race tests and vet for the installer,
guided remote setup, onboarding, rootless engine, and host setup packages.

This is fresh-install development evidence. The release-wide 0.3.7 upgrade,
other distro images, and any individually authorized rooftop trial have their
own gates. No release qualification tuple was added.

## Lab disposition

The test root was private temporary storage with an 8 GiB scratch budget.
The two retained historical evidence directories measured 18 GiB and 8.2 GiB
before and after. The test overlays used one shared, read-only base. Each used
its own disk directly backed by that base.

Removal manifest for this run: the RootlessKit and CPU-reproduction overlays;
the healthy-workload and interruption-recovery overlays after shutdown; the
downloaded base and signed manifests; cloud-init seed, disposable keys and
desktop profiles; raw logs, test binaries, temporary harness and source clone.
The development worktree holds the source, tests and compact evidence.
