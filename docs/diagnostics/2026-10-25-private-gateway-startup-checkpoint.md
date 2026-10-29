# Gateway startup durability checkpoint — 2026-10-25

M392 adds an explicit initial startup checkpoint. The initializer observes the
expected chain, closes the provider, syncs its private workspace, reopens the
database with no signing accounts, and repeats the observation before a final
sync. Both observations check chain ID, genesis, the exact 0x50 head/hash, all
nine contract code hashes, and both public key rows through Azimuth getKeys at
that explicit head. Each read sequence brackets the code/key reads with the
head; wrong values or changed observations refuse startup.

The sync walks only the selected workspace, with bounds on depth, entries and
regular-file bytes. It requires private modes, the current owner, one device,
regular single-link files and directories. Held file descriptors and named
objects must agree before and after fsync. Child objects are synced before their
parents. It reads no key contents and does not chmod, repair or retry retained
state. A private umask is set after the container, QEMU, interface and namespace
guards, so new LevelDB files are private as well as the copied foundation.

The HTTP listener starts only after this checkpoint. The ready document is
created exclusively and both it and its parent directory are synced before
in-memory readiness is set. Listen, address or publication failures close the
listener and its connections. The new read-only omabit_getStartupStatus request
returns only a fixed format, readiness, head/hash and durable/reopened flags.
A saved ready document alone is still not authority for runner acceptance.

All 114 integrated source checks pass: the new extracted functions, fresh-copy
tests, initial-contract tests and guarded-resume regressions. Tests use disposable
synthetic files, inert server/RPC adapters and injected failures. They exercise
all nine code reads, both key rows, moving heads, every lifecycle failure stage,
descriptor cleanup, object replacement, filesystem privacy, exclusive publication
and failed listener cleanup. The whole initial gateway entrypoint and its chain
or key libraries were not executed. Guarded-resume tests still exercise their
existing host-refusal gate. git diff --check passes.

This is source evidence, not a power-loss or runtime durability result. The
second sync happens after reopening; actual LevelDB behavior must be qualified
in a fresh no-NIC VM. The concrete runner observer must still verify the live
engine/container/namespaces, private filesystem identities and fresh RPC/code/key
state, and must not send mutating negative probes. Input generation06 remains
unchanged and unsuitable for the updated initializer; a new sealed generation
is required. Existing ships, host Docker, installed plugins and services were
not accessed. Schema43 and public lifecycle availability are unchanged.

This supersedes M391's missing checkpoint implementation only. External readiness,
managed backend wiring and runtime qualification remain outstanding.
