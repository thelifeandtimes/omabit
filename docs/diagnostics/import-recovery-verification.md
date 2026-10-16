# Import compatibility and recovery verification

This change separates epoch provenance from the pinned runtime identity.
It also checks epoch compatibility before a move and adds explicit recovery for a retained enrollment failure.
Runtime image pins, executable checks, and the management schema remain unchanged.

## Failure reproduced

The pinned runtime reports `4.6-5a7614f`. Stock Vere 4.6 writes `4.6` to each epoch's `vere.txt`.
The previous epoch scanner required an exact match with the pinned runtime string.
It therefore rejected stock-release epochs after the importer had moved the pier and removed the held source.

The affected registration retained `external_import.stage=enrolling`, `Prepared=true`, and `Initialized=false`.
A normal import retry could not resume it. Stop did not complete enrollment.
Stop followed by Delete could remove the sole remaining pier and was not a recovery procedure.

## Compatibility evidence

The pinned Vere source distinguishes release and nonrelease version strings:

- [`build.zig`](https://github.com/urbit/vere/blob/5a7614f69bbc783736b08965fdc0a2534bb42e82/build.zig) uses `4.6` for release builds and adds a commit suffix for nonrelease builds.
- [`disk.c`](https://github.com/urbit/vere/blob/5a7614f69bbc783736b08965fdc0a2534bb42e82/pkg/vere/disk.c) writes epoch format and runtime provenance to separate files.
- The epoch disk implementation is identical between the public `vere-v4.6` source and the pinned commit.

The scanner accepts only format `2` with provenance `4.6` or `4.6-5a7614f`.
It does not accept arbitrary version prefixes or rewrite metadata.
Runtime executable and image checks remain exact.

Header compatibility does not prove boot compatibility.
Vere can reject a runtime change when the saved snapshot is behind the event log.
Use the original runtime's `prep` utility on a stopped pier before the runtime change.
Do not edit historical metadata to bypass a stale-snapshot error.

## Regression tests

Each primary behavior had a failing test before its implementation:

| Behavior | Regression coverage |
| --- | --- |
| Epoch provenance | Accept the two supported strings. Reject unsupported versions and formats. Preserve metadata. |
| Pre-move validation | Reject an incompatible source before mount preparation or source movement. Recheck in the worker. |
| Enrollment recovery | Complete the retained prepared failure before or after explicit Stop. Leave the ship stopped and identity pending. |
| Recreated source paths | Refuse copied piers, locked copies, and unrelated replacement directories. Preserve all data. |
| Filesystem traversal | Refuse symlinks, inaccessible evidence, source-device mismatch, malformed holding paths, and late source recreation. |
| Destination checks | Refuse replaced mount, pier, or state directories, unsafe links, lock markers, and concurrent tree changes. |
| Journal publication | Publish completed state and the succeeded receipt in one transaction. Preserve the private fence after interruption or transaction failure. |
| Authorization and replay | Check operator access, runner identity, configuration revision, actor-scoped replay, and host-work exclusion. |
| Runtime isolation | Make no Docker or Vere calls during recovery. Do not copy, remove, bootstrap, or rewrite pier data. |
| Workspace limits | Reject unsafe workspaces. Explicitly allow safe replacements without claiming historical inode verification. |

The publication tests close and reopen a fixture database before reconciliation.
They cover a crash before the terminal transaction, an injected receipt-write failure, cancellation, and a late backend error.
Polling and Stop cannot clear an interrupted recovery's private fence. Start remains blocked.

## Executed checks

Go 1.27.1 completed these checks successfully:

```sh
go test -race \
  ./internal/runtimequalify ./internal/managed ./backend/cmd/urbit-collector \
  -run '^(TestEpochProvenanceAllowsOnlyVettedFormatTwoVersions|TestEpochInspectionRejectsUnsafeAndUnsupportedLayouts|TestImportEpochPreflight|TestPreviewEpochs|TestRecoverImport|TestRecoverRejects|TestRecoverJournalFailure|TestRecoverRecreatedPaths|TestRecoverPathTraversal|TestRecoverWorkspaceChecks|TestRecoveryJournal|TestRecoveryInterrupted|TestRecoveryGates|TestRecoveryRPC|TestRecoveryCLI|TestRecoveryPublication|TestRecoveryActorReplay|TestRecoveryPostGateMaintenance)$' \
  -count=1

go test -race ./internal/runtimequalify ./internal/managed \
  -run '^(TestDurableIdentityRevisionAndReplay|TestExternalImportMovesAndEnrollsWithSameAndCrossDevicePaths|TestImportChangedSourceDoesNotMoveAndInterruptedCopyStaysFenced)$' \
  -count=1

go vet ./...
go build ./...
git diff --check
```

An independent review found two defects in the first implementation: missing source reconciliation and a separate completion-state transaction.
The corrected implementation passed independent review and the race tests above.

The unfiltered local suite had pre-existing environment failures:

- Long temporary-directory paths exceeded Unix socket pathname limits.
- `TestHostPathAllowsOnlyTheRunnersNarrowACLGrant` failed when its presumed foreign UID matched the test user's UID.

These failure classes also occurred on the unmodified base.
A supplemental filtered run passed. This is not a claim that the unfiltered local suite passed.
No privileged mount-namespace integration test was performed.
The traversal code fails closed when the required kernel operation is unavailable.

## Live verification

The reviewed RC6-based build completed these operations on one owner-user host:

1. Recovered one retained post-move enrollment failure without starting the ship.
2. Preserved all 15 file hashes across recovery.
3. Prepared that pier with its original Vere 4.6 binary and started it with the pinned runtime.
4. Imported a second stopped, prepared pier across filesystem boundaries.
5. Matched all 565 destination file hashes before its first managed start.
6. Started the second ship and verified both expected identities and HTTP endpoints.
7. Verified owner-user writes and normal read/write access to an existing mounted desk file.

Both ships reported healthy. The first container remained running during the second migration.
These results do not establish general compatibility for other runtime versions, platforms, or incomplete import phases.

## Recovery limits

Recovery supports only a prepared, uninitialized external import at `enrolling`.
Earlier copy/removal phases, missing workspaces, and unrelated uncertainty still require review.
The operation does not open LMDB or prove historical file-content integrity.

Legacy fingerprints bind destination directory identity, not historical LMDB bytes.
Workspace checks verify current paths, ownership, and permissions, not historical workspace inodes.
Legacy journals also lack source-parent inode and mount identities.
Keep external writers and supervisors stopped throughout recovery.

See the [recovery procedure](../storage-management.md#reviewed-recovery-procedure-not-deployment-instructions) for the required checks and explicit command.
