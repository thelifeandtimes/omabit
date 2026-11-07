# App development permissions (0.4.0 development)

**Setup → App development and publishing** configures two independent gates:
**Commit to Clay** and **Publish or distribute updates**. Both start blocked.
This applies to existing ships, fresh fakes and project members.

Choose Global, Runner, Ship or Desk. Global offers Allow and Block; narrower
scopes also offer Inherit. The most specific explicit value supplies each gate.
For example, allow one desk while its siblings remain blocked. Saving both
values as Inherit removes that override. Setup shows the saved effective values,
their source scopes and the runner policy revision. Enter a desk's exact Clay
name without `%`; installed-desk discovery is a later workflow addition.

The policy editor and read-only status are implemented. Typed desk operations,
Unix/Clay change alerts, ship-purpose labels, source bindings, frontend deployment
and the Urgit adapter remain part of the [next pipeline steps](app-development-0.4.0.md).
Enabling a permission does not add those commands or modify a running ship.

## Global changes and multiple collectors

The runner stores its effective policy, shared by all collectors. Global saves
record a durable rollout in the selected collector profile, then apply the global
values to its currently enrolled runners. Ship and desk overrides are preserved.
Setup reports each runner's applied or pending state and retains the last
acknowledged global values and revision across later rollout attempts. Offline runners retain
their last acknowledged policy; a pending block has not revoked an offline grant.

**Retry pending rollout** uses the originally reviewed runner revision. If another
collector changed that policy, or the runner was unavailable during the original
review, refresh and save the global settings again after reviewing the current
values. A lost response is reconciled against the runner's actual global values
before retrying. Refresh alone never applies changes. New enrollments are excluded
from older rollout intents and remain at their runner's current policy (blocked
on a fresh runner). Pairing an existing runner preserves its existing policy.

Global settings belong to this collector's rollout; they are not a fleet-wide
consensus service. Another collector can author a newer revision. Its changes
remain authoritative on the runner, and Setup reports differences on refresh.

## Agent status and enforcement boundary

```sh
urbit-collector --config-dir REGISTRY --runner NAME --expected-runner-id ID \
  app-policy --instance INSTANCE --desk DESK
```

Without the target flags, `app-policy` returns the saved policy and target
inventory. Targeted status adds the exact target and effective permissions with
their source scopes. Project members include project ID, creator operation and
container generation. Unresolved identities report blocked. This performs saved
state reads only; it does not inspect Clay, mounted files, Docker or ship HTTP.
It requires `app-development-policy-v1`; unsupported runners fail closed.

`urbit-collector` has no policy setter. Permission authoring uses the dedicated
Omabit Setup helper and operator-authorized RPC. Instance configuration, agent
operation flags and acknowledgement flags cannot grant application permissions.
Agents must honor the configured gates and must not invoke the Setup helper,
edit policy storage, or switch transports to bypass a block.

The runner effect guard reloads policy and target identity under its mutation
lock immediately before an effect. It rejects stale revisions, unknown effects,
unresolved publication evidence and unavailable state. A commit to a published
desk requires **both** gates because subscribers may receive the update. Future
typed adapters must use this boundary for each stage, including retries,
Omabit-managed Urgit pushes and equivalent auto-commit writes. No desk mutation
adapter is advertised by this policy-only capability.

This is an Omabit workflow boundary, not an OS sandbox for arbitrary programs
running as the Unix owner. Human Dojo commands and independently configured Urgit
clients retain their own authority. Application Connections' broad Eyre sessions
are not desk-scoped credentials.

## Storage, compatibility and recovery

Runner schema **48** adds a version-1 policy and revisioned audit history. Updates
use compare-and-swap, record the authenticated actor, and commit atomically.
Ordinary ship settings edits preserve the policy. Successful deletion or import
restoration retires that instance's overrides in the same transaction; reuse of
its management name does not reuse its grant. Project grants bind the member
generation; a replacement requires reviewing its own scope.

The supported development migration is schema47 → schema48, alongside the
existing historical database migration coverage. Fresh migration adds no grants
and performs no runtime operations. Schema47 writers refuse schema48. Use matching
collector/runner artifacts; older collectors can continue supported lifecycle
operations but cannot author app permissions. Unknown/corrupt policy state is
refused rather than reset. The local `app-development.json` rollout journal has
its own version1 schema and private registry permissions.

Recovery preserves the database and rollout journal for inspection. After a lost
save response, refresh before retrying. Do not lower SQLite's `user_version` or
delete policy records to make an older runner start. Reverting requires the
supported stopped-management backup/restore procedure and matching artifacts;
never restore an earlier manager database over newer runtime operations. A full
0.3.7 → 0.4.0 exact-artifact/live upgrade remains release qualification work.

## Verification

Repository fixtures cover inheritance, effect classification, CAS conflicts,
schema47 migration, reopened persistence, ordinary settings preservation, audit
integrity, atomic grant retirement, project generations, revocation before a
queued effect, default refusal, and unsupported/mutating CLI requests.
Collector tests cover offline rollout, retry after reopen, lost response,
concurrent edits and new enrollment. The offscreen QML test exercises all scopes,
saved effective values, observer/offline refusal and stale-profile responses.

```sh
go test ./internal/apppolicy ./internal/managed ./internal/client \
  ./internal/access ./internal/collection ./backend/cmd/urbit-collector ./cmd/omabit
python3 -B scripts/test-app-development-offscreen.py \
  --output /tmp/omabit-policy-ui-UNIQUE
```

The UI harness uses Quickshell in a networkless bubblewrap namespace, private
HOME/XDG directories and an inert helper. No active runners, ships, installed
plugins, services or desktop sockets are used by this qualification.
