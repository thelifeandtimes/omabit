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

## Development runtime settings

The0.2.0 remote-onboarding work keeps runner schema21 and desktop state/API1.
Pairing an existing runner modifies only the dedicated access grant and its exact
forced-command SSH key entry. New version1 attempt journals are separate from the
runner database and native-package evidence. Existing operator keys, port22222
connections, piers and running/stopped intent remain valid. Fresh installation and
known interrupted installation use the established owner-user installer; a missing
profile/path never authorizes recreating an existing runtime. See
[remote setup](remote-runner-setup.md) and its development verification record.

RC.15 separates reuse of existing OS-delegated subordinate IDs from allocation of
new ranges. Fresh owner setup validates the local account binding and both
root-owned grant files without requiring an existing Omabit profile. Existing
explicit reservation policies still apply; unknown identity domains do not gain
automatic allocation authority. No runner/desktop schema change is involved.

RC.14 adds a separate version1 desktop activation and
profile store. Default user paths remain unchanged; named profiles have private
XDG roots. Immutable UI/tool generations share a bootstrap/profile contract, and
unknown formats are refused rather than relabeled. Desktop selection never opens
or downgrades a runner database. See [desktop development](desktop-development.md)
for activation, recovery and stock-Omarchy upgrade scenarios.

The **RC.13** host-compatibility work advances to **schema21**. It
accepts schemas11–20 without rewriting historical instance bodies, signatures or
receipts, and adds genuine RC.12/schema20 writer fixtures. New container records
carry the observed engine version; historical omitted fields remain omitted.
Older runners must refuse the successor rather than discard this evidence or
apply obsolete engine-admission assumptions. See
[host compatibility](host-compatibility.md) and its
[upgrade procedure](host-compatibility-upgrades.md). Source and isolated
native-package verification do not qualify every new runtime/distro combination;
the [RC.13 notes](release-rc13.md) and attached qualification record define its scope.

The post-RC.11 source remediation advances to **schema20**, accepting existing
schemas11–19 without resetting instances or receipts. The new version fences
older runners from ignoring durable management-update admission, object-bound
deletion/staging evidence, maintenance dispatch phases and handoff audit receipts.
Historical signature bytes and actor/request fingerprints remain unchanged.
Already-persisted interrupted imports have explicit evidence-checked recovery;
unexplained replacement objects and ambiguous legacy maintenance stay preserved.

Quiescence-aware managers use a durable local-root update fence, verified retained
executable generations and a restart-safe activation journal. Runtime processes
remain independent. A previously published schema19 manager lacks that admission
contract: use the explicit [one-time offline upgrade](management-offline-upgrade.md),
which holds the existing runtime lock, migrates and fences in one transaction,
and preserves owner, piers, configuration, history and running containers. An
older binary is never restored across an incompatible schema migration.

Historical fixtures produced by original RC.1–11 writers and the recovered
schema14/schema18 previews now cover schemas13,14,15,16,18,19. Every published
release has its own pinned original-writer fixture. Original writers for unpublished
schemas11,12,17 remain unavailable; their synthetic migration coverage is not
equivalent to original-writer qualification. These prototypes are outside the
supported published-release window. See the fixture manifest in
`internal/managed/testdata/historical-upgrades/` and the
[release-by-release support matrix](reviews/2026-10-18-rc12-historical-upgrades.md).
The frozen RC.12 artifacts subsequently passed the documented supported-host
upgrade, runtime, guest power-cut and desktop gates. See the
[release qualification record](diagnostics/2026-10-18-rc12-release-qualification.md)
for results and limits; this does not replace the separate stable0.1.0 requirement.

The integrated RC.10 uses schema19 and accepts schemas11–18 transactionally.
This fences runtime-settings-aware handoffs from schema18 preview binaries that
would discard their settings. Historical handoffs retain their original signature
bytes. New nondefault transfers sign loom/native settings and require capability
checks before source fencing. Legacy configuration requests preserve omitted
native-control values; explicit false still disables them on the next replacement.
Release metadata obtains the schema from the built runner's `state-schema` command.

The subsequent boot-form refinement restricts new loom selections to30–34 and
makes native control CLI-only. Existing smaller RC.10 configurations/launches,
signed transfers and actor-scoped creation replays retain their historical contract.
Observation, Stop, resume and maintenance must remain possible without relabeling
or clamping stored values. New configuration requests use the narrower bounds.
Desktop edits omit native control and preserve an existing CLI choice. This is an
in-place source update with no persistent format change or reset requirement.

The contributed schema18 preview broadens explicit native control to all ship classes. Its version-only
transaction accepts schemas11–17 without rewriting bodies or receipts. Schema17
runners refuse the successor because their runtime validator forbids enabled livenet
sockets. Source tests cover repeated schema17 upgrades and preserved request replay.
A separately approved schema18 management update also preserved the four current
instance configurations, running/stopped intent and both running container/process
identities. It did not enable or exercise any livenet socket. Livenet runtime checks
remain separate from both management updates.

Schema17 records loom selection and development-fake native-control opt-in.
The version-only transaction accepts schemas11–16 and leaves instance and operation
bodies unchanged. Missing private launch fields mean the historical loom31 and
native-control-off contract. Existing running containers retain that contract until
an explicit replacement. Older schema16 runners refuse the new schema.

Source tests cover legacy bodies, repeated migration, defaults, desired versus
recorded settings and future-schema refusal. A separately approved management-only
upgrade preserved two running ships' containers, processes, configuration and
operation history. Fresh-fake native-thread acceptance also passed. This is not
full qualification of every stable-release upgrade path. See [the guide](development-runtime.md)
and [verification report](runtime-settings-validation.md).
No database reset or downgrade-marker edit is an upgrade or rollback procedure.

## Current RC integration decisions

The Ames enrollment fix applies to new live enrollments. Opening an existing
database must not silently publish a new UDP port or change a running container's
network contract. Legacy unpublished enrollments remain readable with their current
settings. The post-RC.11 source provides explicit `publish-ames` for a cleanly
stopped enrolled live ship, with capability, identity/provenance, filesystem and
host/reserved-port checks. It advances the configuration revision, records the
authenticated actor/request and prior launch contract, and leaves the ship stopped;
publication happens on the next explicit Start. Exact replay survives restart.
It requires neither deletion nor re-enrollment. See
[legacy Ames publication](legacy-ames-publication.md) for the CLI contract and
qualification limits.

Import recovery is an explicit stopped-state journal operation, not an automatic
upgrade side effect. Existing schema16 records and their private recovery fences
must survive management restarts. New recovery behavior must handle legacy records
as well as guided/named/in-place imports, and must never infer that a missing
historical source path authorizes deleting or replacing another copy.
