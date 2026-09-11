# Tend multi-ship verification matrix

This is the reproducible release evidence for Tend's owner-authoritative peer
path. It intentionally uses fake ships so transport, authorization, persistence,
and UI-facing access state can be exercised without production identities.

Last run: 2026-09-11

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
| Ames pressure/release | Hold 12 concurrently submitted `~bus` edits at the owner's Ames boundary, then release them | Pass; one current-revision edit committed, the other 11 settled as stale, and all replicas converged |
| Outstanding-edit removal | Hold an already-submitted `~nec` edit at Ames, revoke `~nec`, then release the packet | Pass; the edit never changed canonical state and the revoked replica/in-flight state were deleted |
| Owner revocation | Remove `~nec` | Pass; its replica was deleted while `~bus` retained the list |
| Participant leave | `~bus` leaves | Pass; its replica, private settings, collaboration alerts, and list-scoped due-alert state were deleted and the owner ACL became empty |
| Restart/resubscription soak | Two cycles each of participant and owner Gall suspend/revive | Pass; every cycle returned through Checking/Offline to Online with field-equivalent canonical content after normalizing `%zuse` date rendering |
| Cleanup | Owner deletes the temporary list | Pass |

The same live environment also verified durable alert replay/acknowledgement and
owner-only atomic export/restore, including a no-mutation rejection when restore
was attempted against non-empty state. The table-driven
`check-tend-migrations.sh` gate separately passed non-empty `%0` through `%10`
saved-state fixtures against the same library used by Gall load.

## Repeat the automated matrix

The checked-in `scripts/check-tend-multiship.py` harness creates a uniquely
named fixture and removes it in a `finally` path. It requires three running,
already-authenticated fake ships with the same Tend release installed. From the
source tree, set the nine paths and run:

```bash
TEND_OWNER_CONFIG=/path/owner-connection.json \
TEND_OWNER_COOKIE=/path/owner-cookies.txt \
TEND_OWNER_PIER=/path/owner-pier \
TEND_MEMBER_A_CONFIG=/path/member-a-connection.json \
TEND_MEMBER_A_COOKIE=/path/member-a-cookies.txt \
TEND_MEMBER_A_PIER=/path/member-a-pier \
TEND_MEMBER_B_CONFIG=/path/member-b-connection.json \
TEND_MEMBER_B_COOKIE=/path/member-b-cookies.txt \
TEND_MEMBER_B_PIER=/path/member-b-pier \
TEND_MULTISHIP_OPTIONS='--backpressure-count 12 --restart-cycles 2 --timeout 120' \
make check-tend-multiship
```

This is destructive fault injection for disposable test ships: it temporarily
suspends `%tend`, replaces the owner's Ames snub policy, and restores the
default empty deny-list. Do not run it against a production pier or a fake ship
with unrelated networking policy. The harness never reads or prints a `+code`;
it uses the supplied private Eyre cookie jars.

## Remaining release-candidate cases

- Verify supported mixed-client and mixed-desk version combinations.
- Run a longer real-ship soak, including pier continuity/key rotation and
  planet, moon, and comet identities rather than fake identities.

Until those gates pass, the peer protocol is a working pre-release protocol,
not a stable compatibility promise.
