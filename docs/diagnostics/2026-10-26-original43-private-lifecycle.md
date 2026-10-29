# Original schema43 lifecycle preservation baseline

M408, 2026-10-26. The original schema43 writer is frozen before member lifecycle
persistence. The source-only archive contains 2,678 files / 28,139,646 bytes;
nested historical archives are excluded and individually hashed. It is an
unreleased source snapshot, not a release or a relabeled older writer.

Two independent source extractions reproduce the same 1,369,311-byte synthetic
fixture. It retains 8 legacy instances, 54 operations, fake recovery history,
2 private projects / 4 members, 2 workspaces, 2 gateway journals, and 2 each of
workspace and gateway observation receipts. One project has a completed workspace
and uncertain gateway initialization followed by separately attributed recovery.
The other has uncertain workspace creation followed by recovery and completed
gateway initialization. Both observation families retain an earlier interrupted
attempt. Synthetic resources/public keys remain distinct between projects.

The permanent preservation test reopens the database three times, compares every
original row and sidecar, checks original ownership and separate readiness
attribution, verifies actor-scoped replay with no backend inputs or host probes,
and refuses restarting any original initialization or observation attempt.
Original41/42 and support-inventory regressions also pass: 4
top-level / 22 including-subcase checks under the race detector, plus vet.

Schema remains43. This is a dedicated baseline test; add it to the historical
support inventory when an actual successor migration exists. No future migration,
old-reader refusal, installed upgrade, or runtime acceptance is claimed.
The initial generator wrapper refused a source occurrence-count assertion before
extracting source or creating databases; the corrected wrapper used a fresh
private test directory. Failed and successful preparation evidence is retained.

The separate diagnostic VM remains in progress. M407's failed runtime remains
valid; this baseline does not supersede it or any earlier source/runtime evidence.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-26-original43-private-lifecycle`.
