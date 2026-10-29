# Recovered activation and original-profile return

M284 is fixed in development Theme: publish distinguishes passive reconciliation
from a new explicit Publish Now. Passive retries skip entries already matching an
authenticated read; an unknown nonmatching write stays fenced. Explicit Publish Now
still sends both entries to notify subscribers. Partial retry now writes only the
entry still needing change. No transport/state format changes are needed.

287 private Theme tests pass, including three new restart/deduplication regressions
and preserved explicit re-notification. One prior suite failed in the historical
migration fixture: an unreleased copied owner was in another PID namespace, so the
production guard correctly refused it. The test now explicitly verifies unknown
ownership refusal and models a later boot for migration, without editing original
writer/fixture bytes or weakening runtime ownership checks. Both suites are retained.

The corrected HTTP fixture collects worker errors as fatal and verifies its manually
unlocked keyring daemon owns the private bus name before first use. With frozen old
Theme source the same harness now fails on the duplicate poke. With only main.py and
eyre.py changed, the full actual-plugin scenario passes: stable/dev/stable and edge
rollback, six ordinary coordinator kills, owner loss with two pending app writes,
failed deletion, two recovery coordinator kills, exact cleanup, candidate activation
and return to the original profile. Four guardians close; all21 channels delete.

Dev and returned stable each have one positive Theme subscription plus four positive
DMs subscriptions, bound to new loaded instances. Original session fingerprints are
unchanged on return. The unresolved DM remains visible and its file is byte-identical.
The original Theme intent becomes state-confirmed through normal authenticated
readback, without another matching poke or a fabricated ACK. The final explicit
teardown remains pending checkpoint-appearance after cleanly unloading the test set.
Independent source, journal, channel, guardian and receipt checks pass; all scopes
are inactive. Raw keyring/account files are excluded from exported evidence.

This supersedes M284's passive replay/false-positive condition and M283's untested
post-recovery activation only within the private synthetic-Eyre fixture. Native
loader identity/admission, external desktop-service accounting, broader account/
effect cohorts, actual compositor behavior and production deployment remain open.
No installed plugin, ship, host desktop, production credential, tmux or temp-ur
resource was changed. The same-user cooperative locks are not wallet key isolation.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-recovery-resume`.
