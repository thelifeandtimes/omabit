# Project fake-network research

Pinned-runtime research and initial transport qualification, 2026-10-22.
The private two-ship transport/session-recovery gate now passes; managed project
networking is not yet implemented. Keep the remaining acceptance ladder in
[the platform roadmap](platform-roadmap.md#step-3--project-fake-networks).

## What the pinned runtime does

The tested image reports Vere `4.6-5a7614f`. The local Vere repository contains
the full corresponding commit `5a7614f69bbc783736b08965fdc0a2534bb42e82`.
These observations use `git show` at that commit, not the working tree:

- `pkg/vere/io/ames.c`, `_ames_czar_port` and `_ames_czar_lane`: local galaxy
  discovery uses `127.0.0.1` and UDP `31337 + galaxy-number`.
- The same file's `_ames_send_lane` rejects non-loopback destinations in local
  mode; initialization puts fake ships in that mode.
- `pkg/vere/io/mesa.c`, `_mesa_io_talk`: local UDP binds to loopback. Galaxies
  default to the discoverable galaxy port; an explicit incompatible Ames port
  produces a discoverability warning.
- `pkg/vere/main.c` also sets fake ships to local mode. `--local`, `--http-port`
  and `--ames-port` are separate options.

Consequently, merely attaching independent fake containers to one Docker bridge
does not give them the loopback namespace these paths expect. The private shared-namespace application-message test now passes; that does not
qualify an alternative bridge-only topology.

## Smallest next experiment

Use the private VM and rootless engine, with two newly created fake galaxies in
one dedicated network namespace. An anchor container can own the namespace while
each ship retains its own container, pier mount and process lifecycle. Start the
experiment with no external network and no host-published ports; a disposable
test client in that same namespace can verify direct Eyre access. Do not use the
host network namespace or change the runtime to enable live networking for fakes.

Keep the default discoverable fake-galaxy Ames ports. Give each ship a distinct
HTTP port inside the shared namespace. The current Omabit runtime assumes HTTP
8080 in its command, publication, probes and identity checks; changing only its
Docker network flag would therefore be incomplete. Design explicit internal and
external endpoint bindings after the transport experiment, with identity checks
for each endpoint. Omabit must not become an application-data proxy.

Prove a deterministic authenticated Gall message in both directions, then a
restart of one ship without creating a new pier. Repeat the same identities in a
second independent namespace and verify no message or endpoint confusion across
projects. Verify the absence of live-network routes and publications. Namespace
membership trusts all project participants: they share loopback and can reach
each other's administrative HTTP endpoints. It is not a hostile-app sandbox.

Before integrating this into the runner, settle durable ownership of the anchor,
its recovery when a participant exits, explicit project identity independent of
ship and instance names, port allocation, complete namespace/member inspection,
and failure behavior. Replacing an uncertain anchor must never silently attach a
running ship to a new namespace. Two-runner networks and Ethereum private chains
remain separate later tests; this source observation qualifies neither.

## Administrative endpoint discovery

Further inspection of pinned `pkg/vere/io/http.c` found a second shared-namespace
constraint. `_http_serv_start` increments a non-explicit server port on address
conflict. The loopback server begins at 12321, and `_http_write_ports_file` records
the resulting `PORT secure|insecure loopback|public` rows in each pier's
`.http.ports`. Each ship therefore needs a separately verified administrative
endpoint as well as its distinct public HTTP endpoint. Existing Omabit
`internal/runtimequalify/login_code_linux.go` hardcodes 12321; that assumption
cannot be carried unchanged into project namespaces.

The first guest probe reads the exact new pier's port file and checks `our` before
using Lens for that fresh fake's code. It uses authenticated Eyre directly for
application requests. The local Arvo source at
`50c7f5239610fc72263df7aa99b4d171cfe9d408` supplies the expected standard `hi`
thread, `%hood` sender log and `%helm-hi` rejection behavior; that checkout is
reference evidence, not proof of the pinned pill's contents. The runtime probe
must establish actual compatibility. Source is in
`scripts/qualification/fake-networks/`; the first runtime trial failed at the initial cross-ship message.
See [the first trial report](diagnostics/2026-10-22-fake-network-trials.md); it remains
failed. The corrected probe and subsequent pass are documented below.


## Endpoint publication remains a separate design gate

Docker does not permit per-member port publication in `container:` network mode.
The shared namespace therefore needs an explicitly designed publication and
allocation lifecycle before desktop clients can reach each ship. Simply retaining
the current per-container `--publish` call will not work. The first probe publishes
nothing and uses a disposable client inside the namespace. This follows Docker's
[documented container-network restrictions](https://docs.docker.com/engine/network/#container-networks),
checked 2026-10-22; it is not a chosen production forwarding design.

Preserve direct plugin-to-Eyre transport. Omabit's collector must not become an
application-data proxy to paper over this networking requirement. Project endpoint
reservations, anchor replacement, uncertain membership and adding/removing members
need durable runner contracts and their own runtime tests after transport is proven.

## Qualified transport checkpoint

[The private Docker peer/recovery run passes](diagnostics/2026-10-22-docker-peer-recovery.md):
two local controls, peer messages in both directions, a deliberate remote Gall
rejection, then both directions after resuming the same pier in a replacement
container. Original Eyre sessions and the anchor namespace survived. Both ships
stopped cleanly and the VM powered off; original artifacts and receipts were
independently hash-verified.

The corrected probe uses a typed local `khan-eval` thread that sends an ordinary
Gall poke and returns a bound receipt only after its acknowledgement. A deliberate
rejection must name `poke-hi-fail`. Direct remote Eyre channel pokes were disproved
by HTTP500; Gall rejects foreign `%poke-as`. Prior failures remain retained. A
[native peer control](diagnostics/2026-10-22-gall-peer-control.md) proved the corrected
protocol before the passing Docker run.

This is not complete project-networking qualification. Persistent application
state/subscriptions and client reconnect are the next restart checks, followed by
same-identity separation across projects. Anchor recovery, endpoint publication,
runner ownership and private Ethereum chains remain open. The CLI's existing
project folders must not be presented as communicating networks yet.

## Runner integration constraints after the transport ladder

The current runner has no durable project-network identity or membership model;
its existing project labels are filesystem organization. Source inspection shows
HTTP8080 assumptions in command generation, identity probes, publication and
containment validation, plus the administrative12321 assumption noted above.
Introduce explicit bindings behind a new opt-in project contract; do not weaken
legacy containment checks or silently regroup existing instances.

A runner-owned project needs an immutable ID independent of its label, a durable
anchor identity, member instance IDs/revisions, unique fake identities within the
project, unique internal HTTP endpoints, and observed identity-checked admin ports.
Persist uncertain create/start outcomes and reconcile exact IDs/labels/namespace
membership. Never replace an uncertain anchor while members still run in the old
namespace. A collector restart or another operator must not create a second anchor.
Changing storage folders cannot implicitly change network membership.

Desktop endpoint publication is a separate gate. Docker's documented
[container network mode restrictions](https://docs.docker.com/engine/network/)
exclude per-member publication. An anchor with explicitly reserved loopback
publications on a private internal bridge is a candidate to test, not an adopted
implementation. Published ports, internal bridge access and IPv4/IPv6 behavior
must be qualified on the pinned rootless engine; do not infer them from `none`
namespace tests. See Docker's
[publication semantics](https://docs.docker.com/engine/network/port-publishing/).
Keep application traffic direct to Eyre rather than routing it through collector RPC.
Adding/removing ports or recovering an anchor must have a defined lifecycle before
introducing persistence/schema changes. Current experiments publish nothing.


[The Docker state/subscription extension now passes](diagnostics/2026-10-22-application-state-docker.md).
It verifies four writes, durable state and queued/continued/fresh subscription
updates across clean receiver container replacement. This supersedes the pending
state/subscription rung above. Same-identity project separation is next; no runner
network capability or published endpoint is implied by this pass.

The [simultaneous duplicate-project case now passes](diagnostics/2026-10-22-docker-project-isolation.md).
This supersedes the pending same-identity case above. Both pairs recovered state
and subscriptions; crossed sessions were refused and interleaved writes stayed
within the intended project. Endpoint publication/anchor lifecycle and actual
runner integration remain separate gates.

The [administrative discovery prerequisite now passes source tests](diagnostics/2026-10-22-admin-endpoint-discovery.md).
This supersedes the earlier hardcoded12321 implementation description: the Go
runner discovers the selected pier's port and verifies identity before/after
retrieval. Its real-runtime gate remains open, and public internal8080/per-instance
network ownership are still unchanged.

[Trial19 rejects the internal-bridge publication candidate on the pinned engine](diagnostics/2026-10-22-endpoint-trial19.md):
requested mappings remain in HostConfig but active ports are null. This supersedes
the candidate assumption above. A separate ordinary bridge experiment remains
inside the no-NIC guest; active mappings and lifecycle must pass before adoption.

The [runner integration draft](project-network-contract.md) records opt-in identity,
endpoint ownership, whole-project maintenance, resource journals, concurrent-operator
reconciliation and migration requirements. It is a development contract, not a
shipped feature or an adopted unqualified Docker topology.

[The ordinary-bridge HTTP fixture passed](diagnostics/2026-10-23-endpoint-publication.md).
It supersedes the pending ordinary-bridge experiment above: exact loopback
publication, IPv6/cross-bridge refusal and coordinated anchor recovery are verified
for fixtures. Restarting the anchor changes its namespace and does not reconnect
old members. Actual fake ships on this bridge and runner integration are next;
external reachability remains unqualified because the guest has no NIC.

## Direct published Eyre checkpoint (2026-10-23)

[Run23-01](diagnostics/2026-10-23-published-state.md) extends the ordinary-bridge
HTTP fixture to two actual fakes: direct messages, durable state, existing/fresh
subscriptions, same-pier member replacement and session cleanup passed. It uses
an isolated qualification driver, not runner-owned projects. Keep published
multiple-project and anchor-replacement gates separate.

## Published duplicate projects and integration source (2026-10-23)

[VM23-03 passed published duplicate-project state/session isolation](diagnostics/2026-10-23-published-project-isolation.md),
superseding that pending part of the direct-Eyre checkpoint above. The original
large console-summary framing failure remains recorded separately from the
independently verified runtime result. Actual-ship anchor recovery is still its
own gate.

The runner now has source-tested schema24 project reservations/preparation
journals, a fixed private Docker adapter and private server work/update fencing.
See [the integration contract](project-network-contract.md) and memories M045–M051.
No project CLI capability or member lifecycle is shipped. The original descriptions
above remain the history of each checkpoint, not the current implementation state.
The no-NIC guest proofs do not establish arbitrary external-network containment;
keep the pinned Vere local-mode restriction and exact loopback publications as
separate requirements when implementing member commands.
