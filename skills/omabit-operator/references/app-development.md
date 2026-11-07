# App development and publishing policy

The 0.4.0 source implements policy storage, Setup authoring and read-only
`app-policy` status. Desk commit/publication commands and the Urgit adapter are
still unimplemented. Verify the selected installation's capabilities and command
help. Until a supported operation exists and fresh policy permits its effects,
continue with source preparation and available inspection.

## Read the configured policy

Resolve the runner, ship instance and desk. Project targets also require their
project/member identities. Read **Setup → App development and publishing** through
the supported policy-status interface. Inspect independent commit/publication
permissions, their effective values, originating scopes, revisions and freshness.
They inherit global → runner → ship → installed desk, with the most specific
explicit Allow or Block winning. The global default is Block for both permissions.

Use the saved enrollment, without changing its selection:

```sh
urbit-collector --config-dir REGISTRY --runner NAME --expected-runner-id ID \
  app-policy --instance INSTANCE --desk DESK
```

Requires `app-development-policy-v1`. The response includes the policy revision,
effective values and source scopes; project targets include their creator and
generation. Missing/unsupported/unresolved status grants nothing. These are saved
policy observations, not fresh publication or compilation evidence. A global
rollout may be pending on an offline runner; read the selected runner's actual
policy before any supported effect. New runner enrollment does not apply an old
collector rollout automatically.

Read the application's release configuration and relevant repository instructions.
Follow the developer's configured branch, checks and release sequence. For Urgit,
use its existing binding and release rules. A development label or fake identity
does not grant commit or publication permission. Keep user-controlled policy
changes in the Setup flow. Report the blocking scope when action is disabled.
Do not invoke `omabit app-development-save`/`app-development-retry`, write policy
files, submit raw policy-update RPCs, or use Dojo/Eyre/Git as a workaround for a
blocked Omabit operation. Those helpers belong to user-facing Setup authoring.

## Classify the requested effect

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
