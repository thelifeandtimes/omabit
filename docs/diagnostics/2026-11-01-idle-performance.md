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

### Second pass — 2026-11-01 21:23 UTC

Unchanged sample histories now retain their map identity, avoiding QML binding
notifications for empty/stopped/duplicate observations. The last sample is found
by a backwards scan instead of allocating a filtered array on every poll. Changes
still create a fresh map; container replacement, unavailable-data gaps and the
120-sample/256-instance limits retain their previous behavior.

`node scripts/test-panel-model.js` and the actual RC11 offscreen QML checks pass.
`node scripts/performance/model-samples.js BASELINE CURRENT` measured 2,000
unchanged-history iterations over 128 synthetic ships: 509.681 -> 246.655 ms CPU,
and 2,000 -> 0 map replacements. Empty fleets: 9.960 -> 5.748 ms for 20,000
iterations, 20,000 -> 0 replacements. Stopped fleets: 97.345 -> 101.337 ms (no
CPU improvement claimed), 2,000 -> 0 replacements. This is a Node microbenchmark,
not a shell/frame-rate measurement. Full data is in the adjacent JSON evidence.

First source checkpoint: `0ba2f58`. Full Go tests and the SSH benchmark preceded
this JS-only second pass. The final SSH run also proved a hidden-to-visible
refresh in 12.94 ms without an additional handshake. Failed/short-run binary
copies in `/tmp/omabit-ssh-performance-01`, `02`, `03` were removed only after
recording hashes and paths in each `removed-artifacts.json`; compact logs/results
and the final run remain. No production state changed.

### Remaining investigations / next pass

- Target 7: retain existing runner-side caching. Native metadata is cached for
  up to 30 s and live engine evidence for 5 s, with single-flight work and bounded
  failures. An extra collector cache would skip no hidden 30 s observations;
  its small visible-poll saving does not currently justify more reconnect,
  actor/role/capability and expiry state. No freshness limits were extended.
- Target 8: sample-map work is improved. Measure actual hidden Qt binding/paint
  work before adding other visibility gates; avoid destroying view state.
- Target 9: snapshot journal scans improved (third pass below). Profile scheduler
  work separately if justified; current runner polling persists observation
  timestamps/revisions, so do not cache/drop observations or skip runtime safety
  checks. Full managed regression at the next artifact checkpoint must include
  the final row-ID-only query refinement.
- Target 10: log readers are already visibility-owned and upgrade drain tests
  verify shutdown. Inspect idle helpers/retries and use a bounded soak/leak test.
  Keep current SSH liveness deadlines unless measurement justifies changing them.
  A separate battery signal may add little beyond the 30 s hidden cadence; do not
  add a dependency or policy change without evidence.
- Negative SSH cases passed in a private offline namespace (details below).
  Prepare an isolated review bundle from a clean exact source commit and qualify
  the paired UI/collector upgrade after the remaining bounded checks.
- There are no active test processes at this checkpoint. Built tools are in
  `/tmp/omabit-performance-build-01`; the final SSH evidence is run `04` and the
  actual UI drain/cadence evidence is `/tmp/omabit-service-drain-performance-02`.

### Continuation

Same-thread heartbeat `omabit-performance-passes-through-monday-review` runs at
:00/:20/:40 through 2026-11-02 14:00 UTC. Keep this topic checkout and sequential
ownership. Check time at every pass, stop new implementation before the deadline,
finish evidence/review notes, then pause the heartbeat. Do not resume the older
paused platform goal or deploy/publish this candidate.


### Third pass — 2026-11-01 21:44 UTC

Actual OpenSSH negative regressions passed in a fresh offline user/network/PID/
mount namespace. The managed server used its fake runtime backend; no engine or
ship was accessed. Two operator keys, observer restrictions, actor-scoped replay,
reconnect, host/runner identity mismatches, forwarding/arbitrary command refusal
and live revocation of a retained session all passed. Harness:
`scripts/performance/test-openssh-isolated.py`; evidence:
`/tmp/omabit-ssh-regression-01/result.json` and `test.log`.

The snapshot profile found one full epoch-journal scan per fleet member. The
reader now ranks operation row IDs in batches within the existing transaction,
then reads only selected runtime payloads. It retains per-instance 128-record
limits, newest-first ordering, current-runtime binding, validation and immutable
stored state. One-ship fleets retain the original query. No schema, observation
persistence or authorization changes were made.

Three-run median synthetic snapshot times with 2,048 non-roll journal records:
32 ships, 44.77 -> 3.82 ms; 128 ships, 171.91 -> 11.13 ms. With 2,048 completed
rolls: 32 ships, 66.70 -> 25.05 ms; 128 ships, 215.56 -> 30.20 ms. The larger
32-ship/8,192-record retained-log fixture improved from 144.92 -> 74.35 ms. Empty
and one-ship fleets were effectively unchanged; small differences are not claimed
as improvements. Roll-heavy cases allocate somewhat more (for example about
4.73 -> 4.80 MB at 128 ships), so this is primarily a query-time improvement.
Exact measurements, limitations and final source hashes are in
[the snapshot evidence](2026-11-01-snapshot-performance.json). These are Go
benchmark elapsed times, not full-runner CPU or laptop battery measurements.

