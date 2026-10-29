# Recovery history retains explicit completed boundaries

The private history model now retains ordered completed recoveries. Each boundary
keeps the ordinary member replacements and review prerequisites that preceded it;
the current ordinary suffix stays separate. Original launch and AnchorStart receipts
remain immutable. Every boundary validates against its exact prefix, and complete
suffix validation precedes historical lookup. Resource and operation reuse checks
cover earlier boundaries, including containers already retired there.

The model exercises two full drain/namespace/retirement/replacement/readiness cycles,
three ordinary replacements between them and another after the second. Historical
bindings remain resolvable. Changed/reordered/missing/incomplete/duplicate histories,
wrong identities/processes, previous-namespace suffixes and reused authority refuse.
A current-readiness prerequisite from before a peer replacement remains valid only
for its unchanged creator/process; duplicate, foreign and altered proofs refuse.

Focused race61.070s, full projectnetwork race196.440s,
existing recovery adapter regression254.309s and vet/diff pass.
The adapter regression still exercises its existing single-recovery contracts;
it is not concrete qualification of a second recovery. Schema37, queues and public
routes remain unchanged. Post-recovery ordinary Docker contract construction,
concrete history readers, durable coordination and real coordinated recovery are next.
