# Runtime-profile review and owner-operated migration

**Status: implementation under review; do not install this branch on production
or import a live pier yet.** The native artifact and previous-stable upgrade
rehearsals must pass before an immutable review bundle is offered for production
installation. No production service or ship has been changed to prepare this work.

## What the candidate changes

The intended review line is **0.6.0-rc.1**, based on stable0.5.0. It adds per-pier
immutable runtime selection, a reviewed JSON decoder backport, capability-aware
import review, runtime identity reporting and schema66. The existing default
remains official Vere4.6. Existing enrollments and historical rows are retained;
“restore previous ships” means reopening the same state, not reconstructing ships
from keys or importing them again.

The [runtime policy](runtime-policy.md) explains the supported ordinary-managed
factory and explicit transition refusals. This is not a generic version-switcher
or a change to project/private-chain runtimes.

## Exact reviewed artifact

| Item | Value |
| --- | --- |
| Profile | `vere-4.6-jsonfix-0774857c` |
| Binary SHA256 | `0774857cdb4460fc288bfa55e778249456a7d8bd7b6203922455d887a92fc96c` |
| Vere banner | `4.6-82f4558` (local build label) |
| Upstream base | `5a7614f69bbc783736b08965fdc0a2534bb42e82` |
| Patch revision | `fcc9dd214ae4af07e683490a6cc3f2ded1ab7d47` from urbit/vere#1145 |
| Patched local source | `1a9d61c95c727a397d59d1d0450d04ad2e1da524` |
| Build | Zig0.15.2, Linux x86-64 musl, ReleaseFast |
| Base OCI manifest | `sha256:47fbcf0710ded939931ce3c6408864cee5e8848c5765da0b51a90ca0b1ab1afd` |
| New OCI manifest | `sha256:2b6b7939b09aa0be68bf5a80b796f89382040aeb0d3469c032063a4c93fb5b1d` |
| New config digest | `sha256:9aff0a010f4bfecff2233cc5508a0a6deec078239038456c94bbe0666b7a6f58` |
| Archive | `vere-4.6-jsonfix-0774857c.tar.gz`, 54,750,277bytes |
| Archive SHA256 | `c8a412c42d53fa0c9a8c97aa55a580ddf2864bbdf32f80074dfb6bdfd837f510` |
| Required correction | `vere-json-array-error-1145-v1` |

The packaging script is [build-jsonfix-image.py](../scripts/runtime-profiles/build-jsonfix-image.py).
It accepts only the exact reviewed executable and frozen verified base blobs,
adds one deterministic binary layer, and emits both OCI and Docker-load metadata.
It neither executes the binary nor accesses a Docker daemon or pier. The native
incident's successful tests do not qualify this container by themselves.

