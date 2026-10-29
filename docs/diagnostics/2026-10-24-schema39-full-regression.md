# Full schema39 regression — 2026-10-24

The full managed race suite passed in 1722.999 seconds; the following vet command
also completed successfully (session55302 exit0). All production inputs remain
identical to the 443-file pre-run snapshot. The only later change is the inert
missing-container fixture in `internal/runtimequalify/project_anchor_created_start_linux_test.go`,
separately covered by M170's concrete race gate.

The temporary continuation overlay also passed its focused prototype cases in
12.554 seconds (session88636 exit0). That overlay is outside the repository and
is not format, migration, restart, or runtime qualification. No schema40 or public
continuation route has been introduced.

This supersedes M169/M170's pending full39 regression only. VM31 remains a failed
trial because final cleanup rejected recovery-container ownership metadata; its
stopped disk awaits independent extraction. No coordinated runtime pass is claimed.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-schema39-full-regression`.
