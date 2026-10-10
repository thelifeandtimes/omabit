# Container resource acceptance, September 25

The agent verified `container-resources` over the identity-pinned `workstation`
connection to `runner-f496feed1f7b08f31a5d82b94ac2f5fd`, then obtained two snapshots
with a twelve-second wait between requests. Both samples were fresh, from
`docker-stats-v1.47`, and less than thirty seconds old.

| Instance | Daemon sample time (UTC) | Raw memory bytes | Limit bytes | Tasks |
| --- | --- | --- | --- | --- |
| alpha | 18:55:40.274116253 | 43,540,480 | 4,294,967,296 | 6 |
| alpha | 18:55:50.279533458 | 43,618,304 | 4,294,967,296 | 6 |
| panel-check | 18:55:40.284527384 | 314,265,600 | 4,294,967,296 | 10 |
| panel-check | 18:55:50.282540673 | 314,294,272 | 4,294,967,296 | 10 |

The full container IDs remained stable across those observations:

- alpha: `c07d70e5a284f6f9236c79271fa19ac8f2d4e3a68d95b61344441eb815d59aa3`.
- panel-check: `59be6fba033c898e22b35a057390f78f5ab5c0335f528de27805b1a7a6433d89`.

Alpha remained running/healthy at configuration/applied revision 2/2; panel-check
remained running/healthy at 1/1. No incidents or recovery requirements were
reported. Uninitialized ssh-check had no resource sample. No lifecycle or
maintenance mutation was made during this validation.

These are raw container memory/task observations, not loom utilization or
process-only resident memory. Fixture validation separately covers unavailable
and stale data, invalid counters, transport faults, and lifecycle races.
