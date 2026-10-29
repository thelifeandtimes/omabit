# Observe and operate

Examples use the selected collector executable and explicit registry/runner values.
Replace `REGISTRY`, `NAME`, `RUNNER_ID`, `INSTANCE`, `REVISION` and `REQUEST_ID` with
observed task-specific values. Keep global flags before the command.

```sh
urbit-collector --config-dir REGISTRY --runner NAME --expected-runner-id RUNNER_ID hello
urbit-collector --config-dir REGISTRY --runner NAME --expected-runner-id RUNNER_ID snapshot
urbit-collector --config-dir REGISTRY --runner NAME --expected-runner-id RUNNER_ID host-check
```

`hello` establishes the runner identity, mode, access role and advertised features.
`snapshot` reports desired/observed state, configuration revision, active operation,
recovery state and operation receipts. `host-check` through the collector reads
compatibility evidence on the selected runner; it needs no local Docker access.
A successful report command does not by itself mean that hosting is qualified.

`inventory` can exit 1 with a partial inventory on stdout and no stderr error.
Inspect each connection's status, freshness and optional snapshot. Do not discard
that structured result or assume every nonzero exit has the same error envelope.

Treat unreachable and unknown states as unresolved. A desired state of `stopped`
is not proof that the runtime has stopped. Likewise, `identity_pending` after an
import means that its first explicit Start has not yet verified the running identity.

## Start or stop

Use the current `config_revision` of the selected instance. Choose a new request ID
for the intended action and retain it with the runner/instance binding before dispatch.

```sh
urbit-collector --config-dir REGISTRY --runner NAME --expected-runner-id RUNNER_ID \
  operate INSTANCE start --revision REVISION --request-id REQUEST_ID
```

Use `stop` in place of `start` when that is the authorized action. An accepted
operation is asynchronous: retain its `id`, `request_id`, `instance_id`, `kind`
and revision, then poll `snapshot` until that exact operation is terminal. The
operation's terminal result and the instance's resulting state are both relevant.
For a successful start, check running/healthy and resolved identity/recovery state;
for a stop, check stopped state and no active operation. Do not equate exit 0 from
the submission command with successful completion.

## Handle disagreement or interruption

- **Revision conflict:** refresh the snapshot and compare the current state with
  the intended change. If the task still authorizes that exact effect, use the
  reviewed current revision; do not blindly increment or repeatedly retry.
- **Runner identity mismatch:** stop mutations through that selection. Resolve
  which runner the user intended; do not overwrite enrollment to make it match.
- **Unsupported capability / observer role:** explain the missing feature or
  authority. Do not fall back to an unrelated engine, unrestricted SSH command or
  another operator's credentials.
- **Lost acknowledgement:** preserve the original request ID and inspect the
  runner's operation history. Match the actor/request and intended effect. A
  missing receipt during an outage does not prove that nothing was accepted.
  Do not issue a new ID or an inverse operation to guess the outcome.
- **Failed/interrupted operation:** inspect its message, runtime status and any
  recovery evidence. Resolve the particular condition before issuing a new action.
  A deliberate replay of an already terminal request should return its existing
  receipt; it is not a retry mechanism for an uncertain operation.

For runtime diagnostics, `logs INSTANCE --revision REVISION` returns a bounded log
tail for operator access. Review locally and redact credentials before sharing it.
Do not erase operation history, replace the database, delete lock files or change
stored revisions to clear an error.

New collector builds may include a `guidance` object beside the unchanged `error`
object. Its `kind` distinguishes `review_current_revision`,
`reconcile_uncertain_request` and `resolve_runner_identity`; `next` lists read/help
commands, with their effects. Keep the same registry and pinned runner selection
when applying read guidance. The commands are navigation, not an automatic repair
or authorization. Identity mismatch offers local enrollment/help inspection only.
Older builds omit guidance, so retain the error/receipt rules above. Do not infer
mutation rejection from a generic `client_error` or from the absence of guidance.

For direct Eyre command failures, newer collectors also distinguish
`resolve_ship_origin`, `restore_credential_access` and `review_ship_login`. Use the
same registry **without** runner/socket selection: `ships` reads local metadata,
`help connections` and `ship-login --help` describe the next explicit action. A
rejected session is not evidence of a stopped ship or missing backend. Unavailable
credential storage must not trigger a new login, adapter change or plaintext
fallback. Resolve an origin/identity mismatch before changing enrollment.
These hints apply to error envelopes; `ship-status` continues returning its own
structured status row, including on failure.

## When an authorized sign-in needs a code

Ordinary fleet observation does not require a code. For an explicitly authorized
sign-in task, read `login-code --help` and select `config_revision` and
`resources.container_id` from a fresh running snapshot. The completed Start receipt
can appear before a resource observation names the container. If that field is
absent, wait and read the snapshot again; do not substitute a previous container
ID, infer one from the instance name, or bypass selection through Docker.

The runner additionally requires a recent observation, no active operation and no
recovery fence. A refusal is not permission to collect a code another way. Keep
the returned secret in the sign-in flow and out of normal logs/reports. Preserve
existing sessions when sign-in or observation is unavailable.
