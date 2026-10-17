# Stable-release upgrade requirement

The user requires seamless upgrades from the proper **0.1.0 stable release** to
future supported versions. Treat this as a release gate, not a post-release task.
The current repeated fresh-install exercises are development qualification only.

## User contract

- Upgrade in place: preserve piers, stable runner/instance identities, configured
  ports, storage roots and directory identities, enrollments, SSH grants/host-key
  pins, secrets, preferences, operation receipts and replay history.
- A management/UI update should preserve running ships and stopped intent. Runtime
  upgrades that require downtime need a coordinated graceful maintenance path;
  "seamless" does not mean pretending every Vere upgrade is zero-downtime.
- Coordinate collector, runner and QML compatibility. Older remote runners must
  remain observable and expose clear capability-based upgrade guidance.
- Keep migrations versioned, transactional and restart-safe. Interrupted upgrades
  must be resumable without bootstrapping over, deleting or re-enrolling a pier.
- Verify staged artifacts before atomic activation. Define tested rollback rules;
  refuse unsupported downgrades rather than letting old code discard new state.
- Preserve the Unix owner, ordinary filesystem access, unrelated ships/services
  and preexisting host settings. The protected-ship policy applies throughout.

## Required qualification before stable 0.1.0

1. Maintain fixtures from every supported prior schema/release, with populated
   instances, custom roots, access grants, policies and completed/interrupted jobs.
2. Exercise N → N+1, repeated execution, interruption at each migration/activation
   boundary, and the supported rollback/downgrade-refusal path.
3. Verify unchanged identities, data, permissions, ports, intended running/stopped
   behavior and actor-scoped replay. Exercise mismatched client/server/UI versions.
4. Run an upgrade smoke test on disposable ships without resetting the host.
5. Publish a clear support window and upgrade command/UI, including any necessary
   original-runtime preparation and maintenance window.

## Current RC integration decisions

The Ames enrollment fix applies to new live enrollments. Opening an existing
database must not silently publish a new UDP port or change a running container's
network contract. Legacy unpublished enrollments remain readable with their current
settings; a reviewed stopped-state publication migration is a pre-stable follow-up.
Its migration must resolve port conflicts and runtime provenance while preserving
configuration revision/replay rules, without requiring deletion or re-enrollment.

Import recovery is an explicit stopped-state journal operation, not an automatic
upgrade side effect. Existing schema16 records and their private recovery fences
must survive management restarts. New recovery behavior must handle legacy records
as well as guided/named/in-place imports, and must never infer that a missing
historical source path authorizes deleting or replacing another copy.
