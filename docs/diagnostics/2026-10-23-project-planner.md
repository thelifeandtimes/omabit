# Offline project planning prerequisite — 2026-10-23

`internal/projectnetwork` now validates a complete fresh fake-galaxy member set,
explicit host-loopback reservations and acknowledgement of shared administrative
access. It separates instance IDs, ship identities and project labels; allocates
deterministic internal HTTP ports; and binds the canonical plan to a review digest.
Existing or reserved instance IDs/host ports, duplicate ships, unsupported identity
classes, privileged ports and altered plans are rejected without a partial result.
A second project may repeat fake identities using different instances/host ports.

Four race-tested cases (including18 refusal/reservation subcases) pass. This is a
pure planning component, without persistence, Docker dispatch, CLI capability or
migration. Its caller must supply a consistent reservation inventory and later
check actual host/resource ownership. No live endpoint or runtime authority comes
from a plan digest. [Evidence](2026-10-23-project-planner.json) retains exact source.

The next durable store slice needs transactional reservations, actor-scoped replay,
revision-bound release/maintenance and old-reader refusal. Before changing schema22,
583 current source files and an independently compiled schema22 runner were retained
under `2026-10-23-schema22-before-projects` in task evidence. This is an unreleased
source snapshot, not a historical release fixture or a completed upgrade gate.

Separately, `login-code --help`, task help and the operator reference now explain
that completed Start may precede a selectable resource/container observation.
Existing offline help checks and all4 skill package checks pass. No new access
path or weaker admission was introduced; full runtime discovery remains pending.
