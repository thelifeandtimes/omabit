# Checkpoint 6E: epoch inventory and prefix preview

**Live inspection/preview-guard/replay/resume passed September 25.** Alpha's only
epoch, `0i0`, was protected; protected, mismatched-digest, and missing-cutoff
previews were refused. After an initial low-RAM admission refusal, the user freed
memory and a new resume request succeeded in **6.15 s**. Alpha is healthy at 2/2;
running-state replay preserved the original operation without restoring the cache.
See the [accepted live record](../diagnostics/2026-09-25-epoch-inspection.md).
The preceding container-resource gate passed on September 25 for alpha and
panel-check.

## Scope

This is the inventory/selection foundation for epoch retention:

- `operate INSTANCE inspect-epochs --revision N --request-id ID` inspects a
  cleanly stopped, initialized, registered development pier.
- The existing operation endpoint supplies attribution, durable replay, revision
  checks, and per-instance exclusion. The existing SSH bridge remains compatible.
- Inspection reads directory/file metadata and bounded `epoc.txt`/`vere.txt`
  headers. It does not open the event database or run a Vere utility.
- Two scans must agree. Rooted filesystem access and ownership/type checks reject
  symlinks, special files, nested layouts, cross-device entries, unsupported
  versions, and excessive directory sizes.
- Inventory records canonical epoch names, logical bytes, allocated file bytes,
  file counts, shared-hardlink indicators, sample time, and a metadata digest.
- The newest two epochs are protected. A selected cutoff includes itself and
  every older epoch; arbitrary holes cannot be selected.
- `epoch-preview` uses a recent cached inventory, exact digest, and selected
  configuration revision. It performs no mutation and exposes no apply option.

The structural reader targets the pinned Vere 4.6 epoch-v2 layout under
`.urb/log/0i<event>`. It permits at most 128 epochs and 32 regular files per epoch.
This is not LMDB integrity or recovery validation. Previews report
`deletion_supported: false`; actual pruning requires further qualification and
fresh stopped-pier revalidation of the approved selection.

Allocated bytes are the sum of regular files' `st_blocks * 512`. Shared hardlinks
and filesystem extents can make this exceed space a deletion would free. No
reclaimed-space guarantee is inferred from those counts.

## 1. Upgrade management — user intervention

The binaries have been built. Check for active work:

```bash
/home/thelifeandtimes/repos/omabit/build/urbit-collector \
  --format text --runner workstation snapshot
```

After active operations finish, press **Ctrl+C in the existing `serve-fake`
terminal** and wait for it to exit. Existing ship containers continue independently.

Copy the executable:

```bash
sudo install -D -o root -g root -m 0755 \
  /home/thelifeandtimes/repos/omabit/build/urbit-runner \
  /usr/local/lib/urbit-runner/checkpoints/6e/urbit-runner
```

Restart management and leave it running:

```bash
sudo runuser -u urbit-runner -- /usr/bin/env -i \
  HOME=/var/lib/urbit-runner PATH=/usr/bin:/bin \
  /usr/local/lib/urbit-runner/checkpoints/6e/urbit-runner \
  serve-fake --acknowledge-development
```

Keep the existing foreground SSH listener available. Its restart command, if
needed in another terminal, is:

```bash
sudo /usr/bin/sshd -D -e -f /etc/urbit-runner/sshd_config
```

In a desktop terminal:

```bash
/home/thelifeandtimes/repos/omabit/build/urbit-collector \
  --format text --runner workstation hello
```

Expected: the same runner and SSH operator, with **`epoch-inventory`** in the
feature list. Send that output; the coding agent can perform the bounded live
stop/inspect/preview/resume sequence programmatically.

## 2. Live validation sequence

### Historical continuation after the admission refusal

The original stop and inspection requests below have succeeded. The original
`checkpoint-6e-resume-alpha` request records the resource-admission failure; exact
replay returns that failure. After available RAM exceeds 5 GiB (preferably with
headroom), resume using a **new** request ID:

