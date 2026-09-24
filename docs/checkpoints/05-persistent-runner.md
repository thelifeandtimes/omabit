# Checkpoint 3A: persistent fake-ship runner and collector

Prerequisites: checkpoints 2A, 2B, and 2C have passed. This checkpoint turns the
qualified fake-ship primitives into a persistent local management workflow.
Implementation and fixture validation are complete; live acceptance is pending.

## Scope and differences from qualification

- `urbit-runner serve-fake` serves the management protocol on a private Unix socket.
- `urbit-collector create-fake` registers an instance. It does not boot it.
- `operate ... start` bootstraps a new fake ship on first use, or resumes Vere
  against its existing registered pier afterward. `operate ... stop` stops/removes
  its owned containers while preserving the pier, registration, and network.
- SQLite records the stable runner identity, configuration revisions, operator
  attribution, resource intents, operation history, and idempotency keys.
- The management process and ships have separate lifetimes: **stopping the runner
  process does not stop a running ship**. Stop each instance explicitly when done.
- Existing qualification piers are never adopted. New instances live under a
  separate private `managed` directory and use different Docker resource labels.

This is development-only and fake-only. Every ship is a fresh fake `~zod`, with
Ames disabled. Supported settings are a loopback HTTP host port (1024..65535),
loom 31, and the reserved Ames value 34343, which is not published. Images and
bootstrap pills remain pinned to the accepted candidate. Real keys, pier imports,
meld, control-socket metrics, SSH access grants, and production hosting are not
available in this milestone.

Starts use a private per-instance NAT bridge with normal outbound access and
only the configured `127.0.0.1` HTTP publication. Container limits remain 4 GiB
RAM, no extra swap, two CPUs, and 256 PIDs. Starts are admitted one at a time and
require 5 GiB available RAM and 8 GiB disk. This is a capacity check, not a host
resource reservation. At most eight instances may be registered in this prototype.

## Persistence and interruption policy

An initialized Urbit continues its persisted state when Vere resumes. This is not
a conventional guest-OS reboot and does not reapply the bootstrap pill. The CLI
keeps the host-oriented `start`/`stop` names; see the architecture's lifecycle
definitions. The September 24 [resume-delay investigation](../diagnostics/2026-09-24-resume-delay.md)
identified a 120-second preflight-probe delay before Vere was launched, not a
second initialization of the ship.

State lives under:

```text
/var/lib/urbit-runner/.local/share/urbit-runner/managed/
  runner.sqlite                 # plus SQLite WAL/SHM files
  <opaque-instance-id>/data/     # a separately registered fake pier
```

The database uses WAL and full synchronous commits. Intent is committed before
Docker mutations. Existing/unknown database schemas, unsafe files, and conflicting
state writers are refused rather than reset.

Client disconnects do not cancel accepted jobs. Exact request-ID replays return
the same operation, including after a runner restart. Keys are scoped to the
authenticated local actor. The snapshot includes the latest 128 operations;
older idempotency records remain in the database.

Interrupted operations are recorded as `interrupted`, never automatically rerun.
Read-only observation reconciles current runtime state. Pure preparation may be
retried where safe, but an unproven partial first-boot pier is never blindly
bootstrapped again. A known owned container can be explicitly stopped for review.
If a resource outcome is uncertain, do not delete state or rename resources to
force a retry.

## 1. Build and copy both binaries

Development builds now require **Go 1.25+** (the inspected Go 1.27 is sufficient).
The pure-Go SQLite driver is compiled into the binaries; no host SQLite package
or database server is required. The first build may download Go modules.

```bash
go test ./...
go vet ./...
go build -o ./build/ ./cmd/...

sudo install -D -o root -g root -m 0755 \
  ./build/urbit-runner \
  /usr/local/lib/urbit-runner/checkpoints/3a/urbit-runner

sudo install -D -o root -g root -m 0755 \
  ./build/urbit-collector \
  /usr/local/lib/urbit-runner/checkpoints/3a/urbit-collector
```

These `install` commands copy files; they do not reinstall Docker or create a
service. Do not replace the runner binary during an active operation.

## 2. Start the runner (terminal A)

```bash
sudo runuser -u urbit-runner -- /usr/bin/env -i \
  HOME=/var/lib/urbit-runner PATH=/usr/bin:/bin \
  /usr/local/lib/urbit-runner/checkpoints/3a/urbit-runner \
  serve-fake --acknowledge-development
```

Expected: a startup JSON line with `mode: development-managed-fake`, the state
directory, and a socket under `/run/user/<uid>/urbit-runner/runner.sock`. Keep this
terminal open. Another runner using the same state directory must be refused.

The daemon remains foreground for this checkpoint. Unattended service packaging
is separate; the already-installed rootless Docker service remains unchanged.

## 3. Set up the collector helper (terminal B)

The management socket is private to the service account. Until local/SSH operator
enrollment is implemented, invoke the collector under that account too:

