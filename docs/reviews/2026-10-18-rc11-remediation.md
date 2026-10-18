# RC.11 review remediation and qualification record

This records the source changes following the
[architecture review](2026-10-18-rc11-adversarial-review.md) and
[29-finding catalogue](2026-10-18-rc11-findings.md). The reviewed release was
`abfff052`; implementation started from `f141360`. The changes are an uncommitted
successor to RC.11, not modifications to the published RC.11 assets.

## Result

**All 29 implementation findings have source fixes and maintained regression
coverage.** Four independent adversarial reviews of the implementation found
additional boundary failures; their fixes were independently rechecked before
the final combined tests. No actionable issue remained in those rechecks.

The component split, visible desktop layout, controls and navigation are retained.
The intended creation, lifecycle, import, handoff, maintenance and access workflows
remain. Changes correct their target binding, failure/retry semantics and safety
contracts. No live installation, service, credential store or ship was changed.

The five gap items also have implementation or test work, with **release
qualification still outstanding where described below**. Passing fixture checks
does not certify an unpublished artifact, actual power-loss behavior or a live
upgrade. Stable promotion remains a separate decision after those gates.

## Finding-to-fix matrix

Paths below identify principal maintained regression suites; additional tests
exist beside the affected implementations.

| Finding | Implemented behavior | Principal regression coverage |
| --- | --- | --- |
| RC11-01 | Destruction is bound to recorded child/object identities, a durable removal manifest, anchored parent descriptors and lock checks. An empty-mount registration cannot authorize an unrecorded pier. Partial removal resumes only with sufficient retained evidence. | `internal/pierstore/rc11_removal*test.go`; `internal/runtimequalify/rc11_storage*test.go` |
| RC11-02 | Transfer-specific, identity-recorded staging and no-replacement publication; interrupted attempts are retained, and retries never blindly remove destination contents. | `internal/runtimequalify/rc11_storage_regression_linux_test.go` |
| RC11-03 | Immutable move targets bind runner, instance, revision and source evidence; selector/inventory changes invalidate drafts and dispatch rechecks. | `scripts/test-rc11-panel-state.js` |
| RC11-04 | Never-dispatched container intents are resolved without nonexistent-container inspection. A report-only maintenance phase now carries durable non-dispatch evidence through Stop and restart. | `internal/runtimequalify/rc11_maintenance*test.go`; `internal/managed/rc11_lifecycle_reconciliation_test.go` |
| RC11-05 | Local-root quiescence atomically excludes API, scheduler, startup and bulk admissions; busy work is refused without cancellation. Fence survives restart. | `internal/managed/rc11_management_update_test.go`; `internal/hostinstall/rc11_*test.go` |
| RC11-06 | Automatic sub-ID allocation requires authoritative local identity sources; external NSS requires explicitly reserved, root-owned range evidence. | `internal/hostinstall/rc11_activation_test.go`; hostsetup tests |
| RC11-07 | Every bridge-local handshake checks retained runner ID/mode before forwarding, including bulk and pill traffic; same-identity restarts remain usable. | `internal/access/bridge_binding_test.go` |
| RC11-08 | Handoff retry/activation uses the actual destination revision bound to the transfer, including deleted-name reuse. | `internal/client/handoff_revision_test.go`; `internal/managed/handoff_revision_test.go` |
| RC11-09 | Import completion publishes atomically. Explicit recovery handles historical initialized/enrolling and unprepared/enrolling boundaries after full evidence validation, including missing private workspaces. | `internal/runtimequalify/rc11_storage_boundary_linux_test.go`; managed import-recovery suites |
| RC11-10 | Native-lock checks at pruning admission and between destructive steps; partial work retains uncertainty when activity appears. | `internal/runtimequalify/rc11_storage_regression_linux_test.go` |
| RC11-11 | Request cancellation/deadline errors do not poison the global state-store failure latch. | `internal/managed/rc11_management_update_test.go` |
| RC11-12 | Queued first Start retains its interrupted receipt without inventing provisioning uncertainty; existing real fences remain authoritative. | `internal/managed/rc11_lifecycle_reconciliation_test.go` |
| RC11-13 | RAM and relevant filesystem capacity are rechecked after preparation and immediately before runtime dispatch; refusal remains retryable. | `internal/runtimequalify/rc11_start_admission_linux_test.go` |
| RC11-14 | Reservation and activation record atomic actor/request/fingerprint audit receipts. Legacy reservation replay returns current state of the exact accepted transfer; its original audit remains immutable. | `internal/managed/handoff_receipts_test.go`; `handoff_rc11_compatibility_test.go` |
| RC11-15 | Guarded replay-only creation reaches historical actor-scoped receipts through CLI, transport and runner. Capability is checked on each connection; misses cannot create new small-loom registrations. | `internal/managed/` frozen RC.10 replay tests; collector `rc11_replay_logs_test.go`; client creation-replay and bridge tests |
| RC11-16 | Pending credential references are durable before Save, cleanup deadlines are independent, and logout can drain a full reference journal while preserving concurrent sessions. | `internal/collection/credential_journal_test.go`; collector `rc11_credentials_test.go`, `rc11_logout_capacity_test.go` |
| RC11-17 | Network pills and all redirects require HTTPS; explicit local/custom HTTPS sources remain supported. | `internal/bootpill/source_security_test.go`; collector `rc11_pill_test.go` |
| RC11-18 | Text logs sanitize terminal control and Unicode formatting characters at the local output boundary. | collector `rc11_replay_logs_test.go` |
| RC11-19 | Whole post-update theme state is validated before replacement; migration preflights count/size and preserves cleanup intent and resumable progress. | `internal/omabittheme/rc11_state_test.go` |
| RC11-20 | Readiness binds the intended service invocation/PID, kernel socket peer, executable digest and stable runner identity; foreground responders do not satisfy it. | hostinstall activation/offline/retry suites; `internal/client/rc11_management_peer_test.go` |
| RC11-21 | First publication is staged, validated, synced and no-replace, including recoverable SSH host-key preparation. | hostinstall activation/publication tests |
| RC11-22 | Packaging exports tracked commit blobs, validates the selected Go executable, and builds in a controlled environment. Ambient overlays/workspaces/ignored content cannot be certified as committed source. | `scripts/test-release-source.py --build` |
| RC11-23 | Snapshot installs use destination locking, durable recovery records, atomic publication and retained prior generations; ordinary retry resumes interruptions. | `scripts/test-panel-install.py` |
| RC11-24 | Publication checks the actual retired tree, preserves editor changes, serializes installers and retains backups even for late writes through old descriptors. | `scripts/test-panel-install.py` |
| RC11-25 | Enable/disable changes only `enabled`, preserving downtime and resume permissions. | `scripts/test-rc11-panel-state.js` |
| RC11-26 | Definite inspection refusals are terminal; ambiguous submissions remain request-bound. Completion opens only after QML bindings settle and only once for the matching target. | JS state tests; `scripts/test-rc11-panel-offscreen.py --require-qt` |
| RC11-27 | Import progress retains its own request association. A bound success remains terminal after instance edits/deletion/recreation; unrelated receipts cannot unlock submissions. | JS state tests; offscreen QML tests |
| RC11-28 | Existing root/path controls are synchronized from the source; conflicting prefixes cannot silently override the selected destination root. | JS state tests; offscreen QML selector tests |
| RC11-29 | Session cancellation interrupts reads and writes; framing progress is bounded without expiring healthy idle watches. Production SSH descriptors use the interruptible path. | `internal/access/bridge_cancel_test.go`; runner `ssh_cancel_test.go` |

