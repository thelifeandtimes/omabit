# Duplicate-identity Docker projects — 2026-10-22

**Passed on fresh guest run17 / appliance16.** Two projects each ran new fake
`~zod` and `~nec` in separate anchor namespaces. All four ships were live during
the comparisons. Each namespace had only loopback and no published ports. The
no-NIC guest used 14 GiB RAM/two exposed CPUs inside a 24 GiB/no-swap/one-core/
128-task scope, with its own rootless Docker29.9.1 and pinned Vere/pill.

Both projects passed the seven transport outcomes and clean same-pier receiver
container replacement with original sessions. Each passed four persistent writes,
13 recovery observations, and six subscription snapshots `[0, 1, 2, 3, 3, 4]`.
The old channel delivered the queued update after restart; the new channel
reconciled current state and delivered a subsequent write. Both channels were
deleted, and no write was replayed.

Cross-project checks rejected all four same-identity cookies with HTTP401.
A fifth unique peer write in A changed only A; a fifth in B changed only B.
The six interleaved reads were A4/B4/A5/B4/B5/A5 with disjoint entry lists.
Each final project report contains five writes and 17 state observations.
The frozen case asserts four-container liveness and exact namespace membership
before comparisons; its saved comparison record does not duplicate raw inspect
responses, so this specific assertion is source-bound evidence.

Both projects stopped nec for recovery, then nec/zod for final cleanup. The VM
powered off cleanly in 2744.57 seconds with no timeout, quota/log limit or cleanup
failure. Independent verification matched all original artifact/overlay hashes,
all 17 extracted files, final reports, exact write/state histories, subscription
cursors and the inactive scope. [Evidence](2026-10-22-docker-project-isolation.json)
and originals are retained under
`/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-22-docker-project17/`.
The main result SHA256 is
`d68b1b619d30530ad60adb954e2588db7a4b4901f58849ba1916f834bdba90a0`.

This supersedes the pending duplicate-project runtime status in M015 and the
newer-source limitation of M019. It does not establish published endpoints,
anchor recovery, runner integration, arbitrary hostile-traffic filtering or
multi-host networks. Ships within a project remain trusted peers sharing an
administrative namespace. The next separate fixture tests publication and anchor
lifecycle; no production fleet or installed plugin was accessed.
