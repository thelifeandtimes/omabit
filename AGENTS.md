# Repository operational guidance

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
