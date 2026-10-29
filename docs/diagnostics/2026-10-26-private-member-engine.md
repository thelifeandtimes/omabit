# Private member container and mount ownership

M424, 2026-10-26. A new read-only engine preflight accepts only explicitly
selected member contracts and container IDs. It binds the same plan, workspace,
gateway, engine and image; each member/runtime/container is unique. Contracts are
reconstructed and cloned before inspection. Created containers must never have
started; running and exited containers require a recorded process-start time.
Exit/OOM observations do not constitute acknowledged clean Stop.

Complete member inspections bracket mount and namespace-sharing inventories.
The exact member binds are the only additional workspace exceptions. Unrelated
containers cannot mount the project, a member key/pill, or their aliases/parents;
immutable shared runtime inputs remain read-only. Source-object replacement,
inventory changes and process changes refuse. Initial gateway creation/readiness
still supplies no member exceptions and retains its existing behavior.

A narrow all-container projection reads only ID, name and network mode, including
stopped and never-started containers. Foreign namespace-sharing references to the
gateway or members refuse, whether expressed by name, full ID, prefix or indirect
reference. Unresolved/cyclic references refuse; unrelated resolved projects may
coexist. The known gateway/member projection must agree with its full contract.
No foreign environment, application payload or log is read.

All72 new checks and839 affected gateway/member race checks pass with no skips;
vet and diff checks pass. Tests use temporary synthetic files and inert engine
responses only. The first test run failed because its absent-container mock
treated a legitimate inspect request as an unexpected effect; the corrected mock
returns a missing-container error. Failed evidence is retained. No admission
rule was relaxed.

This preflight is private and not yet wired into the running-member gateway
observer. Fresh kernel namespace observation, held pier identity integration,
durable member lifecycle, first-boot identity capture and real runtime gates
remain. It grants no adoption, dispatch, retry, launch or clean-stop authority.
It cannot establish the historical kernel identity of renamed mount sources.
M424 extends M412/M421 and supersedes only the missing member-aware engine
preflight. Schema43 is unchanged. VM10 is frozen before this change; its ongoing
installation/gateway/key-proof run is independent. Production exclusions remain.
