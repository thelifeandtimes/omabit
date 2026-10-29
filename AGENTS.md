# Repository operational guidance

## Branches and versions (user requirements)

Read `docs/development-policy.md` before branching, publishing, or merging.

- `master` is protected and release-only. Never push directly to it or bypass its
  protections. Merge reviewed release PRs with passing required checks.
- Stable releases may use any newer `X.Y.Z`, including patch releases. The gate
  is a working native plugin and a tested, non-destructive upgrade path from the
  previous stable release; intermediate releases are allowed when documented.
- Collector/runner changes require matching artifacts, migration/recovery tests
  and a stated support window, regardless of the version digit that changes.
- `edge` is the opt-in bleeding-edge/personal channel, explicitly not stable.
  Publish its artifacts as immutable `-edge.N` prereleases, never as latest stable.
- `develop` is unqualified development with no stability guarantee. Start ordinary
  development here or on topic branches; promote tested work deliberately.
- Preserve these rules in contributor docs and CI. Native edge/develop updates must
  fetch their explicit branch; stock Omarchy's updater fetches remote HEAD.

## Remote onboarding (0.2.0 direction)

- Use the user's existing SSH alias/port for the one-time authorized setup, then
  provision a distinct restricted Omabit management key for each desktop/runner.
- Existing-runner pairing is access-only: preserve its binary, services, ships,
  identities and history. Fresh installation must be explicitly detected/approved.
- Verify a fresh unattended connection, runner identity and key role before saving
  enrollment. Keep the existing dedicated-port transport supported for old installs.
- Preserve pending attempt/key/installer bindings on retry; failed revocation must
  never silently become reauthorization. New backend work targets the0.2.0 line.

Before any live pier import, move, reset, retirement, or deletion, read
`docs/protected-ships.md` and apply its identity protection rules. These rules are
user requirements, including for sourcebox fresh-install cleanups.

Never infer that a missing enrolled path means the pier is disposable: the user
may have moved the actual pier elsewhere. Identify the filesystem object and
process ownership before changing it. Prefer data-preserving, same-filesystem
retirement renames for resets; preserve unrelated ships and host resources.

Stable 0.1.0 must ship with a tested, non-destructive upgrade path to later
versions. Read `docs/upgrade-policy.md` before changing persistent formats,
install/update behavior or runtime contracts. Fresh-install resets are development
exercises, not an acceptable upgrade procedure for users.

## Active platform experiments (2026-10-21)

Read `docs/experiment-isolation.md` before any experimental runtime or desktop
operation. The user's current constraints supersede historical deployment/reset
permissions: protect managed `~lismyl-lismyl`; do not inspect or operate on ships
in tmux or anything under `~/temp-ur`. Preserve current installed plugins and
services. Use fresh disposable ships and separate experimental plugin IDs,
labels, state, credentials, IPC, hooks and desktop environments.

## Development memory

Read `docs/development-memory.md` alongside the current roadmap/progress record.
The user requested durable development memories with explicit supersession. Keep
stable decision IDs, date, evidence and status; link replacement decisions rather
than silently rewriting history. Distinguish hypotheses, failed experiments and
qualified behavior. Never store credentials in these notes.

## Release qualification policy clarification (2026-10-29)

The user replaced the minor-only rule and narrow0.3.1 exception: master must
contain installable, working releases with a documented tested upgrade path.
Patch versions are allowed. Preserve PR review, source/race/QML gates, artifact
pins, frozen runtime inputs, and upgrade evidence. Release-governance files may
change after the artifact freeze; runtime, UI, installer and packaging inputs may
not. The user authorized modifying the merge requirement to install this policy;
record the transition and restore the qualification check after that merge.
No production service upgrade is implied by release publication.
