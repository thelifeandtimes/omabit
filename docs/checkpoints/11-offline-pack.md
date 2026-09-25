# Checkpoint 6A: explicit offline pack

**Live gate passed September 25, 2026.** The upgraded runner advertises
`offline-pack`. Alpha's pack and subsequent initialized-pier resume succeeded at
revision 2. Replaying `checkpoint-6a-pack-alpha` returned the original succeeded
operation after alpha was already running; it did not run another pack.

| Operation | Accepted (UTC) | Finished (UTC) | Elapsed |
| --- | --- | --- | --- |
| `checkpoint-6a-pack-alpha` | 16:31:59.368 | 16:32:01.728 | 2.36 s |
| `checkpoint-6a-resume-alpha` | 16:32:49.892 | 16:32:55.805 | 5.91 s |

Logical regular-file pier bytes changed from **1,046,691,852** to **637,779,980**
(408,911,872 bytes / 389.97 MiB / 39.1% smaller). This is not a measurement of
allocated disk space or loom utilization. The final observation was healthy at
`http://127.0.0.1:18081`, with configuration/applied revision 2/2, no active
operation, no recovery requirement, and no incidents. A direct `/~/host` request
returned `~zod`.

See the [evidence record](../diagnostics/2026-09-25-managed-pack.md). The following
steps remain a reproduction guide; this acceptance run needs no additional pack.

This adds the previously qualified offline `pack` primitive to the durable managed
runner API. It requires an initialized pier, an explicit clean stop, no retained
containers, and no unresolved maintenance outcome. Pack and runtime starts share
the host's expensive-work admission gate. Capacity remains at least 5 GiB available
RAM and 8 GiB free disk, with the same pinned Vere image and container limits.

Automated driver/control-plane tests passed for stop/pack/resume, request replay,
running/uninitialized-pier refusal, persisted intent before utility execution, and
interruption that remains recovery-required after observation or explicit stop.
The standalone pack primitive previously passed live checkpoint 2B; the managed
operation has now passed the live gate above.

Older 4B binaries do not advertise `offline-pack`; the collector refuses to submit
pack to them. The 6A executable is now installed on this development host.

## 1. Check current state with readable output

From the checkout, in your ordinary desktop terminal:

```bash
collect() {
  "$PWD/build/urbit-collector" --format text "$@"
}

collect inventory
collect --runner workstation snapshot
```

At the pre-pack observation, **alpha was stopped at revision 2**, **panel-check was
running/healthy at port 18085**, and **ssh-check was uninitialized**. Use the actual
snapshot as authority. The planned pack target is alpha, which already has an
initialized pier. Wait for any active operation to finish.

## 2. Copy and restart management

Press Ctrl+C only in the current `serve-fake` management terminal after all active
operations finish. Existing ship processes continue independently. Then, from the
checkout:

```bash
sudo install -D -o root -g root -m 0755 \
  ./build/urbit-runner \
  /usr/local/lib/urbit-runner/checkpoints/6a/urbit-runner
```

In the management terminal:

```bash
sudo runuser -u urbit-runner -- /usr/bin/env -i \
  HOME=/var/lib/urbit-runner PATH=/usr/bin:/bin \
  /usr/local/lib/urbit-runner/checkpoints/6a/urbit-runner \
  serve-fake --acknowledge-development
```

Keep the existing foreground SSH listener running. Its 4B restricted shell/bridge
is compatible with the new runner; no new account-shell change is required. If the
SSH listener also needs restarting, use a separate terminal:

```bash
sudo /usr/bin/sshd -D -e -f /etc/urbit-runner/sshd_config
```

## 3. Verify capability and perform pack

Back in the desktop terminal with the helper from step 1:

```bash
collect --runner workstation hello
collect --runner workstation snapshot
```

Hello must include **`offline-pack`**. Alpha must be stopped, initialized, and free
of recovery-required state. If alpha is currently running, stop it explicitly at
its current revision and wait for completion before packing.

For the previously observed alpha revision 2:

```bash
PACK_ID="pack-alpha-$(date +%s)"
collect --runner workstation operate alpha pack --revision 2 --request-id "$PACK_ID"
collect --runner workstation watch --interval 2s
```

Wait for the operation to succeed and alpha to remain stopped, then Ctrl+C the
watcher. Use the same request once more to verify replay, then inspect the result:

```bash
collect --runner workstation operate alpha pack --revision 2 --request-id "$PACK_ID"
collect --runner workstation snapshot
```

Expected: the same completed operation ID, SSH-key attribution, no incident or
recovery flag, and maintenance timestamps plus before/after **logical regular-file
pier bytes**. Those counts exclude symlinks and do not represent allocated disk
blocks, reclaimable space, or loom utilization. Pack need not reduce live data.

## 4. Resume the same pier

```bash
collect --runner workstation operate alpha start --revision 2 --request-id "resume-$PACK_ID"
collect --runner workstation watch --interval 2s
```

Wait for running/healthy at its applied endpoint, then Ctrl+C the watcher and send:

```bash
collect --runner workstation snapshot
```

Please report the pack/replay/resume results. This validates the managed primitive
before scheduling or more advanced maintenance is enabled.

## Interruption policy and limits

Before executing pack, the runner durably records that its outcome is uncertain.
Only clean utility exit, resource removal, pier validation, post-measurement, and
successful persistence clear that marker. An interrupted/failed pack blocks both
resume and another pack. Observation or container cleanup does not erase it. Share
the snapshot for review rather than deleting state or trying another bootstrap.

This checkpoint does not schedule pack, run meld, prune epochs, collect unsafe
control-socket metrics, or resize looms. Those need their own qualification and
recovery gates. Keep using explicit runtime stop/start; pack itself leaves the
pier stopped and never reapplies the bootstrap pill.
