# Checkpoint 6W: managed comet epoch inspection and roll

The [6V disposable qualification](32-comet-epochs.md) passed three comet rolls and
identity-preserving restarts. This increment integrates explicit managed comet
epoch inspection, roll and preview-only retention. **Managed live validation
passed:** three rolls/replays/identity-preserving resumes retained four epochs and
restored healthy comet-check at 1/1. Confirmed pruning was refused and both disabled
policies were unchanged. See the [accepted evidence](../diagnostics/2026-09-29-managed-comet-epochs.md).

## Contract

- `comet-epochs` extends existing `inspect-epochs`, `roll` and collector
  `epoch-preview` workflows to initialized comets. Operator actions require the
  selected configuration revision, an idle clean stopped pier and complete private
  identity/provisioning evidence.
- Roll runs `/bin/urbit roll --loom 31 /urbit/comet` with `network=none`, using the
  managed-comet ownership namespace. It rechecks physical RAM/disk admission at
  dispatch and journals uncertainty before starting the utility.
- Completion requires exactly one newer snapshot-bearing epoch and preservation
  of all prior epoch names. The runner records before/after inventories, created
  epoch and logical size measurements. It leaves the ship stopped; Start is explicit.
- A completed roll sets `roll_requires_run`. Another new roll requires a healthy
  identity-pinned runtime cycle and explicit stop. Inspection remains available
  while this guard is set. Exact historical replay returns its original operation
  and never recreates an epoch or resurrects an invalidated inventory.
- Failed/interrupted utilities remain quarantined through observation and Stop.
  Unresolved comet-roll containers also block other expensive host work. Changed
  identity during resume is quarantined by the existing comet identity guard.
- Epoch inspection uses the same rooted/no-follow, bounded pinned-layout scanner.
  Inventory freshness, uint64 epoch strings and the newest-two protected suffix
  apply. Inventory metadata and logical/allocated byte counts are not verified
  backups, loom utilization or guaranteed reclaimable space.
- Comet previews always report **`deletion_supported:false`**, even though a mixed
  runner advertises pruning for qualified fake piers. The panel binds the instance
  mode into review state and independently refuses comet deletion. Runner admission
  and runtime dispatch both reject comet pruning; confirmed requests cannot bypass
  this boundary. Comet pruning/recovery retain a separate qualification gate.
- The first comet inspection or roll atomically advances private SQLite schema to
  **6**. Older-feature policy edits preserve that floor. Use **6W+ `serve-development`**
  afterward. The wire protocol remains version 1.
- Routine policies continue to support pack/meld only. Epoch rolls and retention
  are explicit operator actions.

## Local validation

Runtime fixtures exercise three roll/identity-resume cycles, post-resume inspection,
correct command/path/network, durable intent, resource/inventory limits, failed
growth/snapshot checks, persistence failures, interrupted utilities, identity
mismatch and quarantine after cleanup. Store tests cover schema/replay/cache
invalidation, interruption, runtime-cycle guards and pruning refusal. Collector and
panel fixtures verify that fake pruning capabilities cannot enable comet deletion;
actual QML roll confirmation and preview controls are exercised.

Passed locally:

```text
go test ./...
go vet ./...
go test -race ./internal/runtimequalify ./internal/managed ./cmd/urbit-collector
node scripts/test-panel-model.js
python3 -B scripts/test-panel.py --windows
python3 -B scripts/test-panel-install.py
go build -o ./build/ ./cmd/...
GOOS=linux GOARCH=arm64 go build -o /tmp/opencode/urbit-runner-6w-arm64 ./cmd/urbit-runner
git diff --check
```

The validator subsequently passed live. ARM64 was cross-built only; runtime
qualification remains Linux/AMD64.

## Accepted live gate (reference; do not rerun for evidence)

Press **Ctrl+C** in the management terminal, install the rebuilt binary and restore
management. Runtime containers continue independently.

```bash
sudo /usr/bin/install -D -o root -g root -m 0755 \
  /home/thelifeandtimes/repos/omabit/build/urbit-runner \
  /usr/local/lib/urbit-runner/checkpoints/6w/urbit-runner

sudo /usr/bin/runuser -u urbit-runner -- /usr/bin/env -i \
  HOME=/var/lib/urbit-runner PATH=/usr/bin:/bin \
  /usr/local/lib/urbit-runner/checkpoints/6w/urbit-runner \
  serve-development --acknowledge-development
```

After the user confirms management is running, the agent runs:

```bash
/usr/bin/python3 -B /home/thelifeandtimes/repos/omabit/scripts/validate-managed-comet-epochs.py
```

This fixed-target gate uses existing development `comet-check`, port 18087 and
identity `~nolpel-modsyd-livsep-novmeg--sallen-tassul-padlud-litzod`. It performs three
explicit stop/inspect/roll/replay/resume cycles, rejects consecutive rolls without
a runtime cycle, and checks the exact identity with replacement containers. A final
stopped inspection verifies all four epochs after the last resume; a preview keeps
three epochs with deletion disabled. A confirmed pruning request must be refused.
The final explicit resume restores healthy 1/1 state, and historical replay must
not restore stopped inventory. Existing disabled policies are checked unchanged.

The validator requires an initial genesis-only inventory and refuses prior 6W work,
active policies or unresolved state. Any failed
step halts without automatic resume/retry. Review its fixed request IDs and journal
before further mutation. Panel **0.10.0 is installed**, and a shell restart plus
live monitor status IPC verified the loaded version.
