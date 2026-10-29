# Linked successor ownership before activation

A separate immutable successor intent binds one preserved checkpoint, its original
canonical coordinator/lock/mirror and exact journal, the previously closed plugin
cohort, a new live display/runtime, and a new authority identity. It does not create
an authority descriptor, admission ticket, loader or application. Preparation
requires the same committed collection and no intervening journal events.

The adapter must inspect current plugin state while preserving its admission
boundary. Application hashes, pending IDs, credential references, channel IDs and
closed transport observations must exactly match the verified original cleanup.
Validation runs before staging and again before publication. The new runtime claim
is staged privately, synced, and bound by inode/hash in the immutable intent before
no-replace publication. A lost reply reuses that inode. Unbound stages remain
retained; missing or replaced bound stages/claims are refused, never recreated.
One intent owns a predecessor. Existing fresh provisioning paths refuse the occupied
runtime claim, including from another coordinator. Original mirrors/history stay
unchanged. Intent or claim loss is not a reset or replacement authorization.

22 focused tests and all336 collection tests pass. The process test kills three
actual child controllers: before intent publication, after intent publication, and
after claim publication. A fresh controller retains one intent and exact claim inode,
with no activation. Both focused/full runs pass this sequence. Tests also cover
changed application/session/pending/transport cohorts, competing requests, wrong
checkpoint, unknown formats, pending journal transitions, replaced files, display
replacement, expired leases and failed/lost publications. All four scopes inactive.
The first21-test trial had17 errors from a reference adapter leaking the reader's
internal maximum-size field into stored reference comparisons; fixed before the
passing21/22/336 runs. Rejection tests now assert they reach the cohort observer.
All trial sources/logs and populated original records are retained.

M317 supersedes M316's absent successor-intent/exclusive-claim contract only.
The plugin cohort adapter here is synthetic; native Theme/DMs cohort validation
for this path remains next. M316's actual-plugin checkpoint sources are unchanged.
This is not successor admission, a complete cross-session bootstrap, recovery from
a second runtime loss, supported logout/reboot/keyring unlock, installed migration
or public profile activation. Versioned successor descriptor/admission and compatible
immutable generations must be qualified before constructing any plugin. No
production resources changed or private sessions/keyrings exported.

Evidence: /home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-25-desktop-successor-ownership
