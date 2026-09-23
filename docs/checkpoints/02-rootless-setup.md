# Checkpoint 2A: dedicated rootless Docker setup

Checkpoint 1 was accepted. This checkpoint prepares and optionally installs the
rootless engine that the real runner will use. It does **not** boot a ship, pull
an image, install a production runner daemon, or qualify Vere yet.

Unlike checkpoint 1, the explicitly elevated installation step changes the host.
Stop at any unexpected result and share the error rather than forcing a retry.

## What is implemented

- `preflight`: read-only metadata report; no commands, sockets, or downloads.
- `setup-plan`: the same checks plus a reviewable installation plan.
- `prepare-rootless`: pinned HTTPS downloads and inert review artifacts.
- `install-rootless`: development-only, explicitly authorized first installation.

The eventual plugin will present these phases behind the agreed review and OS
authentication flow. This development checkpoint uses the CLI so we can validate
the installation before building that interface. Package installation and setup
currently require separate explicit administrative actions.

## Changes and boundaries

Installation creates a password-locked, non-login `urbit-runner` system account
with private home `/var/lib/urbit-runner`, a private group, allocated subordinate
UID/GID ranges, and lingering for startup without a desktop login. It installs:

```text
/usr/local/lib/urbit-runner/rootless/29.7.2/
  dockerd-rootless.sh
  check-engine.sh
  daemon.json
  LICENSE
  NOTICE
/etc/systemd/user/urbit-docker.service
```

Only the dedicated account enables `urbit-docker.service`. Its explicit socket is
`/run/user/<assigned-uid>/urbit-docker.sock`; its engine data is under
`/var/lib/urbit-runner/.local/share/urbit-runner/docker`.

No desktop Docker-group grant, existing Docker context change, rootful service
change, global sysctl, firewall rule setup, second static Docker engine, package
auto-upgrade, or privileged fallback is performed. The installed rootless
launcher runs as the dedicated account, never as host root. Docker itself may
configure networking inside its rootless namespaces.

The development helper refuses preexisting account/group, home, installed
artifacts, linger marker, unit, or unit overrides. It is not an updater,
uninstaller, or repair tool. Partial failures preserve their resources and are
reported explicitly. There is no automatic rollback deleting account data.

## Version baseline

This initial qualification candidate deliberately supports Docker **29.7.2** only.
Do not downgrade or freeze system updates to satisfy it. If preflight reports a
different version, stop so we can review a matching launcher and update the test
baseline.

We use the package-managed engine, RootlessKit, and slirp4netns, plus one upstream
launcher from Moby commit `6a43e3d5afddf4111da0f864bbc7cae5d7e95001`.

Launcher SHA-256:

```text
904c9b9e35f6927c0a5e65afb4d35b6bc9eb1278c878044501281fc728c9be46
```

License and notice are also pinned and included. These are project-reviewed
content pins, not a claim of a separate publisher-signed checksum release.
Preparation cannot execute downloads. Installation checks bytes against the
executable's compiled-in pins/templates, not hashes supplied by the bundle's
manifest, and installs the verified byte snapshots without reopening inputs.

## 1. Build and inspect without privileges

From the repository root, as your ordinary user:

```bash
go test ./...
go vet ./...
go build -o ./build/ ./cmd/...
./build/urbit-runner preflight
./build/urbit-runner setup-plan
```

Expected on the currently inspected workstation:

- Platform, systemd, cgroup v2, user namespaces, and Docker version pass their
  static checks.
- `rootlesskit` and `slirp4netns` are missing.
- The dedicated account and linger marker are missing.
- Its groups, subordinate IDs, and private socket remain unknown until setup.
- Existing rootful socket metadata is informational, not a fallback.
- `runtime_qualified` is `false`.

Exit 0 means report generation succeeded, not that all prerequisites passed.
`unknown` is not success. A static executable check does not prove its package
integrity, capabilities, or ability to run a container.

