# SSH inventory churn and sourcebox D-Bus pressure

User-supplied report: `brave-dbus-failure-2026-10-09.md`. The report records Brave
SIGABRT/libdbus failures coinciding with system-bus resource disconnects, including
failure with a fresh browser profile. It reports intact browser data and no OOM.

## Confirmed observations

At 2026-10-10 01:20:57 UTC, read-only sourcebox diagnostics found:

- Boot ID `fb8e7939-0d60-453e-9c1e-d3e40559b5da` (booted September 29).
- About 70 GiB available RAM, no kernel OOM entries, broker memory 119,230,464 bytes.
- Broker journal resource-related peer disconnects matching the reported failure
  period. `coredumpctl` had no Brave cores; the reported Crashpad/assertion evidence
  is supplied by the sourcebox report rather than a newly symbolized core.
- **421 new `urbit-runner` login sessions in 15 minutes.** The workstation journal
  independently showed the same approximately two-second cadence.
- Queued system-bus sockets belonged to native `urbit` processes owned by the
  desktop user, UID 1000. Follow-up identified `~/.local/bin/urbit`, launched from
  tmux scopes. Their queue-draining behavior was not traced inside the binary.
- Omabit's two containerized ships and manager run as UID 954 on sourcebox. The bus
  inventory did not identify those managed runtimes as the backed-up subscribers.

## Cause and scope

`internal/collection/inventory.go` called `inspect()` for every watch poll.
`inspect()` constructed a new pinned client and closed it on return. With the panel's
two-second inventory cadence, each connection cycle launched OpenSSH, and the
dedicated daemon's existing PAM stack created/closed a logind session.

This is a confirmed Omabit defect and a source of excessive logind signals. Combined
with the observed unread native-Urbit bus queues, it supports the reported resource
pressure mechanism. It does not establish that Omabit was the sole cause of every
Brave failure or that the separately launched binaries' bus behavior is repaired.

## Fix in 0.22.3

- Each watched runner owns one client/transport for the watch lifetime.
- Healthy observations reuse the connection; roles/capabilities remain refreshed.
- Failed observations discard the client and use bounded exponential retry delay.
- Cancellation closes retained clients, pending dials and idle SSH transports.
- Every new connection still verifies the enrolled identity; the bridge and runner
  still recheck operator/observer grants and revocation on every request.

No change to system D-Bus quotas, PAM policy or unmanaged ship configuration was
needed for this fix. The user's original report is preserved unchanged.

## Validation and deployment

Full Go tests/vet and collector/client/access race checks passed. New fixtures
verify connection reuse, cancellation cleanup, role refresh, changed identity after
reconnection, and revocation within a persistent bridge connection. Existing slow/
unavailable-host isolation tests passed. UI model and cold-installer checks passed.

A real watch completed four polls for each runner using the same two SSH processes.
After deployment, sourcebox recorded **6 runner sessions in 15 minutes**, including
diagnosis and update activity; the measured minute following the manager update
contained one reconnect, rather than continuous two-second logins.

Desktop IPC reports 0.22.3. Verified artifacts:

- Collector/plugin: `575a2e1a7d41e46d256e542b9aee930ef434bd635dd870f7f37b0c62849b4c65`.
- Local runner: `bc8d2f449b903110fc1a4f29b0fea2b9c9ced66aec4f5ec01d92cdc0df7358c4`.
- Sourcebox runner: `683fd13d7369dbaddac726509fcb2edfad6cce092402703494d7dd02de0cf667`.

These builds also include the previously prepared 4 GB boot-pill allowance.
Before/after snapshots preserved all managed ship identities, states, containers,
ports, revisions, policies and moved-source fencing. The sourcebox installer
returned ready=true and its installed binary passed checksum verification.

Private evidence under `/tmp/opencode/`:

- `sourcebox-dbus-before.txt`, `sourcebox-dbus-pre-reboot.json`
- `sourcebox-dbus-update.log`, `sourcebox-dbus-update.exit` (0)
- `dbus-fleet-before.json`, `dbus-fleet-after-update.json`

## Post-reboot recovery verified

The user subsequently rebooted sourcebox and managed the native ships manually.
The new boot ID, active services, two successful managed startup-resume operations
and intentionally stopped `panel-check` were verified. The user confirmed normal
Brave operation. Broker memory fell to about 9 MB and no resource-exhaustion messages
were found in the new boot. A one-minute window showed one runner login and stable
bus queue totals. Some nonempty native-client queues remain; longer observation is
still appropriate. See the [reboot acceptance record](2026-10-10-sourcebox-reboot.md).
