# Docker image identity admission — first runtime refusal

The first actual no-NIC Docker run completes guest setup and verifies rootless
Docker29.9.1 plus the pinned official Vere image. The new gateway admission refuses
before Start because this engine reports the manifest digest as its image identity,
where the inert helper expected only the config digest. No gateway process, fresh
Urbit keys or ship processes were created. Guest shutdown is clean after371.99s;
the test remains failed. A preceding copied-driver source-hash path failure never
launched a VM and is retained separately.

A second no-NIC guest reads the stopped disk with both a read-only QEMU drive and
an ext4 `noload` read-only mount. Its retained receipt matches the serial digest,
proves the gateway was never started, and preserves both overlay and base hashes.
It exports code/opaque metadata only. No piers, cookies, credentials or application
values are exported. This avoids the full raw-disk conversion's larger quota need.

The source helper now accepts the verified config digest or manifest digest while
requiring the exact pinned image reference. It does not accept tags or unrelated
hashes. All15 inert lifecycle checks pass in private namespaces; the gateway draft
uses the same identity predicate. This supersedes M338's config-digest-only image
assumption, not its shutdown requirements or the pending Docker runtime gate.

The failed overlay is retained privately as gzip. Independent decompression
reproduces its complete original SHA-256 before removing the redundant raw copy.
Code-only Go caches and public runtime artifacts move to verified local evidence
with their previous paths preserved as symlinks. A read-only module-cache cleanup
refusal is recovered only after checking the retained copies. No ship data is
removed or moved. The image builder's earlier cross-filesystem hard-link refusal
is resolved by an explicit read-only binding of the unchanged pristine base.

Next: a fresh VM/overlay with the corrected pin check, new keys and two keyed
containers. Messaging, state/subscription recovery and runner integration remain
unqualified for Docker private chains. The active goal continues without user input.
