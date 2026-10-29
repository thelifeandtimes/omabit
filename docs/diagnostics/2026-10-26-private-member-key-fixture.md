# Guest-only member key proof and independent stopped-disk reader

M420, 2026-10-26. A build-tagged qualification bridge exercises the M419
primitive against each fresh private-chain member. It requires a new no-NIC
QEMU guest, the exact private owner root and the management lock. Every proof
is bracketed by complete gateway observations; the returned key hash is checked
by the real held-file verifier. Original ready/recovery records must remain
unchanged. No member is booted.

Each member has one valid case and two explicit negative controls. The controls
change only the public point supplied to the evaluator (authentication key or
life); key files and chain remain unchanged. A negative case requires %.n.
An evaluator failure or timeout cannot pass it. The complete private proof stays
on the guest disk. The ordinary runner excludes the qualification bridge.

A separately tagged PID1 reader mounts the stopped disk read-only. The existing
gateway/history audit remains mandatory, followed by six proof checks against
the audited public chain and exact retained key bytes. Canonical receipts reject
duplicates, aliases and unknown fields. Only counts, booleans and receipt/log
hashes leave the reader: actual key contents and their individual hashes remain
private. The reader does not re-run the evaluator or grant launch authority.

Four bridge source checks,32 tagged audit checks and three ordinary audit checks
pass. Integrated affected suites pass170 checks under the race detector plus
three ordinary-reader checks, with no skips. Vet, diff, static builds, fixture,
reader/reader-child and guest-entry host refusals pass. No engine, real key or
live chain was accessed. Real runtime qualification is next.

Two synthetic-test mistakes were retained and corrected: the historical writer
contains a ready row paired with an interrupted workspace, so the test must use
the separate recovered ready result with the complete workspace. A head-hash
negative initially reused the existing all-f value; it now changes that value.
Neither correction changes production or audit admission rules.

M420 extends M419 and supersedes its pending fixture/auditor preparation only.
Real evaluator/key-chain proof, member-aware gateway observation, durable member
lifecycle, schema migration and public dispatch remain unqualified. Production
ships, tmux, ~/temp-ur, installed plugins and host Docker remain excluded.
