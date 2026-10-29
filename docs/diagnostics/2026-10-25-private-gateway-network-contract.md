# Private gateway network contract — 2026-10-25

M383 integrates the pure network portion of the runner gateway adapter. It binds
bridge creation/inspection to a validated retained gateway selection, with exact
runner/project/generation/operation, plan, workspace and input-generation labels.
The constructor does not inspect an engine or grant runtime authority.

Inspection requires one exact network ID, ordinary local IPv4 bridge options,
private canonical RFC1918 allocation and either no participant or exactly the
recorded gateway. The latter also binds endpoint identity, usable IPv4 address,
subnet width and absence of IPv6. Wrong labels, allocations, configuration source,
participants, IDs, omitted booleans and ambiguous/repeated JSON fields refuse.
Unambiguous extra Docker metadata remains compatible. Names and arguments are
deterministic; caller changes cannot mutate generated labels.

Eight top-level / 76 race checks and vet pass, including the existing bridge
contract checks. The initial five / 67 checks predate the ambiguity cases. After
byte-identical integration, six / 74 overlapping checks pass from the actual
checkout. These are synthetic source tests; do not add overlapping counts or
claim Docker execution. Full schema43 evidence remains the earlier M381 baseline.

No persistent format, public route, installed plugin, host engine or ship changed.
The contract still needs concrete engine/filesystem/input preparation, container
configuration and publication checks, runtime qualification and public lifecycle
integration. An ordinary bridge is not an air gap; the experiment's no-NIC VM
remains essential. The running VM15/audit30 sources remain unchanged. Its runtime
result and the separate concurrent-project memory admission are still pending.

This supersedes M382's pending network-contract source only. See the adjacent JSON
for source hashes, test results and the retained source-only evidence directory.
