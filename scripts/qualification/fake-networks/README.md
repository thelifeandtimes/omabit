# Private fake-network transport qualification

Read `docs/experiment-isolation.md`. This probe runs only inside a newly prepared
no-NIC QEMU guest with the guest's own rootless engine. It never enrolls or operates
an existing user ship, uses the host engine, or installs an active desktop plugin.

The existing `imports/appliance.py` image builder accepts the mutually exclusive
`--exercise-fake-network` case. Use the signed package cache, pinned runtime and
kernel as with the import harness, but do not pass `--exercise-import` or a copy
case. Every preparation/run directory must be fresh. The original driver names
are retained so the same quota checks, private mounts, evidence extraction and
scope admission apply.

The probe creates an anchor container with `--network none`, then fresh fake
`~zod` and `~nec` containers in that anchor's network namespace. Each has its own
pier mount and HTTP port (8081/8082). Ames keeps the discoverable local galaxy
ports. No ports are published. The anchor has only loopback and no IPv4 route.
The two ship namespaces must match the anchor and differ from the guest host.

For each ship the driver reads its generated `.http.ports` and verifies the Lens
endpoint's `our` identity before obtaining the fake's code. Codes pass only in
memory/stdin; private cookie jars live in the anchor's tmpfs. Authenticated Eyre
scries must work, and presenting the other ship's cookie must be refused.

The probe starts a built-in local `khan-eval` thread through authenticated Spider
HTTP. The typed `ted-eval` input contains a fixed, validated program: send an
ordinary Gall `%helm-hi` poke to `%hood`, wait for its application acknowledgement,
then return the unique expected receipt. Ship names and nonce/receipt strings
are allowlisted before constructing code. No arbitrary Hoon input is accepted.
The typed mark avoids the earlier unqualified generic-noun thread assumption.

Every positive reply must be HTTP200 with the exact receipt. The deliberate `fail`
poke must return HTTP500 with the `child-failed` result and exact `thread-fail`,
`poke-fail`, and `poke-hi-fail` trace labels. A generic HTTP500, timeout, submission
success or unrelated mark error cannot pass. Each attempt is recorded before
sending and is never retried. App data travels directly to the ship.

Same-ship controls precede bidirectional peer messages. Both directions run again
after cleanly stopping `~nec` and resuming the same pier in a new runtime container
within the same anchor namespace, using the original session jars. The boot
container is retained stopped. Resume omits `-F`, `-B` and `-c`: Vere rejects
creation arguments on an existing pier. Both container IDs and the pier device/inode
are recorded. This tests process/container replacement with durable ship state;
it does not claim that replaying the boot container command can resume.

Direct remote Eyre channel pokes are **not** the network probe: both local channel
controls passed, but the pinned runtime returned HTTP500 for remote submission.
Gall rejects foreign `%poke-as` operations. See [the failed trial and native
control](../../../docs/diagnostics/2026-10-22-fake-network-channel-trial.md).
The earlier generic-noun `hi` timeout is retained as failed evidence too. Batch
runtime stdout is not used as an application transcript.

`thread_probe.py --khan-receipt --vere PINNED_BINARY --pill PINNED_PILL --output
/tmp/omabit-thread-probe-FRESH` tests the thread's positive and negative receipts
on one fresh native fake. `--peer` tests two fresh native fakes and same-pier process
restart. The single-ship scope allows 4 GiB and the pair 8 GiB, both with no swap, one core,
128 tasks, and private network,
PID and mount namespaces. They accept only the exact pinned static binary and pill;
these native controls do not establish a Docker pass. The two-peer outer deadline
is 15 minutes. `--channel --remote-submission` preserves the remote Eyre failure
control; bare `--channel` tests local Eyre receipts. No option reuses a user pier.

Run `python scripts/qualification/fake-networks/test_receipts.py` for six offline
receipt-matching and request-boundary tests. Retain the precise current live status
in the checkpoint evidence; source availability is not a passed runtime gate.

