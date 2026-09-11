# Tend multi-ship verification matrix

This is the reproducible release evidence for Tend's owner-authoritative peer
path. It intentionally uses fake ships so transport, authorization, persistence,
and UI-facing access state can be exercised without production identities.

Last run: 2026-09-10  
Runtime: Vere 4.6  
Userspace: `%zuse` 409 on the owner; locally compatible `%zuse` 408 fake-ship
bases on the two participants  
Tend state: `%10`

## Topology

| Role | Test ship | Behavior under test |
| --- | --- | --- |
| Owner | `~zod` | Canonical list, ACL, sequencing, restart/session rotation |
| Inviting participant | `~bus` | Replica, delegated invite policy, remote edits, leave |
| Read/write participant | `~nec` | Replica, stale edit, targeted revocation |

These names are test identities only. Production authorization is rank-neutral:
the marks and Gall agent accept arbitrary valid `@p` values, and client tests
cover planet-, moon-, and comet-shaped identities.

## Verified scenarios

| Scenario | Expected invariant | Result |
| --- | --- | --- |
| Owner invites two ships | Each target receives an authenticated invitation; disclosure is limited to the selected complete list | Pass |
| Invite policy | `~bus` receives `can-invite`; `~nec` does not | Pass |
| Both accept | Each home agent allocates a local alias and stores a non-authoritative replica | Pass |
| Participant edit | `~bus` adds one reminder while owner is Online | Pass; owner and both replicas converged |
| Activity convergence | Owner and participant mutate the shared list | Pass; authenticated actors, local aliases, and a recurring occurrence timestamp converged through snapshot and live subscription paths |
| Private presentation | Owner and participant choose different list order/sort settings | Pass; settings persisted locally and did not enter peer snapshots |
| Collaboration policy | Participant suppresses added alerts but enables completed/assigned alerts | Pass; only enabled categories arrived, durable alerts replayed until ack, and acknowledged alerts did not replay |
| Owner offline | Stop `~zod` and wait beyond the liveness threshold | Pass; both replicas remained readable and became Offline |
| Offline write | Attempt `add` through `~bus` | Pass; CLI rejected before poke and Gall remained unchanged |
| Owner restart | Restart `~zod` with the same pier | Pass; rotated session forced catch-up, then both replicas returned Online |
| Duplicate operation | Replay one operation ID twice at the same base revision | Pass; one revision and one reminder were committed |
| Stale operation | Submit from `~nec` against an older base revision | Pass; rejection left all canonical content unchanged |
| Owner revocation | Remove `~nec` | Pass; its replica was deleted while `~bus` retained the list |
| Participant leave | `~bus` leaves | Pass; its replica, private settings, collaboration alerts, and list-scoped due-alert state were deleted and the owner ACL became empty |
| Cleanup | Owner deletes the temporary list | Pass |

The same live environment also verified durable alert replay/acknowledgement and
owner-only atomic export/restore, including a no-mutation rejection when restore
was attempted against non-empty state.

## Remaining release-candidate cases

- Automate the topology and assertions as a repeatable harness rather than a
  recorded live drill.
- Exercise explicit Ames congestion/backpressure and subscription kicks.
- Remove a participant while its already accepted edit remains in flight.
- Verify supported mixed-client and mixed-desk version combinations.
- Run repeated owner and participant restarts, pier continuity/key rotation,
  and a longer real-ship soak across planet, moon, and comet identities.

Until those gates pass, the peer protocol is a working pre-release protocol,
not a stable compatibility promise.
