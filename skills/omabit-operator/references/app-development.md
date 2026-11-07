# App development and publishing policy

The 0.4.0 source implements policy storage, Setup authoring, read-only status,
typed desk inspection/creation with durable receipts (`desk-create-v1`),
live installed-desk discovery (`desk-discovery-v1`), and runner-shared workflow
settings (`desk-workflow-settings-v1`), and source comparison/alerts
(`desk-source-status-v1`), and private source staging (`desk-source-staging-v1`).
Staged commits are implemented on `desk-commit-v1` runners. Separate app-stage
observations use `desk-application-status-v1`. Reviewed local activation and
source/Treaty publication use `desk-release-v1`. Explicit existing-source ownership
uses `desk-source-adoption-v1`. Named receiver installation and pending-source status
use `desk-receiver-v1`. Full managed-runtime workflow
qualification remains in progress.
Urgit follows its native workflow; no dedicated Omabit adapter is planned. Verify
installed capabilities and command help rather than assuming development source
is installed. Keep prepared source outside a live mount while its effects are
blocked or unresolved.

## Read the configured policy

Resolve the runner, ship instance and desk. Project targets also require their
project/member identities. Read **Setup → App development and publishing** through
the supported policy-status interface. Inspect independent creation/commit/publication
permissions, their effective values, originating scopes, revisions and freshness.
They inherit global → runner → ship → installed desk, with the most specific
explicit Allow or Block winning. All three permissions default to Block. Creation stops at Ship scope; a desk
cannot grant permission to create siblings. New desks inherit commit/publication
settings unless the user explicitly configured that desk scope.

Use the saved enrollment, without changing its selection:

```sh
urbit-collector --config-dir REGISTRY --runner NAME --expected-runner-id ID \
  app-policy --instance INSTANCE --desk DESK
```

Requires `app-development-policy-v2`. The response includes the policy revision,
effective values and source scopes; project targets include their creator and
generation. Missing/unsupported/unresolved status grants nothing. These are saved
policy observations, not fresh publication or compilation evidence. A global
rollout may be pending on an offline runner; read the selected runner's actual
policy before any supported effect. New runner enrollment does not apply an old
collector rollout automatically.

Read the application's release configuration and relevant repository instructions.
Follow the developer's configured branch, checks and release sequence. For Urgit,
use its existing binding and release rules. A development label or fake identity
does not grant creation, commit or publication permission. Keep user-controlled policy
changes in the Setup flow. Report the blocking scope when action is disabled.
Do not invoke `omabit app-development-save`/`app-development-retry`, write policy
files, submit raw policy-update RPCs, or use Dojo/Eyre/Git as a workaround for a
blocked Omabit operation. Those helpers belong to user-facing Setup authoring.

On capable runners, use the matching profile's `omabit` binary to read both levels
of release configuration (the command precedes these flags):

```sh
omabit --config-dir REGISTRY desk-settings --runner NAME --expected-runner-id ID \
  --instance INSTANCE
omabit --config-dir REGISTRY desk-settings --runner NAME --expected-runner-id ID \
  --instance INSTANCE --desk DESK
```

Ship settings describe its purpose and development instructions. Desk settings
record Unix source directory or Urgit workflow, optional repository/branch,
publication method, release checks and instructions. They are versioned metadata,
not executable hooks, credentials, verified Urgit bindings or permission grants.
A default source directory equal to the desk name is a suggested input, not proof
that a native Unix mount points at that desk. Confirm the source before deployment.
Follow the configured checks; do not change them to make a release pass. Setup's
`app-development-configure` helper belongs to user-directed configuration, not an
agent's automatic release path. These reads resolve the current project generation
even while its runtime is stopped; they do not provide live source/compilation
evidence. Unavailable settings must not be mistaken for an unconfigured release.

## Inspect and create a desk with the supported CLI

Use the matching `omabit` binary from the selected profile. Its argument order
is distinct from the collector: the command precedes runner/ship flags.

