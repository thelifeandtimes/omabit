# Owner-user runtime and ordinary pier file access

RC.3 uses the existing Unix user selected during installation for both the runner
and its separate rootless Docker engine. Container UID 0 maps to that host user.
Vere-created pier/desk files and files copied from the user's shell therefore have
the same host owner. No ACL-repair daemon, FUSE view or per-file chown is involved.

## Fresh installation

```bash
omarchy plugin install https://github.com/thelifeandtimes/omabit.git --enable
```

In **Setup → Set up this computer**, the desktop selects its own Unix user and asks
for OS authorization. The host step verifies the existing account, allocates missing
non-overlapping subordinate IDs, enables lingering, installs owner-bound services
and creates **`/srv/urbit/piers/`** for that owner. It does not change the user's login
shell, personal SSH configuration, group membership or unrelated Docker services.

The equivalent headless installation is:

```text
urbit-runner prepare-rootless --output /path/to/new-bundle
sudo urbit-runner install-host --owner YOUR_USER --bundle /path/to/new-bundle \
  --public-key /path/to/operator.pub --acknowledge-host-changes
```

The owner must already exist. One owner installation is supported per host by the
current service/socket configuration. Additional collectors use distinct management
keys; they do not require a second runner or Docker daemon.

## Where data lives

- Human-facing default: `/srv/urbit/piers/`.
- Piers retain the category/exclusive-parent layout, for example
  `/srv/urbit/piers/fakeships/dev/pier/`.
- Engine data: `~/.local/share/urbit-runner/docker/`.
- Runner database and metadata: `~/.local/share/urbit-runner/managed/`.
- Explicit rootless socket: `/run/user/UID/urbit-docker.sock`.
- Management socket: `/run/user/UID/urbit-runner/runner.sock`.
- Root-owned installation profile: `/etc/urbit-runner/owner.json`.

The profile binds the owner UID/GID, home and default root. A changed owner or
default root requires explicit migration; setup does not recursively chown or
silently reinterpret an existing installation. Missing or invalid profile evidence
does not authorize another user to operate this engine.

## Ordinary desk editing

In the ship:

```hoon
|mount %some-desk
```

In the owner's ordinary shell on the runner host:

```bash
ls /srv/urbit/piers/fakeships/dev/pier/
cp -R developer-directory/some-desk/. /srv/urbit/piers/fakeships/dev/pier/some-desk/
```

Then in the ship:

```hoon
|commit %some-desk
```

These are direct host directories bound into Vere, not asynchronously synchronized
copies. Desk editing works while the ship runs. Existing Unix permissions still
apply, just as with bare-metal Vere; the application does not require a second
service account to be granted access to the user's files.

## Additional locations

Storage locations can use new directories under the owner's home or another
normally writable path. Missing directories are created as that user. A selected
shortcut is resolved to its canonical directory before identity pinning. Private
management state is excluded. The runner does not elevate to commandeer a directory
belonging to someone else, nor silently retarget a configured location later.

Normal group-writable, owner-controlled workspace paths are supported; world-writable
trees and foreign-owned pier data remain subject to storage checks. `.urb` state
retains its private ownership/permission checks. Cross-host transfers normalize
copies to the destination runner's owner instead of preserving a foreign numeric UID.

## Management access and runtime isolation

The separate loopback SSH endpoint uses root-owned, key-specific forced commands
and verifies the exact RPC request. It uses the owner's ordinary account without
replacing its login shell or personal authorized_keys. Operator/observer roles and
per-key revocation remain in effect for remote requests. Local processes belonging
to the same Unix owner have that owner's filesystem authority, as with bare-metal
Vere.

The engine still verifies rootless mode, the pinned version, cgroup v2 and its
isolated data root. Container mount restrictions, dropped capabilities, no-new-
privileges and resource limits remain. It never falls back to the system Docker
socket. Ships run through the user manager with lingering, independently of the
desktop plugin and collector sessions.

## Legacy hosts and updates

Existing dedicated-account installations are refused by the new owner installer
until explicitly retired or migrated. Their piers, account and grants are not
automatically repurposed. Schema 16 prevents older binaries from ignoring the new
storage/runtime contract; store upgrades alone do not migrate an OS account.

For an existing owner-user installation, `install-host --owner ...` reuses reviewed
artifacts, identities and grants, and updates management when no operation is active.
The historical `install-services.sh` is not the owner-user update path. The new UI
refuses local setup with an older helper that lacks the owner-user setup contract.
