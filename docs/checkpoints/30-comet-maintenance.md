# Checkpoint 6T: comet pack/meld qualification

The 6S timed fake-pier pack routine passed and its test policy is disabled.
This next increment qualifies the pinned utilities on a **new disposable comet**
before enabling manual/routine comet compaction. **Live qualification passed** for
`5418c87ccd3aac4ef8a1597adedeae0c`: both utilities and all same-identity restarts
completed, with seven containers and the bridge removed. See the
[accepted evidence](../diagnostics/2026-09-29-comet-maintenance.md).

## Workflow

1. Verify pinned engine/image/pill and live container limits.
2. Create an owned outbound NAT bridge and bootstrap a new comet. Publish only
   loopback HTTP, verify the generated public identity, and cleanly stop.
3. Recheck RAM/disk admission; run `pack --loom 31 /urbit/comet` with `network=none`.
   Record logical bytes before/after; require zero exit and container removal.
4. Resume the same pier, require the exact original public identity, and stop.
5. Repeat the isolated utility and same-identity restart checks with `meld`.
6. Perform one final initialized-pier replacement resume and clean stop.
7. Remove all **seven containers** and the bridge; retain the private pier/report.

Utility intent is uncertain before dispatch. Failed utilities do not lead to a
restart or the next utility; cleanup cannot clear the quarantine. Prior report
admission requires both complete utility records and verified post-utility
restarts. The generated identity is retained exactly throughout.

Each expensive phase requires **5 GiB available RAM / 8 GiB free disk**. Runtime
readiness has a twenty-minute bound; each offline utility has a ten-minute bound.
The existing 4 GiB/no-extra-swap/two-CPU/256-PID/loom-31 limits apply. Utilities have
no network; runtimes use the owned NAT bridge with outbound Ames but no host UDP
publication. Logical bytes are not loom utilization or guaranteed reclaimed space.

This qualifier does not accept an existing comet, key, path or image override. It
does not enable comet maintenance in the management API, roll/pruning, or automatic
history deletion. Managed integration follows the live result.

## Local checks

Fixtures exercise successful pack/meld with measured size changes, each exact-
identity restart, network-isolated utility commands, admission refusal before
dispatch, changed post-pack identity, failed utilities without continuation,
missing/uncertain report evidence, and refusal in managed contexts. Existing
fake/comet lifecycle and routine workflows remain regression-tested.

## Accepted live gate (reference; do not rerun for evidence)

Review the plan:

```bash
/home/thelifeandtimes/repos/omabit/build/urbit-runner comet-maintenance-plan --format text
```

Press **Ctrl+C** in the management terminal; runtime containers continue
independently. Install the rebuilt qualifier:

```bash
sudo /usr/bin/install -D -o root -g root -m 0755 \
  /home/thelifeandtimes/repos/omabit/build/urbit-runner \
  /usr/local/lib/urbit-runner/checkpoints/6t/urbit-runner
```

Run as the dedicated account:

```bash
sudo runuser -u urbit-runner -- /usr/bin/env -i \
  HOME=/var/lib/urbit-runner PATH=/usr/bin:/bin \
  /usr/local/lib/urbit-runner/checkpoints/6t/urbit-runner \
  qualify-comet-maintenance --acknowledge-disposable-run --format text
```

Expected: `Profile: comet-offline-maintenance`, `Checkpoint: PASSED`, both utilities
uncertain=false with identity-preserving restarts=true, seven containers removed
and network cleanup complete. Share the qualification output. If interrupted,
allow cleanup to finish and retain the report for review before any retry.

After qualification exits, restore management:

```bash
sudo runuser -u urbit-runner -- /usr/bin/env -i \
  HOME=/var/lib/urbit-runner PATH=/usr/bin:/bin \
  /usr/local/lib/urbit-runner/checkpoints/6t/urbit-runner \
  serve-development --acknowledge-development
```
