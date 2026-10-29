# DMs uncertain send survives helper and profile replacement

The existing isolated DMs development checkout now saves exact send intent before
PUT using its instance-owned account lock, private atomic file and directory
fsync. An unknown response remains uncertain across restart. Exact authenticated
history on the same origin/author/recipient and message ID/content can resolve it;
a missing page, transport failure or different endpoint cannot. Public summaries
omit draft text; the private record retains it without session credentials.
Unknown journal versions and write failures prevent dispatch. Bounded history
retains all unresolved records and refuses admission rather than dropping them.

86 Python regressions pass, including10 new persistence/dispatch/reconciliation
cases. Original first failure was a test expecting raw OSError instead of the
existing SessionStore's wrapped StoreError; retained. JS and actual inert profile
QML tests pass. The real helper/private-keyring fixture sends one synthetic DM,
withholds its Gall receipt for the normal12second deadline, and sees busy clear.
Its new separate unresolved indicator remains. Stable→dev→stable stops exact old
helpers/streams before replacements; dev has no stable history and returning to
stable restores the same operation with an unchanged journal hash and no second
send. Final exact helpers/shell exit cleanly; all scopes are inactive.

This supersedes M249's in-memory-only send recovery limitation. It does not close
M246's undeleted-channel problem or qualify invitations/read markers, real Gall
message reconciliation, arbitrary plugins, graceful drain or collection activation.
No installed plugin, host keyring/desktop or production ship changed. Original
source snapshots, all attempts and private synthetic evidence are retained.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-dms-outbound-recovery`.
