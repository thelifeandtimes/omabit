# Runner protocol: checkpoint 1

## Development storage extension (schema 14)

Capabilities `storage-locations`, `import-pier`, and `delete-pier` add operator-only
storage actions. `POST /v1/storage-roots` binds a location name to an absolute owned
directory; `POST /v1/pier-import-preview` reviews a stopped host source, and
`POST /v1/pier-import` journals the approved move/enrollment. Deletion is a
`delete-pier` instance operation with `confirm_delete: true` and the selected
configuration revision. All mutations carry actor-scoped request IDs.

The pier catalogue includes `roots`. Qualified paths such as
`@ssd/fakeships/test/pier` select a pinned additional root; unqualified paths always
select the original default. Snapshots expose pending deletion and external-import
phase/holding-path/progress so interruptions remain visible. Completed deletion
retains operation history and the final revision; a reused instance name receives
a new runtime ID and a strictly greater initial configuration revision.

See [storage management](storage-management.md) for the filesystem contract and GUI
workflow. Current schema 16 upgrades schema 11–15 databases additively; older
binaries refuse the upgraded database.

### Owner-user extension (schema 16)

Installed owner hosts advertise `owner-user-runtime` and `human-pier-storage`.
The root-owned installation profile binds the existing Unix owner and default
`/srv/urbit/piers/` root. Custom roots use normal owner permissions and pin a
canonical directory. OS account migration remains explicit and separate from
database migration. See [owner-user runtime](owner-user-runtime.md).

### Reboot resume extension (schema 15)

Automatic-resume services additionally advertise `unclean-reboot-resume`. Private
state records the last healthy host boot, container and configuration revision and
durable pending authorization. A later host boot can authorize one initialized-pier
resume despite a nonzero exit. Operations retain kind `startup-resume` and actor
`system:startup`; no caller-accessible startup RPC is added. See [services](services.md)
for migration, once-per-boot accounting and independent recovery guards.

Status: experimental version 1. The original `simulation` server implements this
checkpoint contract. The `development-managed-fake` server adds the durable
extension below. Neither mode is a supported production runtime manager.

## Transport and negotiation

### Runtime diagnostics extension (0.19)

Managed snapshots include optional `instance.runtime_status`: `stage`,
`initialized`, container ID/kind, observed running/exit/OOM
evidence with its own timestamp, and the last lifecycle operation. This is
diagnostic metadata; it does not override operation/revision/recovery guards.

From 0.21, `default_snapshot_minutes` reports the new-launch policy (60), and
optional `snapshot_minutes` reports the recorded launch configuration of the shown
runtime container. An older omitted-flag launch reports Vere 4.6's two-minute
default until replaced; manager upgrades do not relabel it as already changed.

The `runtime-logs` capability enables operator-only
`POST /v1/instances/:id/logs`, body `{"expected_revision":N}`. This is a read-only
bounded tail, available during active operations and recovery. Responses bind
runner ID, instance ID, configuration revision and container ID; include `source`
(`docker`, `retained`, `none`), `observed_at`, `lines`, and `truncated`. Both Docker
streams are decoded; terminal controls and known credential patterns are removed.
Output is capped at 200 lines / 64 KiB, with a 1 MiB raw transport ceiling. Errors
do not reflect raw backend output. A changed runtime/configuration rejects the read.
Logs are not included in general fleet snapshots. A bounded sanitized tail is
retained when an owned container is removed; older removed containers may have none.

CLI: `urbit-collector --runner NAME --expected-runner-id ID logs INSTANCE --revision N`.

### Base transport

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

## Named development fakes

### First-boot selection (0.23)

Capability `bootstrap-selection` extends new fake/comet registration with optional
`bootstrap:{kind:"bare"}` or `bootstrap:{kind:"recommended",pill:{sha256,bytes}}`.
Omitted selection on a new registration defaults to bare. Bare forbids a pill
descriptor; recommended requires a valid, already staged artifact. Imports cannot
request bootstrap selection. The complete descriptor participates in the durable
request fingerprint; replay precedes artifact-availability checks.

Owned-key registration retains its existing optional `pill` field: absent means
bare, present means the selected Recommended Apps content. Snapshots expose the
chosen `bootstrap` on new registrations. Selection is immutable after registration.
The collector exposes `--bootstrap bare|recommended` and `--pill URL_OR_FILE`;
recommended without a source is an error, never a fallback.

Bare causes no pill download in the collector/runner and passes no `-B`/`-u` to
Vere. Recommended content is uploaded and verified before registration, then bound
read-only on first boot. Subsequent initialized starts need neither the pill nor
its cache entry. Bootstrap choice does not change Ames publication or ship mode.
Schema 13 adds downgrade protection for this selection; existing schema-11/12
records retain their prior bootstrap/runtime metadata, including uninitialized
registrations and handoff fences.

