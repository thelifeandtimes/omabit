# Omabit 0.3.0

Linux/AMD64 release. See the [qualification record](releases/0.3.0/qualification.json)
for exact frozen artifacts, completed upgrade tests, required CI and support limits.

## What changes from 0.2.0

- Projects organize fakeships and give them an isolated Docker/Ames network.
  Create an empty project, add members later, and use ordinary fleet health,
  logs, +code, browser, start and stop controls for its ships. Multiple projects
  can each contain their own ~zod. Duplicate labels are refused.
- Application Connections discovers reachable fleet ships and declared plugin
  receivers. Approve, connect, disconnect and recover independent plugin sessions
  without repeatedly entering +code. Theme and DMs still connect directly to the
  ship and retain standalone sign-in. Hosted ships have their own simple pane.
- Import diagnostics identify rejected paths; guided layout uses instance/@p.
  Durable recovery preserves uncertain imports instead of resetting state.
- New identity setup accepts a key path or a pasted key through private input.
  Compact comet setup defaults to recommended settings and automatic naming/ports.
  New and existing ship resource thresholds are separately configurable in CLI.
- Reliability fixes cover lifecycle progress, stale project operations, project
  metrics/identity, clean stop, revocation UI, narrow layouts and loading states.
  Local operator logs preserve @p/+code; boot keys/cookies remain filtered.
- All three CLIs have an offline command reference, progressive help and tested
  request/revision workflows. Operator and plugin-development skills ship in the
  verified bundle and a separate skills archive. Theme and DMs use matching
  native headers, spacing, sign-in choices and private DEV profiles.

## Scope and limits

Stable owner-user0.1.0/0.2.0 is the intended in-place upgrade window. Runner schema47
fences older writers; desktop/bootstrap state1 and management protocol1 remain.
Read the [upgrade guide](releases-0.3-upgrade.md). A desktop update is separate from
an explicitly authorized runner management update. Running and stopped intent,
piers, identities, access grants, ports and operation history must be preserved.

Initial project networking, member lifecycle and direct application authentication
have source, isolated-runtime and human DEV evidence. Expanded anchor/engine
failure recovery, project maintenance/import, turnkey Ethereum private chains and
collection switching UI remain experimental/incomplete. Private-chain planning
and internal journal code do not imply a supported one-click chain launcher.
Those are follow-up roadmap work, not a reason to expose a misleading control.

Omabit remains a host manager and authentication helper. App discovery belongs
in Omart; missing-backend installation belongs in the requesting plugin. Wallet
key custody and a broader authentication security/privacy audit remain open.
Ordinary Eyre sessions are not desk-scoped permissions, and functional approval
checks do not establish malicious-plugin isolation.

## Validation and publication

The frozen binaries passed the actual published0.1.0→0.2.0→0.3.0 upgrade
rehearsal with a running fake and a stopped fake, repeat activation, old-client
observation, schema47 downgrade refusal and manager restart. Native startup and
activation recovery, package checks, all130 offline CLI help entries and the
five skill-package checks passed. UTC recovery regressions and the complete Go
suite pass; the source gate also requires all four managed-state race groups.
Earlier development evidence includes57 simulated CLI workflow commands, paired
native UI renders, and DEV lifecycle, maintenance and direct transport checks.

A final UTC portability fix preserves timestamp instants across journal JSON
reopen without accepting changed ownership or acknowledgements. The qualification
record distinguishes completed results, superseded failures and scope limits.
Publication requires the protected release PR and every required check to pass;
the pinned artifact source remains unchanged by distribution documentation.
