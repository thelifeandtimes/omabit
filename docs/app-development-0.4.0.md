# App development, rollout and publication for 0.4.0

Proposed architecture, recorded 2026-11-06. The user added this work to the
0.4.0 scope after remote-runner installation testing. Source review is complete.
The APIs and CLI commands below are proposed, with runtime qualification pending.
Default release policy and companion-plugin scope are awaiting user feedback.

## Product workflow

An agent should be able to prepare a desk, deploy its source, check the running
application, publish from a selected distributor and verify installation on
selected receivers. Support ordinary managed ships, standalone development fakes
and members of fake project networks through the same operation model.

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

## Ship intent and desk bindings

Add a runner-owned **App workflow** setting with `off`, `development` and
`distribution` values. Show it in the ship profile and expose it through the CLI.
It controls Omabit's automation and appears consistently on every collector.

- Existing ships migrate to `off`. New development fakes and project members
  default to `development`, with that choice visible at creation.
- `development` enables desk preparation, source deployment, local activation and
  test-network distribution. `distribution` enables configured release targets.
- Persist settings against runner/instance identity. Project members also carry
  the project identity and current member generation. Repeated fake ship names
  across projects require those additional identifiers.
- A desk binding records its desk name, source directory or Git source, source
  mode, expected baseline, optional Urgit repository/branch, publication method,
  release policy and chosen receiver checks.

These settings govern Omabit operations within the existing operator role.
Unix-owner access and Urgit's own Git endpoint retain their existing authority.
An Urgit pipeline has an explicit disable action that reconciles its binding and
Omabit-managed credential. Changing a profile label alone cannot stop direct
pushes that an existing Urgit binding accepts.

Use a versioned configuration migration and feature negotiation. Older clients'
ordinary settings edits must preserve the new fields. Stopped host transfers
must retain intent and bindings, with credential reconfiguration where required.

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
5. Submit the typed commit operation and observe the resulting Clay revision.
   Install locally on first activation when requested, then observe Kiln/Gall and
   app-specific readiness. Record compilation rejection, activation failure and
   successful readiness separately.
6. Resolve frontend delivery from the desk's declared layout. Support desk-served
   assets and existing docket/glob metadata. A required glob upload has its own
   result and must complete before claiming the frontend is ready. Confirm the
   runtime-specific upload adapter in the first implementation spike.

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

Proposed default is explicit promotion, with per-desk automatic release from a
configured branch available after setup. A developer authorized to push that
branch has the corresponding release authority. Branch protection and a
successful Ford result alone cannot establish application-level tests.

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

After setup, ordinary Git handles pushes and Urgit performs its Clay validation.
Omabit records or observes the resulting commit/revision pair, readiness and
publication state. Poll authoritative state for operation completion. Webhook
events can trigger observation, with signed delivery and duplicate handling if
added later. Begin with direct ship pushes, keeping GitHub-triggered automation
as a separately qualified extension.

## Agent-facing operations and recovery

Use the existing collector CLI and runner operation store. Suggested command
families are `desk-status`, `desk-prepare`, `desk-stage`, `desk-deploy`,
`desk-publish`, `desk-install` and `desk-urgit-configure`. Expose individual steps
for agents as well as a composed deployment. All names remain provisional.

Each write binds the runner, ship instance, optional project/member, desk,
configuration revision, source digest and request ID. Return durable operation
receipts with stage, result and evidence. Persist intent before dispatch and
serialize effects against concurrent desk changes and instance lifecycle actions.
After a lost response, inspect Clay, Git, installation and publication state
before deciding whether any further mutation is needed. An unresolved outcome
remains visible with its original receipt.

Keep source transfer, Clay acceptance, app activation, publication and receiver
readiness as independently observable stages. Success at one stage can survive
failure at the next. Reconcile existing state rather than rerunning a complete
pipeline after an interrupted terminal or desktop restart.

## Implementation sequence and acceptance

1. Complete the transport spike on disposable fakes. Exercise prepare, mount,
   commit, local activation, Treaty publication, source-only publication and
   receiver installation with native control disabled. Confirm frontend/glob
   handling and actual compilation/activation failure reporting.
2. Add per-ship intent, desk bindings, bounded source transfer, typed runner
   actions and durable receipts. Implement the collector workflow for ordinary
   instances and project members together.
3. Exercise one fake publisher and two receivers. Deploy a first version, verify
   it, then promote an update and verify receipt on both receivers. Cover wrong
   project/ship selection, stale revisions, local source conflicts, path escapes,
   generated-file preservation, interruption and concurrent lifecycle operations.
4. Add the optional Urgit adapter against the reviewed revision. Test successful
   push, failed compilation, initial binding drift, invalid projected files,
   lost HTTP response, agent reload during a push and recovery of Git/Clay
   disagreement. Check that direct pushes need only the configured integration.
5. Add the ship-profile control, operation status and Agent Help context. Extend
   the operator and plugin-development skills with supported commands and tested
   tuples after implementation. Integrate with fleet URLs as that feature lands.
6. Qualify the 0.3.7 upgrade with existing ships disabled by default, preserved
   settings and history, older-runner capability handling and matching artifacts.

Agent Help and skill installation remain the next small 0.4.0 implementation
slice. This workflow follows, alongside the shared origin work it can reuse.
Remote-runner distro testing is complete for this stage. Clearweb access remains
in scope. The new app work concerns Urbit desks and their frontend assets under
the proposed scope; coordinated companion Omarchy plugin releases await the
user's scope decision.
