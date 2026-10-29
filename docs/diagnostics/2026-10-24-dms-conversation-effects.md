# DMs durable invitation and read-marker intent

Invitation/read-marker operations now persist exact ship/origin/peer/kind references
before HTTP dispatch in a separate version1 private journal. Existing message
journal format is unchanged. Unknown formats/private-state failures refuse dispatch;
bounded retention never evicts unresolved actions. Only exact Gall acknowledgement
or rejection completes an effect. Read markers retain local unread counts until
acknowledged and reject UI effects from a stale connection generation.

All101 Python tests and six profile QML checks pass. Initial failure was the test
expecting raw OSError rather than SessionStore's documented StoreError wrapper;
the no-dispatch assertion remains. The first runtime fixture incorrectly returned
an inbox-shaped list for DM history and never reached mark_read; the corrected
fixture returns the existing client's required writs/older page. Both failures
are retained. The passing private actual-helper/keyring run sends exactly three
synthetic actions with missing receipts: message, invitation, read marker. Across
stable/dev/stable/edge/stable their unknown identities/files persist only in stable,
with no replay. All ten channels are deleted before replacement and helpers exit.
The exact passing fixture is applied as optional --connected --drain --effects.

This extends M253/M256 from sends to these two conversation effects, not all app
or external-launcher effects. Authenticated observation-based reconciliation of
unknown invitation/read intent is still pending; absence is not completion.
collectionReady=false remains explicit. Source hashes and inactive scopes verify.
No installed plugin, real ship, desktop, host service or keyring was changed.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-dms-conversation-effects`.
