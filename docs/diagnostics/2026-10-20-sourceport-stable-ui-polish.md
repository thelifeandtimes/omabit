# Sourceport stable runner and UI-only polish deployment

The user requested updating the current runner and interface on sourceport.
The host remains Omarchy4.0.4-1; owner UID/GID1000, runner identity
`runner-70a48c60cb4e396c1b9628224f229a1a` and default desktop profile were retained.

## Runner

Upgraded RC.12/schema20 to the published stable0.1.0/schema21 binary, source
`9cc1d217df19c0aa9bcf6d92a5f8da500f022ffc`, SHA-256
`72bd87cdbb56b800730200e3f5c5c16d677355949ede3563f57dba5c9d9e1a37`.

Prepared a new rootless bundle with that exact released executable, then used the
normal OS-authorized `install-host` path with the existing owner and operator public
key. The initial attempt refused before host mutation because Docker/runc lacked
authenticated cached payload evidence. Restored the original signed packages for
installed Docker1:29.7.2-1 and runc1.5.1-1 from the retained RC.12 package inputs to
Pacman's native cache, without overwriting files or installing any packages.
The unmodified verifier then authenticated all eight required roles and the normal
installer completed the online management upgrade.

Post-upgrade host report: compatibility **compatible**, provenance **verified**,
restart-pending **false**, observe/Stop/mutate **true**. Its exact-stack qualification
label remains **unqualified**; native provenance is not relabeled as qualification.

## Desktop

The installed plugin checkout was clean at `ffaa8b1`. Stock
`omarchy plugin update thelifeandtimes.omabit --yes` advanced it to stable master
`d069544a7662cd332c9f59ae2efe7d53baa1a739`. Its original explicit desktop selection
remained active until the replacement was prepared and selected.

Imported and activated a private UI-only development bundle containing the current
polish and the **byte-identical published stable0.1.0 tools**. No Go compiler was
invoked and no new backend binary was built. Provenance records distinguish the
modified UI snapshot from the released tools' source commit:

- UI base: `5c9cb1e0582928196133871b047bb125688a34b9`, with uncommitted UI changes.
- Development label: `sourceport-ui-polish-stable010`.
- UI/binary input digest: `501424407532f69416261731c06adcd0e099bed932443bb0365d822eeff0fdb7`.
- Loaded release: `1b76401cb33c4db1dd83c93e1d1711a2919f91ec81eca2475227026982f50a9f`.
- Displayed application version: **0.1.0 / DEVELOPMENT**. The development label
  describes the unpublished UI changes; all three executable hashes match stable.

The actual packaged UI/helper bootstrap tests passed in isolated HOME/XDG before
live activation. Live bootstrap reports ready, no error, default profile, one runner
and two ships. The open monitor resumed its Maintenance page. The default profile
is explicitly pinned to this local bundle. To deliberately return to the published
UI, select `default --release follow` with `scripts/omabit-dev.py`; ordinary native
checkout updates alone do not replace an explicit development selection.

## Preservation

Before/after snapshots verified unchanged ship identities, configuration/revisions,
desired/observed state, ports/mounts, pier directory objects/ownership, enrollment,
SSH/access files, subordinate-ID grants, shell configuration and all operation,
maintenance-policy and deletion-history rows.

- `bare-comet` remained healthy/running: container
  `c08cbe24ed355e7a0980ef3957a0381072eb5b9a5da2d6f142675b8a015aade3`, PID1094395;
  its original worker PID/start time also remained unchanged.
- `also-a-comet` remained stopped.
- Rootless engine PID1078779/invocation `8b8a2d8a5b5f42a08b6cc490dab49b13` persisted.
- Stock shell PID1148390 persisted.
- Management alone changed from PID3657443 to254488, invocation
  `f5e495def6b849c68215e5fcfe61989e`.

Private evidence, a pre-upgrade SQLite backup, installer receipts and the exact
local bundle are retained under `build/sourceport-v010-update/`. The backup is
evidence, not a procedure for downgrading a live schema21 deployment.

## Follow-up: inline running loom

At the user's request, removed the standalone Runtime status card and placed the
loom beside the ship-context status, e.g. `healthy (2GB loom)`. The suffix is shown
only for a fresh running observation; stopped ships show only `stopped`. Current
runtime/recovery notices remain conditional text, and full facts stay in Technical
details. Actual Qt/Quickshell tests verified the running and stopped presentations.

Activated UI-only bundle `sourceport-inline-loom-stable010`, release
`81d6f9887bf7d1d85296c36543f07abd7798afee6df3f77003474e30b730edd7`,
with input digest `00503dea607c5fd8883e530be9de918d9a6bda0cd7156d57f7d54e9a3031b5c7`.
All three tools still match the published stable binaries byte-for-byte. Live
activation and before/after preservation checks passed, including unchanged
management, engine and ship processes. Evidence is in `build/sourceport-inline-loom/`.

## Follow-up: log filter toggle

Replaced the filter button with the existing themed switch labeled **Priority logs
only**: off shows all available output; on filters priority matches with context.
Actual Qt toggle/filter checks passed. Activated `sourceport-log-toggle-stable010`,
release `20bdc8aefd7e35f5a52fd8c1d9b751588a60dcd3e973fe2ce9d1a31f7ee31985`.
Both comets were running before this update and remained healthy with their same
processes afterward. Stable tools, manager and engine were preserved. Evidence is
in `build/sourceport-log-toggle/`.

## Follow-up: provisioning explanations and default pill

Recommended Apps is now selected by default in new boot forms and when resetting
the owned-identity form to register another ship. Provisioning choices are **Boot
Ephemeral Identity**, **Boot Permanent Identity**, and **Import existing pier**,
each with its own explanatory section. Actual onboarding and mapped form checks
passed, including the existing CLI's recommended-pill arguments, explicit bare
selection, and missing-recommendation handling.

Activated `sourceport-provisioning-stable010`, release
`dda8dfb920a1aad6ed853aa329dc118965c67c99b217bfbb8d6c0b1642c6ab14`.
The stable tools remain byte-identical; both healthy running ships and all
management/engine processes were preserved. Evidence is in
`build/sourceport-provisioning-copy/`.
