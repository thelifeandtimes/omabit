# Public initial project actions

The new `project-operate` CLI exposes five explicit actions through
`POST /v1/projects/<id>/operations`: project preparation, initial anchor Start,
member storage preparation, fresh member launch and initial readiness observation.
Each action advertises its own backend capability. Operator authentication, exact
project/revision/digest/request/member selection and confirmation are required.
The route delegates to the existing durable queue and resource/update gates.
There is no automatic startup dispatch or mutation retry.

Lookup-only replay uses the original actor/request/fingerprint and starts no
worker. Responses are validated durable operation receipts, not runtime health.
CLI help, error guidance and the operator skill explain the action effects and
the distinction between lookup and explicit recovery. The runtime wire endpoint
has a refusal stub outside the qualified Linux/AMD64 platform.

All five HTTP routes pass queued/terminal replay and existing inert execution.
Refusal cases cover role, missing capabilities, irrelevant/missing member,
confirmation, revision, digest, management fence and unknown project/action.
Client tests cover lost replies and mismatched returned bindings; CLI tests cover
all five actions in text/JSON with lookup-only headers.

An opt-in test additionally runs the actual built collector against a private Unix
server and temporary store with an inert engine. It plans and reserves a project,
accepts preparation then loses that reply, verifies no automatic resend, retrieves
the same queued receipt with lookup-only, completes one worker and reads the same
successful operation through the actual saved-status CLI. No ordinary ship runtime
is invoked. This passes1.142s separately and is included in the applied race pass.
The retained collector was built before the final help-only update.

Applied focused races pass: managed17.223s, client1.057s, collector1.046s; vet,
diff and skill validation pass. Schema40 and persisted fingerprints are unchanged.
Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-public-initial-project-actions`.

Actual Docker execution through these public commands, public member Stop/resume
and recovery, and QML remain pending. Frozen VM33 predates these public mutations.
No installed plugin, production service or existing ship was changed.
