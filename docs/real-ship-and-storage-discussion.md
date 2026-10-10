# Real-ship and storage decision

The initial flat-directory proposal was superseded by the user's selected
`piers/fakeships/[project]/...`, classified `piers/livenet/...`, and inert
`piers/archive/...` convention. Version 0.20 refines every active ship directory
into a permanent mount root with a `pier/` child and migrates the whole local fleet.

See [pier-storage.md](pier-storage.md) for the current layout, filesystem workflow,
discovery/enrollment, moves, key boot and livenet import contracts, and the
[live migration record](diagnostics/2026-10-02-permanent-piers.md) for acceptance.

Real moon key boot has passed. Remaining import acceptance needs a stopped owned
livenet pier. The implemented importer enrolls a fully transferred directory
inside the runner tree; desktop archive upload/streaming is a later interface.
