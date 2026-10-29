# Immutable packages for reusable session recovery

Artifact7 bundles the reusable authority, intent, checkpoint, cleanup and inspection
readers, with explicit support for authorities1/2/3/4. It includes every transitive
local import and uses the new guard. Original artifact writers remain unchanged.
The format6 writer and its deterministic synthetic generation were frozen once;
all83 provenance files verify, original output reproduces byte-for-byte, and the
old reader refuses7. A mixed6/7 collection supports authority3 and refuses4.

18 package tests pass, including isolated imports with no checkout/site packages,
static checks that do not execute bundled code, tampered payload/guard refusal,
unsupported declarations, lost publication reply and exact retry. Independent
verification of all three exported generations passes for authorities1/2/3/4 and
refuses5. The first run passed17 checks but exposed the missing public CLI reader
update. Its original failure is retained. The CLI now recognizes artifact7 and
explicit authority4 while keeping its default authority1. Two unchanged historical
CLI cases pass against the installed source update; all three scopes are inactive.

M336 supersedes M335's missing compatible immutable packaging only. These are
synthetic plugin bundles; repeated actual native Theme/DMs sessions remain next.
The608-test core result is retained separately and was not rerun for this additive
package module plus the narrowly tested public CLI change. This is not a claim
that a628-case combined suite ran. No installed plugin, ship or service changed.
