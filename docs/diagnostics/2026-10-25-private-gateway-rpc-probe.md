# Read-only gateway RPC probe — 2026-10-25

M393 adds an embedded probe and its internal Go command/parser. It performs 19
read-only requests against the selected container's fixed loopback port 25432:
startup status before and after, chain ID, genesis, head observations, all nine
contract code reads and both Azimuth getKeys rows at explicit block 0x50. Code
is hashed with SHA-256 and compared with the selected public foundation. Key
rows must be distinct, nonzero canonical values with life/suite one, and retain
the selected plan's member order. The network namespace must remain unchanged.

The request layer uses native HTTP with no proxy, redirects, credentials or
caller-selected endpoint. Only the status method and five read RPC methods are
admitted. It checks response IDs and shapes, limits each response to 1 MiB,
the total to 4 MiB, each request to at most two seconds and the sequence to
twelve seconds. Importing the probe is inert. Its explicit main checks the
guest container boundary before loading the pinned ABI/ethers and emits only
bounded public metadata. It never reads chain files, keyfiles or process logs.

The Go wrapper invokes one fixed program using the bundled Node loader in an
exact container ID through the owned Docker client, with a fifteen-second
deadline and fixed environment. It uses no shell. Parsing refuses duplicate or
case-aliased fields, unknown fields, oversized/trailing data, changed foundation,
unready startup, key reuse and invalid namespace numbers. Client errors and
cancellation discard output and return a fixed error; they never retry the call.

All 46 JavaScript checks and 40 focused Go race checks plus vet pass. After
integration all 251 private-gateway Go race checks and the 46 probe checks pass;
git diff --check passes. RPC, HTTP and Docker adapters are inert, and filesystem
fixtures are temporary. The first JavaScript run expected twenty requests;
the actual correct sequence has nineteen. That fixture assertion was corrected
and the original failed output is retained. No live connection was attempted.

The parsed result deliberately does not confer CompleteContract, Private,
Synced or RPCReadOnly. The complete observer must still bracket this probe with
fresh engine/image/container/network/namespace checks and private workspace
identity observations. It must verify the qualified initializer before trusting
its startup flags. No mutation probes may be sent as an observer test. A new
sealed generation, managed backend wiring and fresh no-NIC VM qualification
remain outstanding. Existing generation06, ships, installed plugins, Docker
and services were unchanged. This supersedes only M392's missing RPC probe.
