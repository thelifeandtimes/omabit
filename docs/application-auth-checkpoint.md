# Application authentication checkpoint

Recorded 2026-10-21 against Omabit v0.2.0 (`b233128`), with uncommitted development
changes on `codex/platform-overhaul-plan`. This is an internal protocol experiment,
not a released connection feature. The later [Theme/DMs helper gate](diagnostics/2026-10-21-plugin-handoff.md)
now verifies private keyring handoff and direct transports. Product decisions are in the
[platform roadmap](platform-roadmap.md).

## Contract to prove

Theme and DMs are the first pair. Omabit assists authentication; each plugin owns
its session, credential storage, Eyre channel and direct application requests.
Ordinary URL + `+code` login must keep working without Omabit installed. Both
plugins use agents supplied by `%groups`; separate sessions are still required.
There is no inter-desk authorization enforcement in this checkpoint.

The smallest experiment starts from one authenticated owner session and creates
two distinct sessions. After the issuing clients close, fresh clients restored
from the two credentials can read directly from the ship. Logging out one rejects
replay of that cookie while the other plugin and original owner session survive.

## Implemented primitive

`internal/eyre/independent_session.go` adds `Client.NewIndependentSession(ctx)`:

1. Check the public ship identity and existing authenticated session.
2. POST JSON `null` to the fixed authenticated route
   `/spider/base/json/code/ship`, requesting JSON output.
3. Accept only a bounded JSON string matching the login-code format. Keep the
   code transient and omit it, response bodies and Spider stacks from diagnostics.
4. Create a fresh origin/identity-bound client and log in without a parent cookie.
5. Return the independently authenticated client; reject a reused parent token
   without logging that token out.

This reuses the transport's exact-origin restriction, identity checks, redirect
refusal, TLS validation, login expiry bounds and private session serialization.
404/405/501 responses identify unavailable assisted creation; other failures are
reported without replaying login or silently changing authentication mode. The
future UI should keep standalone login available on failure, with no automatic
request for secrets and no automatic backend installation for this experiment.

Neither the code nor the parent cookie goes to a plugin. Only the new session is
intended for private delivery. The method is internal and has no CLI, QML, approval
UI, receiver or persistence integration. Its caller must implement those lifecycle
steps before exposing assisted login. Existing `+code` login remains unchanged.
Go byte buffers are cleared where practical; this does not claim complete memory
zeroization of strings or runtime copies.

## Source evidence and qualification limit

The mechanism is inferred from Urbit source commit
`50c7f5239610fc72263df7aa99b4d171cfe9d408`:

- [`ted/code.hoon`](https://github.com/urbit/urbit/blob/50c7f5239610fc72263df7aa99b4d171cfe9d408/pkg/arvo/ted/code.hoon)
  ignores the argument and returns the local login code as an `@p` vase.
- [`mar/ship.hoon`](https://github.com/urbit/urbit/blob/50c7f5239610fc72263df7aa99b4d171cfe9d408/pkg/arvo/mar/ship.hoon)
  converts that value to a JSON string with its leading `~`.
- [`app/spider.hoon`](https://github.com/urbit/urbit/blob/50c7f5239610fc72263df7aa99b4d171cfe9d408/pkg/arvo/app/spider.hoon)
  requires authentication and dispatches the desk/input-mark/thread/output-mark
  route. The [thread HTTP API](https://docs.urbit.org/urbit-os/base/threads/http-api)
  describes this route structure.
- [`vane/eyre.hoon`](https://github.com/urbit/urbit/blob/50c7f5239610fc72263df7aa99b4d171cfe9d408/pkg/arvo/sys/vane/eyre.hoon)
  replaces the supplied session on login; a cookie-free login creates another.
  Logout without an `all` or `sid` parameter closes the request's session. See
  also the [Eyre HTTP reference](https://docs.urbit.org/urbit-os/kernel/eyre/external-api-ref).

A fresh fake ship has now exercised this route successfully in an isolated
namespace; see the [real-runtime record](diagnostics/2026-10-21-isolated-auth.md)
for executable/pill hashes, test results and limits. A hosted ship's runtime,
pill, proxy or Spider availability may differ. This does not establish hosted
compatibility or actual plugin integration.

## Automated evidence

`internal/eyre/independent_session_test.go` exercises synthetic loopback HTTP:

- Three distinct sessions, with no old cookie attached to either new login.
- Private serialization/restoration into fresh Go clients after all issuing
  clients close; direct reads at the paths used by Theme (`settings /desk/talon`)
  and DMs (`chat /dm`). No auth helper participates in those reads.
- Server rejection of a copied cookie after logout, with the other two sessions
  still authenticated.
- Missing thread support, redirects, thread errors, incorrect content type,
  malformed/oversized credential responses and a reused session token.
- Identity mismatch before requesting a code. Failures preserve the parent
  session and do not leak the fixture code or private error output.

These are Go transport tests at representative plugin paths, not tests of the
actual Python plugins, real Secret Service, SSE lifecycle or a hosted ship.
The later receiver checkpoint edits separate Theme/DMs worktrees and tests copied,
profiled helper artifacts. The current installed plugins remain unchanged.

Validation passed with the live fake-ship test disabled and temporary Go caches:

```sh
OMABIT_TEST_FAKE_ZOD=0 go test -race ./internal/eyre ./internal/shipaccess ./backend/cmd/urbit-collector
OMABIT_TEST_FAKE_ZOD=0 go vet ./internal/eyre ./internal/shipaccess ./backend/cmd/urbit-collector
```

`internal/shipaccess` has no test files; it was compiled and vetted. Eyre and
collector suites passed. Formatting, diff whitespace and local documentation
links were checked. The earlier import checks are recorded separately in the
[import diagnostic record](diagnostics/2026-10-21-import-diagnostics.md).

## Next small integration gate

The fresh-fake route and independent logout gate passed with the original
session preserved. Failed issuance remains covered by synthetic fixtures.
Keep the user's existing ships and services excluded under the latest
[experiment constraints](experiment-isolation.md).

The private, profile-bound receivers now pass the separate headless plugin gate.
The internal [durable issuer fixture gate](diagnostics/2026-10-21-durable-issuer.md)
also passes, followed by the [verified receiver/recovery runtime gate](diagnostics/2026-10-21-receiver-recovery.md).
Next connect installed-plugin declarations, retained inventories and explicit
approval/recovery actions to the isolated desktop UI. The
receiver validates and durably stores its own session before acknowledging it;
cancelled or failed handoffs require cleanup, with uncertain remote logout retained
as pending rather than reported as success. Define ownership across interruptions
before adding the approval UI. Do not print a cookie as CLI JSON or pass it in
argv, environment variables or URLs.

Acceptance for the pair: assisted and standalone login; direct reads, writes and
subscriptions with Omabit stopped or absent; independent restart/logout; locked
keyring and expired sessions; and profile switching with no stale watcher/result
writing into the new profile. A shared package follows only after both plugins
work. Desk capabilities/Gall 2026 and wallet implementation remain deferred.

No installed plugin, production credential store, collector/runner service,
Docker resource or existing ship was changed. Only the isolated fresh fake was
booted and stopped. The implementation remains on the development worktree.
