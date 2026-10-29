# Fresh database copy from sealed inputs — 2026-10-25

M391 fixes a sealed-input startup defect found while preparing fresh gateway
readiness observation. Node cpSync preserves source modes. A synthetic fixture
reproduces mode0400 on both source and copied database file; opening the copied
file read/write as the unprivileged owner fails with EACCES. The gateway had only
changed the copied root directory mode, leaving these database files unwritable.
Earlier VM evidence used its own prepared inputs; it did not qualify booting the
new sealed generation06 through the resource adapter.

The initializer now copies into a fresh exclusive chain directory. New directories
are0700 and only newly copied regular files receive0600. Linked/unsupported files,
aliased roots, path nesting and preexisting destinations refuse. Each copied file
and directory is synced, followed by the destination parent. Failed/partial copies
are retained and cannot be resumed by rerunning this helper. Input bytes/modes
are never changed. Existing retained chains are never chmodded or repaired.

The extracted helper passes disposable filesystem tests covering writable private
copies, unchanged sources, existing destinations, links, nested directories, sync
failure and refusal to reuse partial output. It also copied the actual generation06
unkeyed foundation:7 files /291,373 bytes. All source hashes/modes stayed unchanged;
new file bytes match, every new file opens read/write, private modes hold, and a
second copy is refused. No database library, gateway entrypoint, ship or engine
was executed. The copied database remains private in the staging directory and is
excluded from this evidence export.

After integration,58 copy/initial-contract/guarded-resume checks pass with explicit
TAP and test-isolation=none. The first wrapper expected TAP but received Node's
spec/file summary; its assertion stopped before repository edits. A failing test
sentinel confirms callbacks execute, and the detailed runner reports18 staged
copy/initial checks. The wrapper refusal and original output are retained.

The new initializer SHA-256 is
5c1f9da910456ad76c708153380c940b288ab8c0a2334971f9c8b8cc3eb55d66.
Generation06 and its old manifest remain unchanged. A new appliance/input generation
is required before adapter runtime qualification; do not relabel or repair06.
This supersedes its suitability as a future sealed-bundle boot input, not M386's
read-only inspection result or M384's separate runtime evidence. No runtime pass
is claimed for the changed initializer, and no public dispatch is enabled.

Fresh readiness remains in progress. It must query the exact running gateway's
RPC for chain ID, genesis, head/hash, all nine contract code hashes and the two
public key rows at that same head. getPoint uses the initializer's earlier pinned
block; use an explicit-head Azimuth getKeys eth_call for current readiness. Do not
infer acceptance from gateway-ready.json alone or send mutation probes from an
observer. The copy sync only covers the initial copied foundation, not later
key generation/chain transactions. Post-initialization durability must be made
explicit before the observer reports Synced=true. These are pending requirements,
not implemented observation or production/wallet-storage claims.
