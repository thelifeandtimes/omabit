# Omabit 0.1.0-rc.9 — Ames publication and import recovery

RC.9 incorporates and adapts the contributions in
[PR #2](https://github.com/thelifeandtimes/omabit/pull/2) and
[PR #3](https://github.com/thelifeandtimes/omabit/pull/3) by
**wundrellama (~dinnyt-divsud)**.

## Changes

- New live enrollments/imports allocate and reserve an available host Ames UDP
  port. Initial identity verification keeps Ames disabled; normal resume publishes
  the selected mapping. Fake ships retain their isolated Ames behavior.
- Enrollment records resolved pill/runtime provenance so publication configuration
  passes initialized-pier startup checks without a key, download or fresh bootstrap.
- Every external import path now checks epoch compatibility before moving source
  data, including the legacy CLI without `--identify`.
- Explicit `recover-import` completes eligible prepared imports retained at
  `enrolling`. It handles moved and in-place imports, named piers, guided identity
  checks and the new Ames publication configuration.
- Recovery validates retained evidence and completes the journal while leaving the
  ship stopped. Recreated source/holding paths, unsafe filesystem state, unrelated
  uncertainty and stale/unauthorized requests are refused. Completion and its
  success receipt commit atomically; interruption retains the recovery fence.
- The desktop offers **Review recovery → Validate and finish enrollment**. Reviews
  survive inventory updates, duplicate clicks share one request, and definite
  failures require an explicit reviewed retry. Start remains a separate action.

## Verification and upgrade scope

Full Go tests/vet, stateful race checks, static builds, model and mapped
recovery/onboarding/panel/log tests, and installer/bootstrap checks passed locally.
Crash/publication tests reopen the database and verify that polling or Stop cannot
clear a failed recovery's private fence. Schema15/16 preservation tests retain
runner identity, runtime/enrollment records, ports, intent and replay history.

Stable **0.1.0** now has a documented release gate for tested, non-destructive
upgrades. Fresh-install resets are development qualification only. Existing
unpublished Ames configurations are preserved on database open; their reviewed
stopped-state migration is tracked before stable. See
[upgrade policy](https://github.com/thelifeandtimes/omabit/blob/master/docs/upgrade-policy.md)
and [integration provenance](https://github.com/thelifeandtimes/omabit/blob/master/docs/diagnostics/pr2-pr3-integration.md).

This candidate retains schema16, the pinned Docker/Vere runtime, the frozen
supplied pill, RC.8's guided import/setup UI and RC.7's persistent log watcher.
It does not claim live recovery qualification on protected piers, general partial-
copy resumption, or completion of the stable updater's end-to-end qualification.

## Fresh installation

```bash
omarchy plugin install https://github.com/thelifeandtimes/omabit.git --enable
```

Install matching runner, collector and QML to expose the new capabilities. Native
Vere mDNS recovery remains a separate verified-runtime rollout.
