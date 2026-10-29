# Durable project member storage preparation — 2026-10-23

Schema26 records an actor-scoped operation, the reserved runtime/member identity,
reviewed plan and exclusive `fakeships/<instance>/<ship>` claim before filesystem
work. Its journal commits mkdir intent, returned directory identity and verified
completion separately. A late sync failure preserves the identity; interruption
never adopts an existing parent, erases partial data or silently retries mkdir.
The parent remains empty: no pier or ship is created.

The private worker and real filesystem helper pass temporary-directory tests and
race checks for each commit boundary, cancellation, competing claims, atomic mirror
rollback, corruption, replaced parents, existing data, legacy move/path conflicts
and external-import overlap. Reopen preserves journals and interrupts unfinished
receipts once. Full managed-runner race tests pass in80.555 seconds; project race
regressions and vet pass. No installed state was opened or changed.

The frozen original schema25 writer reproduces its73,636-byte fixture with completed
creation/Start journals byte for byte and refuses actual schema26 dispatched member
storage unchanged. Snapshot SHA256 is recorded in the JSON report. Redundant nested
source archives are excluded from this new snapshot and listed by original hash;
the unchanged original archives remain separately retained. Earlier release and
historical source claims stay frozen.

Initial source-test failures are retained: an overbroad retry expectation after a
failed pre-dispatch commit, then two synthetic historical-export setup errors
(reserved port collision and an intentionally forbidden storage validator). Their
corrected reruns pass. Server resource/update scheduling, actual private-VM storage
integration, member container lifecycle and public interfaces remain unfinished.
