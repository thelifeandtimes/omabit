# Remote caller-loss and request-replay acceptance

At 02:47:59–02:48:11 UTC, the disposable managed `remote-smoke` fake on sourcebox
passed a caller-acknowledgment-loss and observer-disconnect check. The target was
pinned to runner `runner-b4043fa2d0aee501aa2cf78322aed911`, instance revision 2,
identity `~zod`, initially running/healthy.

## Procedure and results

1. Explicit Stop succeeded as `op-c28922990ebd1fa2b3d885d667b7d23b`.
2. Submitted Pack with request `release01-ackloss-pack-20261010`, deliberately
   discarding the CLI's acknowledgment instead of retaining its returned operation.
3. Recovered the accepted operation from the runner journal while it was running:
   `op-44031b276077035e9d2f872f0f00a037`.
4. Terminated the test's persistent observer/SSH connection during that job and
   reconnected with a fresh pinned observer.
5. Explicit replay of the identical request returned the same running operation,
   actor and creation timestamp. No second operation was created.
6. Pack succeeded at 02:48:03 UTC. Replay after completion returned an identical
   terminal receipt.
7. Explicit Start succeeded as `op-e2a410a355ba4e57537e840c80c6ed5d`, restoring
   healthy/running `~zod` at revision/applied 2/2.

The before/after comparison preserved all other managed states and container IDs,
identities, ports, revisions, policies, uninitialized registrations and move fences.
The user-managed native ships were outside this test's target set.

Private receipt: `/tmp/opencode/remote-ackloss-20261010.json`, complete=true.
Executed one-shot helper: `/tmp/opencode/remote-ackloss-20261010.py`.

## Scope

This is application-level acknowledgment loss and observer connection loss during
an accepted durable operation. It does not claim a physical network partition,
packet loss before server acceptance, abrupt runner death, or interrupted offline
utility recovery. Those remain separate qualification cases.
