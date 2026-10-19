# Host compatibility development verification

Implementation follows RC.12 / repository base `b5ec5ab`. This record captures
verification performed before the source freeze for RC.13; the changes were
uncommitted and unreleased at that time. It is not final-artifact or live-host
migration evidence. The running user's packages, services and piers were not changed
by this development work. See [RC.13](../release-rc13.md) and its attached
qualification record for subsequent publication/artifact checks.

## Implemented scope

- Generic runner-owned policy and Pacman/Debian/Ubuntu providers, with separate
  installed/running engine versions, authentication and qualification status.
- Engine API1.47 and supported stable-family checks, retaining rootless, owner,
  socket, data-root, containment and resource controls.
- Actual native package signature/index/payload authentication, retained receipts,
  explicit missing-evidence retrieval, and native post-transaction refresh hooks.
- Read-only identity-bound RPC and collector CLI reports; collector-only hosts
  require no local engine. Existing Omabit setup/monitoring consumes the same data.
- Known-content startup-guard migration, resumable initial setup and update hooks,
  existing subordinate-ID preservation, and schema21 downgrade fencing.
- Typed host-compatibility failures preserve genuine recovery fences without
  inventing pier damage or poisoning the state journal.

## Combined source checks

Passed on both the minimum Go1.25.0 toolchain and local Go1.27:

```sh
go test ./...
go vet ./...
GOTOOLCHAIN=go1.25.0 go test ./...
GOTOOLCHAIN=go1.25.0 go vet ./...
```

Expanded race checks passed for hostcompat, hostsetup, hostinstall, rootless,
runtimequalify, managed, access, client, collection, protocol, onboarding, both
generic commands and the desktop helper. JavaScript panel/runtime models, the
new host-report model, both actual offscreen Qt suites, snapshot installer,
native bootstrap and desktop-qualification boundary tests also passed.
Actionlint passed; the workflow now includes the new Qt report test and native
authentication tools. No remote CI execution is claimed for this uncommitted work.

Static command builds are under `build/host-compatibility/`; the runner reports
schema21. A read-only local `host-check --live --format text` successfully queried
the existing private engine and reported installed/running29.7.2, API1.55 and
compatible/unqualified status. It correctly reported metadata-only package
provenance because the new authentication receipts have not been installed on
the user's host. That check did not open/migrate the management database or
change any host configuration. Govulncheck v1.9.0 reported no vulnerabilities.

## Original-writer upgrade preservation

`scripts/test-historical-upgrades.py --check-generated` reproduced all14 fixtures:
every published RC.1–12 writer plus original schema14/18 previews. Each original
reader refused a database actually migrated by the schema21 implementation,
without modifying it. Published RC.12/schema20 facts remain historical; the
manifest records a separate unreleased development target. Unavailable unpublished
writers11/12/17 remain explicit gaps.

Tests cover version-only migration from11–20, unchanged legacy bodies/signatures,
exact replay, omitted historical engine fields, new per-container engine evidence,
normal20→21 activation, offline19→21 fencing, incompatible rollback refusal,
repeated opens and forward recovery. Fixture inputs are synthetic original-writer
exports, not copies of user piers or production credentials.

## Native authentication and transaction tests

Actual native execution used a fresh signed Ubuntu24.04 QEMU guest with2 vCPUs and
1536MiB RAM, plus a signed Arch bootstrap contained inside that guest. The payload
packages were inert, locally signed **test fixtures**; Docker/Vere were not launched.

The completed matrix contains seven groups,40 verifier calls,31 UID1000 reader
calls and four native package transactions. Fifteen deliberately invalid cases
failed closed. Covered behavior includes:

- Native APT update/download, `apt-helper`, `dpkg-deb`, GnuPG/gpgv and armored keys.
- Source-scoped Signed-By, exact package versions, installed bytes, retained proof,
  OS-cache cleanup, unrelated dpkg changes and offline renewal.
- `_apt` UID42/GID65534 acquisition with empty supplementary groups; permission,
  missing-user and HTTP403 failures refuse without a root-download fallback.
- Native Pacman detached/repository `PGPSIG`, epoch-bearing identities,
  `pacman-conf`/curl acquisition, tampering and revocation.
- Both real package managers invoke the refresh before later cleanup; refreshed
  proof remains readable by UID1000 afterward, including unchanged support roles.
- First and replacement hook invocations under umask077 publish public receipts
  as root-owned0644 and parents0755; private evidence remains0700/0600.
- Legitimate multisigned Release handling, unavailable extra signers, and refusal
  of invalid/revoked selected authority.

Native tools included APT2.8.3, dpkg1.22.6ubuntu6.6, Ubuntu GnuPG2.4.4,
Pacman7.1.0/libalpm16.0.1, Arch GnuPG2.4.9, libarchive3.8.9 and curl8.22.0.

Main evidence: `build/hostcompat-native/run-fixed-20261019T023329Z/REPORT.md`.
Core manifest at that run:
`8c610d20cf023aae9ef33af692098e80e9452bafd8e357612cc820a28b4f3a20`.
The native run found and verified the correction of an earlier real umask bug;
failed snapshots remain preserved rather than counted as passing evidence.

A focused native APT2.8.3 rerun then verified the final colon-fragment authority
correction: `99:authority.conf` changes actual native source selection and withdraws
reader/root authorization; unrelated colon-named configuration preserves it.
Only the matcher and its test changed from the preceding native-tested core;
other crypto/fetch/cache/I/O files were byte-identical.

Evidence: `build/hostcompat-native/run-colon-20261019T033848Z/REPORT.md`.
Final core manifest:
`396150ec2aca6b512b377d3b606c7f88fd8ee2263ab4fa329d7bb7d3092b5e28`.
The disposable guest and seed server were powered off after testing.

The reusable sanitized harness is retained under
`scripts/qualification/host-packages/`, including provenance notes. Its default
help/plan is harmless; mutation requires the dedicated guest/identity/candidate
guards. All18 unprivileged guard tests pass. Moving that harness into source is
not represented as another execution of the complete native matrix.

## Independent review and corrected boundaries

Separate reviewers challenged package authenticity/hooks and runtime/RPC/recovery.
All reported witnesses were closed and independently rechecked. Corrections include:

- Public receipt permissions independent of restrictive caller umask.
- Actual APT source/config filename selection, effective authority removal,
  scoped multisignature handling and named post-invoke ordering.
- Resumable initial owner activation distinct from an established stopped engine.
- Completion of hooks on every successful offline-upgrade continuation.
- Atomic bootstrap uncertainty/dispatch evidence, consistent callback-error
  rollback and real startup reconciliation of explicitly undispatched work.

The latter has23 real runtime-driver→SQLite completion/reopen scenarios across
fake, comet and owned-key boots. Both before-commit and commit-with-lost-ack
errors, pre-dispatch crashes, real create failures, exact replay, repeated Stop,
retry and prevention of duplicate bootstrap are covered. Legacy, contradictory
or possibly dispatched states stay fenced. No uncertainty is cleared merely
because a daemon listing is empty.

## Remaining qualification boundaries

Native package tests do not qualify every Docker27.2–29.x/Vere/distro combination.
The new layer intentionally reports unqualified stacks honestly. RPM-family
providers, arbitrary native configuration layouts and full runtime qualification
of new distribution/engine profiles remain separate work. No custom engine or
package distribution is introduced, and no release, commit or deployment is
claimed by this development record.
