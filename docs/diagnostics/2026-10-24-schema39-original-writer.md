# Original schema39 and actual successor refusal

The frozen unreleased schema39 snapshot contains1625 files,3545973 archive bytes,
SHA256 `e9afaa95aa2d4ad0531baee741925c4fb8e823ce2644118888bcb7633fb3d356`.
Its original writer reproducibly creates a1095923-byte,21-table fixture with one
completed recovery, a second interrupted recovery, an interrupted observation and
a completed observation. Fixture SHA256:
`349291bec95a728311d5d9c6b9193b22a88ca98c55caeecfd760eefc35d04d8d`.

The actual schema40 migrator preserves prior bytes and adds continuation ownership.
The focused migration/unknown-Create gate passes in14.460s. The frozen original39
reader refuses actual populated40 state without modifying it. No marker-only
relabeling is used. These are source/format tests, not a release or runtime pass.

Failed fixture attempts retained: wrong store method name, then a shallow clone
that altered its own input observation. The corrected generator clones before
modification and uses the existing original writer unchanged. Initial migration
checks also caught omitted explicit target/window assertions; those were updated
to the reviewed40 target and11–39 input window. Prior release claims stay frozen.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-schema39-original-writer`.
