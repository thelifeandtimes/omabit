# Managed comet compaction accepted — September 29 UTC

The 6U runner advertised `comet-compaction` on the existing identity-pinned
`development-managed` endpoint. The live validator ran once on `comet-check`,
configuration/applied revision 1/1, port 18087, pinned public identity
`~nolpel-modsyd-livsep-novmeg--sallen-tassul-padlud-litzod`.

## Manual pack

| Step | Operation | Completed UTC |
| --- | --- | --- |
| Stop | `op-7b95c4d16950b90223f8e3c3a721ca0e` | 14:44:13.998409933 |
| Pack | `op-08202bebf1a0b276975479e91adb4f3d` | 14:44:20.081287767 |
| Resume | `op-d9cdd46dcc02d0567d45eccee84cf8f1` | 14:44:28.274580183 |

Requests were `checkpoint-6u-{stop,pack,start}-comet-check`. All terminal requests
replayed exactly, including pack replay after the ship was running again. Pack
measured **1,042,391,052 → 690,331,660 logical bytes**, between
14:44:16.235059338 and 14:44:20.078474172 UTC. Admission reported
7,620,755,456 bytes available RAM / 433,412,608,000 bytes free disk.

Running-pier pack was refused with `invalid_state`; comet roll and epoch inspection
were refused with `unsupported_for_instance`. The explicit resume used a replacement
container and passed direct loopback `/~/host` identity verification.

## Timed meld

- Policy: `6e62568f371be676cf1eb5b26a9b97c0`, initial revision 1, instance revision 1.
- Owner: `ssh-key:SHA256:eVDrU/23cFq0QD1D5uuM10niBgNueT+kVji4wXdJNYc`.
- Saved: `2026-09-29T14:44:34.201853792Z`.
- Slot: Tuesday **14:46:00 UTC**, five-minute window, stop_running=true, resume=true.
- Operation: `op-ed9d54077cea536efdfa98953604944c`.
- Request: `schedule-6e62568f371be676cf1eb5b26a9b97c0-1790693160`.
- Created: `14:46:03.444076482Z`; completed: `14:46:21.134544305Z`.
- Meld: `14:46:05.369118682Z` through `14:46:14.554333105Z`.
- Logical bytes: **694,472,716 → 660,377,612**.
- Admission: 7,402,528,768 bytes available RAM / 433,411,624,960 bytes free disk.
- Polled stages: meld → start → complete; stop was too brief for the one-second
  poll to capture. The completed routine records was_running=true and resume=true.
- Final container: `15bdb3d59da9f4935ecc4eabcb64231e35aa578500b8ddef8db1b7063d510db3`.

Fresh telemetry, a replacement container and the same direct public identity were
verified. Multiple subsequent scheduler ticks produced no duplicate slot operation.
The exact test policy was **disabled at revision 2**,
`2026-09-29T14:46:34.713821984Z`, with no next slot; an independent read confirmed it.
The earlier import-check test policy remains disabled at revision 2.

Alpha remained healthy at 2/2; panel-check and import-check remained healthy at
1/1. Comet-check ended running/healthy at 1/1. No active work, recovery requirement
or incident remained. This exercises managed pack and scheduled meld; logical bytes
do not measure loom utilization or guarantee reclaimed filesystem blocks.

Private schema 5 now requires **6U+ `serve-development`** for management.

## Desktop

Panel **0.9.0** was installed with the owned-snapshot installer and the shell was
restarted. Live `thelifeandtimes.omabit.monitor status` returned version 0.9.0.
The prior 0.8 snapshot is archived at
`/tmp/opencode/omabit-panel-before-6u-20260929T144725Z.tar`.
Comet pack/meld and routine editing are capability-gated; comet roll and retention
remain disabled pending their own qualification.
