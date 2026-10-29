# Direct private HTTP publication and recovery passes

VM14/appliance15 completes the keyed two-ship lifecycle, receiver replacement,
whole-gateway replacement and retained application recovery. The replacement
gateway publishes only the two Eyre TCP ports to the disposable VM's IPv4 loopback.
A direct client outside the containers uses the original sessions and saved
subscription cursor, without relogin or write replay.

The independent stopped-disk audit verifies23 direct HTTP requests: two new
application writes, four SSE reads, four state scries and seven channel PUTs,
plus both host/inventory and crossed-session checks. Five connection probes
refuse RPC, both administrative endpoints and both IPv6 HTTP endpoints. Exact
bridge ownership/options, requested and active loopback port bindings and both
network observations match. No RPC, admin or Ames publication is present.

The same audit verifies the original peer messages, four-write receiver recovery,
exact retained chain/genesis/head/public keys, original piers and sessions, queued
revision5 and old/fresh subscription progression through revisions6 and7. Five
ship shutdowns and both gateway shutdowns are clean. The runtime VM powers off
cleanly in2000.18s; final read-only audit29 passes in11.29s with unchanged backing
and overlay hashes.

Audit27 initially refused direct-request coverage. A fixed-counter diagnostic in
fresh read-only audit28 established seven channel PUTs, not the verifier's expected
three. The shared Recovery implementation sends four cursor acknowledgements in
addition to one subscribe and two deletes. The verifier and synthetic fixture
were corrected to require exactly seven; missing/extra requests still refuse.
All146 reader checks and vet pass. Audit29 reran on the same unchanged stopped
disk. No runtime rerun or evidence alteration was needed. Audits27/28 and their
original reader sources remain retained.

The runtime and all three audit scopes are inactive. The disk is privately retained
with verified SHA256 `afb95762635721ae0f310475f310a0e228cf3167146735a087761e609255138c`
and a compatibility link. Keys, cookies, chain/pier data and raw private logs remain
private. This supersedes M368–M371's pending publication status and M368's
three-PUT verifier assumption. It does not qualify simultaneous private projects,
public runner initialization/launch or a supported installed release. Schema42
workspace persistence is the separate M371 source slice; the frozen VM used the
older schema40 runner only for guest installation. Production ships, tmux,
temp-ur, installed plugins and host Docker remain excluded.
