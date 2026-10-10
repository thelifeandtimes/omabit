# Checkpoint 6B: disposable offline meld

**Live qualification passed September 25, 2026.** The user's report for run
`9b5dfcb57a677f7cf1a2b833e6f65274` shows all phases passed, including same-pier
post-meld restart, complete cleanup, and no uncertain outcome. Meld took **17.32 s**;
logical pier bytes changed from **1,016,840,204** to **571,572,236** (43.8% smaller).
See the [acceptance record](../diagnostics/2026-09-25-offline-meld.md).
The commands below remain a reproduction guide; this run does not need repeating.

## Scope and workflow

`qualify-meld` creates a fresh, network-isolated fake `~zod` under the dedicated
account's qualification directory. It accepts no pier, identity, image, or command
override. It uses the same pinned Docker image, Vere `4.6-5a7614f`, and upstream
pill as the accepted lifecycle/pack checkpoints.

The workflow is:

1. Acquire the qualification lock and the existing management-process lock. Both
   remain held through cleanup, excluding concurrent managed start/pack work even
   with the previously installed 6A management binary.
2. Verify the private rootless engine, image/pill pins, live cgroup limits, and
   Vere version.
3. Boot a fresh fake, check private HTTP reachability, stop it cleanly, and remove
   its container.
4. Measure logical regular-file pier bytes and recheck available RAM/disk.
5. Persist an uncertain-maintenance marker before dispatching
   `/bin/urbit meld --loom 31 /urbit/fake-zod` in an exclusive utility container.
6. Require successful wait **and** independent clean-exit/no-OOM inspection,
   remove the utility, validate pier metadata, measure again, and persist the
   completed outcome before clearing uncertainty.
7. Start a replacement against the same pier, check private HTTP reachability,
   then stop/remove it. Bootstrap flags and the pill mount are used only once.
8. Retain the private report, downloaded pill, and disposable pier.

Each container has **4 GiB RAM, no additional swap, two CPUs, and 256 PIDs**;
the loom exponent is **31**. Admission requires **5 GiB available RAM** and
**8 GiB free disk**, initially and again immediately before meld. This is a
fixed budget being qualified for this fresh fake; it is not a memory estimate
for arbitrary populated piers. An OOM or capacity refusal is a failed gate,
never an implicit limit increase or retry.

