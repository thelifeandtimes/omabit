# Remote comet and maintenance acceptance — 2026-10-09

Sourcebox / `remote-test`, runner `runner-b4043fa2d0aee501aa2cf78322aed911`.
Management used the existing identity-pinned restricted SSH connection over Tailscale.

Fresh `remote-comet` boot succeeded at 04:15:46Z, producing
`~sapbyt-lignum-widnec-mirbud--faslec-hobnet-pinlup-wanzod`. Its permanent pier is
`livenet/comets/remote-comet/pier`, HTTP 18082, configuration revision 2.
The first boot container was
`edb5e33bb94b7498e7f78c05724c794eafc434248106f8d3a1ed40531d6f525a`.

Both disposable remote ships then completed, in order:
**Stop → Pack → Meld → Inspect epochs → Roll → Inspect epochs → Start**.
Every operation succeeded with a unique request ID. Both returned healthy with
their original identity and the 60-minute snapshot interval. No epochs were pruned.

| Ship | Roll created | Final container after explicit resume |
| --- | --- | --- |
| `remote-comet` | `0i3526` | `49fdc91191e5d3923fd7bcf940803055ea334a3e4721b26acf371d8344f65d2d` |
| `remote-smoke` / `~zod` | `0i7742` | `d5b6443dc464b1aee83b285515141cde202a6542153e935c9ecc4337b3a339a1` |

The sequence ran 04:29:51–04:30:57Z. The local private receipt is
`/tmp/opencode/remote-maintenance-20261009.json` (`complete: true`), with before/after
snapshots, all 14 terminal operations, and the two before/after epoch inventories.
The helper is an executed receipt-producing script, not a command to rerun blindly.

## Port failure follow-up

Actual TCP/UDP bind checks now run at registration/configuration/start admission and
again at runtime dispatch. Docker start failures retain safe actionable categories
without echoing daemon stderr. Real socket-conflict fixtures and dispatch-race tests
cover these checks. The sourcebox 18080 registration attempt was accepted; that trial
did **not** prove live occupied-port rejection. The new comet was explicitly
reconfigured to 18082 before its successful first Start.

## Remaining live gates

- Tailnet HTTP/+code authentication and a stopped cross-host move subsequently
  passed after 0.22 deployment; see the [handoff record](2026-10-09-stopped-host-handoff.md).
- Deferred sourcebox reboot, reconnect/interruption and inbound Ames qualification.

The already accepted fake HTTP 18081 route and local fleet were not changed by this
remote maintenance sequence. Historical pier backup/restore is excluded from the UI.
