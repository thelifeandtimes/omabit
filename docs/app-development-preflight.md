# Selected-runner application preflight

Issue21 adds `omabit app-preflight` before app-specific project creation. The
shared typed `collection.Registry.AppDevelopmentPreflight` reads only the selected
enrollment's authenticated Hello and policy. It pins runner identity/mode, reports
the key role, and separates missing API support, missing native operation adapters,
missing local framework adapters and saved-policy denials. It performs no launch,
source transfer, policy edit or installation. Generic project creation is unchanged.

Use `--workflow gall|grubbery` from inspected source and explicit effects
`create-desk,commit,publish` (default `create-desk,commit`). Gall uses native desk
primitives. Grubbery reports its adapter as unavailable until that separate work
is qualified. No missing capability is an invitation to use a raw command bypass.

Without a target, success means environment compatibility, not a future ship's
permission. With `--instance` and `--desk`, policy reads resolve the current project
creator/generation and report the exact effective scope. A nonzero exit with a
normal structured result identifies incompatibility; transport/identity failures
remain errors. Policy authoring stays in Setup, with narrow ship/desk grants.

All results state `capability-and-saved-policy-only`. They grant no action
authority: source, runtime, publication, compile and served-byte evidence still
belong to effect-specific review. Commit on a published desk additionally needs
Publish permission. Preflight cannot establish that from saved gates alone.

New managed runners advertise `runner-build-identity-v1`, compiled source revision,
dirty status, Go/module versions, executable SHA256 and state schema in Hello.
These are authenticated self-reports, separate from artifact verification. Older
peers omit them; unknown is preserved. The helper's compiled build is reported
separately. A plugin manifest or client version does not identify the running
runner. No supported old API is removed by these optional Hello fields.

Selected-only, wrong-identity/role, unknown build, API/adapter separation and
global/runner/ship/desk policy-source tests use inert Unix fixtures. They verify no
writes and preserve the registry bytes. Offline help creates no state. Matching
DEV deployment and actual minimal native app operation remain live qualification
gates; these source tests do not establish the target of the historical incident.
