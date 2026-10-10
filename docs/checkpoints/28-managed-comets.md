# Checkpoint 6R: managed comet creation and identity-pinned resume

The 6Q disposable comet gate passed. Managed registration/lifecycle and panel
**0.7.0** are implemented; Go/vet/race and model/QML checks pass. **Managed live
acceptance passed**; see the [evidence](../diagnostics/2026-09-29-managed-comet.md).
Desktop 0.7.0 and its bundled binaries are installed. Shell status IPC confirmed
the loaded version after restart. The previous snapshot is archived at
`/tmp/opencode/omabit-panel-before-6r-20260929.tar`.

## Mode and identity boundaries

The legacy `serve-fake` endpoint remains fake-only. The new
`serve-development --acknowledge-development` endpoint uses mode
`development-managed` and advertises `create-comet`. Existing fake/imported-fake
registrations remain in the same store with the same runner ID.

Collectors do not silently accept this broader mode. `runner-accept-mode` verifies
the same pinned runner ID at the same enrolled endpoint, checks the selected
connection revision and the explicit target mode, then atomically updates only
the mode/revision. Normal `runner-edit` still rejects unacknowledged mode changes.
Other independently enrolled collectors must make their own explicit transition.

`create-comet INSTANCE --http-port PORT --request-id ID --confirm-network` requires
operator access, the new mode/capability and an identity-pinned runner selection.
Registration is durable/idempotent and leaves an uninitialized stopped instance
with no invented ship name. Its first explicit Start creates a new identity inside
the private pier with the pinned image/pill, outbound NAT networking and loopback
HTTP. It accepts no caller-provided identity or key and publishes no host UDP port.

The first Start journals uncertainty **before** creation dispatch. Public identity
and initialized state commit after inspected HTTP readiness and private metadata
checks. Later resumes omit bootstrap flags and require the same public identity.
Observation cannot adopt an interrupted initial boot, and Stop cannot clear its
quarantine. A changed identity also quarantines the managed comet.

Resource telemetry and operator login-code access support the comet container
profile. Pack/meld/roll/epoch operations on comets are refused until separately
qualified. Fake maintenance behavior retains its accepted contracts.

## Compatibility

The first comet registration advances private SQLite metadata to **version 3**.
Afterward use **6R or newer with `serve-development`** for management. Older
binaries reject the schema instead of interpreting comet data as fake data. The
new binary also refuses fake-only startup when comet registrations exist.

Comets use a separate managed-comet Docker name/label namespace; existing managed
fake names/labels are preserved. The public RPC protocol remains version 1 with
explicit capability negotiation.

## Upgrade and live gate

1. Press **Ctrl+C** in the management terminal. Runtime containers continue
   independently; keep the SSH listener running.
2. Install the rebuilt runner:

   ```bash
   sudo /usr/bin/install -D -o root -g root -m 0755 \
     /home/thelifeandtimes/repos/omabit/build/urbit-runner \
     /usr/local/lib/urbit-runner/checkpoints/6r/urbit-runner
   ```

3. Start the **new command**, then leave it running:

   ```bash
   sudo runuser -u urbit-runner -- /usr/bin/env -i \
     HOME=/var/lib/urbit-runner PATH=/usr/bin:/bin \
     /usr/local/lib/urbit-runner/checkpoints/6r/urbit-runner \
     serve-development --acknowledge-development
   ```

The collector initially refuses the changed mode. The agent will use the new
collector to explicitly accept `development-managed` for the already pinned ID
and current enrollment revision, then refresh desktop observations. The same
operation is available in Setup → Manage runners after a mode mismatch.

The live gate will register a fresh `comet-check`, verify registration replay,
start it, record its internally generated public name, and test stop/replacement
resume and exact operation replay. Fresh starts require at least 5 GiB available
RAM and 8 GiB free disk, with a twenty-minute initial readiness bound. There is no
standalone comet qualification rerun and no adoption of the retained 6Q comet.

Key provisioning, external real-pier import, host Ames/tailnet reachability and
routine maintenance remain subsequent backend work.
