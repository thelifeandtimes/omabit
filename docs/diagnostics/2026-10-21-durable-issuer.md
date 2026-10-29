# Durable application-session issuer checkpoint

2026-10-21, unreleased source on `codex/platform-overhaul-plan`, based on
`b233128`. The source hashes and validation commands are in
[the evidence manifest](2026-10-21-durable-issuer.json).

This checkpoint implements the internal journal and issuer engine around the
[private handoff](../application-handoff-v1.md). It does not expose a command,
install a receiver, or enable an approval UI. The earlier
[real Theme/DMs helper run](2026-10-21-plugin-handoff.md) remains separate evidence;
this new engine has been qualified with synthetic HTTP and process-crash fixtures.

## Implemented behavior

`collection.ApproveApplication` records approval for an exact ship connection and
revision, plugin ID, profile, immutable artifact digest and ordinary owner-session
authority. Its trusted caller must already have obtained that approval. It is not
itself a prompt or permission check against arbitrary same-account code.

The new private version-1 `applications.json` is independent of `ships.json`,
`runners.json` and the runner database. It contains opaque credential references,
not cookies or codes. File and directory sync precede each successful transition.
Readers reject unknown formats/fields, duplicate JSON keys and invalid lifecycle
records; unsafe permissions, links and special files are refused without reset.
At most 64 attempts are retained. Automatic pruning is deliberately not implemented.

| Phase | Meaning / restart action |
| --- | --- |
| `approved` | No issuance attempted; recovery can close as `not-issued` |
| `issuing` | Intent and credential reference are durable before login; recovery never retries login |
| `saved` | Exact credential saved and read back before delivery; interrupted work is revoked |
| `delivering` | Receiver may have stored the account; a missing durable acknowledgement triggers cleanup |
| `connected` | Exact accepted acknowledgement was durably recorded; ordinary recovery preserves it |
| `revoking` | Cleanup owns this exact session; network/keyring failure retains the reference |
| `revoked` | Remote revocation or rejection is recorded before local credential deletion |
| `closed` | No issuance occurred, or confirmed remote outcome plus local deletion completed |
| `unresolved` | A session may exist but its token/outcome is unavailable; no automatic new approval/delivery |

Separate attempt locks cover issuance, credential IO, delivery and cleanup. They
are released by process exit. Short registry locks serialize metadata updates;
HTTP does not lock the entire collector registry. Concurrent recovery cannot race
an in-flight handoff. Duplicate request IDs cannot be rebound or mint another token.

The engine rechecks the enrolled connection revision before issuance and verifies
the receiver binding before issuance and again before delivery. Its receiver
interface requires a verified immutable artifact and bounded private transport.
**The production artifact resolver/launcher is not implemented in this checkpoint.**
Tests supply trusted fixture receivers; a claimed hash alone is not execution
verification. The approval UI stays disabled until that adapter is implemented.

The collector retains a private copy of an accepted session for an explicit later
disconnect. The plugin owns its own copy and all data traffic. Neither direct
plugin use nor standalone login depends on this journal or a running collector.
Credential keys are domain-separated from parent ship-session slots. The adapter
is inherited from the explicitly enrolled ship: Secret Service normally, or its
explicit headless plaintext-file choice, with no fallback.

Cleanup checks public ship identity and logs out only the exact stored session,
including when the local expiry has elapsed. Ordinary reads still enforce expiry.
HTTP 401/403 is recorded as `rejected`, separately from `revoked`; redirect,
identity, network and server failures remain pending. Parent enrollment removal
cannot erase the target/adapter/reference needed for application cleanup.

## Interruption limits

A login can succeed remotely before its response arrives or its token is saved.
A process dying in that interval leaves an unresolved issuance record. Recovery
does not invent a revocation, log out all sessions, or issue another token. Missing
credentials after delivery likewise do not establish remote cleanup. Known
pre-login failures and a missing code thread are instead closed as `not-issued`.

Once delivery is attempted, `receiver_reconciliation` remains set even after
remote revocation. The receiver might have committed its account before its
acknowledgement was lost. A subsequent approval is fenced until that exact local
account is reconciled. **The receiver reconciliation operation is the next
checkpoint; no general “forget pending” bypass is supplied.**

## Evidence

The following scoped race tests and vet passed, with `OMABIT_TEST_FAKE_ZOD=0`:

```sh
go test -race ./internal/applicationauth ./internal/collection ./internal/eyre ./internal/shipaccess ./backend/cmd/urbit-collector
go vet ./internal/applicationauth ./internal/collection ./internal/eyre ./internal/shipaccess ./backend/cmd/urbit-collector
```

New fixtures demonstrate:

- Actual helper-process exits before/after issuance, after credential save, during
  delivery, after remote logout, after local deletion and after connection commit.
- Parent/other-plugin sessions surviving an exact disconnect, with no duplicate
  issuance/delivery on repeated approval or recovery.
- Lost, mismatched, duplicated, oversized, extra-field, trailing-data and failed-exit
  acknowledgements triggering cleanup without exposing their private payloads.
- Failed durable acknowledgement commit retaining recoverable delivery state;
  unavailable cleanup, missing credentials and forgotten parent enrollment.
- Changed artifact selection, stale ship revision, concurrent recovery, failed
  storage and cancellation; no receiver gets a credential before verified storage.
- Local expiry enforced for normal traffic but exact expired-session cleanup
  reaching the verified origin; no redirect following or cross-session logout.

The first broader run was blocked by the command sandbox's prohibition on fixture
listeners. The same tests passed with scoped execution permission. This was not a
live-ship test or a code failure. One early fixture needed a private child config
directory instead of using Go's temporary-directory root directly; it was fixed.

No installed plugin, production credential store, runner/collector service,
Docker resource, tmux ship, `~/temp-ur` data or managed `~lismyl-lismyl` was accessed
or changed. No real runtime was started for this checkpoint.

Next: implement the immutable receiver launcher and attempt-specific receiver
reconciliation, then run this durable engine with both real Python receivers in
the existing private lab. Approval UI and full profile switching follow that gate.
