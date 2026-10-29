# Gateway observation with retained members

M425, 2026-10-26. The private member-aware observer now composes the M421 held
filesystem verifier and M424 engine ownership preflight with the existing full
gateway observation. It accepts only an exact retained ready-gateway snapshot,
clones member contracts, and binds previously recorded pier/.urb identities.
Created first boots require empty member mounts; created resumes and all running
or exited members require retained piers. Resume identities must match the launch
contract. A retired member's retained pier may remain without a live container.

Every running member's actual kernel network namespace must freshly equal the
saved gateway namespace and the newly observed live gateway. Only a fixed stat
command runs inside selected running containers; created/exited members receive
no exec. Exact process snapshots span both namespace checks, the chain RPC and
held filesystem observations. Changed processes, namespaces, inventories or
objects yield no partial observation. No ship request is sent.

The existing two-minute immutable-input preparation and25-second mutable
freshness window are unchanged. Initial gateway creation/recovery still uses
the original empty-member mode. The new mode is a separate private constructor;
it exposes observation/Close only, with no lifecycle effects, public runner route
or ownership discovery. A saved ready receipt does not itself authorize a launch.

All35 composition/selection checks and874 affected gateway/member race checks
pass with no skips; vet and diff checks pass. Engine responses and filesystem
handles in composition checks are synthetic; M421 independently tests the real
held filesystem implementation against synthetic objects. No actual pier or
engine was accessed by these checks. Runtime cost/freshness must still be tested
with real members; no deadline was relaxed.

M425 supersedes M421/M424's pending internal observer composition only. Durable
member ownership, first-boot identity capture, schema44 migration/refusal gates,
member lifecycle and real runtime qualification remain. Schema43 and public API
are unchanged. Frozen VM10 excludes M424/M425; its gateway/key-proof run remains
separate. Existing successful fixture peers do not qualify this new adapter.
Production ships, tmux ships, ~/temp-ur and installed plugins remain excluded.
