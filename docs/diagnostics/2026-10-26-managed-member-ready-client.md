# First member VM refusal: ready client phase

M436, 2026-10-26. The M434 fresh no-NIC VM completed host/runtime verification and
managed gateway initialize/recover/verify. The first member launch then failed.
The guest shut down normally after 1,156.33 seconds; no outer timeout, storage
failure, host Docker access or protected resource operation occurred.

The independent schema45 read-only disk audit confirms all original gateway
receipts, separately attributed recovery, retained filesystem identities and
five operations. The member journal remains `planned`, with its operation
`interrupted`, and no member launch is qualified. Source inspection identifies
an unconditional factory refusal before the member controller: the new factory
passes its validated `ready` gateway to `openPrivateGatewayClient`, whose reuse
branch admitted only `start-dispatched` and `started`. Thus this attempt never
reached member artifact creation or Docker Create/Start. The guest pill cache was
staged before queue admission and is retained with the disk.

An inert regression reproduces exactly that mismatch: valid retained metadata
fails for the ready phase; missing metadata, foreign selection, changed marker,
and fresh creation at ready all refuse. The fix also permits ready-phase reuse,
while retaining the exact original selection marker, ownership/permission checks,
no-repair behavior and read-only opening. The caller still validates complete
ready-gateway evidence and holds its management fence. 198 affected race checks,
vet and diff checks pass without skips. Metadata bytes, object identity, mode and
mtime remain unchanged; no engine calls occur in the regression. This source fix
is not yet a successful managed member boot.

The independent auditor now reports its private key read explicitly when a full
member audit reaches that check. Final auditor SHA256:
`22368065868ea4d38d703d96434b1699b762aec560e8c493c7e3e8457aebad6f`.
This supersedes M434's earlier auditor binary hash only. This failed run did not
reach the key-reading member audit stage. Both disk and base remained unchanged
under inspection. The stopped 3,413,573,632-byte disk was copied, synced and
hash-verified on the private lab volume, retaining its original compatibility
path. Disk SHA256:
`7b8b3ed11502ee6c8ff67c7ef5b94d95322e3409622d92bed17a6fdfb6ac0ee8`.
The supervisor, runtime scope and audit scope are terminal.

M436 resolves M434's pending first-run outcome and fixes the M433 ready-client
integration gap. It does not supersede pending runtime readiness, second member,
managed Stop/recovery or public orchestration. Schema45 is unchanged, and the
original45 source freeze remains a persistence baseline. The next runtime attempt
must use a fresh VM/chain/member plan, never replay this interrupted launch.
