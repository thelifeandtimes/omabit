# Explicit recovery continuation source gate

Schema40 now persists an actor/request-owned continuation of an interrupted
coordinated recovery. It binds the exact parent checkpoint and history, retains
prior interruption and attribution, and commits parent/member/continuation
progress atomically. Restart reconciliation interrupts pending work without
replaying runtime actions. Unknown Create outcomes remain refused; a returned
exact Create ID has a separate Start-only path.

Focused managed race tests pass in 861.164 seconds, followed by vet and diff
checks. Coverage includes restart at queued/begun/dispatched boundaries,
transaction rollback, corrupted or stale mirrors, ordinary-callback bypass,
unknown Create and inert replay. This supersedes M174's pending focused race
status only; the full schema40 regression is running.

A guarded disposable-VM fixture now injects lost replies after actual Stop,
Create and Start, reopens the SQLite store between attempts, and requires three
explicit continuations plus two read-only observations. It expects exact effect
counts and the same sixteen-write/four-subscription-cycle application check.
The fixture compiles, its 44 Python harness checks pass, and an explicit host
invocation is refused before runtime access. It has not run in a VM. Database
reopening is not an OS runner-service restart.

VM32 remains frozen schema39 and cannot qualify this continuation. Public routes,
CLI/UI integration and the roadmap remain incomplete. No production operation or
installed plugin change occurred.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-recovery-continuation-source`.
