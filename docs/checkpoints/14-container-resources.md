# Checkpoint 6D: container resource observations

**Live sample validation passed September 25, 2026.** Alpha and panel-check
returned fresh samples with advancing daemon timestamps, stable container IDs,
and the expected 4 GiB limits. Both remained healthy at their applied revisions;
the uninitialized instance had no invented resource sample. See the
[evidence record](../diagnostics/2026-09-25-container-resources.md).
The commands below remain a reproduction guide; this gate is complete.
The managed meld/replay/resume gate passed immediately before this work; alpha
was left running/healthy at revision 2/2 on port 18081.

## Measurements and policy

This milestone adds `container-resources` to Hello and `resources` to running
instance observations. The runner reads Docker's exact one-shot stats for the
authenticated container over its private engine socket:

- Raw cgroup memory usage in bytes, including charged cache/kernel memory.
- Observed memory limit, verified as the fixed 4 GiB development budget.
- Cgroup task count, including threads.
- Full container ID, daemon sample timestamp, and source `docker-stats-v1.47`.

The existing nominal ten-second poll performs a bounded three-second stats GET.
It reads at most 256 KiB and uses a fixed API path with `stream=false&one-shot=true`.
The collection does not start guest processes. The implementation preserves the
daemon's sample timestamp and treats missing/invalid counters as unavailable.

Complete samples are fresh for up to 30 seconds. Older samples, or timestamps
more than five seconds in the future, are stale. Failed reads clear cached
numeric values. A metric read failure does not change a successful HTTP health
observation into a runtime failure. Start/stop/maintenance transitions clear the
sample; runtime revision checks discard observations that arrive after a lifecycle
change.

The collector displays exact counters as MiB/task counts with status and sample
time. The JSON API retains integer byte counts. Raw memory usage can exceed
Docker CLI's displayed Linux usage because the CLI subtracts inactive cache.

**Container RAM is not loom utilization.** Loom capacity/open/free-space telemetry
and utilization-based advice are deferred under the official-release-only policy. This milestone
does not infer loom pressure from RAM, trigger compaction, or change scheduling.

## 1. Upgrade management — required user step

The binaries have been built. Check for active operations:

```bash
/home/thelifeandtimes/repos/omabit/build/urbit-collector \
  --format text --runner workstation snapshot
```

After active jobs finish, press **Ctrl+C in the current `serve-fake` terminal**
and wait for management to exit. Existing ship containers continue independently.

Copy the new executable:

```bash
sudo install -D -o root -g root -m 0755 \
  /home/thelifeandtimes/repos/omabit/build/urbit-runner \
  /usr/local/lib/urbit-runner/checkpoints/6d/urbit-runner
```

In the management terminal, start it and leave it running:

```bash
sudo runuser -u urbit-runner -- /usr/bin/env -i \
  HOME=/var/lib/urbit-runner PATH=/usr/bin:/bin \
  /usr/local/lib/urbit-runner/checkpoints/6d/urbit-runner \
  serve-fake --acknowledge-development
```

Keep the existing foreground SSH listener running. If it needs restarting, use a
separate terminal:

```bash
sudo /usr/bin/sshd -D -e -f /etc/urbit-runner/sshd_config
```

Then, in a desktop terminal:

```bash
/home/thelifeandtimes/repos/omabit/build/urbit-collector \
  --format text --runner workstation hello
```

Expected: the same runner identity and SSH operator, with **`container-resources`**
alongside `offline-pack` and `offline-meld`. Send the Hello result; the coding
agent can perform the remaining read-only sample validation programmatically.

## 2. Live sample validation

For manual inspection, after one polling cycle:

```bash
/home/thelifeandtimes/repos/omabit/build/urbit-collector \
  --format text --runner workstation snapshot
```

Running alpha and panel-check should show resource status `fresh`, a full container
ID, a recent sample timestamp, **4096.00 MiB** memory limit, and measured usage/task
counts. The uninitialized ssh-check should have no live container sample.

Observe at least two samples more than ten seconds apart. Expect sample timestamps
to advance while the same running container retains its ID. Numeric usage may
stay the same; that does not mean the timestamp failed to refresh. Existing ship
health and configuration/applied revisions should remain healthy and unchanged.

If readings are unavailable, retain the snapshot for investigation rather than
interpreting the absence as zero or restarting a healthy ship. Stale/failure and
late-poll behavior were exercised with controlled fixtures; live validation does
not require interrupting a healthy runtime to recreate those faults.

## Local validation and provenance

Passed local checks:

- `go test ./...`
- `go vet ./...`
- `go test -race ./internal/runtimequalify ./internal/managed ./cmd/urbit-collector ./cmd/urbit-runner`
- `go build -o ./build/ ./cmd/...`
- Linux/ARM64 runner cross-compilation (build-only, not runtime qualification).
- `git diff --check`

Tests include real temporary Unix HTTP sockets, exact fixed request routing,
proxy-environment isolation, redirect/status/oversize/deadline refusal, redacted
errors, container-ID binding, missing/overflow/partial counter rejection, preserved
daemon timestamps, independent health on metric failure, durable snapshot aging,
late-observation suppression, genuine numeric zero, and readable units/status.

Pinned engine source reviewed:

- [Stats API types](https://github.com/moby/moby/blob/6a43e3d5afddf4111da0f864bbc7cae5d7e95001/api/types/container/stats.go)
- [One-shot collection](https://github.com/moby/moby/blob/6a43e3d5afddf4111da0f864bbc7cae5d7e95001/daemon/stats.go)
- [Linux cgroup v2 counter mapping](https://github.com/moby/moby/blob/6a43e3d5afddf4111da0f864bbc7cae5d7e95001/daemon/stats_unix.go)
