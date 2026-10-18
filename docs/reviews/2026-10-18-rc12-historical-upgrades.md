# RC.12 historical-writer research and upgrade support matrix

Follow-up: frozen RC.12 artifacts and the supported owner-user schema19 host
bridge subsequently passed live acceptance. See the
[release qualification record](../diagnostics/2026-10-18-rc12-release-qualification.md).
The database/host support distinction and unpublished-prototype limits below remain.

## Decision and qualification status

**Target: `v0.1.0-rc.12`, schema20, prerelease.** Every published Omabit release
RC.1–11 has a reproducible fixture produced with its original writer. The recovered
schema14 stash adds original preview coverage: **13 fixtures**, covering schemas
**13,14,15,16,18,19**. Original writers for **11,12,17 remain unavailable** in the
searched record; none is the writer of a published RC.1–11 release.

The proposed RC.12 support window, encoded in the
[manifest](../../internal/managed/testdata/historical-upgrades/manifest.json), is:

- **Database compatibility:** original RC.1–11 state → schema20, with the
  preservation, replay, reconciliation and downgrade-refusal checks below.
- **Installed-host upgrade:** RC.10/11 **owner-user Linux/amd64 schema19**
  installations satisfying the [one-time offline procedure](../management-offline-upgrade.md).
  This is a narrower contract than database readability.
- **Previews/prototypes:** original schemas14/18 receive additional database
  regression coverage. Schemas11/12/17 have synthetic migration coverage only.
  None of these previews is a supported published release or host-upgrade source.

