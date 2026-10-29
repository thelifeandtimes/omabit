# Full schema40 managed regression

The complete managed-runner race suite passes in 2164.666 seconds, followed by
successful vet and diff checks. All 874 recorded Go/module/testdata inputs match
the starting snapshot; their exact bytes and logs are retained. This baseline
includes explicit interrupted-recovery continuation and its guarded VM fixture.
It excludes the separately staged public saved-status API/CLI changes.

This supersedes M174/M175's pending full40 regression only. VM32 reports a clean
runtime pass at frozen schema39, with independent disk verification next. Actual
schema40 continuation runtime and public API runtime qualification remain open.
No installed database, plugin or ship was changed.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-schema40-full-regression`.