## 2. Prepare and review the bundle

```bash
BUNDLE="$PWD/build/rootless-29.7.2"
./build/urbit-runner prepare-rootless --output "$BUNDLE"
```

Use a fresh directory name if that path already exists. Preparation refuses
overwrites and symlinked paths. It fetches only fixed HTTPS URLs from the pinned
upstream commit, rejects redirects and hash mismatches, and caps downloads.

Inspect these files inside the directory named by `$BUNDLE` before elevation
(not an older bundle left at the example path):

```text
$BUNDLE/manifest.json
$BUNDLE/urbit-docker.service
$BUNDLE/check-engine.sh
```

Expected: bundle directory mode 0700; all files mode 0644, including scripts.
There is no executable installer script in this bundle. Do not run or `source`
the downloaded launcher directly, and do not edit the bundle to bypass checks.
Rebuild and prepare a fresh bundle if source/templates change.

The unit specifies slirp4netns, the private socket, dedicated storage, and
`--group=0`. That group is **namespace GID 0**, which maps to the dedicated
account's host GID; it is not the host's Docker or root group.

You can stop here without any host installation or cleanup beyond optionally
removing your own build artifacts.

## 3. Install missing repository prerequisites

This step requires OS authorization and adds packages. For this workstation the
missing packages are `rootlesskit` and `slirp4netns`:

```bash
omarchy pkg add rootlesskit slirp4netns
```

Review the requested packages before authorizing this command; Omarchy's wrapper
uses pacman's noninteractive installation mode. Use the configured Omarchy
repositories. Do not add AUR packages, swap mirrors, use `pacman -Sy`, or pipe a
remote script into a shell. If repository metadata is stale or versions conflict,
stop and use the normal Omarchy update workflow before regenerating the preflight
report.

If a package download returns HTTP 404 and pacman says the transaction was not
committed, no packages from that transaction were installed. A stale local
database can request an archive the mirror no longer serves. Refresh through a
complete system update, not a standalone `pacman -Sy` or a manual package download:

```bash
omarchy update
omarchy pkg add rootlesskit slirp4netns
./build/urbit-runner preflight
```

`omarchy update` updates the system, not just repository metadata, and may offer a
reboot. Recheck the Docker version afterward; stop on a changed version rather
than bypassing the launcher's version check. If an update/reboot opens a new
terminal, reestablish `BUNDLE` as the absolute path to the bundle already reviewed.
If a download still returns 404 after a successful update, stop and inspect the
mirror state rather than changing mirrors or repeatedly forcing refreshes.

An installer error beginning `missing prerequisite /usr/bin/rootlesskit` or
`missing prerequisite /usr/bin/slirp4netns` means the dependencies are absent, not
that their ownership needs changing. Do not try to solve it with `chmod` or
Docker-group membership. Earlier binaries used an ambiguous ownership/permissions
message for missing files; rebuilding gives the clearer diagnostic.

Other Quattro systems may need additional prerequisites reported by preflight;
the helper does not install them automatically. In particular, newuidmap and
newgidmap come from Arch's `shadow` package. Their packaged file capabilities are
valid; do not add setuid bits or modify global user-namespace policy blindly.

```bash
./build/urbit-runner preflight
```

The two executable checks should now pass. Account-related missing/unknown checks
are still expected. Stop for a version mismatch or blocked kernel prerequisite.

## 4. Authorize the development first install

Review `setup-plan` first. This invocation explicitly authorizes the fixed host
changes described above, then starts only the dedicated account's engine:

```bash
sudo ./build/urbit-runner install-rootless --bundle "$BUNDLE" --acknowledge-host-changes
```

This authorizes the locally built executable from the reviewed checkout. A
production release will use packaged, trusted installation artifacts instead of
executing a development binary through sudo.

Expected successful result:

