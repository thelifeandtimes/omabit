# Checkpoint 6K: managed prefix and repeated pruning

**Live validation passed September 27 UTC.** Two confirmed pruning rounds on
alpha completed the five-to-three-to-two sequence. Retained records matched,
stopped/running historical replay remained exact, and final alpha was healthy at
revision 2/2 with `0i33536` and `0i33561` retained. See the
[acceptance evidence](../diagnostics/2026-09-27-managed-prefix.md).
The commands below remain a reproduction guide; this gate is complete.
6J's snapshot-bearing and repeated-prefix deletion/recovery gate passed with run
`5c72c70d9ed83d9b9f199d48c24c860c`.

## Supported behavior

- `epoch-prune-prefix` advertises confirmed oldest-prefix deletion through the
  existing `prune-epochs` endpoint and collector `epoch-prune` command.
- A selected cutoff includes itself and every older epoch. At least the newest
  two are protected; the inventory remains capped at 128 epochs.
- The pinned epoch-v2 layout, version headers, snapshots, exact file sets,
  exclusive file links, and metadata fingerprints are required.
- Admission binds the selected runner/instance, configuration revision, cutoff,
  inventory digest, explicit history-loss confirmation, and actor-scoped request ID.
- The shared qualified engine revalidates disk state before mutation and journals
  the whole prefix, each confirmed unlink, and each removed directory.
- New pruning jobs can follow completed jobs. Historical runtime evidence stays
  attached to prior operations, and exact replay remains valid after later pruning,
  resume, configuration drift, or management restart.
- A partial result remains quarantined. Stop and observation cannot turn missing
  completion or uncertain deletion evidence into permission to resume.
- The report includes selected/removed epoch names, aggregate file progress,
  retained names, and completion/uncertainty. Success leaves the pier stopped.

This is explicit development-pier maintenance. Metadata/allocation counts do not
establish backup integrity or guaranteed disk space reclaimed. Latest-epoch recovery
was qualified separately; automatic scheduling and older-epoch recovery have their
own gates.

## 1. Install and start 6K — user intervention

The agent has built the binaries. After active operations finish, press **Ctrl+C
in the current `serve-fake` terminal** and wait for it to exit. Existing ship
containers continue independently.

```bash
sudo install -D -o root -g root -m 0755 \
  /home/thelifeandtimes/repos/omabit/build/urbit-runner \
  /usr/local/lib/urbit-runner/checkpoints/6k/urbit-runner
```

Start management and leave it running:

```bash
sudo runuser -u urbit-runner -- /usr/bin/env -i \
  HOME=/var/lib/urbit-runner PATH=/usr/bin:/bin \
  /usr/local/lib/urbit-runner/checkpoints/6k/urbit-runner \
  serve-fake --acknowledge-development
```

In another terminal:

```bash
/home/thelifeandtimes/repos/omabit/build/urbit-collector \
  --format text --runner workstation hello
```

Expected: the same runner identity and SSH operator, with **`epoch-prune-prefix`**.
The legacy genesis capability remains advertised for compatible older clients.
Send the Hello output; the agent can run the bounded managed validation below.

If the SSH listener also needs restarting, use a separate terminal:

```bash
sudo /usr/bin/sshd -D -e -f /etc/urbit-runner/sshd_config
```

Use 6K or newer for subsequent management starts so the latest prefix record and
its recovery requirements remain fully understood.

## 2. Planned live gate

The target is development alpha, last healthy at revision 2/2 on port 18081.
Its current retained epochs are `0i22185`, `0i22199`, and `0i22213`; its old 6I
genesis operation remains replayable.

The agent will check current identity, state and RAM headroom before each stop:

1. Perform two explicit roll/resume cycles to produce five epochs. Use fixed IDs
   `checkpoint-6k-{stop,roll,resume}-prepare-{1,2}` with no automatic retries.
2. Stop/inspect alpha (`checkpoint-6k-stop-alpha-1`,
   `checkpoint-6k-inspect-alpha-1`) and record a fresh digest.
3. Preview through the second-oldest epoch, selecting both oldest snapshot-bearing
   epochs and retaining three. Submit explicit confirmed pruning with
   `checkpoint-6k-prune-alpha-1`.
4. Verify the exact selected/removed/retained sets, unchanged retained metadata,
   certain completed result, and exact replay. Resume with
   `checkpoint-6k-resume-alpha-1` and verify healthy 2/2 state.
5. Stop/inspect again with the round-2 IDs, select the oldest remaining epoch,
   and explicitly prune it with `checkpoint-6k-prune-alpha-2`, retaining newest two.
6. Verify the second report, resume with `checkpoint-6k-resume-alpha-2`, and replay
   both historical pruning requests while running. Prior operations must keep
   their original IDs/status/timestamps, and no replay may restore stale inventory.
7. Confirm direct HTTP `~zod` identity, healthy applied revision, fresh resource
   observations, and no unresolved maintenance outcome.

No current checkpoint is removed during this managed live gate; controlled
checkpoint-loss/recovery was the separate disposable 6J workflow.

## 3. Manual command reference

Define this helper in each terminal where it is used:

```bash
collect() {
  /home/thelifeandtimes/repos/omabit/build/urbit-collector \
    --format text --runner workstation \
    --expected-runner-id runner-f496feed1f7b08f31a5d82b94ac2f5fd "$@"
}
collect snapshot
free -m
```

After clean stop and successful `inspect-epochs`, use the actual cutoff and digest
from that stopped snapshot (replace placeholders before running):

```text
collect epoch-preview alpha --revision 2 --cutoff <selected-epoch> --expected-digest <actual-digest>
collect epoch-prune alpha --revision 2 --cutoff <selected-epoch> --expected-digest <actual-digest> --request-id <unique-request-id> --confirm-history-loss
```

Watch with `collect watch --interval 2s` and Ctrl+C the watcher when the operation
reaches its terminal result. A successful pruning report must have the intended
selected and removed names, complete file progress, no uncertainty, and the
protected retained suffix. Resume explicitly only after that verified result.

For replay, retain the original revision/cutoff/digest/confirmation/request ID.
Those describe the accepted operation even after its inventory changes. A new
pruning action always requires a fresh stopped inventory and a new request ID.

## Local validation

Passed checks:

- `go test ./...`
- `go vet ./...`
- `go test -race ./internal/epochs ./internal/runtimequalify ./internal/managed ./cmd/urbit-collector ./cmd/urbit-runner`
- `go build -o ./build/ ./cmd/...`
- Linux/ARM64 runner cross-compilation (build-only).
- Built collector's `epoch-prune --help` and `git diff --check`.

Tests cover multi-epoch and repeated deletion, latest-report replacement with
historical journal preservation, upgrade from a completed legacy genesis record,
independent quarantine markers, partial failure between selected epochs, stale
selection, protected suffixes, inventory bounds, capability fallback, confirmation,
observer restrictions, host-gate exclusion, and replay after management restart.
The disposable 6G/6J workflows continue to exercise the same unlink paths.
