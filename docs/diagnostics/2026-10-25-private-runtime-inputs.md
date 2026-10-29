# Local private-chain runtime input inspection

The new development collector command `private-chain-inputs` verifies a complete
local input generation against a separately reviewed exact manifest hash and the
reviewed plan's foundation digest. It executes no inputs and contacts no runner,
engine or chain service. Topic help, command help, JSON/text output and both existing
skills distinguish this inspection from reservation and runtime authorization.

The Linux reader checks a canonical owner-only sealed tree, exact file hashes,
sizes and modes, compiled pins and required roles. Links, unexpected entries,
unsupported formats and mutation during reading refuse. openat2 prevents mount
or symlink traversal; there is no weaker fallback. Metadata, file count and total
bytes are bounded. This is an input snapshot under the caller's trusted UID, not
publisher authentication or protection against other code running as that UID.

The first producer refused before creating output because the appliance embeds
its payload in initramfs and removes the temporary payload archive. A separate
fresh attempt verified that embedded payload and the pinned pill, then prepared
an owner-only bundle in private storage. The read-only source command verified
12,352 files / 834,035,190 bytes. The built collector independently passed JSON,
text, help and five refusal cases; no registry was opened or manifest changed.

Affected package tests/race checks pass468 checks including subcases, vet passes,
and both skills validate. The exact original version1 manifest, producer, source
reader and uninstalled binaries are retained for later format compatibility work.
No raw database, key, pier, cookie or application value is in this evidence folder.

This closes the initial input-inspection boundary proposed in M354, not supported
runtime delivery. The current gateway inputs are guest-only qualification candidates.
The pinned image reference is metadata; the command does not observe an engine's
image. Schema41 reservations and their historical bytes are unchanged. Whole-gateway
VM11 continues with frozen inputs; initialization/retained journals, actual runtime
dispatch, publisher verification, endpoint publication and simultaneous private
projects remain separate acceptance gates.
