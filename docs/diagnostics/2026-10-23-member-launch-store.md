# Durable initial member launch — 2026-10-23

Schema27 adds the private member-launch journal and operation mirror transactionally,
preserving schemas11–26 and all original project/storage/anchor receipts. Acceptance
is actor/request/revision/review-bound and reserves one initial launch per member.
Initial launches are admitted serially per project; completed peer receipts supply
ownership bindings, never a substitute for fresh runtime observation. The worker
binds its fixed contract before any effect and commits each model phase atomically
with its mirror. Startup interrupts queued/running work without replay or adoption.

All nine acceptance/binding/dispatch/result boundaries survive close/reopen with
unchanged journals and earlier receipts. Replays, late callbacks and new attempts
cannot take over interrupted work. Competing workers have one winner; a mirror
write failure rolls back the journal update. Corrupt ownership, namespace, mirror,
status or orphan receipt refuses opening. A second member retains the first's exact
container/runtime binding and cannot omit it. Inert worker callbacks read SQLite
inside every effect to verify committed ordering; late Create errors retain the ID,
and lost Start replies or post-ack cancellation preserve their precise phase.

Focused project race suites and vet pass. Full managed race tests pass in90.780s.
The original schema26 source snapshot contains1201 files; its2567294-byte archive
is pinned by SHA-256 in the JSON report. The unchanged writer reproduces an80238-byte
fixture containing completed anchor and both storage receipts. Its reader refuses
actual schema27 workspace-dispatched launch state without modifying it. Earlier
fixture and RC.12 qualification claims remain unchanged. No installed database
was migrated, and no real member container/pier/ship was created.

The terminal launch receipt records a running container in the original namespace.
It does not establish initialized pier storage, HTTP/admin identity, healthy apps
or subscription recovery. Concrete workspace/pill and Docker dispatch, server
scheduling, private-VM qualification and readiness remain. Later Stop/resume and
anchor recovery must keep this initial receipt and record their own evidence.
