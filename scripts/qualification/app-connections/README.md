# Isolated application-authentication qualification

Read [experiment isolation](../../../docs/experiment-isolation.md) first. This
harness creates a fresh fake ship, never selects or resumes an existing pier.
It does not run an Omabit collector/runner service or use Docker.

Prerequisites: Linux user namespaces, Bubblewrap, a user systemd manager with
memory/CPU controllers, Python, Go, a standalone Vere executable, and the exact
pill pinned in `internal/runtimecandidate/metadata.go`. No packages are installed
by this script. Namespace or resource-limit failures stop the run; there is no
fallback to the host network or an unlimited process.

```sh
python3 scripts/qualification/app-connections/qualify.py \
  --vere /path/to/standalone/urbit \
  --pill /path/to/verified/boot.pill \
  --output /tmp/omabit-auth-run-UNIQUE-NAME
```

Output must be a new `/tmp/omabit-auth-run-*` directory. The pill is verified
before and after copying. Source hashes must remain unchanged during the static
Go test build. The executable's hash is recorded; this is not qualification of
the pinned Docker image. The entire lab runs in a uniquely named transient scope
with 4 GiB actual memory, no swap, one CPU core and 128 tasks. A 240-second boot
deadline, 180-second test timeout and 45-second shutdown grace bound the run.

The namespace mounts copied artifacts read-only and only its fresh lab data
writable. There is no host home, Docker/service/keyring socket, external network
or host PID access. Port 18081 is inside the private network namespace. The test
refuses to connect unless it finds a lab guard and different network/PID namespaces.
`--inside` is an implementation entry point and fails outside this sandbox.

`inputs.json` records provenance; `sandbox.json` records the exact boundary;
`lab/result.json` records readiness, test outcome and clean shutdown. The new pier
and bounded runtime log remain for diagnosis. All session material is synthetic
fake-ship data and is omitted from test results. A failed run is retained rather
than automatically reset. Re-running requires another fresh output directory.

Normal `go test ./internal/eyre` skips the isolated live test. Do not set
`OMABIT_AUTH_LAB` manually to test an arbitrary ship. The historical
`OMABIT_TEST_FAKE_ZOD` test is independently disabled by this harness.

See [the executed checkpoint](../../../docs/diagnostics/2026-10-21-isolated-auth.md).

## Real plugin receivers and direct transports

Add both source worktrees to stage separately named, headless Auth Lab artifacts:

```sh
python3 scripts/qualification/app-connections/qualify.py \
  --vere /path/to/standalone/urbit --pill /path/to/verified/boot.pill \
  --theme-source /path/to/theme-development-checkout \
  --dms-source /path/to/dms-development-checkout \
  --output /tmp/omabit-auth-run-UNIQUE-NAME
```

This also requires `dbus-run-session`, `gnome-keyring-daemon`, `secret-tool` and
`busctl` for exact Secret Service metadata checks, plus Quickshell/Qt Quick
Controls and its offscreen platform plugin for the real approval-pane gate.
The sandbox uses synthetic passwd/group records and its own bus/keyring, never
the desktop's. Source and staged-file hashes are recorded. Python helpers, declarations and manifest/profile metadata are copied, along with
Theme Service/Model and the actual Omabit approval components. The UI runs offscreen
with inert styles; Theme desktop publishing and DMs native launcher adapters are
disabled. This is not an installation into the active desktop.
The outer run is bounded to 500 seconds. There is no fallback to production
credentials or a plaintext credential adapter if private Secret Service fails.

See [the executed pair checkpoint](../../../docs/diagnostics/2026-10-21-plugin-handoff.md).


The harness now builds both `eyre.test` and `applicationauth.test` and records
source hashes for the issuer/registry/credential packages as well. The second
guarded binary exercises retained artifacts, lost acknowledgements, issuer
reopening, remote revocation and exact receiver-account reconciliation, followed
by new approvals and direct plugin transports. The combined test budget remains
185 seconds. See [the final receiver gate](../../../docs/diagnostics/2026-10-21-receiver-recovery.md).


The harness also builds `urbit-collector` and runs `public_flow.py` against the
same fresh fake. Separate `cli-lab` and `ui-lab` copies exercise declared-helper
discovery, explicit approval, replay/stale refusal, actual QML enrollment/sign-in/
approval, direct reads/SSE after the panel closes, independent logout and retained
helper cleanup after installed-source updates. The real Theme service and DMs
bridge begin empty before UI approval and later adopt the accounts. No persistent
collector or runner service is started. The final
[public-flow checkpoint](../../../docs/diagnostics/2026-10-21-public-auth-flow.md)
records evidence and the remaining full-native/profile limits. The combined test
budget remains 185 seconds and the outer deadline remains 500 seconds.


## Locked credential storage and daemon restart

`keyring_lock.py` tests the actual copied Theme/DMs credential-store code against
a new private Secret Service. It requires the same Bubblewrap/systemd and
keyring tools, but no Vere, Docker, display or ship. For example:

```sh
PYTHONDONTWRITEBYTECODE=1 python3 -B scripts/qualification/app-connections/keyring_lock.py \
  --theme-source /path/to/theme-development-checkout \
  --dms-source /path/to/dms-development-checkout \
  --output /tmp/omabit-keyring-lock-UNIQUE-NAME
```

Four synthetic sessions use separately labelled stable/dev copies at the same
origin and ship. Sixteen fresh helper processes exercise save/read, locked read
and replacement-save refusal, then original-session recovery after restarting
only the fixture's keyring daemon with its original password. DMs account files
must stay unchanged and no cookie may appear in state/config/cache. Namespace
checks precede store access; the capped scope has256MiB/no swap/one CPU/64 tasks
and a180-second deadline. The host's bus, keyring, HOME and network are absent.

This proves the credential-store boundary, not UI unlock prompts, a real desktop
login/reboot or application subscription recovery. Nonsecret results/hashes may
be retained; do not export raw keyring files or the private console. A failed
invocation is preserved; use a fresh output path for a corrected experiment.
