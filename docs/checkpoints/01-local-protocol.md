# Checkpoint 1: local runner/collector protocol

This is the first manual acceptance gate. Allow about 10 minutes and use two
terminals, with an optional third for watching updates.

## Scope and safety

This runs an explicitly simulated runner. It uses no Docker, Vere, systemd
services, network ports, keys, login codes, or piers. It installs nothing. Its two
instances are synthetic, and all state disappears when the runner exits.

This checks client/runner behavior, not production readiness. SSH, real access-key
attribution, persistence, pill fetching/booting, and actual runtime operations are
not implemented yet. Two CLI processes here demonstrate independent clients of
one local account, not two authenticated operators.

Requirements: Linux, Go 1.24 or later, and Bash. `curl` is optional for the protocol
mismatch check. No third-party Go dependencies. Run repository commands from the
repository root in each terminal.

## Verification record

September 22, 2026, Linux x86-64, Go 1.27.0:

- `go test ./...` and `go vet ./...` passed.
- `go test -race -shuffle=on -count=10 ./...` passed.
- Built executables passed this sequence with a 2-second operation delay,
  including simultaneous work, exact replay, configuration/request conflicts,
  protocol mismatch, watch cancellation, shutdown, and fresh-state restart.
- All validation processes were stopped; no production resources were touched.

This records developer validation. User acceptance of the steps below is still
pending. The declared Go minimum has not yet been independently tested with a
Go 1.24 toolchain.

## 1. Build and test

```bash
go test ./...
go vet ./...
go build -o ./build/ ./cmd/...
```

Expected: tests pass; `build/urbit-runner` and `build/urbit-collector` are produced.
The build directory is ignored by Git. For extra concurrency checking:

```bash
go test -race ./...
```

## 2. Start the simulator (terminal A)

```bash
DEMO_DIR="$(mktemp -d "${XDG_RUNTIME_DIR:-/tmp}/urbit-demo.XXXXXX")"
./build/urbit-runner demo --socket "$DEMO_DIR/runner.sock" --operation-delay 10s
```

Expected: a `SIMULATION ONLY` message with the socket's complete path. Leave this
running. The longer operation delay makes conflicts easier to observe.

The private directory is required. The runner refuses a public directory or an
existing socket/file rather than changing or replacing it.

## 3. Connect (terminal B)

Set `SOCKET` to the actual path printed in terminal A, for example:

```bash
SOCKET=/run/user/1000/urbit-demo.ABC123/runner.sock
./build/urbit-collector --socket "$SOCKET" hello
./build/urbit-collector --socket "$SOCKET" snapshot
```

Expected:

- `protocol_version` is `1` and `mode` is `simulation`.
- Two instances: `alpha` and `bravo`, both at configuration revision `1`.
- Alpha is `failed` with one active synthetic `unexpected_exit` incident.
- Bravo is `stopped`; operation history is empty.
- The runner ID is consistent between commands.

Do not supply a real ship name or pier path. The only instances are `alpha` and
`bravo`.

## 4. Optional live observer (terminal C)

Set the same `SOCKET` value in this terminal, then run:

```bash
./build/urbit-collector --socket "$SOCKET" watch --interval 1s
```

Expected: one compact JSON snapshot per line. This is polling, not an event
stream. Leave it running while performing the following steps in terminal B.
Ctrl+C stops the watcher without stopping the runner.

## 5. Stage configuration and reject a stale edit

In terminal B:

```bash
./build/urbit-collector --socket "$SOCKET" configure alpha --revision 1 --http-port 8080 --ames-port 34343 --loom 32
```

Expected: `config_revision: 2`, `applied_revision: 1`, and `loom: 32`. This stages
configuration; it does not start the instance.

Now simulate a second operator's stale form:

```bash
./build/urbit-collector --socket "$SOCKET" configure alpha --revision 1 --http-port 8080 --ames-port 34343 --loom 33
```

