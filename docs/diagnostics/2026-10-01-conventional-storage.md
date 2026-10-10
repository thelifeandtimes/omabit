# Conventional pier storage — panel 0.16

Final verification: panel 0.16.0 is loaded on Setup. Built, host-installed and
panel-bundled runner hashes match (`79a3ed2d9615a5bf2dd9eca04601954385f87e21c595d48f97a7e8c265fe78c1`).
The complete Go suite, vet and post-fix runtime/store/filesystem race checks passed.
Final inventory has seven registrations, five correctly associated discovered
piers, three healthy running ships, no active work, recovery requirements or incidents.

## Live migration accepted

The user authorized the convention and migration. The management database was
snapshotted with SQLite online backup and checked before the service update:

`/var/lib/urbit-runner/.local/share/urbit-runner/managed/before-pier-layout-16.sqlite`

The five initialized piers were inspected while stopped before and after moving.
Each retained exactly the same epoch entries, inventory digest and private pier ID.
No event database or checkpoint was rewritten by the move. The three previously
running ships resumed healthy with the same identities/configuration revisions;
intentional stops and uninitialized registrations were preserved.

| Instance | Path relative to `/var/lib/urbit-runner/piers` | Final state |
| --- | --- | --- |
| alpha | `fakeships/zod` | Running / healthy, 2/2 |
| bus-lab | `fakeships/bus` | Running / healthy, 1/1 |
| comet-check | `livenet/comets/comet-check` | Running / healthy, 1/1 |
| import-check | `fakeships/import-check/zod` | Stopped, 1/1 |
| panel-check | `fakeships/panel-check/zod` | Stopped, 1/1 |
| rus | `fakeships/rus/zod` | Uninitialized, 1/0; path reserved only |
| ssh-check | `fakeships/ssh-check/zod` | Uninitialized, 1/0; path reserved only |

All saved policy settings/revisions remained unchanged. Existing live container
replacement was intentional as part of stop/move/resume, not a management-restart
side effect.

### Integration finding and explicit recovery

The first alpha resume stopped at strict containment: binding a child at
`/urbit/pier` left Docker's image-declared `VOLUME /urbit` as an extra anonymous
volume. The probe had not started Vere; alpha's relocated data was intact.

Read-only Docker inspection confirmed both mounts. The driver now binds each
initialized pier directly at `/urbit`, and the fake Docker fixture models that
image-volume behavior so a regression fails tests. The failed probe was explicitly
removed, the corrected runner installed, and alpha resumed healthy via a new
request. The unused anonymous volume created by that probe was removed by exact ID.
The already completed moves were not repeated; an explicit `--only bus-lab,comet-check`
run completed the remaining two ships.

Receipts: `/tmp/opencode/omabit-storage-migration.json` (original attempt, retained)
and `/tmp/opencode/omabit-storage-migration-remaining.json` (remaining work).
Key recovery operations:

- Probe cleanup: `op-92e32fb7e0d60479ffd95c04cd0656c5`
- Corrected alpha resume: `op-61fed9ab46b718df3afeb3a9ae40a0e0`
- bus-lab final resume: `op-9fdfa65140da090b01d7e7dc1e5d0d03`
- comet-check final resume: `op-f9263ab8adf46f9be3f6a1a7f64f4f55`

## Direct filesystem acceptance

The stopped panel-check pier was renamed outside management to
`fakeships/panel-check/moved-zod`. Polling reported its original location missing;
discovery identified the moved filesystem object without rebinding or starting it.
Explicit fingerprint/revision-bound reconnection succeeded, then a managed move
restored `fakeships/panel-check/zod`:

- Reconnect: `op-801844e67cbadc3bb917596ae3518bb7`
- Restore location: `op-a18d558acc16d974af3891bd74617d70`

A real stopped copy was then made at `fakeships/discovery-validation-zod`. It
appeared as unenrolled, unlocked and without storage problems. It was never
registered or started. Moving it to
`archive/panel-check-zod.20261001.bak` removed it from discovery, leaving exactly
five discovered initialized piers, all associated with the correct registrations.
That inactive backup remains in archive.

## Implemented and checked

The runner/collector/panel support conventional new locations, bounded discovery,
explicit enrollment/reconnection, stopped no-overwrite moves, missing/replaced
directory refusal, key-file registration and isolated-first-resume livenet import.
Tests cover namespace traversal/archive/symlink refusal, metadata preservation,
duplicate physical/path reservations, bootstrap promotion, lifecycle/maintenance
path use, private key staging/replay without journal disclosure, and import resume
without creation flags. UI fixtures cover discovery consent, fresh fingerprints,
runner binding and key-file-only helper routing.

Full Go/vet, targeted races, native/ARM64 builds, model/installer/QML tests and
screenshots passed before deployment. The corrected mount code passed runtime
regressions and the live resumes above. Real owned-key boot and owned livenet
import still await supplied test material; no existing personal key/pier outside
the managed development fleet was read or booted.