### Owned boot defaults and pill upload (0.17)

`key-boot-options` extends owned registration. Omitted HTTP/Ames ports are selected
atomically from available unprivileged ports, considering registrations, still-applied
ports and host listeners. Replay precedes allocation. The instance ID and tier path
are derived from the expected @p. The API keeps explicit network intent, while the
desktop's Register action supplies it without another required form control.

An optional `pill:{sha256,bytes}` selects an already staged artifact. Blank means
canonical Vere bootstrap with no `-B` override. Operator `POST /v1/boot-pill` accepts
a bounded binary body, explicit Content-Length and `Urbit-Runner-Pill-SHA256`.
The receiver verifies content before publishing it to the private cache. Upload is
inert and creates no ship. The SSH bridge streams this exact bulk route, after
grant validation and same-connection delegation handshake; ordinary requests retain
their 4096-byte limit. Upload has a bounded ten-minute deadline and no automatic retry.

New owned runtime commands and containment checks enforce the selected guest/host
Ames UDP port, bound explicitly on IPv4 all interfaces; HTTP remains IPv4 loopback.
Verification-only imported runtimes do not publish Ames. The current managed store
requires schema 13; boot/port choices are recorded independently of storage layout.

### Stopped cross-host handoff (0.22)

Operator capability `host-handoff` adds these identity-pinned endpoints:

- `POST /v1/instances/:id/handoff`: source revision, request ID and destination
  runner/instance. Journals a permanent outgoing fence before exporting stopped data.
- `POST /v1/handoffs`: reserve a fresh incoming instance from the source offer and
  destination HTTP/Ames overrides. Duplicate receipts must match the complete offer.
- `GET /v1/handoffs/:id`: current durable phase/descriptor, including bounded errors.
- `GET|PUT /v1/handoffs/:id/data`: one streamed binary exchange; explicit length,
  `Urbit-Runner-Handoff-SHA256` and verified content. No ordinary JSON size-limit bypass.
- `POST /v1/handoffs/:id/commit`: revision/digest-bound source archival and a signed
  transfer-specific commitment, after the collector observes destination receipt.
- `POST /v1/handoffs/:id/activate`: matching revision/digest and source commitment.
  Releases only the destination fence, leaving it stopped for explicit Start.
- `POST /v1/handoffs/:id/retry`: explicit new attempt for an interrupted receive.

Source prepare/commit and destination receive operations are durable journal jobs;
bulk disk work shares the host work gate. The collector coordinates the phases
without persisting a desktop pier copy or automatically retrying failed mutations.
Observers cannot access any handoff route. The bridge verifies operator delegation
and the handoff capability before forwarding a bounded stream on the same connection.
See [host-handoff.md](host-handoff.md) for limits, failure states and fencing.

### Permanent storage and owned ships (0.20)

`pier-storage` adds optional `pier_path` to new fake/comet registration and exposes
the selected relative location on instances. The current runner defaults new
registrations into `piers/fakeships/<ship-directory>/pier` or
`piers/livenet/comets/<ship-directory>/pier`. Every pier path ends in `/pier` and
reserves its whole parent mount directory, including ancestor/descendant conflicts.
The driver records the parent directory identity and the pier/`.urb` identities.
Old-layout execution and its incremental schema upgrades were removed after the
one-shot local migration. Version 0.22 adds an independent 11→12 upgrade recording
host-handoff fences within this permanent layout.

Snapshots include a bounded `piers` catalogue with runner ID, absolute root,
observation time and candidates (`path`, `kind`, fingerprint, lock/problem status,
optional `enrolled_by` and `moved`). Archive is excluded. Discovery is read-only;
problems do not disable management or Stop. Fingerprints bind directory and `.urb`
device/inode/birth-time identities, not an inferred @p or an LMDB integrity claim.

Operator `POST /v1/piers` accepts `id`, `label`, `ship`, `path`, `fingerprint`,
`http_port`, `request_id` and `confirm_stopped`. It journals `enroll-pier`, validates
the stopped state layout and leaves the instance stopped with `identity_pending`.
Livenet path rank must match the expected @p; duplicate live identities and duplicate
physical/overlapping pier locations are refused. First livenet Start verifies with
Ames disabled, then uses an initialized networked replacement after identity match.

For the same stopped pier moved externally, `relink:true` and `expected_revision`
bind explicit location review; the terminal `relink-pier` operation preserves the
registration/identity. Replaced objects do not inherit the old registration.
`move-pier` on the operation endpoint takes `pier_path`, ordinary revision and
request identity. It requires stopped/uninitialized state, journals uncertainty
before a no-replacement rename of the entire mount root and leaves the ship stopped. Interrupted outcomes
are not automatically resumed or cleared by Stop.

