# Omabit 0.1.0-rc.13 — distribution-native host compatibility

RC.13 is a Linux/AMD64 **prerelease** for continued installation testing, including
the planned fresh-install checks on two Omarchy versions. It introduces schema21
and retains the existing desktop layout and runner/collector responsibilities.

## Changes

- Stable Docker **27.2.0–29.x** replaces the exact29.7.2 admission requirement.
  Actual API1.47, rootless, owner/socket identity, systemd cgroup-v2 and data-root
  checks still apply. The pinned rootless launcher is separate from engine support.
- Pacman and Debian/Ubuntu providers authenticate native package signatures,
  repository authority and installed payload bytes. Root-issued retained receipts
  survive ordinary package-cache cleanup; native transaction hooks refresh them.
- Explicit download-only verification can retrieve missing evidence for the exact
  installed builds. It does not install packages, change repositories or import keys.
- Local `host-check`, read-only runner RPC and collector fleet reports distinguish
  installed/running versions, compatibility, authentication, qualification and
  restart-pending status. Collector-only machines require no local Docker stack.
- Omabit presents the shared reports in existing setup/monitoring areas. Fresh,
  identity-bound evidence governs actions; observation and Stop remain independently
  available when their runtime identity checks pass.
- Known-content startup-guard migration, interrupted initial setup recovery and
  existing subordinate-ID preservation support ordinary native package updates.
- Bootstrap admission/recovery distinguishes proven-undispatched work from real
  uncertainty, preserving durable recovery and avoiding duplicate bootstrap.

## Fresh installation

Inside an Omarchy desktop, as the ordinary user:

```sh
omarchy plugin install https://github.com/thelifeandtimes/omabit.git --enable
```

Open **~+ → Setup → Set up this computer**. The native checkout's checksum-pinned
tools, root manifest and monitor status must identify **0.1.0-rc.13**. Hosting
packages are provided by the OS; the setup flow verifies them and requests OS
authorization for the existing-user services. See [host compatibility](host-compatibility.md)
for native prerequisites, reporting and package-evidence guidance.

The release also includes the complete desktop archive and three standalone
executables; see [binary installation](release-install.md).

## Existing hosts and schema21

RC.12/schema20 owner-user hosts use the ordinary management update, preparing a
**new rootless bundle with the new executable** first. RC.10/RC.11/schema19 hosts
use the documented offline bridge. Both paths preserve historical records and
identities; neither requires deleting or re-enrolling piers. See the
[upgrade procedure](host-compatibility-upgrades.md).

Schema21 accepts historical schemas11–20 without rewriting original instance
bodies, signed handoffs or receipts. New containers record their observed engine
version; historical omissions stay omitted. Once migrated, schema20 and older
binaries are not rollback targets. Retain the target executable, activation journal
and generations for forward recovery.

## Verification and scope

Source tests/vet, targeted race checks, JavaScript and actual offscreen Qt tests,
all14 original-writer migration/refusal fixtures, and native Ubuntu/Arch package
authentication tests passed during development. Native testing used signed inert
synthetic packages, including four real package transactions and negative authority,
tampering and cache-cleanup cases. See the [development verification record](diagnostics/host-compatibility-development.md).

Artifact-specific build, checksum, provenance, installation and CI results are
recorded in the attached `qualification.json`. Development evidence is not relabeled
as a fresh full-stack run of the final release binaries.

Docker29.9.1 is contract-compatible when its checks pass, but is **not yet
live-runtime-qualified**. Fresh-install results on the two requested Omarchy versions
are pending user testing. RPM providers and full Docker/Vere qualification across
new distribution/engine profiles remain future work. This is not stable0.1.0.

The official Vere4.6 image and Recommended Apps pill bytes remain pinned to their
existing content. RC.13 adds no custom engine or package distribution.
