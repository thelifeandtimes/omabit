# Sourceport moon boot: registered and actual identities differ

Confirmed 2026-11-05 local time (2026-11-06 UTC), candidate
`0.3.7-dev.20261105.5`. Read-only investigation requested by the user after their
Recommended Apps moon boot remained in Recovery.

## Evidence

- Runner: `runner-448a268320cb07b0662e01b6f6c7b681`, enrolled as `local`.
- Registered identity: `~somrep-mitfyl-dozzod-sarlev`.
- Actual public `/~/host` response: **`~somryp-mitfyl-dozzod-sarlev`** (HTTP 200
  through the inspected `127.0.0.1:18080` binding). The first suffix differs:
  `rep` versus `ryp`.
- Exact retained container:
  `a05c2399d458f043b50a550882f783db3b38eea2cf064e1d06a1e9a4455fb7ee`.
  Docker reports running since `2026-11-06T00:26:34.704870947Z`, no OOM kill.
- CLI logs show bootstrap complete at `00:30:50Z`, the public and private HTTP
  listeners live at `00:30:53Z`, and later app installation/update activity. Runtime
  sponsor/boot requests also use `~somryp-mitfyl-dozzod-sarlev`.
- Initial Start operation `op-e0d244d970d621097f76c1a4c31fb534` failed at
  `00:31:37Z`: `public comet identity is invalid or changed across runtime replacement`.
- Subsequent user Stop operations were admitted but failed with
  `graceful shutdown could not verify private Eyre identity; runtime retained`.
- Snapshot: config revision 1, applied revision 0, initialized false,
  recovery_required true. Desired stopped is the user's later Stop intent, not
  evidence that the running ship stopped.

The collector `runners`, identity-pinned `snapshot`, and revision-pinned `logs`
commands supplied management evidence. Exact-container Docker inspect and a
credential-free GET to its inspected public identity endpoint supplied independent
runtime evidence. No key contents or pier contents were read directly, and no
runtime, registration or service mutation was attempted.

## Meaning

This is a durable identity mismatch, not stale UI status or a wait for Recommended
Apps to finish. `readyCometIdentity` in `comet_linux.go` also handles keyed owned
ships: its managed/native branch requires equality with the registered ship.
The error's "comet" wording is misleading for this shared path.

`ManagedState.UncertainProvisioning` retains the unresolved first-boot flag.
`observeManagedRuntime` returns recovery immediately for that flag, rather than
silently adopting a different identity. Ordinary Start is blocked. Stop is allowed
through admission, but the keyed first-boot shutdown path independently checks
private Eyre `our` against the expected name and refuses a mismatch. Repeated
refresh or Stop cannot correct the registration.

## Proposed next step

Confirm that **`~somryp-mitfyl-dozzod-sarlev`** is the intended moon. If so, the
required repair is an explicit, audited correction of the mistaken registration
to the independently verified identity, preserving the existing pier and runtime.
The present CLI has no identity-correction/reconcile-boot operation;
`enroll-pier --relink` requires the existing identity and a stopped source, and
`recover-import` is for interrupted imports, not this keyed first boot.

Do not bypass the check by clearing the recovery boolean or editing the live
database. Design/test a narrow correction path that binds the runner, revision,
expected and observed identities, exact retained container and original storage,
rejects duplicate identity enrollment and unrelated uncertainty, retains the
failed operation history, and re-establishes normal observation through verified
boot completion. Its applicability to this protected moon requires the user's
identity confirmation and a reviewed live repair plan. Error/UX follow-up should
show expected versus observed identity prominently, and key identity validation
before first boot should be evaluated.

## Approved repair and source qualification

The user confirmed `~somryp-mitfyl-dozzod-sarlev` and explicitly approved
"Yes, repair and restart": verify this exact moon, gracefully stop only it, correct
registration, and resume its existing pier. This is the specific exception to the
protected-moon operational restriction for this repair.

The implemented [offline CLI repair](../boot-identity-repair.md) binds the existing
runner, revision, old/new identities, exact container and filesystem objects.
Identity evidence and removal progress are committed before effects; final
registration, completion provenance and audit receipt commit atomically. Tests
cover clean correction, refusals, interruption/replay, failed commit, history and
storage preservation, unchanged-schema reopen and normal same-pier resume.
Live application evidence follows after the broader source checks.

The first whole-package race run hit Go's aggregate ten-minute timeout in both
large packages: runtime tests were in `TestRecoveredResumeContractRejectsForeignProject`
and store tests in `TestG02HistoricalConfigurationOmissionAndSmallLoom/schema41`.
There was no reported assertion failure; this run is **not a passing full-suite
gate**. Runner CLI tests passed. Focused repair/stop race tests and offline repair
commit/replay race tests passed, as did `go vet` for all three changed packages and
the network-isolated generated CLI reference check (133 help entries). A bounded
affected-lifecycle run also includes both tests active at the aggregate timeout.

## Completed live correction — 2026-11-06 UTC

The bounded race run passed: runtime/lifecycle tests in15.054s, managed/store
tests including historical omission/small-loom cases in63.622s, and selected CLI
tests in1.017s. This supersedes the pending bounded-run status above, not the
recorded full-suite timeout.

Only the owner's management service was stopped. The exact standalone helper
(`sha256:9997f4aeae4de88f570a71be3140215d8cadf89805d9f3f3c413ed46060515a0`)
completed repair operation `op-13e2ba665d4ffbade31a2eee59531ba0` between
`01:56:11Z` and `01:56:16Z`. Configuration revision advanced1→2. The original
boot process exited cleanly; its container was removed after durable terminal
evidence. The dedicated engine remained running with unchanged PID404613.

Management restarted using its existing installed executable. Ordinary collector
Start operation `op-2083a1c2414aa17de4f64dc4228e7129` succeeded at `01:56:31Z`.
The resulting `comet-resume` container is
`8bff0084eeb4ab492bb54c1cab8c931aeab03ec1fea1ebc5345eacf4fdcb4076`.
Both management and the public endpoint report **`~somryp-mitfyl-dozzod-sarlev`**;
the snapshot reports running, healthy, initialized, fresh CPU/memory metrics and
applied revision2, without recovery. The default display label is corrected too.

Final verification confirmed unchanged mount/pier/`.urb` device and inode values,
the same runtime image, ports, loom and Recommended Apps pill, and all nine prior
operation receipts preserved. The original stable instance ID and pier path still
contain `somrep`; they were intentionally retained as storage/history identifiers.
The new command line resumes the pier (no keyed `-w`/create `-c` arguments).

Compact artifact hashes, before/after snapshots and metadata verification are kept
in `~/.local/share/omabit-review/identity-repair-20261106/`. The versioned recovery
journal remains in the managed database. The first verification watcher expected
a false `recovery_required` JSON member; the API omits false values, so the watcher
was corrected and rerun read-only. No lifecycle request was repeated for that
verification-script error.

The helper was built separately; installed runner/plugin artifacts remain the
user's `.5` trial versions. Source changes remain uncommitted and unpushed on
`feature/one-click-host-setup`.
