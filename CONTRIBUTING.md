# Contributing

Read [development and release policy](docs/development-policy.md),
[upgrade requirements](docs/upgrade-policy.md), and `AGENTS.md` first.

- `master` accepts reviewed, CI-verified **qualified stable release PRs only**.
  Never push directly or bypass protection.
- `edge` carries opt-in personal/bleeding-edge improvements awaiting qualification.
- `develop` is unqualified development. Use topic branches and keep test state
  separate from live fleets.
- Stable releases use a newer `X.Y.Z`; patch releases are welcome when qualified.
- Every release needs a documented tested upgrade path from the previous stable
  version, possibly through intermediate releases. Backend changes also need
  matching binaries and migration/recovery qualification.
- Edge artifacts use `-edge.N` prerelease versions.
- Preserve state/identities across upgrades and test the actual update path.

For candidate builds and isolated profiles, see
[desktop development](docs/desktop-development.md). The release policy describes
the exact PR, artifact, pinning and channel-installation workflow.
