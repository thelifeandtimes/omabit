# Checkpoint 6G: disposable prefix deletion and latest-epoch recovery

**Live qualification passed September 26 UTC**, run
`e31441cd9345af32611b0153afc0cff7`. Only `0i0` was removed; all three retained epoch
fingerprints matched. Ordinary restart and recovery from `0i128` after deleting
the current checkpoint both passed, and all eleven containers were removed.
See the [acceptance record](../diagnostics/2026-09-26-prefix-prune-recovery.md).
The commands below remain a reproduction guide; this gate needs no repetition.
The prerequisite 6F live gate passed with run `c81cf35049c0b2f6b78b728af5bb9802`.

## Exact scope

`qualify-prune` creates a **new disposable fake pier**. It does not adopt the
retained 6F pier or accept an existing-pier, cutoff, image, or path override.
It holds the qualification and management-process locks throughout the run and
cleanup, using the pinned image/pill and the existing fixed development limits.

The sequence is:

1. Reproduce the qualified fresh boot and three roll/restart pairs, yielding four
   epochs and an oldest-only preview retaining three newer epochs.
2. Re-read the stopped pier and require the preview's exact inventory digest.
3. Validate and persist an exact manifest for the four regular files in genesis
   epoch `0i0`: `data.mdb`, `epoc.txt`, `lock.mdb`, and `vere.txt`.
4. Persist uncertainty before any unlink. Recheck file identity immediately before
   each unlink, sync the directory, and journal progress. Remove only the now-empty
   `0i0` directory and sync its parent. No recursive deletion or stock chop is used.
5. Require exactly three retained epochs, with unchanged per-epoch metadata
   fingerprints, sizes, identities, and protection flags.
6. Restart the same pier normally, check private HTTP reachability, and stop it.
7. Persist a separate recovery intent, then remove only this disposable pier's
   current `.urb/chk/image.bin`. Verify that it is absent.
8. Run `/bin/urbit play --full --loom 31 /urbit/fake-zod`. Require clean exit/removal,
   a restored non-empty checkpoint, and the same three retained epoch names.
9. Restart once more, verify private HTTP reachability and the retained epoch set,
   then stop/remove all owned containers. Retain the report and resulting pier.

Pinned Vere's `play --full` restores from the **latest** epoch snapshot. This gate
therefore tests latest-retained-epoch recovery after losing the current checkpoint;
it does not establish recovery from every older retained epoch or backup integrity.
The runtime's normal replay integrity checks stay enabled.

The three-epoch retained suffix distinguishes this selection from stock chop,
which retains only two. Managed roll/prune remain disabled, and public previews
continue to report `deletion_supported: false`.

## Bounds and failure behavior

- Only the new qualification directory and fixed generated fake pier are eligible.
- All preceding containers must be removed with resolved outcomes before deletion
  or checkpoint removal. Existing managed piers are not targets.
- Rooted filesystem access, ownership/type checks, exclusive links, exact file
  names, and inode/metadata checks precede deletion. Selected or retained hardlinks
  are refused in this bounded profile.
- Partial unlink, failed persistence, changed retained data, or interrupted replay
  remains uncertain after resource cleanup. No automatic retry, rollback, or boot
  of that uncertain pier follows.
- At least 5 GiB available physical RAM and 8 GiB free disk are required before
  expensive work. Containers retain the 4 GiB/no-extra-swap/two-CPU/256-PID budget.
- Allocated file bytes are not guaranteed disk space reclaimed; shared filesystem
  extents can affect actual reclamation.

## 1. Review and copy the executable

The agent has built the binary. From any desktop terminal:

```bash
/home/thelifeandtimes/repos/omabit/build/urbit-runner prune-plan --format text

sudo install -D -o root -g root -m 0755 \
  /home/thelifeandtimes/repos/omabit/build/urbit-runner \
  /usr/local/lib/urbit-runner/checkpoints/6g/urbit-runner
```

