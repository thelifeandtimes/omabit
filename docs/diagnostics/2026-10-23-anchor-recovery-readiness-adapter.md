# Concrete recovery readiness remains observation-only

The adapter reconstructs every explicit successor creation contract, retains all
initialized pier identities, checks complete inventory for retired containers and
pins every replacement's exact recorded start. It brackets direct selected host
identity reads with complete project observations and unchanged Docker metadata.
Historical anchor FinishedAt remains visible through the recovery-only reader.

Twenty inert scenarios plus incomplete-intent rejection cover wrong final identity,
HTTP failure, either process restarted before/after the first read, anchor changes,
publication loss, engine change, retired IDs returning under another network,
changed configuration/piers, unobserved/unselected/stale HTTP and cancellation.
Commands are restricted to read-only inspection and namespace stat; preserved
state bytes/inodes and zero new lifecycle dispatches are checked.

The first run exposed a test alias mistake: mutating its caller input also mutated
the collector's independent history variable. That test now clones authority before
changing the caller-owned input; production code did not change. The original
failed run is retained. Focused race89.024s and vet/diff pass. No coordinated runtime,
persistence, later history, public capability or installed software is qualified.
