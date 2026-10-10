# Unified desktop 0.4 validation — September 28 UTC

The unified Omabit snapshot is installed as **0.4.0**. It includes the compact
fleet menu, setup pane, and separate monitoring/maintenance/activity window.

## Automated validation

- Node model checks passed for existing revision/confirmation guards and new
  summarized statuses, offline last-known inventory, identity replacement,
  browser-host composition (including staged-port behavior and remote-loopback
  refusal), activity filters, resource freshness and sample discontinuities.
- The isolated QML fixture passed persisted presentation preferences, actual
  maintenance/prune confirmation buttons, preview invalidation, ambiguous-result
  request retention, no automatic retry and secret-stdin login.
- `scripts/test-panel.py --windows` additionally mapped a real fixture window and
  rendered overview, maintenance, activity, the chart canvas and the themed filter
  popup. Selecting a runner filter updated the expected runner identity.
- Snapshot replacement/discovery/enable tests and `git diff --check` passed.

All fixture mutations used the isolated executable and state directory. No new
runner backend functionality was introduced in this pass.

## Live visual inspection

The shell was restarted after snapshot updates to bypass the previously observed
stale-QML cache. The new read-only monitor status IPC confirmed version 0.4.0.
Screen captures were inspected for:

- compact three-ship fleet grouping and lifecycle/status hierarchy;
- setup navigation cards;
- windowed and full-screen monitoring with RAM limit bars, observed-sample chart,
  historical retained epochs, runtime connectivity and expandable detail;
- unified activity with themed runner/ship selectors and compact local timestamps;
- maintenance cards, the routine-policy entry point and stopped-state admission.

The window follows the compositor layout; its expand control and F11 use actual
full-screen mode rather than relying on client maximization requests, which this
desktop's tiling policy did not honor visually.

The workstation SSH configuration was read and confirmed `HostName 127.0.0.1`.
Its local presentation preferences now use browser base `http://127.0.0.1` and
location label `This computer / 127.0.0.1`. Collector enrollment and identity pins
remain authoritative. The previous presentation file was backed up at
`/tmp/opencode/omabit-panel-preferences-before-0.4.ini`; the pre-redesign plugin
archive is `/tmp/opencode/omabit-panel-before-0.4-20260928.tar`.

A final identity-pinned snapshot showed alpha and panel-check running/healthy,
ssh-check uninitialized, and no active operations/recovery requirements. The latest
request was still `checkpoint-6k-resume-alpha-2`: visual inspection added no live
runner operations.

The [checkpoint](../checkpoints/23-fleet-desktop.md) records the backend gaps and
next decisions, especially secure code retrieval, provisioning, connection edits
and schedules. User design feedback is the next refinement input.
