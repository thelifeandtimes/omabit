# Initial anchor Start model — 2026-10-23

The private pure controller separates completed creation from a distinct Start
operation. It requires complete fresh ownership observations, writes intent before
Start, preserves uncertainty on a lost reply, and records a host-boot-bound network
namespace only after a successful acknowledgement and verified publication.
Cancellation and failed commits cannot cause replay, adoption or cleanup. Starting
an anchor does not establish member readiness. Race tests and vet pass.

Bridge endpoint lists are insufficient to exclude namespace-sharing containers.
The pure membership resolver checks the complete container network-mode inventory,
including exact IDs, unambiguous names/ID prefixes and indirect container joins.
Missing references, ambiguous IDs, duplicates and cycles refuse completion.
Names resolve observations only and never authorize adoption. Tests pass.

These types/controllers are not yet persisted. A future adapter must collect and
bracket the complete inventory and inspect full contracts under the mutation fence.
No Docker operation was performed by these tests. The separate actual-ship
recovery retry is still running in fresh private VM23-08.
