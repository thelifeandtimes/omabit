---
name: omabit-app-developer
description: Develop and test Urbit Gall or Grubbery application backends with Omabit's isolated projects, staged source, native desk operations and exact recovery receipts. Use for standalone apps and games; Omarchy frontend integration belongs to omabit-plugin-developer.
---

Inspect the repository instructions and actual backend before choosing Gall or
Grubbery. This choice is independent of Omabit's mounted/urgit source workflow.
Preserve the existing publisher, source ownership and state/API compatibility.
For Gall, read [gall.md](references/gall.md); for Grubbery, read
[grubbery.md](references/grubbery.md). Do not load both paths automatically.

Before booting an app-specific project, select the intended bundle/profile,
enrollment and expected runner ID. Run matching `omabit app-preflight` with the
framework and effects requested. Inspect actual Hello capabilities, role and
reported build, rather than inferring the runner from the desktop version. An
unsupported environment is a preflight result: continue useful source work, but
do not boot an app project or bypass a missing adapter to discover incompatibility.

Use fresh disposable projects for qualification. A fake identity or desktop
profile alone does not establish isolation. Keep existing/manual-review ships
and excluded piers untouched; fleet provisioning and lifecycle recovery belong
to the operator workflow. Never inspect an excluded ship to confirm its exclusion.

Read [the common app-development contract](references/app-development.md) for
selected-target policy, workflow settings, staging, commit, release, receivers
and frontend delivery. This package includes its required references; sibling
global skill installation is unnecessary. Permission authoring stays in Setup.
Report the exact blocked scope without broadening grants or using raw effects
to evade a gate. Existing task authorization does not need another prompt, but
it cannot grant a saved policy permission.

Prepare a bounded source tree outside live mounts, stage only required files,
review and save complete plans before submitting once. Verify source accepted,
native compilation, active agent/framework, loaded instance, permits, route and
exact served client bytes separately. A version marker or HTTP200 is insufficient.
Use generated command help and the shared operation catalog as the authority for
available CLI/MCP handlers; do not assume a proposed adapter is installed.

Follow [testing and recovery](references/testing-and-recovery.md) for native
behavior, restart/subscription/state evidence, uncertain effects, retention and
cleanup. Report source-only, live single-ship, peer, browser and upgrade evidence
separately. Never edit a runner database to unblock an application test.
