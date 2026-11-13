# Stock Debian 13 remote runner setup

Fresh SSH setup on Debian 13.7 amd64 stopped at the Docker candidate check
after refreshing APT metadata. The configured repositories offered
`docker.io 26.1.5+dfsg1-9+deb13u1`. Omabit requires stable Docker from 27.2.0
up to, but excluding, 30.0.0 and a running daemon supporting API 1.47.
The tested outcome is an actionable refusal and recoverable attempt. Successful
Debian provisioning with another native package source is still unverified.
[Debian's Trixie docker.io package](https://packages.debian.org/trixie/docker.io)

## Inputs

- Debian genericcloud amd64 qcow2, release 13.7, verified against the
  SHA-512 manifest downloaded over HTTPS from the same distribution directory.
  Digest: `d25999de0770d4cde1be72c299ba3ad94eac7ea8fd4d18f8cc86c69df69337e2e8a91d64465a9bfd8f2a78c566fcbcb0842eacc91c09efdef72d719d9f13b535`.
  [Debian cloud image directory](https://cloud.debian.org/images/cloud/trixie/latest/)
- KVM/Q35 with UEFI, two vCPUs, 7 GiB RAM, a 12 GiB virtual disk,
  kernel `6.12.115+deb13.1-cloud-amd64`, systemd 257 and cgroup v2.
  An initial legacy-BIOS launch exited at GRUB. The two test runs each used
  a new UEFI overlay, a private SSH configuration and isolated desktop state.
- Both guests used the image's stock Deb822 sources with Debian's archive
  keyring. The mirror lists pointed to `https://deb.debian.org/debian` and
  `https://deb.debian.org/debian-security`. The suites were Trixie, updates,
  backports and security, component main. `apt-cache policy` and `madison`
  showed the same eligible binary version after refresh.
- The baseline used the 0.4.0 rooftop trial binaries (runner SHA-256
  `6dd8367e8ebe7180d5d97399d212d0b15d0879aad0680948005cf0ab972ee12f`).
  The revised run used base `2cae8d7` plus the accompanying candidate
  diagnostic change (runner SHA-256
  `2bacf1186229bf2fedde6fff264e4dd6c2296aafba61b7c75d0cc87170241d52`).

## Observations

1. Through the regular `omabit remote-setup` terminal, approved initial
   owner setup for the guest's existing `debian` user. The trial installer
   detected the incompatible candidate, but its diagnostic omitted the
   supported range and the command needed to check the host.
2. On a fresh overlay, repeated the flow with the revised helper. The terminal
   and `omabit remote-setup-status` reported the candidate, supported range,
   `apt-cache policy docker.io`, and **Resume setup** after a compatible
   candidate becomes available. The desktop status model accepted both saved
   responses with their full diagnostic.
3. Resumed using the same setup ID. The same key fingerprint and installer
   hash remained bound. Setup checked the configured repositories again and
   returned the same diagnostic. The collector had zero enrollments.
4. Compared the guest's dpkg status, APT sources and mirror lists, and owner
   authorized keys against their pre-test SHA-256 hashes. All matched. No owner
   profile, runner/engine units or pier root existed after either attempt.
   Setup retained its retry journal and refreshed package indexes.

The source change affects the existing native candidate check. It reads APT's
`Candidate` or Pacman's `Version` field and uses the shared engine compatibility
policy. Tests cover the observed Trixie package, no candidate, a missing field,
the Jammy candidate, and supported/unsupported Pacman versions.

## Lab disposition

A single guest ran at a time. The experiment used approximately 1.3 GiB of a
5 GiB scratch budget, separate from the retained 18 GiB and 8.2 GiB historical
lab/evidence directories. Removal manifest: the private test root, comprising
the three disposable overlays and their common read-only base, UEFI variables,
seed, test SSH keys, desktop profiles, binaries, logs and temporary clone.
The development worktree preserves the source changes and this record.
