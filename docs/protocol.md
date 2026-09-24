# Runner protocol: checkpoint 1

Status: experimental version 1. The original `simulation` server implements this
checkpoint contract. The `development-managed-fake` server adds the durable
extension below. Neither mode is a supported production runtime manager.

## Transport and negotiation

HTTP/JSON over a private Unix socket. The runner does not bind a TCP port. The
socket parent must be an existing, owned, private directory; paths must be
absolute and clean. Existing socket/files are never replaced. Socket permissions
are 0600. This trusts the local account, not an application sandbox.

`GET /hello` returns protocol version, runner ID, mode, and features. It needs no
version header. All other requests require `Urbit-Runner-Protocol: 1`; missing or
different values return HTTP 426 and `protocol_mismatch`. Responses carry the
same header. Each collector CLI invocation negotiates before further requests.

The simulator always reports `mode: "simulation"`. Its runner ID is new on each
launch. State, accepted request IDs, and history do not survive runner restart.
Do not retry uncertain operations across a simulator restart expecting durable
idempotency.

Mutations require `Content-Type: application/json`, one JSON value, no unknown
fields, and a maximum 4096-byte body. Browser Origin headers are rejected; the
future authenticated gateway is a separate surface. Responses are not cacheable.
The client caps response bodies at 1 MiB and refuses redirects or incompatible
protocol responses. Mutations are never automatically retried; transport errors
report an uncertain outcome, since acceptance may already have occurred.

## Endpoints

| Method | Path | Result |
| --- | --- | --- |
| GET | `/hello` | Protocol, runner ID, mode, features |
| GET | `/v1/snapshot` | Instances, operation history, incident snapshot |
| PUT | `/v1/instances/{id}/configuration` | Updated instance |
| POST | `/v1/instances/{id}/operations` | Accepted or replayed operation |

IDs in the CLI/client use 1..64 ASCII letters, digits, underscores, or hyphens.
Only the preseeded instances `alpha` and `bravo` exist. They are not real ship
identities. There is no create/import/pier-path or arbitrary-command endpoint.

Snapshots expose desired and observed state, desired configuration revision,
applied revision, and an optional active operation ID separately. `alpha` starts
with a synthetic failure incident; `bravo` starts stopped. Starting alpha resolves
the synthetic incident without removing its history. No telemetry is fabricated.

### Configuration

```json
{
  "expected_revision": 1,
  "config": {
    "http_port": 8080,
    "ames_port": 34343,
    "loom": 32
  }
}
```

The revision must match current configuration. Actual changes increment it; a
no-op does not. Configuration stays pending until a simulated start applies it.
Changes during an operation are rejected. Each instance reserves its configured
HTTP/TCP and Ames/UDP port independently; same-protocol conflicts are rejected.
These ports are labels in the simulator, not actual bound listeners.

Demo validation accepts nonzero 16-bit ports and loom exponents 31..33. This is a
demo constraint, not an assertion about all Vere versions. The production runner
must qualify and validate the selected runtime's limits.

Stale writes return HTTP 409:

```json
{
  "error": {
    "code": "revision_conflict",
    "message": "configuration changed; inspect a fresh snapshot before editing",
    "current_revision": 2
  }
}
```

The client must show the conflict; it must not fetch the new revision and silently
resubmit an operator's stale edit. Configuration PUT has no idempotency key at
this checkpoint; reconcile an uncertain response through a fresh snapshot.

### Operations

```json
{
  "request_id": "walkthrough-start",
  "kind": "start",
  "expected_revision": 2
}
```

Only `start` and `stop` are supported. New accepted requests return HTTP 202 with
an operation ID and `status: "running"`. Completion occurs after the configured
demo delay and becomes `succeeded` with `finished_at`. Work belongs to the runner
process, not the accepting HTTP connection. A runner shutdown discards it.

Request IDs are scoped to the current simulator process. Exact replays return
HTTP 200 and the same operation, including its current status. Reusing an ID for
a different instance, kind, or expected revision returns `request_conflict`.
Replay lookup precedes revision, busy-state, and capacity checks. A client should
retain its request ID to reconcile an uncertain submission.

Incompatible new operations return `instance_busy`; unrelated instances can work
concurrently. Start on an already running instance and stop on an already stopped
instance return `invalid_state`. To apply pending config on a running instance,
explicitly stop then start it.

The simulator stamps `actor: "local-uid:<effective runner UID>"`. It relies on its
private same-account socket, does not inspect SSH keys, and does not demonstrate
real multi-user attribution. Caller-supplied actor fields are rejected. A real
runner will derive identity from authenticated transport context.

The simulator retains at most 256 accepted operations and never evicts request
identity to make room. At capacity, new work returns `demo_capacity`; existing
requests remain replayable. Restarting clears all demo state.

## Error and CLI conventions

