# Checkpoint 6F: offline roll and multi-epoch lifecycle

**Live qualification passed September 26 UTC**, run
`c81cf35049c0b2f6b78b728af5bb9802`. All three rolls and subsequent restarts passed,
the preview selected `0i0` while retaining `0i66`, `0i101`, and `0i118`, and all
eight containers were removed with resolved outcomes. See the
[acceptance record](../diagnostics/2026-09-26-epoch-lifecycle.md).
The commands below remain a reproduction guide; this run needs no repetition.
The preceding
6E gate passed after the user freed host RAM: alpha resumed in 6.15 seconds and is
healthy at revision 2/2. Its single protected epoch could not exercise a positive
multi-epoch cutoff selection, so this checkpoint creates a fresh disposable pier.

## Workflow and scope

The new `qualify-epochs` command:

1. Holds both qualification and management-process locks through cleanup.
2. Verifies the pinned rootless engine, image, pill, live cgroup limits, and Vere.
3. Boots a fresh network-isolated fake `~zod`, checks private HTTP reachability,
   and stops/removes it. Records the initial `0i0` epoch.
4. Runs `/bin/urbit roll --loom 31 /urbit/fake-zod` in a dedicated utility container.
5. Requires clean exit/removal and exactly one newer epoch with a non-empty
   `image.bin`; every prior epoch name must remain present.
6. Restarts the same pier in a replacement container, checks private HTTP
   reachability and the same epoch set, then stops/removes it.
7. Repeats the roll/restart pair twice more: **three rolls, four epochs**.
8. Previews selecting only the oldest epoch and retaining **three** newer epochs.
   This exercises a selection different from stock chop's fixed two-epoch suffix.
9. Removes all eight owned containers and retains the report, pill, and multi-epoch
   pier in the dedicated account's qualification directory.

There is no epoch deletion in this workflow. Ordinary same-pier restart is not
proof of retained-suffix replay or backup recovery; those are subsequent gates.

The same qualified budget applies: 4 GiB container RAM, no extra swap, two CPUs,
256 PIDs, loom 31. At least **5 GiB available physical RAM** and **8 GiB free disk**
are required, rechecked before every boot and roll. Retained epoch snapshots consume
additional disk. The command accepts no existing pier, image, identity, cutoff,
or arbitrary utility override.

Uncertainty is persisted before each utility dispatch and cleared only after
clean completion, resource removal, inventory/snapshot checks, and successful
persistence. Zero exit without a new epoch fails qualification. An interrupted or
ambiguous roll is never retried or followed by another restart. Cleanup does not
clear that uncertainty; future qualification attempts require review of the report.

## 1. Review and install

The agent has built the executable. From any desktop terminal:

```bash
/home/thelifeandtimes/repos/omabit/build/urbit-runner epochs-plan --format text

sudo install -D -o root -g root -m 0755 \
  /home/thelifeandtimes/repos/omabit/build/urbit-runner \
  /usr/local/lib/urbit-runner/checkpoints/6f/urbit-runner
```

The plan is read-only. Review the fixed utility, resource budget, four-epoch goal,
and preview-only outcome. `install` copies the executable into the new checkpoint
directory; normal success is silent.

## 2. Pause management and run qualification

Check inventory and available RAM:

```bash
/home/thelifeandtimes/repos/omabit/build/urbit-collector \
  --format text --runner workstation snapshot
free -m
```

Allow some headroom above **5120 MiB** in the `available` column. After active
management jobs finish, press **Ctrl+C in the current `serve-fake` terminal** and
wait for it to exit. Existing ship containers continue running independently.
Keep management paused for the qualification so it can hold the process lock.

Run as the dedicated account:

```bash
sudo runuser -u urbit-runner -- /usr/bin/env -i \
  HOME=/var/lib/urbit-runner PATH=/usr/bin:/bin \
  /usr/local/lib/urbit-runner/checkpoints/6f/urbit-runner \
  qualify-epochs --acknowledge-disposable-run --format text
```

Expected phases:

```text
engine
image
pill
probe
boot
epoch-baseline
roll-1
epoch-restart-1
roll-2
epoch-restart-2
roll-3
epoch-restart-3
epoch-preview
cleanup
```

Allow several minutes for the fresh bootstrap, pinned ~207 MiB pill download, and
snapshot work. Each utility/readiness phase has a ten-minute deadline. Ctrl+C
requests bounded cleanup; let that cleanup finish.

## 3. Review the result and restore management

Expected final summary:

- `Checkpoint: PASSED`, profile `epoch-lifecycle`, and the exact report path.
- Cleanup complete, live limits observed, pier metadata seen.
- All three rolls have `uncertain=false`, completed timestamps, admission samples,
  and verified post-roll restart inventories.
- Epoch counts increase to 2, then 3, then 4. New nonzero epochs have snapshot bytes.
- Final preview selects one epoch and retains three; deletion supported is false.
- All eight containers are removed with resolved create outcomes; boot and all
  three replacement runtime containers recorded private HTTP reachability.
- Production runtime qualified remains false.

When the command has exited with **complete cleanup** (or refused before creating
a run), restore management and leave it running:

```bash
sudo runuser -u urbit-runner -- /usr/bin/env -i \
  HOME=/var/lib/urbit-runner PATH=/usr/bin:/bin \
  /usr/local/lib/urbit-runner/checkpoints/6f/urbit-runner \
  serve-fake --acknowledge-development
```

In another terminal:

```bash
/home/thelifeandtimes/repos/omabit/build/urbit-collector \
  --format text --runner workstation snapshot
```

The existing SSH listener remains compatible. If it needs restarting in another
terminal, use `sudo /usr/bin/sshd -D -e -f /etc/urbit-runner/sshd_config`.

**Send the qualification summary beginning `Checkpoint:` and the restored snapshot.**
If cleanup is incomplete, send the report before further resource work. Preserve
uncertain outcomes for review; do not edit journal flags or reuse the failed pier.
Use 6F or newer for future qualification commands because older qualifiers do not
recognize the new profile and eight-container report shape.

## Local evidence and next gate

Passed local checks:

- `go test ./...`
- `go vet ./...`
- `go test -race ./internal/epochs ./internal/runtimequalify ./internal/managed ./cmd/urbit-collector ./cmd/urbit-runner`
- `go build -o ./build/ ./cmd/...`
- Linux/ARM64 runner cross-compilation (build-only).
- Built runner's readable `epochs-plan` and `git diff --check`.

Tests cover the exact utility arguments, bootstrap separation, ordered epoch
growth, new snapshot presence, restart checks, prefix selection, persistence before
dispatch, resource re-admission, exit/OOM/timeout/no-op failures, uncertainty after
cleanup, management-lock exclusion, and strict historical-report validation.
Fixtures use temporary files and a simulated runtime; they do not establish live
Vere behavior or recovery semantics.

Pinned source reviewed:

- [`_cw_roll` argument handling](https://github.com/urbit/vere/blob/5a7614f/pkg/vere/main.c)
- [Epoch roll, snapshot copy, and fixed-suffix chop](https://github.com/urbit/vere/blob/5a7614f/pkg/vere/disk.c)

After this gate, selected-prefix deletion and retained-suffix replay/recovery need
their own disposable qualification. Roll remains off in managed operations, and
no automatic chop or maintenance schedule is introduced here.
