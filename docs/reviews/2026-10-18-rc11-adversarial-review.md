# Omabit 0.1.0-rc.11 adversarial release review

Date: 2026-10-18. Deliverable: architecture assessment, adjudicated findings,
and focused remediation plan.

Follow-up: [source remediation and qualification record](2026-10-18-rc11-remediation.md).
This document retains the original assessment of the published RC.11 baseline.

## Release decision

**Recommendation: do not promote RC.11 to stable 0.1.0 yet.** The architecture
has sound foundations, but interruption recovery does not consistently preserve
filesystem identity, some cross-component identity/replay contracts fail, and
the required restart-safe upgrade path remains incomplete.

The review identified **29 implementation findings: six High, 22 Medium, and one
Low**, plus five separately described implementation/qualification gaps. These
are severity assessments of demonstrated behavior and its prerequisites, not
CVSS scores. They are not all newly introduced by RC.11.

The most consequential findings are:

| ID | Severity | Finding |
| --- | --- | --- |
| RC11-01 | High | Interrupted deletion can recursively delete a replacement pier that fails the recorded identity check. |
| RC11-02 | High | Incoming handoff retry can recursively delete an unrecorded destination pier. |
| RC11-03 | High | A desktop storage-move draft can be retargeted to a different runner after selection changes. |
| RC11-04 | High | A maintenance intent proven never dispatched can permanently block unrelated runtime admission until state is repaired. |
| RC11-05 | High | An update's idle snapshot races with new work; shutdown can interrupt accepted maintenance and leave recovery fences. |
| RC11-06 | High, conditional | Automatic subordinate-ID allocation can overlap NSS-only host identities. |

Also prioritize wrong-runner forwarding through the SSH bridge, stranded
handoffs/imports, incomplete handoff request receipts, and historical creation
replay through the collector. These are Medium findings with direct implications
for the published identity and upgrade contracts.

Full triggers, evidence, source locations, limitations, fixes and regression
criteria are in the [findings catalogue](2026-10-18-rc11-findings.md).

## Baseline and method

