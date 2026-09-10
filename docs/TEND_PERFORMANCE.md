# Tend performance and resource budgets

These are release gates, not marketing targets. They provide deterministic
ceilings that fail CI when a change makes the desktop model or transport grow
without bound.

| Surface | Current gate |
| --- | --- |
| Eyre JSON response | At most 32 MiB |
| One Eyre SSE line/event | At most 32 MiB |
| One frontend action | At most 1 MiB |
| One batch reminder selection | 1–500 IDs |
| 10,000-reminder snapshot normalization | Under 5 seconds |
| Today query over 10,000 reminders | Under 5 seconds |
| Text search over 10,000 reminders | Under 5 seconds |

The JavaScript timing ceiling is intentionally generous for slow CI and is not
the interaction target. A release-candidate desktop pass should still confirm
that warm list switching, Today, and search feel immediate on supported
hardware. The scale fixture also asserts that its 10,000-reminder snapshot fits
inside the transport ceiling.

Run the deterministic scale gate with:

```sh
make test-js
```

Multi-ship throughput, reconnect soak duration, and Gall-state growth budgets
remain gated on the peer transport and its explicit data-egress approval.
