# DMs drains when its shell closes the input pipe

The development DMs bridge now calls its existing transport drain from a finally
block when input reaches EOF or output breaks. It waits for admitted workers,
deletes the exact owned channel, joins listeners and preserves unresolved app
intent. A successful explicit shutdown is not repeated. Unconfirmed final cleanup
returns a nonzero exit and a generic error, without emitting an orphaned stdout
receipt. This does not add a persistent transport format or clear collection fences.

All118 DMs checks pass in a private no-network namespace, including six new EOF,
broken-pipe, failed-close and explicit-close cases. The actual private shell-loss
scenario now observes deletion of all16 channels, including the DMs channel left
undeleted in M271. Both controlled shell replacements, six coordinator kills,
failed candidate/restore and rollback remain passing. Unexpected owner loss still
persists reader-owner-lost, and a new shell or exact private process cleanup cannot
clear it. The two surviving helpers are the fixture's inert Omabit inventory/log
readers; they were identified against retained process records and cleaned up.
They are not evidence of actual collector behavior.

M272 supersedes M271's undeleted DMs channel for EOF-driven parent loss only.
SIGKILL of the helper itself, unavailable network/credentials and older channels
lost during unsuccessful reconnect still need durable plugin-owned transport
records and independently verified recovery. The coordinator must retain uncertainty
until that evidence exists. Current installed plugins and all production ships,
credentials and services remain unchanged. Source hashes, journal replay and exact
scope exit were independently verified.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-dms-eof-drain`.
