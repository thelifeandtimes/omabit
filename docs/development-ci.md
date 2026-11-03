# Development feedback and release CI

Daily local feedback uses [named check lanes and cached builds](desktop-development.md#daily-development).
Full release checks remain mandatory for promotion; no path filter can silently
skip them. Local checks never install a candidate on an operational fleet.

The Checks workflow runs topic/release changes through their pull request, without
a duplicate release-branch push run. Pushes to master, edge and develop retain
checks. Manual dispatch can validate an unpublished topic branch. A newer commit
cancels only the superseded run for the same PR/ref.

## Parallel release gates

- `core`: complete ordinary Go tests/vet, offline fixtures, binaries, desktop models,
  installer/bootstrap/tooling tests and independent release reproducibility.
- `transport-race`: the existing stateful transport/runtime race package list.
- `managed-race`: eight complete, disjoint subsets of the compiled managed-test
  inventory, each with the race detector and test-result caching disabled.
- `source`: always runs and succeeds only when all three groups succeeded.
  Failure, cancellation and skipping all fail this aggregate.
- `qml-offscreen`: actual Qt binding and recovery interactions.
- `release-policy`: independently validates stable promotions using trusted base
  code. The Checks workflow does not replace this protected-branch gate.

Managed tests are assigned longest-estimated-first to the least-loaded shard using
`scripts/managed-race-durations.json`. Its source run is recorded in the file.
Unknown tests receive an estimate and still execute; removed test timings are
ignored. The compiled inventory must match declared source and every test must
appear exactly once. Timing data affects scheduling, never inclusion.

Inspect the actual assignment without executing the tests:

```sh
python3 -B scripts/ci-go-tests.py --managed-plan 8
```

Refresh measurements from top-level Go JSON pass/fail events after major test
changes. Do not count nested subtests again. Each job adds its slowest declared
public test names to the GitHub step summary; complete JSON timings remain in the
job log. Keep unknown runtime values out of public summaries.

The historical 0.3.5 run spent about 31 minutes in its slowest four-way managed
shard and ran transport race checks after ordinary tests in `core`. The new
schedule balances those tests and separates the sequential jobs. Actual completion
time still depends on runner availability, test growth and machine performance;
adding workers does not make the underlying tests cheaper.

Live Docker/Omarchy, fleet acceptance and supported upgrade rehearsals remain
separate gates. Offline CI success does not establish pier safety for an untested
adoption, runtime replacement or restore.
