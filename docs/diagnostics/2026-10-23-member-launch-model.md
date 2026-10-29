# Initial member launch model — 2026-10-23

The private pure controller binds the completed empty-storage and anchor-Start
receipts, its distinct operation, exact container contract and already recorded
peers. It commits workspace dispatch and identity, container Create dispatch and
returned ID, Start dispatch and acknowledgement, then a fresh running namespace.
Every mutation follows its committed phase. Known returned identities survive late
provider errors/cancellation. Missing acknowledgements retain uncertainty; this
controller cannot replay, adopt, clean up or resume an existing pier.

Complete observations before workspace preparation, before Create, before Start
and after acknowledgement must match the original anchor namespace, publications
and exact peer contracts. An old-namespace peer, unknown member, changed container
contract, stale/incomplete observation or non-running result refuses advancement.
Storage/pill and unresolved-use checks are an explicit driver preflight before
Create and Start. Mutable driver/callback arguments cannot rewrite prior receipts.

Twenty-four controller outcomes cover every commit boundary, provider failures,
late cancellation, replay refusal and observation changes. Eleven malformed binding
cases, skipped-phase and copied-input tests also pass. The focused three-package
race suite and vet pass. No engine or ship ran and schema26 is unchanged.

`running` is container execution evidence only. It does not claim HTTP/admin
identity, initialized pier storage, healthy applications or recovered subscriptions.
SQLite persistence, actual workspace/pill and Docker adapters, server scheduling
and private-VM qualification are the next implementation work. Preserve the earlier
receipts and record readiness separately; never bootstrap over uncertain storage.
