# Initial operator skill and offline help

The source now includes [the operator skill](../../skills/omabit-operator/SKILL.md),
with focused operation and stopped-import references. It starts from the selected
profile/registry and pinned runner, distinguishes management instance IDs from ship
identities, and describes asynchronous receipts, uncertain outcomes, revisions,
retained import sources and the separate explicit Start. It preserves existing task
authorization and exclusions rather than imposing repeated confirmation prompts.
The skill has not been installed into the user's global agent or plugin directories.

Checking its actual command help exposed two existing defects: `logs --help` and
`login-code --help` treated the help flag as an instance and demanded live runner
arguments. Both now display their requirements without opening a registry or
fetching a secret. Help works before or after an instance and with `-h`; normal
required identity/revision/reveal checks remain in place.

Validation on 2026-10-22:

- Skill frontmatter/reference structure validation passed.
- Eleven actual built-CLI help paths passed inside a private Bubblewrap namespace
  with no host HOME, runner socket or network. The initial failed help records and
  corrected results are retained separately.
- The complete collector package passed `go test -race`; `go vet` and diff
  whitespace checks passed. New tests verify help without registry side effects
  and refusal of incomplete operational inputs.

See [exact source and evidence identities](2026-10-22-operator-entry.json).
The running import VM uses its previously frozen collector, so its results are
not claimed as tests of this later help-only change.

This records the initial operator entry point. Later checkpoints add
[task-family disclosure](2026-10-22-task-help.md) and
[five scripted CLI tasks](2026-10-22-operator-walkthrough.md). Further
realistic task walkthroughs across uncertainty/stale revisions, developer skills
and distribution remain. Schema validation and successful `--help` calls alone do
not establish the roadmap's agent-usability acceptance gate.
