# Backend increment 6N — September 28 UTC

Connection editing and login-code access are implemented. The installed desktop
snapshot and its bundled collector/helper are **0.5.0**; shell status IPC confirmed
the loaded version after restart. The pre-upgrade desktop archive is
`/tmp/opencode/omabit-panel-before-6n-20260928.tar`.

## Verification completed

- `go test ./...` and `go vet ./...`.
- Race checks for collection, managed runner, runtime qualification, client,
  collector CLI and desktop helper.
- Node panel-model checks, isolated QML interaction checks, and the mapped-window
  fixture including the themed activity filter.
- Snapshot installer checks and `git diff --check`.
- Rebuilt `build/urbit-runner`, `build/urbit-collector`, and `build/omabit`.
- Linux/ARM64 runner cross-build (build-only, not runtime qualification).

A live, identity-pinned SSH **no-op** `runner-edit` checked workstation at its
existing alias/config. It retained runner
`runner-f496feed1f7b08f31a5d82b94ac2f5fd`, connection revision 1 and enrollment time
`2026-09-24T18:59:52Z`. Actual rename/rebind, collision and concurrent-update
behavior were exercised with isolated Unix-socket fixtures.

Login-code runtime/RPC/client checks used fixture output only. The clipboard test
used a fake `wl-copy` executable to verify exact non-secret arguments, sensitive
mode and secret stdin handoff; it did not replace the real desktop clipboard.
QML received only the helper's status receipt.

## Live runner and Eyre validation

The user started 6N. Hello retained the pinned runner ID and operator actor and
advertised `login-code`. Alpha and panel-check were running/healthy, with no
incidents or recovery requirements. Alpha remained at configuration/applied 2/2
and container `78116edfda0f9f10ae5ee0d385699b3656dc87639fbac7fa0176e095d38850a7`.

Initial retrieval at `19:32:31.307859441Z` produced audit
`op-21360ad0c0a9a00b1705499a0656f18c`. Selection fields and code shape matched;
the snapshot contained no code. `scripts/validate-code-access.py` then validated
the complete private-memory flow: verified `~zod` on the fixed local development
origin, retrieved the code, created a temporary login, checked authenticated
`/~/name`, and logged out. The final run additionally verified that the old cookie
could no longer authenticate. It completed at `19:51:49.634400Z` with audit
`op-56650ba7572d171246eeb0cefa9399d1`, created `19:51:49.277810238Z`.

Both audit records identify the SSH operator fingerprint and contain outcome text
only. The script prints no code/cookie and uses no credential adapter or file.

The real clipboard already held a selection, so it was not replaced for testing.
An isolated foreground provider cancellation test passed, but **real clipboard
publication and its 30-second expiry are not claimed as live-tested**. This does
not block the separately validated code/Eyre path or subsequent provisioning work.
