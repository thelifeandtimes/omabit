# 0.5.0 release preparation

Status: bounded candidate qualification and all nine draft downloads passed;
exact distribution pins are recorded. Final checks, review and protected publication
remain pending. Product, UI, skills and packaging are frozen at
`59d83aab49384d2f26531320bb5ab68e1257aeb5`. Subsequent documentation and distribution
pins must refer to these exact artifacts. Do not rebuild them after pinning.

Compiler provenance correction: candidate02's release manifest and all three
product build-info records identify **Go 1.27.4**. M774's prose says 1.27.3; that
wording is incorrect and superseded here. The pinned source and artifact hashes
are unchanged. Older standalone qualification boot helpers retain their own
separately recorded compiler/source identities.

## Delivered scope

- Stop recovery retains the original operation receipt and pier identity. Typed
  diagnosis, review and reconciliation cover a clean interrupted member Stop;
  ordinary Start remains a separate explicit action. The historical incident's
  exact failure predicate is not established by this bounded interleaving.
- App preflight checks the selected runner, required APIs, adapter, role and
  named policy before app-specific project creation. The separately packaged app
  development skill describes staged source, saved receipts and versioned Gall
  and Grubbery workflows.
- The shared operation catalog supplies typed CLI and MCP handlers, saved lookup
  after lost replies, and explicit capability/authority refusals. The pinned SDK
  stdio smoke cases cover the tested transport; interoperability with every named
  coding client and complete parity with legacy commands remain follow-up work.
- The opt-in recording pilot captures bounded raw/timed output and preserves
  original history. Timing queries, indexed exports and complete derived redacted
  exports are separate explicit choices. Pressure or capacity failure must retain
  evidence and report incomplete coverage instead of silently claiming completeness.

## Upgrade support

The tested stable starting point is published0.4.0/schema59. The candidate manager
uses schema65. Matching collector, runner and desktop artifacts are required.
The exact0.4.0→0.5.0 rehearsal passes original app-history and table-definition
preservation, running/stopped intent, manager restart, authorization denial/retry,
actual old-reader refusal, a candidate desk workflow and final Stop. Actual native
desktop cold/hot activation, preferences and failed-candidate recovery are checked
separately. See the [matching stable upgrade evidence](../../diagnostics/2026-11-20-release050-final-upgrade.json)
and [frozen artifact validation](../../diagnostics/2026-11-20-release050-candidate02-frozen-validation.json).

An older manager must refuse newer state. Do not downgrade schema markers, reset
piers, discard operations or re-enroll identities to make an update proceed.
Active recording-worker upgrades, remote service upgrades, power-loss recovery,
other runtime factories and arbitrary older development snapshots are outside
this tested support window. Earlier stable versions must follow their documented
upgrade path to0.4.0 first.

## Remaining qualification and explicit exclusions

Candidate02 pressure qualification and independent original-recording/job/SQL
verification pass ([M775 evidence](../../diagnostics/2026-11-20-recording-pressure-candidate02.json)).
Matching Grubbery recovery/two-version/state/restart and independent original
history/pier/audit verification also pass
([M776 evidence](../../diagnostics/2026-11-20-grubbery-candidate02.json)). Exact native
Stop and independent trace/receipt/pier/peer/history verification pass
([M777 evidence](../../diagnostics/2026-11-20-project-stop-candidate02.json)).
Closed original/indexed/redacted exports and independent raw frame/archive/SQL
verification pass ([M778 evidence](../../diagnostics/2026-11-20-recording-exports-candidate02.json)).
The preserved review040 DEV manager/desktop update passes
([M779 evidence](../../diagnostics/2026-11-20-preserved-dev050-upgrade.json)), including
original history/intent/pier/container preservation and exact loaded tools.
The fresh minimal Gall app workflow also passes
([M780 evidence](../../diagnostics/2026-11-20-dev050-app-workflow.json)): scoped
temporary permissions, create/stage/commit/activate, authenticated API checks,
source reload, same-pier restart/state preservation and final clean Stop. Both
grants close before restart; Publish stays blocked and manual state is preserved.
Candidate source/core, all four managed-race shards, QML and aggregate source
checks pass ([CI rollup](../../diagnostics/2026-11-20-candidate02-source-ci.json)).
The [qualification record](qualification.json) binds all seven final evidence
records and the actual stable upgrade. [Draft downloads](../../diagnostics/2026-11-20-release050-draft-downloads.json)
match every exact qualified byte/hash. Required final release-head CI, review,
protected merge and stable publication remain gates. No current pending gate is represented as passed.

Broader peers, permits, routes, assets and application failure combinations remain
separate qualification. Named test execution is unavailable; a historical report
does not execute tests. Remote MCP OAuth/TLS deployment remains excluded. Legacy
task-oriented help is not a complete parity inventory; use the typed operation
catalog and matching bundled workflow references for this release's supported
commands. Broader legacy help/parity cleanup remains under issue19.

Recording has no automatic deletion or age-based enforcement. Existing-process
adoption, host-wide oldest-output inventory and retirement/pressure recovery remain
follow-ups. Resource allocation observations are explicitly incomplete and grant
no automatic Stop, deletion or containment authority. These bounded milestones do
not close the original issues in their entirety. Issue22 runtime pins is excluded.

## Publication sequence

After all release gates pass, attach exact qualification and support records to a
draft0.5.0 release at the frozen source; verify every downloaded asset. Add the
distribution pins on this release branch, pass required checks and review the
whole PR. Merge using a merge commit through protected master, then publish the
verified draft as stable/latest and verify public downloads and native startup.
Release publication does not authorize a production service deployment.
