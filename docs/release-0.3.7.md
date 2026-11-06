# Omabit 0.3.7 — one-click setup and verified identity recovery

**Release preparation; not published.** Source is frozen at `076f972`; local
artifact, picker and exact-bundle native desktop checks pass. Source/race CI is
pending. See [preparation evidence](releases/0.3.7/preparation.md) and
[draft PR #14](https://github.com/thelifeandtimes/omabit/pull/14).
The required exact-artifact 0.3.6 → 0.3.7 service-upgrade
rehearsal is pending because the existing isolated sourcebox lab is unavailable.
Do not treat the user-confirmed fresh-install trial as upgrade qualification.

## Changes

- **One-click setup on Arch/Omarchy:** choose **Set up this computer**, then
  authorize the OS prompt. Missing native prerequisites are installed and verified
  as part of that operation. Package details are under **View details**.
- **Clearer first-run flow:** direct local setup, ship creation after a runner is
  connected, and progress through native packages, services and enrollment.
- **Bundled SSH key generation:** initial setup no longer requires `ssh-keygen`
  before native packages are installed. Existing OpenSSH-format Ed25519 desktop
  keys are reused, including after an interrupted attempt.
- **Working key-file selection:** the non-native Qt chooser stays on its owning
  layer-shell surface and supports selection, cancellation and reopening.
- **Verified identity correction:** local owner CLI `repair-boot-identity` can
  recover a mistyped keyed first-boot identity. It verifies public and private
  identity, gracefully stops the exact boot process, corrects registration and
  preserves the same pier and history. See [procedure](boot-identity-repair.md).

Runner schema47, desktop state1, bootstrapAPI1, Vere4.6 and the Recommended Apps
pill are unchanged. Source builds require Go1.26 or later; binary users need no Go.
Native prerequisite installation is limited to initial Arch/Omarchy setup;
existing management updates retain their identity-preserving upgrade path.

## Planned installation and upgrade

After publication, use the standard commands:

```sh
# New desktop
omarchy plugin install https://github.com/thelifeandtimes/omabit.git --enable

# Existing desktop
omarchy plugin update thelifeandtimes.omabit
```

For a fresh local host, select **Set up this computer** and authorize the OS prompt.
For an existing host, allow the verified bundle to load, then use **Update
Dependencies** when offered. Remote runner updates remain explicit. A desktop
update does not itself restart a ship or change its runtime.

The target support path is 0.3.6 → 0.3.7 on Linux/AMD64 owner-user installations.
Earlier stable versions follow the documented intermediate paths to0.3.6 first.
Identity repair supports the current schema47/pinned keyed-runtime contract and
does not migrate older schemas. Stopped or interrupted work needs its existing
recovery procedure; resetting or re-enrolling is not an upgrade.

## Qualification to finish

1. **Done:** freeze source and build matching runner, collector and desktop assets.
2. **Done:** verify native hot/cold activation, failed-candidate recovery and retained
   desktop preferences with exact published0.3.6 and frozen0.3.7 assets.
3. Rehearse published0.3.6 → frozen0.3.7 services in the isolated VM, preserving
   running containers/processes, stopped intent, identities, storage, access and
   operation history; include retry and manager restart.
4. Record exact artifacts, upgrade evidence and limits; add distribution pins.
5. Pass protected source/race/QML/release-policy checks, review and merge the PR
   with a merge commit, then publish verified stable artifacts.

The user confirmed the `.5` fresh-install/key-picker trial and the repaired
Recommended Apps moon. That is useful development evidence, not a substitute for
the frozen-release gates above. See [trial history](one-click-setup-trial.md) and
[repair evidence](diagnostics/2026-11-05-moon-identity-mismatch.md).
