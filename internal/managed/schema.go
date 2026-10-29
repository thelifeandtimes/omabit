package managed

// StateSchemaVersion is also queried from the built runner by release tooling.
// Schema40 retains explicit recovery continuation ownership, checkpoints and parent attribution.
// Schema39 retains explicit read-only recovery observations and exact checkpoint ancestry.
// Schema38 retains coordinated anchor recovery, exact source history and child creators.
// Schema37 fences Stop/lifecycle intents bound to current-context readiness.
// Schema36 retains separate current-context readiness without rewriting old observations.
// Schema35 retains repeated member replacements without erasing prior generations.
// Schema34 fences replacement-bound Stop intent and historical generation selection.
// Schema33 fences read-only readiness observations of completed replacements.
// Schema32 fences explicit retained-pier replacement intent and dispatch.
// Schema31 fences explicit stopped-member retirement and absence evidence.
// Schema30 fences separate read-only observations of retained Stop exits.
// Schema29 fences initial graceful member Stop intent and exit evidence.
// Schema28 fences historical read-only member readiness attempts.
// Schema27 fences initial member launch intent and workspace/container evidence.
// Schema26 introduced storage claims; schema25 retains initial anchor Start.
// Migration adds empty project tables while preserving legacy bodies and replay
// receipts. Opening the store does not create or adopt any runtime resources.
// Schema41 fences distinct private-chain project ownership and shared reservations.
// Schema42 fences private workspace ownership and interrupted directory setup.
// Schema43 retains gateway initialization and separately attributed workspace/gateway observations.
// Schema44 retains the initial private member launch and refuses implicit retry.
// Schema45 binds captured first-boot pier identities to version2 member launches.
// Schema46 retains first-boot capture before later running observation.
// Schema47 permits empty project network reservations; older runners must refuse them.
const StateSchemaVersion = 47
