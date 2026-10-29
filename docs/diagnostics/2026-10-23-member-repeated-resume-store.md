# Repeated member replacement through retained causal history

Schema35 permits multiple replacement receipts for one project member while each
completed retirement remains consumable once. The migration rebuilds the parent
resume table and its readiness child in one transaction, with foreign keys enabled
and the original bodies copied verbatim. Unknown references, interrupted migrations
and failed initialization roll back without changing prior state.

New lifecycle work selects the latest exact causal binding. Historical validation
reconstructs the prefix belonging to each old receipt. The private exit, retirement,
resume and readiness queues use those prefixes and retain old uncertain Stops;
a completed replacement accounts for an old container without rewriting its history.
Missing later-generation adapter capabilities are refused before admission.

Six queue cases reach generations3/4 or preserve injected uncertainty. Sixteen
persistence interruption boundaries, nine migration rollback cases, original
schema34 reproduction and unchanged old-reader refusal pass. The full managed race
suite and final vet also pass; exact durations and artifact hashes are in the JSON.
The original34 fixture includes actual generation2 Stop history; the successor has
two same-member replacements and a dispatched second Create.

Earlier fixture failures remain recorded: historical selection was too strict,
a synthetic exit invented a different completed-Stop timestamp, and a two-second
fixture wait was too short under the race detector. Product admission/deadlines
were not relaxed. A full run hit temporary quota during concurrent extraction;
a later run reached Go's aggregate ten-minute limit while still progressing. The
successful run uses twenty minutes for the whole suite and no concurrent conversion.

This supersedes M110's one-resume-per-member persistence limit and completes the
source scheduling slice. It does not qualify later-generation real Docker behavior,
runner application recovery, changed-peer readiness or public lifecycle APIs.
The fresh VM22 fixture is compiled and refuses execution outside its disposable
no-NIC guest. No installed database, runtime or plugin was touched.
