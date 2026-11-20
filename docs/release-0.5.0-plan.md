# 0.5.0 release completion plan

Status: frozen candidate with partial final qualification, not a qualified release. The user authorized completing the
issues16–21 work on2026-11-20, including a preserved DEV update and the protected
release process. Production deployment and issue22 are outside this goal.

The release branch starts from4856f158f941dc1d62f2dbcaeda6ce853f5d3496 and retains
the complete issue implementation history. Original development work remains in
its original worktree. The pending M755 harness was copied with verified hashes;
subsequent corrections and evidence belong to this release branch.

## Finite delivery scope

- **16 and the first17 milestone:** bounded legitimate Stop-transition handling,
  reliable saved receipt lookup, supported diagnosis/review/reconciliation of a
  clean interrupted member Stop, linked history, and ordinary same-pier Start.
  Qualify the native transition and final release upgrade without treating the
  historical incident's exact predicate as proven.
- **21:** preflight the selected runner's actual capabilities, adapter and policy
  before app-specific creation. Verify the installed DEV tools agree and complete
  a fresh minimal app workflow while preserving the existing DEV state.
- **18:** deliver the independently packaged app-development skill and explicitly
  versioned Gall/Grubbery source, state, recovery and restart workflows. Rehearse
  the supported minimal workflows with final tools. Broader peer discovery,
  permits/routes/assets and failure combinations remain separate qualification;
  guidance must not claim complete general application readiness from one sample.
- **19:** deliver the shared typed catalog and tested CLI/MCP operations with
  explicit exclusions for unported legacy operations. Keep named test execution
  unavailable until implemented and qualified. SDK protocol tests do not establish
  interoperability with every named coding client. Publish only tested transport
  support; retain experimental labels for unqualified client recipes and local
  HTTP support. Remote OAuth/TLS deployment is excluded.
- **20:** retain the explicit opt-in recording pilot, bounded capture and immutable
  original history, verified timing queries, and original/redacted export choice.
  Qualify burst/long-line capture, limit and disk-pressure behavior, gaps,
  manager lifetime, same-pier replacement, and exports for its supported runtime
  path. Do not advertise adoption of existing processes, every runtime factory,
  automatic deletion, or age-based enforcement. Broader oldest-output inventory
  and retirement workflows stay follow-ups unless needed for the pilot's safety.

Project resource containment, broader lifecycle recovery and complete app/MCP/
recording coverage remain open portions of their original issues. Resource usage
observations must retain their explicit incompleteness; they grant no automatic
Stop or deletion authority. No issue is closed merely because code is included.

## Required release evidence

1. Native Stop capsule and independent receipt/pier/peer/history verification pass
   at frozen68bf233 (M760) and final bdf4397 release binaries (M771). Preserve
   both exact records. Corrected candidate02 at59d83aa now passes matching native
   Stop and independent preservation at M777; its actual stable59→65 upgrade
   passes at M774. DEV remains separate.
2. Audit all exposed capabilities and documentation against the scope above.
   Complete the scoped runtime checks and correct discovered product defects.
3. Set all component/UI versions to0.5.0, freeze the complete runtime/UI/skill/
   packaging inputs, and build matching immutable artifacts from that clean commit.
4. Pass full source, race, CLI/MCP contract, UI/QML, package/provenance and native
   bootstrap gates. Earlier isolated passes remain evidence for their actual
   sources; they do not qualify the final combined candidate.
5. Qualify published0.4.0/schema59 to the final candidate (currently schema65):
   original identities, piers, access, settings, operation/application history,
   running/stopped intent, restart/interruption handling, mixed versions and
   older-reader refusal. Record exact artifacts, starting state and support limits.
   Do not repair state or reset a host to make an upgrade pass.
6. Update only the authorized DEV namespace with state preserved; verify exact
   loaded tools, plugin startup and the fresh minimal app workflow. Existing DEV
   ships/history are not disposable test fixtures.
7. Prepare and verify draft release downloads, add exact distribution pins, review
   the release PR and pass required checks. Preserve the frozen source with a merge
   commit; publish stable/latest only after the protected merge. Verify downloads
   and cold native startup. Publication does not deploy to production services.

## Operational limits

The approved48GiB aggregate budget and one-runtime-experiment rule remain in
force. Account for retained builds across worktrees. Retire only classified
obsolete artifacts or use verified lossless archives, retaining required original
writer/reader and state-transition fixtures. Read-only fixture backings remain
immutable; restored archives require hash and storage admission checks.

See [next gates](issues-16-21-next-gates.md), [upgrade policy](upgrade-policy.md),
[experiment isolation](experiment-isolation.md), and
[release policy](development-policy.md). Protected, production, tmux and manual
review exclusions continue to apply.
