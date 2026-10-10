# Checkpoint 6J: snapshot-bearing and repeated-prefix qualification

**Live qualification passed September 27 UTC**, run
`5c72c70d9ed83d9b9f199d48c24c860c`. The first prefix removed `0i0` and `0i77`,
retaining three epochs; the second removed `0i103`, retaining `0i129` and `0i146`.
Both deletion/restart/recovery rounds passed, and all sixteen containers were
removed with resolved outcomes. See the [evidence](../diagnostics/2026-09-27-repeated-prefix.md).
The commands below remain a reproduction guide; this gate is complete.
6I's managed genesis prune/replay/resume passed on alpha, which finished healthy
at revision 2/2. This checkpoint creates a separate new fake pier.

## What is being qualified

The existing live evidence covers removing genesis `0i0` while retaining three
newer epochs. This workflow additionally tests deletion of snapshot-bearing epochs
and another cleanup after genesis has already been removed:

1. Verify the pinned engine/image/pill and live resource limits.
2. Boot a fresh isolated fake and perform **four roll/restart pairs**, producing
   five epochs.
3. Preview and delete the **oldest two** epochs: genesis and the first epoch with
   a snapshot. Retain the newer three with unchanged metadata fingerprints.
4. Verify ordinary restart, remove the disposable current checkpoint, run pinned
   `play --full` recovery from the latest retained epoch, and verify restart again.
5. Obtain a fresh bound inventory, select and delete the **oldest remaining**
   snapshot-bearing epoch, and retain the newest two.
6. Repeat ordinary restart, current-checkpoint removal, latest-epoch recovery,
   and post-recovery restart.
7. Remove all **sixteen** owned containers and retain the report and two-epoch pier.

Selection is always a contiguous oldest prefix. The first round retains three,
so it cannot be substituted with stock chop's fixed newest-two behavior. Both
rounds respect the protected newest suffix. Managed pruning remains restricted
to its existing genesis-only capability while this live gate is pending.

## Mutation and recovery boundaries

- The command has no existing-pier, path, image, or caller-cutoff override.
- Qualification and management locks remain held through cleanup.
- All owned containers must be removed before filesystem mutation.
- Each selected epoch has a complete, bounded, exclusive-file manifest. Nonzero
  epochs include `image.bin`; genesis has the four previously qualified files.
- The whole prefix is durably recorded as uncertain before any unlink. Each file
  is rechecked, unlinked, directory-synced, and journaled. Each empty epoch directory
  is then removed and its parent synced. There is no recursive delete or hidden
  directory manipulation.
- Retained epoch metadata must match exactly after deletion. A partial failure,
  changed file, failed persistence, or interrupted replay remains quarantined.
  Resource cleanup does not authorize a retry or boot of that uncertain pier.
- Each recovery removes only the new fake's current checkpoint before invoking
  `play --full`. This is latest-retained-epoch recovery, not proof that every older
  retained snapshot is independently recoverable or that a backup is intact.

The fixed budget remains 4 GiB container RAM, no additional swap, two CPUs, 256 PIDs,
loom 31. At least 5 GiB available physical RAM and 8 GiB disk are rechecked before
expensive work. Epoch snapshots consume additional storage; allocated file-byte
counts are not guaranteed reclaimed disk space.

## 1. Review and install

The binaries have been built. From any desktop terminal:

```bash
/home/thelifeandtimes/repos/omabit/build/urbit-runner prefix-plan --format text

sudo install -D -o root -g root -m 0755 \
  /home/thelifeandtimes/repos/omabit/build/urbit-runner \
  /usr/local/lib/urbit-runner/checkpoints/6j/urbit-runner
```

The plan describes the two destructive rounds and the two checkpoint-loss tests
on the newly created fake.

## 2. Pause management and run the complete command

Check current work and RAM:

```bash
/home/thelifeandtimes/repos/omabit/build/urbit-collector \
  --format text --runner workstation snapshot
free -m
```

After active jobs finish, press **Ctrl+C in the current `serve-fake` terminal**
and wait for it to exit. Keep management stopped for qualification. Existing
managed ship containers continue independently; allow headroom above 5120 MiB
in the `available` RAM column.

```bash
sudo runuser -u urbit-runner -- /usr/bin/env -i \
  HOME=/var/lib/urbit-runner PATH=/usr/bin:/bin \
  /usr/local/lib/urbit-runner/checkpoints/6j/urbit-runner \
  qualify-prefix --acknowledge-disposable-run --format text
```

After the familiar setup phases, expect four roll/restart pairs, `epoch-preview`,
then `prefix-delete-N`, `prefix-restart-N`, `prefix-recovery-N`, and
`prefix-recovered-N` for rounds 1 and 2, followed by cleanup. Allow several minutes.
Each utility/readiness phase has a ten-minute deadline. Let cancellation cleanup
finish if Ctrl+C is used.

## 3. Check the report and restore management

Expected final evidence:

- `Checkpoint: PASSED`, profile `epoch-prefix-recovery`.
- Four successful roll/restart pairs, with five prepared epochs.
- Round 1: two selected epochs removed, three retained unchanged, `uncertain=false`.
- Round 2: one selected non-genesis epoch removed, newest two retained unchanged,
  `uncertain=false`.
- Both ordinary restarts and both post-recovery restarts verified.
- Both current-checkpoint removals confirmed; both restored checkpoints nonempty;
  both recovery outcomes certain.
- All sixteen containers removed with resolved outcomes; cleanup complete true.
- Production runtime qualified remains false.

After the command exits with complete cleanup (or refuses before creating a run),
restore management and leave it running:

```bash
sudo runuser -u urbit-runner -- /usr/bin/env -i \
  HOME=/var/lib/urbit-runner PATH=/usr/bin:/bin \
  /usr/local/lib/urbit-runner/checkpoints/6j/urbit-runner \
  serve-fake --acknowledge-development
```

In another desktop terminal:

```bash
/home/thelifeandtimes/repos/omabit/build/urbit-collector \
  --format text --runner workstation snapshot
```

The existing SSH listener remains compatible. If needed, restart it in another
terminal with `sudo /usr/bin/sshd -D -e -f /etc/urbit-runner/sshd_config`.

**Send the qualification summary beginning `Checkpoint:` and the restored
snapshot.** If cleanup is incomplete, share the report before further resource
work. Preserve uncertain outcomes for review. Use 6J or newer for later
qualification commands: older binaries do not understand the new profile,
fourth-roll records, repeated-prefix history, or sixteen-container shape.

## Local evidence

Passed checks:

- `go test ./...`
- `go vet ./...`
- `go test -race ./internal/epochs ./internal/runtimequalify ./internal/managed ./cmd/urbit-collector ./cmd/urbit-runner`
- `go build -o ./build/ ./cmd/...`
- Linux/ARM64 runner cross-compilation (build-only).
- Built runner's readable `prefix-plan` and `git diff --check`.

Tests cover exact two-round unlink sets (including snapshot files), manifest
persistence before mutation, interrupted work between epochs, partial snapshot
deletion, stale selections in either round, retained metadata drift, journal
failures, both recovery interruptions, failed final readiness, protected suffixes,
strict history validation, and management-lock exclusion. The same regression
suite continues to cover accepted genesis-only managed and disposable behavior.

The existing pinned Vere source basis applies: `roll` creates the snapshots,
`play --full` restores from the latest epoch, and the runtime performs its normal
replay integrity checks. This live gate is required before broader managed
prefix deletion is enabled.
