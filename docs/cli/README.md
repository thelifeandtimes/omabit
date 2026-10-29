# Omabit CLI

- [omabit](omabit.md): desktop setup/service updates, optional appearance policy and private clipboard handoff.
- [urbit-collector](urbit-collector.md): runner selection, fleet and ship operations, projects, imports and application connections.
- [urbit-runner](urbit-runner.md): local hosting, installer, resource policy and runtime qualification.

Start with `urbit-collector --format text help`, then `help TOPIC` and
`COMMAND --help`. All help is offline. JSON is the collector's default output;
`--format text` gives readable tables and watch emits NDJSON. Place global
selection/format options before the command. Subcommand options follow its
positional arguments. No command silently chooses a runner for a mutation.

```sh
urbit-collector runners
urbit-collector --runner local --expected-runner-id RUNNER_ID hello
urbit-collector --runner local --expected-runner-id RUNNER_ID snapshot
urbit-collector --runner local --expected-runner-id RUNNER_ID operate INSTANCE start --revision REVISION --request-id YOUR_UNIQUE_REQUEST
```

Replace the placeholders with saved runner/instance identities and a freshly
observed revision. Inspect the returned operation receipt through `snapshot`;
submission is not completion. On a lost reply reuse the original request and
review its receipt. Never invent a new request ID simply to bypass a pending
operation or stale revision. Project operations also bind project revision,
review digest and exact member/container. Read their help before dispatch.

`configure` changes requested settings; the documented restart/maintenance path
applies them. `resource-policy show|set` separates new-ship provisioning from
existing-ship minimum/recommended thresholds. It is a local owner command on the
runner host. A desktop profile isolates preferences, not a remote runner's piers.

Lifecycle and import commands have different effects: a stopped-pier import
preserves identity and does not start it; delete/prune permanently remove data
and require explicit confirmations. Do not substitute reset/re-enrollment for an
upgrade. `qualify-*` creates disposable test resources and is not a health check.
Check capabilities using `hello`, especially with older remote runners. Private
chain planning/reservation does not imply a supported turnkey chain launcher.

Omabit-assisted plugin connections create separate ordinary Eyre owner sessions;
plugins use their own direct ship clients. These are not desk-scoped grants.
Prefer private stdin for +code and the private clipboard helper for interactive
use. Never include boot keys or session cookies in a command line or bug report.

The operator and developer skills are included under `skills/` in the source and
binary bundle. Read their SKILL.md and adjacent references together. The separate
skills archive is useful for explicit import into an agent's skill manager; no
installer modifies agent settings automatically.

## Reproduce the reference

Build the three binaries, then run:

```sh
python3 -B scripts/generate-cli-reference.py --bin-dir build --output docs/cli --check
```

Omit `--check` to regenerate after a help change. This runs every advertised help
entry in Bubblewrap with no network, host HOME, Docker, SSH, keyring or compositor.
It fails on a bad help exit or any created HOME state. Runtime correctness is
covered separately by source tests, the operator simulator walkthrough and the
release's explicit live qualification record.
