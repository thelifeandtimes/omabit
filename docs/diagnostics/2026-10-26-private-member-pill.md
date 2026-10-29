# Private member bootstrap-pill verification

M416, 2026-10-26. The read-only verifier binds the selected first-boot contract
and recorded runtime workspace. Production entry pins the existing release pill
SHA256 and217,421,172-byte length. It holds the private root, workspace and0400
single-link file; rejects symlinks, cross-device children, extra workspace files
and changed objects; and hashes the complete artifact before accepting it.
Later checks compare held/named identities and immutable metadata, excluding
atime. Reopening after Close requires hashing again. Resume is refused by this
bootstrap-only helper. It never stages, repairs, syncs or starts anything.

All32 targeted pill checks and109 combined private-member race checks pass,
along with package vet and diff checks. Tests use a tiny synthetic artifact via
an internal test constructor; the production constructor refuses that substitute.
No real pill, key, pier or engine was accessed. This is source evidence only.

The initial happy-path fixture had a non-private temporary root and correctly
refused. The fixture now explicitly creates the required0700 root and separately
tests refusal of0755 roots; verifier rules did not change. The second wrapper
initially counted the old failed log's successful cases. Its retained correction
counts the correct new log:32 passes, no failures. Integrated tests independently
confirm the count; the old summary remains for provenance.

Member staging, exact retained-pier/stopped-state checks, key/chain proof,
member-aware gateway observation and durable lifecycle remain. Schema43 and
public interfaces are unchanged. This extends M412–M414 and supersedes only
their pending pill-verifier source gate, not runtime qualification. Same-UID
applications remain trusted; file identity checks are not wallet custody.
