# 0.4.0 release preparation

In progress; this is not a release qualification or publication record.

The user accepted RAM/CPU controls and Agent Help launch on2026-11-12, delegated
app-development and cross-runner testing, and authorized release qualification
and an isolated0.3.7→0.4.0 rehearsal. Live public-provider configuration remains
for the user to test. No production or human-review ship is a rehearsal target.

The user explicitly authorized a temporary48GiB aggregate experiment allowance.
Return below40GiB by retaining compact evidence and retiring only newly generated
disposable outputs. Both native DEV roots and historical manual-review state stay.

## Checks and findings before artifact freeze

- The first fresh native publisher passed default gates, creation, Unix/Clay
  alerts, scoped commits, publication, frontend upload, lost-ack read-only recovery
  and exact replay. The first receiver passed source/frontend delivery. A changing
  second receiver returned the expected unavailable status during asynchronous
  installation; the fixture now polls only that read with a bounded deadline.
  No install or uncertain mutation is retried. The fresh three-ship rerun passed,
  including both receivers' exact initial and updated source/frontend bytes.
- The actual Theme and DMs receiver helpers, private Secret Service and running
  consumers pass same-origin renewal, alternate-URL replacement and lost-reply
  recovery. This assembly uses synthetic runner/Eyre peers; the upgrade guest adds
  the real managed-runtime, SSH and network-namespace scenario.
- The full source suite exposed stale historical-test schema targets52/56.
  Both now explicitly name59, preserving the original RC12 qualification and
  historical reader window. Full managed race coverage is in progress, with the deadline and remaining
  inventory accounted for below.
- Two filesystem ownership/ACL tests cannot use the single-UID Bubblewrap user
  namespace. Their unchanged binaries ran as the ordinary guest owner on the
  VM filesystem; both passed there with race instrumentation. This is an
  environment-specific rerun, not a waived gate.
- The upgrade harness now checks schema versions from both exact manifests,
  actual old-writer refusal, and all experiment storage roots. It retains the
 4GiB per-guest scratch limit and stops its own scope on measurement failure.
- Candidate01 at `f35e602` passed archive/provenance, independent reproducibility,
  all172 help entries, full-panel/QML and cold/hot native desktop checks. The
  completed fresh VM rehearsal preserved running/stopped intent, container/engine
  processes, identity, history and access across the actual0.3.7→0.4.0 update.
  Both real receivers reconnected over a separate private client network under
  standing approval, without authenticating to a wrong ship or remote localhost.
- Early harness runs exposed cross-filesystem staging, downloaded executable mode
  and the pristine guest's loopback SSH/private-network firewall assumptions.
  The corrected fixture preserves those host/guest boundaries and uses only an
  exact private test link. These are retained failed qualification attempts.
- The unsharded managed race run reached its30-minute deadline after285 passing
  top-level tests with no assertion failure. The remaining executable inventory
  is being completed in four isolated groups; no unfinished test is counted passed.
- The next frozen candidate adds an actual installed-runner/SSH/agent-CLI check
  for blocked and explicitly permitted desk creation, commit and source publication.

Evidence is retained privately under `/tmp/omabit-040-release-qualification`,
`/tmp/omabit-desk-probe-release040-*` and
`/tmp/omabit-real-reconnect-release040-01`. Final compact results and exact hashes
must replace pending claims before stable release qualification.

## Remaining gates

The clean second candidate is `88c95d795b163de98e42691d885f480018e7616c`. Its
archive/provenance and actual cold/hot desktop activation pass. Complete the
exact-candidate service, remote-receiver and installed-runner CLI scenarios, all
managed race coverage and [source CI](https://github.com/thelifeandtimes/omabit/actions/runs/43331849506).
Then attach qualification, prepare immutable draft artifacts and distribution
pins, and review the release PR before publication. No stable artifact, installed
runner or distribution pin has yet been changed by this preparation.

## Installed-runner SSH blocker and correction

Candidate02's actual CLI created its new desk, then source staging failed because
SSH forwarding dropped the transfer descriptor and used ordinary RPC body limits.
The local Unix-socket tests did not cover this boundary. The bridge now forwards
only a validated bounded descriptor and streams a bounded body, pins the local
runner and capability, and closes after one upload. Desk RPC deadlines match the
public client; ordinary framing and generic requests retain their short bounds.
Regression tests cover known/chunked transfers, malformed framing, observer and
missing-capability refusal, replacement identity/role, cancellation and deadlines.
The access/client/collection suites pass with race instrumentation. Candidate02
is not release-qualified; the corrected source requires a fresh candidate.

All520 managed tests are now accounted for:498 passed and22 explicitly guarded
native/VM tests skipped in source capsules. The fresh native publisher/receivers
and upgrade VM provide their separately stated runtime evidence; these skips are
not represented as tests of every historical external-runtime scenario.
