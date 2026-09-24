# Checkpoint 4A: collector enrollment and multi-runner inventory

Prerequisite: the exercised checkpoint 3A lifecycle is accepted, including the
probe timing fix. This checkpoint adds a persistent collector connection registry
and verifies routing, identity continuity, and independently polled inventory.
Implementation and automated checks are complete; live acceptance is pending.

The first part observes the existing healthy managed runner. The second part uses
two memory-only simulators for connection loss and identity replacement. All
connections are private Unix sockets; SSH grants/key attribution and Eyre sessions
have their own subsequent phase-4 gates.

## 1. Copy the updated collector

From the checkout:

```bash
go test ./...
go vet ./...
go build -o ./build/ ./cmd/...

sudo install -D -o root -g root -m 0755 \
  ./build/urbit-collector \
  /usr/local/lib/urbit-runner/checkpoints/4a/urbit-collector
```

Keep the current `serve-fake` terminal running. This checkpoint is compatible
with its existing binary and database. The copy above installs only the new
collector executable at a separate checkpoint path.

## 2. Enroll the existing managed runner

In your collector terminal, define a new helper without the old global `--socket`:

```bash
SOCKET="/run/user/$(id -u urbit-runner)/urbit-runner/runner.sock"

collector4() {
  sudo runuser -u urbit-runner -- /usr/bin/env -i \
    HOME=/var/lib/urbit-runner PATH=/usr/bin:/bin \
    /usr/local/lib/urbit-runner/checkpoints/4a/urbit-collector "$@"
}

collector4 enroll workstation --socket "$SOCKET"
collector4 runners
collector4 --runner workstation snapshot
collector4 inventory
```

Expected:

- Enrollment records `workstation`, `transport: unix`, mode
  `development-managed-fake`, and the existing runner ID
  `runner-f496feed1f7b08f31a5d82b94ac2f5fd` on this development host.
- The named snapshot shows the same alpha at configuration/applied revision 2,
  running/healthy at port 18081, with retained operation history.
- Inventory contains that snapshot under `name: workstation`, with
  `status: available` and `complete: true`.

The registry is separate collector state at
`/var/lib/urbit-runner/.config/urbit-collector/runners.json` in this temporary
same-account walkthrough. A normal per-user collector uses that user's XDG
config location. The dedicated-account helper is still needed for access to the
private local management socket.

Repeat enrollment and watch:

```bash
collector4 enroll workstation --socket "$SOCKET"
collector4 inventory --watch --interval 1s
```

The enrollment timestamp and pin must stay the same. Watch emits one compact
observation per completed poll. Ctrl+C stops only the collector watcher. Running
`collector4 runners` again proves saved enrollment survives CLI process exit.

## 3. Prepare isolated multi-runner fixtures

Use your ordinary desktop account for this section. These simulator processes
consume no ship memory, invoke no Docker, and use no existing managed state.

In the collector terminal, from the checkout:

```bash
LAB="$(mktemp -d /tmp/omabit-collector-4a.XXXXXX)"
install -d -m 0700 "$LAB/a" "$LAB/b"
declare -p LAB

lab() {
  ./build/urbit-collector --config-dir "$LAB/collector-one" "$@"
}

lab2() {
  ./build/urbit-collector --config-dir "$LAB/collector-two" "$@"
}
```

Copy the printed `LAB=...` value into two additional terminals, also in the
checkout. In fixture terminal A:

```bash
# Set LAB to the exact directory printed above first.
./build/urbit-runner demo --socket "$LAB/a/runner.sock"
```

In fixture terminal B:

```bash
# Set LAB to the same printed directory first.
./build/urbit-runner demo --socket "$LAB/b/runner.sock"
```

Keep both fixture terminals open. In the collector terminal:

```bash
lab enroll fixture-a --socket "$LAB/a/runner.sock"
lab enroll fixture-b --socket "$LAB/b/runner.sock"
lab2 enroll fixture-a --socket "$LAB/a/runner.sock"
lab inventory
```

Expected: two different runner IDs, both mode `simulation`, each with its own
synthetic alpha/bravo. The simulator's initial synthetic alpha failure incident is
expected. Identical instance names must remain nested under their own runner.

## 4. Named routing and independent collector conflicts

```bash
lab --runner fixture-a configure alpha --revision 1 --http-port 19080 --ames-port 34343 --loom 31
lab2 --runner fixture-a configure alpha --revision 1 --http-port 19081 --ames-port 34343 --loom 31
lab inventory
```

The first command stages revision 2 on fixture-a only. The second must fail with
`revision_conflict`, current revision 2, without retrying. Fixture-b's alpha must
still have revision 1 and HTTP port 8080. This checks independent collector state,
not distinct authenticated SSH operators; both are local-account clients here.

## 5. Unavailable connection isolation

```bash
lab inventory --watch --interval 1s
```

Stop **only fixture terminal B's simulator** with Ctrl+C. The watcher must keep
emitting available fixture-a snapshots and report fixture-b as `unavailable`,
with an error and no current snapshot. It must not invent a ship-crash incident.

Ctrl+C the watcher, then:

```bash
lab inventory
```

Expected: both connection entries are present, `complete: false`, exit status 1.
Partial inventory is intentional JSON output on stdout, not a fatal loss of the
healthy runner's result.

## 6. Identity replacement is refused

Restart the simulator in fixture terminal B with the same command/socket path.
A simulator generates a new runner ID on every launch, deliberately exercising
replacement rather than persistent managed-runner recovery.

```bash
lab inventory
lab --runner fixture-b operate alpha start --revision 1 --request-id must-not-run
lab runners
```

Expected: fixture-b reports `identity_mismatch`; the operation fails with
`runner_identity_mismatch` before sending the mutation. The registry keeps the
original pin. Fixture-a remains available. Copy fixture-b's **old** runner ID from
`lab runners`, then explicitly forget and re-enroll the disposable fixture:

```bash
lab forget fixture-b --runner-id OLD_RUNNER_ID_FROM_RUNNERS
lab enroll fixture-b --socket "$LAB/b/runner.sock"
lab inventory
```

Expected: inventory is complete again, with fixture-b's new ID. Its operation
history is empty, proving the refused request did not start a simulated operation.
Forgetting removes only collector enrollment and does not send a runner mutation.

## Cleanup and feedback

Stop both simulator terminals and any inventory watcher with Ctrl+C. The small
`$LAB` fixture directories can remain for inspection. Keep the real workstation
enrollment for the next phase. The actual managed alpha lifecycle follows the
existing checkpoint 3A explicit-stop procedure whenever you finish using it.

Please share `collector4 inventory` and whether fixture routing, the stale revision
rejection, unavailable-host isolation, and replacement refusal behaved as above.
Automated tests cover stalled polls, reconnect recovery, pinning after a new dial,
concurrent registry writers, invalid/unsafe registry files, and cancellation.
Live two-host SSH acceptance follows in checkpoint 4B.