The case stops only its own new ships and anchor, retains the containers/piers and
original guest disk, then lets the admitted guest power off. The runtime case has
7 GiB guest RAM, two exposed CPUs and an outer 12 GiB/one-core/no-swap/128-task scope.
The guest unit has a 55-minute deadline and its outer driver a 60-minute deadline
for two fresh boots. Boot/readiness and individual commands have tighter bounds.

This is a transport experiment. It does not implement runner project ownership,
anchor recovery, endpoint allocation/publication, cross-project duplicate identity
isolation, persistent application subscriptions/state recovery, two-runner networks
or private Ethereum chains. Results must be recorded separately; presence of this
script is not evidence that any runtime gate passed.

TCG first-boot readiness allows 15 minutes per ship. Trial03 approached the former
12-minute limit; the extra margin changes no memory/CPU/network isolation limit.

Current checkpoint: [Docker peer messaging and session recovery passed](../../../docs/diagnostics/2026-10-22-docker-peer-recovery.md).
This is the first transport rung plus clean process/session recovery; persistent
application state/subscription recovery and project isolation remain open.

## Persistent application-state extension (native and Docker passed)

`thread_probe.py --peer --application-state` runs the state/subscription control
in the existing capped native boundary. The Docker counterpart is
`imports/appliance.py prepare --exercise-fake-network --network-state` with the
same pinned input requirements. Neither flag installs anything on a user ship.

`omabit-state.hoon` is a disposable test fixture, not a plugin backend. It records
four unique peer writes in order, publishes JSON snapshots, and rejects duplicate
entries. The client acknowledges an initial snapshot and one update, disconnects,
queues another write, restarts the receiver from the same pier, and reconnects
using the original channel/cursor/cookie. It then verifies live updates on that
subscription and snapshot reconciliation on a fresh subscription. No write is
retried. Mutating fixture setup uses acknowledged Gall requests; read-only checks
wait for the asynchronous Clay commit before app startup.

The bounded SSE readers intentionally close each HTTP connection after collecting
the needed evidence. This proves channel reconnection and durable subscriptions;
it does not simulate a TCP connection spanning a process restart, nor power loss.
A clean process restart does not by itself exercise Gall agent upgrade/on-load.
Use `test_state.py` for five offline evidence/input-boundary checks in addition to
the six transport receipt tests. A runtime pass must be recorded separately.

## Duplicate-identity projects (source prepared; runtime pending)

`imports/appliance.py prepare --exercise-fake-network --network-isolation` implies
`--network-state`. It creates project A, completes recovery, then keeps A live
while a second pair boots in project B. Both pairs use zod/nec and the same internal
HTTP port numbers. Both anchor namespaces have only loopback and no publication.
The second project is admitted only when the engine contains exactly the first
project's recorded container IDs. Unexpected containers stop the experiment.

The comparison requires all four current ship containers live in distinct project
namespaces, refuses each same-identity cookie in the other project, then performs
one additional peer write in each project and checks that the other state remains
unchanged. Cookie copies travel through private stdin into the other anchor's
private tmpfs, never argv/reports, and are removed after the negative check.
Five `test_project_isolation.py` cases reject false evidence offline. The runtime
case has not yet passed. The running two-ship state appliance is frozen separately.

This four-ship case requires 32 GiB of available host memory before scope admission,
uses 14 GiB guest RAM and a 24 GiB/no-swap/one-core/128-task outer cap, and has a
110-minute guest / 120-minute outer deadline. Run it only after the previous VM
stops. Earlier two-ship cases keep their 7 GiB guest / 12 GiB outer limits. All
cases retain no-NIC VM isolation; none uses host Docker or publishes a host port.


The [native control](../../../docs/diagnostics/2026-10-22-application-state-native.md)
and [Docker state extension](../../../docs/diagnostics/2026-10-22-application-state-docker.md)
now pass. These supersede the earlier state/subscription-pending text above;
duplicate-project isolation remains under qualification. Each frozen report binds
its own source/artifacts; current source changes do not inherit a runtime pass.

## Endpoint publication experiment (prepared, unqualified)

`imports/appliance.py prepare --exercise-endpoints --endpoint-network internal`
uses the same no-NIC guest and private rootless engine with small HTTP identity
responders, not ships. The native responder control and offline publication-shape
checks pass; Docker publication and anchor lifecycle have not run yet.

