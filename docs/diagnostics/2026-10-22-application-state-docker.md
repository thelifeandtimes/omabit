# Persistent application state and subscriptions — Docker, 2026-10-22

**Passed on fresh guest run16 / appliance15.** Both new fakes ran in the guest's
own rootless Docker29.9.1, using pinned Vere4.6-5a7614f and the frozen pill. The
no-NIC VM used 7 GiB RAM and a 12 GiB/no-swap/one-core/128-task outer scope. It
published no host ports and mounted no user piers, production engine or desktop.

Four unique Gall peer writes were acknowledged and read back in order. After the
second write was queued while the client was disconnected, nec stopped cleanly
and resumed from the same pier in a different container. The original session,
channel and event cursor delivered that queued revision2 update; a new write
arrived as revision3. A fresh channel reconciled revision3 and delivered revision4.
All 13 state observations were independently checked against the exact four writes.

The six subscription snapshots were `[0, 1, 2, 3, 3, 4]`; cursors were
`[1, 2, 3, 4, 1, 2]`, with the last two on the fresh channel. Both channels were
deleted. No write was replayed. The existing seven transport outcomes, cross-ship
cookie refusal, unchanged pier/namespace and original-session checks also passed.
Clean stops were nec for restart, then nec/zod for final cleanup. The VM powered
off normally after 1450.33 seconds, with no timeout, log overflow, quota problem
or cleanup failure. Its exact transient scope was verified inactive.

[Recorded evidence](2026-10-22-application-state-docker.json) includes frozen source
hashes, all observations and independent verification. Original backing images,
overlay, extracted receipts and metadata are retained with hashes under
`/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-22-docker-state16/`.
Only the verified regenerable inspection conversion was removed. The result hash is
`6c88e46f3e2d63fa599413371ed294c73f07415c17b58ec8789bd2d92d46733a`.

This extends the native control and earlier Docker transport checkpoint. It proves
clean same-pier process/container replacement, state, session and subscription
recovery. It does not prove power loss, Gall agent upgrades, real Theme/DMs writes
or reconnect, two simultaneous projects, anchor recovery, published endpoints or
runner-managed networking. The newer project-isolation source was not in this
frozen appliance and needs its own runtime result. All earlier failed trials keep
their original scope and retained evidence.