`key-boot` in development-managed mode adds operator `POST /v1/ships`: `id`, `label`,
`ship`, `pier_path`, `config`, `request_id`, `key`, `confirm_network`. Keys are
bounded 0w text and transient in the request. Only the request's key digest enters
the durable fingerprint/private state; contents are staged privately in a mode-0600
file. `register-ship` completes registration without runtime startup. First Start
uses the official runtime's `-w` and read-only `-k` file and pins identity. It creates
the pier in its permanent location and keeps the original runtime running after
readiness; the private host key file is then removed. Explicit later starts use
initialized-resume flags without creation/key flags.

Every ship's exclusive parent is mounted at `/urbit`, overriding the official
image's declared volume. Boot, resume and utilities all use `/urbit/pier`. The parent
contains no sibling ship directories. Private work directories hold metadata and
artifacts, never staging piers; first boot has no promotion or automatic restart.

The `fake-ship-options` capability enables optional `fake_ship` and `pier_name`
fields on fake-registration requests. They are immutable registration metadata,
separate from the instance ID and HTTP configuration. The runner normalizes a
missing leading `~`, validates the standard ship syllables/grouping, and restricts
the directory to one alphanumeric-leading basename with letters/digits/`_`/`-`.
The fields are rejected for comet registration and qualified-source imports.

The optional request `pier_name` chooses a parent-folder basename when no full
`pier_path` is supplied. Snapshots expose the actual path through `pier_path` and
its fixed leaf `pier_name: "pier"`. Only new fake creation uses `-F <selected-identity>`; initialized resume
never repeats creation flags. HTTP identity must match the stored fake name, and
identity mismatch remains quarantined across Stop/observation. Lifecycle, utilities
and epoch paths use the same registered directory inside runner-owned storage.
All managed fakes use the same permanent layout. Disposable qualification fixtures
have separate isolated storage; importing one copies its pier into the permanent target.

## Service startup policy

`serve-development --resume-running` advertises `startup-resume` in Hello. It opts
the management process into resuming eligible initialized ships with persisted
desired state `running`, matched applied/configuration revisions, resolved prior
work and freshly authenticated clean stopped containers. The policy skips existing
running ships, intentional stops, incomplete provisioning and uncertain outcomes.

The server journals internal `startup-resume` operations with actor `system:startup`
and boot/instance-derived request IDs. At most one attempt per pier per host boot is
allowed, including failures. The normal host work gate, resource admission and
identity checks apply. A startup operation replaces the old stopped container with
an initialized-pier runtime; creation flags are never replayed. This kind is not
accepted by the public operation endpoint. It is visible through normal snapshots
and operation history, without new credentials or RPC endpoints.

Read-only engine-availability observations may be repeated. Mutating startup work,
interrupted maintenance and failed startup attempts are not automatically retried.
See [services.md](services.md) for startup/shutdown ordering and deployment.

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
and continues through connection errors. A healthy watcher owns and reuses one
identity-pinned client/transport per runner, so SSH polls do not create a new
PAM/logind session each time. Hello still refreshes roles/capabilities, and the SSH
bridge and runner recheck grants on every request. Failed observations close the
client; reconnects verify identity again. Retry delay doubles up to 30 seconds
(or the requested interval if longer), resetting after a successful observation.
The healthy interval is a delay after each poll, at least 100 ms (default 1 s).
Cancellation closes retained transports. Ctrl+C exits 0; an output failure exits 1. The
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

## Checkpoint 6A: offline pack

Managed runners advertise `offline-pack` and accept operation kind `pack` through
the existing revision-checked, actor-scoped durable operation endpoint. Collectors
check that capability before submission. Pack requires an initialized, explicitly
stopped pier with no retained container or unresolved recovery state. Starts and
packs share host-wide expensive-work admission; incompatible work is serialized
per instance. Configuration changes remain prohibited during the operation.

The operation stays host-side: same registered pier and pinned image, no pill or
identity reapplication, no automatic stop or resume. A successful pack leaves the
instance stopped and preserves applied configuration revision. The instance may
include `maintenance` with kind, timestamps, and logical regular-file bytes before
and after the utility. These are not loom utilization or allocated-space metrics.

An unresolved `maintenance_uncertain` marker in private state prevents resume or
repeat pack after interruption, even if explicit stop removed the container.
Successful idempotency replay returns the same operation without running pack
again. Recovery requires review rather than inferring safety from absent processes.

## Checkpoint 6C: managed offline meld

Managed runners additionally advertise `offline-meld` and accept `kind: "meld"`.
The collector requires that exact capability before submitting the mutation.
The pinned utility passed disposable live qualification in checkpoint 6B; the
[managed live gate](checkpoints/13-managed-meld.md) is a separate checkpoint.

Meld uses the same initialized/stopped-pier, revision, actor-scoped replay, resource
ownership, and interruption-quarantine rules as pack. It shares the host expensive-
work gate with starts and packs. Success leaves the pier stopped; resume is an
explicit operation. Exact replay is a lookup even after the pier is running or
management has restarted.

