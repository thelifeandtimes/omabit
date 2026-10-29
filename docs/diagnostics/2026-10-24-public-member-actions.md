# Public member lifecycle and admission diagnostics

`project-operate` now additionally exposes `stop`, `observe-exit`, `retire`,
`resume`, `observe-resumed` and `observe-current`. Every action has its own backend
capability and requires a pinned enrolled operator runner, exact project/member,
reviewed revision/digest, request ID and confirmation. Each action selects its
prerequisite with `--after`; current observation also requires the original
initialized-pier readiness receipt with `--original-readiness`.

The public route delegates to the existing queues. No effects are automatically
retried or scheduled at startup. Lookup-only replay retains actor/request/input
bindings and returns the original queued or terminal receipt without a new worker.
Retirement removes only the exited container; resume retains the original pier.
CLI progressive help and operator skill describe the sequence and current-context
requirements after peer replacement. Coordinated anchor recovery is still private.

A public-admission regression was reproduced: an unknown user-selected prerequisite
caused five lifecycle actions to return a storage error and disable later mutations.
Fresh admissions now distinguish missing/mismatched selections from corruption,
after existing journal validation and after replay lookup. Historical readers,
operation fingerprints and schema40 are unchanged. Initial launch/readiness also
report missing prerequisites without disabling a healthy runner. Damaged stored
history still fails closed; the corruption test verifies that distinction.

Expanded member/recovered lifecycle race checks pass321.622s, with client,
protocol and CLI races. Applied public-action/refusal/lookup checks pass24.172s;
client1.026s, protocol1.008s and collector1.051s pass, as do vet, diff and skill
validation. All six public queues exercise their inert backend through success,
queued/terminal lookup and changed-input refusal. CLI tests cover all eleven
actions, including exact prerequisite transport and lookup-only headers.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-public-member-actions`.
This is source/inert qualification. Actual Docker through these public mutation
commands and QML remain pending. VM34 contains this source but continues the
existing private runtime test path plus the actual public saved-status CLI.
No installed plugin, production service or existing ship was changed.
