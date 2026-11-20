# 0.5.0 source review

Review target: frozen product source
`59d83aab49384d2f26531320bb5ab68e1257aeb5`, compared with protected master
`10962196a19c349e540c9b980d4d4da53c8657b6`. Final distribution pins and release-head
checks remain pending. This record is not release approval.

## Contract review

- Stop diagnosis selects the original operation, actor, project revision, plan,
  ledger, container generation and initialized pier. A fresh clean exit must
  match engine, anchor, namespace and original process timestamps. Reconciliation
  reloads and compares these bindings under the manager lock; exact receipt replay
  precedes any new observation. It neither signals a ship nor starts a replacement.
  The native transition test is a bounded interleaving, not a reproduction of the
  original incident's unestablished failure predicate.
- Guided project creation retains one canonical plan and parent request. Each
  advance selects an exact derived native phase ID. Saved status refuses changed
  history and does not imply current readiness. App preflight checks capabilities
  and role before reservation, then checks the exact ship/desk policy separately.
  Environment compatibility grants no future permission. Grubbery advertises only
  its implemented source/restart contract.
- CLI and MCP call the same typed service handlers. MCP defaults to no mutation
  tools, requires explicit runner selection, and excludes application-permission
  authoring and generic command execution. Local source staging opens an explicitly
  selected root and retains source digest, stage ID and reviewed plan. Lost replies
  use original receipts rather than replacement requests.
- Recording controls preserve original request, revision/digest and job bindings.
  Admission charges every retained job's maximum, including closed jobs. Unknown
  capacity refuses admission. Automatic retirement explicitly refuses; an age or
  policy field does not implement deletion. Original, indexed and derived redacted
  exports have separate explicit contracts and preserve original content.
- Schema initialization creates the new journals without lowering a version or
  discarding predecessor history. Exact stable0.4/schema59 upgrade qualification
  checks original operation and app records, running/stopped intent, restart and
  actual older-reader refusal. Product artifacts carry the same clean source.

Relevant implementation: `internal/managed/project_stop_recovery_linux.go`,
`internal/managed/project_stop_recovery_store.go`, `internal/projectflow`,
`internal/appdev/preflight.go`, `internal/service`, `internal/mcpserver`,
`internal/managed/recording_control_api.go`, `internal/recording`, and
`internal/managed/store.go`. This is a contract-focused review alongside automated
source, race, QML, interface and native evidence; it does not claim exhaustive
line-by-line review or independently reproduce each historical development test.

## Qualification evidence and remaining review

The [source CI rollup](../../diagnostics/2026-11-20-candidate02-source-ci.json)
records successful core, four managed-race shards, QML and aggregate source checks.
The release-policy refusal is expected until exact final distribution pins and
qualification are recorded. Matching
[artifact/interface/desktop validation](../../diagnostics/2026-11-20-release050-candidate02-frozen-validation.json),
[stable upgrade](../../diagnostics/2026-11-20-release050-final-upgrade.json), and
[recording pressure](../../diagnostics/2026-11-20-recording-pressure-candidate02.json),
[Grubbery recovery/state/restart](../../diagnostics/2026-11-20-grubbery-candidate02.json),
[native Stop](../../diagnostics/2026-11-20-project-stop-candidate02.json), and
[closed recording exports](../../diagnostics/2026-11-20-recording-exports-candidate02.json)
pass. The preserved DEV gate remains in [preparation](preparation.md); do not
infer installed qualification from these disposable-runtime results.

Legacy task help still contains older app-policy wording. The typed operation
catalog and matching bundled workflow references define the supported commands;
broader legacy parity and named-client interoperability remain issue19 work.
Peers, permits, routes, assets, broader failure combinations, host-wide recording
inventory and retirement are explicit exclusions, not completed issue claims.
Issue22 runtime pins remains outside this release.

Final review must verify that all post-freeze changes are permitted documentation
or exact distribution pins, that draft downloads match the qualified bytes, and
that required checks pass on the final PR head. Protected merge must preserve the
frozen source with a merge commit. Publication does not authorize a production
service update.
