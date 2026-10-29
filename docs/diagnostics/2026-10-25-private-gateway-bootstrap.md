# Journal-gated private gateway metadata — 2026-10-25

M389 adds GatewayBootstrap, a filesystem helper for the guest gateway's private
home, marker and namespace metadata. Opening requires the selected planned
journal and an observed, still-empty workspace. It holds the original workspace
objects and clones the plan. Open and VerifyEmpty create nothing.

Initialize accepts only the same selection in start-dispatched phase. Its caller
must already have committed that receipt and checked the selected engine,
container and inputs. It reads the current process's net/pid/mount namespace
identities; caller JSON cannot supply them. It creates exactly home/0700,
marker/0600 and guest-namespaces.json/0600 under the held root, exclusively, with
file/directory sync and repeated current-root checks. It creates no chain, keys,
piers or engine resources and runs no input code. The existing guest guard still
requires a disposable QEMU container with distinct namespaces. This metadata is
not proof of a no-NIC VM or permission to use the host engine.

The handle fences its first write attempt. Any partial failure remains in place;
no retry, adoption, repair or cleanup is supplied. Close is idempotent and
serializes with methods. The read-only WorkspaceVerifier remains unchanged.

Six top-level /31 focused race checks plus vet pass. Tests use fresh synthetic
workspaces, including changed/aliased directories, foreign files, wrong receipt
selection/phase, cancellation, sync failures and root replacement after the first
write. A combined controller test uses real metadata writes and an inert engine:
failed/lost startup commits produce no writes, partial bootstrap prevents Start,
failed Start preserves metadata, and a lost post-Start commit cannot repeat effects.
The inert engine intentionally refuses readiness; these are not Docker results.
After byte-identical integration,57 top-level /395 complete privatechain race
checks pass. Counts overlap and should not be summed.

This supersedes M387-M388's missing journal-gated metadata helper only. The helper
has no concrete runtime caller yet. Next compose it with held inputs, the private
engine/image and resource contracts, then add fresh readiness observation and
qualify that complete adapter in a new no-NIC VM. Shared parent provisioning,
public lifecycle, concurrent private projects and supported delivery remain open.
Schema43, installed services/plugins and protected ships remain untouched.
