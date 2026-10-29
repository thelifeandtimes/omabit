# Durable retained-pier replacement — 2026-10-23

Schema32 adds a separate resume journal that consumes a completed generation-one
retirement once. Acceptance preserves that retirement and prior Stop/exit receipts.
The concrete adapter binds the generation2 contract before the first dispatch;
Create/Start progression and operation mirrors commit atomically. Every unfinished
phase becomes interrupted on restart, retaining returned identities and evidence.
Neither restart nor a new request silently retries the consumed retirement.

Unresolved resume fences new work in the same project through the existing shared
lifecycle gate. Earlier request receipts remain replayable; unrelated projects are
unaffected. Eight restart boundaries, eight invalid transitions, five worker cases,
three corrupt-state cases and concurrent admission/project fencing pass. Initial
historical checks, focused race tests, the full managed race suite (281.786seconds)
and vet pass. Private queue files were added after the full suite started and have
separate focused evidence.

The original schema31 writer was frozen before migration: archive SHA256
`74dcd3b8847474f3d2772af0177e0c85573f5a231fdab958b9a3f9af4829ebf9`.
Its211127-byte populated fixture reproduces exactly and retains completed retirement
plus prior uncertain Stop/clean observation. Its original reader refuses real
schema32 resume state without modification. Only schema31's fixture gets a224KiB
bound; older bounds and published-release qualification remain unchanged.

The journal currently handles the first replacement only. It does not establish a
general generation ledger, actual runtime readiness or public project API. The
schema31 VM18 retirement result cannot qualify this source. No live state changed.