The plan describes the two destructive actions on the new fake: deleting its
selected genesis epoch and later deleting its current checkpoint for recovery.

## 2. Pause management and run the qualification

Check active work and available RAM:

```bash
/home/thelifeandtimes/repos/omabit/build/urbit-collector \
  --format text --runner workstation snapshot
free -m
```

After active operations finish, press **Ctrl+C in the current `serve-fake`
terminal** and wait for it to exit. Keep management stopped during qualification.
Existing ship containers continue independently. Allow some headroom above
5120 MiB in the `available` column.

Run this complete command:

```bash
sudo runuser -u urbit-runner -- /usr/bin/env -i \
  HOME=/var/lib/urbit-runner PATH=/usr/bin:/bin \
  /usr/local/lib/urbit-runner/checkpoints/6g/urbit-runner \
  qualify-prune --acknowledge-disposable-run --format text
```

After the familiar 6F phases, expect `prune-prefix`, `post-prune`, `epoch-recovery`,
`post-recovery`, and `cleanup`. Allow several minutes for the fresh bootstrap,
snapshot work, and replay. Ctrl+C requests bounded cleanup; let cleanup finish.

## 3. Check the report and restore management

Expected final evidence:

- `Checkpoint: PASSED`, profile `epoch-prune-recovery`.
- All roll/restart phases pass, followed by all four new phases.
- Prefix deletion: cutoff `0i0`, `uncertain=false`, `epoch-removed=true`, files
  removed **4 / 4**, retained metadata verified true, and three retained epochs.
- Ordinary post-prune restart verified true.
- Latest-epoch recovery: `uncertain=false`, current-checkpoint-removed true,
  positive restored-checkpoint bytes, and post-recovery restart verified true.
- All **eleven** containers removed with resolved outcomes; cleanup complete true.
- Production runtime qualified false.

When the command exits with complete cleanup (or refuses before creating a run),
restore management and leave it running:

```bash
sudo runuser -u urbit-runner -- /usr/bin/env -i \
  HOME=/var/lib/urbit-runner PATH=/usr/bin:/bin \
  /usr/local/lib/urbit-runner/checkpoints/6g/urbit-runner \
  serve-fake --acknowledge-development
```

In another desktop terminal:

```bash
/home/thelifeandtimes/repos/omabit/build/urbit-collector \
  --format text --runner workstation snapshot
```

The existing SSH listener remains compatible. If needed, restart it in a separate
terminal with `sudo /usr/bin/sshd -D -e -f /etc/urbit-runner/sshd_config`.

**Send the final qualification summary beginning `Checkpoint:` and the restored
snapshot.** If cleanup is incomplete, send the report before further resource work.
Keep failed/uncertain data and report for review; do not change flags to bypass the
gate. Use 6G or newer for future qualification commands because older versions do
not understand this profile, deletion/recovery records, or eleven-container shape.

## Local validation and source basis

Passed local checks:

- `go test ./...`
- `go vet ./...`
- `go test -race ./internal/epochs ./internal/runtimequalify ./internal/managed ./cmd/urbit-collector ./cmd/urbit-runner`
- `go build -o ./build/ ./cmd/...`
- Linux/ARM64 runner cross-compilation (build-only).
- Built runner's readable `prune-plan` and `git diff --check`.

Fixture tests verify exact unlink paths, intent-before-mutation, directory sync and
progress recording, stale selection, changed files, unexpected files, partial
unlink, retained metadata drift, journal failures, checkpoint loss/restoration,
replay failure, failed post-recovery readiness, history quarantine, and management
lock exclusion. Temporary fake epoch data is not a live recovery qualification.

Pinned source:

- [`play --full` argument handling](https://github.com/urbit/vere/blob/5a7614f/pkg/vere/main.c)
- [Latest epoch selection and checkpoint restore](https://github.com/urbit/vere/blob/5a7614f/pkg/vere/disk.c)
- [Replay and mug checks](https://github.com/urbit/vere/blob/5a7614f/pkg/vere/mars.c)
