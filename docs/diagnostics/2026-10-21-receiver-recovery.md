# Verified receivers and interrupted-account recovery

2026-10-21, unreleased development work. The final frozen headless artifacts ran
in `/tmp/omabit-auth-run-20261021-06`; [the evidence manifest](2026-10-21-receiver-recovery.json)
records their hashes, source hashes, namespace IDs, exact sandbox command and
result. Omabit remains based on `b233128`, Theme on `1719e9f` and DMs on `cf236fd`,
with uncommitted changes in the separate development worktrees.

## Behavior demonstrated

Both real Python receivers now recover the exact account created by an
interrupted handoff. Before the first possible keyring write they persist a
nonsecret attempt record with request/connection/plugin/profile binding and a
credential fingerprint. Theme's existing account ID identifies the attempt; DMs
adds an optional `application_request_id` to its nonsecret account metadata.
Standalone DMs login writes the original account shape, replacing that binding.

After the issuer confirms remote revocation or rejection and deletes its own
credential copy, it sends a bound, credential-free reconciliation request. The
receiver clears only the matching stored session/account. A later standalone
login survives. Repeating a completed reconciliation returns its saved receipt
without touching a newer account. A keyring clear or account removal interrupted
before the receipt can safely resume.

The exact origin's keyring metadata is checked through `SearchItems`, which
distinguishes unlocked and locked matches. No match is different from a locked or
unavailable service; duplicate matches also remain unresolved. This uses
`/usr/bin/busctl` only for nonsecret metadata and the existing `secret-tool` adapter
for the cookie. See the [Secret Service API](https://specifications.freedesktop.org/secret-service/latest-single/)
and [busctl documentation](https://www.freedesktop.org/software/systemd/man/latest/busctl.html).

The Go launcher hashes an explicit Python/JSON inventory, including the manifest,
profile and entry point, and durably retains that generation before approval.
Loading by digest survives source/plugin updates; a changed retained file is
refused. Each execution uses a new private copy of the approved bytes, explicit
HOME/XDG/bus environment, isolated Python startup, private stdin, bounded stdout
and a deadline. Arbitrary stderr and failed output are not returned to the UI.
The owned process group is terminated while its leader's PID is still reserved,
before reaping; unrelated processes are never selected for cleanup.

This pins helper code. It trusts the system Python/standard library and does not
claim a security sandbox against malicious same-account plugins. Full stable/dev/edge
desktop isolation and installed-plugin discovery are separate remaining work.

## Live qualification

The final run used the existing [fresh-fake boundary](../experiment-isolation.md):
new mount/PID/network/user namespaces, no host home/desktop/keyring/Docker sockets,
private loopback, its own D-Bus and GNOME Keyring, and the bounded transient scope.
Vere was the read-only copied 4.6 executable and verified pinned pill used in the
prior authentication checkpoints; no installed runtime was changed.

- Independent-session test: **PASS**, 0.31 seconds.
- Original Theme/DMs private helper, direct read/SSE, independent logout and
  standalone-login test: **PASS**, 16.32 seconds.
- New durable-engine/real-receiver test: **PASS**, 6.27 seconds. For each plugin:
  retained artifact → approval → receiver commit → deliberately lost acknowledgement
  → remote session revocation → reopen journal/load retained artifact → matching
  local-account removal → new approval → successful direct client use.
- Revoking/reconciling Theme leaves DMs and the parent login usable. Both plugin
  connections can be cleaned independently.
- Overall `test_exit: 0`, `clean_shutdown: true`, `log_truncated: false`.

Run `05` also passed; run `06` froze the final DMs different-origin cleanup case.
The new engine is tested alongside the earlier primitive/helper tests rather than
replacing their independent coverage. No collector or runner service exists in
the lab, and the plugins use their own Eyre clients for all data requests.

## Maintained fixture evidence

Go race tests and vet passed for `internal/applicationauth`, `internal/collection`,
`internal/credentials`, `internal/eyre` and `backend/cmd/urbit-collector`, with the
historical `OMABIT_TEST_FAKE_ZOD` live opt-in disabled.

Theme's affected baseline set passed **137 tests**; the final recovery plus
installer set passed **26 tests**. DMs' final affected set passed **57 tests**.
These exercise partial credential/account writes, locked/missing keyrings,
wrong bindings, newer standalone accounts, receipt replay, source changes,
retained-artifact tampering, bounded output and owned-child termination.

An oversized-output fixture caught Go's promoted `bytes.Buffer.ReadFrom` bypassing
the custom `Write` limit during `io.Copy`. Named buffer storage fixes that in both
the new receiver launcher and the existing credential subprocess adapter; a file
copy regression test now covers the credential adapter as well.

## Remaining work and exclusions

There is still no exposed application approval CLI or QML flow. Next connect
installed-plugin capability declarations to retained inventories and explicit
approve/disconnect/reconcile actions, then run the complete UI with separately
labeled experimental artifacts and private desktop boundaries. Completed-record
retention, writes/reconnect, hosted compatibility and profile switching also need
their remaining gates. Unknown issuance without a received/stored token remains
explicitly unresolved; it cannot authorize another automatic login.

No current installed plugin or production credential was changed. Managed
`~lismyl-lismyl`, tmux ships, `~/temp-ur`, the Docker daemon and runner services
were neither inspected nor operated on. Only fresh disposable fake ships were
started and cleanly stopped.
