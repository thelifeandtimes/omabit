# Read-only private member mount verification

M413, 2026-10-26. A member-local verifier now holds the existing private-projects
parent, project root, piers directory and selected member mount by their recorded
Linux directory identities. It reopens the named chain around each check and
refuses replacement, symlinks, cross-device descent, changed permissions and
unexpected membership. It creates, repairs and syncs nothing.

Verify checks those objects without interpreting their contents. VerifyEmpty
adds a first-boot precondition: even a partial child is refused and retained.
Neither method reads peer piers, chain databases, key files or engine state.
The handle supplies no dispatch or general project-readiness authority; same-UID
code remains trusted. Resume still needs its exact pier/.urb identity and stopped
state, and boot still needs independently verified input files and live gateway.

All17 targeted race-enabled checks pass with package vet and diff checks. The
tests use synthetic temporary directories and cover caller edits, peer contents,
partial boots, same-shaped replacement at every retained level, unsafe selection,
membership changes, cancellation and concurrent close. Refusals preserve data.

This extends M412's source-only member work. The schema and public interfaces
remain unchanged. The concurrently running diagnostic VM07 uses its already
frozen earlier binaries and does not qualify this helper. M413 supersedes no
previous runtime evidence. Managed member launch, member-aware gateway observation,
durable lifecycle and a fresh runtime acceptance test remain.
