# Checkpoint 6V: disposable comet epoch lifecycle

The [6U managed compaction gate](31-managed-comet-compaction.md) passed; its temporary
policy is disabled and panel 0.9 is loaded. This increment qualifies **epoch rolls
on a new disposable comet** before managed comet epoch controls are enabled.
**Live qualification passed** for `e84ffd83afcfbbe71e6cceca2d69c30f`: all three
rolls, exact-identity restarts, four-epoch preview and nine-container/bridge cleanup
completed. See the [accepted evidence](../diagnostics/2026-09-29-comet-epochs.md).

## Workflow and boundaries

1. Verify the pinned engine/image/pill, live container limits and private ownership.
2. Bootstrap a new comet using an owned outbound NAT bridge and loopback-only HTTP.
   Pin the generated public identity, then cleanly stop.
3. Inspect the initial canonical `0i0` epoch using bounded, rooted/no-follow metadata
   inspection. The comet path is selected only for this standalone profile.
4. Perform three explicit `/bin/urbit roll --loom 31 /urbit/comet` invocations with
   `network=none`. Each must add exactly one newer, complete snapshot-bearing epoch.
   Each roll is followed by a same-identity replacement restart and clean stop;
   the complete epoch set must survive each restart.
5. Perform one additional initialized-pier replacement resume. Reinspect the four
   epochs and produce an oldest-only preview retaining three newer epochs. **No
   epoch or current checkpoint is deleted.** The newest two remain protected.
6. Remove all **nine containers** and the bridge; retain the private comet pier and
   report. Overall uncertainty clears only after the final restart and preview.

Fresh admission requires **5 GiB available physical RAM / 8 GiB free disk** before
expensive phases. Containers retain the qualified 4 GiB/no-extra-swap/two-CPU/
256-PID/loom-31 limits. Each readiness check is bounded at twenty minutes; each
utility is bounded at ten minutes. Runtime outbound Ames is available, but no host
UDP port is published and no peering claim is made.

There are no supplied identity, key, pier, cutoff or image inputs. A failed utility,
missing snapshot, changed epoch set or changed identity prevents continuation.
Cleanup cannot erase unresolved roll/provisioning evidence. Prior-report admission
requires all three complete roll/restart records, final preview, exact comet
outcome, and matching cleaned-up container/network records.

The shared fake epoch workflow remains qualified separately. The managed comet API
still permits lifecycle, pack/meld and routines; roll/inspection/pruning/recovery
need subsequent integration and qualification. Metadata inventories and logical/
allocated byte counts are neither verified backups nor loom telemetry.

## Validation

Fixtures cover three rolls, snapshots, before/after and post-restart inventories,
command/network/profile boundaries, durable intent before dispatch, unchanged
identity, final preview without deletion, and complete cleanup. Failure cases
include timeout, absent new epoch, missing snapshot, restart epoch drift, resource
refusal and changed identity during an intermediate or final restart. Report tests
remove required evidence and require refusal. CLI tests reject existing-pier/key/
cutoff arguments and missing acknowledgement.

Passed locally: `go test ./...`, `go vet ./...`,
`go test -race ./internal/runtimequalify ./cmd/urbit-runner`, native command builds,
the plan/help CLI checks, and `git diff --check`. Linux/ARM64 cross-build succeeded
at `/tmp/opencode/urbit-runner-6v-arm64`; this is not ARM64 runtime qualification.

## Accepted live gate (reference; do not rerun for evidence)

Review the plan:

```bash
/home/thelifeandtimes/repos/omabit/build/urbit-runner comet-epochs-plan --format text
```

Press **Ctrl+C** in the management terminal. Runtime containers continue
independently; the qualifier holds the management lock through cleanup.

```bash
sudo /usr/bin/install -D -o root -g root -m 0755 \
  /home/thelifeandtimes/repos/omabit/build/urbit-runner \
  /usr/local/lib/urbit-runner/checkpoints/6v/urbit-runner

sudo /usr/bin/runuser -u urbit-runner -- /usr/bin/env -i \
  HOME=/var/lib/urbit-runner PATH=/usr/bin:/bin \
  /usr/local/lib/urbit-runner/checkpoints/6v/urbit-runner \
  qualify-comet-epochs --acknowledge-disposable-run --format text
```

Expected: `Profile: comet-epoch-lifecycle`, `Checkpoint: PASSED`, three resolved
rolls with post-restart inventories, four final epochs, selected=1/retained=3,
`Deletion supported: false`, comet uncertainty=false, and nine containers plus the
bridge removed. Share the result. If interrupted or failed, retain the report and
allow cleanup to finish for review before any retry.

After qualification exits, restore management:

```bash
sudo /usr/bin/runuser -u urbit-runner -- /usr/bin/env -i \
  HOME=/var/lib/urbit-runner PATH=/usr/bin:/bin \
  /usr/local/lib/urbit-runner/checkpoints/6v/urbit-runner \
  serve-development --acknowledge-development
```

Use 6U+ for the current schema-5 managed state; 6V includes that support. Panel 0.9
is already installed; this standalone qualification does not require a UI update.
