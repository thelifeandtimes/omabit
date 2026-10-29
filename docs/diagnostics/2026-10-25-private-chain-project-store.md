# Private-chain project persistence

Schema41 adds distinct immutable private-project plans and member claims. Registration
commits runner/project/generation identity, runtime identity tokens, the reviewed
foundation and an actor/request receipt atomically. Opening validates the plan,
claims, original receipt and shared fake/private reservations. Standalone creation,
configuration and port suggestions also honor those reservations. No key, chain,
pier, container, publication or runtime dispatch is created by registration.

The actual schema40 source was frozen before changes: 2,429 files, archive SHA256
`052de08d5f4aaec592af98396bf457c3b21bc16847e14d59ae1ae6843e58e14f`.
Its synthetic writer reproduces identical bytes twice: eight instances,44 operations,
22 tables and two continuation receipts. Fixture SHA256
`006a7a33198f014a10bb33d240e5a1c37bd1f45cd5b25b946d5526b4eed5cac1`.
The unchanged original reader refuses actual migrated41 containing private ownership
without modifying it; no schema-marker relabeling is used.

Five focused tests/28 including subcases,17 historical/reservation tests/279 including
subcases, six race tests/29 including subcases and vet pass. Cross-kind and standalone
conflicts, partial-insert rollback, corrupt/missing claims/receipts, replay after
restart and concurrent admission are covered. The full managed suite is running
separately and is not claimed by this checkpoint. Public API/CLI, runtime lifecycle,
simultaneous-project VM isolation, installed upgrades and release remain open.

VM09's stopped key-bearing disk was also hash-preserved in private home storage;
its former lab path is a compatibility symlink. Only relocation metadata is exported.
This supersedes its physical temporary-storage location, not M352's qualified result.
Production ships, installed plugins/services and excluded paths were untouched.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-25-private-chain-project-store`.
