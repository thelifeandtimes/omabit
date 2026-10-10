# Sourcebox reboot and desktop recovery acceptance

The user rebooted sourcebox and managed the separately launched native ships
manually. Omabit verification did not stop, move, initialize or reconfigure those
ships. Post-reboot checks ran through the pinned management connection and a
read-only administrator diagnostic.

## Host and managed fleet

- Previous boot: `fb8e7939-0d60-453e-9c1e-d3e40559b5da`.
- New boot: `f293e1a1-d22e-4746-b13e-e4e827d25f0b`.
- Host boot time: October 9, 2026, 19:15:18 PDT / October 10, 02:15:18 UTC.
- Dedicated Docker started at 02:15:49 UTC; runner at 02:15:50 UTC. Both remained
  active/running, and dedicated SSH was active.
- Runner identity remained `runner-b4043fa2d0aee501aa2cf78322aed911`.
- Installed runner SHA256 remained
  `683fd13d7369dbaddac726509fcb2edfad6cce092402703494d7dd02de0cf667`.

Exactly two new `system:startup` resume operations appeared:

| Instance | Operation | Result |
| --- | --- | --- |
| `remote-smoke` | `op-1f1f1d249c10bd4f9b5aca8f3fb1f078` | Succeeded 02:15:59 UTC; healthy `~zod`, revision/applied 2/2 |
| `remote-comet` | `op-d54ad15d28a058c0765030145b329b40` | Succeeded 02:16:07 UTC; healthy original comet, revision/applied 2/2 |

New runtime containers:

- Fake: `c0e61c6e12283e52371c3f1c144c1076974538d79a8e04170a7e95d23768bf08`.
- Comet: `66254e771f4d32f873f79ae777c0edd96dc183812e4138a71a26d9deff68232e`.

Both retained the 60-minute snapshot interval. Comet HTTP 18082 over the tailnet
returned `~sapbyt-lignum-widnec-mirbud--faslec-hobnet-pinlup-wanzod`.

Transferred `panel-check` remained stopped at revision/applied 1/1, with its completed
handoff record and port 18585 preserved. Its old workstation source remained moved
and fenced. The workstation fleet's states, containers, ports, revisions, policies
and uninitialized registrations matched the pre-reboot baseline. Neither runner
reported active operations, recovery requirements or incidents.

## D-Bus and Brave

Read-only diagnostic window: 02:35:09–02:36:09 UTC.

- Broker memory: **9,187,328 bytes**, compared with 119,230,464 before recovery.
- No broker resource-exhaustion/quota messages found in the new boot.
- System-bus listing completed normally.
- One `urbit-runner` login occurred during the measured minute; the previous
  approximately two-second session-creation pattern was absent.
- Brave processes were running, and the user explicitly confirmed Brave launches
  and behaves normally after the reboot.

The bus was not entirely queue-free: seven send queues were nonempty, totalling
646,656 bytes with a maximum of 213,504. These values were unchanged across the
minute. This accepts recovery from the observed exhaustion and confirms reduced
Omabit session churn; it does not claim that all native clients' queue-draining
behavior is fixed. Longer observation remains part of soak qualification.

Private receipts:

- `/tmp/opencode/dbus-fleet-after-reboot.json`
- `/tmp/opencode/sourcebox-postboot.txt`
- `/tmp/opencode/sourcebox-postboot.exit` (0)

## Release follow-up

The sourcebox clean-reboot gate has passed for the exercised permanent-layout
managed fleet. The subsequent [caller-loss check](2026-10-10-remote-client-interruption.md)
passed on a disposable managed fake. Forced runner/utility interruption and actual
network partitions remain separate qualification work.
Pill construction can proceed with the 12 required desks in `docs/pill-plan.json`;
`%omachan` is deferred and is not a 0.1.0 dependency. The pill is a bundled initial
state followed by normal publisher-driven OTA, not a version lock on running desks.
