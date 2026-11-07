# App development and publishing policy

The 0.4.0 source implements policy storage, Setup authoring, read-only status,
typed desk inspection/creation with durable receipts (`desk-create-v1`),
and live installed-desk discovery (`desk-discovery-v1`).
Commit/publication, source bindings and Unix/Clay alerts are still being connected.
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

These commands do not grant commit/publication permission. Source staging,
commit, Treaty and receiver/front-end end-to-end qualification remain pending.

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

Inspect Unix/Clay status before and after deployment. Report uncommitted additions,
edits and deletions, including changes made directly by the user. Separate those
from Clay-side changes, divergence and an unavailable comparison. An alert is
pending work for the configured release flow. It never supplies permission to
commit automatically.

Before an allowed write, re-read the effective policy and source baseline, bind
the request to the exact target and source digest, and retain its operation ID.
Follow [operations](operations.md) for receipt reconciliation. Distinguish source
transfer, Clay acceptance, app activation, publication and receiver readiness.
After a lost response, inspect the original operation and authoritative ship or
Urgit state before another mutation. Check again for edits made during the commit.
