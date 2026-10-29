# Combined private gateway readiness observer — 2026-10-25

M395 combines the fresh RPC probe with engine/image identity, exact running
container and bridge contracts, configured mount overlap checks, independent
host/container namespace observations and the initialized-workspace verifier.
Inputs and filesystem/runtime observations bracket the RPC read. Changed state,
resources, namespaces or startup process refuse; only a complete result passes
the existing gateway journal validator. The operation is bounded to 25 seconds.
The observer exposes no Create/Start methods and never modifies a journal.

A new held-input helper hashes only the fixed initializer path against an
implementation pin, supplementing full manifest/tree verification. Readiness
trusts startup flags and the read-only RPC contract only with the M392 source
hash. The current configured mounts are checked for other workspace writers and
writable input overlap, including stopped containers. They cannot prove a
running container's historical kernel mount identity after a same-UID rename;
the lifecycle fence and trusted owner remain part of the contract.

All 91 focused race checks and vet pass, including resource-preflight regressions,
changed inputs/program/engine/mounts during observation, process restart, changed
host/boot/container namespaces, cancellation and stale observations. After
integration all 434 privatechain and 285 private-gateway race checks pass, with
git diff --check clean. Transports and engine adapters were inert.

The actual sealed generation06 was fully rehashed read-only: 12,352 files and
834,036,904 bytes still match its manifest. Its original initializer matches its
old source hash but fails the new implementation pin. The real observer factory
refuses it before any engine inspection. No bundled program or live gateway ran.

This is complete observer source composition, not runtime or release acceptance.
The initializer pin must also be enforced in resource preflight before creation
and startup; currently this new enforcement belongs to observation. A first-start
driver must wait read-only for initialization within its startup deadline, while
recovery remains separately attributed observation. Concrete managed integration,
owned client/parent provisioning, new sealed inputs and a fresh no-NIC VM test
remain. Existing protected ships, installed plugins/services and generation06
remain unchanged. This supersedes M394's pending observer composition only.
