# App development, rollout and publication for 0.4.0

Proposed architecture, recorded 2026-11-06. The user added this work to the
0.4.0 scope after remote-runner installation testing. Source review is complete.
The [policy layer and Setup editor](app-development-policy.md) are implemented
and fixture-tested. Gated creation and live desk discovery are implemented;
source transfer/commit/publication remain in progress. Full managed-runtime
qualification is pending.
The user's follow-up fixes the scope to Urbit desks and their frontend assets.
Agent desk creation, commit and publication start disabled, with permissions configured in
Omabit Setup. Each application retains its developer's release configuration.

## Product workflow

An agent should be able to prepare a desk, stage its source, inspect pending
changes and, when the applicable gates permit it, commit and publish. Follow the
application's configured release steps and verify its results on selected
receivers. Support ordinary managed ships, standalone development fakes and
members of fake project networks through the same operation model.

Provide two source paths with shared status and release records.

| Source path | Intended use | Omabit's responsibility |
| --- | --- | --- |
| Mounted desk | Local files, an assembled build directory, development iteration | Create and mount the desk, transfer a reviewed file set, commit, observe compilation and activation, publish and check receivers |
| Urgit | Ordinary Git pushes to a configured ship/repository/branch | Document and qualify the workflow using Urgit's own tools; observe Git/Clay agreement and verify publication/receivers |

Start with the mounted-desk path. Urgit is an optional, tested skill-guided
workflow; the user explicitly declined a dedicated Omabit software integration. Both
paths support local and remote runners. The application build remains a developer
workspace action, with its output frozen before transfer. Omabit can record the
source commit, dirty-tree digest and build result supplied by the developer.

## Existing foundations

Owner-user rootless execution already makes mounted desk files writable by the
runner's Unix owner. The current managed CLI exposes lifecycle and runtime
settings. A supported remote desk-file deployment API still needs implementation.
[Owner-user file access](owner-user-runtime.md)

Private publisher experiments already exercised desk creation, mounting, copied
source, commit, local install, publication through Clay read permission and
installation on a receiver. They preserved ship-generated files and checked app
readiness separately from the install acknowledgement. Their Lens trials also
show why the correct mark must accompany a typed payload.
[Publisher/receiver evidence](diagnostics/2026-10-24-tend-backend-runtime.md),
[Lens mark correction](diagnostics/2026-10-24-tend-publisher09-mark-failure.md)

Reusable code includes `internal/eyre` acknowledgement handling,
`internal/shipaccess` credentials, existing artifact-upload patterns,
`internal/managed` operations and `internal/projectnetwork` member identity and
lifecycle checks. `internal/runtimequalify/project_fleet_linux.go` already selects
a project member's private HTTP port and verifies its identity inside the correct
container. These supply implementation patterns with bounded, existing uses.

## Urgit findings

Reviewed upstream commit `7834a4e186998265f5175fd527bc17ea8819c44f`.