The utility was reviewed in the pinned
[Vere source](https://github.com/urbit/vere/blob/5a7614f/pkg/vere/main.c): `_cw_meld`
parses `--loom`, loads the pier, runs `u3_meld_all`, then saves state. Source review
does not substitute for the live run.

## 1. Review and copy the built executable

The following commands use absolute paths for this workstation and work from any
desktop terminal. The binary has already been built by the coding agent.

```bash
/home/thelifeandtimes/repos/omabit/build/urbit-runner meld-plan --format text

sudo install -D -o root -g root -m 0755 \
  /home/thelifeandtimes/repos/omabit/build/urbit-runner \
  /usr/local/lib/urbit-runner/checkpoints/6b/urbit-runner
```

Review the image/pill pins, resource budget, and workflow. `install` copies the
executable; it is not a rerun of engine setup. Normal copy success is silent.

For a later rebuild, from the checkout use:

```bash
go test ./...
go vet ./...
go build -o ./build/ ./cmd/...
```

Copy the rebuilt executable again before the next authorized run.

## 2. Pause management after active jobs finish

```bash
/home/thelifeandtimes/repos/omabit/build/urbit-collector \
  --format text --runner workstation snapshot
```

Wait until there are no queued/running operations. Then press **Ctrl+C in the
`serve-fake` management terminal** and wait for it to exit. Leave the SSH listener
available for the final inventory check.

Existing ship containers continue running independently while management is
paused. Collector/panel management inventory will temporarily be unavailable.
At the last pre-run observation, alpha and panel-check were running/healthy,
at revisions 2/2 and 1/1 respectively. The qualification creates its own new pier.

If management remains active, qualification refuses with a management-lock error
before contacting Docker or creating a qualification run.

## 3. Run the disposable qualification

In the now-idle management terminal, or another desktop terminal:

```bash
sudo runuser -u urbit-runner -- /usr/bin/env -i \
  HOME=/var/lib/urbit-runner PATH=/usr/bin:/bin \
  /usr/local/lib/urbit-runner/checkpoints/6b/urbit-runner \
  qualify-meld --acknowledge-disposable-run --format text
```

The outer sudo authorizes switching accounts. The qualification itself runs as
the dedicated unprivileged account and discovers its UID/socket.

Expected progress is readable:

```text
Phase: engine
Phase: image
Phase: pill
Phase: probe
Phase: boot
Phase: meld
Phase: post-meld
Phase: cleanup
```

Allow several minutes for image/pill preparation and fresh boot. The pill download
is approximately 207 MiB. Meld and each HTTP-readiness phase have ten-minute
deadlines. Ctrl+C requests bounded cleanup; allow that cleanup to finish.

Expected successful final output includes:

- `Checkpoint: PASSED`, profile `offline-meld`, and the exact private report path.
- `Cleanup complete: true`, live limits observed, and pier metadata seen.
- Every phase marked `PASS`.
- `Meld outcome uncertain: false`, before/after logical byte counts, fresh
  admission observations, and start/finish timestamps.
- Four removed containers with resolved create outcomes: `probe`, `boot`, `meld`,
  and `post-meld`. Both runtime records show private HTTP reachability.
- `Production runtime qualified: false`.

The bytes are summed regular-file lengths, not allocated disk space, reclaimed
blocks, or loom utilization. A reduction is not required for success. The reported
meld interval includes utility lifecycle, removal, and post-measurement work.

## 4. Restore management and provide evidence

When qualification has exited with **complete cleanup** (or refused before
creating a run), restore management in its dedicated terminal:

```bash
sudo runuser -u urbit-runner -- /usr/bin/env -i \
  HOME=/var/lib/urbit-runner PATH=/usr/bin:/bin \
  /usr/local/lib/urbit-runner/checkpoints/6b/urbit-runner \
  serve-fake --acknowledge-development
```

Leave that command running. In a separate desktop terminal:

```bash
/home/thelifeandtimes/repos/omabit/build/urbit-collector \
  --format text --runner workstation snapshot
```

If the SSH listener also needs restarting, its command in another terminal is:

```bash
sudo /usr/bin/sshd -D -e -f /etc/urbit-runner/sshd_config
```

Send the final readable qualification result and restored management snapshot.
The private `report.json` remains the machine-readable evidence even when console
output uses `--format text`.

## Failure and review

An uncertain meld is never retried or followed by a restart. Container cleanup
does not clear the pier's uncertainty. Subsequent qualification commands refuse
that unresolved report before making Docker calls. Share the result for review;
do not edit the marker or reuse that pier.

If cleanup is incomplete, retain the exact resource names and report and request
review before further resource work. Cleanup resolves only owned containers; the
disposable pier/report are retained for investigation rather than auto-deleted.

Use the new executable for future qualification commands: older qualifier binaries
do not understand `offline-meld` reports. The management protocol remains at its
currently supported maintenance capabilities until managed meld is qualified.

## Developer evidence

Injected-runtime tests cover the exact utility command and same-pier sequence,
pre-dispatch persistence, fresh admission refusal, wait/exit disagreement, OOM,
accepted-start cancellation, missing post-utility metadata, failed final
persistence, failed post-meld readiness, management-lock exclusion, durable
quarantine after cleanup, and malformed/legacy report handling. They use temporary
files and simulated Docker responses, not real piers or runtime processes.

Local validation passed on September 25:

- `go test ./...`
- `go vet ./...`
- `go test -race ./internal/runtimequalify ./internal/managed ./cmd/urbit-runner`
- `go build -o ./build/ ./cmd/...` and the built executable's readable `meld-plan`.
- Linux/ARM64 cross-compilation of the runner verifies the unsupported-platform
  stub still builds; it does not qualify ARM64 runtime execution.
- `git diff --check`.

The live primitive gate is accepted above. Larger-pier admission policy, the
[managed meld live gate](13-managed-meld.md), loom telemetry/advice, schedules, and
epoch retention require subsequent evidence.
