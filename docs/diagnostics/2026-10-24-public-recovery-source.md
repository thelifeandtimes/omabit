# Public saved recovery source

`project-recovery-source PROJECT` uses observer-readable
`GET /v1/projects/<project>/recovery-source` with its own
`project-recovery-source-v1` capability and a pinned enrolled runner.
It returns the saved project/history digests and each member's creator, generation,
original-readiness receipt and available current-readiness receipt. These public
references support explicit review without reconstructing private journals.

The response is a consistent validated database read: runtime observation is
not-performed and action authority is not-granted. It opens no Docker backend,
creates no worker and changes no journal. Missing initial launch/readiness is an
ordinary prerequisite refusal; corrupted retained history still fails closed.
For an already interrupted recovery, use the preserved source/checkpoint of its
exact parent rather than replacing it with the latest history.

Managed inert tests pass9.988s, matching the existing reviewed source digest and
actual member receipt ownership while retaining database change counts and backend
calls. Observer/missing-prerequisite/corruption tests pass. Client validation checks
identity, digest, members/generations, distinct required receipt references and
explicit lack of health/action authority. Applied client1.023s, access1.008s and
CLI1.024s pass with vet, diff and operator-skill validation. Tested store bytes were
applied unchanged; only guidance was added afterward.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-public-recovery-source`.
Schema40 and historical fingerprints are unchanged. Frozen VM34 predates this
source. Public recovery effects/observations/continuation, actual public-command
Docker qualification and QML remain pending.
