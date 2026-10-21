# 0.2.0 remote onboarding development verification

Implementation branch: `feature/0.2.0-remote-onboarding`, target version
`0.2.0-edge.1`, with unqualified development channel metadata. This work carries
the prior UI polish and introduces Go onboarding/access code; it is not a UI-only
stable patch. Published stable0.1.0 pins remain the distribution baseline.

## Real SSH and disposable-host evidence

Used new QEMU/KVM overlays over the previously authenticated Arch cloud image
`Arch-Linux-x86_64-cloudimg-20261015.611680.qcow2`, SHA-256
`08754e28d91b1680086c6211ef4df13559e766fff7cdb4006489b14b5d281f08`.
The backing image stayed read-only. Fixture administrator keys and host keys were
separate from the user's real credentials. The administrator key was encrypted;
the worker ran without an SSH agent and authenticated through a real PTY.

1. **Clean host installation:** the final clean-overlay pass installed missing
   native Docker29.9.1, containerd, runc, RootlessKit and slirp4netns packages using
   Pacman, prepared the pinned rootless helper as the selected non-root owner,
   installed the owner-user runner, added the restricted management key to normal
   SSH, verified fresh unattended RPC/inventory, and saved enrollment. No manual
   SSH alias/key or port22222 exposure was needed inside the guest.
2. **Existing stable runner:** installed the actual published stable0.1.0 runner
   (`72bd87cdbb56b800730200e3f5c5c16d677355949ede3563f57dba5c9d9e1a37`)
   as the guest's management generation and booted a disposable `~zod` fake.
   Pairing another desktop preserved that binary, manager/engine/container process
   identities and start times, mounts, ports, instance configuration and receipts.
3. **Repeat pairing:** reused the same key/attempt and preserved the same runtime
   observations and management generation, without duplicate enrollment.
4. **Restricted key:** real shell execution and TCP forwarding requests were denied.
5. **Revocation:** an already-open collector watch lost authorization; new SSH
   authentication failed afterward. The first desktop's grant and the running fake
   remained intact. Repeated revocation completed without reauthorization.
6. **Cancellation and resume:** declining remote approval produced a recoverable
   failed attempt; retry retained its key/fingerprint and completed enrollment.

Evidence roots:

- `build/remote-onboarding-020/`: encrypted-key PTY transcripts, pairing/repeat/
  revocation snapshots, denial checks, cancellation/retry and source tool binaries.
- `build/remote-onboarding-020-clean/`: a final independent clean-cloud installation
  with the corrected implementation and real native package installation.

Both guests are powered off. The disposable fake was stopped through its normal
operation. The host's real runner/engine and live container observations matched
the controller's before/after records; the cloud backing image hash was unchanged.

The first lab exposed native OpenSSH10.5's directive capitalization, owner-only
rootless preparation, and a private-journal/public-evidence directory interaction.
Those implementation issues were corrected before the independent clean-host pass.
The bounded VM supervisor also stopped one run when host memory headroom fell below
1GiB. The paused guest was resumed to stop the fake and finish recovery checks; this
was not recorded as a successful uninterrupted test run.

## Source and UI coverage

Go regression coverage exercises target/key/installer/profile binding, immutable
installer retention, lost worker detection, malformed state, supported SSH policy,
symlink/hardlink/permission refusal, preservation of unrelated keys, role-change
refusal, idempotent pairing/revocation, native candidate policy and bounded output.
The actual Qt/Quickshell wizard test checks ordinary alias entry, duplicate-action
gating, progress polling, completion and bound revocation using inert CLI fixtures.

## Limits

This is development verification, not a frozen stable0.2.0 release qualification.
APT dependency provisioning, custom SSH/bastion policies and the full release/OS
matrix require their own live qualification. The successful Docker29.9.1 fake smoke
is evidence for this test stack, not a blanket qualification of that engine family.
The backend continues to report qualification independently from compatibility and
authenticated package provenance.

## Local development desktop activation

Built the current source snapshot with the controlled development builder and
qualified the actual packaged UI/helpers through the bootstrap lifecycle suite.
Activated it in sourceport's default profile:

- Application: **0.2.0-edge.1 / DEVELOPMENT**.
- Label: `remote-onboarding-020`.
- Input snapshot: `20e1aa22e8a86fc28ec05b8a3c008eea4c15ad20ad11de4dd0c6eb19b4b170b7`.
- Release identity: `3802ef9cf701ac3352a449f580d27b8cf765349dc819c1e3d4a0085d5169891e`.

The desktop's three tools are the new development generation. The installed
sourceport host runner remains the actual stable0.1.0 executable. Live activation
settled at ready, observing one runner and two healthy running comets. Manager,
engine, ship and shell processes, mounts, configuration and operation history were
unchanged across activation. Evidence: `build/sourceport-remote-onboarding-020/`.