Pack and meld both recheck physical available RAM and free disk immediately before
utility dispatch, after image preparation and the cgroup/version probe. Their
optional `maintenance.mem_available_bytes` and `disk_available_bytes` record that
admission observation; absent fields on older reports mean no recorded observation.
These are historical samples at `started_at`, not reservations or live telemetry.
`after_pier_bytes` is meaningful only when `finished_at` exists; text output prints
`unknown` for an unfinished result.

Before dispatch, the worker also rechecks all managed instances for retained or
uncertain pack/meld container intents. A timed-out Docker client does not prove a
utility stopped. Jobs queued before such a failure, as well as later jobs after
management restart, fail admission without calling the runtime backend until the
retained resources are resolved. The failed operation's message identifies the
instance requiring explicit stop/review; the request remains replayable. This is
not a state-store failure, so cleanup operations remain available. Removing a
utility releases this host-level restriction but does not clear that pier's
maintenance quarantine.

## Checkpoint 6D: container resource observations

The managed runner advertises `container-resources`. Running instances may include
`resources` with source `docker-stats-v1.47`, a full immutable `container_id`, the
daemon's original `observed_at`, and these exact integer counters:

- `memory_current_bytes`: raw cgroup memory usage, including charged cache/kernel
  memory. Unlike Docker CLI's displayed Linux memory usage, no inactive-cache
  subtraction is applied.
- `memory_limit_bytes`: the observed memory limit; currently validated against
  the fixed 4 GiB development budget.
- `pids_current`: the cgroup task count (includes threads, not just processes).

`status` is `fresh`, `stale`, or `unavailable`. A complete sample ages to stale
after 30 seconds; a timestamp more than five seconds in the future is also stale.
Snapshot reads preserve the original sample time. A failed read replaces previous
numbers with unavailable/absent values. Explicit numeric zero remains zero; absent
or malformed values never become zero. Lifecycle work clears the sample, and
runtime-revision compare-and-swap prevents a late poll from restoring old data.

Collection is part of the existing nominal ten-second runner polling cycle. It
authenticates the owned container and private engine boundary, then performs a
three-second, 256 KiB-bounded, non-streaming, one-shot stats GET over the fixed
Unix socket. No proxy, redirect, credential-file lookup, guest command, or Vere
control socket is involved. Health checks finish before optional metrics so a
metric timeout does not convert healthy HTTP reachability into a ship failure.

These are container resources, not loom capacity, allocator open/free space,
allocated pier storage, or host available RAM. They do not authorize maintenance,
drive a loom-pressure schedule or sizing recommendation. Loom-internal telemetry
and that advice are deferred under the official-release-only runtime policy.

### Container CPU observations (panel 0.15)

`container-cpu` advertises optional `resources.cpu_usage_percent` and
`resources.cpu_limit_cores`. CPU is sampled by the same bounded, authenticated
Docker stats request as memory; it adds no guest execution or runtime control
socket. The current inspected quota is two logical CPUs.

The runner derives the Docker-style percentage from two counter readings:
`100 * delta(container_total_ns) / delta(system_total_ns) * online_cpus`.
100% means one occupied logical CPU, so the configured two-CPU quota corresponds
to 200%. Counter subtraction uses exact uint64 arithmetic before conversion. A
single private per-instance baseline persists the raw counters; snapshots expose
only the derived rate and inspected limit, not raw counters to JavaScript.

The pair must refer to the same container, have 1–30 seconds between daemon sample
times, remain fresh, and have monotonic counters and unchanged CPU count/quota.
First observations, resets, duplicate/out-of-order timestamps and missing CPU
evidence leave the percentage absent. Two valid idle readings publish a real zero.
Missing CPU counters do not invalidate usable memory metrics or runtime health.
Lifecycle/unavailable observations break continuity; late polls retain the existing
runtime-revision guard. Snapshot reads never refresh or recompute a sample.

The desktop records bounded session-local CPU/memory series. Both charts use real
sample times on the horizontal axis, explicit percentage or MiB/GiB values on the
vertical axis, gaps for missing samples and a new series on container replacement.

## Checkpoint 6E: epoch inspection and prefix preview

`epoch-inventory` advertises the `inspect-epochs` operation on the existing durable
operation endpoint. It requires a clean, stopped, initialized development pier,
the selected configuration revision, and a caller request ID. It uses the normal
operator authorization, attribution, replay, and instance-exclusion rules. The
operation reads filesystem metadata and bounded epoch/Vere version headers; it
does not open LMDB, invoke Vere, or start a runtime container.

