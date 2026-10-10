# Checkpoint 6X: disposable comet prefix deletion and recovery

The [6W managed epoch gate](33-managed-comet-epochs.md) passed and panel 0.10 is
installed. This next increment qualifies **repeated prefix deletion and latest-epoch
recovery on a new disposable comet**. **Live qualification passed** for
`198fec0f0e49df2bcb0215789f2b5f1d`: both deletion/recovery rounds and all identity
checks completed, retaining `0i557` and `0i592`; all 17 containers and the bridge
were removed. See the [accepted evidence](../diagnostics/2026-09-29-comet-prefix-recovery.md).

## Exact workflow

1. Verify the pinned engine/image/pill and live limits. Generate a new comet in
   fresh private storage, with an owned outbound NAT bridge and loopback-only HTTP.
2. Four isolated roll/identity-preserving restart pairs prepare five complete
   epochs, including genesis and four snapshot-bearing epochs.
3. Revalidate an exact oldest-two preview; journal the complete unlink manifest and
   per-file progress. Delete genesis and the first snapshot-bearing epoch. Verify
   that the three retained epochs have unchanged metadata immediately after deletion.
4. Restart and verify the same identity and retained epoch set. Journal and remove
   only this disposable pier's current `.urb/chk/image.bin`, then run
   `/bin/urbit play --full --loom 31 /urbit/comet` with `network=none`. Require a
   restored nonempty checkpoint and another same-identity restart.
5. With fresh inventory, delete the oldest remaining epoch, retaining the protected
   newest two. Repeat ordinary restart, controlled checkpoint loss, isolated latest-
   epoch replay, and recovered restart.
6. Perform one final initialized-pier replacement and verify the same public
   identity and two retained epochs. Save the separate final retained inventory.
7. Remove all **17 containers** and the bridge; retain the private pier and report.

## Boundaries

- The qualifier accepts no existing identity, key, pier, cutoff, path or image
  input. Destruction is restricted to its newly generated qualification root.
- Managed operations still reject comet pruning. The low-level managed unlink
  boundary continues to admit only the separately qualified fake-pier profile.
- A selected oldest prefix is revalidated against its exact digest. Protected
  newest-two epochs are never selected. No recursive deletion is used.
- Every unlink and checkpoint-loss intent is persisted before mutation. A partial
  deletion, failed replay, changed retained metadata or changed identity halts the
  sequence. Cleanup cannot clear the unresolved outcome; no automatic retry or
  rollback occurs. Prior reports must contain both complete rounds, ordinary and
  recovered restarts, final retained inventory, and all resolved owned resources.
- Runtime phases use outbound NAT and only host-loopback HTTP; roll/replay utilities
  use `network=none`. There is no host Ames/UDP publication or peering claim.
- Each expensive phase requires **5 GiB available physical RAM / 8 GiB free disk**.
  Limits remain 4 GiB RAM, no additional swap, two CPUs, 256 PIDs, loom 31. Readiness
  is bounded at twenty minutes and each utility at ten minutes.
- This demonstrates the latest retained epoch's recovery path on the new comet.
  It does not verify backups, older-epoch recovery, live interrupted-deletion repair,
  or production runtime. Metadata sizes do not measure loom utilization.

## Local validation

Fixtures exercise the exact 19 unlink calls (two prefixes plus two checkpoint-loss
events), durable intent, retained metadata, two successful recoveries, all public
identity checks, final newest-two inventory, and 17-container/bridge cleanup.
Failure cases cover stale selection, partial prefix/snapshot unlink, changed retained
metadata, persistence failure, resource refusal before checkpoint loss, replay
timeout/missing restoration, and changed identity after deletion/recovery or final
replacement. History tests remove required evidence and require quarantine. Existing
fake-prefix and managed-comet refusal tests remain regression gates.

Passed locally: `go test ./...`, `go vet ./...`,
`go test -race ./internal/runtimequalify ./cmd/urbit-runner`, native builds, plan/help
CLI checks, and `git diff --check`. The larger recovery fixture uses a one-second
readiness budget after a 40 ms fixture-only race-test timeout; the affected normal
and race checks passed after that adjustment. Production timeouts were unchanged.
Linux/ARM64 cross-build succeeded at `/tmp/opencode/urbit-runner-6x-arm64`; that is
not ARM64 runtime qualification.

## Accepted live gate (reference; do not rerun for evidence)

Review the destructive scope:

```bash
/home/thelifeandtimes/repos/omabit/build/urbit-runner comet-prefix-plan --format text
```

Press **Ctrl+C** in the management terminal. The qualifier holds the management
lock through its cleanup; runtime containers continue independently.

```bash
sudo /usr/bin/install -D -o root -g root -m 0755 \
  /home/thelifeandtimes/repos/omabit/build/urbit-runner \
  /usr/local/lib/urbit-runner/checkpoints/6x/urbit-runner

sudo /usr/bin/runuser -u urbit-runner -- /usr/bin/env -i \
  HOME=/var/lib/urbit-runner PATH=/usr/bin:/bin \
  /usr/local/lib/urbit-runner/checkpoints/6x/urbit-runner \
  qualify-comet-prefix --acknowledge-disposable-run --format text
```

Expected: `Profile: comet-prefix-recovery`, `Checkpoint: PASSED`, two resolved
prefix/recovery rounds (five → three → two epochs), unchanged public identity,
final newest-two inventory and all 17 containers plus the bridge removed. Share
the result. If failed/interrupted, let cleanup finish and retain the report for
review before retrying.

After qualification exits, restore management:

```bash
sudo /usr/bin/runuser -u urbit-runner -- /usr/bin/env -i \
  HOME=/var/lib/urbit-runner PATH=/usr/bin:/bin \
  /usr/local/lib/urbit-runner/checkpoints/6x/urbit-runner \
  serve-development --acknowledge-development
```

The managed database remains schema 6; 6X includes 6W management support. Panel
0.10 is already installed, with comet retention still preview-only.
