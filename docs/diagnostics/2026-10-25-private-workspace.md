# Private project workspace ownership and interruption recovery

The private-chain workspace controller and Linux filesystem adapter now pass
31 package tests /216 including subcases with the race detector, including
17 workspace tests /85 with subcases. Static checks pass. Actual filesystem
tests create only new temporary directories; no engine, chain, key or pier runs.

The planned layout is an owned `private-projects/<project-id>` workspace with
exclusive `piers/<instance-name>` mount parents. Future piers will be named ship
children of those mounts. This is an internal workspace, separate from the
existing standalone pier catalogue. The model binds runner/project/generation,
reviewed plan, revision, operation and the previously selected parent identity.

The controller commits dispatch before mkdir, preserves a created root identity
even when sync fails, and refuses automatic replay after any uncertain mutation.
The adapter pins the exact parent/root objects, exclusively creates0700 folders,
uses openat2 without symlink or mount traversal, and checks ownership and exact
contents. Tests cover existing objects, aliases, replacements, changed permissions,
cancelled/failed commits, failed sync, partial layouts and changes during creation.

An interrupted but complete layout can be inspected and synced without repeating
mkdir. The observer verifies every empty member parent, syncs the matched objects
and rereads the exact layout/identities before committing success. A missing or
unexpected entry remains preserved and unresolved; recovery does not fill or
delete it. A changed pier-parent during creation stops the remaining creation.
The common runner mutation/update fence remains a caller requirement; this is
not an isolation boundary against other code running as the same Unix user.

This is source and temporary-filesystem qualification only. No public action or
runner queue is wired to it. Schema41 remains unchanged. Persistence, replay,
startup reconciliation and original41 downgrade refusal are next; M369 supplies
the frozen original writer and populated baseline for that migration. The
independent HTTP-publication VM/audit continue with their original frozen inputs.