Regression coverage compares batched and independent reads, replaced runtimes,
duplicated/invalid/stale evidence, per-instance limits before validation,
malformed selected runtime rejection, excluded malformed runtime preservation,
513-member batch boundaries, empty histories and non-persistence of decoration.
The full managed suite passed (261.496 s) before the final row-ID-only query
refinement. Focused epoch/roll/snapshot tests under the race detector passed on
that final query (103.873 s), as did go vet. The next full-suite/artifact checkpoint
must use the final source; this is not a released or deployed runner.

This pass's explicitly named retained `/tmp` fixtures/builds total 221,984 KiB
(216.8 MiB), below the 512 MiB scratch allowance; database benchmark temp files
were removed by the test framework. No VM, production plugin/service or human DEV
state was created or changed.

Checkpoint state: no test or runtime experiment remains active. Current source
changes are ready for the third performance commit. Continue with the bounded
watcher/Qt soak and actual idle Qt measurement, then exact-source paired artifact
qualification; no deployment or publication. A separate battery signal is still
unjustified after hidden 30 s polling and would add another dependency/observer.


### Fourth checkpoint — final measurement and review stop (2026-11-01)

Product source is `4584453`. The final full `TZ=UTC go test -timeout=30m ./...`
suite passed, including managed tests in 262.616 s. This supersedes the third
pass's pending final-query full-suite check. No product changes followed it.

The actual Service/Panel/Monitor QML was measured offscreen in a private offline
namespace, with an inert collector and zero or 32 stopped synthetic ships. One
paired run measured main-thread CPU via schedstat over 32 hidden seconds:

| Fleet | Baseline CPU seconds | Candidate CPU seconds |
| --- | ---: | ---: |
| Empty | 0.04598 | 0.00349 |
| 32 stopped ships | 1.00338 | 0.06070 |

Hidden clock changes fell from 32 to 1 and observations from 16 to 1. Unchanged
sample-map notifications fell from 16 to zero. Across eight visible seconds,
both variants observed four times: empty-fleet CPU 0.01019 -> 0.01049 s;
32-ship CPU 0.25778 -> 0.24328 s. No material visible CPU improvement is claimed.
These are component results, not whole-shell CPU or battery-life improvements.

Fifty panel/monitor open-close cycles retained one collector PID, returned panel
owners to zero, kept 31 file descriptors, and drained cleanly. RSS changed from
178,212 to 179,456 KiB; this bounded run is not proof against long-duration leaks.
The reproducible harness is `scripts/performance/qt-idle.py`; compact evidence is
[Qt idle results](2026-11-01-qt-idle.json). The first smoke fixture lacked its font
mountpoint; it was fixed before the passing smoke and final measurements.

The user now permits stopping when a further loop produces less than 10%
improvement, and requests a mobile HTML review over their tailnet. This
supersedes the earlier requirement to keep iterating until Monday morning.
The final loop added measurements, not another optimization, and the remaining
cache/battery-observer ideas have no demonstrated additional >=10% gain. Stop
new implementation here; do not misrepresent cumulative gains as a new loop's
improvement. Target 7 keeps existing caches; target 10 retains liveness and
authorization checks. Targets 8/10 now have actual QML measurement/churn evidence.

Retained named scratch grows by 14,200 KiB for Qt fixtures/baseline copies to
236,184 KiB (230.7 MiB), under the 512 MiB allowance. Combined charged usage is
approximately 35.0 GiB including the 4 GiB unreadable-DEV reserve, under 40 GiB.
The starting combined audit is an accounting baseline, not a new traversal of
protected directories. Report assets add only a small amount of static text.

Review scope: source candidate and isolated evidence only. A new paired
UI/collector/runner release build, actual installer upgrade rehearsal and
real-laptop CPU/power measurements remain open. Schema 47, desktop schema 1 and
bootstrap API 1 are unchanged. Do not install, deploy or publish this candidate.
Production services/plugins, protected ships, ~/temp-ur and human DEV review
state remain untouched. The older platform goal remains paused. Tailnet serving
of the sanitized report is specifically authorized by the latest user message.


Review hosting verified at 22:04 UTC: mobile page
`2026-11-01-performance-review.html`, served through a new private Tailscale
path to a loopback-only single-file process. Existing Serve routes remain
identical; no Funnel enabled. HTTPS bytes match the report; supporting files
return 404. Browser validation at 390 x 844, including the expanded table,
has no document overflow. Hosting metadata is local qualification evidence,
performance-2026-11-01/mobile-review/verification.json. The performance heartbeat
was deleted to stop further scheduled work under the new stopping instruction.
No active test remains; only the explicitly requested static report server runs.