- Published release: [v0.1.0-rc.11](https://github.com/thelifeandtimes/omabit/releases/tag/v0.1.0-rc.11).
- Published binary source: `abfff0529a19adc02cf9f4b93eeef78c645dfa8f`.
- Reviewed checkout: `f141360c7563e2069da7411662d2ba6208d512fb`, initially clean.
- The two subsequent commits change documentation and
  `native/bootstrap/release.json`; tracked application implementation is identical.
  Native distribution was reviewed with the updated RC.11 pins. Published assets
  were also checked against release metadata and those pins.
- Contracts: protocol version 1, schema 19, Linux/AMD64, existing-owner-user
  rootless deployment. The protected-ship and stable-upgrade policies were read.

The review adapted the evidence-constrained reviewer/critic protocol in
[Adversarial Review](https://arxiv.org/abs/2608.18167). Seven reviewer agents
inspected separate implementation areas. Seven independent critic agents then
audited their findings, inspected callers and safeguards, and ran or extended
isolated witnesses. Reviewer responses resolved material disagreements and
critic-discovered additions. The coordinating agent assessed architecture,
cross-component implications, release evidence and prioritization.

Critics used `AGREE`, `DISAGREE_EVIDENCE`, or `DISAGREE_CONCERN`; agreement was not
the acceptance criterion. Source evidence and reachable behavior were. Important
outcomes of that process:

- Pruning with recognized lock markers is confirmed, but live-writer corruption
  is not; its severity was reduced to Medium.
- Plain-HTTP pill acceptance requires an explicitly selected CLI source; the
  HTTPS desktop default is not vulnerable to ordinary HTTP interception.
- The SSH identity defect requires a runner replacement between checks; it is
  not an observer privilege escalation or an indefinitely unchecked CLI session.
- The import form has reset-based escape paths; the report does not call it
  permanently irrecoverable.
- Missing destination-directory sync ordering is separated from an unperformed
  power-loss experiment.
- The critic added a named-storage-root UI mismatch and extended handoff receipt
  and bridge cancellation findings; the respective reviewers verified them.

All review agents used the same available model family. Independent contexts and
concrete witnesses reduce unchecked agreement, but do not constitute independent
human audits or model diversity.

## Architecture assessment

### Implemented system and authority boundaries

```text
Omarchy QML/JS -> omabit / urbit-collector
                         |-- private local RPC ------------------+
                         |-- pinned SSH -> restricted RPC bridge +-> urbit-runner
                         |                                            |-- SQLite journal
                         |                                            |-- rootless Docker/Vere
                         |                                            +-- owner-user piers
                         +-- origin-bound Eyre -> ship applications
                         +-- private credential storage

OS-authorized installer -> owner profile, service units, restricted SSH,
                          verified helpers and management executable
Release tooling -> binary/UI artifacts -> pinned native cache or snapshot install
```

The runner is the lifecycle authority; desktop/collector disconnection does not
cancel accepted work. Collector inventory and application authentication are
separate. Runtime processes can outlive a management restart. The optional web
gateway, peer-monitoring system and social/gossip application described in the
architecture document are planned surfaces, not implemented services assessed
as production-ready here.

### Foundations worth retaining

1. **Appropriate privilege separation.** There is no public runner management
   listener. Local peers derive identity from Unix credentials; SSH delegates
   authenticated key identity through a restricted command. Operator/observer
   enforcement and retained-session revocation have meaningful tests.
2. **Explicit persistence and concurrency contracts.** SQLite transactions,
   actor-scoped request receipts, configuration revisions, a host work gate,
   immutable launch settings and interruption fences are appropriate primitives.
3. **Conservative storage and handoff design.** Recorded directory identities,
   no-replacement renames, validated archive entries, content verification,
   signed commitments and permanent source fencing provide substantial protection
   on the ordinary paths.
4. **Good remote/secret boundaries.** SSH host-key verification, bounded responses,
   Eyre origin binding, redirect refusal for sessions, private credentials, secret
   stdin handoffs and independently bounded host polling are implemented.
5. **Useful separation of intent and observation.** Desired state, observed state,
   health, configuration/applied revisions and recovery-required status are not
   collapsed into a misleading single running flag.
6. **Real upgrade evidence exists.** The recorded RC.10-to-RC.11 update preserved
   a running ship's container/processes, identity, configuration and history.
   That is valuable positive evidence for that specific path.

The owner-user model intentionally shares filesystem/engine authority with the
owner's other processes. Rootless execution is not isolation from malicious code
running as that same owner. This review did not treat that documented choice as
a vulnerability, or find an observer-to-operator bypass in inspected dispatch.

### Architectural weaknesses behind the findings

**A. Safety proofs are not carried across durable phases.** Initial admission
often validates the correct object, but restart continuation validates only its
parent or a weaker state. A persisted mount identity cannot authorize deletion
of any future child. Conversely, queued work is sometimes treated as if effects
may already have occurred. The result is both destructive continuation and
unnecessarily stranded registrations.

**Recommendation:** define each destructive operation's phases with explicit
evidence: intended object identity, whether dispatch occurred, observed effect,
durability barrier, and permitted recovery. Revalidate identity at every effect
boundary. Treat proven-not-dispatched, dispatched-but-uncertain, and completed
states distinctly. Add crash-point matrices around every persistence callback.

**B. Identity binding is lost between layers.** UI selections, the outer SSH
connection and the bridge's inner Unix connection can each be individually valid
while the combined operation targets the wrong runner. Destination revision 1
is also assumed despite deliberate tombstone-based name-reuse revisions.

**Recommendation:** carry an immutable operation target through the whole path:
runner ID/mode, instance ID/revision, request ID, and applicable storage or
transfer evidence. Revalidate it at each transport reconnection and UI dispatch.
Use authoritative response/snapshot revisions rather than inferred initial values.

**C. The generic mutation contract has endpoint exceptions.** Most mutations use
actor-scoped receipts; handoff reservation and activation bypass them. The CLI
also rejects historical input before the runner can honor preserved receipts.

**Recommendation:** make mutation acceptance/replay a shared transactional
primitive, with endpoint-specific validation after replay lookup. Exercise
contracts through actual collector entrypoints, not only store methods.

**D. Management availability has excessive failure coupling.** A canceled port
suggestion can set the global state-store failure latch; an undispatched utility
record blocks unrelated ships. These mechanisms are necessary for genuine
uncertainty, but error classification and evidence need to be narrower.

**Recommendation:** distinguish request cancellation, validation conflict,
runtime uncertainty and journal failure. Keep the host-wide safety fence for
unresolved effects, with a supported recovery path for proven non-effects.

**E. Update is not yet a transaction spanning admission, files and readiness.**
An installer-only lock cannot exclude scheduler/API work. Atomic replacement of
one executable does not provide recoverable activation. A responding runner ID
does not prove that the newly installed service is the responder.

**Recommendation:** introduce runner-owned update quiescence, verified deployment
generations, a durable activation journal and readiness bound to the intended
service/build. Define rollback relative to completed state migrations. Apply the
same recoverable-publication principle to desktop snapshots and first-created
host configuration files.

These recommendations preserve the current component split. The evidence calls
for stronger shared contracts and recovery state machines, not a wholesale rewrite.

## Coverage

| Pair | Implementation coverage |
| --- | --- |
| R1/C1 | Managed journal/server/schema, lifecycle/bootstrap, startup/reboot, runtime settings, routines, resources, runtime execution and rootless checks |
| R2/C2 | Pier storage/copy/metadata, epochs/pruning, import/recovery, storage actions and cross-host handoff |
| R3/C3 | Protocol, local RPC, client transports, restricted SSH bridge, grants/authorization, runner entrypoints and simulator contracts |
| R4/C4 | Collector CLI/inventory/logs, credentials, Eyre/session/channel handling, pill provisioning, theme synchronization/migration |
| R5/C5 | Host installation/preflight, onboarding, owner profiles, subordinate IDs, service templates, rootless authority and activation |
| R6/C6 | Release/plugin packaging, native bootstrap/cache/pins, snapshot installation, CI, provenance and update qualification |
| R7/C7 | QML/JS creation, storage/import/recovery, maintenance, policy, access, monitoring/logs and runtime-setting workflows |

Coverage means these implementation surfaces were examined; it is not a claim
that every execution path or every shipped dependency was exhaustively proved.

## Verification and limits

The coordinating agent ran these successfully against the checkout:

```bash
go test ./...
go vet ./...
go test -race ./internal/managed ./internal/runtimequalify ./internal/client ./internal/access ./internal/collection ./internal/piercopy ./internal/pierstore ./internal/piermeta
CGO_ENABLED=0 go build -trimpath -o /tmp/opencode/omabit-rc11-review-build/ ./cmd/... ./backend/cmd/...
node scripts/test-panel-model.js
node scripts/test-runtime-settings-model.js
python3 -B scripts/test-panel-install.py
python3 -B scripts/test-native-plugin.py
go mod verify
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

Toolchain: `go1.27.0-X:nodwarf5 linux/amd64`. Vulnerability tool resolved to
`golang.org/x/vuln v1.9.0` and reported **“No vulnerabilities found.”** This is a
Go-source/database result on the review date, not an OS-package, Docker-image,
Vere, pill-content or complete published-binary vulnerability assessment.

Reviewers and critics additionally ran isolated filesystem/SQLite/RPC witnesses,
mock-engine runtime tests, CLI tests, release-bundle checks and extracted UI
JavaScript. Several critics extended witnesses to actual store reopen, managed
server acceptance and authoritative backend admission. The standard tests pass
while those witnesses demonstrate missing failure coverage.

**Evidence levels matter:** replacement deletion was demonstrated on disposable
directories; wrong-runner API acceptance was demonstrated with real temporary
managed journals; UI behavior was traced/executed through extracted JavaScript,
not a mapped live desktop. Host-install witnesses use injected command results;
actual systemd/NSS failures were assessed through code and supporting contracts.

There were no live pier operations, service updates, real credential access,
privileged installation, live OpenSSH integration or power-loss experiments in
this review. The reported defects do not imply an observed loss of the user's
ships. Existing live qualification records were read as historical evidence.

## Focused remediation sequence

| Work package | Scope | Acceptance gate |
| --- | --- | --- |
| 1. Data-preserving continuation | RC11-01, 02, 09, 10; G01 | Interrupt at every durable boundary; replacement/locked objects survive; intact imports have an explicit completion path; destination publication precedes source removal. |
| 2. End-to-end selection and handoff contracts | RC11-03, 07, 08, 14, 15, 28 | Cross-runner selection/reconnect tests emit zero wrong-target mutations; reused-name handoff completes; actor-scoped reserve/activate receipts and exact historical CLI replay survive restart. |
| 3. Bounded lifecycle recovery | RC11-04, 11, 12, 13, 29 | Never-dispatched intents can be resolved; canceled reads do not disable control; queued creation can retry; capacity is rechecked at launch; bridge I/O exits on shutdown. |
| 4. Transactional installation/update | RC11-05, 06, 20–24; G02 | API/scheduler drain exclusion; safe subordinate-ID policy; atomic first publication; crash-resumable generations and compatible rollback; verified active process/build; reproducible tracked-input packaging. |
| 5. Credentials, artifacts and UI intent | RC11-16–19, 25–27; G03 | Cleanup intent survives failed CAS/logout; HTTPS and release digest enforced; terminal output sanitized; valid bounded theme state; policy toggles preserve downtime choices; receipts remain request-bound. |
| 6. Frozen release qualification | G02, G04, G05 and regressions above | Repair recovery fixture, run supported-version/client matrix and interruption suite, then controlled disposable-ship/live desktop/upgrade acceptance on the frozen artifacts. |

Work packages are intentionally separable; the data-preserving continuation and
selection fixes should lead. A fix that changes persistent recovery evidence or
receipts must follow the upgrade policy, including compatibility with already
persisted interrupted states. Clearing a schema marker, deleting a database, or
re-enrolling a retained pier is not an acceptable upgrade remedy.

Before stable promotion, close all High findings, core identity/replay/recovery
contract findings and the stable upgrade/durability gates. Resolve remaining
Medium findings or record an explicit, narrowly scoped support decision with
accurate user-facing behavior. The dormant legacy-theme surface, for example,
should not silently enlarge the release promise.

Finally, publish a current normative contract/support matrix separate from the
historical checkpoint narrative: supported schemas/releases, client/UI feature
compatibility, update command, recovery paths, rollback restrictions, artifact
trust roots, and which live scenarios were actually qualified. Historical
documentation contains assumptions such as destination revision 1 that no longer
hold for reused names; that divergence contributed to an implementation defect.
