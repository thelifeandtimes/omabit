# Private application-session handoff, version 1

Development contract for the Theme/DMs experiment, not an exposed Omabit command
or a completed approval feature. See [goal progress](goal-progress.md) and
[experiment isolation](experiment-isolation.md).

## Ownership and transport

The plugin owns its credential store and direct Eyre transport. Omabit assists
with approval and creation of a new session; it does not serve application data.
Ordinary URL + `+code` login remains available without any Omabit executable.
Both sessions are ordinary broadly authorized owner sessions. No desk permission
system or application-data proxy is part of this contract.

The production sender must launch the approved, immutable receiver artifact with an
argument vector, not a shell command, and sends one bounded JSON offer on private
stdin. Only a nonsecret acknowledgement returns on stdout. No cookies or codes
pass through QML, public command results, argv, environment variables or URLs.
The receiver has no Omabit runtime/library dependency.

Receivers are implemented at:

- Theme: `client/application_connection.py` in the Theme repository.
- DMs: `application_connection.py` in the DMs repository.

They are independently implemented while the pair is being qualified; a shared
client package has not been extracted. Their `application-connection.json` declarations and the unreleased
[Omabit CLI/approval panel](application-connections.md) now have fixture and
offscreen coverage and a [real-pair public-flow runtime gate](diagnostics/2026-10-21-public-auth-flow.md).

## Offer and acknowledgement

The offer has exactly these fields; duplicate or unknown fields are rejected.
Input is at most 16 KiB and the receiver operation has a 25-second deadline.

| Field | Requirement |
| --- | --- |
| `version` | Integer `1` |
| `request_id` | Fresh 32-character lowercase hex ID for this attempt |
| `connection_id` | Collector's 32-character lowercase hex ship connection ID |
| `plugin_id` | Exact locally installed receiver identity |
| `profile` | Exact local instance profile, including `default` when unprofiled |
| `session` | Private object with `origin`, `ship`, `cookie_name`, `cookie_value`, `expires`, matching Omabit's session serialization |

`origin` is canonical HTTPS or literal-loopback HTTP, without credentials, paths,
queries or fragments. `ship` matches the cookie name and expected ship identity.
`expires` is an aware RFC3339 timestamp, still valid and no more than 30 days away.
Local expiry remains effective on credential lookup and during direct transport;
the server may expire a session earlier.

The receiver checks public `/~/host` first without the cookie, then `/~/name` with
the cookie, using direct connections with no redirects or environment proxies.
Authentication and backend readiness are distinct: it does not require a desk to
exist before saving the connection, and never installs a backend during handoff.

On success the acknowledgement echoes `version`, `request_id`, `connection_id`,
`plugin_id` and `profile`, with `status: accepted`. It contains no session or code.
On failure it returns `version`, `status: rejected` and a fixed diagnostic code,
with nonzero process exit. Arbitrary HTTP bodies, exception payloads and keyring
output never become acknowledgements.

## Receiver persistence

Each receiver verifies the keyring write before committing nonsecret account
metadata. File writes and parent directories are synced before acknowledgement.
Theme uses its existing state lock; DMs now serializes account writes/removal
across processes with a private file lock. Neither silently falls back to plaintext.

The first implementation accepts an empty destination account only. Theme may add
another origin to its multiship state; DMs may add its one account. Neither
replaces an existing origin/account during handoff. Explicit replacement and reauthentication remain follow-up work. Interrupted
receiver recovery is implemented as the bound reconciliation operation below.

Theme initially records the account without enabling publication or desktop sync.
Authentication alone does not change the desktop or publish its theme. DMs also
starts no watchers or launchers from its receiver; the plugin owns those lifecycles
when it subsequently loads the saved account.

Current local account formats remain unchanged except for an optional `expires`
field in the private keyring session. Legacy standalone sessions remain readable.
The collector now has an internal persistent approval/attempt journal; see the
[durable issuer checkpoint](diagnostics/2026-10-21-durable-issuer.md). Each receiver also persists a bounded, nonsecret `application-attempts.json`
before the first possible keyring write. It binds the request, connection, plugin,
profile, origin, ship and session fingerprint. Theme uses its attempt-derived
account ID; DMs adds an optional `application_request_id` to account metadata.
Standalone DMs login writes the original metadata shape, replacing that binding.

## Sender lifecycle and remaining production adapter

The trusted caller must obtain approval for the exact plugin artifact/profile, target
connection and ordinary owner-session authority. Resolve the receiver from that
approved source; do not execute arbitrary command text supplied by a request.