Two bridge projects each share an anchor namespace and the same member ports.
Only exact IPv4 loopback ports on the guest host are published. The test verifies
identities, rejects unintended IPv6 publications and unpublished cross-bridge
access, then observes anchor restart with members still alive. Coordinated repair
stops all project members before recreating them against the current anchor.
The alternate explicit `egress` mode is a separate experiment if internal-bridge
publication fails; it is never an automatic fallback. Neither mode makes the
existing host engine or a production ship accessible. Preserve failed evidence.

## Actual plugin transport control (prepared, unqualified)

`thread_probe.py --peer --plugin-transport --theme-source SOURCE --dms-source SOURCE`
stages visibly separate `auth-lab` copies of the experimental Theme and DMs clients.
Their direct transports write to two fresh fake ships, then reconnect after one
ship's same-pier restart. Theme uses its real appearance writer/reader and watch;
DMs uses the real bridge, acknowledgements, invitation acceptance and live/history
reads. Every write has a unique recorded identity and is sent once.

Desktop application, native launchers and keyring persistence are deliberately
inert: sessions use an in-memory fixture. The earlier separate Secret Service gate
remains the storage evidence. This control cannot qualify native desktop/profile
activation. Its five offline evidence tests and isolated import/profile preflight
pass; actual writes/reconnect are not yet qualified. The normal eight-GiB/no-swap
native pair scope still applies, with an additional eight-GiB temporary-storage
admission and two-GiB monitored reserve for this mode.

The actual plugin transport control now passes in native runs01 and02. Run02
qualifies strict DMs receipt validation and independently matches every write to
its wire poke/receipt/live event/history. See
[the runtime report](../../../docs/diagnostics/2026-10-22-plugin-receipts-native.md).
This supersedes the unqualified runtime status above, not its explicit inert
desktop/keyring limits.

## Direct published Eyre gate

`appliance.py prepare --exercise-fake-network --network-state --network-published`
selects the single-project gate inside the same no-NIC guest. An ordinary
project bridge and init-managed anchor reserve exact guest127.0.0.1 ports18281/2
for public Eyre8081/2. The fakes' public HTTP servers bind within that private
namespace; Lens remains loopback-only. No host port is published by QEMU.

The actual message/state/subscription sequence uses direct guest-loopback HTTP,
with independent cookies held only in memory. Administrative setup still uses
identity-checked Lens from inside the namespace. Exact active bindings, IPv6
refusal, Lens refusal on the bridge address, unchanged publications after member
replacement and independent logout are required. Request receipts omit bodies,
cookies and codes; uncertain writes are not retried. This does not yet exercise
real-ship anchor replacement, duplicate published projects or runner ownership.
`test_published.py` checks the client with inert connections, including uncertain
writes, redaction, session binding and bounded event streams.

The single-project gate passed on run23-01; see
[verified runtime evidence](../../../docs/diagnostics/2026-10-23-published-state.md).
Adding `--network-isolation` prepares two published projects with duplicate zod/nec
identities and disjoint guest-loopback ports18281/2 and18291/2. Cross-project
sessions remain in memory for refusal checks; neither cookie files nor credentials
are logged. Both projects stay live during independent state mutations. Exact
publications are checked alongside container/namespace membership. The four-fake
resource/deadline admission applies. This combined gate is prepared source until
its separate actual runtime receipts are verified; single-project evidence does
not qualify it. No real-ship anchor replacement or runner integration is implied.

## Actual-ship anchor recovery (prepared source)

`--network-anchor-recovery` requires the separate single `--network-published`
fake-project case and enables state recovery. After the normal member-restart gate,
it creates a third subscription and queues a fifth write, stops/restarts only the
owned anchor, and requires both public endpoints to refuse connections while old
members remain live in the old namespace. It explicitly stops both old members
before creating either replacement, verifies original pier inodes/new namespace,
then recovers the queued event and performs a sixth write using original sessions.
Old containers and piers remain retained. Four inert ordering/refusal checks pass;
this is not actual runtime evidence until a fresh case completes. It cannot be
combined with the duplicate-project flag in this first anchor-recovery case.
