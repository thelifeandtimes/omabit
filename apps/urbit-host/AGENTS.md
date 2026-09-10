# Omarchy Urbit — contributor instructions

## Begin here

This is an existing Go implementation, not a blank-slate project. Read
`docs/HANDOFF.md`, `docs/DECISIONS.md`, and `docs/STATUS.md` before planning changes.
Read `docs/SECURITY.md` before touching permissions, SSH, Docker, Hoon, or secrets;
read `docs/PILOT.md` before running live integration tests. Load `docs/DEVNET.md`
when working on fake-ship networking and `docs/SOURCES.md` for upstream references.
Do not assume this session has access to the original planning chat.

`docs/DECISIONS.md` records product direction; the source defines current
behavior; `docs/STATUS.md` records the extent of validation. Keep all three
consistent. Preserve decisions unless the user deliberately changes them.

## Immediate priority

Audit and reproduce the source build, then validate one disposable fake `~zod`
on the workstation, managed from the laptop over SSH/Tailscale. Fix runner
compatibility and lifecycle failures before building the QML UI, deep metrics,
or the cross-tailnet relay. The UI and relay are requirements, not implemented
features. Do not interpret simulated Docker tests as a successful Vere boot.

## Commands and source map

```sh
make check          # race tests, go vet, shell syntax checks
make build          # builds ./urbitctl from this checkout
./urbitctl version
```

The module currently specifies Go 1.23. Race tests require a working native
C toolchain. See the Makefile rather than inventing build commands.

- `cmd/urbitctl/`: CLI parsing, command dispatch, client interaction.
- `internal/fleet/`: Docker lifecycle, state, jobs, metrics, runtime notices.
- `internal/fleet/scripts/`: embedded container/runtime adapter scripts.
- `internal/transport/`: SSH and Unix-socket RPC, fixed-role access policy.
- `packaging/`: per-user systemd unit.
- `scripts/`: preflight, installer, and disposable pilot.
- `skills/`: operational instructions for agents using the product. These are
  distinct from this contributor file.

Use gofmt, targeted tests, and then `make check`. Add regression coverage for
bugs. Report what actually ran and distinguish simulated, local, and remote
checks. Prefer bounded changes over an unsolicited rewrite.

## Safety and scope

Keep real-network creation disabled during this pilot. Do not use real keys,
existing valuable piers, or `rooftop` as test targets. Ask for the intended
execution host and permission before first installing a service, booting a
container, or changing machine configuration. Repository read/edit/test work
does not authorize infrastructure changes.

Never auto-failover a real identity, treat an unreachable host as stopped,
delete pier locks, silently replace a failed keyed boot with a comet, or weaken
SSH verification. Archive and purge remain distinct. Do not add backups, hard
resource limits, cloud-provider provisioning, or hosted-Eyre support to the
current scope.

Keep piers, keys, `+code`, private host configuration, SSH credentials, and raw
runtime logs out of the repository and agent transcripts. Redact integration
reports. Repository instructions and `--agent` do not sandbox a process with
unrestricted shell/Docker access; read the credential boundary in SECURITY.md.

## Development installation gotchas

The original installer prefers `dist/urbitctl-linux-<arch>` over building source.
This handoff is intentionally source-only. Do not reintroduce stale binaries and
then assume the installed executable includes recent edits.

The checkout's `./urbitctl`, installed `~/.local/bin/urbitctl`, already-running
host daemon, and existing ship containers are separate lifetimes. Rebuilding
one does not automatically replace the others. Plan and test deployment
explicitly; do not restart services in the middle of an operation.

Runtime scripts are embedded and materialized under a versioned directory.
`Engine.InstallScripts` refuses changes at an existing version. Review
`internal/fleet/docker.go` and `internal/fleet/types.go` before changing embedded
scripts; use a new version or an explicitly isolated disposable data root,
not an overwrite of scripts mounted into live containers.

## End each implementation session

Update `docs/STATUS.md` and the next-step section in `docs/HANDOFF.md` when
behavior or evidence changes. Record the current commit in live-test reports.
Explain remaining blockers and the next executable step. Do not describe a
feature as integrated merely because its command construction has a unit test.
