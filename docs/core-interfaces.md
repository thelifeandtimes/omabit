# Shared CLI and stdio MCP interfaces

Issue19's development slice adds one typed service catalog over existing
`collection`, `client`, `protocol`, project and desk libraries. Run
`omabit operations` for offline JSON schemas, capabilities, roles, effect classes,
target bindings, exact-request recovery rules and the public CLI inventory. The
catalog identifies excluded legacy families and their reasons. An inventory entry
with unreviewed legacy effects is deliberately not an exposed effect contract.
Existing CLI spellings and JSON remain supported unchanged.

`omabit --config-dir REGISTRY call OPERATION_ID --input-file FILE` invokes the
same handler as the matching MCP tool `omabit.OPERATION_ID`. Input is bounded
to128KiB and results to2MiB. New calls use a version1 envelope with either typed
`data` or a structured `error`; legacy commands retain their old shapes. Schemas
are inferred from the actual input/output and protocol Go types with the same
schema library as the official MCP SDK. Semantic invariants stay in typed
validation and authoritative runner admission. Large integer bindings remain
exact; they are never decoded through float64 before typed dispatch.

Each catalog descriptor declares its trusted `timeout_ms`. CLI and MCP enforce
the same bound, preserving an earlier caller deadline. Most calls allow90seconds;
bounded desk reads, deployments, app phases and source uploads allow their native
RPC deadline plus15seconds for identity/capability checks. Tool arguments cannot
extend these bounds. A timed-out mutation remains uncertain and requires lookup
with its original IDs; a transport deadline never authorizes redispatch.
The runner's `app_journal_uncertain` error on a mutation also produces
`outcome_uncertain`, retaining that original error code. A received error does
not establish that dispatch was rejected: the exact phase may already be
acknowledged. Read its original saved receipt before read-only reconciliation.

The current shared catalog covers selected runner identities/Hello, saved project
inventory/status/recovery source/checkpoints, Stop diagnosis/reconciliation,
offline fake-project planning, project reservation/lifecycle/coordinated recovery,
application preflight/policy, desk discovery/settings/status/receipt/reconciliation,
and existing desk review/plan/create/commit/adopt/release/frontend/receiver
primitives, bounded creation handles and local source staging from explicitly allowed roots.
Frozen framework metadata and exact local source verification are offline shared
operations; see the [versioned Grubbery contract](grubbery-adapter.md). They do not
install the framework or establish live adapter readiness.
Ship routes are read-only. Sanitized snapshot/watch/log exports, broader private
projects, named qualification, credentials, policy authoring, destructive
maintenance and host installation remain explicit exclusions or later slices.
This is not a claim of complete issue19 parity.

`project.stop-diagnostic-bundle.v1` derives a bounded shareable view from the exact
saved operator diagnosis. Its fixed categories/messages omit private bindings,
actor/request/member names, free-form diagnostics and sealed reviews. It probes
no runtime and cannot authorize recovery. Unsupported categories cannot reflect
arbitrary runner text into the export. Read the separate private diagnosis/review
for action bindings; sharing output remains an explicit owner decision.

Start a local server with:

```sh
omabit --config-dir REGISTRY mcp serve --transport stdio \
  --allow-runner NAME:RUNNER_ID
```

Repeat the allowlist flag for each authorized enrollment. An empty list denies
all live targets. Read tools are the default. `--enable-mutations` enables the
typed mutation tools; it does not grant operator role or application permission.
Every call carries its own explicit target. There is no shared mutable selection,
CLI subprocess parsing, runner database access, generic shell/Dojo/SQL/Docker
escape hatch or login-code tool. Startup performs no enrollment or deployment.
Registry reads can acquire/create their private local lock file, but never enroll
a runner. An unavailable registry is refused instead of created.