```bash
/home/thelifeandtimes/repos/omabit/build/urbit-collector \
  --format text --runner workstation \
  --expected-runner-id runner-f496feed1f7b08f31a5d82b94ac2f5fd \
  operate alpha start --revision 2 \
  --request-id checkpoint-6e-resume-alpha-after-admission
```

This recovery request has now succeeded and its healthy-resume/running-state
replay checks passed. No additional work is needed for this acceptance run.

### Original reproduction sequence

The target is development instance alpha, last observed running/healthy at revision
2/2, port 18081. This gate briefly stops alpha to obtain a stable inventory.

For reproduction, define this helper in each terminal where it is used:

```bash
collect() {
  /home/thelifeandtimes/repos/omabit/build/urbit-collector \
    --format text --runner workstation \
    --expected-runner-id runner-f496feed1f7b08f31a5d82b94ac2f5fd "$@"
}
collect snapshot
```

If alpha is still at revision 2 and no operation is active:

```bash
collect operate alpha stop --revision 2 --request-id checkpoint-6e-stop-alpha
collect watch --interval 2s
```

After successful clean stop, Ctrl+C the watcher. Inspect:

```bash
collect operate alpha inspect-epochs --revision 2 --request-id checkpoint-6e-inspect-alpha
collect watch --interval 2s
```

After inspection succeeds, Ctrl+C the watcher and inspect/replay its result:

```bash
collect snapshot
collect operate alpha inspect-epochs --revision 2 --request-id checkpoint-6e-inspect-alpha
```

Expected: stopped alpha at revision 2/2, a source-bound inventory with canonical
names and the newest two entries protected, and exact operation replay. Preserve
this snapshot before resuming: lifecycle work invalidates the cached inventory.

Preview uses the exact digest from that snapshot and a listed cutoff:

```text
collect epoch-preview alpha --revision 2 --expected-digest <inventory-digest> --cutoff <listed-epoch>
```

Replace both placeholders with the actual values; they are not shell variables.
If there are only one or two epochs, every cutoff must be refused as protected.
With more epochs, selecting an older cutoff must show the exact oldest prefix and
the retained suffix. A successful preview states that deletion is unsupported.
Invented or protected cutoffs and a mismatched digest must be refused.

Once the inspection result has been reviewed, resume the same pier:

```bash
collect operate alpha start --revision 2 --request-id checkpoint-6e-resume-alpha
collect watch --interval 2s
```

After alpha is healthy at revision 2/2, Ctrl+C the watcher. Its cached epoch
inventory should be absent, while normal resource samples resume. Replaying the
old inspection request must not create a new disk scan or repopulate the cache.
Refreshing an inventory always uses a new inspection request ID while stopped.

## Local validation and follow-up

Passed local checks:

- `go test ./...`
- `go vet ./...`
- `go test -race ./internal/epochs ./internal/runtimequalify ./internal/managed ./cmd/urbit-collector ./cmd/urbit-runner`
- `go build -o ./build/ ./cmd/...`
- Linux/ARM64 runner cross-compilation (build-only).
- Built collector's `epoch-preview --help` and `git diff --check`.

Tests cover numerical ordering beyond JavaScript's exact-integer range, exact
prefix sets, protected suffixes, digest/age checks, allocation overflow, metadata
read boundaries, concurrent changes, hardlinks, symlink/special-layout refusal,
entry caps, durable inspection replay, late-poll suppression, and cache invalidation.
Fixture epoch databases are deliberately not real LMDB files: inspection must
not open or interpret event contents.

The pinned [Vere disk implementation](https://github.com/urbit/vere/blob/5a7614f/pkg/vere/disk.c)
defines these names/version headers. Its stock `chop` deletes all but the newest
two epochs and cannot express an arbitrary selected cutoff. A later destructive
gate must qualify exact prefix deletion and the retained suffix's claimed recovery
paths on a disposable multi-epoch pier.
