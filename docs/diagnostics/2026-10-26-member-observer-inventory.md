# Reduce repeated member gateway observations

M443, 2026-10-26. M442 identifies the post-capture25-second gateway observation
deadline as the diagnostic run's first failure. Two changes reduce work on that
path without changing the deadline, freshness limits, resource limits, schema45,
or the selected container/runtime contracts.

Mounts and namespace-sharing references now use one narrow Docker projection
from a complete container list, checked before and after each read. Two combined
snapshots still bracket path-object checks and compare every mount/name/reference.
Exact member process and gateway/engine checks remain. Foreign environment,
commands, logs and application state are not requested. The same inert complete
one-member observation requires45 Docker commands instead of57 (12 fewer).
This is command-count evidence, not a measured runtime speedup.

After the chain RPC returns, full held-input content and initializer verification
can run concurrently with the independent final storage/runtime observations.
Both must finish within the same original25-second window. All exits cancel and
join the verifier before releasing handles; Close remains serialized. No hash
check is replaced with metadata-only trust, and errors return no partial proof.

791 affected race checks pass without skips, including malformed/partial combined
projections, changing inventory, existing ownership/process/namespace regressions,
overlap ordering, verifier failure, runtime failure, cancellation and concurrent
Close. Vet and diff checks pass. The ordinary runner and VM fixture build, and
the fully armed fixture refuses execution on the host before runtime access.
The inventory-only predecessor passed785 checks; that evidence is retained.

Packaging first refuses at8,533,725,184 available temporary bytes, below the
unchanged8 GiB floor, before creating an appliance. The completed M440 stage and
audit16 are synced/hash-verified into private retention with compatibility links,
recovering106,134,550 bytes. A distinct packaging attempt passes. The unbooted
453,463,404-byte appliance is also durably retained before launch.

Fresh ordinary appliance11/run11 is dispatched under the same no-NIC QEMU and
dedicated rootless Docker boundary. It uses the same pinned image/pill/chain inputs,
12 GiB/no-swap host cgroup, two CPU cores and original timeout/storage limits.
Actual member launch, reopen, readiness and independent audit17 remain pending.
Do not treat source tests or the older logging-only run as runtime qualification.
M441 capture persistence remains a separate future format/migration requirement.

Build SHA256: `bf7c5a1b93491e776a9862aab27f3019c527d814bc05dcf3431344d0eb7549cc`.
Runner SHA256: `c533176d05aa40934b34d4b27a20e912705ed3a8641ae37c6a86ffea9aba2577`.
Fixture SHA256: `80cd918aec8d81d1e6ac6fabf5ac0918e31d86bbc88d23d539e07d8e3cb8d576`.
Evidence: `omabit-qualification-evidence/2026-10-26-member-observer-inventory`.
Protected/tmux/temp-ur ships, installed plugins, production profiles and host
Docker remain excluded. No release or deployment is performed.
