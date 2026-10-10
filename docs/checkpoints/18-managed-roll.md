# Checkpoint 6H: explicit managed offline roll

**Live validation passed September 26 UTC.** Three roll/replay/resume cycles on
alpha created `0i22185`, `0i22199`, and `0i22213`, retaining `0i0`. Exact replay,
runtime-cycle refusal, positive prefix preview, and healthy 2/2 resume all passed.
See the [acceptance evidence](../diagnostics/2026-09-26-managed-roll.md).
The commands below remain a reproduction guide; this gate is complete.
Standalone roll and multi-epoch restart passed 6F. Oldest-only deletion and
latest-epoch recovery passed 6G on a separate disposable pier.

## Implemented contract

- `operate INSTANCE roll --revision N --request-id ID`, gated by `offline-roll`.
- Clean stopped, initialized registered development piers only.
- The same pinned utility, loom 31, 4 GiB RAM/no-extra-swap/two-CPU/256-PID budget.
  At least 5 GiB available RAM and 8 GiB disk, rechecked immediately before work.
- One expensive start/pack/meld/roll worker at a time. Retained utility intents
  remain a host admission barrier after a worker failure or management restart.
- Exactly one new, newer, snapshot-bearing epoch must be observed after success.
  Existing epoch names remain present. The inventory cap is checked before dispatch.
- Success leaves the pier stopped and preserves its configuration/applied revision.
  It records before/after epoch names, created epoch, file sizes, and admission
  evidence, and refreshes the cached inventory.
- Uncertain results remain quarantined after stop/observation. Empty/no-growth
  outcomes do not silently become a completed rollover.
- After successful roll, another new roll requires a runtime start and clean stop.
  The `roll_requires_run` marker survives stop, other maintenance, and management
  restart. Exact replay remains a lookup and bypasses new-operation admission.

This creates epochs explicitly; pruning and automatic scheduling retain separate
gates. Roll can increase logical/allocated pier size by adding a snapshot.

## 1. Install and start management — required user step

At the last check, management was stopped after qualification, while the SSH
listener remained active. Direct HTTP checks for alpha and panel-check both
returned HTTP 200 and `~zod`.

The agent has built the binaries. Copy the runner:

```bash
sudo install -D -o root -g root -m 0755 \
  /home/thelifeandtimes/repos/omabit/build/urbit-runner \
  /usr/local/lib/urbit-runner/checkpoints/6h/urbit-runner
```

If management has since been restarted, wait for active operations to finish and
press Ctrl+C in that `serve-fake` terminal before starting the new executable.
Existing ship containers continue independently.

Start management and leave it running:

```bash
sudo runuser -u urbit-runner -- /usr/bin/env -i \
  HOME=/var/lib/urbit-runner PATH=/usr/bin:/bin \
  /usr/local/lib/urbit-runner/checkpoints/6h/urbit-runner \
  serve-fake --acknowledge-development
```

In another desktop terminal:

```bash
/home/thelifeandtimes/repos/omabit/build/urbit-collector \
  --format text --runner workstation hello
```

Expected: the existing runner identity and SSH operator, with **`offline-roll`**
alongside the accepted maintenance/inventory capabilities. Send this Hello output;
the agent can perform the remaining development-pier validation programmatically.

The existing SSH bridge remains compatible. If its listener also needs restarting,
use another terminal with:

```bash
sudo /usr/bin/sshd -D -e -f /etc/urbit-runner/sshd_config
```

Use 6H or newer for subsequent management starts after the first managed roll:
older versions do not recognize its persisted roll-container references.

## 2. Bounded live sequence

The intended target is alpha, previously healthy at revision 2/2 on port 18081,
with a single epoch 0i0. Inspect current identity, inventory, and available RAM
before any stop. The known 6E resume refusal demonstrated why headroom matters.

The live gate performs **three explicit roll cycles**, each consisting of:

1. Clean stop at the freshly verified configuration revision.
2. One roll; wait for completion and verify exactly one new epoch and snapshot.
3. Exact replay while stopped; verify the same operation and timestamps.
4. Resume the same initialized pier and verify healthy applied revision.
5. Replay while running; verify no new roll and no stale inventory resurrection.

Request IDs are distinct per cycle:

| Round | Stop | Roll | Resume |
| --- | --- | --- | --- |
| 1 | `checkpoint-6h-stop-alpha-1` | `checkpoint-6h-roll-alpha-1` | `checkpoint-6h-resume-alpha-1` |
| 2 | `checkpoint-6h-stop-alpha-2` | `checkpoint-6h-roll-alpha-2` | `checkpoint-6h-resume-alpha-2` |
| 3 | `checkpoint-6h-stop-alpha-3` | `checkpoint-6h-roll-alpha-3` | `checkpoint-6h-resume-alpha-3` |

Before the final resume, the four-epoch inventory also supports an oldest-only
preview retaining three epochs. That preview remains non-executable.

For an operator reproducing round 1, define this helper in that terminal:

```bash
collect() {
  /home/thelifeandtimes/repos/omabit/build/urbit-collector \
    --format text --runner workstation \
    --expected-runner-id runner-f496feed1f7b08f31a5d82b94ac2f5fd "$@"
}
collect snapshot
free -m
```

For alpha still at revision 2:

```bash
collect operate alpha stop --revision 2 --request-id checkpoint-6h-stop-alpha-1
collect watch --interval 2s
```

After successful clean stop, Ctrl+C the watcher, then:

```bash
collect operate alpha roll --revision 2 --request-id checkpoint-6h-roll-alpha-1
collect watch --interval 2s
```

After successful roll, Ctrl+C the watcher, preserve the snapshot, replay, and resume:

```bash
collect snapshot
collect operate alpha roll --revision 2 --request-id checkpoint-6h-roll-alpha-1
collect operate alpha start --revision 2 --request-id checkpoint-6h-resume-alpha-1
collect watch --interval 2s
```

Once healthy/running at 2/2, Ctrl+C the watcher and replay the same roll request.
Rounds 2 and 3 use the matching IDs above and the same ordering, including runtime
start between rolls. Inspection/preview caches are invalidated by resume. The
final evidence should include healthy HTTP identity `~zod`, completed roll reports,
the final four-epoch preview, and no unresolved maintenance outcome.

## Validation and limitations

Passed local checks:

- `go test ./...`
- `go vet ./...`
- `go test -race ./internal/runtimequalify ./internal/managed ./cmd/urbit-collector ./cmd/urbit-runner`
- `go build -o ./build/ ./cmd/...`
- Linux/ARM64 runner cross-compilation (build-only).
- Built collector's `operate --help` and `git diff --check`.

Local tests cover qualified command construction, three managed roll/resume cycles,
fresh resource and inventory-cap refusal, no-growth/missing-snapshot/changed-epoch
outcomes, OOM/interruption and journal failures, quarantine after cleanup,
cross-kind host serialization, durable replay, and the persistent runtime-cycle
guard. Collector tests verify capability checks before mutation.

Vere's zero-exit empty-log path is deliberately not treated as proof of rollover.
If an unexpected no-growth result occurs after dispatch, share the operation and
snapshot for review. Cleanup alone does not establish that a quarantined pier is
safe to resume.