## Gap closure and remaining qualification

### G01 — Cross-filesystem durability

The destination namespace is synced before authorizing source removal. The source
removal manifest is captured before copy; after publication barriers the source
and destination are remeasured against the verified copied content and expected
filesystem objects. New source bytes or changed destination evidence cause refusal
before any source unlink. Fault injection covers sync ordering, each failed barrier,
post-copy edits and changed parent/path evidence.

**Remaining:** controlled power-loss qualification on supported filesystems.
These tests establish explicit ordering and conservative failure behavior, not
every filesystem's physical persistence guarantees.

### G02 — Upgrades and rollback

Source state is now **schema20**, preserving accepted schema11–19 records and
fencing old binaries from ignoring new safety contracts. Quiescence-aware updates
use verified generations, a durable activation journal, service/process readiness
and migration-compatible rollback. Interrupted acquire, stop, start, verification
and release have repeat-command recovery. A live released manager is not stopped
merely to replay an acknowledgement.

Published schema19 runners lack quiescence. The explicit
[one-time offline migration](../management-offline-upgrade.md) verifies a stopped
manager, holds the actual runtime ownership lock, inspects state as its owner,
commits migration and admission fencing transactionally, then activates the new
generation. It neither resets the database nor re-enrolls piers. Published schema19
binaries are not rollback targets after schema20 commits.

Twelve sanitized fixtures were generated by compiling the original RC.1–11 writers
and the schema18 preview. Regeneration was byte-for-byte reproducible. The real
schema20 migrator preserves records and receipts; all twelve original readers
refused its resulting databases without modifying them. A frozen RC.10 receipt
fixture exercises actual new-collector small-loom replay, and the hash-verified
published RC.11 collector passes same-request handoff continuation after restart.

**Remaining:** original writers for schemas11,12,14,17 are unavailable in the
published history; the manifest records this honestly. Available original-writer
coverage is schemas13,15,16,18,19, supplemented by existing synthetic tests. Real
host privileged isolation, systemd activation and live disposable-ship upgrade
qualification of the final frozen artifacts remain required. The explicit offline
host path supports existing owner-user schema19 installations; historical database
readability does not imply automatic migration of every historical OS deployment.

### G03 — Pill provenance

The built-in Recommended Apps source, including its old S3 alias, resolves to the
versioned RC.11 artifact with mandatory expected SHA-256 and size checks before
upload. Explicit custom sources remain available and may supply their own expected
descriptor. Native binary pins continue to identify the published original RC.11
tools. The accepted source/release channel remains the artifact trust root;
independent publisher attestation is not implied by a content hash.

### G04 — Desktop recovery qualification

