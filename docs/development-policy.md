# Development and release policy

These are user requirements, established for the v0.1.0 stable release.

## Branches

| Branch | Purpose | Delivery contract |
| --- | --- | --- |
| `master` | Stable minor/major releases | Protected; PR merges only; required CI; no direct/admin bypass pushes |
| `edge` | Personal/bleeding-edge improvements | Explicit opt-in; not stable; immutable pinned bundles when published |
| `develop` | Unqualified development | No guarantee of working source, artifacts, setup or upgrades |

Use topic branches for work and `release/X.Y.0` for stable promotions. Small UI and
quality-of-life updates belong on edge, not standalone master merges. They can be
included in a later reviewed minor/major release. Branch protection also forbids
force-pushing/deleting master and requires resolved PR conversations.

The initial protections require PRs and passing CI, including for administrators.
They do not require a second person's approval: this personal repository must allow
its owner to review and merge a PR without an unavailable independent approver.

## Versioning

- Stable: `X.Y.0`, for example `0.1.0`, `0.2.0`, `1.0.0`.
- UI/QoL edge iterations: for example `0.1.1-edge.1`, `0.1.1-edge.2`.
- Collector/runner-affecting work: target a new minor or major line, for example
  `0.2.0-edge.1` followed by stable `0.2.0`. Prerelease iterations may refine that
  unreleased line; they do not create another stable patch release.
- Release candidates may use `X.Y.0-rc.N` on preparation branches, not as the final
  version merged into master.
- Development snapshots keep explicit labels/source digests and are not stable
  releases merely because their base version is stable.

Backend-affecting paths conservatively include `backend/`, all shared `internal/`,
`deploy/`, `go.mod`, `go.sum`, and shared packaging/bootstrap-verification code.
Any actual effect on a shipped runner/collector
contract also requires a minor line even when its source is outside those paths.
Changing only a binary's VCS stamp does not count as a backend implementation change.

The stable release policy check enforces a newer minor/major, patch-zero stable
version and `release/X.Y.0` source branch. Initial RC→stable promotion of the same
version is permitted. Root/component manifests and the compiled UI version must
agree. Final pins must identify that version and a frozen source ancestor with
matching backend/UI/build inputs.

## Stable publication: one reviewed release PR

1. Prepare a release branch with its version, support window, notes and required
   non-destructive upgrade evidence. Keep master untouched.
2. Commit and freeze the source. Build/test immutable artifacts from that exact
   clean commit; include a qualification record identifying results and limits.
3. Create a draft GitHub release/tag at the frozen source. Verify the draft downloads.
4. Add a distribution commit on the same release branch with exact bundle/tool pins
   and publication documentation. Pins reference the already built source commit;
   they are not self-referential. Do not rebuild artifacts after pinning them.
5. Review the entire PR and pass CI. **Merge with a merge commit**, preserving the
   frozen source commit in master's history. Do not squash/rebase a release PR after
   building its artifacts. Never force a merge or bypass required checks.
6. Publish the verified draft as stable/latest. Verify public downloads and cold
   native startup. Publication metadata and release-note verification updates do
   not require unrelated follow-up code pushes to master.

`scripts/check-release-policy.py` is exercised in CI. The protected-master policy
workflow uses code from the trusted base commit and inspects head blobs as data;
it never executes the PR's code with a privileged token.

The repository's [upgrade policy](upgrade-policy.md) remains a release gate. A
stable label is not permission to reset state, silently downgrade a runner or
re-enroll ships. Future releases must qualify their own actual migrations.

## Edge/develop installation and updates

A new native checkout may be installed without enabling it, then switched to the
chosen channel before enabling:

```sh
omarchy plugin install https://github.com/thelifeandtimes/omabit.git
python3 -B "$HOME/.config/omarchy/plugins/thelifeandtimes.omabit/scripts/omabit-channel.py" switch edge
omarchy plugin enable thelifeandtimes.omabit
```

For subsequent edge updates:

```sh
python3 -B "$HOME/.config/omarchy/plugins/thelifeandtimes.omabit/scripts/omabit-channel.py" update
```

The helper uses the explicit checked-out branch, preserves the canonical remote,
refuses dirty/divergent checkouts, and requests the normal shell rescan. It changes
only the plugin checkout; application/profile/runner state remains separate.

Stock `omarchy plugin update` fetches `origin HEAD`, which is master. Use the
channel helper for edge/develop instead of assuming Git tracking settings change
that behavior. Stable users continue using the stock command.

Replace `edge` with `develop` only when intentionally testing unqualified work.
The UI reports the distribution channel separately from the loaded bundle version
and any development label. Initially these branches are seeded from stable0.1.0
and can share its verified bundle; that does not give future branch changes a
stable guarantee. Explicit profile pins remain explicit across channel changes.

To return to stable, use `omabit-channel.py switch stable`. Returning to a stable
desktop does not downgrade an installed runner database. Use isolated profiles and
disposable guests for experiments involving incompatible state.
