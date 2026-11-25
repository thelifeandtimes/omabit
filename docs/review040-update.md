# Update the existing review040 DEV environment

**rc.4 is prepared for the owner to apply; live DEV remains on rc.3.** The exact
package, native runtime paths, both upgrade starting versions and separate DEV
widget activation pass. The final readiness check reports ready with no live changes.

“DEV environment” is the clearest name for the complete setup: widget, collector
profile, runner, engine and data. “Namespace” describes its isolation boundaries;
“configuration” means its settings. It retains the historical review040 identifier.

This owner-run adapter targets only `/srv/dev/urbit-040`, profile `review-040`,
`omabit-dev-040-runner.service` and plugin `thelifeandtimes.omabit.review040`.
It verifies the exact installed rc.3 build and runner identity before proceeding.

## Commands for this machine

Close pending DEV operations. Run as your normal user, **without sudo**:

```bash
cd /home/thelifeandtimes/repos/omabit
python3 -B scripts/qualification/runtime-profiles/update_review040_rc4.py --check
python3 -B scripts/qualification/runtime-profiles/update_review040_rc4.py --apply
```

Require a final `"status": "complete"`, then inspect:

```bash
omarchy shell thelifeandtimes.omabit.review040.review status
```

Expected: version `0.6.0-rc.4`, profile `review-040`, `loaded:true`,
`local_setup:false`, and helper paths under `thelifeandtimes.omabit.review040-v060rc4`.
The bar button stays **DEV**. Use the [focused manual review](runtime-profiles-manual-review.md)
to check recovery, pending buttons, the permission map and runtime selection.

The completed rc.3 update used `update_review040.py` and
`build/review040-update-060rc3-owner`. Do not rerun or reset that earlier attempt.
Its [completion receipt](diagnostics/2026-11-24-review040-update-completed.json)
and backups remain intact.

## Preservation and failure handling

The rc.4 preparation lives in `build/review040-update-060rc4-owner`. Apply creates
`/srv/dev/urbit-040/state/upgrade-060rc4-owner`, preserving management state,
runtime profiles/archives, old binaries, desktop data and shell configuration.
It stops only the DEV manager, installs matching rc.4 tools and verifies the
already selected reviewed archive, then starts the manager with its existing
no-resume service arguments. It compares original configuration, intent, history,
engine identity and container processes before/after. It neither creates a ship
nor changes a ship's selected runtime.

The old widget is retained in the new work directory's `plugin-before`; the new
versioned widget keeps the same plugin ID, profile and desktop paths. Only that
widget is unloaded/reloaded. Original shell configuration bytes are restored.
Production services and the separate older DEV root `/srv/dev/urbit` are untouched.

Runner state advances from schema66 to67. An older writer must refuse it;
restoring older binaries alone is not a rollback. If apply fails, retain the
reported phase and `build/review040-update-060rc4-owner/failed.json`. Do not rerun,
reset state, edit the schema marker or discard the checkpoint. Repeat apply is
refused once an attempt starts. The updater does not automatically downgrade on
failure. Its fault tests cover changed inputs, repeat attempts, capacity refusal
and a late UI failure retaining migrated state and backups.

## Preparation and qualification

`--prepare` stages the exact candidate after read-only identity checks. `--check`
remeasures all backup/image scratch under the unchanged48GiB aggregate cap and
verifies the unstarted plan; neither modifies installed files or services.
`--refresh-plan` retains and renews only an unstarted plan after a reviewed adapter
correction. It cannot retry a partially executed update.

- [Frozen package and provenance](diagnostics/2026-11-25-runtime-feedback-build.json).
- [Source, all race shards, QML and reproducibility](diagnostics/2026-11-25-runtime-feedback-ci.json).
- [Bundled runtime:14 checks and16 container launches](diagnostics/2026-11-25-runtime-feedback-bundled-native.json).
- [Local-build fresh/switch/import:5 checks and8 launches](diagnostics/2026-11-25-runtime-feedback-local-native.json).
- [Direct0.5.0→rc.4 upgrade](diagnostics/2026-11-25-runtime-feedback-stable-upgrade.json) and
  [rc.3→rc.4 upgrade](diagnostics/2026-11-25-runtime-feedback-rc3-upgrade.json): real native
  authorization denial/retry, schema migration, older-writer refusal, retained
  running containers and original ship/application/access history.
- [Full isolated Omarchy shell](diagnostics/2026-11-25-runtime-feedback-full-shell.json).
- [Exact DEV wrapper and prepared owner update](diagnostics/2026-11-25-review040-rc4-prepared.json).


The assistant has not applied the live update. A local preparation and isolated
rehearsal do not establish that this custom DEV layout has already completed it;
the owner's complete receipt establishes that final result.
