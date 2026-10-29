# Durable project reservations — 2026-10-23

The development store now accepts a complete canonical project plan and its
review digest in one transaction. It assigns stable project/member runtime IDs,
reserves every instance ID and HTTP port, and writes an actor-scoped terminal
receipt. No instance, pier or Docker resource is created. The store API is private;
no project capability, RPC or CLI command is advertised yet.

Standalone registration/import revision admission, HTTP configuration, allocation
and suggestions respect project claims. New projects respect standalone desired
and retained runtime ports. Admission checks current host availability; replay
returns the original receipt without probing again. Invalid ownership records
refuse reopening or new allocations rather than releasing their claims.

Schema23 adds the project tables and prevents older runners from ignoring this
ownership. Historical bodies, signatures, port/runtime intent and replay receipts
are preserved; opening a store does not adopt members or launch anything.

Validation: full managed/projectnetwork/protocol race suites passed, including
concurrent independent store connections, actor/request collisions, rollback on
receipt insertion failure, changed reviews, host/legacy reservation conflicts,
restart replay and corrupted membership. Vet passed. All checked-in historical
fixtures migrated without altering original records. The frozen schema22 source
reproduced its synthetic fixture, then its original reader refused an actual
schema23 database containing two project reservations without modification.

The schema22 source is explicitly an unreleased snapshot, identified by archive
SHA256`4f5029ac1469d6cb8b74c99e3fe88eb6932ee2f561460e7a48dd02834b6f8ed5`.
It is not the b233128 release tree; that commit is only its source base. The
original fixture and generator are retained in testdata. Full regeneration of
all earlier writers stopped at a missing local preview18 Git object`ba4ba805…`;
this does not invalidate the checked-in fixture migration tests or targeted
schema22 reconstruction/refusal. It is recorded as an incomplete broader check.

Source hashes and logs are retained at
`/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-23-project-store`.
Next: revision-bound resource journals and a driver, then runner/CLI/UI and private
VM qualification. This checkpoint neither releases nor installs schema23.

Reproduce the immediate-predecessor gate with `python3 scripts/test-historical-upgrades.py --check-generated --only schema22`. Targeted checks now include the actual successor refusal and do not require the unrelated missing preview18 object.
