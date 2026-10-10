# Checkpoint 6O: stopped-pier copy and resume qualification

The first provisioning foundation is implemented and locally tested. **Live
qualification passed**, run `185ff2ed58b9abdd6861dfe64d6650f5`; see the
[acceptance record](../diagnostics/2026-09-28-stopped-pier-copy.md). This proves the transfer primitive before a managed
import/upload API or the Setup “Existing pier” action is enabled.

## Scope

`urbit-runner qualify-import` creates a new isolated fake, initializes it with the
pinned image/pill, verifies public identity `~zod`, and cleanly stops/removes its
container. It then:

1. Measures the stopped tree and rechecks disk capacity.
2. Journals uncertain transfer intent and reserves the original under `source/`.
3. Copies into a fresh `data/` tree with owner-only permissions.
4. Verifies file contents and sorted tree layout using SHA-256 on both sides.
5. Resumes the copy without creation flags, checks `~zod`, and cleanly stops it.
6. Repeats with a replacement runtime, then verifies the original source's content
   is unchanged.

All four containers are removed. Both private piers and the report remain.
Management's process lock is held throughout, including cleanup. Running managed
ships can continue independently. No existing-pier path, identity key, image,
network binding or command override is accepted by this qualifier.

The copier bounds the tree to 8,192 entries, depth 32, and 8 GiB logical bytes. It
rejects symlinks, hardlinked regular files, special files, cross-device entries,
unexpected ownership, changed metadata/content, and a nonempty destination. Reads
use rooted handles and no-follow opens; writes are exclusive and synced. Progress
is journaled per completed file. Failures retain the partial copy and uncertain
record; cleanup does not clear quarantine or retry the transfer.

This is content-transfer evidence for the pinned fresh fake, not proof that an
arbitrary external/production pier is stopped, compatible, or a valid backup.
Managed staging, external import validation, keys/comets, Ames and schedules remain
later backend increments.

## Local checks

Go tests/vet and relevant race checks passed. Fixtures cover content identity,
private permissions, unsafe trees, oversized/deep input, source/target mutation,
interrupted progress, target-only resumed writes, replacement persistence, partial
outcome quarantine and missing prior-report evidence. CLI tests reject path
overrides and missing acknowledgement. The ARM64 cross-build is build-only.

## Live gate — user intervention

The check needs **at least 5 GiB available RAM** before source boot and each resume.
Before copying, free disk must exceed source logical bytes **plus 8 GiB**. The last
workstation check showed about **4.5 GiB available RAM**; free some memory first.

Inspect the plan from any directory:

```bash
/home/thelifeandtimes/repos/omabit/build/urbit-runner import-plan --format text
free -h
```

Once headroom is sufficient, press **Ctrl+C in the management `serve-fake`
terminal**, then install the rebuilt qualifier:

```bash
sudo /usr/bin/install -D -o root -g root -m 0755 \
  /home/thelifeandtimes/repos/omabit/build/urbit-runner \
  /usr/local/lib/urbit-runner/checkpoints/6o/urbit-runner
```

Run the standalone qualification as the dedicated account:

```bash
sudo runuser -u urbit-runner -- /usr/bin/env -i \
  HOME=/var/lib/urbit-runner PATH=/usr/bin:/bin \
  /usr/local/lib/urbit-runner/checkpoints/6o/urbit-runner \
  qualify-import --acknowledge-disposable-run --format text
```

Expected result: `Profile: stopped-pier-copy`, `Checkpoint: PASSED`, cleanup true,
uncertain false, verified copy, both resumes true, and original unchanged true.
Share this **qualification result**, not the management snapshot. Allow cancellation
cleanup to finish if interrupted; do not rerun a failed/uncertain transfer blindly.

After qualification exits, restore management and leave it running:

```bash
sudo runuser -u urbit-runner -- /usr/bin/env -i \
  HOME=/var/lib/urbit-runner PATH=/usr/bin:/bin \
  /usr/local/lib/urbit-runner/checkpoints/6o/urbit-runner \
  serve-fake --acknowledge-development
```