This closes the missing-*published-release*-fixture question for RC.12. It does
not qualify frozen RC.12 artifacts or promote the candidate to stable. The manifest
labels its evidence `source-tests-only`. This record updates the older twelve-fixture,
four-missing-writer inventory in the [G02 remediation record](2026-10-18-rc11-remediation.md#g02--upgrades-and-rollback)
and [upgrade policy](../upgrade-policy.md); the schema14 source was recovered from
local history after that inventory was written.

## Evidence searched

Research date: 2026-10-18. Local integration HEAD was
`f141360c7563e2069da7411662d2ba6208d512fb`; qualification used the current schema20
working-tree successor, including the historical tests. All research fetches were
into `/tmp/opencode/rc12-writer-provenance.59myau/history.git`.

1. **Full local Git history:** all 85 commit objects, branches, tags, review refs,
   reflogs, `refs/stash` and its index/untracked parents. `git fsck --full
   --no-reflogs --unreachable` found only additional trees/blobs, no additional
   commits. A read-only `git cat-file --batch-all-objects` survey included those
   unreachable blobs and inspected the managed-store writer constants and SQL.
   Local objects contained schema1,13,14,15,16,18,19 writers; no 11/12/17 writer.
2. **Public Omabit:** refreshed all advertised heads, tags and PR head/merge refs
   in scratch. GitHub's paginated releases API lists exactly RC.1–11, all published
   prereleases. RC.2's annotated tag peels to `691eea7f`; its release API's
   `target_commitish: master` is not used as a source pin. RC.10 and RC.11 tags/API
   agree with the already-pinned `f6981e6c` and `abfff052` source commits. The
   retained scratch history also includes the contributor branches for PRs1–4.
3. **Public [`thelifeandtimes/tend`](https://github.com/thelifeandtimes/tend):**
   fetched all advertised heads/tags/PR heads. Both heads point to
   `bd99c33ee90dd1376a78fdbc88220bc0d4fb3c01`; all **134 ancestor commits** were
   available. The tags, releases and PR API inventories were empty. Its old
   consolidated root is
   [`67686ece8d9ca7781ea5d86d17ff3d2cb033c4d1`](https://github.com/thelifeandtimes/tend/commit/67686ece8d9ca7781ea5d86d17ff3d2cb033c4d1),
   tree `063141d327b89777cd5fddc3d10ad378ef5d9016`.
   `apps/urbit-host/internal/fleet/types.go` identifies `0.1.0-pilot`, **JSON
   schema1**; `fleet/store.go` implements `ReadJSON`/`AtomicJSON`, not this runner's
   SQLite store. `apps/urbit-host/IMPORT.md` identifies the original starter archive
   SHA-256 `813fe20dae0d2c2b30270374da616d2511ba1dfbcefee18a1b2d2d09b88ad4bc`.
   Those fleet files persist unchanged until their removal in `3fd51882`.
   They are not source evidence for SQLite schemas11/12/14/17. The combined public
   scratch history contains 219 distinct commits.

The unavailable authenticated `origin` was not contacted. These are bounded
findings about available history and published releases: they do not prove a draft
writer never existed or was never run locally. Documentation mentioning a schema,
or a later reader accepting its number, cannot reconstruct its original writer.

### Missing-writer results

| Schema | Original source result | Classification for RC.12 |
| --- | --- | --- |
| 11 | `34d8ea55f216ebe968cc5a7714aea43c41380490` changes the preceding schema1 writer directly to schema13; 11 is an accepted input. No original writer found. | Unpublished prototype gap; synthetic coverage only |
| 12 | Same consolidated commit accepts 12 but writes 13. No original writer found. | Unpublished prototype gap; synthetic coverage only |
| 14 | Original storage-UI stash `9e956bbab517dc9debe8161c09100e3211df0786`, plus its original untracked-file parent | Unpublished local preview; original fixture reproduced |
| 17 | `ba4ba805221a6ef3e29829c9bda1bfeff0c0c7c2` changes schema16 directly to schema18. Its schema17 tests/documentation are not a schema17 writer. | Unpublished prototype gap; synthetic coverage only |

### Recovered schema14 provenance

The cancelled task had already left `preview14.json`, the source bundle and
generator support in the workspace. This continuation verified their provenance
and reproduced their output, rather than assuming that partial work was evidence.

- Tracked stash commit: `9e956bbab517dc9debe8161c09100e3211df0786`.
- Tracked tree: `9170c9feea349a678f8153fe106d12e20657353a`.
- Original `internal/managed/store.go` blob:
  `afedd916ab2bce1e4efd3d041e80bde92cb6b24e`, writing schema14.
- Required untracked-file parent: `043e1b971800f4e4543e6884d20ef625ec55abf4`;
  tree `e0e58b7f91a9637920e124191d7e662feb1bb584`. It supplies the original storage
  implementation/types needed to compile the tracked snapshot.
- The index parent `07457c189811b7248dd8131afade53b79529af48` still contains the
  schema13 store; it is not substituted for the working-tree snapshot.
- Retained `schema14-source.bundle` SHA-256:
  `09e2f733ae992826ee21aeee32cc2826bed5b46cd8f9db613b747ecf0ca86556`.
  `git bundle verify` passes. Its prerequisite is the public RC.1 ancestor
  `87c8dd3fce80a1692436cd61941e2143093aef12`.

The generator imports the bundle into a private temporary repository, archives the
two exact trees and refuses file overlap. It injects only the test harness; it
does not repair or rewrite historical production code. `preview14.json` reproduced
byte-for-byte (60,326 bytes). No schema11/12/17 fixture was fabricated.

## Release-by-release tested matrix

“Pass” means original-writer regeneration, current database migration/regression
checks and original-reader refusal of the resulting schema20 database all passed.
Full commit/tree hashes and generator digests are retained in each fixture; the
manifest additionally binds each release to its source commit. Release links are
available under `https://github.com/thelifeandtimes/omabit/releases/tag/<tag>`.

| Published tag | Original source | Schema | Fixture | Database →20 / old-reader refusal | Initial host path |
| --- | --- | ---: | --- | --- | --- |
| `v0.1.0-rc.1` | `d67ea3ad` | 13 | `rc1.json` | Pass | Unsupported |
| `v0.1.0-rc.2` | `691eea7f` | 15 | `rc2.json` | Pass | Unsupported |
| `v0.1.0-rc.3` | `1b6b7919` | 16 | `rc3.json` | Pass | Unsupported |
| `v0.1.0-rc.4` | `919ba0ce` | 16 | `rc4.json` | Pass | Unsupported |
| `v0.1.0-rc.5` | `b973b5a9` | 16 | `rc5.json` | Pass | Unsupported |
| `v0.1.0-rc.6` | `2786969d` | 16 | `rc6.json` | Pass | Unsupported |
| `v0.1.0-rc.7` | `07e407a4` | 16 | `rc7.json` | Pass | Unsupported |
| `v0.1.0-rc.8` | `f972dee2` | 16 | `rc8.json` | Pass | Unsupported |
| `v0.1.0-rc.9` | `d9b180fd` | 16 | `rc9.json` | Pass | Unsupported |
| `v0.1.0-rc.10` | `f6981e6c` | 19 | `rc10.json` | Pass | Owner-user offline bridge; source-tested |
| `v0.1.0-rc.11` | `abfff052` | 19 | `rc11.json` | Pass | Owner-user offline bridge; source-tested |
| Local storage preview, no release tag | `9e956bba` + `043e1b97` | 14 | `preview14.json` | Pass | Unsupported |
| PR4 runtime-settings preview, no release tag | `ba4ba805` | 18 | `preview18.json` | Pass | Unsupported |

`internal/managed/store.go` accepts existing schemas11–19 transactionally; it also
opens current schema20 and initializes an empty database. That reader window is
not the installer's deployment support window. The actual initial offline path in
`internal/hostinstall/offline.go` requires unfenced schema19 and a matching verified
schema19 installed executable. `internal/managed/offline_update.go` admits schema19
or an exact journaled current-schema continuation. It refuses other historical
schemas before migration. Existing owner profile, reviewed unit, grants, host
identity, idle/reconciled state and runtime ownership locking remain prerequisites.

For RC.10/11, use the **new** executable's documented `install-host
--offline-upgrade` procedure after original-runtime preparation and stopping only
the management unit. Following completion, compatible new-to-new updates use
normal quiescence-aware installation. After schema20 commits, recovery is forward
retry with the retained target; a schema19 binary is not a rollback target.

RC.1–9, preview installations, legacy dedicated-account deployments and custom
unit invocations have **no qualified installed-host migration in this matrix**.
A chain through RC.11 is not implicitly supported by these tests. Such an expansion
requires its own deployment-preserving procedure and qualification. Database
fixtures are not instructions to open or copy a live database as an upgrade shortcut.

## Maintained gates and results

Fixtures contain wholly synthetic identities, keys, filesystem observations and
runtime observations persisted through original historical code/types. They are
logical SQLite exports with original DDL, cells, recorded schema and blob bytes,
not production snapshots or contemporary objects with a lowered schema marker.

`internal/managed/g02_historical_upgrades_test.go` now enforces:

- A distinct pinned original fixture for **each supported release**. Losing RC.3
  cannot be hidden by retaining RC.4's same-schema fixture. The complete published
  RC.1–11 inventory is an explicit gate, not a fixture-count heuristic.
- Separate preview coverage and explicit unavailable-writer entries for11/12/17.
  Every accepted historical schema has original coverage or an honest gap.
- RC.12 prerelease/source-only qualification and the narrower schema19 owner-user
  host contract. Negative tests reject missing/substituted releases, changed source
  provenance, previews presented as releases, concealed gaps, overstated host
  support and a stable-promotion claim.
- The actual offline identity/schema gate against every original fixture. Only
  RC.10/11 pass the initial schema/identity check. All full fixtures deliberately
  contain unresolved work, so repeated offline inspection **and fencing** refuse
  them without modifying database bytes. This verifies original-runtime
  preparation is still required; it is not a successful host activation exercise.

Existing historical tests exercise three opens/reconciliation passes, stable
runner/instance identities, configuration/ports, running/stopped intent, original
signed handoffs and permanent source fences, actor-scoped replay/conflict behavior,
custom-root sidecars, enrollment/SSH pins and grants, owner/modes, omitted native
settings and RC.10 small-loom preservation, failed-migration retry and future-schema
refusal. The driver compiles all 13 old readers against databases actually upgraded
by the current migrator; each refuses schema20 twice without modifying it.

Passed on this continuation (`go1.27.0-X:nodwarf5 linux/amd64`):

```sh
python3 -B scripts/test-historical-upgrades.py --check-generated --only preview14
python3 -B scripts/test-historical-upgrades.py --check-generated
go test -count=1 ./internal/managed -run '^(TestOpeningExistingStatePreservesLegacyAmesAndRecovery|TestSchemaMigrationFailureRollsBack)$'
go test -count=1 ./internal/managed ./internal/hostinstall ./backend/cmd/urbit-runner -run 'TestRC11Offline|TestRC11.*Activation|TestRC11.*Prepared|TestRC11.*ManagementUpdate'
gofmt -d internal/managed/g02_historical_upgrades_test.go
git diff --check
```

The full driver also checks the manifest byte-for-byte and runs all
`TestG02Historical*` tests. The existing synthetic preservation test separately
passes all nine reader inputs11–19; it deliberately relabels contemporary test
state and therefore supplies no original-writer evidence for11/12/17.
The installer suites use real temporary SQLite/locks
with synthetic host/process/artifact evidence and injected activation failures.
They supplement, rather than replace, original-writer database coverage. No live
ship, service, installer, credential or host configuration was operated on here.

## RC.12 versus stable gates

**RC.12:** missing unpublished prototype writers are not a missing-supported-release
blocker under this explicit window. All published RC.1–11 original-writer database
paths pass. Final release acceptance still needs the frozen RC.12 artifact/pin and
supported-host upgrade evidence tracked by the parent qualification effort,
including privileged isolation/systemd readiness and disposable-ship preservation.
This research did not perform that live qualification. Advertising automatic
installed-host upgrades from RC.1–9 would be an unsupported claim and would require
additional work before publication with that claim.

**Stable 0.1.0:** [the upgrade policy](../upgrade-policy.md) remains a separate
mandatory gate. Preserve original fixtures for every release/schema in the stable
support window, including the actual stable writer when frozen; prove a
non-destructive stable→successor path, N→N+1 and repeated/interrupted boundaries,
compatible rollback or downgrade refusal, mixed client/server/UI behavior, unchanged
identities/data/permissions/intent/replay and a disposable-ship upgrade without a
host reset. Publish the exact support window and command/maintenance requirements.
Neither these RC fixtures nor a fresh-install exercise supplies that future-upgrade
proof. Unsupported, unavailable prototypes should stay documented as gaps, not be
invented to satisfy a count or silently treated as supported releases.
