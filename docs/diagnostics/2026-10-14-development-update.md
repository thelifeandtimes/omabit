# Full development update — 2026-10-14

**Deployed:** the sourceport desktop and both managed runners now use the current
**0.1.0-rc.2-dev** development build. Both runners use **schema 14** and advertise
`bootstrap-selection`, `storage-locations`, `import-pier` and `delete-pier`.

## Included updates

- Confirmed pier deletion from ship Overview and registered storage locations.
- Configurable host storage roots and reviewed host-path move/import.
- Revision-safe instance-name reuse after deletion and persistent operation history.
- The btrfs stable-directory-identity fix from PR #1, including the new storage paths.
- The dedicated SVG logo, muted active-icon color and workspace-aware monitor focus.
- Clearer recovery status, fake-networking labels and bootstrap selection.

The default storage layout, parent-directory container mounts, official Vere 4.6
image and 60-minute snapshot policy are preserved. The Recommended Apps default
still points to the published rc.1 draft pill; this was not a new pill release.

## Build provenance

Built with **Go 1.25.0**, Linux/AMD64, `CGO_ENABLED=0`, `-trimpath`. The source base
was `bf854ad5a789b7f30e20e6ed3fe6fe1f4c1cd60e` plus the pending development changes.
The private deployment directory `build/deploy-20261014/` contains the frozen
binaries/plugin/install inputs and a `provenance.json` inventory of source-file
and artifact hashes. This is an explicitly modified-tree development deployment,
not a newly published GitHub release.

| Artifact | SHA-256 |
| --- | --- |
| `urbit-runner` on both hosts | `76b42bb5c26c682a95e914c1274a6c318623a1cd1a48860f498552e45f0dd49b` |
| Bundled `urbit-collector` | `4a060b6f0e3bf3fffa5d0d9e5c9a754b42c75e94d24da9c1912c28b84fa62dc0` |
| Bundled `omabit` | `d61b8724afd17c0d86dbdc53b08e518924d588cfed027bdf5cf4e187e49437e6` |

## Deployment and verification

1. Captured a fresh, idle fleet baseline: eight local registrations and four remote
   registrations, including seven running/healthy ships.
2. Full Go tests and vet passed. The frozen plugin passed real-compositor focus
   checks with isolated collector fixtures.
3. Each host retained a private pre-upgrade management database snapshot and runner
   executable under `/var/lib/urbit-runner-upgrades/deploy-20261014/`. These are
   diagnostic records, not an automatic downgrade/restore mechanism.
4. Authorized `install-host` updated sourceport **12 → 14** and sourcebox **13 → 14**.
   Both returned ready, retained their pinned runner/SSH/operator identities, and
   passed installed runner hash and protected-registration/storage comparisons.
5. The approved cold-replacement installer installed the complete frozen desktop.
   Live monitor IPC reports `0.1.0-rc.2-dev`; its new QML components, SVG, ownership
   marker and all three bundled binary hashes match the deployment.
6. Evaluated the installed GUI's deletion availability rules against live inventory.
   Delete was enabled for stopped `import-check` and uninitialized `rus`/`ssh-check`
   locally, and stopped `rc1-pill-check` remotely. Running instances and the local
   moved-source record remained guarded. No deletion request was submitted.
7. Final comparison at **04:23:52 UTC** preserved all 12 registrations' identities,
   configuration/applied revisions, paths, desired/observed states, health, policies,
   bootstrap metadata, move fences and container IDs.

Running local ships: `alpha`, `bus-lab`, `comet-check`,
`dibpen-fogweg-dozzod-sarlev`. Running remote ships: `panel-check`, `remote-comet`,
`remote-smoke`. Intentional stops/uninitialized entries and the local moved-source
fence stayed as recorded.

## Where the controls are now

- **Delete:** select a ship → Overview → Connectivity & runtime → **Delete pier…**.
  Stop a running ship first, wait for completion, then review and type its name.
- **Import/storage:** Setup → **Import a pier & storage locations**.
- **Location selection:** the new-ship and import forms select from configured roots.

See [storage management](../storage-management.md) for host permissions and import
interruption behavior. Live deployment/capability checks do not replace the separate
real-pier destructive/import qualification gates. Published rc.1 downloads still
contain the earlier software, so the clean-install walkthrough's pinned assets
must be updated before using it to exercise these newly deployed features.
