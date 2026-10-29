# Initial graceful member Stop model — 2026-10-23

The pure model binds an initialized member to its original complete launch set
and readiness storage identity. Historical readiness is never current authority:
a fresh complete observation must establish the exact selected running container,
unchanged project resources, peer state and pier objects before signal dispatch.
The caller commits dispatch intent before one graceful signal. The driver contract
forbids forced termination and removal; waiting is read-only and context-bounded.

Completion requires the same container process start time, a valid exit time,
unchanged pier identities and project/peer observations, with only the selected
member becoming stopped. Nonzero/OOM exits are retained and return failure. A
late observation error can retain complete exit evidence without claiming success.
Lost signal, cancellation, failed commits and incomplete observations preserve the
last durable phase. Execution cannot automatically resume a partial/completed Stop.

Fifteen controller scenarios, twenty changed-observation cases, transition/binding
checks, already-stopped peer preservation and pre-observation cancellation pass.
Focused project race regressions and vet also pass. Only inert drivers were used.
Schema remains28: no persistent Stop journal or actual runtime adapter is added.
Uncertain-launch cleanup, concrete Stop, resume/recovery and public APIs remain.
The current VM15 contains the earlier frozen readiness binary.
