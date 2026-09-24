# Collector ship sessions (checkpoint 4C)

Ship enrollment is separate from runner enrollment. A ship connection contains a
name, random local ID, canonical origin, expected ship identity, credential adapter,
revision, and opaque credential references. It does not need a runner or SSH.
Runner inventory is never treated as permission to authenticate to a ship.

## CLI

Global `--config-dir` and `--format text` options precede the subcommand:

```text
ship-enroll NAME --url ORIGIN --ship ~SHIP [--credentials secret-service|file]
ships
ship-login NAME --code-stdin
ship-status NAME
ship-scry NAME --app APP --path /PATH
ship-watch NAME --app APP --path /PATH [--events N] [--duration 30s]
ship-logout NAME
ship-forget NAME --url EXPECTED_ORIGIN
```

`ships` lists saved enrollment, not live health. `ship-status` checks current public
and authenticated identity and exits 1 if the session is unavailable, rejected, or
missing. `ship-scry` prints requested application data; the path omits the `/x`
care and `.json` extension. `ship-watch` prints connected/reconnecting notices and
subscription events, stopping after N diff events or the requested duration.
Ctrl+C cancels the stream and attempts bounded channel cleanup.

All commands support readable text. JSON remains the default; stream output is
NDJSON. Application data is intentionally returned by scry/watch, while session
cookies, login response bodies, and login codes are never CLI result fields.
Agent error stacks and arbitrary HTTP error bodies are not copied to diagnostics.

## Origin and identity boundary

Only HTTPS origins and HTTP on literal loopback addresses are accepted. Plain
`http://localhost` is intentionally not a literal loopback address: use
`http://127.0.0.1` or `http://[::1]`. Remote HTTPS uses the system trust store;
there is no skip-certificate-verification option. URLs cannot contain userinfo,
paths other than `/`, queries, or fragments. Default ports and host case are
canonicalized; distinct non-default ports remain distinct origins.

Each HTTP client has one immutable scheme/host/port origin and one expected ship
identity. There is no shared cookie jar or environment-proxy use. A restored
credential must match both origin and ship. Redirects are never followed, even to
the same host; the expected logout 303 is consumed without following Location.
Public identity is checked at `/~/host`, authenticated identity at `/~/name`.
Login sends a bounded form POST to `/~/login`, with the code read only from stdin.

Owner Eyre sessions are not application-scoped capabilities. Connection failure
establishes unreachability, not a crashed ship. Login/session authorization is
independent of runner SSH key authorization.

## Persistence and credential adapters

`ships.json` lives beside `runners.json` in the private collector config directory.
It contains no login code or cookie. The desktop default is Secret Service via
`/usr/bin/secret-tool`, with secrets passed on stdin and lookup output captured
privately. A locked/unavailable keyring is an error, never a silent fallback.

The explicit `file` adapter is for headless use. It stores private **plaintext**
session material under `<config-dir>/credentials/`, with an owned 0700 directory,
exclusive 0600 files, no symlink/hardlink acceptance, and file/directory sync.
This adapter does not claim encryption or isolation from same-account code.

Every login gets a new random credential reference, scoped to local enrollment ID
and exact origin. A compare-and-swap revision prevents a slow login/logout from
overwriting a newer session. Old references remain in a bounded cleanup journal
until credential deletion succeeds; `ship-logout` retries pending cleanup.
Forgetting enrollment requires logout/cleanup first. Watchers never write stored
credentials back and therefore cannot resurrect a detached session.

Session expiry is bounded by the login cookie and a conservative 30-day local
cap. Background activity does not silently extend the persisted local deadline;
log in again on expiry. Codes are not retained for automatic reauthentication.
Logout revokes only this enrollment's current/retired sessions and clears its
credentials, not other browser sessions. If the server is unavailable or the local
session expired, output distinguishes local clearing from confirmed remote logout.

## Subscription delivery and reconciliation

Each watch owns a cryptographically random Eyre channel and one subscription. It
creates the channel with a subscription PUT (no initialization poke), consumes
SSE, and acknowledges only after the consumer callback accepts an event. Event
**zero is valid**. Reconnect sends the last delivered ID, including zero, in
`Last-Event-ID`; duplicate events are suppressed. Partial frames are not delivered.

The parser bounds lines/events to 1 MiB. Eyre's 20-second heartbeat is monitored
with a 45-second idle timeout. Reconnect is bounded to three attempts with delays
of 250 ms, 1 s, and 3 s, resetting on delivered progress. Subscription creation
and application mutations are never blindly replayed after uncertain transport
outcomes. There is no generic poke/thread CLI in this checkpoint.

A 404 channel loss, or a gap after an established event cursor, requires
**application-specific reconciliation**. The generic transport reports that and
stops rather than pretend a new subscription restores missed state. Agent rejection
or quit also terminates the watch. An unavailable session requires explicit login.
New CLI invocations create fresh channels; there is no cross-process durable
subscription cursor or exactly-once application delivery claim.

On exit the collector tries a bounded delete of its own channel. Failure to confirm
cleanup is reported where it would otherwise look like normal completion. Eyre
eventually expires abandoned channels; the collector does not delete other clients'
channels.

## Evidence

The actual development fake passed authentication, persistent sessions through both
credential adapters, session replacement, authenticated status, desk scry, initial
SSE fact/ack/delete, logout, and local credential cleanup using the built CLI.
The upstream pill's observed desks were **base, groups, kids, landscape, webterm**.
This qualifies those installed desks, not the future Omabit pill bundle.

The verified desk scry is `hood /kiln/pikes`; the verified subscription is
`hood /kiln/updates`. The old `/kiln/vats` path is unsupported on this runtime.
Automated fixtures cover redirect/origin isolation, certificate trust, expiry,
revision races, retryable credential cleanup, event zero, replay deduplication,
consumer failures, agent rejection, truncated streams, and channel loss.

Sources consulted September 24:

- [Eyre HTTP API guide](https://docs.urbit.org/build-on-urbit/tools/js-libs/http-api-guide.md)
- [Eyre source](https://github.com/urbit/urbit/blob/5a187fededc4582a34fcd6055c67bb63e0917b94/pkg/arvo/sys/vane/eyre.hoon)
- [Hood/Kiln source](https://github.com/urbit/urbit/blob/5a187fededc4582a34fcd6055c67bb63e0917b94/pkg/arvo/lib/hood/kiln.hoon)