```sh
omabit --config-dir REGISTRY desk-list --runner NAME --expected-runner-id ID \
  --instance INSTANCE
omabit --config-dir REGISTRY desk-inspect --runner NAME --expected-runner-id ID \
  --instance INSTANCE --desk DESK
omabit --config-dir REGISTRY desk-plan-create --runner NAME --expected-runner-id ID \
  --instance INSTANCE --desk DESK
```

`desk-list` reads installed names on the exact current ship/project generation.
Setup's **Discover desks** adds the same observation to its permission tree, with
an observation time. Neither discovering a desk nor entering its name grants
permission. Unavailable/older runners return an error, not an empty successful
inventory. A refresh clears Setup's temporary discovery view.

`desk-inspect` obtains a live observation and effective gates. `desk-plan-create`
requires observed absence and returns an exact plan with its request ID, runtime,
configuration, project generation and policy revision. Neither changes a desk or
a permission. Save that plan before submitting it. If the creation gate is Block,
report its scope and leave the user to change Setup.

When authorized, pass the **unchanged full plan JSON** to:

```sh
omabit --config-dir REGISTRY desk-create --runner NAME --expected-runner-id ID \
  --plan-json JSON
omabit --config-dir REGISTRY desk-operation --runner NAME --expected-runner-id ID \
  --id OPERATION_ID
```

An accepted request or `acknowledged` step is not completion; wait for `observed`.
After a lost submission reply, query the saved ID or resubmit the exact original
plan. Never generate a replacement ID as an automatic retry. `blocked` means the
runtime, desk, policy or authority failed a fresh check; inspect again before
planning different work. An interrupted plan is not automatically dispatched.

For `uncertain`, `desk-reconcile` with the same runner and ID requests read-only
inspection of the exact original runtime. Query the receipt afterward. It can
record that creation's desired desk exists; it never repeats creation, and a
missing desk or unavailable runtime leaves uncertainty intact. It does not prove
which external actor created a desk. If reconciliation remains unavailable,
preserve the receipt and ask for review rather than bypassing it with Dojo.

Creation does not grant commit/publication permission. Native Treaty and frontend
workflow observations are tested as described below. Receiver installation has a
supported gated CLI; the native upload agent adapter remains pending.

## Freeze and transfer prepared source

Use a prepared local source directory outside any live mount. Read the ship/desk
settings and follow the configured build/release checks. On an operator connection
advertising `desk-source-staging-v1`, the matching profile's CLI supports:

```sh
omabit --config-dir REGISTRY desk-plan-stage --runner NAME --expected-runner-id ID \
  --instance INSTANCE --desk DESK --source-dir DIRECTORY
omabit --config-dir REGISTRY desk-stage --runner NAME --expected-runner-id ID \
  --plan-json JSON --source-dir DIRECTORY
omabit --config-dir REGISTRY desk-stage-status --runner NAME --expected-runner-id ID \
  --id STAGE_ID
```

Save the unchanged full plan before upload. It freezes target/project generation,
workflow revision, source digest/count/size and request ID. A changed source or
workflow is refused. File bytes stream through the enrolled runner transport into
a private cache outside every pier; they are not command arguments. Staging works
with a stopped ship and closed gates because it neither writes a live mount nor
commits/publishes. Urgit uses its native source workflow instead.

A verified receipt proves only transfer. After an interrupted/lost reply, query the
original stage ID. An exact resubmission can verify the existing cache; it cannot
replace different or corrupt contents. A corrupt/incomplete stage is not usable.
Never treat this receipt as Clay acceptance, compilation or publication evidence.
Bounds: 8192 files, 32 MiB per file, 256 MiB per tree, 2 MiB manifest; 16 retained
stages and 1 GiB of source per runner. Hidden paths, links and unsafe paths are
refused. Prepare only the desk's required files rather than an entire checkout.

When a stage is no longer needed, explicitly discard its cached copy:

```sh
omabit --config-dir REGISTRY desk-stage-discard --runner NAME --expected-runner-id ID \
  --id STAGE_ID --source-digest SHA256
```

This checks the original operator/digest and refuses while a desk operation on
that target remains unresolved. It never deletes original source, a live mount or
Clay. The same selection can finish an interrupted upload/discard; an absent stage
is idempotent success. Malformed metadata, unknown entries and ambiguous duplicate
cache entries are preserved for review. Do not manually clear a runner's cache to
bypass an unresolved operation. A stage no longer available must be prepared again
before a later supported commit can use it.

