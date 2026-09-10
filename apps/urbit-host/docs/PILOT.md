# Disposable remote-runner acceptance test

## What the user needs to provide now

Run the installation and disposable pilot on the workstation and laptop. No
networking key, wallet seed, `+code`, or access to an existing pier is needed.
The first useful report is the workstation's read-only preflight output, remote
`doctor` response, and the disposable ship's `ship inspect` / operation response.
Review outputs before sharing: labels and identities are personal, and runtime
logs or manually executed Hoon can contain secrets.

This artifact is not installed on either of your machines by downloading it.
Nothing has been pushed to a GitHub repository or deployed to `rooftop`.

## 1. Host and client

Use the README installation commands. Confirm on the workstation:

```sh
systemctl --user status omarchy-urbit.service
urbitctl --host local doctor
loginctl show-user "$USER" -p Linger
```

For an always-on user service after logout, enable lingering deliberately:

```sh
loginctl enable-linger "$USER"
```

This may request administrator authorization. The installer does not do it.
Do not alter Docker privileges solely to satisfy this pilot without reviewing
SECURITY.md. Rootless Docker and SELinux-specific bind mounts have not been
validated here; the primary pilot targets the user's existing rootful Docker.

On the laptop, `ssh workstation 'true'` must work with your normal keys and host
verification. Register that alias and run `./scripts/pilot.sh workstation`.
The pilot creates a uniquely named development group and fake `~zod`, then
prints its IDs and follow-on commands. It never purges anything.

## 2. Runtime acceptance

Inspect the operation and ship separately:

```sh
urbitctl --host workstation operation inspect OPERATION_ID
urbitctl --host workstation ship inspect INSTANCE_ID
urbitctl --host workstation ship logs INSTANCE_ID
```

Expected: operation `succeeded`, container `running`, eventually runtime
`http-responsive` and endpoint `http-responsive`. The runtime value is currently
an HTTP proxy for readiness, **not** an independent Arvo/kernel probe. A completed
container start alone is not evidence of successful boot.

An unsupported image or missing runtime tool should produce a stopped/restarting
container and a diagnostic, not silently switch images. Collect the operation,
inspect response, and relevant `journalctl --user -u omarchy-urbit.service` lines.
`ship logs` can expose runtime-produced secrets; only share disposable logs after
review. Never delete a pier lock to force a second runtime into an existing pier.

## 3. Client access

```sh
urbitctl --host workstation ship open INSTANCE_ID
urbitctl --host workstation ship code --clipboard INSTANCE_ID
urbitctl --host workstation ship dojo INSTANCE_ID
```

The browser tunnel listens on a per-host/per-instance `127.x.y.z:8080` address.
This deliberately avoids sharing cookies across several ships on different
ports of the exact same hostname. It requires no public DNS or `/etc/hosts`
changes. SSH forwarding and local proxy records are in the client's runtime
directory. They carry only the ship's public web interface, not its admin
loopback API. Host HTTP publishing is loopback-only.

`Open` invokes the client browser. `code --clipboard` writes to the client
Wayland clipboard without printing the code. Clipboard managers may retain
copied secrets; this implementation does not claim to clear their history.

Dojo uses the GroundSeg-style tmux console inside the existing container.
Detach with **Ctrl-b, then d**. Do not use **Ctrl-d** as a detach shortcut: that
can shut down the ship. Test a harmless expression such as `(add 2 2)`.

Close the browser forwarding session explicitly with:

```sh
urbitctl --host workstation ship close-tunnel INSTANCE_ID
```

## 4. Disconnect, restart, archive

Disconnect laptop SSH/network access and verify the workstation container keeps
running. Reconnect and inspect the same instance ID. `ship list --all-hosts`
should retain last-known inventory but set `reachable: false` / `stale: true`
when a configured host is unreachable. The cache is never permission to start a
replacement ship on another host.

Submit stop and start, waiting for each returned operation before the next:

```sh
urbitctl --host workstation ship stop INSTANCE_ID
urbitctl --host workstation operation wait STOP_OPERATION_ID
urbitctl --host workstation ship start INSTANCE_ID
urbitctl --host workstation operation wait START_OPERATION_ID
```

Inspect and open the same pier again. Test a workstation reboot only with the
disposable group: namespace keeper restart ordering and Docker daemon recovery
are explicit integration checks, not verified assumptions. A crash in a lifecycle
job is marked `interrupted`; inspect before retrying.

Finally:

```sh
urbitctl --host workstation ship archive INSTANCE_ID
urbitctl --host workstation operation wait ARCHIVE_OPERATION_ID
```

The container must be gone and `archive/<instance-id>/pier` retained. Purge is
optional and irreversible:

```sh
urbitctl --host workstation ship purge --confirm INSTANCE_ID INSTANCE_ID
```

`archive` is a move, not a snapshot or backup. Do not mistake it for protection
against drive failure. The empty development network/namespace keeper is retained
for later reuse; group deletion is not implemented yet.

## 5. Same-host fake group check

Create fake `~nec` in the same group as `~zod`. Create a second group with another
fake `~zod`. Confirm identities, Dojo and login codes never cross between groups.
Then test a small Urbit message/poke between the same-group galaxies. This pilot
provides the shared-loopback topology, but actual Ames/Mesa traffic still needs
runtime validation. **Do not place the group's `~nec` on the laptop yet and expect
it to communicate:** that needs the planned cross-tailnet relay in DEVNET.md.

## Real-ship gate: only after acceptance

This gate is an intentional safety brake for this unvalidated pilot, not the
final product's normal restriction. To experiment later with another disposable
real-network identity, stop the host service, set `allow_real_ships` to `true` in
`host.json`, and restart it. Do not do that for an important pier yet.

The explicit commands are:

```sh
urbitctl --host workstation ship boot --comet --label disposable-comet
urbitctl --host workstation ship boot --ship '~YOUR-SHIP' --key-stdin \
  --label disposable-keyed < /path/to/network.key
```

A networking key is **not** a wallet seed or ownership private key. It is sent via
stdin/RPC over SSH, never a command-line argument or Docker environment variable.
It is briefly stored mode 0600 at the instance's `.boot-secret/network.key` to
support host-service restart recovery. After the runtime's identity is verified,
bookkeeping removes the temporary key. Failed/incomplete boots can retain it
until archived or deliberately cleaned up. Filesystem snapshots and SSD storage
can retain old blocks; this is not a secure-erasure guarantee. A Go process may
also retain copies in memory until garbage collection.

A failed keyed boot never silently becomes a comet. No cross-host identity lock
or migration protocol is implemented, so never register/boot the same real
identity independently on another execution host.
