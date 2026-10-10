# Checkpoint 6Y: confirmed managed comet prefix pruning

The [6X disposable gate](34-comet-prefix-recovery.md) passed repeated prefix
deletion and latest-epoch recovery with identity-preserving restarts. This increment
integrates explicitly confirmed prefix pruning into managed comets. **Managed live
validation passed**: five → three → two pruning retained `0i7082` and `0i7847`,
with exact immediate retained metadata, replay and identity-preserving resumes.
Comet-check finished healthy at 1/1; disabled policies were unchanged. See the
[accepted evidence](../diagnostics/2026-09-29-managed-comet-pruning.md).

## Contract

- New capability **`comet-prune-prefix`** extends the existing `prune-epochs` request
  to initialized, identity-pinned comets. Runner-wide fake pruning capabilities do
  not authorize a comet preview; the collector and panel require the new capability.
- A request binds runner/instance identity, configuration revision, exact stopped
  inventory digest, canonical cutoff and explicit history-loss confirmation. Only
  a contiguous oldest prefix may be selected, retaining at least the newest two.
- Admission requires an idle, initialized, clean stopped instance with completed
  private comet provisioning evidence. The worker revalidates the pinned layout,
  exclusive files and exact inventory before unlinking, under the host work gate.
- The registered root/profile boundary supports the qualified fake and comet paths.
  There is no arbitrary path or recursive deletion. Every intent and per-file
  progress update is durable; retained epoch metadata must match immediately after
  deletion. Each new prefix has a separate record, preserving historical evidence.
- Successful pruning leaves the pier stopped. Explicit Start checks the same public
  comet identity. A partial deletion remains quarantined through Stop, observation
  and restart even if the generic maintenance flag is absent; no automatic retry,
  rollback or resume occurs.
- Exact actor/request replay returns the original operation before current-state
  admission and never performs another deletion or resurrects stale inventory.
- The first accepted comet prune atomically advances private SQLite schema to **7**.
  Subsequent older-feature edits preserve that floor. Use **6Y+ `serve-development`**
  afterward; wire protocol version remains 1.
- Panel 0.11 requires a fresh matching preview and a separate history-loss checkbox.
  Capability changes alone cannot upgrade a previously non-deletable preview.
- Managed checkpoint removal/recovery is not exposed. Routine policies remain
  pack/meld only; pruning stays an explicit operator action. Logical/allocated byte
  counts are not guaranteed reclaimed space or verified backups.

## Validation

Runtime fixtures cover five → three → two pruning in the comet root, exact unlink
sets and retained metadata, persisted confirmation before unlink, state reload,
identity-pinned resumes and disposable-only recovery boundaries. Refusal tests cover
missing consent, stale/changed inventory, protected cutoffs, incomplete provenance,
shared files and running state. Partial-file/epoch and journal failures retain
quarantine through cleanup and observation.

Store tests cover confirmation, schema 7, interruption, repeated replay, archived
evidence and monotonic compatibility guards. Collector/model tests distinguish fake
and comet capabilities. The actual QML fixture exercises the transition from a
preview-only comet to a fresh deletable preview and explicit checkbox confirmation.

Passed locally:

```text
go test ./...
go vet ./...
go test -race ./internal/runtimequalify ./internal/managed ./cmd/urbit-collector
node scripts/test-panel-model.js
python3 -B scripts/test-panel.py --windows
python3 -B scripts/test-panel-install.py
go build -o ./build/ ./cmd/...
GOOS=linux GOARCH=arm64 go build -o /tmp/opencode/urbit-runner-6y-arm64 ./cmd/urbit-runner
git diff --check
```

The live validator subsequently passed. ARM64 was cross-built only; runtime
qualification remains Linux/AMD64.

## Accepted live gate (reference; do not rerun for evidence)

Press **Ctrl+C** in the management terminal, install the rebuilt runner and restore
management. Runtime containers continue independently.

```bash
sudo /usr/bin/install -D -o root -g root -m 0755 \
  /home/thelifeandtimes/repos/omabit/build/urbit-runner \
  /usr/local/lib/urbit-runner/checkpoints/6y/urbit-runner

sudo /usr/bin/runuser -u urbit-runner -- /usr/bin/env -i \
  HOME=/var/lib/urbit-runner PATH=/usr/bin:/bin \
  /usr/local/lib/urbit-runner/checkpoints/6y/urbit-runner \
  serve-development --acknowledge-development
```

After the user confirms management, the agent runs:

```bash
/usr/bin/python3 -B /home/thelifeandtimes/repos/omabit/scripts/validate-managed-comet-prune.py
```

This is **destructive history-retention validation on the existing development
`comet-check`**, pinned to identity
`~nolpel-modsyd-livsep-novmeg--sallen-tassul-padlud-litzod`, pier
`ee7b362a3bc23bc78f911ed1e08fb6e9`, configuration 1 and HTTP port 18087.

1. Require the accepted 6W baseline `0i0`, `0i7046`, `0i7064`, `0i7082`.
2. Perform one explicit preparation roll/resume to create a fifth epoch.
3. Stop and inspect; preview and confirm cutoff **`0i7046`**, removing genesis and
   that snapshot-bearing epoch while retaining three. Verify exact immediate
   retained metadata, per-file completion, replay and same-identity resume.
4. Stop and inspect again; preview and confirm cutoff **`0i7064`**, retaining the
   newest two (`0i7082` and the new preparation epoch). Verify metadata, replay and
   same-identity resume again.
5. A final stopped inspection verifies those two protected epochs after restart;
   another prune must be refused. Final explicit resume restores healthy 1/1 state.
   Historical replay must preserve the latest pruning result and cache invalidation.

Missing consent, stale digests and protected cutoffs are checked for refusal before
deletion. The validator refuses prior 6Y work, active policies, unresolved state,
changed baseline/identity, or disabled Python assertions. Existing policies must
remain unchanged. It never removes the current checkpoint. A failed step stops
without automatic resume/retry; reconcile its fixed request IDs before proceeding.
Panel **0.11.0 is installed**. A shell restart and live monitor status IPC verified
the loaded version; the previous 0.10 snapshot was archived before replacement.