## Review and commit staged source

Read the application's configured checks and release instructions before planning.
Do not deploy while another process is editing the same desk. The runner serializes
its own operations; native developer tools retain their independent permissions.
On a `desk-commit-v1` runner, inspect the proposed changes, then save a plan:

```sh
omabit --config-dir REGISTRY desk-review-commit --runner NAME --expected-runner-id ID \
  --stage-id STAGE_ID
omabit --config-dir REGISTRY desk-plan-commit --runner NAME --expected-runner-id ID \
  --stage-id STAGE_ID
omabit --config-dir REGISTRY desk-commit --runner NAME --expected-runner-id ID \
  --plan-file SAVED_PLAN_JSON
```

Save the **complete unchanged JSON** from `desk-plan-commit` to the file before
submission. `--plan-json JSON` is also supported for small plans; do not supply both.
Reviews report exact counts and the first256 sorted changes/conflicts. The plan's
hash binds the whole inventory, including omitted details. It also binds the original
stage, exact runtime and project generation, workflow/policy revisions, pier identity,
current Clay revision/hash and raw source digest. Planning grants no permission.

A no-op needs no commit. Resolve conflicts in the intended source/release workflow;
do not delete live files or change gates to force a plan through. Omabit replaces or
deletes only its previously verified owned files that still match. Identical unowned
files can be shared without adoption; different unowned files are conflicts. Unspecified
ship-generated files are preserved. New marks can be introduced in the same delta;
post-commit verification converts source at the resulting revision. One atomic delta
is limited to32MiB of changed source and8192 changes; a larger staged tree can include
unchanged files. Frontend artifacts need their app's separate build/glob flow.

Submission is blocked by default. The runner rechecks all bindings and the effective
policy before one journaled dispatch. A private desk needs Commit; an already published
desk needs Commit and Publish because subscribers may receive the update. Commit does
not enable Treaty, make a private desk public, activate an app or run configured shell
checks. Follow those checks independently and report failures honestly.

Query `desk-operation --id OPERATION_ID`. `observed` proves a newer Clay revision,
normalized source equality and preservation of the rest of the tree. It does **not**
prove Hoon compilation, Gall activation, frontend readiness or receiver installation.
After a lost reply, query the original ID; an exact resubmission returns its receipt
without another dispatch, including after a completed stage was discarded. Never
mint another request ID as an automatic retry. `desk-reconcile` can read and verify an
uncertain commit on the original runtime using its retained stage. It never writes.
Unavailable or contradictory evidence remains uncertain and blocks further effects
on that target. Keep the stage until reconciliation succeeds.

Verified source ownership and the completion receipt are saved atomically. Ownership
survives a runtime restart only for the same physical pier, target generation and
source name. A different pier/source starts without owned paths and does not adopt
existing content. No Unix mount is created or written by these commands. Check the
configured mounted source separately if a developer also uses a Unix/Clay workflow.
These paths have private fixture coverage and a native kernel/API/collector pass
on disposable `~zod` (Vere4.6-5a7614f, pinned pill, probe21/M545). That pass covers
creation, discovery, canonical alerts, staged updates/deletions, generated-file
preservation, lost-acknowledgement reconciliation and published-desk gates. It uses
a synthetic lifecycle backend: it does not requalify Docker/SSH or receiver
readiness. Probe34/M546 additionally covers the app-stage API, native local activation,
source/Treaty publication, and Docket upload with exact served fixture bytes.
Those probes used private effect helpers. The subsequent `desk-release-v1` API
provides gated activation/publication; native Docket upload still has no agent adapter. Check installed capabilities rather than assuming
the development source is deployed.

## Adopt selected existing source

Use `desk-source-adoption-v1` when a mounted-workflow desk already contains files
that subsequent staged commits should manage. Stage only the selected current
files in an isolated source directory. They must already match Clay after mark
conversion. If the intended next version differs, first adopt the matching current
version, then stage the intended update separately. Never treat a source conflict
as permission to overwrite it or change the configured source workflow to evade it.

