# Named development fake and panel 0.13 acceptance

Accepted on the existing development host, October 1, 2026 (UTC).

## Deployment and preserved fleet

- Rebuilt native runner, collector and Omabit helper; deployed the runner through
  `scripts/install-services.sh` with OS authorization.
- Management and dedicated SSH restarted; the rootless engine and existing ship
  containers stayed running. Hello advertises `fake-ship-options`.
- Installed panel 0.13.0 with the three bundled binaries, preserving the installer
  ownership checks. After `omarchy restart shell`, monitor IPC reported 0.13.0.
- Runner identity remains `runner-f496feed1f7b08f31a5d82b94ac2f5fd` in
  `development-managed` mode. All collector calls pinned that identity.
- `workstation` remains enrollment revision 3, using the app-owned `omabit-local`
  SSH configuration established by the preceding local-setup flow.
- Existing `alpha`, `comet-check`, `import-check` and `panel-check` remained healthy
  with unchanged container IDs, ship identities and configuration revisions.
  `rus` and `ssh-check` remained stopped/uninitialized at 1/0. Both existing routine
  policies remained disabled at revision 2.

## Independent names and official runtime

| Field | Accepted value |
| --- | --- |
| Local instance | `bus-lab` |
| Fake Urbit identity | `~bus` |
| Pier directory basename | `bus-data` |
| Private instance ID from epoch inventory | `2d0300061f8766c1416a0ea1f49c05ba` |
| Host HTTP | `http://127.0.0.1:18088` |
| Configuration / applied revision after boot | 1 / 1 |
| Runtime | Official pinned Vere `4.6-5a7614f` |

The managed path is
`/var/lib/urbit-runner/.local/share/urbit-runner/managed/2d0300061f8766c1416a0ea1f49c05ba/data/bus-data`.
Lifecycle containment and the stopped epoch scanner validated the registered
directory. No arbitrary host path or existing-pier adoption was requested.

The normal fake command retains `-L` and `--no-conn`: Ames and the control socket
are disabled. Host socket inspection found HTTP listening only on `127.0.0.1:18088`
and no UDP listener on 34343. Direct, proxy-free public `/~/host` checks returned
HTTP 200 and `~bus` both before and after replacement.

## Durable operations

Actor: `ssh-key:SHA256:JJaz17a/T+cdWH+HFcksEjJm95sPQIOlcadVVNa5Rrc`.

| Request ID | Operation | Result |
| --- | --- | --- |
| `named-fake-register-bus-lab` | `op-67510a90591e70b9b69ac0837c9ca122` | Registered stopped/uninitialized; same request replay returned this operation without another instance |
| `named-fake-start-bus-lab` | `op-28e322255b4a0d0313f25078bdcfa3a4` | First boot succeeded, 01:14:51–01:17:49 UTC, healthy `~bus` |
| `named-fake-stop-bus-lab` | `op-42494cc39e5bac0a41d54e239115396e` | Clean stop/removal succeeded in about 1.5 seconds |
| `named-fake-inspect-bus-lab` | `op-1f2078da3e1754fee7eec5840d24e7f1` | Stopped metadata inspection succeeded; one protected `0i0` epoch |
| `named-fake-resume-bus-lab` | `op-7c32fd59af2dbf04eee32079c07fd569` | Initialized replacement/resume succeeded in about 7 seconds; same identity, directory and revision |

Available physical RAM before submission was 5,738,713,088 bytes for first boot
and 5,659,348,992 bytes for resume; the runner also performed its normal admission
checks. The fixed 4 GiB runtime memory limit remained in place.

- Creation container:
  `e797a2822a7e02aacc7ef6d23630520df74a9371ec8c3257aec59b520945cc95`.
- Initialized replacement:
  `493addd69a2c0f818fb4f39e309480432ff32b59ecf4dcb62f60af7b7fbc73ee`.
- Stopped inventory digest:
  `1eac5d3ca0e052953a4cca99007228c21a9958d031b2f326e8673b8fa1d74589`.
  Genesis held 218,837,004 logical bytes / 218,841,088 allocated bytes in four files.

Final snapshot: seven registered instances, five running/healthy, two uninitialized,
no active operations, recovery requirements or incidents. `bus-lab` remains running
for development use; no existing registration was renamed or removed.

## Automated checks and scope

Passed before deployment:

- `go test ./...` and `go vet ./...`.
- Race tests for protocol, managed runtime/store, collector/runner CLI, onboarding
  and host installation.
- Native builds for all commands and Linux/ARM64 runner cross-build.
- Panel model, installer and mapped-window QML tests; `git diff --check`.
- Named-fake fixtures cover boot/resume, pack/meld/roll, epoch inventory, prefix
  pruning, routine maintenance and startup resume with the selected directory.
  Wrong identities remain quarantined; unsafe directory names are refused before
  runtime dispatch; schema 8 prevents an older runner ignoring the new metadata.
- Real QML fixtures verify independent name fields, left-aligned consent controls,
  local duplicate name/HTTP-port refusal, retained inputs on backend conflicts,
  correctable CLI validation failures and request-bound success feedback.

The live named-fake gate exercised registration, boot, stop, inspection and resume.
Its maintenance/pruning/startup variants were fixture-tested, not rerun live here.
This is development-runtime acceptance; clean-host first installation and the
broader production gates remain separate.
