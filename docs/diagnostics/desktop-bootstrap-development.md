# Desktop bootstrap/profile development verification

This records development verification performed before the initial bootstrap/profile
commit, following RC.13 distribution commit `b70f013`. It is not a new published
release or a completed two-version Omarchy VM matrix.

## Implemented

- Native API-v1 service/widget adapters use stock `keepLoaded` behavior and load
  immutable, authenticated UI/tool archives from release-specific paths.
- A process-held lease, atomic versioned profile state, durable activation intent,
  runtime version/API handshake and service/widget readiness distinguish selection
  from successful activation. Busy work defers switching; admission is frozen
  before asynchronous activation. Failed candidates retain the previous desktop.
- Lost commit acknowledgements restore only the exact transaction's previous UI
  state. Unknown future state/contracts are refused without rewriting them.
- Default paths are retained. Named profiles isolate all writable application XDG
  roots, disable implicit host setup and permit explicit release pinning.
- The development CLI builds immutable worktree snapshots with honest source
  provenance, imports releases, selects profiles/releases and runs paired tools.
- Existing setup surfaces show profile/release/collector diagnostics. Version labels
  share one compiled application version; reconnects retain stale observations.
- Selected page/ship preferences are explicitly flushed. The actual restart test
  caught Qt's debounced Settings property writeback; syncing explicit keys corrected
  the loss before the final candidate was built.
- The guarded guest harness implements stock fresh-install/update scenarios,
  optionally including authorized host setup, two fake ships and runner upgrades.

## Passed checks

- `go test ./...` and `go vet ./...`.
- Panel, runtime-settings and host-compatibility JavaScript models.
- Both existing actual offscreen Qt suites.
- Twelve desktop bundle/profile/lease/activation tests, including tampered UI with a
  rewritten inventory, invalid archive members, future schema refusal, stale sessions,
  private helper environments and lost-ack recovery.
- Five VM-harness admission/projection tests; its default plan and CLI help are inert.
- Eight existing native bootstrap tests and four desktop qualification boundary tests.
- Snapshot installer/recovery tests and release-source isolation checks. The latter's
  optional full-reproduction mode was not rerun for this task; development builds
  have their separate modified-source provenance.
- Actual Quickshell tests verify fresh nested JavaScript and paired tools, busy
  deferral, failed service/widget rollback, wrong runtime-version refusal, profile
  switching and process restart.
- A built application archive with the real helpers loaded through the production
  bootstrap in a private fresh profile. Both the actual service and widget loaded;
  its selected page/ship survived an immediate process restart.
- Actionlint and whitespace checks.

The actual-application run used installed stock Omarchy QML modules read-only and
the Wayland socket for PanelWindow type support. Every writable HOME/XDG root was
private, windows stayed closed, and no live desktop IPC or runner mutation was used.

Final tested development output: `build/desktop-bootstrap-candidate-3/`.
Label: `bootstrap-profiles-3`; base source `b70f013f9398b6b4acad610966a23ab5f8953c7e`.
Source snapshot SHA-256:
`cc163e2782e3ce3c2cffb56d023afb3ec50fa308f6f77bf2d2f575305aef4bff`.
Its metadata correctly records modified source and base application version RC.13.
Earlier candidate directories remain retained; the second candidate's preference
restart failure is not counted as passing evidence.

## Remaining release acceptance

The full `omarchy plugin install`/`omarchy plugin update` guest scenarios have **not
been executed on both target Omarchy versions**. They require committed distribution
inputs with complete pins and provisioned clean guests. Profile selection and
isolated Qt tests do not substitute for that Git-update/filesystem-watcher boundary.

The source was ready for that qualification stage. Published RC.13, the installed
desktop and host services were not replaced during these development tests. Full
runtime and stable-upgrade qualification requirements continue to apply.
