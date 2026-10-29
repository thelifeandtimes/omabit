# Public saved project status

The runner advertises `project-saved-status-v1` and serves the exact read-only
`GET /v1/projects/<id>/status` route, including observer access. The collector's
`project-status PROJECT_ID` validates the pinned runner and reservation, then
renders text or JSON. The newest 128 operation summaries are bounded by insertion
order with an explicit truncation marker. Private runtime bodies, container IDs
and freeform operation messages are omitted.

The view explicitly reports saved operation history, with runtime observation not
performed; a historical running/succeeded status is not current health. Operator
help and the bundled skill explain this boundary. There is no mutating project
route or persistent-format change in this slice.

Applied focused race tests pass: managed12.449s, client1.033s, access1.008s and
collector1.028s. Vet, diff and skill validation pass. The cases include interrupted
recovery history, malformed/unbound responses, capability refusal, observer route
scope and zero database changes. The guarded VM helper compiles and rejects host
execution; it will exercise the actual collector in the forthcoming continuation
VM. The fixture now binds the owner UID explicitly. The appliance hashes all Go
inputs for that public-client experiment; 44 inert harness tests pass.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-project-saved-status`.
Runtime CLI qualification remains pending. No installed software or live ship was touched.
