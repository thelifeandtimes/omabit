# Explicit Ames publication for legacy enrollments

The post-RC.11 source includes a stopped-state migration for live enrollments
whose historical runtime contract did not publish Ames UDP. Existing databases
remain unchanged on opening; management updates do not silently change networking.

Use an updated operator collector and runner advertising `publish-ames`. After
cleanly stopping the ship and obtaining its current revision:

```bash
urbit-collector --runner NAME --expected-runner-id RUNNER_ID \
  publish-ames INSTANCE --revision N --ames-port 34567 \
  --request-id UNIQUE_ID --confirm-network
```

The operation validates completed enrollment, ship identity, runtime provenance,
retained container/network outcomes, storage identity and locks. The selected
unprivileged UDP port must be available on the host and unreserved by another
instance or retained launch. Running, uncertain, fenced, unsupported or already
published instances are refused without changing their launch contract.

Success advances configuration revision once, keeps the instance desired-stopped
and observed-stopped, and retains the previous private launch contract alongside
its actor-scoped receipt. Ports, loom/native settings, storage objects, history
and signed handoff provenance otherwise remain intact. The same actor/request and
input returns the original receipt before stale-revision checks; changed input
conflicts. Another actor does not inherit that receipt.

The next explicit Start resumes the existing pier; it does not bootstrap it or
apply a pill. Any required enrollment identity verification still occurs before
UDP publication. Completed unpublished comet handoffs are supported, including
historical records whose identity is in `Comet.Ship` without `NativeShip`.

Older runners remain observable. The new command refuses without the capability,
and there is no new desktop control or automatic publication step.

Regression tests exercise transaction/replay failure, restart, conflicts,
provenance/identity rejection and actual command construction through fake-engine
resume. Actual livenet connectivity and live migrated-ship acceptance remain
release qualification tasks.
