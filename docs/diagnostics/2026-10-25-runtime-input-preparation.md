# Repeatable inert runtime-input preparation

`scripts/qualification/private-chains/prepare_runtime_inputs.py` now replaces the
one-off preparation recipe. Explicit inputs are a separately reviewed appliance
build hash, that existing private-chain appliance, the pinned pill, and a fresh
output beneath a canonical owned0700 parent. It checks build/inventory/initramfs/
payload/pill hashes, bounds metadata and archive entries, selects the dependency
and foundation files, then fsyncs and seals the new generation. Nothing from the
bundle is executed, and no database library, engine, ship or VM is opened.

Actual preparation from VM12's frozen appliance produced12,352 files/834,035,190
bytes. Its format1 manifest is byte-identical to M362's separately prepared VM11
bundle: `2e501616bb6421b701a5c9faffdabdd286c4599e44658a6721b2d419e583a560`.
The independently retained Go reader accepts the complete new tree against that
hash and the original reviewed foundation digest. The peer helper changed between
those appliances, but is correctly excluded from this dependency generation.

Eight inert tests cover changed hashes, malformed/duplicate/truncated/special CPIO
entries, inventory bounds, path traversal, hardlinks/symlinks/FIFOs/directories,
output preservation and refusal before creation. A directory-input test exposed
an FD ownership bug; validation now occurs before creating a stream, and every
error closes the descriptor. The corrected tests also check descriptor counts.

The first actual copy reached its240-second wrapper deadline after10,970 files.
It was reaped and remains private and incomplete, with no manifest. A fresh output
passes with a900-second copy budget; hash, storage-headroom, fsync, sealing and
independent-inspection requirements were unchanged. Neither partial nor original
input trees were repaired in place. Raw unkeyed foundation data remains only in
private storage; public evidence includes code, metadata, manifest and test results.

This closes M362's missing reusable producer, not supported runtime delivery,
publisher authentication, proof that an arbitrary database is unkeyed, or runner
startup authority. Schema41 and input format1 are unchanged. Whole-gateway VM12
and its shutdown-gated read-only audit continue separately.
