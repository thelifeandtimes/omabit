# Persistent application state and subscriptions — native control, 2026-10-22

**Passed on fresh native control18. Docker qualification remains pending.**
Two newly booted fakes used the pinned Vere/pill in a private Bubblewrap namespace
under an admitted 8 GiB/no-swap/one-core/128-task scope. No host services, Docker,
production files, desktop sockets, user piers or excluded directories were mounted.

A disposable Gall fixture on nec accepted four unique peer writes from zod. The
control independently verified 13 state observations:

1. Initial subscribe acknowledgement and revision0 snapshot.
2. First peer write acknowledged, read back, and delivered as revision1.
3. Client connection closed; a second write acknowledged and queued as revision2.
4. Nec stopped cleanly and resumed from the same pier, with the original session.
5. State read back at revision2; reconnecting the original channel/cursor delivered
   the queued update. A new write then arrived through that subscription as revision3.
6. A fresh channel received the full revision3 snapshot; a fourth write arrived as
   revision4. Final state contained exactly the four original values in order.

Subscription revisions were `[0, 1, 2, 3, 3, 4]`: the repeated 3 is deliberate full
state reconciliation on a new subscription, not a replayed mutation. All writes
had unique bound Gall acknowledgement receipts. Both channels were explicitly
deleted. All seven original transport outcomes also passed. Clean stops were nec
for the planned restart, then nec/zod for final cleanup. No log overflow or cleanup
error occurred. Eleven focused offline tests pass.

The independent result hash is
`9226e485b5cc34e343f69a97a0a19c0ff478cae13201ea4e2e62034d694891d6`.
The accompanying JSON binds frozen input/source hashes and all observations.
Original evidence is retained under
`/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-22-state-recovery/native18/`;
the original /tmp directory is a compatibility link after verified relocation.

This qualifies a clean process restart and application/channel/session recovery on
the pinned native runtime. It does not qualify Docker/container replacement, power
loss, agent upgrades, production Theme/DMs reconnect behavior, two projects with
identical identities, or runner-managed project networking. Prior failed controls
remain retained; none is relabeled as passing.
