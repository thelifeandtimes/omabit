# Read-only completed-workspace verification — 2026-10-25

M387 adds OpenWorkspaceVerifier for a retained, completed private workspace. Its
interface exposes only VerifyEmpty and Close. It holds the selected parent and
clones the retained plan/workspace; callers cannot rebind it by mutating slices.
Creation/recovery interfaces and their accepted phases remain unchanged.

Verification opens and reads the exact private empty tree twice and rechecks the
parent. It compares root, piers parent and every member mount identity with the
retained observation. Matching names and an empty layout do not permit replacement
directories to be adopted. Links, changed permissions, unexpected entries, replaced
parents/children, cancellation and closed handles refuse. It performs no mkdir,
repair, chmod, sync, Docker request, chain access or ship operation.

Sixteen top-level / 62 focused race checks plus vet pass. A negative source overlay
that removes the retained child-identity comparison fails both the piers and member
replacement tests, confirming those checks test identity rather than only shape.
After byte-identical integration, the full privatechain package passes46 top-level /
354 race checks. Tests use fresh synthetic directories; overlapping counts must
not be summed, and this is not a full managed-store or live-runtime rerun.

This supersedes M386's pending ready-workspace verification only. It supplies a
fresh precondition, not start authority or isolation from other processes under
the same UID. Concrete engine/input handles, dispatch, shared parent provisioning
and public lifecycle still need integration and runtime evidence.

Keep the empty-workspace condition until a journalled effect owns initialization.
The current guest candidate expects home/marker/namespace metadata under its lab
root; a future adapter must arrange that initialization deliberately. Do not weaken
this verifier or create those files as an unrecorded preflight shortcut. Existing
disposable-VM guards and installed/protected-state exclusions remain.