[Developer approval and replay constraints](https://github.com/nisfeb/talon/issues/112#issuecomment-6997450705)
and [issue #22](https://github.com/thelifeandtimes/omabit/issues/22) are the source
of the compatibility requirement. Stock Vere can crash replay after the corrected
error path has committed an event. Do not substitute another “4.6” binary.

## Release gates before installation

Required evidence remains separate for source fixtures, real runtime behavior,
and installed upgrades. The final bundle must identify its clean frozen source
commit and hashes. In particular, it needs:

- Two fresh disposable ships retaining different pinned artifacts; native
  startup, stop/start, crash recovery, replacement and maintenance; a malformed-
  JSON negative control that distinguishes the unpatched decoder.
- Native image loading and executable checks, missing/corrupt/wrong-platform
  refusal, and evidence that utilities never invoke the stock binary for a
  patched pier. Checkpoint transition refusal/recovery must preserve history.
- Actual published0.5.0→candidate service upgrade with unchanged identities,
  piers, settings, access, original receipts and peer processes; candidate
  reopen and genuine0.5.0 reader refusal of schema66.
- Matching desktop/helper activation, selected-runtime import and mixed-version
  refusals, signed handoff binding, and source/race checks.

Experiment authorization is subject to [isolation and storage limits](experiment-isolation.md).
Protected ships and the existing manual DEV installations are not test fixtures.
The [source-review evidence](diagnostics/2026-11-20-runtime-profiles-source-review.json)
records passing source fixtures, focused race checks, UI checks and archive
reproduction separately from the pending native gates. The broader historical
race matrix exceeded its eight-minute deadline; it is not recorded as a pass.

## Manual update sequence once qualified

The review delivery must supply an exact checkout revision, verified download
pins and tested installation commands. Do not use the stock updater to select a
topic branch: it fetches remote HEAD and therefore stable master. Preserve any
local checkout modifications before switching to a review revision.

Update the desktop first, then use **Setup → service update** for the separately
selected local runner. Confirm the expected runner identity, existing install and
schema in its review. A fresh-install/reset offer is not an upgrade path. Retain
the existing collector profile, management credentials, runner identity, SQLite
state, storage roots and all pier directories. Do not run an old manager against
schema66 or change its marker to force a downgrade. Earlier installations should
first reach0.5.0 through the documented intermediate stable upgrades.

Afterward, compare the fleet to its saved pre-update view before any new import:
ship identities, storage locations, settings, desired state and operation history.
A manager upgrade must not silently start stopped ships, repair an interrupted
operation, or replace a running peer's runtime. Resolve an uncertain update using
its original attempt and receipt, not another installation or reset.

## Stage the artifact before inspecting a patched pier

These are manual commands for the qualified bundle, not commands run by this
change. Use the bundled matching executables and the actual installed runner
owner/namespace. For a remote runner, stage locally on that host under its owner.
The archive is public runtime code, not a pier backup or credential.

```sh
/path/to/review/urbit-runner runtime-profiles
/path/to/review/urbit-runner runtime-profile-install \
  --profile vere-4.6-jsonfix-0774857c \
  --archive /absolute/path/to/vere-4.6-jsonfix-0774857c.tar.gz
```

From the matching selected collector profile, preflight without opening a pier:

```sh
/path/to/review/omabit --config-dir /absolute/path/to/registry \
  call runtime.profile-preflight.v1 \
  --input-json '{"name":"RUNNER_NAME","runner_id":"EXPECTED_RUNNER_ID","runtime_profile":"vere-4.6-jsonfix-0774857c"}'
```

Require `available:true` and the exact metadata above. The same read is available
through MCP. This does not enroll a ship or grant permission for a live move.

## Owner's later ~sarlev plan

This documents prerequisites only. The native ship remains excluded from
inspection and operation during development. Read [protected-ships.md](protected-ships.md)
before any separately authorized live work.

1. Review the currently running native launcher, actual filesystem object and
   process ownership yourself. The historical path and port in #22 are context,
   not evidence of where the ship is now. Verify that the reviewed patched binary
   is still the runtime in use. Record endpoints and any reverse proxy mappings.
2. Stop that exact native process gracefully and verify it has no remaining
   holders or automatic restart path. A missing Omabit enrollment is not proof
   that the ship is stopped. Never run the native and managed copies together.
3. Preserve a verified stopped-pier backup and the original patched executable,
   provenance and launcher. Keep the pre-patch backup separate. If testing from a
   copy, retain the original stopped copy without importing or booting both.
   A copy does not become a disposable identity.
4. Stage and preflight the reviewed image on the selected runner before import.
   In **Add a ship → Import an existing pier**, select **Reviewed Vere4.6 · JSON
   decoder fix #1145** before inspection. Check returned identity `~sarlev`, live
   mode, exact source fingerprint, destination and chosen host ports. Import stays
   stopped. A runtime/epoch mismatch requires review; do not edit files to pass.
5. Before Start, set **loom34** (16GiB address space), native control off and the
   independent container RAM/CPU limits appropriate to the actual ship. The
   historical launcher used HTTP8888, Ames34348 and `--snap-time 60`. Omabit uses
   an internal HTTP8080 mapped to a chosen host port; preserve or deliberately
   update reverse proxy mappings. Verify public UDP publication separately.
   Do not equate loom34 with a required or measured16GiB resident set.
6. Start only after the native copy is confirmed stopped and all earlier gates
   pass. Require a successful terminal operation, matching requested/launch/image
   identity, the correct `/~/host`, healthy HTTP and expected app state. Inspect
   bounded logs for replay/decoder failures. A displayed version alone is not
   enough. Recording coverage is explicit; retain existing native incident logs.
7. If Start or import is interrupted, retain the exact request, operation,
   original/holding/destination paths and logs. Use supported receipt lookup and
   recovery review. Do not repeat bootstrap, remove locks, reset history or start
   another copy. Stop an uncertain managed runtime before considering any native
   recovery. Returning to native still requires the patched runtime and a
   reviewed choice of the authoritative pier copy.

Retain the image archive, active Docker image, current stopped backup, original
native runtime provenance and pre-patch recovery copy. A future official runtime
needs explicit correction/compatibility review before it can replace this pin.
