# Omabit 0.1.0-rc.1

First public **prerelease** of the Linux/AMD64 ship-management stack, consolidating
development checkpoints through 0.23. Final 0.1.0 is still pending.

## Included

- Rootless, pinned official Vere lifecycle and maintenance; local and identity-pinned
  remote management; fleet/diagnostics UI for Omarchy.
- Bare bootstrap and Recommended Apps choices for new ships.
- A **frozen draft** Recommended Apps pill, separate from the binary bundle.
- Prebuilt desktop CLI, collector and runner; binary-bundle installer, SHA-256
  checksums, source/build metadata and qualification report.
- No fixed registration-count cap; 4 GB pill provisioning and 100 GB stopped-pier
  content bounds, subject to resource/storage/port admission.

## Install

Download `omabit-0.1.0-rc.1-linux-amd64.tar.gz` and verify it against `SHA256SUMS`.
Extract it, verify the inner `SHA256SUMS`, then run `./install.sh --enable`.
For an existing unmodified owned plugin, use `./install.sh --replace --enable`.
The bundle's README has host setup/update prerequisites and commands. Go is not
required to use the package. Standalone binaries are provided as separate assets.

**Compatibility:** Linux x86-64, Omarchy/Quickshell for the desktop, and
systemd/cgroup v2 with the verified rootless Docker dependencies for hosting.
Management schema 13 upgrades schema 11/12 records; older runners refuse that
database afterward. The runtime remains pinned to official Vere 4.6; this candidate
does not add automatic runtime updates.

## What passed

The exact packaged runner and collector booted a new comet from the frozen pill on
sourcebox. First boot reached HTTP/identity health, a 60-second observation passed,
and clean stop → same-identity initialized resume → final intentional stop passed.
The test ship remains stopped. Existing managed states, container IDs, revisions,
policies and handoff fences were preserved against the fresh baseline.

Go tests/vet, targeted CI race checks, desktop model/installer/mapped QML fixtures,
and isolated binary-bundle install/update/content verification passed.

Build/tag revision: `d67ea3ad6d1a3d768bf22bea73e594d8c799742e`.
See `release.json` for hashes/provenance and `qualification.json` for the exact scope,
operations and observed issues. Documentation commits after that revision do not
rebuild the tested binaries.

## Known draft issues

- **MCP and Omachan are missing.** A replacement pill with MCP and draft Omachan is
  being prepared; it will not replace these frozen rc.1 bytes.
- **Grubbery reported 23 files under `/code` that did not compile.**
- **Azimuth snapshot loading logged HTTP 504/content-length failures on boot and
  resume.** Complete application assets, connectivity and publisher OTA have not
  been accepted. Process/HTTP health is not full application readiness.
- Two pre-existing sourcebox ships were already recovery-required after a later
  host reboot, before this candidate deployment. Their state was preserved; this
  qualification does not resolve that separate incident.
- Clean-machine host installation, broader forced-interruption/network-partition
  recovery, inbound Ames and full app UI/OTA qualification remain open.

Pill size: **57,011,409 bytes**. SHA-256:
`51f0be723918ab7feaaf28b3c599f36f0f0fca6e68453f84517ba1659f07d57e`.
The automated pill builder remains deferred to **0.2.0**.

Tend is maintained separately at <https://github.com/thelifeandtimes/tend>.
