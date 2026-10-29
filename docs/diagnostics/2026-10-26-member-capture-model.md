# Staged capture journal before running observation

M444, 2026-10-26. A synthetic-only Go overlay prototypes M441/M442's missing
capture commit. It is not integrated into the repository, database, concrete
adapter or active M443 VM. The ordinary repository remains schema45.

The preview version3 launch has an explicit `captured` phase and independently
cloned capture evidence. Its controller calls the original held capture adapter
once after acknowledged Start, validates fresh evidence, and commits capture
before invoking later running observation. Running evidence must contain the
same capture, process generation and pier binding. The legacy controller remains
restricted to version2; the preview controller accepts only fresh version3.
Historical version1/2 data gains no new field or authority.

237 affected race checks and vet pass. Cases cover capture failure/cancellation,
commit failure before persistence, lost commit reply after persistence, later
observation failure/cancellation, terminal commit loss, invalid/stale/substituted
evidence, callback aliasing, serialization and phase skipping. Interrupted or
captured journals cannot rerun first boot. A lost commit reply leaves the local
return uncertain; the durable receipt, when committed, remains captured and is
the required lookup source. No recovery command is implemented by this preview.

The first test run retains one expected compatibility-test failure: its old
unknown-version test incremented2 to3, which the preview now recognizes. The
test now rejects4. This updates that test boundary; it does not alter or relabel
the original45 writer fixture or any production format.

Before integration, implement schema46 transaction handling with canonical
launch/operation mirrors, global capture-object claims, fresh commit validation,
reopen/interruption preservation and actor-scoped replay. Prove original45 rows,
sidecars and identities survive, and the actual original45 reader refuses the
populated successor unchanged. Then wire concrete capture/observation methods
and qualify a fresh VM. Separately attributed read-only recovery must consume
the committed identity without reopening first-boot capture or repeating Start.

M443's ordinary VM continues unchanged. This is progress on the pure model only;
M441's missing durable runtime capture remains unresolved until store/adapter
integration and runtime qualification. Evidence is private at
`omabit-qualification-evidence/2026-10-26-member-capture-model`.
