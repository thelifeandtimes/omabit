# Documentation boundaries

Repository documentation serves users and contributors who have no access to a
maintainer's desktop or private knowledge store.

| Keep in Git | Keep in private development notes |
| --- | --- |
| Installation, upgrades, support windows and recovery contracts | Personal fleet inventories, host aliases and machine-specific paths |
| Architecture, protocols, schemas and consequential technical decisions | Debugging chronology, tentative ideas, session handoffs and experiment scratch |
| Reproducible build/test/development instructions | Local workspace/worktree maps and temporary manual-review instructions |
| Compact qualification results, limits and provenance | Raw operational evidence and personal environment observations |
| Required sanitized migration fixtures and reproducible regressions | Credentials never belong in either location |

Use the maintainer's private knowledge store for the right column; a local private
notes directory is sufficient for other contributors. Lattice is the maintainer's
store, not a product/build dependency. Record dates, source commits/digests,
measured outcomes, uncertainty and explicit replacement links. Keep a small offline
copy of critical operational instructions when that store depends on a managed
ship. Exporting instructions is distinct from backing up a pier.

A discovery belongs in both places only when it yields a general contract: keep
the personal investigation privately and write the supported rule/regression in
Git. Public docs should explain behavior without referring readers to a private
chat. Never imply that an experiment passed a gate it did not exercise.

## Pre-1.0 cleanup

Historical development logs remain available while they are classified. Do not
append new session transcripts to them. Before 1.0:

1. Inventory historical documents and inbound links, including release-policy
   allowlists and test fixtures that consume documentation paths.
2. Extract durable public decisions into the appropriate subsystem guide with
   provenance and a clear current status.
3. Archive environment-specific notes privately and verify the stored content
   before removing the repository copy. Preserve supersession/history privately.
4. Retain required sanitized fixtures and compact release evidence in Git. Update
   links and check that a public reader can build, test, install and upgrade using
   the repository alone.

This is a deliberate migration, not permission to bulk-delete historical evidence
or rewrite Git history. Release assets and frozen source identities stay immutable.
