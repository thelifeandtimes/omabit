# Idle CPU and battery optimization pass

Review deadline: 2026-11-02 06:00 America/Los_Angeles (14:00 UTC).
Started: 2026-11-01 20:50 UTC. Baseline: stable 0.3.4 distribution 0744fe6.
Topic: perf/idle-observation-20261101. No release or production deployment yet.

The user reported 0.3.2 shell/SSH CPU overhead; their percentages are supplied
measurements, not independently reproduced results. No UI/layout/workflow changes
are requested. Maintain prompt visible updates, action outcomes, identity pins,
revocation, mutation non-replay, freshness limits and bounded failure recovery.

## Ranked investigation queue

1. Inventory transport fan-out and idle pool eviction: reproduce all three current
   request streams, then retain bounded authenticated connections across polls.
2. Visibility-aware inventory cadence: fast while panel/monitor/action is active,
   slower hidden, with immediate wakeup and no collector restart on visibility.
3. Bootstrap resolve process churn: replace repeated interpreter startup with a
   bounded long-lived observer without weakening archive verification/activation.
4. Bootstrap activation timers: no 250 ms readiness polling in stable ready state.
5. Hidden QML clock/binding work: avoid updating hidden detail trees every second;
   preserve fresh action checks and visible status timing.
6. Duplicate capability/identity RPCs: reuse an observation's verified Hello only
   within its request scope, preserving per-connection pins and response checks.
7. Host compatibility polling: honor existing report expiry and avoid repeated
   transfer/decoding before expiry; invalidate on role/identity/feature changes.
8. Hidden chart/delegate reconstruction: measure JS/sample/paint and model churn;
   retain samples and visual behavior while avoiding no-op work.
9. Runner observation overhead: profile empty/stopped/running inventories and
   assess safe reductions in repeated filesystem/engine/database work.
10. Idle/failure/background helper lifetimes: inspect log watchers, retry pacing,
    timer allocation, wakeups and cancellation; battery awareness only if it
    demonstrably helps beyond visibility and has a reliable optional signal.

This is an investigation order, not a claim that ten changes are warranted.
Record rejected approaches and measured limits as well as improvements.

## Isolation and storage

No protected ships, ~/temp-ur, production services/plugins, host SSH/Docker or
manual DEV state are used as fixtures. Synthetic tests and fresh offline labs only.
The broader platform goal remains paused; this is the separately requested pass.
Storage at start: DEV 5,874,765,824; labs 19,072,114,688; evidence 8,092,348,416
allocated bytes visible. Add the existing 4 GiB reserve for 18 unreadable rootless
DEV overlay work directories. Combined charged usage about 34.77 GiB / 40 GiB.
Initial synthetic-test/evidence allowance: 512 MiB; measure before any larger lab.
Retain compact measurements and do not duplicate VM bases or production state.

## Results

### First pass — 2026-11-01 21:15 UTC

- A ten-poll three-report fixture reproduced 25 opened transports / 85 RPCs on
  the original implementation. Retaining at most three transports and sharing
  only the current observation's verified Hello reduced this to 3 / 43. Ten
  repeated pool checks stayed bounded; cancellation closes every connection.
  These are Unix HTTP fixture measurements of the pool, not SSH CPU measurements.
- Inventory watches now accept bounded stdin cadence controls: 2 s while the
  panel/monitor is open or work is pending, 30 s when hidden. Opening requests an
  immediate observation without restarting the collector or replacing its pins.
  Existing CLI callers without the optional control flag retain their behavior.
- The bootstrap's existing lease process now checks selection every three seconds;
  it avoids starting another Python interpreter for every check. Forty synthetic
  checks used 1.742204 CPU seconds with subprocess resolves and 0.014401 with the
  retained interpreter, with 40 additional process launches reduced to zero.
  State/pins are still reread and validated; archive checks remain unchanged.
- The bootstrap's 250 ms activation timer stops in the ready state, and its lease
  restart timer runs only when the lease is absent. The hidden application clock
  no longer invalidates QML bindings every second; incoming observations and
  opening a view update time immediately, and visible views keep their clock.
- The first offscreen cadence test caught QML signal ordering: active changed
  before the interval binding. Sending controls from intervalChanged fixed that.
  A host-fixture namespace typo was also caught and fixed before qualification.

Passing checks: client/collection/access/collector race suites; CLI control EOF,
invalid input and cancellation; bootstrap unit and actual offscreen activation,
rollback and restart; native plugin registry; panel model; RC11 offscreen controls;
host compatibility controls; actual Service/Panel/Monitor private namespace drain
and replacement across stable/dev/edge. The second actual-service run additionally
verified 30 s -> 2 s -> 30 s for both panel and monitor with the same collector PID.
Evidence: `/tmp/omabit-service-drain-performance-02/result.json` (inert helpers;
no real compositor/runner). The full `TZ=UTC go test -timeout=30m ./...` run passed, including managed-state
coverage (263.124 s). Service-upgrade offscreen controls also passed.

The generated-key real OpenSSH benchmark also passed. In 30-second visible
runs, the released 0.3.4 collector opened 47 SSH sessions for 15 observations;
the candidate opened 3 for the same 15 observations. At hidden cadence it kept
those 3 sessions for two observations across 35 seconds. An explicit wakeup took
12.94 ms and opened no new SSH session. Child CPU including enrollment/teardown
was 0.906081 s baseline versus 0.118449 s candidate-visible in this run. The RPC
server is synthetic, with a fixed 10 ms report overlap; these are transport
measurements, not actual runner/desktop/power measurements. Exact counts and
binary hashes: [SSH measurement](2026-11-01-inventory-ssh.json).

The benchmark's first attempts exposed two fixture-only issues: copied-script
path depth and an unnecessary loopback-up command. Bubblewrap already provides
private loopback; the passing experiment requests no extra capabilities. No
production network or credentials were mounted.

No laptop CPU percentage or battery improvement is claimed. Remaining
host/report/model/runner profiling and upgrade/review packaging are pending.

### Continuation

Same-thread heartbeat `omabit-performance-passes-through-monday-review` runs at
:00/:20/:40 through 2026-11-02 14:00 UTC. Keep this topic checkout and sequential
ownership. Check time at every pass, stop new implementation before the deadline,
finish evidence/review notes, then pause the heartbeat. Do not resume the older
paused platform goal or deploy/publish this candidate.

