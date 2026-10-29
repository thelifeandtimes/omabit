# DMs durable channel cleanup and recovery

The development DMs bridge writes version1 `transport/channels.json` before its
first channel request. Each row binds plugin/profile, origin, ship, a one-way
session fingerprint, channel, creator process identity and cleanup attempt.
A private lifetime lock excludes competing helpers. PID/start/boot/namespace
checks distinguish verified death from an unknown owner; unknown owners refuse.
No cookies, codes or application bodies are written to this journal.

Cleanup saves a stable request before the exact direct-to-Eyre delete. An empty
HTTP200/204 response closes it; ambiguous replies retain the same attempt. The
saved origin selects the target. Local persistence failures also pause application
writes. Restore cleans the earlier channel before a replacement can subscribe;
failed cleanup preserves credentials and blocks account replacement. Application
send/effect journals remain separate and recovery never replays them.

All147 private DMs tests pass. The runtime fixture uses unchanged actual DMs main,
separate plugin labels/state and a real private Secret Service, against synthetic
Eyre on namespace loopback. It sends three deliberately unacknowledged app actions,
rejects a simultaneous helper, kills the owner, drops an accepted delete reply,
kills its cleaner and restarts. The next helper reuses the same cleanup request,
deletes the old channel and restores all four subscriptions. Another stream drop
and lost delete reply exercises in-process reconnect. All three channels close,
application journals are byte-identical, the private credential remains usable,
and there are exactly three original pokes with no replay. Six genuine writer
snapshots cover open, closing, recovered, reconnect and closed history.

The final source also passes the existing real-QML private collection fixture:
stable/dev/stable, failed edge appearance and rollback, six coordinator kills,
two controlled shell replacements and final unclean shell loss. All16 channels
delete; reader-owner-lost remains durable. This evidence does not authorize the
coordinator to erase that fence. Source hashes and named inactive scopes were
independently checked. An earlier test-fixture indentation error is retained;
corrected trials and final source pass.

M273 supersedes M272's missing DMs helper-death/reconnect transport evidence for
this private synthetic-peer scenario. Actual Urbit transport, missing/expired/
changed credentials, independent coordinator recovery authorization, native desktop
behavior and production rollout remain unqualified. This journal cannot recover
channels opened by older helpers that never recorded them. Adoption requires a
verified old-helper drain; downgrading to code that ignores it is unsupported.
History is bounded at1024 rows/2MiB and refuses intact when full; supported archive
handling remains a release gate. State is0700/files0600, but Omarchy plugins still
share the user's OS authority: this is crash coordination, not a hostile-plugin
security boundary or wallet-key-storage solution.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-dms-durable-transport`.
