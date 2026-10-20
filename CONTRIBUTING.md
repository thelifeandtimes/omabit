# Contributing

Read [development and release policy](docs/development-policy.md),
[upgrade requirements](docs/upgrade-policy.md), and `AGENTS.md` first.

- `master` accepts reviewed, CI-verified **minor/major stable release PRs only**.
  Never push directly or bypass protection.
- `edge` carries opt-in personal/bleeding-edge improvements; UI/QoL patch work goes
  here until a later reviewed release promotion.
- `develop` is unqualified development. Use topic branches and keep test state
  separate from live fleets.
- Runner/collector-affecting changes target a new minor or major release line.
- Stable releases use `X.Y.0`; edge artifacts use `-edge.N` prerelease versions.
- Preserve state/identities across upgrades and test the actual update path.

For candidate builds and isolated profiles, see
[desktop development](docs/desktop-development.md). The release policy describes
the exact PR, artifact, pinning and channel-installation workflow.
