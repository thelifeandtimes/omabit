# RC.11 in-place update — 2026-10-18

The user requested simpler boot defaults, CLI-only native control and an in-place
update of the existing installation.

Source: `abfff0529a19adc02cf9f4b93eeef78c645dfa8f`.
Published release: [v0.1.0-rc.11](https://github.com/thelifeandtimes/omabit/releases/tag/v0.1.0-rc.11).
Installed native pins: `c26fc5da75f4092e001eb713c108e688fe4331aa`.

## Verification

- Full Go tests/vet and command builds passed.
- Targeted manager/runtime/collector race checks covered settings, historical
  small looms, legacy HTTP edits and handoffs.
- Both JS suites and mapped QML through direct/native wrappers passed: collapsed
  defaults, five loom choices (30–34), Urbit ID guidance, absence of native toggles,
  and no native-control option emitted by desktop creation/configuration.
- Historical smaller configurations/replays and their immutable launch arguments
  remain manageable, while new registration/configuration choices require30–34.
- [Source CI39103210641](https://github.com/thelifeandtimes/omabit/actions/runs/39103210641) passed.
- Frozen and downloaded release bundles passed checksum, source/schema, installer
  and executable checks. Empty-cache native downloads verified all three tools,
  their clean build revision, help text, cache reuse and clean plugin validation.

## Running ship preservation

The existing healthy `bare-comet` remained running during the final-binary
RC.10→RC.11 `install-host` update, verified at 06:13:48 UTC:

- Same container `c08cbe24ed355e7a0980ef3957a0381072eb5b9a5da2d6f142675b8a015aade3`.
- Same king PID1094395 and worker PID1099612, with unchanged launch arguments.
- Same configuration/applied revision1, loom31, native-off state and ports.
- Same storage mount identity, pier location, runner identity and operation history.
- Same SSH identity/grants, collector enrollment, engine process and shared host state.
- Schema19 retained; no persistent-format migration was needed.

## Desktop verification

The git-managed plugin was updated with the stock Omarchy workflow and cold-loaded
into shell instance `2evqu3b3nt` (PID1253738). At 06:23:59 UTC its live IPC status
reported `version:"0.1.0-rc.11"`, `first_run:false`, `busy:false`, with the supplied
pill default resolved. The original bar position, panel preferences and unrelated
plugin enablement were restored and compared. The comet remained healthy with
the same container, both processes, configuration and receipts after desktop reload.

Private machine-readable evidence is under `/tmp/opencode/rc11-update/`:
`update-review.json`, `update-result.json`, `desktop-before.json`,
`desktop-result.json`, and `native-verification.json`.
The release includes a sanitized `qualification.json`.
