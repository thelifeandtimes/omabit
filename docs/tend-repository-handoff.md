# Sourcebox agent handoff: separate Tend from Omabit

**Historical handoff — migration completed.** The current GitHub `omabit` repository
is the new runner project; do not apply the old rename step below to it. See
[repository.md](repository.md) and the linked Tend migration record for the result.

## Objective

Make Tend a complete, independently buildable application in
`https://github.com/thelifeandtimes/tend`, with sourcebox checkouts and tooling
pointing to that repository. Reserve the `omabit` name for the newer ship-running
and orchestration project maintained on sourceport.

The user will return to the sourceport agent to establish the replacement GitHub
`omabit` repository and cut the Omabit release. This handoff covers Tend only.

## Known starting point — verify before acting

At inspection, `thelifeandtimes/omabit` on GitHub was primarily Tend: its README
was headed “Tend by Omabit.” It contained `desk/`, `omarchy-plugin/`, `bin/omabit`,
Tend packaging, tests, docs and skills, plus an older host-manager import in
`apps/urbit-host/` and Pals/messenger planning stubs. No published GitHub releases
were listed. `thelifeandtimes/tend` was not found.

The GitHub snapshot may lag current sourcebox work. The newer runner checkout on
sourceport has its own substantial development history and working changes; it is
not the old `apps/urbit-host/` import.

## Instructions

1. **Inventory the actual sourcebox work.** Locate the current Tend checkout and
   any additional Tend source trees. Read applicable `AGENTS.md` files. Inspect
   status, branches, remotes, recent commits, ignored files and untracked source.
   Coordinate with other active work before moving a checkout. Identify all Tend
   code and build dependencies; do not assume GitHub is the newest copy.

2. **Preserve the complete current state.** Checkpoint intended source changes in
   ordinary reviewed commits, staging only the relevant files. Preserve local
   work that does not belong in those commits separately. A Git bundle preserves
   committed refs but does not capture uncommitted or untracked files. Keep private
   runtime data and credentials out of Git. Do not reset, clean or force-push away
   other work.

3. **Rename the existing GitHub repository to `tend`.** This preserves its Tend
   history and GitHub metadata. First verify the source repository is still the
   expected Tend project and the destination name is available. If the rename is
   already complete, verify ownership/history and continue idempotently. If an
   unrelated destination exists, report the conflict rather than overwriting it.

   Once verified, the intended GitHub operation is:

   ```bash
   gh repo rename tend --repo thelifeandtimes/omabit
   ```

   Do not create a replacement GitHub `omabit` repository in this task.

4. **Update every Tend checkout's remotes explicitly.** Identify which remote
   currently represents GitHub; update that remote to the dedicated Tend URL.
   Do not repoint an unrelated self-hosted remote merely because it is named
   `origin`. For a checkout whose GitHub remote is `origin`, for example:

   ```bash
   git remote set-url origin https://github.com/thelifeandtimes/tend.git
   git remote -v
   ```

   Use the owner's established SSH transport instead if appropriate. GitHub's
   old-name redirect is temporary for this migration: creating the new `omabit`
   repository will take that URL back. Finish remote updates before handing off
   the namespace. Verify the push destination before every push.

5. **Make the current repository tree Tend-specific.** Keep the complete product:
   `%tend` desk/marks/libs/migrations, browser UI, Omarchy plugin and transport,
   CLI, tests, fixtures, packaging, docs, and skills. Update the root README,
   architecture/workstream notes, agent instructions, build references, clone URLs
   and release links. Preserve genuinely shared dependencies needed by Tend.
   Retire unrelated host-manager imports/planning stubs from the current Tend tree
   only after confirming no active Tend build depends on them; their committed
   history remains available. Keep unresolved or actively developed unrelated work
   preserved and report it rather than discarding it.

6. **Give Tend its own CLI/package names.** The current Tend installer owns
   `~/.local/bin/omabit`, which conflicts with the newer Omabit CLI. Make the Tend
   source and packaging provide `tend …` and install `~/.local/bin/tend`. Update
   plugin invocations, tests, docs and skill examples accordingly. Report any need
   for an `omabit tend …` compatibility adapter to the sourceport agent; do not
   introduce a second installer that owns the global `omabit` command.

   Keep `%tend`, its existing publisher, wire/mark names, plugin ID
   `io.omabit.tend`, and current settings/credential locations stable during this
   repository move. Rebranding those persistent identities is separate migration
   work. This task does not require replacing deployed plugins or moving user data.

7. **Rename the checkout directory only when its references are accounted for.**
   A path such as `~/repos/tend` is appropriate, but first find scripts, mounts,
   editor/agent sessions and tooling that refer to the old path. Repository/remote
   migration can be completed before the directory rename if necessary. Do not
   move piers or running ship data with the source checkout.

8. **Validate and publish the Tend migration.** Inspect the current Makefile and
   run the existing Tend checks. The inspected snapshot provided:

   ```bash
   make check-tend
   make dist-tend
   make check-tend-release
   ```

   Verify the resulting archive contains the complete Tend product, its installer
   owns the `tend` executable, and it can coexist with the ship-manager plugin.
   Test installation in an isolated destination. Commit the migration in logical
   changes and push the intended Tend branch to the verified `tend` repository.
   Do not force-push, rewrite shared history, publish a release, or change the
   sourceport runner project as part of this task.

## Operational boundary

This is a source/repository migration. Do not stop or move ships, alter Urbit runner
services, rebootstrap piers, or modify existing login codes, keys, cookies or user
data. Repository naming alone does not require a `%tend` agent-state migration or
a change to its Urbit OTA publisher.

## Return to the user

Provide:

- Final GitHub URL and sourcebox checkout path.
- Branch and final commit SHA, plus `git remote -v` output.
- Confirmation that current local Tend work was included or explicitly preserved.
- Summary of CLI/package naming changes and any compatibility work still needed.
- Test/build results and any blockers.
- Any remaining references to the old GitHub `omabit` URL or checkout path.
- Confirmation that the replacement GitHub `omabit` repository was not created
  and that the sourceport agent can now take over that namespace.
