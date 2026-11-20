# Omabit 0.5.0 — project recovery and bounded app workflows

Candidate preparation: final artifacts are frozen at59d83aa. Matching desktop,
interface, stable0.4→0.5 upgrade, recording-pressure, Grubbery, native Stop and
closed-export gates pass (M774–M778); preserved DEV, review and publication
are pending. This
document describes the finite delivery scope; it does not announce a qualified
release or authorize a production update.
The [completion checklist](release-0.5.0-plan.md) tracks the remaining gates.

## Changes

- Project Stop tolerates the verified native running-to-exited observation
  transition. Saved operation lookup remains authoritative after a lost reply or
  manager restart. Diagnosis and reviewed reconciliation can resolve a verified
  clean interrupted member Stop while preserving its original receipt and peers.
  This does not establish the exact cause of the historical incident.
- App-specific creation checks the selected runner's build, role, APIs, adapters
  and current policy before reservation. Generic creation remains available.
  The app-development skill describes versioned Gall and Grubbery source,
  state/recovery and restart workflows. The Grubbery preflight advertises only
  the bounded source/restart contract; peers, permits, routes and served assets
  are separate qualification work.
- CLI and MCP share a typed catalog and handlers for the explicitly included
  operations. The catalog lists excluded legacy commands. Named checkpoint
  plans and historical reports are metadata, not an executable test service.
  Named-client recipes and local HTTP remain experimental; remote OAuth/TLS
  deployment is outside this release.
- An opt-in recording pilot preserves immutable original output, receive timing,
  explicit incomplete coverage and saved capture history across manager lifetime.
  Private reads, timing queries and explicit original/redacted ZIP export share
  runner and operator checks. The desktop provides bounded views and settings.
  Ordinary service startup keeps recording disabled.
- Project allocation and growth observations expose their measured components
  and unknowns. They do not establish a complete project budget or authorize
  automatic containment, Stop or deletion.

## Intended upgrade and support window

The release must qualify **0.4.0/schema59 → 0.5.0/schema65** with matching
Linux/AMD64 desktop, collector and owner-user runner artifacts. Older stable
installations should first follow their documented path to0.4.0. No broader
direct migration or remote operating-system upgrade is claimed by this path.

Update the desktop plugin and allow its verified bundle to load, then review and
apply the separate service update in Setup. Publication does not update any
running service automatically. Capability checks continue to distinguish an older
runner; loading a newer desktop alone does not grant new backend behavior.

The migration must preserve enrolled identities, piers, running/stopped intent,
access, settings, original operations and application history. It grants no new
app permission and does not enable capture. Older binaries must refuse schema65
without changing it. Do not lower schema markers, reset state or replace a pier
to make an older runner start. An intentional rollback requires the documented
stopped-management backup/restore procedure and matching artifacts.

The recording pilot applies only to newly admitted ordinary managed launches
using its supported native Docker path. It does not adopt existing processes or
enable project/maintenance factories. Capture failure retains the verified prefix
and incomplete status while the ship continues. Closed jobs keep their original
capacity commitment. There is no automatic deletion, age enforcement or history
retirement workflow. See [recording bounds and limits](runtime-recordings.md).

## Evidence and exclusions

Earlier isolated native passes are linked from the [next-gates record](issues-16-21-next-gates.md).
They identify their actual source and artifacts and do not qualify this combined
candidate. Final evidence must identify the exact frozen source, assets, source
and race checks, QML/package/bootstrap checks, native workflows and actual
non-destructive upgrade. The stable qualification record will be added only
after those checks pass.

The offline upgrade rehearsal uses disposable fakes and an exact-program
authorization denial/retry fixture. Human password entry, power-loss durability,
remote service upgrades, production ships and provider/public-network deployment
remain outside its scope. The authorized DEV review preserves its existing
piers/history/intent and adds only a fresh minimal workflow.

Issues16–21 retain their broader unfinished milestones; inclusion in this release
does not close an issue. Per-ship runtime pins in issue22 are outside this release.
