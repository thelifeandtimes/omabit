# Repository operational guidance

Before any live pier import, move, reset, retirement, or deletion, read
`docs/protected-ships.md` and apply its identity protection rules. These rules are
user requirements, including for sourcebox fresh-install cleanups.

Never infer that a missing enrolled path means the pier is disposable: the user
may have moved the actual pier elsewhere. Identify the filesystem object and
process ownership before changing it. Prefer data-preserving, same-filesystem
retirement renames for resets; preserve unrelated ships and host resources.
