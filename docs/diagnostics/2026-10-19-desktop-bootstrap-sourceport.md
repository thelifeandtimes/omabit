# Retained bootstrap: local sourceport deployment — 2026-10-19

The user authorized committing and deploying the bootstrap/profile implementation
on this machine. Implementation commit:
`35be5a92cb07ca44b871dfd11cd5294435e5d71a`.
Host: sourceport, stock Omarchy **4.0.4-1**.

## Committed candidate

Built a new immutable bundle from the clean commit with:

```sh
python3 -B scripts/omabit-dev.py build \
  --output build/desktop-bootstrap-35be5a9 \
  --label sourceport-bootstrap-35be5a9
```

The metadata reports `modified:false`, base application version0.1.0-rc.13 and
source snapshot SHA-256:
`c00801ca8847b5400a0a61fd8d89c6f518161f4bce993f58ed83ca22c1a55f4d`.
Its descriptor/release identity is:
`4301356b24d9e2de4ccac5cf79a13b42414f83893cbfab79e9dc791ccaf4cf8d`.

The real Quickshell lifecycle suite passed against this exact bundle, including
actual application/widget loading, paired helpers, busy deferral, failed-candidate
recovery, runtime-version refusal, isolated profiles and persisted view restoration.

## Native deployment

The installed checkout was clean at `b70f013`, with the old RC.13 static wrappers.
The desktop was unlocked and idle. Both comets were healthy and enrolled.

Prepared the authenticated published RC.13 archive first, then ran the **installed
stock** update command against the local commit:

```sh
env GIT_CONFIG_COUNT=1 \
  GIT_CONFIG_KEY_0=url.file:///home/thelifeandtimes/repos/omabit.insteadOf \
  GIT_CONFIG_VALUE_0=https://github.com/thelifeandtimes/omabit.git \
  omarchy plugin update thelifeandtimes.omabit --yes
```

This redirects only that command's Git fetch to the local repository. It does not
change the checkout's GitHub remote, persistent Git configuration, or Omarchy code.
The installed checkout fast-forwarded to the implementation commit. The new
entry-point URLs loaded the retained bootstrap without a cold shell restart.

The bootstrap first activated the known published RC.13 bundle and recovered the
one-runner/two-ship fleet. Then selected the committed candidate in the **default**
profile. Its runtime version/API handshake and readiness checks passed, reporting
the expected source commit, release digest, development label and both ships.

The default profile is explicitly pinned to this local development bundle. It does
not automatically follow later public pins until the operator selects `follow`:

```sh
python3 -B scripts/omabit-dev.py select default --release follow
```

The public RC.13 release remains the published baseline; this is a local deployment.

## Profile and visual acceptance

Created the new isolated profile `deploy-check-20261019`, with local host setup
disabled. Switched to it and verified a ready, empty collector with private XDG
roots. Switched back to default and verified both original ships returned healthy.
The empty test profile is retained for inspection; it contains no enrolled runner.

The actual bar entry was visible in the original right-section position, its popup
opened through normal IPC, and the mapped monitor showed both healthy ships and
the current application version. The monitor-only capture is private under
`build/desktop-bootstrap-deployment/monitor.png`. Existing window PIDs/workspaces
and the prior desktop focus were preserved; the monitor was returned to closed.

## Preservation checks

| Resource | Unchanged across migration, candidate activation and profile round trip |
| --- | --- |
| Stock shell | PID1148390 |
| Runner | PID3657443, invocation `2b471f28128840e2a6d5f1721f6793a5` |
| Rootless engine | PID1078779, invocation `8b8a2d8a5b5f42a08b6cc490dab49b13` |
| `bare-comet` | Container `c08cbe24ed355e7a0980ef3957a0381072eb5b9a5da2d6f142675b8a015aade3`, PID1094395, original start time and mount |
| `also-a-comet` | Container `f7c90e46054c6875b5553ae72302bba18644717d1ed899fd3a98fbc1fff57ac3`, PID1269096, original start time and mount |

Runner/instance identities, ship configuration/revisions, desired state, collector
enrollment, shell configuration and existing machine/fleet preferences matched
before/after evidence. No host setup, runner migration or ship operation was
submitted. Legacy UI teardown emitted null-binding warnings during the first
rescan; the new bootstrap subsequently reached ready with an empty error field.

Read-only deployment evidence is retained under
`build/desktop-bootstrap-deployment/`: `before.json`, `bootstrap-baseline.json`,
`candidate-active.json`, `profile-roundtrip.json` and `profile-restored.json`.
An initial evidence probe used the standalone native launcher and exceeded the
evidence helper's30-second wait during cold-cache preparation; subsequent checks used the already verified collector
paired with the active UI. The live fleet feed remained available.

This establishes a successful stock-command migration and profile round trip on
this existing Omarchy4.0.4 desktop. Fresh guest installation, the second Omarchy
version and the full host-upgrade VM matrix remain separate acceptance work.