The resulting instance `epochs` contains source `vere-4.6-epoch-v2`, the registered
opaque pier ID, original `observed_at`, a SHA-256 inventory digest, and numerically
ordered entries. Epoch names remain canonical `0i<decimal uint64>` strings to avoid
JavaScript precision loss. Each entry has logical bytes, allocated file bytes
(`st_blocks * 512`), file count, a shared-hardlink indicator, and a protected flag.
The newest two entries (or all entries when fewer exist) are protected.

The reader is rooted at the registered instance and refuses symlinks, special
files, nested epoch directories, cross-device entries, unsupported version headers,
unexpected log-root entries, and over-limit inventories. It accepts at most 128
epochs and 32 regular files per epoch. Two complete metadata scans must produce
the same digest. This fingerprint includes inode/device identity, sizes, blocks,
link counts, mtime/ctime, and version headers; it is not a content or recovery hash.

Lifecycle requests and actual configuration changes invalidate the cached inventory.
Polling preserves the original inspection time. Exact request replay returns the
old operation, not a new disk inspection; refresh uses a new request ID.

### Journal-backed history dates (panel 0.14)

Instances additionally expose optional `epoch_history:[{name,rolled_at}]` and
`last_roll` (a maintenance report). These are bounded display metadata from the
latest 128 successful roll operations for that specific registered pier, including
operations older than the activity feed's 128-event window. The report must match
the private pier ID, contain a valid nonzero epoch name and have completion times
inside that roll's operation. Failed rolls and inherited reports from other actions
are not creation evidence.

`rolled_at` means recorded roll completion, not filesystem birth time. Missing
evidence is displayed as “Creation time not recorded.” These fields are derived on
snapshot reads; they do not change the store schema, private runtime, live epoch
inventory, its digest, or deletion admission. A newer roll's resulting epoch list
takes precedence over an older pruning result in the historical desktop timeline.
Only a fresh stopped inventory authorizes retention review.

The desktop binds inspection feedback to runner, instance, configuration revision,
request ID and the completed journal operation. Once its fresh inventory arrives,
the history review opens with no cutoff selected. It never interprets the queued
acknowledgement alone as completion or retries a mutation automatically.

`epoch-preview INSTANCE --revision N --expected-digest HASH --cutoff 0iEVENT`
fetches a snapshot and calculates a non-executable preview from its cached inventory.
The instance must still be cleanly stopped and idle at the selected revision, the
digest must match, and the sample must be no more than five minutes old (with five
seconds of future-clock tolerance). Selecting a cutoff selects that epoch and
every older entry. A cutoff touching the protected suffix is rejected.

Preview output binds runner, instance, revision, digest, cutoff, and the exact
selected/retained sets. `deletion_supported` is false and
`requires_stopped_revalidation` is true. Allocated file bytes may double-count
hardlinks or shared extents and are not guaranteed reclaimable space. LMDB integrity,
additional recovery dependencies, multi-epoch replay, and actual selected-prefix
deletion remain separately qualified; no prune/chop operation or `--apply` flag
is exposed by this checkpoint.

The optional epoch-entry `snapshot_bytes` records a non-empty `image.bin` logical
file size when present. It is metadata evidence, not validation of snapshot
contents or recovery. The 6F standalone `qualify-epochs` profile requires a new
snapshot-bearing epoch after each roll and validates the final prefix preview;
it adds no managed roll/prune operation or destructive confirmation token.

Epoch entries may additionally carry a `digest`: a per-epoch metadata fingerprint
covering its directory identity and file manifest. It excludes the parent log
directory, allowing unchanged retained epochs to be compared after prefix removal.
It is not a content hash or a backup-integrity guarantee. Older cached entries
without this optional field remain valid for non-executable previews.

The standalone 6G `qualify-prune` command is confined to its own fresh disposable
run and does not add a management RPC operation. It separately journals and
revalidates the fixed genesis-epoch deletion manifest, records every confirmed
unlink after directory sync, verifies the exact retained set, and quarantines
partial/ambiguous outcomes. Its recovery test removes the disposable current
checkpoint and invokes pinned `play --full`, which restores from the latest
retained epoch. Public preview `deletion_supported` remains false.

## Checkpoint 6H: explicit managed offline roll

`offline-roll` adds operation kind `roll` to the existing revision-checked,
actor-scoped endpoint. The collector requires the capability before submitting.
Roll requires a cleanly stopped, initialized registered fake pier with all prior
containers removed, no active operation, and no recovery requirement. It shares
the expensive-work gate and retained-utility admission checks with start/pack/meld.
Configuration changes and conflicting work remain excluded while it runs.

The runner invokes the qualified pinned `roll --loom 31` utility once. After clean
exit/removal, a fresh inventory must contain every prior epoch name plus exactly
one newer epoch with a non-empty snapshot. Success preserves configuration/applied
revision and leaves the pier stopped. Its `maintenance` report adds optional
`epochs_before`, `epochs_after`, and `created_epoch`; the cached epoch inventory is
refreshed for immediate preview. `finished_at` still identifies a completed result.

