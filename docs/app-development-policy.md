# App development permissions (0.4.0 development)

**Setup → App development and publishing** configures three independent gates:
**Create new desks**, **Commit to Clay**, and **Publish or distribute updates**.
All start blocked, including on existing ships, fresh fakes and project members.

Select Global → Runner → Ship → Desk in the permission tree. Choose the permission
to inspect, highlight Allowed or Blocked nodes, and search by runner, ship or desk.
Matching paths stay highlighted while unrelated branches dim. An ancestor on a
highlighted path still displays its own permission; the path does not grant it
access. The map shows saved, effective permissions, not unsaved editor choices.
Runner values come from their own acknowledged policy, even during a pending
global rollout. Unavailable runners are not represented as known blocked nodes.

Global offers Allow and Block; narrower scopes also offer Inherit. The most
specific explicit value supplies each gate. **Create new desks** stops at Ship
scope: a desk cannot authorize creating its siblings. Newly created desks inherit
commit/publish permissions unless the user has explicitly configured that desk's
scope. Creation permission grants neither commit nor publication.

The tree includes saved desk policies and any supplied known desk names. Complete
installed-desk discovery is not connected yet. Select a ship and use **Review
another desk's permissions** with its exact Clay name, without `%`, to edit an
unlisted scope. This does not create a desk or verify that it exists.

The policy editor and read-only status are implemented. Typed desk creation and other desk operations,
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
It requires `app-development-policy-v2`; unsupported runners fail closed.

`urbit-collector` has no policy setter. Permission authoring uses the dedicated
Omabit Setup helper and operator-authorized RPC. Instance configuration, agent
operation flags and acknowledgement flags cannot grant application permissions.
Agents must honor the configured gates and must not invoke the Setup helper,
edit policy storage, or switch transports to bypass a block.

The runner effect guard reloads policy and target identity under its mutation
lock immediately before an effect. It rejects stale revisions, unknown effects,
unresolved publication evidence and unavailable state. A create-desk effect also
requires fresh evidence that the exact desk is absent. A commit to a published
desk requires **commit and publish** gates because subscribers may receive the update. Future
typed adapters must use this boundary for each stage, including retries,
Omabit-managed Urgit pushes and equivalent auto-commit writes. No desk mutation
adapter is advertised by this policy-only capability.

This is an Omabit workflow boundary, not an OS sandbox for arbitrary programs
running as the Unix owner. Human Dojo commands and independently configured Urgit
clients retain their own authority. Application Connections' broad Eyre sessions
are not desk-scoped credentials.

## Storage, compatibility and recovery

Runner schema **49** stores version-2 policy and revisioned audit history. Updates
use compare-and-swap, record the authenticated actor, and commit atomically.
Ordinary ship settings edits preserve policy. Successful deletion or import
restoration retires an instance's scoped grants in the same transaction; reuse
of its name does not reuse a grant. Project grants bind the exact creator and
generation.

Schema47 → schema49 creates a default-closed policy. Schema48 → schema49 preserves
commit/publish settings, closes desk creation, retains the exact historical audit
receipt and appends a migration revision. Reopening does not replay migration;
corrupt receipts abort it. Schema48 and earlier writers refuse schema49. Use
matching runner/collector builds with `app-development-policy-v2`; earlier policy
clients are not authorized to write the expanded policy.

The collector rollout journal advances from version1 to version2. Legacy reads
preserve its bytes and pending intent while reporting creation as blocked;
subsequent authoring persists version2. Existing acknowledgments retain their
commit/publish values. Missing gates in a version2 record fail closed. Migration
can invalidate an old rollout's reviewed runner revision, requiring a fresh
review before applying it. This is development fixture qualification; an exact
0.3.7 → 0.4.0 artifact/live upgrade remains a release gate.

Recovery preserves the database and rollout journal for inspection. After a lost
save response, refresh before retrying. Do not lower SQLite's `user_version` or
delete policy records to make an older runner start. Reverting requires the
supported stopped-management backup/restore procedure and matching artifacts;
never restore an earlier manager database over newer runtime operations. A full
0.3.7 → 0.4.0 exact-artifact/live upgrade remains release qualification work.

## Verification

For interactive UI review, use the [persistent isolated Setup preview](app-development-preview.md).
It runs the actual pane with simulated runners and saved fixture policies; it
does not connect to an installed collector or runner.

Repository fixtures cover inheritance, effect classification, CAS conflicts,
schema47/48 migration, reopened persistence, ordinary settings preservation, audit
integrity, atomic grant retirement, project generations, revocation before a
queued effect, default refusal, and unsupported/mutating CLI requests.
Collector tests cover offline rollout, retry after reopen, lost response,
concurrent edits and new enrollment. The offscreen QML tests exercise all scopes,
tree selection and path highlighting, new-desk inheritance, full-panel/sidebar
navigation, saved effective values, observer/offline refusal and stale responses.

```sh
go test ./internal/apppolicy ./internal/managed ./internal/client \
  ./internal/access ./internal/collection ./backend/cmd/urbit-collector ./cmd/omabit
python3 -B scripts/test-app-development-offscreen.py \
  --output /tmp/omabit-policy-ui-UNIQUE
python3 -B scripts/test-app-development-preview.py \
  --output /tmp/omabit-policy-tree-test-UNIQUE
```

The UI harness uses Quickshell in a networkless bubblewrap namespace, private
HOME/XDG directories and an inert helper. No active runners, ships, installed
plugins, services or desktop sockets are used by this qualification.