For shared source staging, add `--allow-source SOURCE_NAME:/absolute/source` to
`mcp serve` or the individual `call`. Each root is opened and pinned at process
startup. Discovery lists only names; calls select a relative ordinary subtree.
Links, parent traversal and arbitrary absolute tool paths refuse. Source tree and
bundle limits remain8192 files,32MiB per file and256MiB total. Planning reads the
source and runner settings; upload writes the runner cache/journal, with no Clay
commit or Unix mount. `desk.plan-stage.v1` returns a versioned source selection
and the unchanged native stage plan. Submit both through `desk.stage.v1`. Preserve
the exact stage ID/digest after a lost reply and read `desk.stage-status.v1` without
requiring the local source again. Definite pre-upload refusals are distinguished
from transport uncertainty. A replaced root pathname cannot rebind a running
server's directory handle.

`desk.export-source.v1` reads at most64explicitly selected files/256KiB from the
exact observation returned by `desk.inspect.v1`. It requires operator role and
`desk-source-export-v1`; binary content is canonical base64 with its SHA256.
Process, pier, revision/hash and selected paths are checked independently. This
is private source content, not sanitized diagnostics. It changes no ownership
and grants no commit permission; source adoption remains separately gated.

Guided creation shares its coordinator and deterministic phase IDs with the
legacy `urbit-collector project-create` progress loop. Call `project.plan.v1`
first, then retain the complete canonical plan, target and parent request ID.
`project.create-begin.v1` checks all required adapters before reserving only;
`project.create-advance.v1` submits at most one missing native phase, with a
90-second client bound. Advance additionally requires the exact
`native_request_id` returned by begin or saved status. Repeating that same call
after a phase finishes returns its historical `completed` handle; it cannot
become consent to the next phase after a lost reply. Read status and explicitly
select its next ID to advance again. Missing, foreign or out-of-order phase IDs
refuse before phase dispatch. The legacy progress loop keeps its original IDs
and NDJSON behavior. Cache preparation can download only the compiled pinned
runtime inputs before the first prepare phase. Pending, failed and interrupted
phases return their original receipt instead of being resubmitted. After a lost
reply, use `project.create-status.v1` with the unchanged selection; its original
reservation lookup cannot create a missing project. Historical `ready` describes
initial receipts, not current fleet or application health.

An optional declared `workflow` (`gall` or `grubbery`) and `app_effects` run the
selected app preflight before reservation or advancement. Missing app APIs,
adapters or role refuse before project creation. Omit the workflow for generic
project creation. This advisory preflight never grants later app policy, and
every actual app effect still has its own admission checks.

`project.create-wait.v1` polls saved receipts for up to30000ms; zero requests one
bounded read without waiting. It never advances a phase. Cancelling this wait or
disconnecting MCP leaves runner-owned work intact. Full history, actor, revision,
plan and derived request bindings must match; incomplete, foreign or unexpected
history refuses further advancement. This does not introduce a second workflow
journal or require optional protocol task extensions.

The server uses pinned official Go SDK1.8.0. Its tested negotiated revisions are
2024-11-05,2025-03-26,2025-06-18 and2025-11-25; the SDK's newer2026 protocol is
not advertised yet. The stdio input frame is bounded. Stdout carries only protocol
messages; diagnostics use stderr. HTTP deployment and actual T3 Code, Claude Code,
Codex/ChatGPT and OpenCode connection qualification are separate pending gates.
SDK fixture interoperability does not establish those clients' installed support.

Queued/running receipts are operation handles, not completion. Read saved status
with the same IDs. A lost response produces uncertainty and supported lookup
guidance; the facade never retries an effect or invents request IDs. Cancelling
a wait/request or closing the MCP process does not cancel runner-owned work.

Desk reconciliation returns an optional `reconciliation_admission` status:
`busy` carries the unchanged saved receipt and queues no new observer; `queued`
admits one bounded read-only observer and still carries saved evidence. Neither
is a fresh observation or native-effect completion. Poll the saved operation,
then explicitly request reconciliation again only while uncertainty remains.
Long lifecycle/native work does not block this admission response, and repeated
polls cannot build an observer backlog. Completed receipts replay outside that
admission guard. Current operator authentication, management/failed-store fences
and each worker's actor/runtime/pier/configuration checks remain authoritative.

