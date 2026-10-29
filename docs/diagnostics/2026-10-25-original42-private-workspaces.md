# Original schema42 workspace preservation baseline

M376, 2026-10-25. The original schema42 writer is frozen before adding lifecycle
persistence. Its archive contains2,552 source files/26,033,409 bytes, with nested
historical archives excluded and individually hashed. This is an unreleased source
snapshot, not a release or relabeled older writer.

Two independent extractions reproduce the same1,316,452-byte synthetic fixture.
It contains8 legacy instances,48 operations, existing fake recovery history,
2 private projects/4 members and2 workspace journals: one completed/observed and
one interrupted after layout dispatch with its known root identity retained.
The generator uses the original store functions with synthetic filesystem/runtime
observations; no real workspace, engine, key or ship is involved.

A permanent race-tested preservation check reopens the database3 times, retains
every original table value and sidecar, verifies actor-scoped registration/workspace
replay without port/storage probes, and refuses restarting either old workspace
operation. Original41 and support-manifest regressions also pass:3 top-level/21
including-subcase checks, plus vet.

Schema remains42. The new fixture is a dedicated baseline test; it will enter the
older-reader support inventory when a real successor migration exists. No future
migration or old-reader refusal is claimed. The original42 generator and archive
are retained so that future persistence cannot be tested with rewritten history.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-25-original42-private-workspaces`.
