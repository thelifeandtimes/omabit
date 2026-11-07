# App development, rollout and publication for 0.4.0

Proposed architecture, recorded 2026-11-06. The user added this work to the
0.4.0 scope after remote-runner installation testing. Source review is complete.
The [policy layer and Setup editor](app-development-policy.md) are implemented
and fixture-tested. The desk-operation APIs below remain proposed, with runtime
qualification pending.
The user's follow-up fixes the scope to Urbit desks and their frontend assets.
Agent commit and publication start disabled, with permissions configured in
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
| Urgit | Ordinary Git pushes to a configured ship/repository/branch | Prepare the desk and Urgit binding, configure Git access, observe Git/Clay agreement, establish publication and check receivers |

Start with the mounted-desk path and make Urgit an optional integration. Both
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
  token through HTTP Basic authentication. The adapter can use existing HTTP
  endpoints for repository creation, binding and status. Its native `git-action`
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
for **Commit to Clay** and **Publish or distribute updates**. Both default to
blocked for existing ships, new ships, development fakes and project members.
The user enables agent access through this section.

Resolve each permission from global → runner → ship → installed desk. Scoped
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
deployment commands and acknowledgement flags cannot grant commit/publication
permission to themselves. Recheck effective policy immediately before every
effect, including queued work and retries after a restart. A policy change
fences future effects while completed stages retain their receipts.

Apply permissions to the effects of an operation. A commit to a published desk
requires commit and publication permission because subscribers can receive that
revision. An Omabit-initiated push to a Clay-bound Urgit branch has the same
requirements. Include equivalent resync, merge, generic control and composite
deployment entry points in enforcement. Source writes into an auto-committing
mount also require the permissions their resulting effects would use. When the
auto-commit or publication state is unresolved, keep files in staging and report
the unresolved evidence.

The boundary covers Omabit's agent-facing operations and managed Git integration.
Human-operated Dojo and independently configured Urgit clients use their native
authority. An unrestricted process with Unix-owner access can reach other host
interfaces. Skills must honor a blocked gate across alternate transports, and
the UI should describe this scope accurately.

Keep ship purpose as optional descriptive metadata, such as standard,
development or distribution. New fakes can carry a development label while both
permissions remain blocked. Persist policy and bindings against runner/instance
identity. Project members additionally carry project identity and current member
generation, so repeated fake ship names resolve correctly.

A desk binding records its desk, source directory or Git source, source mode,
expected baseline, optional Urgit repository/branch, publication method, the
developer's release configuration and chosen receiver checks. An Urgit pipeline
has its own disable action for the binding and Omabit-managed credential.

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

Use debounced file observation, bounded reconciliation and refresh on reconnect.
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
the release sequence after permission is enabled. Report the checks actually
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

## Urgit adapter

Detect or install the selected supported Urgit release, verify its APIs, create
or reconcile the repository, prepare the target desk, and bind the exact branch.
Record the installed Clay hash and API observations. Upstream names `~matwet` as
its installation source. Establish correspondence with the reviewed source before
claiming that a live installation matches this research. For isolated fake
networks, build the reviewed source in the developer workspace and deploy that
artifact through the mounted-desk path. Include its three declared agents and
frontend assets in qualification. [Urgit setup](https://github.com/yapishu/urgit/blob/7834a4e186998265f5175fd527bc17ea8819c44f/README.md)

Use a full desk tree, including required generated marks and Kelvin-compatible
inputs. A project with `desk/` can export a deployment repository while retaining
its source repository and a recorded source-to-artifact mapping.

Create the first commit on `master` and set Urgit's default branch to it when
Omabit creates a repository for this user. Urgit's current empty-repository API
initially seeds a `main` symbolic head, which the adapter must reconcile.

Store the repository token using the existing credential infrastructure and a
Git credential helper bound to the verified origin and repository path. Show the
chosen Git remote. Reuse the planned fleet-origin registry to discover usable
routes, with explicit reconciliation after an uncertain push response.

After setup, ordinary Git handles pushes and Urgit performs its Clay validation
under the developer's existing release configuration. Omabit-managed agent pushes
check the effective commit/publication policy before submitting an update.
Omabit records or observes the resulting commit/revision pair, readiness and
publication state. Poll authoritative state for operation completion. Webhook
events can trigger observation, with signed delivery and duplicate handling if
added later. Begin with direct ship pushes, keeping GitHub-triggered automation
as a separately qualified extension.

## Agent-facing operations and recovery

Use the existing collector CLI and runner operation store. Suggested command
families are `desk-status`, `desk-prepare`, `desk-stage`, `desk-commit`,
`desk-deploy`, `desk-publish`, `desk-install` and `desk-urgit-configure`. Expose
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
   All desk mutation commands remain unavailable until their typed adapters use
   that guard and pass runtime qualification.
2. Complete the transport spike on disposable fakes with explicit test policy.
   Exercise prepare, mount, commit, local activation, Treaty publication,
   source-only publication and receiver installation with native control disabled.
   Confirm frontend/glob handling and compilation/activation failure reporting.
3. Add desk bindings, bounded source transfer, change observation, typed runner
   actions and durable receipts for ordinary instances and project members.
   Verify alerts after external copy/edit/delete, reconnect, failed commit,
   Clay-side changes and edits arriving during an in-flight commit.
4. Exercise one fake publisher and two receivers. Deploy a first version, verify
   it, then apply an update under the app's release configuration and verify
   receipt on both receivers. Cover wrong project/ship selection, stale revisions,
   local source conflicts, path escapes,
   generated-file preservation, interruption and concurrent lifecycle operations.
5. Add the optional Urgit adapter against the reviewed revision. Test successful
   push, failed compilation, initial binding drift, invalid projected files,
   lost HTTP response, agent reload during a push and recovery of Git/Clay
   disagreement, blocked Omabit-managed pushes and the application's existing
   release rules.
6. Add ship/desk operation status and Agent Help context. Extend
   the operator and plugin-development skills with supported commands and tested
   tuples after implementation. Integrate with fleet URLs as that feature lands.
7. Qualify the 0.3.7 upgrade with commit/publication blocked by default, preserved
   settings and history, older-runner capability handling and matching artifacts.

Agent Help and skill installation remain the next small 0.4.0 implementation
slice. This workflow follows, alongside the shared origin work it can reuse.
Remote-runner distro testing is complete for this stage. Clearweb access remains
in scope. This app workflow covers Urbit desks and their frontend assets. The
user deferred coordinated companion Omarchy plugin releases beyond this scope.
