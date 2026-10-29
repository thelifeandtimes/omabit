---
name: omabit-operator
description: Manage Urbit runners, ship operations and stopped-pier imports through Omabit's collector CLI. Use for fleet status, explicit start/stop, import diagnostics, operation recovery, fake-project lifecycle and private-chain review, reservations and local runtime input inspection.
---

# Operate Omabit

Use `urbit-collector` from the intended Omabit installation or development profile.
The collector selects runners; the runner owns piers and runtime operations.
A local desktop profile does not isolate the remote runner or its ships. Honor the
user's excluded ships, hosts and directories without probing them.

Start with the shortest relevant read path:

1. Resolve the selected tool generation and collector registry. In a source
   checkout, `scripts/omabit-dev.py exec --profile NAME urbit-collector ...` runs
   that profile's tool without selecting it as the active desktop. Otherwise use
   the configured tool path and registry; do not silently substitute another
   binary from `PATH` or enroll a new runner to repair a missing selection.
2. For offline project planning, follow [project review](references/projects.md)
   without selecting or enrolling a runner. For fleet operations, run
   `urbit-collector --config-dir REGISTRY runners`. This reads saved connections
   without contacting runners. Select the requested name and its `runner_id`.
3. Use `--config-dir REGISTRY --runner NAME --expected-runner-id ID` before every
   selected-runner command. Read `hello` for capabilities/role and `snapshot` for
   the current instances and operations. Use `--format text` before the command
   for a readable view; JSON is the default.
4. Use `urbit-collector help` for offline task topics (`--format text help` for
   readable output), then `help TOPIC` and the relevant command's `--help`. Older
   builds without topic help still expose `--help` and `COMMAND --help`. Required
   effects/flags vary by command; this guide does not establish runtime support.

An instance ID is a management name, not necessarily its Urbit `~ship` identity.
Several independent fake instances can share an identity. Bind work to the runner
ID, instance ID, current configuration revision and the intended filesystem object.
Current managed fakes have container-local Ames: a common project directory does
not connect them. Do not infer networking support from a path or shared identity;
use the project capability and the project/member IDs to select a shared Docker
namespace. Project members support start/stop, logs, code and resource observations;
ordinary maintenance/import actions are not yet supported for project members.

- For fake-galaxy planning, reservations, lifecycle actions or saved project status, read
  [project review](references/projects.md).
- For private local chain/genesis/contracts review, reservations or sealed input inspection, read
  [private-chain review](references/private-chains.md). Its offline plan does not
  authorize registration or runtime actions.
- For observation, start/stop, revision conflicts and receipts, read
  [operations](references/operations.md).
- For a stopped pier, layout, import diagnostics or retained-source recovery, read
  [imports](references/imports.md).

Use the authorization already present in the task. Before an effect outside that
scope, explain the exact proposed change and resolve the missing authorization.
Do not ask again for an already authorized action merely because the CLI requires
an acknowledgement flag. A failed or timed-out client is not evidence that the
runner rejected a mutation; reconcile its request/operation receipt first.

Keep secrets out of arguments, logs and normal status reports. `ship-login NAME
--code-stdin` accepts the login code through standard input. `login-code` explicitly
reveals a secret and is unnecessary for ordinary fleet observation. Do not turn a
normal diagnostic into credential collection.

Use the collector for normal management. `urbit-runner doctor` is local metadata
inspection; `host-check --live` explicitly probes the installed owner's engine.
Installation, package verification/repair and `qualify-*` commands have different
effects: qualification creates disposable runtime resources and is not a read-only
health check. Run those only within the task's authorized host and test boundary.

Application connections use separate ordinary Eyre sessions. Omabit assists auth;
plugins send their own traffic directly to Eyre and can also sign in independently.
An Eyre cookie is broad ship access, not a desk permission boundary. Inspect the
specific application command's help and the approved plugin/profile before granting
access; do not copy a parent cookie into several plugins.
