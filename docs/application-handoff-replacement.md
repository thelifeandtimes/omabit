# Fleet session replacement — implementation plan

Status: in progress after M571. This is not an advertised protocol or a completed
fleet feature. Both receivers' preparation, activation, recovery and exact
transport lineage pass private source fixtures, including DMs cooperative bridge
control. Collector lineage, capability compatibility and issuer recovery pass private fixtures
(M570). Public reconnect/recovery and desktop polling pass private CLI/full-panel fixtures
(M571). Real receiver capability advertisement and assembled native qualification
remain.

## Required behavior

Standing app/profile/ship permission authorizes selection among eligible registered
URLs. A reconnect creates a fresh session for the chosen origin. It never copies
an old cookie to another URL or replays an application write. Current permission,
exact helper, runner/ship incarnation, route eligibility and a bounded identity
probe remain required. Changed/missing credentials, wrong identity, redirects or
unknown effects must not be interpreted as permission to log in again.

## Receiver sequence

1. Bind preparation to the accepted old request/connection, exact account and
   credential fingerprint, plus a fresh successor request ID. Reject an existing
   successor receipt. The receiver fences old local use, joins workers/streams,
   and checks unresolved application journals. Preparation changes no account or
   credential and performs no remote cleanup. Its response is not a reservation.
2. After preparation, the issuer may create one independent session. Before
   delivery the receiver repeats all old-account checks under the account lock,
   verifies the new origin/session, and proves local quiescence again. A concurrent
   standalone account change wins; do not overwrite it.
3. Save nonsecret replacement intent before any credential change. Preserve the
   old exact-origin credential in a request-specific Secret Service namespace,
   verify its fingerprint, then write the new credential and account. A retired
   item must use a distinct **application** attribute, not merely an extra tag on
   the ordinary application/origin pair: an old lookup by those two attributes
   would otherwise match both items. No plaintext fallback.
4. Preserve user preferences when replacing that account, without implicitly
   enabling preferences the user had disabled. Record accepted delivery only after
   the account and new credential are verified. Keep both old/new attempt ancestry
   and the original cleanup/write evidence. New transport admission must recognize
   only the explicitly authorized successor; retirement alone permits no new channel.
5. Cleanup of the old generation uses its original credential/origin and retained
   helper. It must not clear the new origin slot or new account. An unavailable old
   origin remains cleanup-pending while the verified new connection can operate.
   Bound history limits remain explicit; do not discard unknown work to make room.

An interrupted replacement needs dedicated recovery: distinguish old account/key,
new key before account commit, accepted successor, and a later standalone account.
Do not apply version1's early "different account means superseded" shortcut to a
new-key write that occurred before its account replacement. Retain the old backup
until its own cleanup is confirmed. Lost acknowledgements must not cause another
issuance or delivery. Reconciliation remains available after permission revocation.

Theme supports multiple accounts in one profile. Any transport extension must let
retained older helpers finish cleanup of unrelated accounts; a global format change
cannot silently break their cleanup. Prefer an explicit compatible sidecar for
successor lineage while retaining the existing version2 transport shape. Validate
and bound that sidecar with the same ownership/symlink/duplicate-key discipline.

DMs supports one active account and holds a transport lease for the bridge
lifetime. M569 implements the bounded same-user local request path on the main
loop, outside the pool it drains. It keeps the bridge alive while paused; verified
successor adoption creates a fresh pool and discards only old in-memory views.
Original cleanup can run through that current bridge without stopping new traffic.
Private fixtures cover this sequence; assembled native qualification remains.

## Issuer and UI sequence

The collector needs durable predecessor/successor lineage before preparation or
issuance, with one active successor per exact integration/ship binding. Extend the
actual journal validation and migration using previous-writer fixtures; do not
relax the existing cross-URL uncertainty fence for unrelated attempts. Preserve
old cleanup records rather than marking the predecessor disconnected prematurely.

Advertise complete receiver capability only after preparation, activation, recovery
and transport admission work together. A changed helper digest still requires the
user's current grant; legacy grants are not silently widened. Cached generations
remain available for their own cleanup. Automatic health/reconnect polling must be
bounded, serialized with disconnect/revocation, and must not replay an uncertain
request after UI refresh or process restart.

Verify the assembled path with private synthetic endpoints, both real receiver
sources, isolated Secret Service transport, active workers, URL changes, expiry,
revocation, standalone replacement, lost replies and original-writer preservation.
Production/DEV/trial plugins, runners, protected ships and the persistent manual
preview remain excluded. Release qualification/publication remains deferred.


## M568 implementation boundary

Theme's isolated4365455 implements the receiver sequence and preserves older
unrelated-account cleanup formats with a sidecar. Its complete helper inventory
includes the new modules. No capability flag is advertised yet. Automatic handoff
must require a predecessor helper that understands replacement cleanup; a changed
installed helper/grant cannot silently upgrade an older cached predecessor. Bound
status observation handles lost delivery replies without another issuance/delivery.
See [source evidence](diagnostics/2026-11-07-theme-session-replacement.json).


## M569 implementation boundary

DMs464076b implements the cooperative receiver sequence and source-level successor
adoption/recovery. The helper inventory includes all imported modules. Theme's
201fdd5 follow-up preserves a later independent key/metadata change during partial
swap cleanup. Both remain unadvertised and isolated; see
[source evidence](diagnostics/2026-11-07-dms-session-replacement.json).


## M570 implementation boundary

Collector format4 ancestry, shared generation locks, exact approved capability2
and one-time preparation/issuance/replacement are implemented in the isolated
branch. Lost acknowledgements use bound observation rather than another login or
offer. Full registry/issuer and focused Eyre fixtures pass, including the actual
previous writer. See [issuer evidence](diagnostics/2026-11-07-fleet-replacement-issuer.json).
Public commands, automatic polling and assembled receiver qualification remain.


## M571 implementation boundary

Public reconnect, recovery observation and bounded maintenance are implemented.
Desktop polling is serialized and respects activation, exact current permission
and helper capability. Explicit disconnect cancels unaccepted handoff intent;
accepted successors retain independent cleanup. CLI, full registry/issuer, UI and
hosted regressions pass. See [CLI/UI evidence](diagnostics/2026-11-07-fleet-reconnect-cli-ui.json).
The remaining assembled test must use both real receiver sources, isolated
Secret Service and active transports before declaration2 is advertised.