```sh
omabit --config-dir REGISTRY desk-review-adoption --runner NAME --expected-runner-id ID \
  --stage-id ORIGINAL_STAGE_ID
omabit --config-dir REGISTRY desk-plan-adoption --runner NAME --expected-runner-id ID \
  --stage-id ORIGINAL_STAGE_ID > adoption-plan.json
omabit --config-dir REGISTRY desk-adopt --runner NAME --expected-runner-id ID \
  --plan-file adoption-plan.json
```

Review the selected paths, new/refreshed ownership counts, retained ownership and
conflicts. Details are bounded to the first256 paths; the digest binds the complete
inventory. Save and submit the complete unchanged plan. Adoption records a verified
baseline in the runner only: it performs no Unix write, Clay commit, activation or
publication. It requires Commit permission and also Publish for a distributed desk.
Defaults remain blocked. It cannot grant itself permission or clear an earlier
uncertain effect. A missing, differing or unconvertible selected file blocks adoption.
An identical existing ownership baseline needs no new adoption.

Only selected files are newly owned or refreshed. Unselected generated files stay
unowned, and prior ownership outside the selection keeps its original baseline.
Later external edits still conflict rather than being silently adopted. A future
commit's stage must include every owned file that should remain; omitting an owned
unchanged file means deletion in that commit's review. The same pier, source name,
project generation, runtime, settings, policy and reviewed Clay state are pinned.

Adoption and its receipt are one database transaction, with `observed` meaning the
ownership baseline was saved. It does not advance Clay. After a lost response,
query the original operation ID; an exact resubmission returns the same receipt,
even after the stage has been discarded. Do not invent another ID to retry. Native
probe36/M548 verifies default blocking, one explicitly selected existing file, no
Clay mutation, replay after stage discard and subsequent commits. The lifecycle
backend is synthetic; managed Docker/SSH are separate checks. Installed capabilities
must still be checked before use.

## Install and inspect a named receiver

Use `desk-receiver-v1` for a new receiver desk. Inspect the publisher's exact
current Clay hash first, and follow the application's release configuration.
The receiver ship's Create desk gate and selected desk's Commit gate must both
allow installation. Publish is not required: installation keeps the received
desk private. A desk scope cannot grant Create desk permission. Agent permissions
control these requests; closing them does not cancel an existing Kiln subscription
or stop the publisher's later updates.

```sh
omabit --config-dir REGISTRY desk-review-install --runner NAME --expected-runner-id ID \
  --instance RECEIVER --desk DESK --publisher-ship '~publisher' \
  --publisher-desk SOURCE_DESK --expected-source-hash CLAY_HASH
omabit --config-dir REGISTRY desk-plan-install --runner NAME --expected-runner-id ID \
  --instance RECEIVER --desk DESK --publisher-ship '~publisher' \
  --publisher-desk SOURCE_DESK --expected-source-hash CLAY_HASH > receiver-plan.json
omabit --config-dir REGISTRY desk-install --runner NAME --expected-runner-id ID \
  --plan-file receiver-plan.json
omabit --config-dir REGISTRY desk-operation --runner NAME --expected-runner-id ID \
  --id SAVED_OPERATION_ID
omabit --config-dir REGISTRY desk-receiver-status --runner NAME --expected-runner-id ID \
  --instance RECEIVER --desk DESK
```

For a project receiver, add `--publisher-instance PUBLISHER_INSTANCE` to review
and planning. It must resolve to the selected publisher identity and a current
member of that same project. Another project, an old member generation or a
standalone selector cannot substitute for that binding. Standalone receivers use
Urbit's existing network reachability; a fake name does not establish a network.
The receiver name must have neither a Clay desk nor a pending Kiln subscription.
An Urgit-owned workflow cannot be replaced through this command.

Save the entire plan before submitting. It binds publisher, expected source hash,
receiver generation/runtime/pier, workflow settings and policy revision. The hash
is an initial verification condition, **not a version pin**: native Kiln continues
to follow that publisher. Review a changed source explicitly rather than silently
accepting a different initial hash.