Keep exact sealed review JSON from the text result when saving a request: clients
must not round large integers and reserialize a changed review. `lookup_only`
repeats the original request through the existing guarded transport. A matched
Stop reconciliation lookup neither probes nor reapplies; the original interrupted
receipt remains separate from its linked resolution.

Error envelopes expose fixed guidance and the runner's typed diagnostic code,
without reflecting arbitrary transport/runtime strings into transcripts. Fresh
desk observations and policy checks are still required before each effect. Saved
workflow check names are descriptive metadata; release/frontend check arrays are
caller-supplied evidence, not executable jobs or proof that Omabit ran tests.

To serve skill guidance, configure global `--skills-bundle DIR` and
`--skills-manifest-sha256 SHA` from a previously verified matching bundle.
`BUILD.json` and `SKILLS.json` must agree; the exact manifest hash, complete
ordinary Markdown inventory and every content hash are checked. A hash computed
from an arbitrary unverified directory does not establish release authenticity.
Resources use `omabit://guidance/skills/...`; `guidance.list.v1` and
`guidance.read.v1` offer the same bounded bytes for clients without resource UI.
Each read rehashes its pinned content. Reading installs nothing and grants no
authority. A standalone development binary without a bundle reports guidance
unavailable rather than borrowing an installed production skill tree.

Tests exercise canonical schemas, inventory coverage, wrong/absent allowlists,
role/capability/policy refusals, error sanitization, precise uint64 bindings, all
four protocol revisions and verified resource/tool equality. Actual CLI dispatch
and stdio subprocesses share a policy fixture. An actual built Omabit binary
submits clean-stop recovery through MCP, loses the reply, then reads the same
receipt through CLI after manager/store reopen with no runtime observer backend.
This uses real API/SQLite and an inert runtime observer, not a live Urbit. CI runs
that test explicitly with built helpers. Disposable real-runtime and matching DEV
deployment/upgrade gates remain required before any installed-support claim.

The staging parity fixture uses the actual CLI argument dispatcher and official
SDK stdio subprocess: CLI plan → MCP upload with deliberately discarded response
→ CLI lookup of the same durable verified cache receipt. Changed-source resubmit
refuses before a second upload. Directory replacement, links and traversal have
separate confinement tests. This fixture is inert; the native Gall gate composes
the existing real collector/staging path with a fresh kernel separately.

A separate actual CLI/stdio fixture reserves through CLI, deliberately loses an
MCP phase response, closes MCP, reads the original handle through CLI, waits and
continues through a new MCP process. App preflight refuses before reservation;
generic creation works. Pending/interrupted phases cause no resubmission. Core
race fixtures reconstruct the coordinator at every phase and reject altered or
truncated history; cancellation/deadline tests preserve queued work. These use
an inert protocol server, not installed-runtime durability evidence.

The existing0.4.0 feature-based read contracts remain usable. New schema60 Stop
recovery requires its advertised feature and a compatible runner; older runner
absence is a refusal, never an upgrade inference. No stable version, frozen pin,
installed support window or production deployment is changed by this source slice.

Development adds `app.source-operation.v1` to generic `omabit call` and stdio
MCP. It requires the selected runner's separate `grubbery-app-receipts-v1`
feature and current operator access. This is saved metadata only: original
creator, plan, phase history and uncertainty remain unchanged. Lookup uses a
single validated SQLite snapshot outside the lifecycle mutex and requires no
source cache or native adapter. An operator other than the original creator can
read it; this grants no mutation authority on that creator's operation. Missing,
changed-runner and invalid-history responses refuse. Startup runner allowlists
still apply. This feature does not advertise live Grubbery workflow readiness.