Failure, interruption, persistence failure, or an unverified/no-growth outcome
leaves `maintenance_uncertain` set. Stop may remove resources but cannot clear
that pier quarantine. A retained roll container also blocks later expensive host
work until resources are resolved. Exact request replay never re-executes the roll.

Vere may exit zero when a log is empty. After a successful roll, the durable
`roll_requires_run` flag is set and exposed on the instance. A different roll
request is refused until a successful runtime start (or verified healthy running
observation) clears the flag, followed by explicit clean stop. Stop, pack, meld,
configuration changes, and management restart do not reset it. Replay is checked
first and remains available. This avoids dispatching obvious consecutive empty
rolls; unexpected no-growth after dispatch still requires outcome review.

Roll remains operator-requested; this extension adds no automatic roll, chop,
epoch deletion, or maintenance schedule.

## Checkpoint 6I: confirmed managed genesis pruning

`epoch-prune-genesis` advertises the exact deletion scope qualified in 6G: genesis
epoch `0i0` in a four-epoch development pier, retaining three newer snapshot-bearing
epochs. Other cutoffs/counts remain non-executable even when the general preview
can describe them. Shared files, missing fingerprints, and unsupported layouts
are refused.

The existing operation endpoint accepts `kind: "prune-epochs"` with its usual
`request_id` and `expected_revision`, plus:

- `epoch_cutoff`: currently exactly `0i0`.
- `inventory_digest`: the selected inventory's SHA-256.
- `confirm_history_loss`: required true.

Pruning fields are invalid on other operation kinds. The collector exposes the
dedicated `epoch-prune` command with `--cutoff`, `--expected-digest`, `--revision`,
`--request-id`, and `--confirm-history-loss`; it checks the exact capability before
submission. It does not fetch a new digest/revision to silently repair a stale
request. Operator authorization is enforced by the existing endpoint; observers
cannot submit deletion.

Admission requires a stopped initialized pier, no conflicting work/recovery, a
recent matching cached inventory, and the qualified scope. The complete request
participates in actor-scoped replay/fingerprint checks. Accepted intent is durable,
the cached inventory is invalidated, and the host work gate serializes dispatch.
The driver then performs fresh filesystem revalidation against the original digest,
persists uncertainty and an exact manifest before the first unlink, and preserves
the existing per-file sync/progress and retained-fingerprint checks.

The instance's optional `pruning` report shows cutoff, inventory digest, timestamps,
file progress, epoch removal, observed retained names, and uncertainty. Success
leaves the pier stopped and refreshes its three-epoch inventory; resume is explicit.
Partial deletion or failed persistence cannot be cleared by stop or observation.
Exact replay remains a lookup after resume, inventory invalidation, or management
restart and never repeats the unlink operation.

On a capable runner, `epoch-preview` sets `deletion_supported: true` only for this
qualified scope. Explicit confirmation and fresh stopped-pier revalidation remain
required. Older capabilities and broader previews continue to report false.

The standalone 6J `qualify-prefix` profile tests a broader fixed recipe on a new
disposable pier: five epochs to three, then three to two. Its whole-prefix
manifests include snapshot-bearing epochs and per-target unlink progress. Each
round records a separate latest-epoch recovery after removing the current
checkpoint. Report reads are capped at 128 KiB, with at most two prefix records
and sixteen containers; uncertainty remains explicit and is not cleared by cleanup.
This profile changes no managed pruning capability or caller-selectable deletion
scope until its live qualification is accepted.

## Checkpoint 6K: confirmed managed prefix pruning

`epoch-prune-prefix` extends the same `prune-epochs` request to any canonical
cutoff in the bounded pinned-layout inventory, provided the cutoff and **every
older epoch** are selected and at least the newest two remain. Inventories are
still limited to 128 epochs. Each selected nonzero epoch must contain its snapshot
and the pinned five-file layout; genesis uses the four-file layout. Complete
fingerprints and exclusive file links are required throughout the inventory.

The explicit confirmation, digest/revision binding, operator authorization,
fresh on-disk revalidation, host work gate, per-instance exclusion, and durable
unlink/progress rules remain. Each managed job installs a new prefix record only
after selection and manifest validation. Successful completion leaves the pier
stopped and refreshes inventory. Previous completed records remain in the operation
journal; replay after a later pruning job is still a lookup, not a second deletion.
The new private prefix marker and missing completion evidence independently enforce
quarantine, alongside the generic maintenance flag and legacy genesis marker.

The public `pruning` report adds `selected_epochs` and `removed_epochs`, aggregate
file progress, and retained names. For compatibility, `epoch_removed` now means
all selected epoch directories were removed. Older genesis reports are preserved
until a new pruning record supersedes the displayed report.

