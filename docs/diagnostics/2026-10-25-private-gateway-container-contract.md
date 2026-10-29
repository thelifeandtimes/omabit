# Private gateway container contract — 2026-10-25

M385 integrates the pure container command/inspection portion of the gateway
adapter. The constructor requires a retained valid workspace/gateway selection,
separate canonical mount paths and the pinned image. It copies member choices.
The root mount is derived from the runner state root and recorded project path.
Unsafe CSV mount arguments and overlapping input/workspace paths refuse.

Only the exact project directory is writable. Seven input mounts are read-only;
the root filesystem is read-only, temporary storage is bounded and the image's
implicit volume is replaced by a read-only tmpfs. Exact command, environment,
labels, resource limits, logging, namespace modes, capabilities, privileges and
mounts are checked. A fresh created container is distinguished from a running
initial process; restarted, failed or uncertain process states are not adopted.

Exactly the two reviewed ordinary Eyre TCP ports may bind IPv4 loopback. Actual
publication must match configured ports and the freshly inspected bridge endpoint,
address and subnet. The allocation now carries its network/container IDs so an
observation from another resource cannot be substituted merely because addresses
match. RPC, administrative, extra, wildcard and IPv6 mappings refuse.

Fourteen top-level / 168 race checks plus vet pass. After byte-identical
integration, twelve / 166 overlapping checks pass. Trial1's one failure came from
a synthetic fixture sharing configured and active port maps: adding an unpublished
port accidentally changed configuration too. Independent maps fix that test; the
failed result remains retained. The ambiguity and storage/authority refusals use
synthetic metadata only. No Docker command or gateway is executed.

This supersedes M383's unbound allocation metadata and M384's pending container
contract source. Schema43 and public routes are unchanged. The current input
program still enforces its disposable-VM guard; these builders do not relax it or
grant runtime authorization. Concrete engine/filesystem/input admission, shared
parent provisioning, runtime evidence and public lifecycle remain. A fresh inert
appliance21/input-generation preparation proceeds separately; it is not booted.