Errors are `{"error":{"code":"...","message":"..."}}`, optionally carrying a
`current_revision`. HTTP 400 means invalid input, 403 disallowed browser origin,
404 missing instance/route, 409 a state/revision/request conflict, 415 wrong
content type, 426 protocol mismatch, and 503 shutdown/demo capacity.

The collector prints successful JSON to stdout and errors to stderr. Exit codes
are 0 for success, 1 for request/transport failures, and 2 for usage errors.
`watch` polls snapshots and emits newline-delimited JSON. Cancellation exits 0;
connection failure exits 1 rather than presenting stale state as current. There
is not yet an event stream or automatic reconnect loop.

The collector's multi-runner `inventory --watch` command described below has a
separate per-connection polling/reconnect contract; single-runner `watch` retains
the fail-on-connection-error behavior above.

## Checkpoint 3A: durable development extension

`hello.mode` is `development-managed-fake`; features include `create-fake`,
`durable-operations`, and `local-peer-credentials`. The collector checks the
creation capability before sending a new registration. Existing simulation
behavior remains available and memory-only.

`POST /v1/instances` accepts:

```json
{
  "id": "alpha",
  "label": "Workstation test",
  "request_id": "register-alpha",
  "config": {"http_port": 18080, "ames_port": 34343, "loom": 31}
}
```

It registers a fresh fake instance without booting it and returns a completed
`register-fake` operation (201 new, 200 exact replay). Start/stop retain the
existing operations endpoint, now with durable `queued`, `running`, `succeeded`,
`failed`, and `interrupted` states. `message` optionally describes a failure.
All mutations, including configuration edits, are attributed to the Unix peer
obtained using SO_PEERCRED; JSON/header actor overrides are not trusted.

Request IDs are scoped to the stable runner and authenticated actor and survive
restart. Replays are checked before current configuration/busy-state checks.
Changing an existing ID's input is a conflict. Configuration continues to use
explicit compare-and-swap revisions, without automatic retry. Its audit entry
uses a server-generated request ID because the configuration endpoint itself has
no caller idempotency key.

Instance responses add optional `mode`, `ship`, `endpoint`, `health`,
`recovery_required`, and `observed_at`. Desired configuration can differ from the
applied revision/endpoint until stop/start. The desired and still-applied HTTP
ports remain reserved while needed. Observations older than 30 seconds display
unknown health rather than claiming current reachability.

Snapshots return the latest 128 operations, but older replay records remain
stored. SQLite transactions atomically record registration, revision changes,
operation acceptance, and runtime intents. Runner restart marks unfinished work
interrupted and observes the runtime; it does not blindly re-execute requests.

Development constraints: private same-account/root Unix clients, at most eight
registered fake ships, pinned image/pill, loom 31, reserved/unpublished Ames value
34343, and IPv4 loopback HTTP ports 1024..65535. Keys, pier import, deletion,
maintenance scheduling, SSH authorization, and multi-operator key attribution
remain later milestones. Stopping management leaves running ships alive.

## Checkpoint 4A: collector enrollment and inventory

This is a collector extension; runner protocol version 1 and existing runners
remain compatible. No runner-side endpoint or authorization change is required.

```text
urbit-collector [--config-dir DIR] enroll NAME --socket /absolute/runner.sock
urbit-collector [--config-dir DIR] runners
urbit-collector [--config-dir DIR] --runner NAME hello|snapshot|watch
urbit-collector [--config-dir DIR] --runner NAME configure|operate|create-fake ...
urbit-collector [--config-dir DIR] inventory [--watch] [--interval 1s]
urbit-collector [--config-dir DIR] forget NAME --runner-id EXPECTED_ID
```

Global options precede the subcommand. `enroll` takes its socket after NAME.
`--format text` selects readable tables for snapshots, inventory, and watches;
JSON remains the default, with NDJSON for watch output.
Existing `--socket PATH` command selection remains available and is mutually
exclusive with `--runner NAME`. There is no implicit default runner for mutations.
UI callers can add `--expected-runner-id ID` beside `--runner NAME`. This binds a
cached UI selection to its observed runner even if another client forgets and
re-enrolls that name before the command executes. A mismatch stops before mutation.

Enrollment contacts `/hello`, checks protocol compatibility, and saves the
connection name, transport (`unix`), absolute socket path, runner ID, mode, and
enrollment time. The same enrollment is a no-op. Conflicting names, identities,
or socket addresses are refused rather than updated silently. At most 32 runners
are enrolled in one registry. `forget` checks the expected identity and removes
only collector-local enrollment, even when offline.

Registry format version 1 lives in `runners.json` under
`$XDG_CONFIG_HOME/urbit-collector`, or `~/.config/urbit-collector` by default.
`--config-dir` supplies an explicit alternative. The directory is owned/private
0700 and files are 0600. Updates use an interprocess lock, file sync, atomic rename,
and directory sync. Invalid/unknown registries are refused rather than reset.
Credentials, piers, and runner journals are not stored there.

