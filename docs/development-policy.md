# Development and release policy

These are user requirements, clarified on2026-10-29: master must remain a
working, installable Omarchy plugin with a tested upgrade path between releases.
This supersedes the earlier minor-only numbering rule and narrow0.3.1 exception.

## Branches

| Branch | Purpose | Delivery contract |
| --- | --- | --- |
| `master` | Qualified stable releases | Protected; PR merges only; required CI; no direct/admin bypass pushes |
| `edge` | Personal/bleeding-edge improvements | Explicit opt-in; not stable; immutable pinned bundles when published |
| `develop` | Unqualified development | No guarantee of working source, artifacts, setup or upgrades |

Use topic branches for work and `release/X.Y.Z` for stable promotions, including
patch releases. A stable label requires qualification; it is not determined by
which version digit changes. Branch protection forbids force-pushing/deleting
master and requires resolved PR conversations.

The initial protections require PRs and passing CI, including for administrators.
They do not require a second person's approval: this personal repository must allow
its owner to review and merge a PR without an unavailable independent approver.

## Versioning and the installable-master contract

- Stable versions are strictly increasing `X.Y.Z`, including patches such as0.3.1.
- Prereleases use `-rc.N` or `-edge.N`; development snapshots are not stable.
- Published versions and artifacts are immutable. Local packaging may reproduce
  an existing version into a fresh output directory; this does not authorize
  republishing it or merging a same-version product change into master.
- Runner/collector changes require coordinated artifacts and migration/recovery
  qualification regardless of whether a patch, minor or major version is chosen.

The required release-policy check verifies a matching `release/X.Y.Z` branch,
newer stable version, matching root/component/UI versions, exact frozen artifact
pins, backend source identity and passed non-destructive upgrade evidence.
The qualification record must declare `stable_upgrade.paths`: an ordered list of
stable versions ending at the proposed release, a repository JSON evidence path
and its SHA256. At least one path must include the previous stable release.
Intermediate versions are allowed (for example0.2.0→0.3.0→0.3.1), with the tested
steps and supported starting state documented in the linked evidence. CI verifies
the evidence bytes, successful result and frozen target identity; review must
assess whether the actual rehearsal supports the claimed path and its limits.

Source, race, QML, packaging/provenance and native bootstrap tests remain release
gates. The current plugin must load with its selected bundled helpers; unsupported
or intermediate upgrades must have explicit guidance instead of silently leaving
mixed tools or broken state. A numbering rule cannot establish that contract.

Only release pins, documentation and the exact governance-file allowlist in
`scripts/release_policy.py` may change after the artifact freeze. Governance means
contributor instructions and release-policy checks/tests, not packaging code,
bootstrap, CI workflows, skills, QML, Go, installer or runtime inputs. The frozen
source identity and rehearsed assets remain authoritative.

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

## Transition from the old gate (2026-10-29)

The user authorized replacing the minor-only merge requirement after its effect
was explained. The old trusted-base checker cannot accept its own replacement
inside0.3.1. For this transition only, after all other required tests pass, retain
PR review and source/QML requirements, temporarily remove the old release-policy
requirement, merge the reviewed policy/release PR, then immediately restore and
verify the full protection. Record the exact head, checks, merge and restoration.
Subsequent releases must pass the new trusted-base qualification check normally.
Do not use this transition as authority to waive a product/test failure.