The runner retains `epoch-prune-genesis` for older clients. A current collector
uses `epoch-prune-prefix` for broader cutoffs and refuses non-genesis requests
before submission when only the older capability is present. Its preview sets
`deletion_supported` only when the selected scope matches the advertised capability
and validated layout. No automatic stop, resize, roll, chop, or pruning schedule
is introduced by this extension.

## Checkpoint 6N: connection revisions and login-code access

Collector enrollments expose a positive `revision`. Missing legacy values read as
1 without a write. `runner-edit` supplies the original name, runner ID and revision
plus the full proposed name/transport configuration. The new endpoint must verify
the same ID and mode. A locked compare-and-swap refuses concurrent changes,
removed/re-enrolled entries and collisions. Successful changes advance the revision;
exact no-ops preserve it. This changes local enrollment, not runner configuration.

Runner capability `login-code` supports operator-only
`POST /v1/instances/{id}/login-code` with:

```json
{"expected_revision":2,"container_id":"<full 64-character lowercase container ID>"}
```

Admission requires a fresh running observation, matching configuration and container,
no active operation and no recovery requirement. The runtime read is bounded and
serialized against lifecycle/configuration admission. The response contains
`runner_id`, `instance_id`, `config_revision`, `container_id` and ephemeral `code`.
Clients validate every selection field. Responses are `Cache-Control: no-store`;
errors are redacted. Neither snapshots nor durable state contain the code.

An attributed terminal `login-code` operation records success/failure only, with a
server-generated request ID. This access audit is not a replayable secret cache.
Collector stdout disclosure requires explicit `--reveal`; normal desktop copying
uses the Omabit helper and a short-lived sensitive clipboard provider instead.

## Checkpoint 6P: qualified-fake import

`import-qualified-fake` provides operator-only `GET /v1/imports` and
`POST /v1/imports`. The catalogue is `{runner_id, sources:[...]}`; each source has
`run_id`, `content_digest`, `files`, `bytes`, `ship`, `qualified_at`, and optional
`claimed_by`. It refers exclusively to a completed pinned disposable qualification's
stopped original. A catalogue is metadata evidence; the worker rehashes source
content before transfer.

POST takes `id`, `label`, `config`, `request_id`, `source_run_id`, `expected_digest`,
and `confirm_stopped:true`. Instance, port, source reservation and queued operation
are one transaction. Source reservation survives failure and is unique per source
run. An exact actor/request replay is resolved before current catalogue lookup.
The `import-fake` job copies into fresh owned storage under the host work gate and
returns a stopped initialized instance. There is no implicit runtime start.

Snapshot `import` contains `source_run_id`, `content_digest`, `copied_files`,
`copied_bytes`, `expected_files`, `expected_bytes`, optional `started_at` /
`finished_at`, and `uncertain`. Interrupted operation reconciliation sets the
private import quarantine even if a final copy checkpoint had been persisted.
Observation, Stop and configuration edits do not clear this marker. Start refuses
an incomplete import rather than treating it as an uninitialized fresh fake.

The first import also advances private SQLite schema metadata to version 2 so
older runners fail closed instead of ignoring the new private state. Wire protocol
version remains 1, negotiated by the new capability.

## Checkpoint 6R: explicit mixed development mode and comets

`serve-development` exposes mode `development-managed`, with `create-comet` in
addition to the existing fake/import capabilities. `serve-fake` keeps the legacy
mode and refuses startup if comet registrations exist. Collector
`runner-accept-mode NAME --runner-id ID --revision N --mode development-managed`
performs a same-endpoint, same-ID, revision-checked transition; it cannot approve
a different runner, arbitrary mode or concurrent enrollment replacement.

Operator `POST /v1/comets` takes the registration fields (`id`, `label`, `config`,
`request_id`) plus `confirm_network:true`. It returns a durable terminal
`register-comet` operation. The instance has `mode:comet`, initially empty `ship`,
configuration/applied 1/0, and remains uninitialized until explicit Start. Replay
is actor-scoped and never generates an additional identity.

First Start persists a private uncertain-bootstrap marker before dispatching the
creation container. Completion pins the public comet identity. Resume uses no
creation/pill/key flags; observation and resume require the same identity. An
interrupted initial creation remains quarantined even if a private completion
checkpoint preceded a lost operation outcome. Stop can clean up resources but
cannot authorize another identity generation. Without `comet-compaction`, comet
maintenance/epoch actions are `unsupported_for_instance` (see 6U below).

The first comet registration atomically activates private SQLite version 3, and
subsequent management requires 6R+ in mixed development mode. This metadata change
prevents older binaries from ignoring the new lifecycle/identity fields.

## Checkpoint 6S: maintenance policies and atomic slot admission

