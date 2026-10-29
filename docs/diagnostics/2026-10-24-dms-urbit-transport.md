# Actual Urbit transport recovery and owned-stream cancellation

The fresh-fake test uses the previously verified Vere/pill artifacts, a separate
DMs profile and real private Secret Service in new user/network/PID/mount namespaces.
There is no host home, desktop bus, Docker socket or external interface. A bounded
loopback HTTP relay forwards to actual Eyre and drops one successful delete reply;
it creates no synthetic acknowledgements. All application pokes are forbidden.

Trial02 proved initial four Gall subscriptions, owner SIGKILL, actual204 deletion
with a lost reply, cleaner SIGKILL and recovery using the same session/request.
It then failed shutdown: actual deletion did not unblock the existing event-stream
read. DMs correctly returned stream-still-open. The fixture killed only its owned
helper and stopped the fresh fake cleanly. Trial01 had stopped before boot on a
Python import scoping error. Both failures are retained.

The bridge now keeps a duplicated descriptor for each exact generation/channel
stream. After confirmed channel deletion, it shuts down that owned socket before
joining the reader. This wakes buffered reads without blocking on another thread's
HTTPResponse.close or risking a reused descriptor. Delayed opens reject obsolete
bindings; stale reader cleanup cannot cancel a replacement. Failed remote deletion
still retains the original journal attempt and does not claim closure. Thread exit
remains required even when socket shutdown fails.

All150 private DMs tests pass, including blocked socket reads, late open and stale
reader cases. The corrected fresh-Urbit trial03 passes eight real subscription
acknowledgements, both helper kills, lost-reply replay of the same delete, preserved
session, successful final drain, and independent actual404 responses for both
closed channels. The private session is then revoked and the disposable fake exits
zero after SIGTERM. There are zero application pokes. Four actual-runtime writer
snapshots and exact artifact hashes are retained. The complete private collection
regression also passes all16 deletions, rollback and shell replacement, while its
reader-owner-lost fence remains durable. All named scopes are inactive.

M274 supersedes M273's synthetic-only transport limit for this fresh fake and the
specific stream-close behavior exposed by trial02. The earlier synthetic test is
still the evidence for preserving three uncertain app writes without replay.
Real-Urbit mutation uncertainty, hosted TLS/gateways, absent/expired credentials,
history archival, old-helper adoption, native desktop and independent coordinator
recovery remain separate gates. No installed plugins or existing ships changed.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-dms-urbit-transport`.
Fresh private piers remain in the named trial02/03 directories; raw runtime logs,
keyrings and account credentials are intentionally outside the evidence export.
