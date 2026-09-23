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
