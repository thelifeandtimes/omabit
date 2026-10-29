# Keep input preparation outside mutable observation freshness

The gateway observer now performs its initial full sealed-input verification in
a separate, bounded two-minute preparation stage. It starts the existing25-second
freshness clock immediately before observing engine, container, namespace,
filesystem and RPC state. The final full input verification remains inside that
clock, so changed inputs still invalidate the observation. Both passes use the
same held input handle. Cancellation and preparation failure return no authority.
No hash cache, permission relaxation, schema change or runtime retry is added.

A synthetic41-second first input pass reproduces rejection against the original
source. Corrected source accepts that preparation but still rejects a mutable
snapshot that ages26 seconds. The regression verifies both full input passes and
their distinct bounded deadlines. All355 private-gateway race checks pass before
and after integration, followed by vet and diff checking. Existing changed-input,
namespace, filesystem, engine, mount and stale-observation refusals remain covered.

M405 supersedes the observer's previous preparation/freshness ordering from M395
only. M404's failed managed runtime remains failed; the new observer requires a
fresh no-NIC VM before claiming readiness/recovery. The fixture deadline and all
VM resource/isolation limits are unchanged. No protected or installed state was
accessed. Next: rebuild current runner/fixture and repeat on a fresh supervised disk.
