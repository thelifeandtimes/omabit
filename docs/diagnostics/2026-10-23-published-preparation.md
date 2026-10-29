# Direct fake-ship publication preparation — 2026-10-23

The opt-in `--network-published` case extends the existing two-fake transport and
state/subscription sequence. An ordinary bridge and init-managed anchor reserve
exact guest loopback endpoints. Public Eyre requests originate directly from the
guest and carry per-ship cookies in memory; administrative Lens setup stays inside
the project namespace. The gate requires exact active publications, public ship
identity, IPv6 refusal, Lens refusal on the bridge address, preserved sessions and
application state after member replacement, independent logout and clean stops.

All32 offline fake-network checks pass, including eight direct-client tests.
They exercise session identity, secret-free receipts, no redirect following,
no mutation replay after a lost response, response bounds and SSE cursor/event
handling. The source and appliance artifacts are independently hash-verified;
[metadata](2026-10-23-published-preparation.json) records their locations. The
prepared appliance is `/tmp/omabit-import-appliance-20261023-01`.

No runtime result is claimed yet. The first published case is one project; paired
published projects, real-ship anchor replacement and runner project ownership
remain separate gates. Pinned Vere source confirms that `--http-ip` affects the
public server, while the administrative server binds loopback independently.
The guest has no NIC, host shares or production credentials. No existing desktop,
engine or ship is accessed. This continues the HTTP-fixture pass in M033.
