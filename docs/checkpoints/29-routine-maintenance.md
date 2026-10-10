# Checkpoint 6S: durable UTC maintenance schedules

Runner-owned routine policies and panel **0.8.0** are implemented. **Live timed
acceptance passed**; see the [record](../diagnostics/2026-09-29-routine-maintenance.md).
The test policy is disabled. The first supported actions are
the already-qualified fake-pier **pack** and **meld** workflows.
Desktop 0.8.0 and rebuilt collector/helper binaries are installed. The live shell's
status IPC confirmed 0.8.0 after restart; the previous snapshot is archived at
`/tmp/opencode/omabit-panel-before-6s-20260929.tar`.

## Policy and execution

- One revisioned policy per initialized qualified fake instance. Policies bind the
  selected instance configuration revision and authenticated operator identity.
- Explicit UTC weekdays (0=Sunday), hour/minute and a 5–180 minute **dispatch**
  window. Editing/enabling does not execute a past slot or run immediately.
- `stop_running` explicitly authorizes downtime. Without it, running instances
  are skipped. `resume` is separately explicit and requires downtime consent; it
  applies only if the instance was running beforehand and all maintenance succeeds.
- The runner checks every five seconds, including when the desktop is closed.
  Each due slot is journaled once atomically with its queued operation or skipped
  outcome. No backlog catch-up, same-slot retry or clock-reversal repetition.
- Busy, stale, drifted, unqualified, recovery-required or revoked-owner instances
  are skipped. Owner access is rechecked after acquiring the host gate. An already
  dispatched job follows its accepted policy; later occurrences recheck access.
- The whole workflow is one `routine-pack` or `routine-meld` operation under the
  existing instance exclusion and host work gate. Fresh 5 GiB RAM / 8 GiB disk
  admission happens before any Stop; the maintenance/start drivers recheck their
  own admission too. The complete workflow has a 30-minute bound.
- Stage progress is durable. A failed step never triggers automatic resume.
  Underlying maintenance uncertainty remains authoritative. Management restart
  marks unfinished work interrupted; it does not continue the old sequence.
- Policy changes during active work are refused. Configuration edits make the
  old policy revision binding ineligible until reviewed and saved again.

This is not automatic pruning, roll, loom resizing or comet maintenance. Those
remain separately qualified/policy-gated work. UTC is explicit; local-time and
DST scheduling are not inferred.

## API and UI

Capability: `maintenance-schedules`.

- Operator GET/PUT `/v1/instances/{id}/routine` reads or saves the policy.
- Snapshot `schedule` shows policy, owner, revisions, next UTC slot and last slot /
  operation. Snapshot `routine` shows the latest run's stage and outcome.
- The normal activity feed includes policy edits, routine runs and skipped slots.
- Monitor → Maintenance → Routine maintenance contains the revision-bound editor.

Collector commands:

```text
routine INSTANCE
routine-set INSTANCE --revision N --instance-revision N --enabled=true|false
  --action pack|meld --days 0,1,... --hour-utc N --minute-utc N
  --window-minutes N --stop-running=true|false --resume=true|false
```

Every policy flag is required. `--revision 0` means no policy exists yet. Saving
the first policy advances private SQLite metadata to **version 4**; afterward use
**6S or newer** for management so older binaries cannot ignore routine state.

## Verification

Local checks cover UTC slots across midnight, backwards clocks, no retroactive
enable, policy/configuration CAS, actor attribution, one durable slot claim,
dispatcher execution, revoked access, missed windows, drift, interruption,
downtime consent, resource refusal before Stop, successful stop/pack/resume and
failure without resume. CLI and QML fixtures verify explicit flags and saving a
disabled policy without live runtime/clipboard effects.

## Required runner upgrade

Press **Ctrl+C** in the management terminal, install the rebuilt binary, then
start the same mixed development mode:

```bash
sudo /usr/bin/install -D -o root -g root -m 0755 \
  /home/thelifeandtimes/repos/omabit/build/urbit-runner \
  /usr/local/lib/urbit-runner/checkpoints/6s/urbit-runner
```

```bash
sudo runuser -u urbit-runner -- /usr/bin/env -i \
  HOME=/var/lib/urbit-runner PATH=/usr/bin:/bin \
  /usr/local/lib/urbit-runner/checkpoints/6s/urbit-runner \
  serve-development --acknowledge-development
```

No new collector mode acceptance is needed. The agent will install a short-lived
test policy on the development `import-check`, observe one timed stop/pack/resume
cycle and its journal, verify no duplicate execution, then disable the test policy.
Other policies are not enabled automatically. Existing runtime containers continue
independently during management restart.
