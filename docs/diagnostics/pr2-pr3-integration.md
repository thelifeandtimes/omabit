# Ames publication and retained-import recovery integration

## Contributor provenance

This integration incorporates work by **wundrellama (~dinnyt-divsud)**:

- [PR #2](https://github.com/thelifeandtimes/omabit/pull/2), commit
  `03b10d1abe8dda17d51d73d64c64ab147bbe008e`: live-enrollment Ames allocation,
  publication, resolved runtime pill provenance and networking regressions.
- [PR #3](https://github.com/thelifeandtimes/omabit/pull/3), commits
  `34963008c0ecb2a2ffc5804e0e986af7b42e29ce` and
  `d512fdc242c28c70efbf33f971d0f97c608f94e1`: pre-move epoch validation,
  explicit recovery, conservative source-absence traversal, atomic completion
  publication and authorization/filesystem/crash regressions.

The code and tests were adapted to RC.8 rather than replacing its guided import,
named-pier, historical-epoch, dock-link and progress support. This integration is
prepared for RC.9; published RC.8 artifacts remain immutable.

## Ames fix

New live enrollment/import allocates an available UDP port through the existing
transactional reservation logic and opts into host publication. Initial identity
verification still has Ames disabled and no UDP publication; subsequent normal
resume uses the selected mapping. Fake ships do not publish host Ames.

Enrollment records the resolved runtime pill URL/hash/size after boot/publication
options are applied, avoiding a provenance mismatch on initialized-pier start.
No key, pill download or fresh bootstrap is introduced by this change.

Existing records are preserved on database open, including running containers and
legacy unpublished ports. Their stopped-state publication migration is tracked in
the [stable upgrade policy](../upgrade-policy.md), not silently performed on live
runtime records.

## Recovery and RC.8 adaptations

- Epoch compatibility is now checked during all external previews and in every
  import worker before source movement, including the legacy CLI without
  `--identify`. It uses RC.8's mixed-history import validator. Qualified pruning/
  roll validation and executable/image pins retain their stricter contracts.
- `recover-import` only completes an external import retained at `enrolling`,
  prepared but uninitialized, with recorded mount/pier identities and no runtime
  containers, network, key/bootstrap intent or unrelated uncertainty.
- Empty, valid Ames-publication configuration is allowed. The original PR's
  unconditional `Boot == nil` restriction would reject new live imports from #2.
- Moved imports require both historical source/holding paths to remain absent,
  with conservative no-symlink/no-mount-crossing evidence checks. Recreated paths
  are preserved and cause refusal, even if they appear unrelated.
- In-place imports instead prove the recorded source is the same retained target
  object and that no holding path was used. They do not require their target to
  be absent.
- Recovery validates epoch history, current filesystem safety and stability, and
  the original raw ship/fake-live identity for guided imports. It neither changes
  pier files nor invokes Docker/Vere. First runtime identity verification remains
  a separate explicit Start.
- Recovery callbacks cannot incrementally clear the durable fence. The completed
  state, stopped observation and succeeded receipt commit in one transaction;
  interrupted or failed publication retains the original fence.
- The desktop offers a bound review and one-shot **Validate and finish enrollment**
  action for server-reported eligible instances. Reviews and request identity live
  in the service so inventory delegate recreation cannot erase them.

## Verification scope

Fixtures cover new/legacy Ames behavior, port conflicts and containment, offline
identity checking, all import preflight paths, moved and in-place recovery,
guided identity/network mismatch, mixed history, unsafe/recreated evidence paths,
workspace checks, cancellation, host-work exclusion, actor/revision/replay rules,
failed terminal transactions and crash/reopen/poll/Stop behavior.

Schema15/16 reopen tests preserve runner identity, serialized runtime/enrollment
records, configured ports, running/stopped intent and operation replay. These are
upgrade-preservation regressions, not a completed end-user updater qualification.

Mapped QML checks cover review survival across inventory updates, stale selections,
duplicate clicks, receipt display and no automatic Start. No live pier recovery,
host update or privileged bind-mount qualification is performed by these tests.

Passed locally with Go 1.25.0 (`GOMAXPROCS=2`, `-p 1` for Go commands):

```text
go test ./...
go vet ./...
go test -race ./internal/managed ./internal/runtimequalify ./internal/client \
  ./internal/access ./internal/collection ./internal/piercopy ./internal/pierstore \
  ./internal/piermeta ./backend/cmd/urbit-collector
CGO_ENABLED=0 go build -trimpath -o build/ ./cmd/... ./backend/cmd/...
node scripts/test-panel-model.js
python3 -B scripts/test-import-recovery.py
python3 -B scripts/test-onboarding.py
python3 -B scripts/test-panel.py --windows
python3 -B scripts/test-log-watch.py
python3 -B scripts/test-panel-install.py
python3 -B scripts/test-native-plugin.py
```

The mapped recovery test also verifies that a definite request refusal can be
reviewed and explicitly retried with a new request ID, with no background retry.
