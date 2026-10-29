# Running-peer recovery appliance preparation

Appliance24 independently verifies175 frozen source hashes, all backing artifacts,
the embedded Go test executable, guest selector, Python helpers and Gall fixture.
The live-peer flag and dedicated guest token are present. Staging was removed only
after verification. Its unbooted artifacts were copied and hash-checked outside
temporary quota, with original paths linked to the retained copies.

This advances M120 to preparation only. It has not booted and is gated on the
separate application-state VM23 result and available storage. The newly added
project-reservation inventory RPC/CLI is outside this frozen source snapshot.
Runtime evidence must check all eight remote writes, retained sessions and
subscription cursors, unchanged peer process through both receiver replacements,
clean exact fixture cleanup and independent stored receipts.