Read the saved receipt after a lost response. Exact resubmission returns the same
receipt; a new plan ID is not an automatic retry. `uncertain` uses the existing
`desk-reconcile --id SAVED_OPERATION_ID` read path. Reconciliation can record the
reviewed private source at a live subscription even after gates close; it never
repeats installation. Missing or changed evidence stays uncertain. A reserved
revision-zero Clay name is pending source, not an absent/free name.

`desk-receiver-status` reports subscription separately from `source_ready`.
Received source includes Clay and application-stage observations; it does not
claim agent compilation, frontend HTTP success or general application health.
Native41/M551 exercises the API/store/collector with two disposable native
receivers, exact initial/update source and frontend bytes, and one lost-ack recovery
after revocation. Its lifecycle backend is synthetic; Docker/SSH and installed
upgrades are not requalified. Perform the developer's actual checks. Public installation never republishes the
received desk, and changing existing subscriptions is outside this initial command.

## Inspect application stages separately

On a runner advertising `desk-application-status-v1`:

```sh
omabit --config-dir REGISTRY desk-app-status --runner NAME --expected-runner-id ID \
  --instance INSTANCE --desk DESK
```

This read works with closed gates. It binds the current runner/process/project
selection and checks that Clay did not change during inspection. The reply reports
source revision/hash, installation state/source, declared agents and their observed
liveness, Treaty availability/listing/hash, and current versus realized Docket
metadata and frontend state. Missing or unsupported observations fail visibly.
A source revision is **not** a compiled-agent revision. A live agent may still run
an older version. A `site` or `glob` state is not an HTTP/app health test. Follow the
configured release checks and verify actual frontend behavior separately. No field
claims that the entire release is ready.

Native probe34 tested a local Ames glob upload through Docket's existing authenticated
HTTP interface. It served the exact fixture HTML and changed `desk.docket-0` in Clay,
advancing the revision and advertised Treaty hash. Classify that upload as a commit,
and also publication when the desk is already distributed. Do not bypass closed
gates with a native upload. The resulting external docket edit correctly conflicts
with older frozen source ownership; reconcile that source instead of overwriting it.
Use the explicit source-adoption workflow below only after staging the updated
source that already matches Clay. A public upload adapter is not implemented yet.

An intentionally invalid agent update in that probe returned HTTP500 and left the
observed revision/hash unchanged. The original dispatch remains uncertain; an
unchanged read alone does not clear uncertainty or authorize a retry. This is distinct
from the earlier uninstalled-desk probe where Clay accepted invalid source. Neither
behavior permits interpreting a commit acknowledgement as compilation success.

## Review, activate and publish a selected revision

Use `desk-release-v1` only after checking the installed runner capability. The
three typed actions are `activate-local`, `publish-source` and `publish-treaty`.
Local activation uses the commit gate, plus publish permission if the desk already
distributes updates; it cannot replace a remote source subscription. Publication
uses the publish gate and must match the desk's saved Setup publication method.
No command edits those gates or selects a different publication method for the user.

```sh
omabit --config-dir REGISTRY desk-review-release --runner NAME --expected-runner-id ID \
  --instance INSTANCE --desk DESK --action publish-treaty
```

Read both ship and desk instructions in the review. Run the configured release
checks on that exact source and prepare a JSON array in their displayed order:

```json
[{"name":"Receiver smoke test","clay_hash":"0v123","passed":true,"evidence":"Exact source tested; report path or concise result"}]
```

The hash above is only an example: use the reviewed `observation.clay_hash`.
Evidence is your attestation; the runner does not execute or independently verify
these checks. Never claim a check passed without its result. Keep credentials,
private source and sensitive logs out of this retained summary. Failed, omitted or
mismatched checks block publication. `activate-local` takes an empty check array,
since installation may be needed to run those checks; it does not attest them.
When no release checks are configured, omit `--checks-file` to use an empty array.

```sh
omabit --config-dir REGISTRY desk-plan-release --runner NAME --expected-runner-id ID \
  --instance INSTANCE --desk DESK --action publish-treaty --checks-file checks.json > release-plan.json
omabit --config-dir REGISTRY desk-release --runner NAME --expected-runner-id ID \
  --plan-file release-plan.json
```

