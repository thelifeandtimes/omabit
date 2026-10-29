# Repository ownership and publication

## Two independent products

- **Omabit**: <https://github.com/thelifeandtimes/omabit> — this runner/collector,
  desktop-management and orchestration source tree.
- **Tend**: <https://github.com/thelifeandtimes/tend> — `%tend`, its browser/mobile
  clients, Omarchy plugin, standalone `tend` CLI, tests and packaging.

The previous GitHub `omabit` repository was renamed to `tend`, preserving its
history and repository identity (`R_kgDOUrzoJg`). The replacement Omabit repository
uses this checkout's independent history, starting at
`db7fc4ccc376196a9c19152d8fbfbcfc4a6fb69f`. Do not merge the retired Tend-side
`apps/urbit-host/` import over this implementation.

The sourcebox handoff is documented in
[Tend's migration record](https://github.com/thelifeandtimes/tend/blob/master/docs/TEND_REPOSITORY_MIGRATION.md).
Its canonical checkout is `~/repos/tend`; retained old paths and worktrees do not
retain ownership of the GitHub Omabit name. Once this repository occupies `omabit`,
that URL no longer redirects to Tend.

## Git remotes and Go imports

In the sourceport working checkout:

- `github`: `https://github.com/thelifeandtimes/omabit.git` — public publication.
- `origin`: `https://git.thechristophgroup.org/thelifeandtimes/omabit` — original
  self-hosted history endpoint, retained explicitly rather than overwritten.

The current Go module remains `git.thechristophgroup.org/thelifeandtimes/omabit`.
Clone this GitHub repository and build from the checkout as documented in the
README; the module/import identity does not select the Git push destination.
Any module-path change should be an explicit coordinated source change, not a
history rewrite or a consequence of repointing a remote.

## Command and data ownership

Omabit owns the `omabit`, `urbit-collector`, and `urbit-runner` commands. Tend's new
packaging owns `tend`. Its stable `%tend`, `io.omabit.tend`, `omabit://tend` invitation
URI, and settings/credential identities remain part of Tend's compatibility contract.

Sourcebox still had an old installed Tend executable named `~/.local/bin/omabit`
at migration time. Inspect its ownership and deliberately retire it before installing
a global Omabit command there. An optional `omabit tend` forwarding adapter is
separate work; it is not implemented by this repository cutover.

## Source publication versus release assets

`master`, `edge` and `develop` follow the [user's development policy](development-policy.md).
Master is protected and accepts qualified stable releases, including patches,
by PR only. Every release needs a tested non-destructive upgrade path; backend
changes require coordinated binaries and migration/recovery evidence. Edge is
opt-in; develop is explicitly unqualified.

The authoritative source and native Omarchy installation repository are both
<https://github.com/thelifeandtimes/omabit>. Its root manifest loads small wrappers
under `native/`, referencing the shared `plugins/omabit/` QML and checksum-pinned
release tools. Generic CLI entry points are under `backend/cmd/`, the desktop helper
is `cmd/omabit/`, and shared private Go packages remain in `internal/` under one
module. See [native distribution](native-plugin.md).

The briefly published `omarchy-omabit` repository is a historical generated RC.2
distribution. New installations use `/omabit` directly. The legacy flattened export
generator remains available for reproducibility; it is not the canonical source
or a required installation step.

Repository commits contain source, tests, deployment templates and reviewed
qualification records. Pill bytes and generated packages are release assets,
excluded from source commits. The provided pill's metadata and reported contents
are recorded in [pill-plan.json](pill-plan.json).

Source publication does not itself tag or publish a release.
CI runs the source/model/installer tests; mapped desktop checks and actual pill
bootstrap acceptance remain separately recorded qualification steps.
