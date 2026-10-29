# Recovery reactivation exposes passive Theme re-publication

Trial02's generated passed=true result is rejected by independent audit. Original
supervised recovery succeeds, and new dev/stable subscriptions become ready, but
returning to stable resolves the old Theme intent by authenticated read and then
opens another poke with the identical desired-value digest. The synthetic server
raises its duplicate-write assertion in an HTTP worker, while the main fixture
incorrectly continues and reports success. The old intent is state-confirmed and
a new identical intent is uncertain in the actual final transport journal.

Source cause: Helper.sync treats saved pending=true as reason to invoke publish;
publish deliberately re-notifies matching entries for explicit Publish Now, but
that same path is used by passive automatic recovery. Distinguish passive repair
from a new explicit publication. Preserve explicit Publish Now behavior, prevent
passive duplicate writes after authoritative readback, and keep mismatched unknown
writes fenced. The HTTP fixture must collect worker failures and fail the entire
run instead of relying on accepted-poke counts after an assertion.

Trial01 fails earlier on a private keyring startup race: secret-tool can activate
a second locked daemon before the manually launched daemon acquires its bus name.
Wait for and verify exact manual daemon ownership before first Secret Service use.
An attempted wait patch failed syntax validation before source write; trial02 used
the earlier startup source. Both trials/scopes are retained/inactive. No production
ships, desktop services, plugins or credentials changed.

M283 remains qualified through its explicit checkpoint-appearance boundary. This
finding prevents extending its claim to passive reactivation/return until fixed.
Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-recovery-resume-finding`.
