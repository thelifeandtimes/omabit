# Checkpoint 6C: managed offline meld

**Live upgrade/meld/replay/resume passed September 25, 2026.** Alpha's managed
meld completed in **20.59 s** and resume in **5.94 s**. Exact replay returned the
same completed operation both while stopped and after resume. Alpha was healthy
at revision **2/2**, and direct HTTP returned `~zod`. See the
[acceptance evidence](../diagnostics/2026-09-25-managed-meld.md).
The commands below remain a reproduction guide; this acceptance run is complete.
The prerequisite [disposable meld gate](12-offline-meld.md) passed with run
`9b5dfcb57a677f7cf1a2b833e6f65274` on September 25.

## Implemented behavior

- `operate INSTANCE meld --revision N --request-id ID`, gated by `offline-meld`.
- Cleanly stopped, initialized registered development piers only.
- Same pinned image, direct offline utility, loom 31, and existing pier.
- 4 GiB container RAM, no additional swap, two CPUs, 256 PIDs. At least 5 GiB
  available physical RAM and 8 GiB disk at admission, rechecked after preflight
  immediately before utility dispatch. Pack now gets this fresh recheck too.
- One expensive start/pack/meld worker at a time per host; revision checks and
  per-instance exclusion apply throughout. Configuration cannot change during work.
- Success leaves the pier stopped and preserves its applied configuration revision.
- Actor-scoped exact replay returns the original operation, including its timestamps,
  even after resume or management restart. Reusing the ID for different input fails.
- Before/after logical file sizes, maintenance timestamps, and admission observations
  are exposed in snapshots and readable collector output.
- Failure/interruption leaves persistent uncertainty. Observation, management
  restart, and explicit stop do not clear the affected pier's quarantine.
- Retained/uncertain utility containers also block subsequent expensive workers
  across instances and management restart. Such an admission failure records a
  failed operation without running the backend. Explicit cleanup stays available.

This is bounded development-fake support using the qualified budget. Larger-pier
memory estimates, scheduling, loom advice, and epoch retention have separate gates.

## 1. Install and restart management — user intervention

The coding agent has built the binaries. From any desktop terminal:

```bash
/home/thelifeandtimes/repos/omabit/build/urbit-collector \
  --format text --runner workstation snapshot
```

After all active operations finish, press **Ctrl+C in the current `serve-fake`
management terminal** and wait for it to exit. Existing ship containers continue
running independently. Keep the foreground SSH listener available.

Copy the executable:

```bash
sudo install -D -o root -g root -m 0755 \
  /home/thelifeandtimes/repos/omabit/build/urbit-runner \
  /usr/local/lib/urbit-runner/checkpoints/6c/urbit-runner
```

In the management terminal, start it and leave it running:

```bash
sudo runuser -u urbit-runner -- /usr/bin/env -i \
  HOME=/var/lib/urbit-runner PATH=/usr/bin:/bin \
  /usr/local/lib/urbit-runner/checkpoints/6c/urbit-runner \
  serve-fake --acknowledge-development
```

In another desktop terminal:

```bash
/home/thelifeandtimes/repos/omabit/build/urbit-collector \
  --format text --runner workstation hello
```

Expected: the existing runner identity
`runner-f496feed1f7b08f31a5d82b94ac2f5fd`, the same SSH operator, and features
including **`offline-meld`** and **`offline-pack`**. Send that output to the coding
agent; the remaining bounded fake-ship work can be performed programmatically.

If the SSH listener also needs restarting, its command in a separate terminal is:

```bash
sudo /usr/bin/sshd -D -e -f /etc/urbit-runner/sshd_config
```

Use 6C or newer for subsequent management starts: older executables do not
understand persisted managed-meld container references.

## 2. Live validation sequence

The intended target is the existing development fake **alpha**, last observed
running/healthy at configuration/applied revision **2/2**, HTTP port **18081**.
Inspect fresh inventory first and retain the revision from that observation.

The following is the reproduction sequence for the agent or operator. Define this
helper in each terminal where it is used; it pins the expected runner identity:

```bash
collect() {
  /home/thelifeandtimes/repos/omabit/build/urbit-collector \
    --format text --runner workstation \
    --expected-runner-id runner-f496feed1f7b08f31a5d82b94ac2f5fd "$@"
}
collect snapshot
```

For alpha still at revision 2, explicitly stop it:

```bash
collect operate alpha stop --revision 2 --request-id checkpoint-6c-stop-alpha
collect watch --interval 2s
```

Wait for succeeded/stopped with no recovery requirement, then Ctrl+C **the watch**.
Submit meld:

```bash
collect operate alpha meld --revision 2 --request-id checkpoint-6c-meld-alpha
collect watch --interval 2s
```

Wait for meld to succeed with alpha still stopped. Ctrl+C the watch, replay the
exact request, then resume:

```bash
collect operate alpha meld --revision 2 --request-id checkpoint-6c-meld-alpha
collect operate alpha start --revision 2 --request-id checkpoint-6c-resume-alpha
collect watch --interval 2s
```

Wait for healthy/running at revision 2/2, then Ctrl+C the watch. Replay while the
pier is running and obtain the final evidence:

```bash
collect operate alpha meld --revision 2 --request-id checkpoint-6c-meld-alpha
collect snapshot
curl --noproxy '*' --fail --silent --show-error --max-time 5 \
  http://127.0.0.1:18081/~/host
```

Expected: both replays return the same succeeded operation ID and timestamps;
the maintenance report has kind `meld` and completed measurements; alpha is
healthy on the same initialized pier; `/~/host` returns `~zod`. A reduction in
logical bytes is not required. This sequence uses fixed request IDs for exact
reconciliation; it does not automatically repeat maintenance.

If a step fails, inspect its existing request and snapshot before proceeding. An
uncertain utility's owned resources can require explicit stop, but that cleanup
does not authorize resuming or repacking/remelding the affected pier.

## Local evidence

Passed:

- `go test ./...`
- `go vet ./...`
- `go test -race ./internal/runtimequalify ./internal/managed ./cmd/urbit-collector ./cmd/urbit-runner`
- `go build -o ./build/ ./cmd/...`
- `git diff --check`

Driver fixtures cover the qualified utility invocation, persistence before dispatch,
fresh admission refusal, clean stop/meld/resume, nonzero/OOM outcomes, interrupted
wait, failed journal writes, and quarantine after cleanup. Real SQLite and Unix RPC
tests cover capabilities, revision and state refusal, busy conflicts, cross-kind
host serialization, report persistence, exact replay after resume/management
restart, and retained-resource admission across restart. Collector tests verify
capability refusal before mutation and truthful human-readable measurements.

These local checks use an injected runtime. The independent managed live evidence
is recorded above; live destructive interruption/recovery qualification remains a
separate gate.
