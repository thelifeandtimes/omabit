# Private project preparation adapter — 2026-10-23

The private Docker adapter now implements the durable preparation controller's
bridge and stopped-anchor interface. It uses the verified owner engine, exact
daemon and pinned image IDs, fixed creation arguments and exact returned-resource
inspection. Full admitted settings, labels, loopback publications and attached
resources must match. Unknown attachments, persistent volumes, changed engines,
lost replies and repeated dispatch are refused without automatic cleanup/adoption.

The anchor explicitly covers the pinned image's `/urbit` volume with a bounded
1MiB tmpfs, uses a read-only root, no capabilities and a sleeping process. This
mount contract is source-tested; actual Docker normalization remains a VM gate.
Fresh session setup allocates only private metadata after owner/engine checks and
requires the already installed pinned image. Existing metadata remains retained
for explicit recovery. It neither pulls software nor creates a pier.

Contract, controller-adapter, session-setup and affected runtime race tests pass;
vet passes. Sources and test logs are hash-recorded in the companion JSON/evidence.
There is no server scheduling or public capability yet. The caller still needs
runner admission/work fencing; member lifecycle and recovery remain unfinished.
No installed engine, user ship or active plugin was used.