Named connections verify runner ID and mode through `/hello` on **every newly
dialed connection**, before any command is sent on it. This includes reconnects
between initial negotiation and a mutation. Snapshot identity must also match.
`runner_identity_mismatch` blocks use without overwriting the saved pin. This pin
detects unexpected replacement; it is not cryptographic authentication. Local
socket permissions remain the authentication boundary, and actor attribution
still comes from the runner's existing local transport. Two collector directories
are not two independently authenticated operators.

`inventory` returns:

```json
{
  "connections": [
    {
      "name": "workstation",
      "runner_id": "runner-example",
      "mode": "development-managed-fake",
      "status": "available",
      "checked_at": "2026-09-24T02:28:09Z",
      "snapshot": {"runner_id": "runner-example", "mode": "development-managed-fake", "instances": [], "operations": [], "incidents": []}
    }
  ],
  "complete": true
}
```

Instances remain nested under their connection and stable runner ID; identical
instance IDs or fake ship names across runners do not merge. An unavailable or
identity-mismatched connection carries an `error` instead of a current `snapshot`.
It does not create a runner incident or declare any ship crashed. There is no
persisted observation cache in 4A; `checked_at` records the collector's check,
while nested `observed_at` retains runner-provided freshness.
Available observations also carry Hello's optional `access_role` and `features`
for UI controls. These are presentation hints; the runner authorizes each request.

One-shot inventory polls concurrently with a five-second budget per connection.
It prints the complete result structure, including failures, and exits 1 if any
connection is unavailable/mismatched (`complete: false`); otherwise it exits 0.
An empty inventory is a complete empty result. Diagnostic messages in combined
inventory are fixed/redacted rather than arbitrary remote error text.

`inventory --watch` emits one compact connection observation per line as each
poll completes. Each connection has an independent loop; slow hosts do not hold
up updates from healthy hosts. It retries observation only, retains identity pins,
and continues through connection errors. The interval is a delay after each poll,
at least 100 ms (default 1 s). Ctrl+C exits 0; an output failure exits 1. The
enrollment set is fixed for the invocation: restart the watcher to pick up changes.
Watching an empty registry is an error. Mutations retain explicit revisions and
request IDs, with no automatic retry.

## Checkpoint 4B: restricted SSH access

`enroll NAME --ssh HOST_ALIAS [--ssh-config /absolute/file]` uses OpenSSH's normal
host/user/key configuration and a fixed remote command `urbit-runner-rpc-v1`.
The collector forces batch mode and strict host-key verification, disables
forwarding, PTYs, local commands, and connection multiplexing, and retains runner
identity pinning on every connection. SSH keys remain in OpenSSH/agent custody;
the enrollment registry stores only the alias and optional config path. Version-1
registry records now allow `transport: ssh` and an `ssh` object in place of `socket`.
Older collectors refuse the unknown fields rather than discard SSH enrollments.

The reviewed authorized_keys format supports Ed25519 public keys with `operator`
or `observer` roles. Each key has `restrict` and a forced command containing the
fingerprint derived from that key. A restricted login-shell invocation checks the
original command and that binding, then bridges bounded HTTP/JSON to the private
runner socket. The SSH client cannot choose the actor through JSON, headers, or
command arguments. Arbitrary shell commands, port/agent forwarding, PTYs, and
application proxying are unavailable on this surface.

The bridge constructs fresh local headers. Before forwarding a request it checks
delegation support, actor, and role through Hello on the **same local connection**.
This refuses older runners before they can accept an unattributed mutation.
The managed runner accepts the delegation header only after its existing Unix
peer check (service account or root) and independently loads the public-key grant
on each request. That Unix peer is a trusted host authority, not an untrusted
remote client. Administrators with direct service-account/root access can already
change the journal; this does not claim isolation from a host administrator.

Hello adds optional `actor` and `access_role`. The managed runner advertises
`ssh-key-grants` when configured for delegation. Remote operations/configuration
use `actor: ssh-key:SHA256:...`, and request identity stays scoped to that actor.
Observers may only request Hello and snapshots; mutation requests return
`ssh_access_denied`. Deleting a grant blocks future requests, including on open
sessions, without cancelling accepted jobs. The bridge and client never retry
mutations. Transport failure still requires reconciliation of an uncertain result.

`prepare-ssh` writes inert review files for a standalone loopback SSH listener.
Its installation and account-shell change are explicit administrative steps in
[checkpoint 4B](checkpoints/07-ssh-access.md). The protocol adds no runner TCP
listener. SSH stderr is not included in collector JSON or combined inventory;
diagnose deployment/authentication errors through the server's local log.