Expected: nonzero exit and JSON error `revision_conflict`, with
`current_revision: 2`. A fresh snapshot must still show `loom: 32`. The collector
must not silently retry with a newer revision.

Expected errors below exit `1`; do not run the walkthrough under `set -e` unless
you explicitly handle those expected failures.

## 6. Submit, replay, and conflict

Run these three commands together in terminal B so they occur within the
10-second simulated operation:

```bash
./build/urbit-collector --socket "$SOCKET" operate alpha start --revision 2 --request-id walkthrough-start
./build/urbit-collector --socket "$SOCKET" operate alpha start --revision 2 --request-id walkthrough-start
./build/urbit-collector --socket "$SOCKET" operate alpha stop --revision 2 --request-id walkthrough-too-soon
```

Expected:

- The first response contains an operation ID and `status: running`.
- The second response has the **same** operation ID, not a second job.
- The third returns `instance_busy` while the start is running.
- A snapshot/watch shows `desired_state: running` and `observed_state: starting`.

If the third command succeeds because more than 10 seconds elapsed, let the stop
finish, then repeat with a new start request ID to observe the busy check. Do not
reuse an accepted request ID for a different logical action.

You can also start bravo while alpha is starting; they are independent instances:

```bash
./build/urbit-collector --socket "$SOCKET" operate bravo start --revision 1 --request-id walkthrough-bravo
```

## 7. Observe completion from a fresh client

The submitting CLI processes have already exited. Wait at least 10 seconds from
submission, then use a new command (or a new terminal with the same `SOCKET`):

```bash
./build/urbit-collector --socket "$SOCKET" snapshot
./build/urbit-collector --socket "$SOCKET" operate alpha start --revision 2 --request-id walkthrough-start
```

Expected, assuming the busy test was performed within the operation window:

- Alpha is `running`, with `applied_revision: 2` and no active operation ID.
- Its operation is `succeeded` and has a completion timestamp.
- The synthetic alpha incident remains visible with `active: false`.
- Replaying the original request still returns the same completed operation.
- The operation actor is `local-uid:<your UID>`; this is demo-local attribution.

Now attempt to reuse that request ID for different input:

```bash
./build/urbit-collector --socket "$SOCKET" operate alpha stop --revision 2 --request-id walkthrough-start
```

Expected: `request_conflict`, not a stopped instance.

## 8. Reject an incompatible protocol (optional curl check)

```bash
curl --silent --show-error --include --unix-socket "$SOCKET" -H 'Urbit-Runner-Protocol: 2' http://runner/v1/snapshot
```

Expected: HTTP `426` and JSON `protocol_mismatch`. This connects only to the Unix
socket, not an external host. Note that curl without `--fail` does not itself
return a failing exit status for HTTP 426.

## 9. Stop, restart, and inspect the explicit demo limitation

Ctrl+C in terminal A stops the runner and removes its socket. Any watcher exits
with a connection error instead of presenting old state as current.

Restart in terminal A using the same command:

```bash
./build/urbit-runner demo --socket "$DEMO_DIR/runner.sock" --operation-delay 10s
```

Request a snapshot from terminal B. Expected: a **new runner ID**, initial
revisions/states, and empty operation history. This reset is intentional for the
simulator. The real runner's durable recovery is a later checkpoint.

## Cleanup and feedback

Stop the runner and watcher with Ctrl+C. In terminal A:

```bash
rmdir "$DEMO_DIR"
```

No process, installed service, or real ship should remain from this walkthrough.
Build binaries can remain for development. If the runner was forcibly killed and
a stale socket remains, confirm no runner is using that exact path before
removing the socket; the runner intentionally will not do that for you.

Report which steps passed and any differences in output. In particular, confirm
the stale edit, same-ID replay, busy-operation rejection, completion after the
submitting CLI exits, and explicit reset after runner restart. We will pause at
this gate before host setup or runtime qualification.
