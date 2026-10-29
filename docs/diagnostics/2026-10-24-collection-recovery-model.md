# Explicit collection recovery intent and verification

Collection format2 adds recovery-prepared and recovery-verified events. Preparation
binds the pending old activation, exact uncertainty cohort, original supervisor
identity/root/stop request, saved runtime binding digest, and all three members'
application/pending/credential/channel cohorts. Verification requires that same
plan and original guardian closure plus unchanged application intent, credentials
and channel IDs. Transport journal digests may change as exact cleanup completes.
The event preserves old problem/partial-drain history and advances only to the
existing disable-previous phase. It creates no ordinary application ACKs.

The internal recover_step driver accepts observations from a bound adapter, never
a user-supplied proof object. Its first call persists preparation before permitting
any cleanup. Later calls re-sync the retained prepared event before effects, even
if a prior rename succeeded but directory fsync or its reply failed. Verification
replay also re-syncs without calling the adapter. Busy, mismatched state, lost
observations and unexpected failures retain the fence. The first supported phase
is drain-previous; other phases and unsupervised ownership remain unsupported.

48 private tests pass (30 existing,18 new). They cover original writer reproduction,
read-only format1 opening, history-preserving migration, exact replay, downgrade
refusal, corrupt version relabeling, wrong activation/guardian/state/credentials,
late fences, no effect before durable intent and two actual coordinator SIGKILLs:
after prepared publication and after a modeled cleanup effect before its receipt.
These are model observations, not proof of plugin or kernel closure.

Trial01 ran no tests because standard discovery skips hyphenated filenames; the
wrapper now loads the explicit files and requires a positive count. Trial02 found
a stale closed Store reference in a test after reopen. The test was corrected and
a real guard was added: a closed store now refuses before any relative open rather
than allowing dir_fd=None to fall back to the current directory. All trials are
retained, scopes inactive, final source hashes verified. No production state changed.

Opening format1 never rewrites it. The next new event writes format2 preserving all
old events; exact replay remains byte-identical. The actual M281 format1 writer and
populated runtime journal are pinned in tests/fixtures/collection-journal-v1. That
old reader refuses real format2. Never lower the marker or erase recovery history.

Next bind the model to plugin-owned cleanup and original supervisor evidence in
the private actual-QML fixture, including repeated lost coordinator replies. The
adapter must exclude competing plugin work while freezing/revalidating state; the
reducer does not establish that authority. Native loader/external-service handling,
changed-cohort amendment and lost guardian recovery remain separate gates.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-collection-recovery-model`.