```bash
SOCKET="/run/user/$(id -u urbit-runner)/urbit-runner/runner.sock"

collector() {
  sudo runuser -u urbit-runner -- /usr/bin/env -i \
    HOME=/var/lib/urbit-runner PATH=/usr/bin:/bin \
    /usr/local/lib/urbit-runner/checkpoints/3a/urbit-collector \
    --socket "$SOCKET" "$@"
}

collector hello
collector snapshot
```

Initially, expect an empty collection, a stable `runner_id`, and the mode above.
Operations identify the actual Unix peer as `local-uid:<uid>`. This validates
kernel-derived local attribution, not different human operators or SSH keys.
Do not add your desktop account to the Docker group or broaden socket permissions.

## 4. Register and start a fake instance

Choose an unused high host port. This example uses 18080:

```bash
collector create-fake alpha --http-port 18080 --request-id register-alpha --label "Workstation test"
collector snapshot
collector operate alpha start --revision 1 --request-id start-alpha
```

Registration returns a completed `register-fake` operation; the instance is
initially `uninitialized`. Start returns a queued/running operation immediately.
Its first boot downloads/verifies the pinned pill and uses the qualified image.
The submitting collector may exit while work continues.

Observe from terminal B, or another terminal with the same collector helper:

```bash
collector watch --interval 1s
```

Wait for the start operation to become `succeeded`, with the instance `running`,
health `healthy`, applied revision 1, and endpoint `http://127.0.0.1:18080`.
Ctrl+C stops the watcher only. If the operation fails, stop and share the snapshot;
do not force bootstrap over a partial pier.

Repeat the same request:

```bash
collector operate alpha start --revision 1 --request-id start-alpha
```

Expected: the same operation ID, not another boot. An ordinary desktop request
can also check the public endpoint while the ship remains running:

```bash
curl --noproxy '*' --fail --max-time 5 http://127.0.0.1:18080/~/host
```

Expected: `~zod`. This is public identity, not authenticated application health.

## 5. Restart management without stopping the ship

Only after the start operation has completed, press Ctrl+C in terminal A.
The ship should continue answering the public endpoint above. Restart the runner
using the same `serve-fake` command, then:

```bash
collector hello
collector snapshot
collector operate alpha start --revision 1 --request-id start-alpha
```

Expected: the same runner ID, instance, and original operation ID. The operation
must not execute again. Allow a short interval for a new `observed_at` timestamp;
observations older than 30 seconds have their displayed health marked unknown.

A stale socket left by an abrupt runner death can be removed on startup only
while holding the exclusive state lock, with verified ownership/type and a
confirmed refused connection. Live listeners or unrelated files are never replaced.

## 6. Stage a port change and verify conflict reporting

```bash
collector configure alpha --revision 1 --http-port 18081 --ames-port 34343 --loom 31
collector configure alpha --revision 1 --http-port 18082 --ames-port 34343 --loom 31
```

The first succeeds at revision 2 while the existing ship remains on 18080 with
applied revision 1. The second must return `revision_conflict` and current revision
2. Successful configuration changes are attributed and journaled as well.

Apply the new setting through explicit stop/start:

```bash
collector operate alpha stop --revision 2 --request-id stop-alpha
collector watch --interval 1s
```

Wait for stop success and `stopped`, then stop the watcher and run:

```bash
collector operate alpha start --revision 2 --request-id restart-alpha
collector watch --interval 1s
```

Expected: the same registered pier restarts, applied revision becomes 2, and the
endpoint becomes `http://127.0.0.1:18081`. Restart uses the verified cached image
when available and does not re-download/reapply the bootstrap pill.

## 7. Optional second registration and end-of-session stop

```bash
collector create-fake bravo --http-port 18082 --request-id register-bravo
collector snapshot
```

This adds a second independent registry entry without consuming a running ship's
memory. Starting it is optional and subject to the same capacity checks.

When done, explicitly stop every running instance before stopping management:

```bash
collector operate alpha stop --revision 2 --request-id final-stop-alpha
collector watch --interval 1s
```

Wait for completion, stop the watcher, then Ctrl+C terminal A. Registrations,
database, image cache, piers, and empty per-instance networks remain for the next
session. This checkpoint has no destructive delete/unregister command. Do not
remove the state database to get a fresh start over existing runtime resources.

## Acceptance feedback

Report whether registration/start, request replay, runner restart, fresh
observation, revision conflict, port application, and final explicit stop worked.
Share snapshots/errors, not keys, login codes, or raw pier contents. If a job is
interrupted or recovery is required, share the snapshot before taking further
actions. Neither the runner nor this checkpoint claims full runtime qualification.

Developer tests use real SQLite files and Unix peer credentials with an injected
runtime, including a killed subprocess with committed and uncommitted state.
The live managed Docker workflow remains the purpose of this manual gate.
