# Fleet session replacement — implementation plan

Status: in progress after M567. This is not an advertised protocol or a completed
feature. Initial connections and bounded health checks exist; Theme preparation
and DMs retirement/drain are tested foundations in isolated receiver branches.

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

DMs supports one active account and currently holds a transport lease for the whole
bridge lifetime. Its new local drain is not yet wired to a cooperative request.
The main loop needs a bounded private request/response path that does not run a
pool shutdown from one of that pool's workers. Keep the bridge alive while paused;
after verified activation it can create a fresh worker pool, discard only old
in-memory views, restore the new account and emit normal login/refresh events.
Do not require the user to restart the UI or use remote deletion as local exit proof.

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
