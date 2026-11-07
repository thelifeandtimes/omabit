# App development and publishing policy

The 0.4.0 source implements policy storage, Setup authoring, read-only status,
typed desk inspection/creation with durable receipts (`desk-create-v1`),
live installed-desk discovery (`desk-discovery-v1`), and runner-shared workflow
settings (`desk-workflow-settings-v1`), and source comparison/alerts
(`desk-source-status-v1`), and private source staging (`desk-source-staging-v1`).
Staged commits are implemented on `desk-commit-v1` runners. Publication adapters
and full managed-runtime workflow qualification remain in progress.
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

Creation does not grant commit/publication permission. Treaty and receiver/front-end
end-to-end qualification remain pending.

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
a synthetic lifecycle backend: it does not requalify Docker/SSH, activation, Treaty,
frontends or receiver readiness. Check installed capabilities rather than assuming
the development source is deployed.

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
