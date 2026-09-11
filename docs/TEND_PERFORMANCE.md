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
| Hosted lists / sections per list / reminders per list | 10,000 / 10,000 / 100,000 |
| Reminder title / notes / URL | 1 KiB / 64 KiB / 8 KiB |
| Early offsets / weekdays / month dates / snooze presets | 64 / 7 / 31 / 32 |
| 10,000-reminder snapshot normalization | Under 5 seconds |
| Today query over 10,000 reminders | Under 5 seconds |
| Text search over 10,000 reminders | Under 5 seconds |
| Pending remote operations per participant ship | At most 1,000 |
| Automated Ames-held fault burst | 12 concurrent submissions by default |
| Local restart/resubscription soak | 2 participant/owner cycles by default |

The JavaScript timing ceiling is intentionally generous for slow CI and is not
the interaction target. A release-candidate desktop pass should still confirm
that warm list switching, Today, and search feel immediate on supported
hardware. The scale fixture also asserts that its 10,000-reminder snapshot fits
inside the transport ceiling.

Run the deterministic scale gate with:

```sh
make test-js
```

The three-ship harness gates owner sequencing under an Ames-held burst and
repeated Gall restarts. Twelve submissions and two cycles are a deterministic
local regression floor, not a production throughput claim. Longer real-network
soaks and representative hardware latency measurements remain release-candidate
work. The durable operation-receipt map retains the newest 4,096 results.