An additional real `omarchy launch terminal` check used a private XDG profile and
an owned, non-listening loopback port. The actual worker inherited the correct
profile, attempted SSH, and retained a resumable failure. Only that completed test
worker was closed; existing windows and focus were restored. This verifies the
terminal/profile handoff separately from the successful real VM SSH flow.

Local verification also passed `go test ./...`, `go vet ./...`, race checks for
client/access/collection/hostinstall/onboarding/remote setup, model tests, both Qt
offscreen suites, the mapped wizard/onboarding/full-panel tests, native/bootstrap
tests, and version/channel policy tests. Changes remain on the development branch;
this activation does not publish or promote a stable release.

## User acceptance and UI follow-up

The user subsequently confirmed that sourcebox was successfully added through the
new flow without issue. The observed desktop has two enrolled runners and three
ships. A follow-up UI-only bundle adds shared uppercase monospace card headings,
top-right log playback/refresh icons, overlay progress/error toasts, identity-bound
request failures in Connectivity & runtime, collapsed runner technical details,
and passive local SSH suggestions in an editable host dropdown.

First follow-up label: `sourceport-cards-and-hosts`; release identity
`8562d6388f7be366c714295b7355138514fc2f5b042f1767e37379306050e080`.
Input digest: `b5bdb93340151a745616128791e195459c93986bf5f42bbbe51a874a9e5ec982`.
All three Go tools were copied byte-for-byte from the preceding0.2.0 desktop.
No Go source or executable change was needed for this UI pass.

Host-discovery tests covered Include cycles, wildcard/hashed/revoked entries,
non-default ports, nonregular files, bounds and non-execution of command directives.
Real Qt/Quickshell checks verified custom host input survives suggestion loading,
selected ports are carried through, log buttons sit in the header, technical
sections toggle, and submission/error overlays leave the ship context's position
unchanged. Full mapped-panel and actual packaged bootstrap checks passed.
The screenshot run received a larger bounded timeout after its initial90-second
budget expired while progressing through the full capture sequence.

Before/after checks preserved both enrollments, SSH configuration/key files, all
three ship configurations/states, local processes and the installed host runner.
Evidence: `build/sourceport-card-polish/` and
`build/ui-polish-20261020/cards-verified-evidence/`.

## Comet guidance and compact pier management

A second UI-only follow-up is active in the default profile:

- Label: `sourceport-comet-and-piers` (**0.2.0-edge.1 / DEVELOPMENT**).
- Release identity: `ffff9bb803e37e5a339f10f9d24aa03b565caf170b6ef88915a028cf36dcb8c1`.
- Input digest: `ea53755abb93c19af918b7931621dd95ad3f72331086426c7ee1b9ca817ad31f`.

The comet form now has a matching identity heading and explains generated keys,
the long network identity versus a local name, and persistence when restarting the
same pier. The provisioning import form stays open, uses the explicitly selected
runner, and omits duplicated storage-page controls. Discovered piers and registered
locations use compact rows with full-path tooltips and inline actions. Deletion
still opens a runner/revision-bound review requiring the exact instance name.
Hidden import forms no longer clear the visible form's pending review.

Verification passed the full mapped-panel/CLI-audit suite, onboarding checks,
model/state tests, both Qt offscreen suites, and actual packaged bootstrap lifecycle
and profile-restart checks. The mapped assertions cover row-height limits, inline
actions, always-open import, no fallback to another runner, and retained typed
deletion confirmation. The JavaScript fixture now supplies the QML visibility
property and tests cross-form review isolation. The onboarding helper insertion
also asserts its dispatch marker exists, preventing a silently unmodified fixture.

Earlier mapped attempts had intermittent timing/assertion failures. The fixture
now waits for settled runner/form availability at the affected transitions and
exits on the first assertion with its phase, rather than repeatedly mutating the
same phase until timeout. The final full-panel and onboarding runs passed in
sequence. Screenshot attempts captured the comet/import layouts but later stopped
when the fixture window was no longer on the active workspace; those partial
capture runs are not counted as full-suite successes.

All three executables were reused byte-for-byte from
`build/remote-onboarding-020-desktop/`; their Go input paths and hashes also matched
that baseline. The live inventory had grown to **two runners and four ships** before
this activation. Both immediate and post-readiness snapshots matched the current
pre-activation fleet, including ship identities, configuration, paths, states and
container IDs, enrollment/SSH file hashes, local manager/engine/shell/ship processes,
and installed host-runner binary. Bootstrap reported ready, observing, and no error.

Evidence: `build/sourceport-comet-piers/` (bundle, before/after/settled snapshots and
verified activation), `build/ui-polish-20261020/piers-acceptance-evidence/` (passing
full-panel log and CLI audit), and `build/ui-polish-20261020/piers-capture2-screenshots/`
(partial capture set, including the new forms).
