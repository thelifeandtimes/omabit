# Fake development groups: local foundation and cross-tailnet design

## What exists in this pilot

Every development group has a private Docker bridge and a namespace-keeper
container. All ship containers in that group join the keeper's network namespace
using Docker `--network container:<keeper>`. Their pier mounts and process
namespaces remain separate, but loopback and network sockets are shared.

This supports the topology expected by default fake-galaxy routing: a galaxy's
fake Ames port is 31337 plus its galaxy index, and the default destination IP for
fake galaxies is loopback. See the Vere `ames.c` functions cited in SOURCES.md.
That does not mean fake traffic can never cross a physical network; it means
ordinary separate bridges with matching names do not provide the transparent
cross-host behavior we require. Explicit routing/relaying must preserve it.

A fake `~zod` and `~nec` in the same environment therefore use ports 31337/31338
in the same namespace. A second group can independently reuse those ports and
ship names. The pilot accepts the 256 canonical fake galaxies; fake stars and
planets need an extended routing/identity model.

GUI HTTP ports inside each group are 8080 plus galaxy index. The namespace keeper
publishes a reserved 256-port range on **host loopback only**, starting at 20000
for the first group. This intentionally simple pilot allocation reserves more
ports and may create more Docker forwarding overhead than a final design.
An instance-aware group proxy is a likely optimization, not a proven requirement.
Per-ship laptop tunnels still use distinct loopback IP browser origins.

No fake UDP ports are published to the LAN. Each group shares its loopback admin
APIs and is a single trust boundary. It is not a hostile-tenant isolation unit.
The keeper and bridge remain when the last member is archived. Keeper/member
restart ordering after a Docker/host reboot requires explicit live testing.

## Cross-tailnet behavior is a required next increment, not implemented

A Docker bridge is local to one Docker host. A common bridge name on two hosts
is not an overlay network. Adding a workstation to the laptop's SSH inventory
does not yet join their fake development groups.

The candidate design is an environment-scoped fake-galaxy UDP relay. It should be
validated against actual Ames **and Mesa** traffic for the chosen Vere build
before accepting it as the final transport. Alternatives include supported
explicit fake routing in the runtime, but must meet the same isolation contract.

### Candidate routing mechanics

Assume an environment contains `~zod` on host A and `~nec` on host B:

1. On A, the local zod owns loopback UDP 31337. A relay socket impersonates only
   the declared remote nec's port 31338 in that environment namespace.
2. Zod's outgoing datagram to 127.0.0.1:31338 is received by that relay. The relay
   preserves source galaxy, destination galaxy, environment ID and packet bytes.
3. An authenticated outer transport sends the envelope to host B over its
   Tailscale address. Only enrolled peers for that exact environment are allowed.
4. On B, a relay socket owns the remote zod's loopback port 31337. It injects the
   datagram from that port to local nec at 127.0.0.1:31338.
5. Return traffic takes the inverse route, preserving the fake loopback lanes.

This is a transport hypothesis, not demonstrated behavior. Packet size, reverse
path semantics, runtime port binding, Mesa behavior, PMTU/fragmentation and peer
restart behavior must all be tested. Do not simply forward all 256 fake ports
from the workstation onto its public interface.

A practical host daemon could terminate one tailnet-bound outer relay endpoint
and connect to a per-environment namespace worker through a local Unix socket.
That avoids assigning the host's Tailscale IP directly inside each Docker
namespace. Another option is an explicitly tailnet-bound published port per
environment. The decision should follow the first two-host packet experiment.

### Required enrollment and isolation

- A stable shared environment UUID across participating hosts, not merely equal
  display names. Current group IDs are host-local creations.
- An explicit galaxy-to-host directory with at most one owner per galaxy per
  environment. Loss of contact does not automatically release that assignment.
- Tailscale peer authorization plus environment-scoped credentials/capabilities.
  Tailnet membership alone must not silently enroll a host in every dev group.
- Strict source/destination validation, bounded packets, replay/duplication policy
  as required by the chosen outer transport, and no relay between environment IDs.
- Real ships never use this adapter. No implicit bridge into public Ames/Mesa.
- Expose “peer host unreachable” honestly; do not silently create another zod.

### Acceptance matrix

Same-host zod↔nec; workstation-zod↔laptop-nec; duplicate independent zod groups;
three hosts; peer disconnect/reconnect; host reboot; simultaneous starts; wrong
environment credential; wrong source galaxy; duplicate ownership; unregistered
peer; oversized packet; packet loss and reordering; Ames/Mesa compatibility;
proof that public/other-environment UDP cannot reach a fake member unexpectedly.

The first remote-runner pilot is useful before this relay exists: the laptop
already controls a group that runs entirely on the workstation.
