# Update the existing review040 DEV environment

**Completed by the owner on2026-11-24.** The live DEV widget reports0.6.0-rc.3
and the updater's [complete receipt is recorded](diagnostics/2026-11-24-review040-update-completed.json).
The commands below document that update; do not run `--apply` again. Manual
functional review is now the next step.

“DEV environment” means the complete separate setup: widget, collector profile,
runner, engine and data. “Namespace” describes its isolation boundaries and names;
“configuration” means its settings. The current environment retains its historical
`review040` identifier across version updates.

This owner-run adapter targets only `/srv/dev/urbit-040`, the `review-040` profile,
`omabit-dev-040-runner.service`, and plugin ID `thelifeandtimes.omabit.review040`.
The observed starting build is published0.5.0/schema65 with runner identity
`runner-a9a208a18445179f05df5644ed6d245c`. It does not target the older DEV
environment rooted at `/srv/dev/urbit` or the normal production plugin.

## Commands for this machine

The complete candidate is already staged in
`/home/thelifeandtimes/repos/omabit/build/review040-update-060rc3-owner`.
Close any pending DEV operation, then run as your normal user, **without sudo**:

```bash
cd /home/thelifeandtimes/repos/omabit
python3 -B scripts/qualification/runtime-profiles/update_review040.py --apply
```

The initial fixed256MiB backup-size guard was corrected after the owner's
pre-apply refusal. Both existing boot pills remain included. The refreshed plan
passes measured backup/scratch admission under the unchanged48GiB cap; no live
update started during that refusal. You can check readiness without changing the
environment using the same command with `--check` instead of `--apply`.

Require a final `"status": "complete"`. Then verify the live DEV widget:

```bash
omarchy shell thelifeandtimes.omabit.review040.review status
```

Expected: version `0.6.0-rc.3`, profile `review-040`, `loaded:true`, and
`local_setup:false`. Its helper paths should name
`thelifeandtimes.omabit.review040-v060rc3`. The bar button stays **DEV**.

The script stages no fresh ship and does not perform a ship import or runtime
transition. It stops only the DEV manager, retains an offline management backup
and original binaries/preferences, installs the matching rc.3 helpers and exact
reviewed runtime archive in that DEV engine, then starts the manager with its
existing no-resume service arguments. Existing container processes, fleet
configuration, intent and operation history are compared before/after. The old
widget is retained outside the watched plugin directory; the candidate uses a new
versioned directory with the same plugin ID, profile and desktop paths. Only that
widget is reloaded, and original shell configuration bytes are restored.

Backups are retained at `/srv/dev/urbit-040/state/upgrade-060rc3-owner` and in the
staging directory. The runner advances to schema66; selecting old binaries is not
a supported rollback. If the command fails, stop and share the reported phase
from `build/review040-update-060rc3-owner/failed.json`. Do not rerun it blindly or
reset state. A second apply is deliberately refused after the attempt begins.

## Preparation and evidence

Preparation has already run and does not need repeating. For a separately
reviewed fresh attempt, the preparer takes the published bundle/archive and a new
working directory, checks the exact existing DEV identity, service and binary
hashes, and enforces the approved aggregate storage budget. `--prepare` never
stops services or changes installed files; only `--apply` does the live update.
The estimator counts actual management, SQLite analysis, desktop and tool copies,
plus explicit archive/image and growth reserves. Apply remeasures before any
mutation. `--refresh-plan` is a preparation-only recovery action: it keeps the
old plan and refuses a started attempt, existing backup, changed identity or
changed plugin inputs. It is not a way to retry a partially executed update.

The [preparation record](diagnostics/2026-11-24-review040-update-prepared.json)
binds the final updater and plan hashes. A separate networkless full Omarchy shell
passed loading the old wrapper, targeted unloading, loading the new versioned
wrapper, unchanged profile/local-setup boundaries and exact shell-config
restoration without restarting the shell. That test used empty private DEV-shaped
paths, not this environment's live state. Fault tests cover altered candidate
refusal, repeat-attempt refusal, and a late UI failure preserving the migrated
database/new tools/original offline WAL backup instead of downgrading them.
The later [storage correction](diagnostics/2026-11-24-review040-storage-correction.json)
supersedes the original updater/plan hashes and fixed backup thresholds; four
regression tests pass, including large retained boot pills and refusal before
mutation when the aggregate budget is exceeded.

The [native0.5.0→rc.3 upgrade](diagnostics/2026-11-24-runtime-profiles-upgrade.json)
and [runtime tests](diagnostics/2026-11-24-runtime-profiles-native.json) qualify the
exact published product bytes. **The custom live DEV update has not been run by
the assistant.** Its final receipt will exist only after the owner runs the
command successfully. This is a prepared owner-operated adapter, not a claim that
this custom service layout has already completed the upgrade.
