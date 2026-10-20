# Stable0.1.0 verification record

Frozen source: `9cc1d217df19c0aa9bcf6d92a5f8da500f022ffc`.
The machine-readable [qualification](qualification.json) binds exact artifacts,
backend-source identity, checks and scope. It is also attached to the release.

## Non-destructive forward-upgrade gate

A new QEMU/KVM overlay was created over the stopped, previously qualified RC.12
lab disk. The old disk stayed read-only; its full hash was unchanged afterward.
Only the new disposable guest was booted. No live user pier was copied into it.
The existing fake/comet records were inspected before changes; the comet remained
stopped. The test exercised a running initialized fake and preserved stopped state
for the other registrations.

The actual stable installer upgraded the old schema20 deployment to schema21.
Five engine-package archives previously installed with native `pacman -U` were
still in the lab's retained input directory, but absent from Pacman's cache and no
longer available at their old configured-repository URLs. The first verification
correctly refused them. Their original signature-bearing archives were copied into
the guest's native cache; the unmodified verifier then authenticated all eight roles.
No package installation, forged receipt, key import or engine-policy override was used.

With the stable generation active, a new registration receipt was created by the
stable writer and an initialized fake was started. The real management installer
then performed three update boundaries, each followed by an exact no-restart repeat:

1. Stable21 → distinct, clean same-source schema21 generation (`-ldflags=-s -w`).
2. That compatible generation → official stable21 binary.
3. Stable21 → a deliberately modified successor-schema22 fixture. Its only source
   change was the schema constant, and the actual transactional migrator advanced
   the existing database; no marker was manually rewritten.

All boundaries preserved the running container/PID/start time, mounts/ports/image,
engine invocation, ownership, identities/configurations, stopped intent, historical
operation rows/fingerprints, policies, grants, SSH keys and collector pins. Original
creation receipt replay matched across every generation. An attempted stable21
installation after schema22 returned:

> candidate cannot read current state schema; healthy management was preserved

The running successor manager, its executable and schema22 state were retained.
The fake was cleanly stopped, all lab ships were left stopped, evidence was collected
and the guest powered off. The host's existing runtime observations were unchanged.

The two successor artifacts are explicitly test fixtures, not invented future
release versions. Every real successor must qualify its own changes and migrations.

## Other release checks

- Source/policy/channel/model tests, Go tests/vet and required CI passed.
- Two independent clean builds produced byte-identical artifacts.
- Checksums, schema, VCS/tree/epoch provenance and isolated snapshot installation,
  replacement and customization-preservation checks passed.
- Real Quickshell tests loaded the frozen UI and actual helpers in an isolated
  profile and verified view restoration and activation/recovery boundaries.
- Govulncheck v1.9.0 found no vulnerabilities in the three frozen binaries.
- Earlier RC qualification and the user's successful RC.15 fresh setup remain
  separately identified evidence, not relabeled as new frozen0.1.0 runs.

Initial lab attempts are retained: a missing `hostname` utility in the minimal
guest, a harness receipt-envelope parsing mistake after successful installation,
and an unpinned `suggest-port` request. The harness was corrected, completed state
was inspected rather than reset, and no application admission checks were relaxed.

Detailed local artifacts and before/after observations are under
`build/stable-0.1.0/`. Public download/cold-cache checks are performed after publishing
the already-qualified draft, without rebuilding its artifacts.
