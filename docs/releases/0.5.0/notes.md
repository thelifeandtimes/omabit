# Omabit 0.5.0

Draft release notes. Qualification and publication are still in progress.

This release improves recovery of interrupted project-member Stops, adds a shared
typed CLI/MCP interface and app-development guidance, and introduces an opt-in
bounded runtime recording pilot.

- **Project recovery:** inspect an interrupted Stop, review fresh evidence of a
  clean exit, and reconcile it with a separate audit receipt. The original
  operation and pier identity remain intact. Starting the member is a separate
  action; reconciliation does not send another signal or repair an abnormal exit.
- **App development:** check the selected runner's APIs, role and saved permissions
  before app-specific project creation. A dedicated app-developer skill includes
  reproducible Gall samples and the versioned Grubbery source/restart workflow.
  Saved plans and receipts support recovery after a lost response.
- **CLI and MCP:** the typed operation catalog exposes shared handlers and explicit
  capability errors. MCP requires selected runners and explicit mutation opt-in.
  The tested stdio transport uses the pinned official SDK; experimental local
  authenticated HTTP is opt-in. Existing commands remain available, with
  incomplete legacy parity documented.
- **Runtime recordings:** optional raw/timed capture has bounded sessions and
  storage admission. Inspect selected retained output, query timing, or explicitly
  export original, indexed or derived redacted bundles. Capacity failures retain
  evidence and report incomplete coverage. Recording is disabled by default and
  does not automatically delete old data.

## Upgrading

The supported stable starting point is **0.4.0/schema59**. Use matching 0.5.0 desktop,
collector and runner artifacts; the runner advances management state to schema65.
Earlier stable versions must first follow their documented path to 0.4.0. Preserve
piers, enrollment, preferences and operation history when updating the manager.
Use the supported service-upgrade flow for a runner update; a desktop update does
not itself authorize changing running services.

Older managers refuse newer state. Do not lower the schema marker, reset a pier,
discard operations or re-enroll identities to work around a version mismatch.
Active recording-worker upgrades, remote service upgrades, power-loss recovery
and arbitrary older development snapshots are outside this release's tested
upgrade window.

## Scope

The app workflow is bounded. Broader peers, permits, routes, served assets and
failure combinations remain follow-up work. The Stop test does not establish the
exact cause of the original reported incident. Resource measurements are partial
observations and do not grant automatic containment or deletion authority.

Named test execution, complete legacy-command parity and interoperability with
every coding client remain open. Recording has no host-wide oldest-output
inventory, automatic age enforcement, existing-process adoption or retirement
workflow. Derived redacted exports still need owner review before sharing.
Issues 16–21 retain these follow-ups; issue 22 runtime pins is outside this release.

Frozen product source: `59d83aab49384d2f26531320bb5ab68e1257aeb5`.
Exact artifact hashes, test evidence and support limits will be included in the
final qualification record. Publishing this release does not deploy it to any
production service.