The internal engine durably records an approved attempt and an opaque credential
reference before issuance, then saves and verifies the token before delivery.
The issuer retains cleanup ownership until a valid, matching acknowledgement is
durably recorded. It retains a private credential copy afterward for an explicit
management disconnect; plugins continue independently with their own copy.
A rejection, timeout, cancellation,
invalid acknowledgement or lost acknowledgement must not silently become success
or automatic retry. Revoke only that newly issued session; retain the reference
and pending status until remote cleanup is confirmed. Never revoke the parent or
another plugin's session. Unknown outcomes need reconciliation.

A receiver can commit account state before its acknowledgement is lost. Issuer
revocation then makes that stored session unusable; do not blindly erase unrelated
plugin state. Recovery must reconcile the specific attempt/account before a new
handoff. The durable engine now has maintained crash/restart and failure fixtures;
the [verified artifact and real-receiver recovery gate](diagnostics/2026-10-21-receiver-recovery.md)
also passes. Installed-plugin capability discovery and explicit approval UI/CLI
now have a [fixture/offscreen checkpoint](diagnostics/2026-10-21-approval-flow.md);
the [real public-flow gate](diagnostics/2026-10-21-public-auth-flow.md) also passes. The original live driver stays as an independent baseline;
the harness now also runs the durable engine with both real receivers.

A process can die after a successful remote login but before receiving/storing its
cookie. That attempt remains unresolved, with no automatic issuance or claim of
revocation. Missing credentials never imply cleanup. Known pre-login failures and
a missing code thread are recorded separately as not issued. The active journal
retains at most 64 attempts. Fully closed/reconciled attempts can now be explicitly
archived to permanent replay receipts; unfinished evidence has no force-forget
bypass. Receiver compaction retains reconciled receipts and keeps pending/accepted
entries. See [completed history](application-connections.md#completed-history)
for durability, version fencing and older retained-helper compatibility.

## Retained receiver generation

The trusted desktop adapter supplies an explicit Python/JSON helper inventory.
`OpenReceiver` binds its entry point, manifest, optional profile and all file hashes;
`Retain` publishes that generation durably before approval. `LoadReceiver` checks
the digest again after a restart or plugin update. A launch copies those retained
bytes into a private directory and uses fixed Python arguments, explicit
HOME/XDG/bus environment, bounded private pipes and a deadline. It never executes
a request's shell command or sends a cookie to a mutable checkout. System Python
and same-account code remain trusted; this is not the full desktop sandbox.

## Receiver reconciliation

After remote revocation/rejection and issuer credential deletion are durable,
`Engine.Reconcile` invokes the approved receiver with `--reconcile`. Its private
stdin contains exactly the five acknowledgement binding fields plus
`action: reconcile`, `origin` and `ship`; no session/cookie is sent. The reply has
the same five binding fields, `status: reconciled` and `outcome` equal to `removed`,
`absent` or `superseded`. The issuer clears its sticky reconciliation fence only
after that matching acknowledgement is durable.

The receiver uses its existing account lock, exact attempt binding and credential
fingerprint. It preserves a newer standalone account. Missing keyring items are
checked separately from locked, duplicated or unavailable ones using Secret
Service metadata through `busctl`; secrets still use the existing adapter.
Credential clear and account removal may be repeated after an interruption. A
durably completed receipt returns without touching a later account. These are
trusted local lifecycle operations, not a new interdesk permission system.

## Experimental instances and current limits

An optional `application-profile.json` contains only integer `version: 1` and a
validated profile name. A non-default profile requires the manifest ID to be the
canonical plugin ID plus `.<profile>`. State and Secret Service application names
gain the same suffix. A mismatched manifest/profile is refused.

The lab stages `thelifeandtimes.urbit-theme.auth-lab` and `urbit.dms.auth-lab`, with
visible Auth Lab names, private HOME/XDG roots, an isolated D-Bus and a private
Secret Service. It tests the real Python helper code, not the desktop UI. No
ordinary installed plugin or desktop keyring is involved.

These per-plugin credential namespaces are not the full stable/dev/edge collection
feature. Theme's desktop hooks/config application, plugin IPC, launchers and actual
UI loading still need collection-wide isolation and qualification before enabling
experimental interfaces alongside the current desktop.

DMs' direct HTTP transport now rejects redirects and ignores proxy environment
settings. Its logout requests only its saved session's remote logout, retains
credentials on uncertain network outcomes, and distinguishes observed revocation
from rejection in the helper result. Neither behavior introduces a collector data
dependency.
