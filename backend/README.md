# Generic Urbit tools

This is the desktop-independent portion of the Omabit monorepo. Neither tool
requires Omarchy, Quickshell or a graphical session.

- `cmd/urbit-runner/`: dedicated host management, installation, lifecycle, storage,
  maintenance and the authenticated runner API.
- `cmd/urbit-collector/`: per-user runner inventory/management and generic Eyre
  application sessions, with JSON and readable CLI output.

From the repository root, build only these tools:

```bash
go build -o build/ ./backend/cmd/...
```

Prebuilt standalone binaries are attached to
[Omabit releases](https://github.com/thelifeandtimes/omabit/releases). Downloading
them does not require installing the Omarchy plugin. Hosting currently targets
Linux x86-64, systemd/cgroup v2 and the documented rootless Docker dependencies.
RC.3 hosting uses an explicitly selected existing Unix owner and defaults to
`/srv/urbit/piers/`. See [owner-user runtime](../docs/owner-user-runtime.md) for
headless installation and ordinary desk editing.

Shared Go implementation stays in the root `internal/` tree under one `go.mod`,
allowing atomic cross-component changes without prematurely creating a public Go
library API. The desktop-specific Go helper remains `cmd/omabit/`; its theme and
clipboard features are not part of these generic executables.

The [RC.13 host-compatibility implementation](../docs/host-compatibility.md)
provides native Pacman/Debian/Ubuntu evidence and package verification, shared
Docker API checks, local `host-check`, and collector fleet reports. Collector-only
machines need no local Docker stack. Unknown host providers remain explicit;
metadata compatibility and native-package tests do not imply full runtime/distro
qualification. Source schema21 preserves historical records and fences older runners.

Historical checkpoint documents refer to the old `cmd/urbit-*` source paths. Current
build/package/CI commands use `backend/cmd/`; executable names and protocol/data
identities have not changed.