```json
{
  "account_created": true,
  "account": "urbit-runner",
  "uid": 942,
  "gid": 942,
  "service": "urbit-docker.service",
  "socket": "/run/user/942/urbit-docker.sock",
  "rootless_verified": true,
  "runtime_qualified": false,
  "partial": false
}
```

The UID/GID above are examples, not fixed allocations. Rootless verification here
means the explicitly addressed engine reports the required version, `name=rootless`,
systemd cgroup v2, and dedicated data root. It is not yet a test of real container
resource enforcement, networking, or Vere.

An installer failure exits 1, with progress on stdout and a JSON error on stderr.
A CLI usage refusal (such as missing acknowledgement) exits 2 with only the JSON
error on stderr and no installation attempt. If
`partial` is true, some changes may have occurred even if a corresponding success
field is false. Stop and share both outputs. Do not rerun with a different
account, delete state, add Docker-group membership, disable system Docker, or use
an upstream installer's force option to work around the failure.

## 5. Verify isolation and service state

```bash
id urbit-runner
getsubids urbit-runner
getsubids -g urbit-runner
loginctl show-user urbit-runner -p Linger -p RuntimePath
sudo systemctl --user --machine=urbit-runner@.host is-enabled urbit-docker.service
sudo systemctl --user --machine=urbit-runner@.host is-active urbit-docker.service
systemctl show docker.service docker.socket --property=Id,ActiveState,UnitFileState
./build/urbit-runner preflight
```

Expected:

- The account has only its private group, not `docker` or `wheel`.
- Subordinate ranges have at least 65,536 IDs and do not reuse the desktop's range.
- Lingering is enabled; the private user service is enabled and active.
- Existing system Docker state is unchanged from before installation. On the
  inspected machine its service was inactive/disabled and socket active/enabled.
- An ordinary desktop preflight may report private runtime/socket metadata as
  unreadable (`unknown`), which is expected isolation, not an instruction to
  widen permissions.
- The report still says `runtime_qualified: false`.

There is no requirement to change your Docker context. An ordinary `docker`
command still targets your prior context, not the new account's engine.

For diagnostics, inspect only the dedicated service:

```bash
sudo journalctl _UID="$(id -u urbit-runner)" _SYSTEMD_USER_UNIT=urbit-docker.service --no-pager -n 80
```

## 6. Restart and acceptance

No ship exists yet, so restarting this dedicated engine is safe at this checkpoint:

```bash
sudo systemctl --user --machine=urbit-runner@.host restart urbit-docker.service
sudo systemctl --user --machine=urbit-runner@.host is-active urbit-docker.service
```

Record whether this succeeds. Reboot qualification is separate and should happen
when convenient, not as an automatic installer action. After a later reboot,
repeat the enabled/active checks without logging in as the service account.

Report preflight differences, whether installation succeeded, the result JSON,
and the service/isolation/restart checks. This is the gate before pulling a pinned
Vere image and booting a disposable fake ship using the upstream default pill.
Actual Ames connectivity, resource limits, offline maintenance, and control-socket
safety remain future qualification checks.

## Non-destructive stop

If you want to leave the engine off while we review results:

```bash
sudo systemctl --user --machine=urbit-runner@.host disable --now urbit-docker.service
sudo loginctl disable-linger urbit-runner
```

This preserves the account, IDs, installed files, and engine data. It is not a full
uninstall. Do not use `userdel -r`, change subordinate ranges after data exists,
or delete data directories as a retry shortcut.

## Developer verification

Automated fixtures cover read-only evidence, unsafe account/ID configurations,
download failures, hash verification, modified bundles, unauthorized invocation,
first-install refusal, command/environment boundaries, partial failures, and
rejection of a wrong engine/socket. No test invokes the privileged installer on
the actual host.

Live validation so far is limited to preflight, HTTPS bundle preparation, and
shell syntax checks. Systemd's static verifier reports the expected missing
installed executable paths before installation. The dedicated engine has **not**
yet been installed or started by the developer; this walkthrough establishes that
next layer of evidence.
