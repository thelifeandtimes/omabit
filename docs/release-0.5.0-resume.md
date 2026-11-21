# 0.5.0 pause checkpoint and resumption — 2026-11-20

Current checkpoint: M774 selects source59d83aab49384d2f26531320bb5ab68e1257aeb5,
assets `build/release050-candidate02/0.5.0` and helpers `build/release050-helpers02`.
Matching desktop/interface and stable59→65 upgrade gates pass. The next native
cases are pressure06, Grubbery COW06, Stop COW05 and closed recording COW08, with
original backings restored only after verified full-budget admission. Preserved
DEV and reviewed publication remain pending. No VM remains from upgrade03; its
compact independently verified evidence is retained and classified images retired.

The user explicitly resumed after the machine update on2026-11-20. M773
supersedes the paused status below. Fresh `patch-upgrade-050-final02` passed with
candidate01 bdf4397, including original app history, stable59→65 migration,
runtime continuity, manager restart and old-reader refusal. Independent compact
results/SQL are retained; the two ended upgrade capsules' classified images were
retired. A corrected packaged preflight reference and a CI test deadline now
require a new clean source freeze, matching artifacts and qualification. Do not
treat candidate01's passes as qualification of that next candidate.

The remaining text records the original pause checkpoint, not a renewed pause.

The user explicitly paused for a machine restart/update. Do not resume until the
user asks. No production deployment or existing ship operation is authorized.

Product inputs remain frozen at `bdf43978e1d0ea858c3349b787e2ff8eaa1252ad`.
Final assets are in `build/release050-candidate01/0.5.0`; helpers/read-only inspectors
are in `build/release050-helpers01`. All asset pins are in the release manifest.
Draft PR23 remains open, linked, and not merged or published:
https://github.com/thelifeandtimes/omabit/pull/23
Its last published source head is bdf4397; later evidence commits are local.

Completed final-artifact gates:

- M769: frozen artifacts, private desktop activation/recovery, Recordings UI,
  Agent Help lifetime and legacy compatibility, selected interface races,
  all180 offline help entries, and actual CLI/collector/official SDK clients.
- M770: all nine recording pressure/capacity gates, with independent original
  content, job and SQL verification. Completed images are classified and retired.
- M771: all eight native Stop-transition gates, independently verified exact
  receipt, same-pier restart, peer/anchor and23 inherited-table preservation.
  Completed images are classified and retired.
- CI run44792713306 last observed: core, QML and managed-race shards1/2 passed;
  shards0/3 were running. Recheck remotely after resumption. Release-policy needs
  final upgrade evidence and release pins; never waive it.

The stable0.4/schema59→0.5/schema65 native upgrade rehearsal was interrupted by
this explicit user pause. Its guest powered off gracefully through its own QMP
socket, the exact scope ended inactive/empty, and host exit was0. There was no
completed guest result and **no upgrade pass**. Retain the partial evidence at:
`/home/thelifeandtimes/Documents/ChatGPT/omabit-private-docker-labs/patch-upgrade-050-final01`.
`USER_PAUSED_20261120.json` records the reason. Never resume this ended overlay;
classify it and launch a fresh `patch-upgrade-050-final02` when ready.

`/tmp` is volatile tmpfs. The generated Omabit temporary files are preserved
with full SHA256 manifests under:
`/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/reboot-preservation-20261120-050`.
`manifest.json` is `moved-and-every-file-sha256-verified`:50,638 regular files verified; full post-move charge46,809,341,952bytes is below48GiB. Check this status and `after.json` before using it. Original `/tmp/X`
locations map to `tmp/X` below that directory. The private manifest includes
source, tools, compact evidence, archived recording originals, app62 and current65
fixtures and handoff scripts. Regular files are moved only after verified transfer;
ended ephemeral sockets/pipes are listed separately and are not reusable inputs.
Do not overwrite any changed existing paths during restoration or substitute
symlinks for a harness's required physical inputs. Re-admit restored inputs under
the complete48GiB measurement; verify every needed hash and backing chain first.

Still required:

1. Fresh stable0.4→0.5 native upgrade with app history, original identities/access,
   running/stopped intent, manager restart, old-reader refusal and state preservation.
2. Final Grubbery recovery/two-version/state/restart workflow using
   `scripts/qualification/grubbery-runtime-cow`, final plain managed.test and
   retained init `omabit-issues-runtime-cow65-init-02`. Original app62 is read-only.
3. Final closed original/indexed/redacted export/output-coverage case using
   `scripts/qualification/recording-usage-cow`, original recording07 restored from
   its verified gzip, final plain helper, and retained
   `omabit-issues-recording-usage-init-02`. The final read-only inspector is built:
   `recording-reader-init`, SHA256
   `2737fa4851f96281b6f03c1344cffb00dcc9aa8777913c8f121267eab8069c63`.
4. After actual upgrade qualification, preserve/update only the authorized DEV
   review namespace and complete a fresh minimal app workflow. Read-only inventory
   confirms runner-a9a208a18445179f05df5644ed6d245c, schema59, source9ec8be8 dirty
   development build, runner SHA30aeebd35be7d9b0d6db2c8c553306bd36db098b9874ae230b3a32952a14f963.
   No installed update has occurred. Preserve both DEV roots and manual intent.
5. Exact immutable draft assets/qualification, download verification, distribution
   pins, review and required CI, protected merge commit, stable publication and
   verification. No bypass, squash, issue-wide closure or production deployment.

Keep one runtime experiment; no Go builds/tests or temporary-output fixture suites
while a VM or inspector is active. The complete budget counts all registered
worktree builds/caches, both DEV roots, evidence roots and physical `/tmp/omabit-*`.
No estimated/unreadable fallback. No subagents. Use `/usr/bin/gh`; every push or
PR creation must be followed by the PR-link tool on the next call. Use the watch
tool only when waiting on checks and end that turn; no watch was active at pause.

Read the finite release checklist, next-gates record, development memory,
experiment isolation, protected-ships and upgrade policy before resuming. The
Omarchy skill and its plugins guide were read for eventual DEV updates; no shell
restart or production plugin changes are permitted. Keep the separate plugin ID,
profile, IPC, preferences and services, and stage complete versioned plugin code
outside the watched tree before atomic installation.
