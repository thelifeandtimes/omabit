# Omabit 0.1.0-rc.2

Second Linux/AMD64 **prerelease** of the Omarchy desktop, collector and dedicated
rootless runner. This candidate is intended for fresh-install and multi-collector
experience testing.

## Changes since rc.1

- Stable pier identity across btrfs device renumbering (PR #1).
- Named, identity-pinned storage roots; reviewed host-path pier move/import.
- Confirmed stopped-pier deletion, durable audit history and revision-safe name reuse.
- Automatic existing-pier resume after an unclean host reboot, using durable healthy
  boot/container/revision evidence and at most one attempt per runtime per host boot.
- Dedicated theme-tinted SVG icon, muted active color and workspace-aware monitor focus.
- Clearer current-runtime versus historical-operation and fake-networking labels.

## Installation and compatibility

Download `omabit-0.1.0-rc.2-linux-amd64.tar.gz` and verify the published checksums.
Extract, verify the inner checksums, then run `./install.sh --enable` as the desktop
user. Existing unmodified owned installations use `./install.sh --replace --enable`.
Host setup is a separate explicit step. Go is not required.

Linux x86-64, Omarchy/Quickshell and Python 3 are required for the desktop. Hosting
requires systemd/cgroup v2 and the verified rootless Docker prerequisites. Schema 15
upgrades 11–14 additively; earlier runners refuse the new database. Vere remains
pinned to official `4.6-5a7614f`, with 60-minute checkpoints and the existing limits.

## Qualification

See the attached `qualification.json` for exact build provenance, checks and live
smoke results. A real sourcebox hard reset passed automatic resume for two fakes and
a comet, preserving identities/configuration and the intentionally stopped ship.
The [record](diagnostics/2026-10-15-hard-reboot-acceptance.md) distinguishes lifecycle
and HTTP readiness from full application/network health.

## Known scope and limitations

- The Recommended Apps asset reuses the frozen rc.1 **draft** pill, 57,011,409 bytes,
  SHA-256 `51f0be723918ab7feaaf28b3c599f36f0f0fca6e68453f84517ba1659f07d57e`.
  MCP/Omachan are missing; the prior pill gate found Grubbery compilation and
  Azimuth HTTP 504 snapshot failures. Full app UI/OTA acceptance is pending.
- Interrupted host import, deletion, handoff or maintenance has phase-specific
  recovery rules; it is not equivalent to restarting an ordinary initialized Vere.
- Cross-device import and forced-interruption cases have fixture coverage; consult
  qualification metadata for live coverage. No historical-pier restore UI is offered.
- The sourcebox fresh-desktop install and complete independent-collector UX matrix
  are the next user-driven gates, not claimed as already accepted by this release.
- Startup can log optional mDNS and external-network warnings even after successful
  local identity/HTTP readiness. Broader Ames/application qualification is separate.

Published rc.1 assets are unchanged. Tend remains a separate project and command.
