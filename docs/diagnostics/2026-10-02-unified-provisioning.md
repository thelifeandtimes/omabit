# Unified provisioning and moon validation — panel 0.18

## Moon-name diagnosis

The screenshot submitted `~dipben-fogweg-dozzod-sarlev`. The user confirmed the actual
generated public name is `~dibpen-fogweg-dozzod-sarlev`: the b/p were transposed.
`dip` is not a standard @p prefix syllable; `dib` and suffix `pen` are valid.
The CLI rejected the malformed name before opening a key file, but its combined
validation error incorrectly suggested runner/path/port setup as possible causes.

Validation now reports the specific public-name error. The correct generated moon
has protocol, real CLI/HTTP fixture and store-registration coverage, including
inferred instance/path and automatic ports. All key material used by tests is
synthetic; the user's real key was not opened or submitted.

## Interface changes

- Activity time is top-right; host / instance / status is bottom-right; Details is
  beneath the action. Layout fixtures assert these relationships.
- Add a ship is the common entry for a new comet, an owned fresh key, or an existing
  pier. Development tools remain available in a secondary expander.
- Setup's redundant key/add-ship cards and obsolete key-boot placeholder are removed.
- Manage runners is based on local collector registrations, including offline
  entries. It offers connection/presentation editing and adding a runner, without
  ship listings or ship-port controls.
- HTTP configuration is available in each ship's Connectivity & runtime section,
  retaining expected-revision checks and explicit stop/start application.
- Key registration feedback is scoped to its form. Navigation clears obsolete
  general feedback instead of repeating an old key error on unrelated pages.

## Verification

The following checks passed before deployment:

```sh
go test ./...
go vet ./...
go test -race ./internal/protocol ./internal/managed ./cmd/urbit-collector
go build -o ./build/ ./cmd/...
GOOS=linux GOARCH=arm64 go build -o /tmp/opencode/urbit-runner-provisioning-arm64 ./cmd/urbit-runner
node scripts/test-panel-model.js
python3 -B scripts/test-panel-install.py
python3 -B scripts/test-panel.py --windows
python3 -B scripts/test-panel.py --windows --capture-dir /tmp/opencode/omabit-provisioning
git diff --check
```

The activity, key-boot and Manage runners captures were visually reviewed. Fixtures
exercise the unified Add a ship entry, offline collector registrations, scoped
errors, activity alignment and generated moon defaults. The ARM64 build checks
unsupported-runtime stubs; it is not runtime qualification.

The installed collector was also checked using a deliberately nonexistent key path:
the transposed name reports `Unknown Urbit prefix syllable "dip" in "dipben"`, while
the confirmed `~dibpen-fogweg-dozzod-sarlev` passes name/default validation and stops
at the private-key-file requirement. Neither check registers a ship or reads the
user's real key.

## Live deployment

Panel 0.18.0 was installed and its loaded version confirmed through the monitor IPC
after an unlocked shell restart. The previous owned plugin snapshot is backed up at
`/tmp/opencode/omabit-panel-before-unified-provisioning.tar`.

Graphical host authorization was unavailable (`No authentication agent found`), so
the user ran the service installer with `sudo` in their terminal. Subsequent checks
confirmed that the build, installed host runner and plugin-bundled runner all have
SHA256 `b6b8d558a6032d0ddf1c7fb5a122a9966fd5fa25c0d649092bb3997b49a4fd74`.
The desktop helper and collector also match their built artifacts. Dedicated SSH
is active/enabled, and identity-pinned Hello returns the existing runner and
operator actor with `key-boot-options` available.

The post-update snapshot at `2026-10-02T04:35:22Z` preserved all seven registrations:

| Instance | State | Configuration / applied revision |
| --- | --- | --- |
| alpha | running, healthy | 2 / 2 |
| bus-lab | running, healthy | 1 / 1 |
| comet-check | running, healthy | 1 / 1 |
| import-check | stopped | 1 / 1 |
| panel-check | stopped | 1 / 1 |
| rus | uninitialized | 1 / 0 |
| ssh-check | uninitialized | 1 / 0 |

All three running container IDs match the pre-update baseline. Conventional pier
paths and disabled policy revisions are preserved; discovery reports five enrolled
initialized piers with no move/problem flags. There are no active operations,
recovery requirements or incidents.

Real owned-key bootstrap, owned-pier import, inbound Ames connectivity and
fresh-host installation remain separate live acceptance work.
