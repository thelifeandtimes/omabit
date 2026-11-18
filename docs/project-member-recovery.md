# Project member recovery

Project lifecycle receipts and current process observations are separate evidence.
A queued acknowledgement does not establish shutdown, and an unavailable HTTP
endpoint does not establish a clean exit. Retain the original request and exact
inputs after a transport failure.

## Stop observation and receipt lookup

The development source binds a running member's process start timestamp before
bracketed ownership inspection. A running-to-exited transition may trigger up to
three complete observations with that same binding. It never rebinds a restarted
process or retries a generic ownership, image, namespace or engine mismatch.
Nonzero/OOM exits are observable failures, not clean-stop authority.

Restricted SSH now forwards the replay-only header on guarded project
registration, member action and anchor-recovery endpoints, as well as ship
creation. The bridge rechecks the selected runner's replay capability on the
actual forwarded connection. Unsupported endpoints still refuse the header.
Expanded-member receipt lookup precedes runtime-backend and mutation-admission
checks. It returns only an existing actor/request/fingerprint match and cannot
schedule another signal or worker.

The regression fixture deterministically reproduces the old failure shape at
both preliminary/bracketed read boundaries and during a namespace probe. This
supports the race explanation in issue #16; it does not prove which predicate
failed in that historical incident. Isolated inert fixture and race tests pass
for these paths. No real-runtime, installed upgrade or production qualification
is implied by those results.

## Interrupted expanded-member Stop (schema60 development)

The new commands require matching development tools and a runner advertising
`project-member-stop-recovery-v1`. A concrete runtime observer separately
advertises `project-member-stop-observation-v1`. These are development source
contracts, not commands present in the published0.4.0 bundle.

Start with saved diagnosis of one selected operation. This reads history and
the exact pending fence without inspecting the runtime:

```sh
urbit-collector --runner RUNNER --expected-runner-id RUNNER_ID \
  project-diagnose PROJECT --member MEMBER --operation ORIGINAL_STOP
```

Add `--probe` for a bounded, read-only ownership/process observation. When a
clean exit can be established, the JSON result includes a five-minute review:

```sh
urbit-collector --runner RUNNER --expected-runner-id RUNNER_ID \
  project-diagnose PROJECT --member MEMBER --operation ORIGINAL_STOP \
  --probe > private-stop-review.json
```

Inspect the runner/project/member generation, original operation and request
digests, complete container ID, initialized pier, engine/anchor/namespace and
process start/finish evidence. This exact review is operator-only data. It is
not a sanitized bundle suitable for public sharing.

For a bounded shareable **saved-state** diagnostic view, use the shared CLI/API
operation (also available as MCP tool `omabit.project.stop-diagnostic-bundle.v1`):

```sh
omabit --config-dir REGISTRY call project.stop-diagnostic-bundle.v1 \
  --input-json '{"name":"RUNNER","runner_id":"RUNNER_ID","project_id":"PROJECT","instance_id":"MEMBER","operation_id":"ORIGINAL_STOP"}'
```

This reads the original saved diagnosis with operator access. It sends no live
probe, signal or recovery request. The versioned view retains only enumerated
phase/status/fence/reason categories and timestamps, using fixed messages for
known and unknown reasons. Exact identities, names, request/actor text, endpoints,
paths, free-form messages, raw database bodies and sealed reviews are omitted.
It explicitly states that private bindings are omitted and it is not an action
review. A recorded linked resolution is distinct from current runtime health.
Publishing or sharing this output remains the owner's separate explicit choice.
Actual recovery still requires the complete private fresh review below.

Apply only that review, using the same operator that submitted the original Stop:

```sh
urbit-collector --runner RUNNER --expected-runner-id RUNNER_ID \
  project-reconcile-stop PROJECT --review private-stop-review.json \
  --request-id RECONCILIATION_REQUEST --confirm-action
```

The runner reobserves immediately before a compare-and-swap transaction. It
requires unchanged ledger, original operation/request, exact runtime bindings,
clean exit and no other pending project operation. The transaction marks only
the selected member phase `stopped` and creates a new linked reconciliation
receipt. The original interrupted receipt and diagnostic remain unchanged.
Only that verified receipt can resolve the corresponding admission fence.

After a lost reply, preserve the same file and request ID and add `--replay-only`.
This lookup returns the original result even after its review expires, without
another observation or mutation. Changed, missing or other-actor requests cannot
become fresh effects through lookup. A stale review for a new application requires
explicit fresh diagnosis and review; it is never silently refreshed.

Saved diagnosis then reports `resolved_by`. Ordinary `project-operate ...
member-start` remains a separate reviewed action with the current full container
ID and **project plan** revision/digest from `project-status`. The stop-review
digest is not the project plan digest. Reconciliation sends no SIGTERM, removes
no container, starts no ship and changes no pier or port mapping.

### Evidence and refusal cases

New stops save an exact process start, container and operation binding before
signal dispatch. Historical stops retain their omitted fields. The legacy path
requires the original typed Stop request to name the current exact container,
the retained initialized-pier identity, a process started before original
admission and an exit after admission. It labels this narrower evidence basis
`legacy-admission-window`; migration never invents a historical pre-signal read.

Still-running, OOM/nonzero exit, changed process/container/pier/namespace,
missing evidence, another operator, another pending operation, stale review and
unsupported phases refuse with distinct structured reason codes. An observation
failure establishes no recovery authority. Other interrupted phases require their
own supported recovery path; this command cannot repair them automatically.

Never repair a runner database, repeat SIGTERM with a new request, reset a pier,
or reinterpret an interrupted receipt as a successful shutdown.

### Qualification

The checked-in original59 fixture comes from pinned0.4.0 source and the declared
synthetic writer helper under
`internal/managed/testdata/project-stop-recovery/`. Its hashes bind the SQL and
helper. Source tests cover unchanged historical rows across repeated migration,
rollback, exact receipt replay, original preservation and corrupt/missing ancestry.

To reproduce actual original-writer migration and older-reader refusal:

```sh
python3 -B scripts/test-project-stop-recovery-upgrade.py --output NEW_PRIVATE_OUTPUT
```

The harness checks the experiment storage budget. Its optional `--budget-gib 48`
requires separately authorized temporary capacity; the default remains40GiB.
An opt-in `TestProjectStopRecoveryActualCollectorLostReply` exercises the actual
collector, API routing and SQLite using an inert runtime observer. These gates
are synthetic migration/interface evidence. Installed artifact upgrades and
disposable real-runtime stop/recovery/start remain separate qualification gates.
