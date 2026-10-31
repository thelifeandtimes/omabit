# Omabit 0.3.4 — import piers with macOS metadata

Omabit now accepts regular macOS metadata companions in the epoch-log root:
`._data.mdb`, `._lock.mdb`, `._0i<number>` and Finder's `.DS_Store`.
These files are preserved as opaque metadata, included in source fingerprints
and copy verification, and never interpreted as epochs or runtime state.
Orphaned epoch companions are retained. Actual epoch format, ownership, file-type,
link, source-change and stopped-source checks remain required. The epoch count
limit remains 128, with a separate bounded allowance for companions.

Unexpected log-root entries now report their quoted filename. This corrects the
previous generic “unrecognized epoch-log entry” failure for macOS-origin piers;
it does not repair corrupt state or make every historical Vere format supported.

## Update

Run `omarchy plugin update thelifeandtimes.omabit`, then choose **Update
Dependencies** in Omabit's menubar and approve the OS prompt. The matching
runner is required for the import fix. Remote runners need their own explicit
update. Stop the original ship everywhere before manually reviewing an import;
this release does not import or start existing ships automatically.

## Upgrade and validation

The supported patch path is 0.3.3 → 0.3.4. Runner schema 47, desktop state 1 and
bootstrap API 1 are unchanged. Earlier versions use the documented upgrade paths
to 0.3.3 first. The [qualification record](releases/0.3.4/qualification.json) records exact
artifacts and the passed [upgrade/import rehearsal](releases/0.3.4/rehearsal.json),
including its isolation and validation limits.

Synthetic tests cover same-filesystem moves, cross-filesystem copying, interrupted
copy preservation, enrollment recovery, metadata-change detection, epoch limits,
and unsafe or missing real state. No protected production pier is used for testing.
