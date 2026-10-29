# Durable initial member Stop — 2026-10-23

Schema29 adds an operation-keyed Stop journal with a unique member/container
binding. Its accepted intent references a completed original readiness receipt;
fresh runtime evidence remains mandatory at dispatch. The complete journal is
mirrored in the operation receipt and checked against original launch/readiness
bindings. Actor/request replay returns the original receipt. A different request
cannot silently re-signal that container, and unresolved project Stops prevent
another member Stop from being admitted.

Dispatch, signal acknowledgement and exit observation commit separately. The
worker finalizes success only after a clean acknowledged outcome. Nonzero/OOM
exit evidence is retained as failure; cancellation, lost replies and late wait
errors preserve interrupted evidence. Restart never re-signals or removes a
resource. A complete retained exit may remain interrupted when final acknowledgement
was lost. Earlier readiness/launch/storage records remain unchanged.

Tests cover seven restart boundaries and seven worker outcomes, stale/tampered
proofs, concurrent single-intent admission, changed replay and corrupt/orphaned
receipts. Focused tests, full managed race tests and vet pass. The original
schema28 reader was frozen before migration; populated readiness fixtures reproduce
byte-for-byte, and it refuses actual schema29 Stop state without modification.
Its bounded fixture limit is 160 KiB to retain both full readiness proofs and
mirrors; older fixture limits and published-release evidence remain unchanged.

No concrete Stop adapter, server queue, public route, automatic recovery or live
Stop is introduced. VM15 remains the older frozen schema28 readiness trial.
