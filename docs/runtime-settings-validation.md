# Runtime settings validation

This document records the contributor's pre-integration observations. The final
RC.10/schema19 qualification is recorded [separately](diagnostics/2026-10-18-rc10-qualification.md)
and in the published release's `qualification.json`.

## Scope

Omabit supports loom exponents 20–34 and explicit native-control opt-in for all ship
classes. The default remains loom 31 with native control off. These settings do not
change an existing launch until explicit replacement.

This record distinguishes source tests, real managed-fake execution and management
updates. It does not claim native-socket execution on every livenet ship class,
whole-application compatibility or broad security qualification.

## Source verification

Executed checks:

- `go test ./... -count=1`.
- `go vet ./...` and builds of all three command binaries.
- Race tests for `internal/managed`, `internal/runtimequalify` and the collector.
- Existing and runtime-settings-specific JavaScript model suites.
- Real QML fixtures through both direct and native entry-point wrappers.
- Exact agreement between the CLI warning and the UI warning text.

All changed packages pass. The full Go suite retains a pre-existing failure in
`internal/pierstore`, `TestHostPathAllowsOnlyTheRunnersNarrowACLGrant`:
`unrelated ACL writer accepted`. It also fails in an untouched base-checkout test
on the same host. The patch does not change that test or its production path.

Tests cover the supported loom range, rejected input, exact emitted runtime
arguments, and default-off and explicit-on behavior. They also cover durable
settings, stop/start, startup/routine propagation, desired versus recorded values
and containment drift.
Synthetic runtime fixtures cover comet, moon, planet, star and galaxy contracts.
No real identity key is used by those tests.

Schema18 tests verify additive migration from supported prior schemas, unchanged
instance and receipt bodies, repeat open, preserved request replay and unsafe
version refusal. Compatibility tests distinguish `runtime-settings` from
`native-control-all-ships`, including refusal before key reads or pill uploads.

Physical negative controls separately restored hardcoded loom 31 and retained
`--no-conn` during explicit native opt-in. Both compiled and failed the intended
assertions. A separate recorded-native utility guard mutation also failed its
intended assertion. Setup and compilation errors were not counted as guard kills.

## Real managed-fake execution

An earlier fake-only iteration was exercised through Omabit on fresh disposable
fakes. This is not an exact-final-candidate livenet result.

The test used pinned Vere `4.6-5a7614f` and the existing content-hashed supplied-pill
path. No runtime or kernel patch was used.

Verified observations:

- The king and worker ran with loom 34. The runtime reported a 16 GiB loom.
- The container retained its separate 4 GiB memory cap and existing confinement.
- Native-enabled pier directories and socket had private owner-only permissions.
- A fresh default-disabled fake retained loom 31, `--no-conn` and no socket.
- Host-side `click -b CODEC -k -i FILE PIER` worked without docking the pier.
- A genuine native thread returned `[0 %avow 0 %noun %omabit-native 0 42]`.
- Native desk creation/mount, ordinary owner file copy and native commit succeeded.
- A `%cx` scry returned the exact 20 source bytes before and after runtime replacement.
- Requested loom 32/native-off did not relabel the existing loom 34/native-on launch.
- Explicit Stop/Start applied the changed settings. A later replacement restored
  loom 34/native-on and preserved the committed Clay bytes.
- Exact Omabit Stop receipts succeeded. Recorded runtime containers and processes
  were gone afterward. Stopped piers and empty dedicated bridges were retained.

The codec came from the pinned fixture container's executable, not from a pier
copy or an arbitrary replacement runtime. Raw typed replies were checked rather
than relying on click's exit status.

## Management updates and UI

Two separately approved management-only updates exercised schema 16→17 and 17→18.
Running container/process identities, requested settings, revisions and operation
history remained unchanged. The second update also preserved stopped instances.
Docker and service definitions were not replaced or restarted by those updates.
These observations do not qualify every stable-release upgrade path.

The source-built UI was connected to the updated runner. The loom dropdown exposed
all 15 supported choices. A live moon's native-control checkbox became available
when the all-ship capability appeared, but remained off. No live-ship socket was
enabled or exercised as part of this UI check.

## Remaining boundaries

The pinned Vere release predates the known disconnect fix. A successful small
reply is not a guarantee against a client disconnect with replies pending.
The tested upstream click uses a receive-packet limit rather than a frame-aware
transport. Large replies, forced client termination and crash recovery need
separate qualification. Native control should remain off for routine operation.

Published desktop and binary pins are unchanged by this source contribution.
No automatic enabling, ship restart, pier reset, host-policy change or generic
remote-execution endpoint is introduced.

## 2026-11-12: configurable container resources (M573)

The new schema59 feature supersedes the fixed-cap statements above for individually
managed ships on `container-resources-v1` runners. Historical records still mean
4 GiB RAM / 2 CPUs. New selections allow 1–16 GiB and 1–16 whole cores; limits can
change only while stopped/uninitialized with no operation or recovery pending.
Stop reconciles an exited container; configuration saves the next launch without
replacing the pier or starting automatically. Loom and host admission policy are
independent. Private/project member containers retain their separate contracts.

Creation persists limits before dispatch. Inspect, cgroup probes, maintenance and
metrics use each immutable container record, including the old defaults when its
fields are absent. UI and CLI show requested and recorded values separately.
Nondefault limits are signed into handoffs; both endpoints must support them.
Explicitly restoring 4/2 keeps the historical handoff encoding so compatible older
destinations are not refused merely because defaults were explicitly saved.

Validation used private Bubblewrap namespaces, no host Docker or service sockets,
synthetic runner databases, mocked Docker, and inert QML callbacks:

- 15 runtime/containment/resource tests, 36 managed-store/lifecycle/CPU/handoff
  tests, 3 client transfer tests, 15 protocol tests and 6 CLI tests passed with
  `-race`. Evidence: `/tmp/omabit-fleet-resources-{runtime-final,managed-final,client,protocol,cli}`.
- Actual schema58 writer66c641b →59 migration/reopen and actual older-reader
  refusal passed in `/tmp/omabit-resources-upgrade-01`. The original SQL fixture
  and checksum are retained under `internal/managed/testdata/resources-schema58.*`.
  Final fixture tests also cover preserved receipts and migration rollback.
- `scripts/test-runtime-settings-model.js` passed. The real ShipPortsPane rendered
  at640px and420px and submitted16GiB/8CPUs through an inert service; running ships
  and older runners keep resource controls disabled. Screenshots and source hashes:
  `/tmp/omabit-resources-ui-02`. The first fixture lacked its Hint component; the
  corrected fixture passed. No live service was connected.
- The original restart fixture initially reported old cgroup values for its new
  container; correcting that synthetic response made the resize cycle pass.
- Operator skill validation passed. The collector reference was generated from82
  actual help entries in a networkless private HOME, which remained empty:
  `/tmp/omabit-resources-cli-reference`.

This qualifies source behavior only. It does not claim a real16GiB workload boot,
installed-runner upgrade, every CPU/host combination, or0.4.0 release qualification.
Production/DEV/trial services, protected ships and the persistent preview were
untouched. Cleanup retired only classified superseded compiler/module caches and
retained byte-verified gzip copies of standalone test binaries; the40GiB evidence
budget remains enforced. See the M573 diagnostic for exact sources and evidence.
