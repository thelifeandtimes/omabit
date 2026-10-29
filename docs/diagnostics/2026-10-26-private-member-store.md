# Durable first private member launch

M428, 2026-10-26. Development schema44 adds the initial private member launch
journal. Each row is tied to its reserved member/runtime, original or separately
recovered gateway readiness, actor/request fingerprint and exact operation mirror.
Admission, returned-object checkpoints and terminal publication are transactional.
Opening the store validates history; restart reconciliation marks unfinished
work interrupted without changing its launch evidence or starting any process.
Exact request replay works before asking a backend to reconstruct its contract.

This boundary admits one initial member per project. Second-member admission
requires a separate durable readiness contract and is deliberately unavailable.
The terminal running receipt proves only the selected process observation. It
does not establish HTTP readiness, pier ownership for resume or clean Stop.
The store executor still requires a caller-held common runner work/update fence;
there is no public route, queue, automatic dispatcher or concrete engine driver.

Verification:

- 34 new store checks cover every phase across three reopen/reconciliation cycles,
  actor-scoped replay, second-member refusal, returned objects after driver errors,
  atomic admission/checkpoint/completion failures and unchanged corruption refusal.
- Two migration checks preserve every original43 row and sidecar while adding
  completed/interrupted member receipts, and roll back invalid gateway ancestry.
- 536 integrated private-lifecycle/historical race checks pass with
  no skips; vet and diff checks pass. The initial combined run reached its
  eight-minute suite budget after 289 passes, with no test assertion failure.
  A separate run completed only the uncompleted top-level tests; the timeout
  output and exact partition are retained. No runtime budget changed.
- The pinned original43 writer reproduces the existing 1,369,311-byte fixture
  exactly. Its original reader refuses populated state from the real schema44
  migrator without changing that state. No schema-marker substitution was used.

The first migration-test attempt failed because a reused schema43 helper expected
six additional operations rather than two. The corrected shared helper retains
exact original-cell comparisons and an explicit added-receipt count. That failed
attempt remains in the private evidence. Runtime acceptance rules were unchanged.

All databases, identities, observations and adapters in these checks are synthetic.
No Docker, service, actual key, ship, production profile or installed plugin was
accessed. Existing schema43 VM evidence remains separate. The next managed VM
auditor must explicitly support schema44; its schema43 check must not be bypassed.

M428 extends M427 and supersedes its pending SQL integration, plus M408's pending
original43 successor migration/refusal gate. First-boot capture, durable member
readiness, second-member integration, concrete lifecycle and actual runtime remain.
This is an unreleased backend schema change, not a stable or installed upgrade.