`maintenance-schedules` enables operator GET/PUT
`/v1/instances/{id}/routine`. GET returns `{runner_id,instance_id,policy}` (null
policy means absent). PUT requires `expected_revision` (0 for absence),
`expected_instance_revision`, and `config` containing `enabled`, `action`,
`weekdays`, `hour_utc`, `minute_utc`, `window_minutes`, `stop_running`, `resume`.
Actions are pack/meld; weekdays are unique 0..6, UTC hour/minute bounded normally,
and dispatch windows are 5..180 minutes. Resume requires downtime consent.

The saved policy has a stable ID, increasing revision, bound configuration revision,
authenticated owner, update time and optional last-slot/operation metadata. Slots
before the last update are ineligible. The most recent UTC slot is handled once;
missed windows and unsafe state produce terminal `skipped` operations rather than
catch-up work. Slot advancement and queued operation/instance exclusion commit
together. Owner access is checked at claim and dispatch. Revocation does not cause
another operator's credentials to be substituted.

One `routine-pack`/`routine-meld` operation owns the entire optional-stop,
maintenance, optional-resume workflow and host gate. The action is internal;
ordinary `operate` cannot inject it. Instance `routine` publishes stage progress,
policy identity/revision, slot/deadline and outcome. `schedule` publishes the policy
and computed next UTC slot. Failed/interrupted workflows are never automatically
continued. The first policy activates private schema version 4.

## Checkpoint 6U: managed comet compaction

`comet-compaction` extends the existing operator `pack` / `meld` requests and,
together with `maintenance-schedules`, UTC policies to initialized comets. Ordinary
revision, actor-scoped replay, stopped-pier admission and explicit downtime/resume
rules apply. Private completed provisioning evidence and a pinned public identity
are required; runtime observation alone cannot establish this provenance.

Utilities run without networking using `/urbit/comet` and managed-comet ownership
labels. Manual success leaves the pier stopped. Routines only resume after success
and only if the ship was running beforehand. Every resume checks the exact pinned
identity; a mismatch sets durable provisioning quarantine. Stop and observation do
not erase that quarantine or an uncertain maintenance outcome. Unresolved comet
utility resources block other expensive host work just like fake-pier utilities.

At 6U, comet roll, epoch inspection, pruning and recovery remain unsupported. Protocol
version stays 1. The first accepted comet utility or saved comet policy atomically
advances private SQLite schema to 5; updates to older features preserve that floor.
Such a store requires 6U+ in `serve-development` mode.

## Checkpoint 6W: managed comet epochs

`comet-epochs` extends the existing operator `inspect-epochs` and `roll` operations
to initialized, identity-pinned comets. The same stopped-pier admission, revision
checks, actor-scoped replay, resource gate, bounded scanner and persistent
`roll_requires_run` guard apply. Roll requires exactly one newer snapshot-bearing
epoch; utility success alone is insufficient. Start remains explicit and checks the
pinned identity. Interrupted/uncertain rolls cannot be adopted by observation or Stop.

The collector requires this additional capability for comet `epoch-preview` and,
without the 6Y capability below, returns `deletion_supported:false`. Runner-wide fake pruning
capabilities do not authorize comet deletion. Panel review state includes instance
mode, and independently disables comet deletion even for a malformed true flag.
At 6W, runner admission and runtime dispatch reject comet `prune-epochs`.

The first accepted comet inspection or roll atomically advances private SQLite
schema to 6; subsequent policy/configuration work cannot lower it. Such a store
requires 6W+ `serve-development`. Wire version stays 1; routine actions remain
pack/meld only. Comet pruning and recovery need separate qualification.

## Checkpoint 6Y: confirmed comet prefix pruning

`comet-prune-prefix` qualifies `prune-epochs` on initialized identity-pinned comets.
The existing request requires `expected_revision`, `request_id`, `epoch_cutoff`,
`inventory_digest`, and `confirm_history_loss:true`. The selected cutoff includes
all older epochs and must retain at least the protected newest two. Admission and
worker revalidation enforce stopped state, exact evidence, pinned layout and
exclusive files; the worker holds the host work gate and records each unlink.

The collector only marks a comet preview deletable with the new capability and a
qualified `PrunePlan`. Fake pruning capabilities cannot substitute for it, nor can
the comet capability enable a fake preview. Panel reviews retain mode, role,
runner/instance/revision and exact inventory binding plus explicit history-loss
confirmation. A new capability does not automatically upgrade an older preview.

Success leaves the pier stopped, with fresh retained inventory and a completed
pruning report. Explicit resume verifies the same identity. Partial deletion has
its own durable quarantine marker; Stop/observation cannot erase it. Exact historical
actor/request replay precedes state admission and never repeats filesystem work.

The first accepted comet prune atomically advances private SQLite schema to 7.
Older-feature edits cannot lower it; use 6Y+ `serve-development` afterward. Wire
version remains 1. Checkpoint-loss recovery remains confined to the disposable
qualifier, and routine policies remain pack/meld only.