Development source APIs also expose `app.review-source.v1`,
`app.begin-source.v1`, `app.advance-source.v1`, `app.reconcile-source.v1` and
`app.abandon-source.v1`. Their separate `grubbery-app-source-v1` feature means
these typed endpoints exist; full workflow preflight remains unavailable.
Review derives the actor, policy, verified framework receipt/baseline and pier
continuity on the runner. The caller selects an exact runtime/target, private
stage, app ID and prospective request ID. Save the returned plan unchanged.
Begin persists its namespace lease without a native app effect. Advance requires
both original plan digest and phase ID and can dispatch that phase only once.
Reconcile needs no source bytes and never dispatches. Abandon retains live/source
state and releases only proven preactivation work; unknown activation stays fenced.

Replay reads precede live admission, so an original begun/dispatched/observed
handle remains usable after cache loss or Commit revocation. New effects still
recheck current authority and exact bindings. HTTP requests are bounded to96KiB;
restricted SSH checks the feature again on the delegated connection and retains
ordinary4KiB request limits elsewhere. Calls have four-minute application bounds.
An accepted exact app journal/phase call belongs to the runner, with that native
bound and runner shutdown context. A disconnected client stops waiting without
cancelling the accepted phase. Read-only review retains its caller context.
Saved lookup remains independent of the lifecycle mutex; this does not add a
background queue, extend native deadlines or authorize the next phase.
MCP needs explicit mutation enablement for every journal write, including begin,
reconciliation and abandonment. Recovery currently requires the original actor
and exact original process/configuration; same-pier restart or key rotation does
not implicitly rebind an unfinished operation. Native Docker transport, broader
workflow/recovery and installed qualification remain required.

Schema63 development adds `app.review-runtime-recovery.v1`,
`app.apply-runtime-recovery.v1` and `app.runtime-recovery.v1` to shared CLI/stdio
MCP. The [runtime recovery contract](app-source-runtime-recovery.md) binds a new
immutable audit to exact original history and a separately verified replacement
process. Review has no journal/app effect. Confirmed apply reobserves every fact
and writes only its exact audit CAS, leaving the original phase uncertain. Later
original reconciliation is a separate operation and never redispatches it.
Saved lookup needs no source, runtime or management admission and validates
original history/audit together. Current operator access and startup target
allowlists still apply; review/apply also require the original actor, and apply
needs mutation enablement. Preserve original operation and audit request IDs.

`grubbery-app-runtime-recovery-v1` means the typed API is compiled, including
saved exact apply replay when a native adapter is unavailable. Its separate
`grubbery-app-runtime-receipts-v1` feature denotes saved-only lookup. Neither
qualifies the full workflow. HTTP/SQLite, exact CLI/official-SDK stdio parity and
bounded delegated SSH source gates pass. The matching bc04706 built-client
[lost-reply/manager-replacement source rehearsal](diagnostics/2026-11-18-app-runtime-audit-cli-mcp-source.json)
also passes with real SQLite/API and inert native observation. Real native/
installed qualification remains required. Same-pier restart remains explicit review/apply; key rotation and
legacy plan1 rebinding are unsupported. No production deployment is implied.

The [frozen d2d8cf9 public native gate](diagnostics/2026-11-18-grubbery-public-native-workflow.json)
passes the actual runner source API and pinned registry/client against a fresh
pinned kernel. Both source versions are independently compiled and followed;
saved state survives the update, the new code label is observed, an exact owned
same-pier restart is clean, and another state write succeeds. Both original
SQLite receipts survive reopen and old-phase replay unchanged. Each original
phase is dispatched once. This uses synthetic lifecycle and direct native owner
HTTP, so product Docker/curl, pending-operation process rebinding, durable owner
cleanup on process death, the native CLI combination, permits/routes/assets,
peers and installed upgrades remain unqualified. Full preflight stays unavailable.
