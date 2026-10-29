# Theme credential-preserving disconnect

Disconnect now pauses the selected account, persistently retires its session,
waits for admitted reads/channels to exit, and confirms deletion of every pending
channel before removing the account or clearing Secret Service. Shared per-origin
locks span whole reads and channel operations; exclusive retirement waits for
those locks, including operations with no channel. Watch loops observe the fence;
existing writes may finish their receipt/cleanup, while new work with a retired
session is refused. Timeout, missing credentials for open channels, unknown owner
or failed delete keep the paused account and saved credential for explicit retry.
A new login requires a fresh session. Retired-session fingerprints remain fenced
across process loss. No raw secret or settings payload is added to the journal.

Format2 adds bounded session fingerprints (2048). Format1 migration preserves
all channel records and requires earlier owners to be released or verified dead.
An unchanged M276 writer and its actual populated final snapshot are retained:
they roundtrip under the original reader, migrate intact and then refuse the new
format without changing it. The original channel/owner/byte limits still apply;
full history refuses intact. This does not authorize rolling back to a pre-journal
helper, for which explicit adoption/installer controls remain a release gate.
Handoff cancellation uses the same transport barrier before forgetting a token.
A locally expired token can be presented for exact cleanup/logout, never ordinary
work; remotely expired/revoked credentials with open channels remain unresolved.

All284 private Python tests pass, including13 focused disconnect checks, original
writer upgrade/downgrade checks and receiver preservation. The first full run's
nine failing subcases were stale mock expectations that transport was never
consulted for an unavailable keyring; retained failures precede corrected coverage.

Actual CLI/process qualification uses a private real Secret Service, unmodified
Theme client code and synthetic HTTP peer. A concurrent publisher is killed after
one accepted settings write without a receipt. Disconnect stops the original watch;
failed deletion preserves account and credential. New watch/read/write processes
refuse before HTTP. A retrying disconnect process is killed after its accepted
DELETE reply is withheld. The next CLI preserves the request, closes both channels,
then removes the account, logs out once and clears its private credential. The
uncertain application intent remains unchanged and is never replayed. Five writer
snapshots and exact process exit evidence are retained. The real-QML collection
regression also passes all16 deletions and preserves reader-owner-lost.

M277 supersedes M276's unsafe-disconnect prerequisite for these cooperating paths.
Fresh actual-Urbit Theme qualification is running separately and is not claimed
here. Hosted TLS, independent coordinator recovery, journal archival, native loader
identity, pre-journal helper adoption and full desktop deployment remain open.
Logout/keyring failures after confirmed transport closure retain the existing
explicit unconfirmed-cleanup warning; they are not reported as remote revocation.
No installed plugin, production credential/service or existing ship was touched.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-theme-safe-disconnect`.
