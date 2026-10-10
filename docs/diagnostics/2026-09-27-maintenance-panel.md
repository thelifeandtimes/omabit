# Maintenance panel validation and installation, September 27 UTC

Checkpoint 6L panel **0.3.0** passed automated validation and was installed into
the owned `~/.config/omarchy/plugins/thelifeandtimes.omabit` snapshot. **Visual
acceptance remains pending.**

## Checks passed

- `node scripts/test-panel-model.js`: capability/role/state/freshness admission,
  immutable maintenance revision, full uint64 epoch names, exact prefix evidence,
  protected suffix, altered/stale/replaced observations, unsupported layout,
  observer restrictions, and unknown/zero/stale resource measurements.
- `python3 -B scripts/test-panel.py`: real closed QML panel/service loaded in an
  isolated Wayland/CLI fixture. The actual maintenance-confirmation and pruning
  buttons were exercised. Runner replacement and cutoff changes cleared previews;
  explicit consent was required. A malformed maintenance acknowledgement retained
  the request ID. The CLI audit recorded exactly one stop, one pack, one meld and
  one confirmed prune, with unique IDs and no automatic resubmission. Existing
  application login still handed the fixture secret through stdin only.
- `python3 -B scripts/test-panel-install.py`: owned snapshot install/replacement,
  refusal of local modifications/unrelated files, asynchronous discovery and
  bounded enable confirmation.
- `go test ./...`, `go vet ./...`, and `go build -o ./build/ ./cmd/...`.
- `git diff --check`.

## Installation and read-only live checks

The previous owned plugin was archived to
`/tmp/opencode/omabit-panel-before-6l-20260927.tar`. The installer completed
`--replace --enable`, the installed manifest reports 0.3.0, the shell registry
reports enabled, and the shell log records the Omabit local-plugin reload without
an associated QML error in the inspected log window.

The newly bundled collector fetched an identity-pinned `workstation` snapshot:

| Instance | State / health | Configuration / applied |
| --- | --- | --- |
| alpha | running / healthy | 2 / 2 |
| panel-check | running / healthy | 1 / 1 |
| ssh-check | uninitialized / unknown | 1 / 0 |

No active operation or recovery requirement was present. The latest operation
remained `checkpoint-6k-resume-alpha-2`, operation
`op-5b42ceb7d9b7ed4fd5c8dfddfd52bc71`; panel installation and fixture checks added
no live runner operations. The 6K journal fields were rechecked read-only during
preparation, as recorded in the managed-prefix diagnostic.

The [visual walkthrough](../checkpoints/22-maintenance-panel.md) is the next user
gate. Programmatic loading is not visual acceptance.

## Refresh correction — September 28 UTC

The user reported missing resource readings. The installed 0.3.0 files were
correct and the bundled collector's snapshot and inventory both contained fresh
resource samples. A screen capture nevertheless showed the previous panel:
neither resource lines nor maintenance buttons were rendered. The earlier reload
messages and enabled registry state had not established that the new QML was live.

`omarchy restart shell` followed by reopening the Omabit panel resolved the stale
component display. A second screen capture confirmed container RAM, task counts,
source/timestamps/container IDs, maintenance/pruning reports, and disabled
maintenance controls for the running piers. At 05:08:17 UTC, alpha displayed
67.86 MiB / 4096 MiB with six tasks and panel-check displayed 366.11 MiB / 4096 MiB
with ten tasks, both fresh. ssh-check correctly displayed “not sampled.”

This confirms the refreshed interface is now visible; the user's broader visual
acceptance remains pending. The installation walkthrough now documents this
observed rescan limitation and the shell-restart remedy.
