# Issue22 continuation and review handoff

The active candidate is **0.6.0-rc.3**, frozen at
`747dd810c10b74b4fbcdc10f617e6b15dd545a45`, on
`codex/issue-22-runtime-profiles`, in [draft PR24](https://github.com/thelifeandtimes/omabit/pull/24).
It is not published or ready for a production update. Native bootstrap still pins
stable0.5.0. Keep master protected and leave issue22 open.

The user narrowed this continuation on2026-11-24 to **core functionality**.
Do not resume decoder crash reproduction, malformed-input negative controls or
exploit investigation. Validate the approved runtime's immutable identity and
ordinary Omabit operations. This supersedes the earlier negative-control test
requirement; it does not claim independent proof of the decoder fix.

## Read first

Read [development policy](development-policy.md), [experiment isolation](experiment-isolation.md),
[protected ships](protected-ships.md), the top of [upgrade policy](upgrade-policy.md),
[review instructions](runtime-profiles-review.md) and development memories M782 onward.
Never inspect protected ships, tmux ships or `~/temp-ur`. Preserve installed
services, plugins and human-review DEV state. No production installation is
part of this continuation. The owner-approved limit is48GiB total with one
runtime VM at a time and4GiB reserved scratch; do not request that approval again.

## Recovered evidence

[The resumption record](diagnostics/2026-11-24-runtime-profiles-resumption.json)
contains exact rc.3 asset metadata, package verification, CI and the incomplete
runtime05 result. Source, all four managed-race shards and QML passed for rc.3.
The protected release check refuses the prerelease topic as intended.

Runtime05 passed exact image installation, HTTP/CLI preflight, two fresh fakes
with different pins, manager restart, missing-archive refusal and forbidden
runtime-change refusal. It then tried Start after an intentionally failed Start
without reconciling the resulting recovery state. That failed run is not a
whole-scenario pass. The corrected harness uses Stop after restoring the archive,
and Stop before restart after the controlled process termination. It retains
stock and reviewed pins through those operations.

Earlier rc.1 upgrade evidence is historical and must not qualify rc.3. The rc.1
image-store lookup and rc.2 dotted-ID preflight failures are fixed in rc.3.
Keep their diagnostic records rather than deleting failed attempts.

## Repository-owned harness

[run.py](../scripts/qualification/runtime-profiles/run.py) contains both the
runtime case and a previous-stable upgrade case. The latter uses the existing
patch-upgrade guest with checksummed, chunked console reports, avoiding the earlier
truncated JSON failure. Both cases require a clean checkout and frozen product
inputs. Only this qualification directory plus normal distribution documentation
may differ from the candidate source. This local harness allowance does not change
or satisfy the stable release policy.

[guest.py](../scripts/qualification/runtime-profiles/guest.py) checks fresh install,
preflight, exact binaries, two runtime pins, stop/start, replacement, maintenance,
controlled process recovery and import of a stopped copy of the newly created
fake. It never uses a user's pier. [test_harness.py](../scripts/qualification/runtime-profiles/test_harness.py)
checks incomplete/corrupt report refusal and product-source freeze boundaries.

Run inert checks with:

```sh
python3 -B scripts/qualification/runtime-profiles/test_harness.py
python3 -B scripts/qualification/runtime-profiles/run.py --help
```

These are existing sourcebox inputs, not artifacts committed to Git. The launcher
checks the base, helpers and package hashes before running. Do not recreate a
missing immutable base or replace a missing input with a live installation.

```sh
qualification_labs="$HOME/Documents/ChatGPT/omabit-private-docker-labs"
qualification_inputs="$HOME/Documents/ChatGPT/omabit-qualification-evidence/reboot-preservation-20261120-050/tmp"
systemd-run --user --unit=omabit-issue22-runtime07-driver \
  --property=RuntimeMaxSec=3000 --setenv="PATH=$PATH" \
  /usr/bin/python3 -B "$PWD/scripts/qualification/runtime-profiles/run.py" \
  --case runtime \
  --portable "$qualification_labs/manual-review-20261027-01/portable" \
  --previous /tmp/omabit-050-public-downloads/assets \
  --candidate /tmp/omabit-runtime-profiles-review-assets/0.6.0-rc.3 \
  --rootless "$qualification_inputs/omabit-issues-vm-tools/rootless" \
  --init "$qualification_inputs/omabit-issues-recording-vm-03/init" \
  --output "$qualification_labs/patch-upgrade-060-runtime07" \
  --storage-budget-gib 48 --run
```

The transient driver service avoids tying the long VM run to a command-session
lifetime. Inspect only its named unit with `systemctl --user status` or
`journalctl --user -u omabit-issue22-runtime07-driver`; the existing VM scope still
enforces memory, CPU, network and storage limits.

Choose a genuinely unused output name; the harness refuses an existing directory
and never resumes an old VM. For the upgrade rehearsal, use `--case upgrade
--app-workflow` and a fresh `patch-upgrade-060-upgradeNN` output with the same inputs.
It checks the actual published0.5.0 tools, native QML service update, original
identity/configuration/history/access/process preservation, schema65→66, manager
reopen and old-writer refusal. The final `result.json` must report `passed:true`;
partial check flags or process exit0 alone do not establish success.

After saving compact results and recording the scope's end, inventory only that
VM's generated disks and boot copies:

```sh
python3 -B scripts/qualification/runtime-profiles/retire.py \
  --output "$qualification_labs/patch-upgrade-060-runtime07"
```

Review the exact list, then repeat with `--run` to write a hash/inode manifest and
retire those files. Sources, reports, logs and immutable bases remain. Never delete
whole evidence directories. Admission is remeasured at every launch and throughout
the VM. A storage refusal requires classification, not raising the cap.

## Remaining review gates

1. Complete the current runtime scenario and record all reached and failed checks.
2. Repeat the0.5.0 upgrade against the same frozen rc.3 bytes.
3. Verify matching desktop/helper activation, reviewed-profile import and the
   remaining handoff/interface boundaries, distinguishing native and source tests.
4. Supply immutable delivery pins and tested owner-operated instructions before
   recommending any update. Stable release promotion and production installation
   remain separate decisions; no direct master push or protection bypass.

The latest run status and any next action are recorded below as evidence is obtained.

Runtime06 was interrupted by host signal15 before a complete guest report. The
[interruption record](diagnostics/2026-11-24-runtime-profiles-interruption.json)
records the inactive scope and original log hashes. This is neither a product
failure diagnosis nor a pass. Its generated disks were retired with a separate
interruption receipt; the next run uses the bounded transient driver above.
