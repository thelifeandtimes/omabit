# Private gateway resource adapter — 2026-10-25

M390 integrates the concrete resource methods for the private gateway controller:
VerifyPreparation, CreateNetwork, InspectNetwork, CreateGateway, InspectGateway,
and StartGateway. A private constructor opens the M388 input verifier and M389
bootstrap handle for a selected planned journal. It requires an already-owned
Docker client workspace and the selected pinned image; it creates no workspace,
metadata, engine resource or input generation during construction.

Preconditions freshly check the private rootless engine identity and host policy,
pinned local image, held empty workspace, full reviewed input inspection, the two
loopback ports and container mount inventory. Inventory includes created/stopped
containers and reads only IDs/mounts. Any workspace overlap, writable input overlap,
unresolved volume/source, alias or changing inventory refuses. Other containers
may share the immutable inputs read-only. This is current-path negative admission;
it cannot establish a running container's historical kernel mount identity after
its source path was renamed. The runner lifecycle fence and VM isolation remain
separate requirements.

Effects require the exact selected journal phase and resource IDs. Each create
is fenced before dispatch; a complete returned ID survives a simultaneous client
error/cancellation so the controller can retain it. Unknown creation outcomes are
not retried or adopted by name. Created gateways must match the complete M385
contract and never have started. Their bridge is checked before and after container
inspection. Before Start, the input generation is hashed again after the slower
engine/container/mount checks. The existing recorded start-dispatched receipt
covers M389 metadata initialization and the single Docker Start. Partial bootstrap
or a lost/wrong Start response preserves uncertainty without cleanup or repetition.
Close releases both owned handles exactly once and serializes with methods.

Six top-level /45 focused race checks plus vet pass. Engine calls are injected:
tests cover controller ordering, known IDs with errors/cancellation, unknown replies,
changed engine/container identities, wrong selection, busy ports, mount overlap and
read-only sharing, input changes during startup checks, direct retry refusal and
concurrent Close. The first fixture combined the config-image ID with a simulated
manifest-image ID; admission correctly refused before resource creation. The fixture
was aligned and that failed run is retained. Production admission was not relaxed.

A temporary qualification test uses generation06's actual12,352-file /834,036,904-byte
bundle and a new real workspace created through the layout controller, while every
engine call remains simulated. Construction and repeated preflight pass without
metadata creation; caller slice edits do not rebind the handle, root replacement
refuses, Close prevents reuse, and the input manifest remains unchanged. The final
qualification result records the exact source hashes. No bundle program ran.

After byte-identical integration, all18 top-level /211 private-gateway contract and
adapter race checks pass. Counts overlap; this is not a full runtimequalify, managed
store or live Docker qualification. No actual engine, ship or installed plugin was
accessed. Schema43 and input format/pins remain unchanged.

This supersedes M389's missing concrete resource methods only. The adapter does
not yet implement fresh GatewayObserver/chain readiness or expose a public factory.
Tests deliberately refuse readiness instead of manufacturing a running observation.
Next add fresh chain/runtime observation, compose the managed backend and perform
a new no-NIC VM acceptance run. Shared parent/client-workspace provisioning, public
lifecycle, concurrent private projects and supported delivery remain open.
