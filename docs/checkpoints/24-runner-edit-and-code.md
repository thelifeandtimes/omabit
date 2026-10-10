# Checkpoint 6N: connection editing and secure login-code access

The first backend increment following the accepted 0.4 desktop is implemented in
panel/collector **0.5.0** and the **6N runner**. Automated checks pass. The dedicated
runner upgrade and live code/Eyre validation passed September 28 UTC. Actual
desktop clipboard publication/expiry remains a follow-up; the existing clipboard
selection was preserved, and provider cancellation was tested with an isolated helper.
Pier import, key/comet provisioning, Ames networking and routine scheduling remain
the subsequent backend increments; this checkpoint does not claim those complete.
See the [validation record](../diagnostics/2026-09-28-runner-edit-and-code.md).

## Connection editing

`runner-edit NAME --runner-id ID --revision N --name NEW_NAME` accepts either
`--socket /absolute/path` or `--ssh HOST_ALIAS [--ssh-config /absolute/path]`.
The proposed endpoint must verify the original runner ID and mode. Verification
runs outside the registry lock; the final write rechecks revision, original
enrollment timestamp, endpoint and collisions under the lock. A no-op preserves
the revision. Desktop preferences remain keyed by runner ID through a rename.

Legacy version-1 connections without revisions read as revision 1 without changing
the file. Updates persist revision metadata; use the new collector after editing
because older strict registry readers reject the added field rather than guess.
Setup → Manage runners → Edit display & host → Edit pinned connection exposes the
same operation without a forget/re-enroll window or automatic retries.

## Login-code path

- Operator-only `POST /v1/instances/INSTANCE/login-code`, advertised as `login-code`.
- Request binds configuration revision and the selected full container ID. The
  instance must be freshly observed running, without active work or recovery state.
- The mutation mutex excludes a simultaneous lifecycle/configuration change. The
  driver rechecks private engine ownership, registered container, image, mounts,
  network and running state before its fixed command.
- Fetch uses `/usr/bin/curl --disable` against **container-private**
  `http://127.0.0.1:12321` with the fixed Dojo expression `+code`. No caller-selected
  expression, URL, shell or Docker arguments. Proxy, redirect and curlrc behavior
  are excluded; time/output are bounded. The Unix control socket stays disabled.
- The runner returns a no-store secret response only after recording an attributed
  outcome. Codes never enter instance state, operation bodies, snapshots or errors.
- Collector clients validate response runner, instance, revision and container.
  CLI stdout disclosure requires explicit `login-code ... --reveal`.
- `omabit copy-code` fetches directly and sends the code to foreground
  `wl-copy --sensitive --type text/plain` through stdin. It owns the provider for at
  most 30 seconds and terminates only that provider, never clears a newer clipboard.
  Linux parent-death handling also terminates the child if the helper disappears.
  QML receives status metadata only. Sensitive history handling is advisory; the
  installed Omarchy capture helper explicitly honors the sensitive hint.

Pinned-source references:

- <https://github.com/urbit/vere/blob/5a7614f/docker/get_urbit_code.sh>
- <https://github.com/urbit/vere/blob/5a7614f/pkg/vere/io/http.c>

## Checks

Registry/CLI tests cover legacy revision reads, rename/rebind, wrong identity,
stale revisions, name collisions, concurrent writers and no-op behavior. Runtime
and RPC tests cover bound selection, fixed command, malformed/oversized/error
redaction, observer refusal and attributed secret-free history. A fake clipboard
provider checks secret stdin and exact non-secret arguments. QML/model fixtures
exercise code routing and retain maintenance/pruning regression coverage.

## Accepted host upgrade procedure

The current service is root-owned and passwordless sudo is unavailable.

1. In the terminal running **`serve-fake`**, press **Ctrl+C** to stop management.
   Runtime containers continue independently. Keep the SSH listener running.
2. Install the rebuilt checkpoint:

   ```bash
   sudo /usr/bin/install -D -o root -g root -m 0755 \
     /home/thelifeandtimes/repos/omabit/build/urbit-runner \
     /usr/local/lib/urbit-runner/checkpoints/6n/urbit-runner
   ```

3. Start management and leave this terminal running:

   ```bash
   sudo runuser -u urbit-runner -- /usr/bin/env -i \
     HOME=/var/lib/urbit-runner PATH=/usr/bin:/bin \
     /usr/local/lib/urbit-runner/checkpoints/6n/urbit-runner \
     serve-fake --acknowledge-development
   ```

The agent verified the stable identity and capability, retrieved the fake's code,
authenticated an ephemeral Eyre session, logged it out, verified the old cookie no
longer authenticated, and checked secret-free snapshots/audit. No ship restart was
needed. The panel's `+code` action is now admitted for a fresh operator selection.
