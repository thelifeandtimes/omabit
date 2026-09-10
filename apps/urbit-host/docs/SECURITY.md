# Security and operational boundaries

## Threat model

The pilot assumes one trusted Linux account per execution host, trusted container
images and operator-administered SSH/Docker/Tailscale. It is not a hostile
multi-tenant container platform. Rootless Docker, LSM profiles and a hardened
privilege-separated Docker broker remain untested.

The host daemon opens no TCP management listener. Owner and agent sockets are
mode 0600 inside a mode-0700 user runtime directory. The SSH gateway forwards a
single strict JSON request. Its role is selected by the socket/gateway and
cannot be supplied in JSON. Remote input is stdin, not shell-interpolated text.
Host aliases may contain normal SSH user/host syntax, not arbitrary options.
SSH host verification is not disabled and SSH agent forwarding is disabled.

Docker uses the existing daemon. Access to a rootful Docker daemon is effectively
administrator access: this program cannot constrain a person or model that can
call Docker directly. The manager does not mount the Docker socket into ships.
Containers run with the account's numeric UID/GID and a reduced capability set;
those settings need validation against the selected runtime image. The ship
container receives minimal read-only passwd/group overlays describing that numeric
account; no host /etc files are modified or exposed.

## Agents

The `--agent` option helps an honest agent choose the restricted interface. By
itself it is **not an authentication boundary**: any process with the operator's
unrestricted SSH credentials or a same-UID shell can choose the owner socket or
bypass the manager.

For remote agents, issue a dedicated SSH key and use a forced command on the
host's `authorized_keys` entry. Replace the absolute account path and public key:

```text
restrict,command="/home/YOUR_USER/.local/bin/urbitctl rpc --agent" ssh-ed25519 AAAA... agent-urbit-dev
```

Use an SSH config alias on the agent machine that selects ONLY this key, with
`IdentitiesOnly yes`. Do not give the agent another unrestricted key, another
route to the Docker socket, or filesystem access as that account. `restrict`
also disables forwarding and PTY access; that is intentional. Agent development
Hoon uses the RPC control command, not interactive SSH/tmux.

The fixed agent role allows development creation/lifecycle/Hoon/code/log access,
plus read-only inventory and metrics for real instances. It denies real ship
creation, lifecycle, code, logs, Hoon and Dojo. Real owner lifecycle/Hoon operations
require the exact instance ID as confirmation. A program already holding an
operator credential can supply that ID; this is an accident-prevention measure,
not a sandbox. There is no approval-token delegation for real operations yet.

## Browser and runtime control

A public ship HTTP endpoint is published only on host loopback and forwarded to
client loopback over SSH. Urbit's unauthenticated administrative loopback HTTP
port is never published. `+code` / Hoon RPC runs inside the particular container,
reading that pier's exact `.http.ports` loopback entry, with no guess at port 12321.

A fake environment shares one network namespace and therefore its administrative
loopback endpoints. **All ships in that development group share trust.** Separate
groups isolate namespaces; real ships never join them. Do not put real keys or
untrusted tenants into fake groups. A future relay must preserve this boundary.

Different instances get different browser loopback IP hosts, not merely different
ports on `localhost`. The pilot fails closed if an unknown process occupies the
chosen listener. Hash collisions are possible and reported; automatic alternate
address allocation is not implemented yet. Same-UID local processes and other
host users that can reach loopback are not prevented from accessing a ship's
public login page; normal Urbit authentication still applies.

`+code` returned through RPC is private in transit over SSH but is still a secret.
`--clipboard` avoids printing it; `--show` deliberately outputs it. Clipboard
history is outside this program's control. Human Dojo sessions and arbitrary
runtime output may contain secrets; runtime logs are not guaranteed secret-free.
The runner does not inject key contents or automatic `+code` results into logs.

## Storage, delete and recovery

No management command automatically removes a running pier, escalates SIGTERM to
SIGKILL, deletes runtime lock files, boots an unreadable pier afresh, or falls back
from a failed keyed boot to a comet. Archive stops/removes the container and moves
the directory inside the data root. Purge is a distinct irreversible action.

Path traversal and symlinked management paths are rejected. This is not a
race-proof filesystem sandbox against a hostile same-UID process. Atomic metadata
writes are fsynced and renamed; external Docker and filesystem changes are not a
single atomic transaction. Interrupted jobs remain visible for inspection.

No backups are provided. No automated failover or real-identity fencing across
independent hosts is provided. Never boot two live copies of a real pier. Purged
metadata retains request IDs so old create requests cannot silently recreate data.

For keyed boot, transient secret-file handling is documented in PILOT.md. Stored
piers themselves contain sensitive state and need the same care as networking
keys. No disk encryption or secure erase is implemented by this manager.