The generated collector fixture takes explicit capabilities and asserts its
prerequisites. The actual recovery component, refusals/retries, stopped receipt,
selection handling and asynchronous completion are exercised under isolated Qt,
with display/session variables removed. The offscreen suite ran without skipping.
CI now has a dedicated Ubuntu Qt job using `--require-qt`.

**Remaining:** final packaged/native-wrapper acceptance in a live Quickshell
session. Offscreen execution does not certify compositor focus, clipboard behavior
or stock Omarchy cold-update integration.

The subsequent [RC.12 desktop preparation run](2026-10-18-rc12-desktop-qualification.md)
passed eight mapped source/native-wrapper cases on sourceport, including recovery,
log-watch, onboarding and normal/special-workspace focus. It records exact source
hashes and supplies an archive-aware frozen qualification runner. Final RC.12
artifact/pin acceptance, real clipboard expiry and cold-update integration remain
separate gates.

### G05 — Legacy Ames publication

An explicit operator-only [publication operation](../legacy-ames-publication.md)
handles stopped legacy live enrollments, including unpublished comet handoffs.
It validates identity, provenance, lock/stopped state and port conflicts, records
the prior contract and actor-scoped receipt, and leaves the ship stopped. The
next explicit Start resumes the existing pier without bootstrap; required identity
verification still precedes network publication. Opening old state has no network
side effect.

**Remaining:** live network acceptance of a migrated disposable ship on the frozen
candidate. No user pier or live networking was changed during implementation.

## Final verification

The final combined Go checks passed after the last implementation corrections:

```bash
OMABIT_RC11_COLLECTOR=/tmp/opencode/rc11-published-assets/urbit-collector-linux-amd64 go test ./...
go vet ./...
OMABIT_RC11_COLLECTOR=/tmp/opencode/rc11-published-assets/urbit-collector-linux-amd64 go test -race ./internal/managed ./internal/runtimequalify ./internal/client ./internal/access ./internal/collection ./internal/piercopy ./internal/pierstore ./internal/piermeta ./internal/hostinstall ./internal/hostsetup ./internal/omabittheme ./backend/cmd/urbit-collector ./backend/cmd/urbit-runner
python3 -B scripts/test-historical-upgrades.py --check-generated
CGO_ENABLED=0 go build -trimpath -o /tmp/opencode/omabit-rc11-review-build/ ./cmd/... ./backend/cmd/...
/tmp/opencode/omabit-rc11-review-build/urbit-runner state-schema
go mod verify
go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...
go run github.com/rhysd/actionlint/cmd/actionlint@latest .github/workflows/checks.yml
git diff --check
```

The runner reports **20**; module verification passes; govulncheck reports **No
vulnerabilities found**. This is Go-source/dependency scanning, not a whole-host,
Vere-image or pill-content security certification.
Workflow validation also passed with actionlint v1.7.12. The new hosted CI jobs
have not been executed remotely as part of this local implementation session.

Also passed on the final relevant UI/installer/packaging implementations:

```bash
node scripts/test-panel-model.js
node scripts/test-runtime-settings-model.js
python3 -B scripts/test-import-recovery.py --static
python3 -B scripts/test-rc11-panel-offscreen.py --require-qt
python3 -B scripts/test-panel-install.py
python3 -B scripts/test-native-plugin.py
python3 -B scripts/test-release-source.py --build
```

Installer tests include real temporary-filesystem SIGINT/SIGKILL recovery,
fsync/rename failures, concurrent edits, competing installers and interrupted
A→B recovery followed by requested package C. Host service behavior is injected;
no systemd or real installer operation was performed against the user's host.

A receipt-test race discovered during combined testing was traced to fixture
state writes that failed to advance `RuntimeRevision`, plus comparing legitimate
observation bookkeeping. Deterministic stale-observation checks confirmed production
activation already fenced correctly. The fixed test suite passed 30 repetitions,
then 30 race-enabled repetitions, followed by the combined suite.

The release-source test produced byte-identical binary bundles from isolated
exports of committed **`f141360`**, including contaminated-environment negative
tests and the bundle smoke suite. It separately exercises the current installer.
Because the remediation is uncommitted, this is verification of the packaging
mechanism, **not certification of a new remediation release**. After an authorized
integration commit, build and qualify that exact source and update release pins.

## Preserved limits

- Same-owner malicious code is not a security boundary. Descriptor anchoring
  prevents replacement-ancestor redirection, but Linux does not provide an atomic
  conditional unlink by expected inode for a final basename changed by another
  process at precisely the syscall boundary. An already-authorized unlink can
  finish against the original detached directory; the replacement tree is preserved.
- Historical ambiguous maintenance lacking durable dispatch evidence remains
  fenced. The implementation does not manufacture proof or clear uncertainty to
  make an unsafe legacy record runnable.
- Previous installer generations and partial transfer staging are retained for
  recovery rather than automatically purged. This consumes disk and preserves
  user/editor data; it is not a backup or unattended restore mechanism.
- Fresh initialization, already-running ships, historical settings, signed transfers,
  credentials and original request fingerprints are not rewritten as a shortcut
  to upgrade compatibility.
