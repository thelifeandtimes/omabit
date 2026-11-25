# Issue22 continuation and review handoff

## Stable 0.6.0 promotion — supersedes earlier delivery status

The owner completed RC4/UI1 review and authorized stable publication in M806.
[Release PR25](https://github.com/thelifeandtimes/omabit/pull/25) promotes frozen
source `fcf269ca8aa20aa9e43a70d7e282a91f12659cf9`. Exact stable upgrade, bundled
runtime, local runtime, desktop and source qualification is recorded in
[0.6.0 qualification](releases/0.6.0/qualification.json). This supersedes the older
pending-recovery/DEV-application status below; preserve that history and its
receipts. Production installation remains owner-operated. Final protected PR
checks, merge and verified publication are required for stable delivery.


## Post-rc.4 UI follow-up

The owner reports that rc.4 is looking good and requested one layout change:
Vere runtime selection now lives inside the ship's expandable Runtime Settings
editor. The settings model tests and isolated QML resource-pane check pass at
640px and420px; see M802. M803 supersedes the delivery-pending status: a distinct
local UI preview, `0.6.0-rc.4+dev.ui.1`, is prepared and checked with the exact
activation rehearsed in a private shell. M804 records the owner's failed apply:
Omarchy inserted an absent empty tray section when disabling the widget. The old
and staged plugins remain intact and the DEV widget is disabled. The corrected
checkpoint recovery is verified and awaits the [owner commands](review040-update.md);
do not rerun the original apply or discard its markers. Published rc.4 artifacts
remain immutable. The owner's
rc.4 completion receipt and read-only preflight confirm that DEV is now on rc.4;
the older rc.3 live status below is historical.

M805 adds reboot-safe lock creation. The owner's subsequent reboot cleared /tmp,
but the repository checkpoint/candidate and installed rc.4 widget remain intact.
Recovery never started. The corrected updater passes ten guard/lock tests and the
real post-reboot `--check-recovery`; owner recovery is still pending. Earlier /tmp
rehearsal directories were volatile; durable diagnostic JSON retains their results.

## Current review candidate — rc.4, prepared for owner application

The [rc.4 review prerelease](https://github.com/thelifeandtimes/omabit/releases/tag/v0.6.0-rc.4)
is published; all public assets are verified and stable/latest remains0.5.0.
Product source is27aac5001c43a7a1148df47dc9904bdeb1a4a536, version0.6.0-rc.4/schema67.
Source, all managed-race shards, QML and full release reproduction pass. The bundled
runtime scenario passes14 checks/16 launches; the independent local-build scenario
passes5 checks/8 launches, covering fresh boot, explicit stopped transition, import
and manager restart. Direct0.5.0→rc.4 and rc.3→rc.4 native upgrade rehearsals pass,
including preserved application history and older-writer refusal. Full isolated
Omarchy shell activation and the exact DEV wrapper pass. See [manual review](runtime-profiles-manual-review.md)
for evidence and scope, and [the prepared owner command](review040-update.md).

Live review040 remains rc.3; no assistant apply or review-zod recovery was performed.
The new unstarted work is build/review040-update-060rc4-owner. Preserve the completed
rc.3 work/backup. Do not apply automatically, reset a failed checkpoint or infer
that stable publication/production service updates are authorized. Master remains
protected; its gate intentionally refuses this prerelease topic.

M800 supersedes M798/M799's qualification-pending status. Earlier failed private
harness runs are retained with causes and retirement manifests; product bytes did
not change. Split runtime cases use --case runtime and --case local-runtime with
--local-runtime DIRECTORY to stay within the48GiB cap/4GiB per-VM scratch. All
qualification VMs ended and their generated disks were retired with evidence.
Latest upstream channels returned HTTP403; actual64-bit execution remains unverified.
Project factories keep their separate official-runtime contract.

## Historical rc.3 handoff

The active candidate is **0.6.0-rc.3**, frozen at
`747dd810c10b74b4fbcdc10f617e6b15dd545a45`, on
`codex/issue-22-runtime-profiles`, in [draft PR24](https://github.com/thelifeandtimes/omabit/pull/24).
Native bootstrap now pins the exact qualified rc.3 archive and helpers. The
immutable review prerelease is published and all public downloads are verified; follow the
[manual review guide](runtime-profiles-manual-review.md) for its final published
revision and entry point. This is a prerelease, not a stable promotion. Keep
master protected and leave issue22 open for owner review.

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
systemd-run --user --unit=omabit-issue22-runtime08-driver \
  --property=RuntimeMaxSec=3000 --setenv="PATH=$PATH" \
  /usr/bin/python3 -B "$PWD/scripts/qualification/runtime-profiles/run.py" \
  --case runtime \
  --portable "$qualification_labs/manual-review-20261027-01/portable" \
  --previous /tmp/omabit-050-public-downloads/assets \
  --candidate /tmp/omabit-runtime-profiles-review-assets/0.6.0-rc.3 \
  --rootless "$qualification_inputs/omabit-issues-vm-tools/rootless" \
  --init "$qualification_inputs/omabit-issues-recording-vm-03/init" \
  --output "$qualification_labs/patch-upgrade-060-runtime08" \
  --storage-budget-gib 48 --run
```

The transient driver service avoids tying the long VM run to a command-session
lifetime. Inspect only its named unit with `systemctl --user status` or
`journalctl --user -u omabit-issue22-runtime08-driver`; the existing VM scope still
enforces memory, CPU, network and storage limits.

Choose a genuinely unused output name; the harness refuses an existing directory
and never resumes an old VM. For the upgrade rehearsal, use `--case upgrade
--app-workflow` and a fresh `patch-upgrade-060-upgradeNN` output with the same inputs.
It checks the actual published 0.5.0 tools, native QML service update, original
identity/configuration/history/access/process preservation, schema65→66, manager
reopen and old-writer refusal. The final `result.json` must report `passed:true`;
partial check flags or process exit0 alone do not establish success.

After saving compact results and recording the scope's end, inventory only that
VM's generated disks and boot copies:

```sh
python3 -B scripts/qualification/runtime-profiles/retire.py \
  --output "$qualification_labs/patch-upgrade-060-runtime08"
```

Review the exact list, then repeat with `--run` to write a hash/inode manifest and
retire those files. Sources, reports, logs and immutable bases remain. Never delete
whole evidence directories. Admission is remeasured at every launch and throughout
the VM. A storage refusal requires classification, not raising the cap.

## Qualification and delivery

1. Runtime07 core scenario passed; retain its exact scope and evidence below.
2. The matching 0.5.0→rc.3 upgrade passed; retain its support window below.
3. Full isolated Omarchy shell activation passed; see the current result below.
   Broader native handoff and recording coverage remain separate from source tests.
4. All nine draft and public downloads pass. The non-latest prerelease and
   distribution checkout664f938 are ready for owner review with verified import,
   profile selection and exact prior-profile restoration. Stable promotion and
   production installation remain separate decisions.

The latest run status and any next action are recorded below as evidence is obtained.

Runtime06 was interrupted by host signal15 before a complete guest report. The
[interruption record](diagnostics/2026-11-24-runtime-profiles-interruption.json)
records the inactive scope and original log hashes. This is neither a product
failure diagnosis nor a pass. Its generated disks were retired with a separate
interruption receipt; subsequent runs used the bounded transient driver above.

## Current result — runtime07 passed

The [native core-functionality result](diagnostics/2026-11-24-runtime-profiles-native.json)
passes all 14 checks using unchanged rc.3 artifacts and harness d332c805. It verifies
two different image pins, manager restart, missing/corrupt-archive refusal,
replacement, Pack, controlled process recovery, selected-profile import and the
actual 16-container lifecycle/utility image audit. All three disposable instances
finished stopped. The 3,369,426,944 allocated bytes of generated VM/boot files were
retired with a manifest after preserving the complete report and serial log.

## Current result — matching upgrade passed

The [upgrade result](diagnostics/2026-11-24-runtime-profiles-upgrade.json) passes
with the published 0.5.0 binaries and unchanged rc.3 target. It preserves both
original ship configurations, all 5 original operation records, access, live
engine/container processes and 19 app-history tables containing 38 rows.
Schema65→66, authorization denial/retry, manager reopen, old-writer refusal,
stopped fake import and app operations before/after the update pass. Native
sealed-archive loading and offscreen QML update controls pass; activation
acknowledgements are harness-driven, so full Omarchy shell activation is separate.

Both VM scopes are inactive. Upgrade01's 2,416,541,696 bytes of generated files
were retired after retaining evidence and screenshots. The final aggregate
charge is 47,172,669,440 bytes; remeasure before any future run. No disposable VM remains running.

## Current result — full shell activation passed

The [full-shell record](diagnostics/2026-11-24-runtime-profiles-full-shell.json)
uses the unmodified installed Omarchy shell inside a fresh networkless Bubblewrap
namespace and headless Sway compositor. It loads the exact sealed 0.5.0 and rc.3
archives through the normal loader. Cold startup, hot activation in the same shell,
visible monitor/page/selection preservation, plugin rescan and cold restart pass.
The private profile has no enrollment and local setup is disabled. The test uses
a distinct plugin ID and namespace adapters; host Hyprland integration is outside
this check. Hyprland socket and QML teardown warnings are retained in its logs.

[full_shell.py](../scripts/qualification/runtime-profiles/full_shell.py) retains
the driver, with only the repository path made relative after the run. It uses the
same known local asset and compositor inputs, refuses an existing output directory,
shares the runtime lock, and admits 1GiB scratch under the approved48GiB cap.
For a justified rerun, use a fresh `/tmp/omabit-060-shell-NN` output under a bounded
user service with MemoryMax=2G, MemorySwapMax=0, CPUQuota=100%, TasksMax=256 and
RuntimeMaxSec=300. The completed `omabit-issue22-shell01.service` is inactive and
successful. No host desktop socket or production HOME was mounted.

The candidate assets must remain unchanged. If product inputs change, freeze a
new version and qualify those bytes. The older failed runs remain historical;
do not repeat completed rc.3 scenarios without a new change, failure or unresolved
concern. Owner acceptance, broader excluded scenarios and protected stable
promotion remain separate from this complete prerelease review delivery.

## Current result — published manual review delivery

[v0.6.0-rc.3](https://github.com/thelifeandtimes/omabit/releases/tag/v0.6.0-rc.3)
is public as a prerelease, with0.5.0 still latest. All nine public downloads match
the qualified original bytes. The [publication record](diagnostics/2026-11-24-runtime-profiles-publication.json)
binds the tag, asset IDs/digests and exact distribution commit
`664f938ad15f139dbf79e0baef3f4bda145c9d02`. The product tag remains747dd810.

The [manual entry check](diagnostics/2026-11-24-runtime-profiles-manual-entry.json)
uses that checkout and fresh private state to verify sealed import, new profile
with local setup disabled, matching tools and restoration of a prior non-default
explicit selection. The [manual guide](runtime-profiles-manual-review.md) starts
with desktop review and makes the tested local0.5.0 service upgrade a separate
owner-operated choice. Existing production and manual DEV installations remain
unchanged. Final aggregate charge47,398,850,560 bytes is within48GiB.

**Next action belongs to the owner:** review the candidate and report feedback.
Keep PR24 draft and issue22 open; do not perform a live-pier import, production
upgrade or stable promotion as an automatic continuation of this delivery.

For the user's subsequent request to update their existing review040 DEV
environment, see [the dedicated owner commands](review040-update.md) and M795.
That adapter is staged and privately verified; the live custom DEV update remains
unexecuted until the owner runs it. The normal-plugin review profile is separate.
