# Checkpoint 6U: managed comet compaction and routines

The [6T disposable gate](30-comet-maintenance.md) passed pack/meld and exact-identity
restarts. This increment integrates those utilities into registered, initialized
comets in `development-managed` mode. **Managed live validation passed** on
September 29: manual pack/replay/resume and one scheduled meld/resume retained the
same identity and returned comet-check healthy at 1/1. The test policy is disabled
at revision 2. See the [accepted evidence](../diagnostics/2026-09-29-managed-comet-compaction.md).

## Contract

- New capability **`comet-compaction`** extends existing `pack` / `meld` operations
  and `maintenance-schedules` policies to initialized comets. The collector uses
  its existing identity-pinned operation and routine commands.
- Manual utilities require an explicitly stopped, clean pier and completed private
  comet provisioning evidence. Runtime intent uses the managed-comet ownership
  namespace and `/bin/urbit pack|meld --loom 31 /urbit/comet`, with `network=none`.
- Physical RAM/disk admission is rechecked immediately before utilities. Measured
  logical bytes and outcomes are journaled; completed manual work leaves the pier
  stopped. Every subsequent Start checks the pinned public identity.
- Identity mismatch during resume durably quarantines the comet. Failed/uncertain
  utilities retain maintenance quarantine; Stop resolves resources without clearing
  either marker. Retained comet utilities also block new host-expensive work.
- UTC routines use the same per-instance exclusion and host gate, explicit downtime
  and resume consent, owner/revision checks and atomic slot consumption as fake
  routines. They never create a new identity, resume after a failed utility, or
  continue an interrupted job. An originally stopped pier stays stopped.
- The first comet utility acceptance or saved comet policy advances private SQLite
  schema to **5** atomically. Subsequent fake policy edits cannot lower it. Use
  **6U+ `serve-development`** once this state exists.
- Panel **0.9.0** enables comet pack/meld and routine controls only when the runner
  advertises the new capability. Comet roll, epoch inspection, pruning and recovery
  still require separate qualification.

## Checks

Runtime fixtures cover both utilities, correct pier/command/namespace/network,
durable intent, measured bytes, initialized resume, routine consent, admission
before stop, failed utility cleanup, and changed-identity quarantine. Store tests
cover replay, unsupported retention, provisioning provenance, once-only slots,
interruption and monotonic downgrade guards. Model/QML checks exercise capability
gating and existing confirmation/editor behavior.

Passed locally:

```text
go test ./...
go vet ./...
go test -race ./internal/runtimequalify ./internal/managed
node scripts/test-panel-model.js
python3 -B scripts/test-panel.py --windows
python3 -B scripts/test-panel-install.py
go build -o ./build/ ./cmd/...
GOOS=linux GOARCH=arm64 go build -o /tmp/opencode/urbit-runner-6u-arm64 ./cmd/urbit-runner
git diff --check
```

The live validator subsequently passed the gate below. The ARM64 result is a
cross-build only, not runtime qualification.

## Accepted live gate (reference; do not rerun for evidence)

Stop the current management process with **Ctrl+C**, install the rebuilt runner,
and restore management (runtime containers continue independently):

```bash
sudo /usr/bin/install -D -o root -g root -m 0755 \
  /home/thelifeandtimes/repos/omabit/build/urbit-runner \
  /usr/local/lib/urbit-runner/checkpoints/6u/urbit-runner

sudo /usr/bin/runuser -u urbit-runner -- /usr/bin/env -i \
  HOME=/var/lib/urbit-runner PATH=/usr/bin:/bin \
  /usr/local/lib/urbit-runner/checkpoints/6u/urbit-runner \
  serve-development --acknowledge-development
```

After the user confirms management, the agent runs:

```bash
/usr/bin/python3 -B /home/thelifeandtimes/repos/omabit/scripts/validate-managed-comet-maintenance.py
```

This fixed-development-target gate performs manual stop/pack/replay/resume on
`comet-check` (port 18087), then one timed stop/meld/resume operation. Both returns
must retain `~nolpel-modsyd-livsep-novmeg--sallen-tassul-padlud-litzod` and use
replacement containers. Further scheduler ticks must not duplicate the slot.
The exact temporary policy is disabled afterward. Existing policies or earlier 6U
work cause refusal instead of replacement/re-execution. Inspect any failed or
ambiguous result before further mutation.

Panel 0.9.0 is installed. A shell restart and live status IPC verified the loaded
version; the previous 0.8 snapshot was archived before replacement.
