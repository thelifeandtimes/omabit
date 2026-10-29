# Queued current-context lifecycle and sender recovery

VM26/schema37 independently passes queued Stop, exit observation, container
retirement and same-pier resume for the original sender after two receiver
replacements. The receiver's exact process remains running throughout the sender
replacement. Both members receive fresh current-readiness evidence, and both final
Stops run through private queues with the exact recorded peer state.

Twelve remote writes, three queued-update/subscription recovery cycles, six
channels and eighteen matched subscription snapshots use the original two Eyre
sessions. The third cycle replaces the sender itself. No later login or uncertain
write replay occurs. All original readiness and lifecycle history, operation
mirrors, request replay and store reopening agree with the extracted database.

Outer execution and QEMU exit0, clean poweroff and an inactive exact scope are
verified (2123.97s). Both final pier states and private engine container
inventory match the offline disk. An initial verifier used a nonexistent resume
exit-time field; it now checks the actual receipt fields and the inspected running
container's zero finish time. That failed verifier and a sandbox scope-check refusal
are retained; neither changed guest evidence or product validation. Original overlay/backing artifacts and hashed
metadata remain at `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-23-current-context-lifecycle26`. Only the verified regenerable raw conversion was removed.

This supersedes M131/M133's pending schema37 lifecycle runtime gate. It does not
qualify M134's later published-client transport or M136's anchor observer, which
were not in this frozen appliance. Coordinated anchor recovery, public lifecycle
APIs and the remaining roadmap are still separate work. Production ships, installed
plugins/services, user tmux ships and `~/temp-ur` were not accessed.
