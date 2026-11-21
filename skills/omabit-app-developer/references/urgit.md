# Native Urgit development workflow

Use this path when the developer has chosen `%urgit` to connect a Git repository
to a Clay desk. Read [app development](app-development.md) for the selected runner,
ship/project generation, policy and release settings first. Omabit records the
workflow and gates agent actions; Git and Urgit perform the actual transfer.
No Omabit-specific Urgit service, token store or adapter is required.

## Establish the intended release path

Read the ship and desk settings, repository instructions and actual Urgit binding.
Confirm the repository, full branch ref, destination ship/desk, upstream/downstream
roles, release checks and whether updates already reach subscribers. Saved Omabit
repository/branch fields describe intent; they do not prove the native binding.
Do not replace an existing developer release gate with automatic publication.

Creating a desk needs creation permission. A push/resync/merge that changes a bound
desk needs commit permission; changing an already published desk also needs
publication permission. A fake ship is subject to the same defaults. A native Git
token does not override an Omabit policy refusal. Do not modify policy or switch
to Dojo/Git as a way around a blocked operation. If the requested native workflow
is authorized and its gates allow it, use Urgit's existing controls directly.

Urgit is separate software. Inspect its installed version and documentation before
installation or upgrades; installing it can compile and activate its agents.
The upstream installation path is `|install ~matwet %urgit`. The isolated test
below instead built a pinned source revision and activated it on its own fake.
Neither path authorizes changing an existing user's ship without task authority.

For source development, the tested sequence was: create and mount `%urgit`, build
the source desk, copy it into that mount, commit, and revive its declared agents.
The upstream build is `zig build -Ddesk=/path/to/pier/urgit`. **Its destination must
already exist and it clears that directory before copying.** Verify the exact
mount and preserve existing work; do not aim it at a pier root or unrelated desk.
Keep source outside a live mount until permissions and any autocommit behavior
are understood. The tested tool versions were Zig 0.15.2 and Node 22.23.4.

## Git to Clay

1. In Urgit, create or select the intended repository and bind its chosen branch
   to the existing destination desk. Preserve the project's branch convention and
   the user's requested default for new repositories. The tested upstream creates
   a `main` HEAD initially; the fixture explicitly selected `master` to follow its
   owner's preference. Verify the intended HEAD as well as the bound branch.
2. Configure the repository's write token through Urgit. Smart HTTP uses a remote
   shaped like `https://SHIP-ORIGIN/git/REPOSITORY`. Use an appropriate credential
   helper; never put the token in a remote URL, command argument, report or source.
   Do not substitute the ship's broad web cookie or `+code` for a Git token.
3. Prepare and test the complete desk source, including required marks and any
   frontend build output required by that project's packaging. Recheck the actual
   binding, current permissions and release instructions before the push. Use the
   configured branch/protection/review process; do not force-push to fix a refusal.
4. After a successful push, inspect Urgit's Git/Clay status, remote branch OID,
   destination revision and the expected content. Urgit's native bridge delays the
   ref update until Clay accepts its conversion. A missing-mark rejection was
   verified to leave the branch and destination file state unchanged.
5. Run the application's readiness checks. Clay acceptance is not proof that every
   Gall agent compiled, the app is healthy, frontend assets are usable, or receivers
   have updated. Follow the project's explicit activation/publication sequence.

For a timeout or lost response, inspect the remote ref, Clay state and Urgit status
before another mutation. Do not infer failure from a disconnected Git client or
automatically resend a release. Preserve the original local commit and diagnostics.
If the ref advanced, verify that exact commit's effects rather than creating an
equivalent second release. If the outcome remains ambiguous, leave it unresolved.

Clay marks can canonicalize bytes. In the test, a `%txt` trailing newline was
removed: Urgit reported `in-sync` with `canonicalDifference: true` and
`contentsMatch: false` because its recorded crossing matched. Inspect the mapping
and mark conversion instead of repeatedly pushing to erase that difference.
Conversely, a status label alone does not prove application readiness.

## Clay to Git, publication and frontend assets

Urgit's **Publish desk** / `%publish-desk` action projects the bound Clay desk into
a Git commit. It is not `:treaty|publish %desk`, and it does not by itself establish
an app release on the Urbit network. The tested reverse path was an authorized
Unix edit, explicit Clay commit, Urgit publish, Git fetch and content comparison.
Use the actual binding and current status before choosing a direction; avoid
overwriting a newer Git or Clay edit while resolving divergence.

For distribution, follow the configured source/Treaty release method and frontend
packaging. Omabit's existing release/frontend commands support their advertised
workflow, but do not replace an Urgit-owned source workflow with a Unix-source
commit command. If the developer uses native automation, preserve its gates and
verify the resulting Clay, app, Treaty and frontend state separately. Do not assume
that a Git push uploads a Docket glob, exposes the ship to the clearweb, or proves
that downstream ships received a release.

## Tested boundary and remaining agent work

On 2026-11-07, upstream commit
`7834a4e186998265f5175fd527bc17ea8819c44f` was built without changing its source or
dependency lockfile. One fresh offline fake ran Vere `4.6-5a7614f` with the pinned
bootstrap inputs recorded in Omabit's M565 evidence. It verified:

- Urgit desk activation and its served index matching the built file's SHA256.
- Repository/branch/desk binding and token-authenticated stock Git Smart HTTP.
- Initial Git-to-Clay content, a missing-mark rejection with unchanged ref and
  absent rejected file, then a subsequent valid update.
- A native Clay edit projected back into Git, with matching final content and
  Urgit's `in-sync` status. Each runtime invocation shut down cleanly.

This is a tested development pathway, **not qualification of all Urgit features**.
The pinned upstream frontend suite reported 83 passes and 43 failures, including
source-pattern assertions affected by refactored Hoon arms. Those failures remain
recorded; no upstream tests were rewritten or represented as passing. The native
fixture initially needed corrections to its Hoon text assertions before continuing
from the acknowledged first push; it did not replay that push.

Live remote installation, production upgrades, native peer/GitHub/LFS workflows,
branch protection combinations, arbitrary app compilation, every frontend asset,
network failure recovery and Urgit-driven multi-ship releases were not qualified.
An agent should inspect the chosen upstream revision and test the project's own
checks and packaging on an authorized development ship before changing its release
ship. Never use production state as a disposable test or erase Urgit state to
work around an upgrade failure.

Pinned upstream references:
[README](https://github.com/yapishu/urgit/blob/7834a4e186998265f5175fd527bc17ea8819c44f/README.md),
[build](https://github.com/yapishu/urgit/blob/7834a4e186998265f5175fd527bc17ea8819c44f/build.zig),
[architecture](https://github.com/yapishu/urgit/blob/7834a4e186998265f5175fd527bc17ea8819c44f/specs/architecture.md).