Use the same review/plan sequence with `--action activate-local` or
`--action publish-source`. Save the full plan unchanged. It binds the runner,
process/project generation, physical pier, source revision/hash, current app
observations, policy revision, both ship/desk workflow revisions and check evidence.
Each effect is queued once and rechecks those bindings before dispatch. Already
observed stages return `release_unchanged`; inspect their status instead of forcing
a second effect. Source, workflow or permissions changed since review require a new
review; they do not amend the old intent.

Query `desk-operation --id ID` for the original receipt. Exact resubmission returns
the original receipt even after workflow/policy changes, without another effect.
A lost reply stays uncertain. `desk-reconcile --id ID` reads the original selected
runtime and exact source without repeating activation/publication, including after
gates are revoked. A changed revision, pier, unavailable app protocol or missing
result leaves uncertainty intact. Follow the same review rule for a remaining
uncertain receipt as for commits; never bypass it through another transport.

An observed activation establishes local installation and declared-agent liveness,
not the running code's exact compiled revision or application health. An observed
Treaty publication requires the selected hash in Treaty plus published Clay read
permissions. Neither proves receiver rollout or frontend HTTP correctness.

## Classify the requested effect

- Creating a desk requires the ship’s creation permission and fresh evidence that
  the exact desk is absent. It grants neither commit nor publication, and the
  new desk inherits those settings by default.
- Committing Unix files into Clay requires commit permission.
- Treaty or Clay read publication requires publication permission.
- Updating a published desk can reach subscribers immediately, so it requires
  both permissions. Apply the same rule to Omabit-managed Urgit pushes, resyncs,
  merges and composite deployments that update a bound desk.
- A write into an auto-committing mount can commit or distribute changes. Check
  that behavior before copying files there. Retain prepared files in a staging
  directory while the relevant policy or effect is unresolved.

Honor a refusal across alternate transports. Replacing a blocked desk command
with a shell/Dojo/Lens call, direct Git push, broader credential or policy edit
bypasses the user's gate. Preserve the original target and report the next
user-configurable setting. Ordinary developer-operated tools have their native
permissions, so Omabit policy status describes Omabit's enforcement boundary.

## Observe pending work and results

On an operator connection to a capable runner, compare the saved Unix source
directory with the exact running desk:

```sh
omabit --config-dir REGISTRY desk-source-status --runner NAME --expected-runner-id ID \
  --instance INSTANCE --desk DESK
```

This reads files and Clay, then remembers a shared baseline only if their
mark-normalized contents match. It performs no source write, commit or publication.
`synchronized` means source equality, not compiled/active application readiness.
`unix-changes`, `clay-changes` and `diverged` use the last matching baseline;
`different` has no usable baseline and does not establish which side changed.
Failed reads/conversions are errors, never an empty/clean result. Autocommit can
be active, inactive or unknown; unknown is not safe-to-write evidence.

The baseline is bound to the runner's instance/project generation, runtime,
configuration, workflow revision and source directory. Changing that binding
starts without a baseline. The result includes exact difference counts and the
first 256 sorted file details, plus raw/canonical digests; source bytes are omitted.
The selected Setup desk refreshes while open and offers manual Refresh. After a
failure it labels any prior observation stale. Urgit uses its native Git/Clay
status; this command does not infer a Git binding or treat Urgit metadata as one.

Inspect Unix/Clay status before and after deployment. Report uncommitted additions,
edits and deletions, including changes made directly by the user. Separate those
from Clay-side changes, divergence and an unavailable comparison. An alert is
pending work for the configured release flow. It never supplies permission to
commit automatically.

Before an allowed write, re-read the effective policy and source baseline, bind
the request to the exact target and source digest, and retain its operation ID.
Follow [operations](operations.md) for receipt reconciliation. Distinguish source
transfer, Clay acceptance, app compilation/activation, publication and receiver
readiness. Clay can accept Hoon source that has not compiled; a new revision or
accepted commit alone does not satisfy an application's compilation/release gate.
After a lost response, inspect the original operation and authoritative ship or
Urgit state before another mutation. Check again for edits made during the commit.
