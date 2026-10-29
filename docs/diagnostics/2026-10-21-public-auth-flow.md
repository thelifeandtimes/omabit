# Real approval flow and running plugin adoption

Part of the 2026-10-21 unreleased development series. Final frozen run:
`/tmp/omabit-auth-run-20261021-08`. The [evidence manifest](2026-10-21-public-auth-flow.json)
records source/artifact hashes, namespace IDs, exact sandbox arguments, test
results, nonsecret UI/CLI events and additional callback/unit checks. Source and
plugin hashes still matched when this record was created. All work remains
uncommitted in the three isolated development worktrees; nothing was installed.

## Actual path exercised

The existing fresh-fake namespace lab now builds the actual `urbit-collector`
executable and stages the real Theme/DMs declarations and helpers. It creates
separate `cli-lab` and `ui-lab` instances beneath private homes, with distinct
manifest IDs, visible labels, profile metadata, state roots and keyring namespaces.
No installed plugin or production home is mounted in the lab.

The public CLI gate discovers these installed declarations, enrolls only the
fresh local fake, signs in through stdin and approves each exact helper/ship
revision. Missing consent and stale helper digests are refused. Repeating an
accepted request returns the same attempt/revision; recovery preserves an
accepted session rather than issuing or delivering again.

The UI gate loads the actual `ApplicationsPane.qml` and
`ApplicationConnections.qml` with Quickshell offscreen, inert style values and a
small service adapter that invokes the actual collector. It enters the URL and
ship identity, signs in privately and approves both real receivers. Its audit
contains exactly four mutations: enrollment, parent sign-in and two explicit
approvals. Selection changes clear consent; each receiver becomes connected.
No code/cookie appears in its argument/log audit.

The panel then exits. The real plugin clients independently read and establish
SSE subscriptions with no collector service or parent client running. Each helper
is changed on disk after approval; disconnect and reconciliation still use the
retained approved generation. A revoked plugin cookie fails its direct probe.
Removing Theme's sign-in leaves DMs and the parent usable. DMs also remains usable
after the parent session is logged out and its enrollment forgotten. Cleanup then
closes and reconciles DMs using its independently retained evidence.

## Already running consumers

The real Theme `Service.qml` starts before enrollment/approval with
`desktopEnabled: false`. Its normal refresh finds the newly approved account.
The account's automatic/pending publication flags remain false; authentication
does not authorize desktop changes or publishing a theme. Its desktop adapter,
normal installed hooks and host compositor are not exercised.

A real DMs `Bridge` also starts with no saved account before approval. This
exposed a missing lifecycle step: its existing refresh path did not retry restore
after a receiver added an account. The bridge now restores a saved account on
explicit refresh when it has no active session, under the existing auth lock.
An active session is never silently replaced. The quick-open callback requests
restore when signed out; inbox-open already requests refresh.

The same running bridge picks up the approved account, emits its restored login
and refreshed events, opens its direct subscription and reads DMs. Its native
launcher adapter is disabled. The shipped quick/inbox opening callback separately
passes inert Qt tests; the full DMs overlay is not loaded in this gate.

## Results

- Independent-session runtime test: **PASS**, 0.30 seconds.
- Original real-receiver/direct-client/standalone-login gate: **PASS**, 16.26 seconds.
- Durable lost-acknowledgement/reopen/reconciliation gate: **PASS**, 6.29 seconds.
- New public CLI, real approval UI and running-consumer gate: **PASS**, 13.84 seconds.
- Overall: `test_exit: 0`, `clean_shutdown: true`, `log_truncated: false`.
- DMs affected unit suite: **59 passed**; source opening-callback Qt check passed.

Run `07` passed the public CLI/UI/receiver path before running-consumer adoption
was added. Run `08` includes that additional lifecycle gate and the final DMs fix.
The new panel log has no Qt binding/type errors. Existing race/static evidence for
the unchanged collector/auth implementation remains in the
[approval checkpoint](2026-10-21-approval-flow.md).

## Boundary and remaining work

The fake is freshly booted from the verified pill in new user/network/mount/PID
namespaces, with no external route or host home/desktop/keyring/Docker sockets.
The uniquely named scope remains capped at 4 GiB memory, no swap, one CPU core
and 128 tasks. Its private bus/keyring and the new fake are stopped before return.
Only read-only OS files, copied artifacts and fresh `/lab` state are available.

This proves the approval/receiver path and adoption by the two running consumer
backends. It does not qualify a full native desktop/profile installation, active
Theme publishing, native launchers, data writes, reconnect failure recovery or
hosted endpoints. Journals still cap attempts at 64; completed-record/cache
retention is the next bounded change. Marketplace/backend installation, fake
project networks and private Ethereum chains retain their separate roadmap gates.

Managed `~lismyl-lismyl`, tmux ships, `~/temp-ur`, installed plugins, production
credentials, runner services and the Docker daemon were not inspected or changed.