- Smart HTTP serves ordinary Git clients. A branch can bind to a Clay desk, and
  the linked Git update waits for Clay acceptance. The bridge reports the Git
  commit, Clay revision and content drift. Its `publish-desk` action exports
  Clay contents into Git. [Bridge architecture](https://github.com/yapishu/urgit/blob/7834a4e186998265f5175fd527bc17ea8819c44f/specs/architecture.md)
- Administration uses an authenticated ship session. Git uses a repository write
  token through HTTP Basic authentication. Its native UI uses HTTP endpoints
  for repository creation, binding and status. Its native `git-action`
  mark currently provides noun conversion. [HTTP client](https://github.com/yapishu/urgit/blob/7834a4e186998265f5175fd527bc17ea8819c44f/fe/src/api.js),
  [mark definition](https://github.com/yapishu/urgit/blob/7834a4e186998265f5175fd527bc17ea8819c44f/desk/mar/git-action.hoon)
- Binding selects a branch and desk. Whole-tree projection means a repository
  containing `desk/` needs an assembled deployment tree with desk files at its
  root. Some Git filenames are omitted from Clay projection, so expose the
  projected file manifest for inspection. [Binding type](https://github.com/yapishu/urgit/blob/7834a4e186998265f5175fd527bc17ea8819c44f/desk/sur/git.hoon),
  [tree projection](https://github.com/yapishu/urgit/blob/7834a4e186998265f5175fd527bc17ea8819c44f/desk/lib/git-clay.hoon)
- Incoming GitHub push hooks record notices for an explicit pull. Updating a
  GitHub import currently rejects a Clay-bound destination. Direct pushes to the
  ship are the first integration target. Agent reload clears pending Clay jobs,
  making restart reconciliation an acceptance case. [Agent implementation](https://github.com/yapishu/urgit/blob/7834a4e186998265f5175fd527bc17ea8819c44f/desk/app/urgit.hoon)

Treaty publication is an additional Omabit workflow. Upstream's inspected Git
publication path updates repository state. Test Treaty capability and metadata
on the selected ship before offering app discovery publication.

## Setup permissions, ship intent and desk bindings

**Setup → App development and publishing** now provides independent permissions
for **Create new desks**, **Commit to Clay** and **Publish or distribute updates**. All default to
blocked for existing ships, new ships, development fakes and project members.
The user enables agent access through this section.

Creation inherits global → runner → ship; a desk cannot grant creation of its
siblings. New desks inherit commit/publication permissions unless explicitly
overridden. Resolve commit/publication from global → runner → ship → installed desk. Scoped
entries offer Inherit, Allow and Block. The most specific explicit value wins,
with Block as the global default. For example, allowing one desk permits that
desk while its siblings inherit the global block. Show the effective result and
the scope supplying it beside each ship/desk and in read-only CLI status.

The runner stores and enforces its policy, shared by every collector. Global
changes roll out to the enrolled runners, with per-runner applied/pending state.
An offline runner retains its last acknowledged policy until it receives the
change. Report that condition in Setup. New runner enrollment begins blocked.
Use versioned policy updates, preserve overrides during global changes, and
detect concurrent edits against the runner's current policy revision.

Enforce gates at the runner's mutation boundary. The agent CLI can inspect
policy and report the precise blocking scope. Permission changes use the
user-facing Setup path. Agent requests, ordinary settings edits, convenience
deployment commands and acknowledgement flags cannot grant creation/commit/publication
permission to themselves. Recheck effective policy immediately before every
effect, including queued work and retries after a restart. A policy change
fences future effects while completed stages retain their receipts.

The policy tree highlights effective permissions and matching paths while keeping
unrelated branches visible. **Discover desks** reads the selected running ship
using `desk-discovery-v1`; it marks installed desks without granting permissions
or discarding unsaved edits. The observation belongs to the exact runner and
project generation, is temporary, and clears on refresh/profile change. Saved
desk scopes remain visible without querying Clay.

Apply permissions to the effects of an operation. Creation requires the ship
gate and fresh evidence that the exact desk does not already exist. A commit to a published desk
requires commit and publication permission because subscribers can receive that
revision. An Omabit-initiated push to a Clay-bound Urgit branch has the same
requirements. Include equivalent resync, merge, generic control and composite
deployment entry points in enforcement. Source writes into an auto-committing
mount also require the permissions their resulting effects would use. When the
auto-commit or publication state is unresolved, keep files in staging and report
the unresolved evidence.

The enforcement boundary covers Omabit's agent-facing operations; its skills
also honor the gates when guiding native Git workflows.
Human-operated Dojo and independently configured Urgit clients use their native
authority. An unrestricted process with Unix-owner access can reach other host
interfaces. Skills must honor a blocked gate across alternate transports, and
the UI should describe this scope accurately.

Ship purpose and desk workflow settings are now editable in the selected tree
node's card and readable with `desk-settings` (`desk-workflow-settings-v1`, M541).
Schema51 stores them separately from policy, with revision checks and author
history. Purpose defaults to standard; a desk defaults to Unix source named after
the desk, no publication method and no release checks. These are descriptive
defaults, not verified mount mappings. Settings include source mode/directory,
optional Urgit repository/branch, publication method, checks and agent instructions.
They contain no executable hooks, scheduler or tokens. Urgit settings document the
native configuration without changing or verifying it. All three permission gates
remain independent. Persist policy and bindings against runner/instance
identity. Project members additionally carry project identity and current member
generation, so repeated fake ship names resolve correctly.

A desk binding records its desk, source directory or Git source, source mode,
expected baseline, optional Urgit repository/branch, publication method, the
developer's release configuration and chosen receiver checks. Urgit retains its native binding/disable and credential lifecycle; Omabit does
not create a separate adapter or credential store.

Use a versioned migration and feature negotiation. Older clients' ordinary edits
must preserve the new fields. Stopped host transfers retain explicit intent and
desk bindings, then recompute permission against the destination runner's policy.
Require explicit enablement when the transferred permission cannot be reconciled.

## Uncommitted-change alerts

For mounted desks, compare the current Unix tree with the mounted representation
of the current Clay revision, including additions, edits and deletions. Observe
files changed by editors and direct copy as well as Omabit uploads. Reuse Clay's
mark rendering to avoid false differences from canonicalization.

Display **Uncommitted changes** in the ship/desk view, with the desk name, changed
file count, observation time and a details action. Expose the same structured
status to agents. Offer a Commit action only under the effective policy and the
application's release settings. The alert itself performs observation.

The source implementation uses SHA256 of jammed [mark noun] values and the
selected historical desk's actual MIME converters, in bounded batches. Failed
conversions are unavailable comparisons, never synchronized results. The typed
commit path targets the named Clay desk with reviewed revision/hash and a MIME
delta; it does not rely on a Unix mount name to identify the destination. The
runner source-status API, synchronization baselines and selected-desk UI alerts
are implemented (M542), using the saved descriptive source configuration (M541).
`desk-source-status-v1` and `omabit desk-source-status` return exact counts and at
most256 file details. Source bytes and absolute paths are not returned. Only a
confirmed match seeds the schema52 baseline, which is invalid for a changed
runtime/configuration, project generation, workflow revision or directory name.
The selected Setup desk refreshes90 seconds after each bounded read completes,
while active, and supports manual refresh. A failed comparison retains prior
details with an unavailable/stale label. Unsaved workflow edits and profile
switches invalidate that view. There is no background fleet-wide source watcher
in this slice; a selected-desk refresh reconstructs pending edits after restart.

Use bounded reconciliation and refresh on reconnect; broader debounced file
observation remains a possible later optimization.
Separate synchronized, Unix changes, Clay changes, divergence and unavailable
states. Restarting a collector must rediscover pending changes, and an offline
ship or failed comparison must retain an unavailable/stale indication. Clear an
alert only after the observed trees match. During a commit, track its exact source
digest so later file saves remain visible as new pending changes.

For Urgit bindings, use its Git/Clay bridge status and the developer's existing
branch/release configuration. Show pending or divergent work in those terms.

## Mounted-desk execution

The complete first-use sequence is prepare desk, mount, stage files, apply the
file changes, commit, activate locally, verify readiness, publish, and verify a
receiver. On later deployments, observe the desk's existing installation and
publication state to determine the effects of a commit.

1. Resolve the registered pier and exact running instance. Inspect the desk,
   mount, Kelvin, source subscription, agent state and publication state. An
   existing desk requires reconciliation with its current contents and source.
2. Create the desk using the ship's supported new-desk operation, then mount it.
   Preserve the minimal marks and Kelvin file generated from that ship. Reuse the
   existing source-reviewed publisher templates when their capability checks pass.
3. Freeze a source manifest and content hashes on the desktop. For a remote
   runner, upload a bounded artifact over the restricted management transport.
   Anchor writes to the selected mounted desk. Validate regular files, paths,
   ownership and byte limits before applying changes. Exclude pier state and
   credentials from the payload.
4. Show additions, replacements and deletions in the plan. Delete only files
   tracked by this binding whose prior contents still match. Preserve untracked
   edits and generated prerequisites. Journal partial file application and
   reconcile it before committing. Check both the mounted files and Clay baseline
   again so direct editor saves or another desktop cannot be silently overwritten.
5. Read the effective gates and the application's release configuration before
   submitting the typed commit operation. When blocked, retain the staged or
   uncommitted work and report its status. After an allowed commit, observe the
   resulting Clay revision.
   Install locally on first activation when requested, then observe Kiln/Gall and
   app-specific readiness. Record compilation rejection, activation failure and
   successful readiness separately.
6. Resolve frontend delivery from the desk's declared layout. Support desk-served
   assets and existing docket/glob metadata. A required glob upload has its own
   result and must complete before claiming the frontend is ready. Apply the
   publication gate when that upload changes an already distributed frontend.
   Confirm the runtime-specific upload adapter in the first implementation spike.

Prefer typed operations through the runner's verified private HTTP/Lens access
for managed ships, reusing existing container/port selection and typed payloads.
Use existing Eyre clients for supported ship APIs. This keeps the workflow usable
with native control disabled, including current project-member launches. The
pinned runtime's documented `conn.sock` disconnect defect makes native control
unsuitable as the default transport for this feature.
[Runtime transport constraints](development-runtime.md)

Compilation or a successful Git push establishes one stage. A release result
also needs the declared agents and frontend checks. Arbitrary app tests can run
in the developer's environment and attach results to the frozen source digest.
Application state migrations need app-specific upgrade testing and recovery.

## Release and rollout behavior

Clay subscriptions propagate subsequent changes from their selected source desk.
The release boundary therefore sits before mutation of that desk. For an
explicit-promotion workflow, develop and test on another desk or ship, then
promote the exact tested artifact or commit to the distributor. Initial Treaty
publication establishes discovery, and later distributor commits may update
subscribers immediately. [Clay distribution](https://docs.urbit.org/build-on-urbit/core-academy/ca10)

Follow the developer's configured release gates, including Urgit's configured
branch and validation behavior. Explicit promotion and automatic release can
both be valid application configurations. Omabit's default-closed agent gates
determine whether its CLI may perform a requested effect. The application owns
the release sequence after permission is enabled. Native probe19 also confirms
that Clay can accept raw invalid Hoon source: commit success alone is not a
compilation or Gall-readiness result. Report the checks actually
observed, including any app-specific tests required by that configuration.

For desks with a docket, provide Treaty publication. For source-only desks,
offer explicit Clay read publication and named `|install` on receivers. Read
the selected ship's supported protocol and check the observed result.
[Software distribution guide](https://docs.urbit.org/build-on-urbit/userspace/dist/software-distribution)

Verify rollout on explicitly managed receivers through their configured source,
observed revision and app readiness. Fake-network publication is confined to its
project topology. Public distribution uses a real publisher reachable through
Urbit's networking. Clearweb configuration is independently useful for Git HTTP
and browser access. It is a separate operation from making a desk distributable.

## Urgit workflow (scope clarified 2026-11-06)

Support and test Urgit's intended workflow through the bundled operator and
plugin-development skills. Do not add an Omabit-specific Urgit adapter, setup
RPC, token store or Git credential helper. Existing Urgit tooling owns its
repository, branch binding, authentication and Git/Clay validation. Follow the
application's release rules and Omabit's effective agent permissions; a direct
Git push is not a workaround for a blocked commit or publication gate.

Qualify the documented path using disposable fakes: install the reviewed Urgit
source, configure a repository and bound desk with its own tools, push a complete
deployment tree, inspect Git/Clay agreement and compilation results, and verify
publication/receiver behavior separately. Cover failed compilation, drift,
interrupted responses and agent reload. Record the tested Urgit revision and
runtime tuple in the skills; do not claim untested GitHub automation support.
For newly created Git repositories, use the user's preferred `master` branch.

## Agent-facing operations and recovery

Use the existing collector CLI and runner operation store. Suggested command
families are `desk-status`, `desk-prepare`, `desk-stage`, `desk-commit`,
`desk-deploy`, `desk-publish` and `desk-install`. Expose
individual steps for agents as well as a composed deployment. All names remain
provisional.

Each write binds the runner, ship instance, optional project/member, desk,
configuration and effective policy revisions, source digest and request ID.
Return durable operation receipts with stage, result and evidence. Persist
intent before dispatch and
serialize effects against concurrent desk changes and instance lifecycle actions.
After a lost response, inspect Clay, Git, installation and publication state
before deciding whether any further mutation is needed. An unresolved outcome
remains visible with its original receipt.

Keep source transfer, Clay acceptance, app activation, publication and receiver
readiness as independently observable stages. Success at one stage can survive
failure at the next. Reconcile existing state rather than rerunning a complete
pipeline after an interrupted terminal or desktop restart.

## Implementation sequence and acceptance

1. Policy storage, inheritance, the runner effect guard and Setup controls are
   implemented. See [policy behavior and evidence](app-development-policy.md).
   Typed creation now uses fresh policy/runtime checks and durable schema50
   receipts. Commit/publication stay unavailable until their adapters are connected
   and tested. Full managed-runtime workflow qualification remains pending.
2. Complete the transport spike on disposable fakes with explicit test policy.
   Exercise prepare, mount, commit, local activation, Treaty publication,
   source-only publication and receiver installation with native control disabled.
   Confirm frontend/glob handling and compilation/activation failure reporting.
3. Workflow bindings, canonical change observation and bounded private source
   transfer/recovery are implemented (M541–M543). Connect typed commit/publication
   actions and durable receipts for ordinary instances and project members.
   Verify alerts after external copy/edit/delete, reconnect, failed commit,
   Clay-side changes and edits arriving during an in-flight commit.
4. Exercise one fake publisher and two receivers. Deploy a first version, verify
   it, then apply an update under the app's release configuration and verify
   receipt on both receivers. Cover wrong project/ship selection, stale revisions,
   local source conflicts, path escapes,
   generated-file preservation, interruption and concurrent lifecycle operations.
5. Test and document the optional Urgit workflow against the reviewed revision. Test successful
   push, failed compilation, initial binding drift, invalid projected files,
   lost HTTP response, agent reload during a push and recovery of Git/Clay
   disagreement, blocked agent pushes and the application's existing
   release rules.
6. Add ship/desk operation status and Agent Help context. Extend
   the operator and plugin-development skills with supported commands and tested
   tuples after implementation. Integrate with fleet URLs as that feature lands.
7. Qualify the 0.3.7 upgrade with commit/publication blocked by default, preserved
   settings and history, older-runner capability handling and matching artifacts.

Agent Help and skill installation are implemented and fixture-tested. This
workflow remains active alongside the shared origin work it can reuse.
Remote-runner distro testing is complete for this stage. Clearweb access remains
in scope. This app workflow covers Urbit desks and their frontend assets. The
user deferred coordinated companion Omarchy plugin releases beyond this scope.
