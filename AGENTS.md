# Repository operational guidance

## Branches and versions (user requirements)

Read `docs/development-policy.md` before branching, publishing, or merging.

- `master` is protected and release-only. Never push directly to it or bypass its
  protections. Merge reviewed release PRs with passing required checks.
- Stable releases on `master` are full minor/major versions (`X.Y.0`). Do not ship
  standalone UI/quality-of-life patch updates there; accumulate them on `edge` for
  a later reviewed minor/major release.
- Changes affecting `urbit-collector` or `urbit-runner`, including shared runtime,
  protocol, installer or dependency code, require a new minor (or major) release
  line. Do not disguise backend changes as UI patches.
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
