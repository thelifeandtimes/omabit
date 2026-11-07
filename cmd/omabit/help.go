package main

// Commands with no ordinary flag parser still expose side-effect-free help.
var desktopCommandHelp = map[string]string{
	"ship-routes": `Usage: omabit [--config-dir DIR] ship-routes --runner NAME --expected-runner-id ID --instance INSTANCE

Read the live ship incarnation, saved URL choices and current automatic routes.
Local routes belong to the runner's host. Registration is not a reachability or
login claim. This command sends no credentials and configures no network service.`,
	"ship-routes-configure": `Usage: omabit [--config-dir DIR] ship-routes-configure --runner NAME --expected-runner-id ID --request-file FILE

Save a reviewed ship URL update: {selection,update:{binding,expected_revision,settings}}.
Use the selection, binding and revision from ship-routes; edit settings only.
Settings contain routes and disabled_automatic arrays. Manual/tunnel routes need
id, origin, network, scope, source and enabled. Origins cannot contain paths or
credentials; public routes require HTTPS. Disable automatic routes by their IDs.
A changed runtime, pier or settings revision refuses the edit. Inspect again
after a lost response. This does not expose the ship, configure DNS/tunnels, grant
application access, or stop an external proxy when a route is disabled.`,
	"desk-review-frontend": `Usage: omabit [--config-dir DIR] desk-review-frontend --runner NAME --expected-runner-id ID --stage-id STAGE_ID
Review built assets already transferred with desk-plan-stage and desk-stage.
Requires desk-frontend-v1, a local installation and this ship's Ames glob.
Reads the artifact hash, Clay, Docket, workflow and release checks without upload.`,
	"desk-plan-frontend": `Usage: omabit [--config-dir DIR] desk-plan-frontend --runner NAME --expected-runner-id ID --stage-id STAGE_ID [--checks-file FILE]
Save the complete JSON plan before submission. Distributed updates require every
configured check bound to both clay_hash and artifact_digest; private preparation
uses an empty check array. Limits: 2048 files/32 MiB; index.html required; desk.js
reserved; terminal-NUL assets refused because native multipart would truncate them.`,
	"desk-upload-frontend": `Usage: omabit [--config-dir DIR] desk-upload-frontend --runner NAME --expected-runner-id ID --plan-file FILE
Alternatively use --plan-json JSON. Submit one unchanged saved plan.
Both Commit and Publish gates must allow; both are blocked by default. Upload can
immediately replace served assets and commit desk.docket-0 in Clay. Native HTTP
cannot atomically enforce a reviewed revision; drift fails verification afterward.
Read desk-operation after a lost reply and use desk-reconcile without re-uploading.
An observed glob/source result does not prove HTTP or application health. Check the
separate session_cleanup receipt; unconfirmed means owner-session cleanup is unknown.
Reconcile generated docket changes with source ownership before the next commit.`,
	"desk-receiver-status": `Usage: omabit [--config-dir DIR] desk-receiver-status --runner NAME --expected-runner-id ID --instance INSTANCE --desk DESK
Read the current Kiln subscription and distinguish pending source from received
Clay/application stages. Requires desk-receiver-v1. Never installs or publishes.`,
	"desk-review-install": `Usage: omabit [--config-dir DIR] desk-review-install --runner NAME --expected-runner-id ID --instance INSTANCE --desk DESK --publisher-ship SHIP --publisher-desk DESK --expected-source-hash HASH [--publisher-instance INSTANCE]
Review an unused receiver desk and its continuing publisher without installing.
Project receivers require a current publisher instance from the same project.
The expected hash verifies initial source arrival; future publisher updates continue.`,
	"desk-plan-install": `Usage: omabit [--config-dir DIR] desk-plan-install --runner NAME --expected-runner-id ID --instance INSTANCE --desk DESK --publisher-ship SHIP --publisher-desk DESK --expected-source-hash HASH [--publisher-instance INSTANCE]
Review and save a complete immutable receiver installation plan. Requires
desk-receiver-v1. The ship creation gate and receiver commit gate must allow the
later install. Planning grants no permission. Source workflow must be mounted.`,
	"desk-install": `Usage: omabit [--config-dir DIR] desk-install --runner NAME --expected-runner-id ID --plan-file FILE
Alternatively use --plan-json JSON. Submit the unchanged saved plan once.
Creation and commit are blocked by default; the received desk remains private.
Read desk-operation with the saved ID after a lost reply; desk-reconcile observes
uncertain effects without repeating installation. Closing gates does not cancel
an existing Kiln subscription or prevent its publisher's future updates.`,
	"desk-review-commit": `Usage: omabit [--config-dir DIR] desk-review-commit --runner NAME --expected-runner-id ID --stage-id STAGE_ID
Read the original staged source, current workflow/policy, selected runtime and Clay.
Reports exact change/conflict counts and at most 256 sorted details of each.
Preserves unowned/generated paths and grants no permission or commit authority.`,
	"desk-plan-commit": `Usage: omabit [--config-dir DIR] desk-plan-commit --runner NAME --expected-runner-id ID --stage-id STAGE_ID
Review the staged source and output an immutable commit plan with a new operation ID.
Save its complete JSON before submission. No-op and conflicted plans are read-only
results; neither may be submitted. A published desk requires commit and publish gates.`,
	"desk-commit": `Usage: omabit [--config-dir DIR] desk-commit --runner NAME --expected-runner-id ID --plan-file FILE
Alternatively pass the unchanged full plan with --plan-json JSON (not both).
Requires desk-commit-v1; blocked by default. Rechecks source, runtime, workflow,
policy and Clay before one journaled dispatch. Preserves unspecified/generated files.
The observed receipt proves source acceptance, not application compilation, activation
or publication. After a lost reply, query desk-operation using the saved ID; use
read-only desk-reconcile for uncertainty. Never generate a replacement ID to retry.`,
	"desk-plan-stage": `Usage: omabit [--config-dir DIR] desk-plan-stage --runner NAME --expected-runner-id ID --instance INSTANCE --desk DESK --source-dir DIRECTORY
Read the prepared local source tree and saved workflow, then output a frozen
transfer plan. Save its full JSON before uploading. No source is sent to a ship,
and no commit or publication permission is granted. Requires the Unix workflow.`,
	"desk-stage": `Usage: omabit [--config-dir DIR] desk-stage --runner NAME --expected-runner-id ID --plan-json JSON --source-dir DIRECTORY
Upload the exact planned source into a private runner cache outside every pier.
Requires desk-source-staging-v1 and operator access. Does not write a live mount,
commit, publish or alter permissions. A lost reply must be reconciled using the
original stage ID. Limits: 8192 files, 256 MiB per tree, 32 MiB per file, 2 MiB
manifest; cache holds at most 16 stages and 1 GiB of source.`,
	"desk-stage-status": `Usage: omabit [--config-dir DIR] desk-stage-status --runner NAME --expected-runner-id ID --id STAGE_ID
Verify the original operator's staged bytes and read its immutable receipt.
This is transfer evidence, not a Clay commit or release receipt. Changed or
missing staged content fails visibly; it is never silently repaired.`,
	"desk-stage-discard": `Usage: omabit [--config-dir DIR] desk-stage-discard --runner NAME --expected-runner-id ID --id STAGE_ID --source-digest SHA256
Remove only the original operator's cached upload with this ID and digest. Refuses
while a desk operation on its target is unresolved. Supports interrupted uploads
and discards; never removes original source, live mounts or Clay. Malformed cache
metadata is retained for review. An already absent stage returns success.`,
	"desk-reconcile": `Usage: omabit [--config-dir DIR] desk-reconcile --runner NAME --expected-runner-id ID --id OPERATION_ID
Observe an uncertain creation, staged commit, release, frontend upload or receiver install on its exact selected runtime.
Commit reconciliation requires its retained stage and verifies the whole result.
Never repeats the effect or changes permissions. Query desk-operation afterward;
unavailable evidence leaves the outcome uncertain.`,
	"desk-source-status": `Usage: omabit [--config-dir DIR] desk-source-status --runner NAME --expected-runner-id ID --instance INSTANCE --desk DESK

Compare the configured Unix source with Clay on the selected running ship.
Requires desk-source-status-v1 and desk-workflow-settings-v1. Performs no source
write, commit or publication. Remembers a canonical baseline only when both trees
match. Without a baseline, different means unclassified drift. Unknown or failed
comparison is not clean. Details show at most 256 paths plus the exact total.
`,
	"desk-settings": `Usage: omabit [--config-dir DIR] desk-settings --runner NAME --expected-runner-id ID --instance INSTANCE [--desk DESK]

Read ship purpose or the desk's source and release configuration. Follow the
configured checks and release instructions. This command cannot change settings
or grant permissions; use Setup to author workflow settings.
`,
	"desk-list": `Usage: omabit [--config-dir DIR] desk-list --runner NAME --expected-runner-id ID --instance INSTANCE

Read installed desk names from the selected running ship. Discovery does not
create desks or grant permissions; unsupported or unavailable runners fail visibly.
`,
	"desk-inspect": `Usage: omabit [--config-dir DIR] desk-inspect --runner NAME --expected-runner-id ID --instance INSTANCE --desk DESK
Read the selected live desk and its effective permissions. DESK omits the % prefix.
Requires a compatible runner and operator access; never creates or commits a desk.`,
	"desk-plan-create": `Usage: omabit [--config-dir DIR] desk-plan-create --runner NAME --expected-runner-id ID --instance INSTANCE --desk DESK
Observe desk absence and output a creation plan with an immutable request ID,
runtime identity, generation and policy revision. Save the JSON before submission.
This read-only step grants no permission and does not change Setup.`,
	"desk-create": `Usage: omabit [--config-dir DIR] desk-create --runner NAME --expected-runner-id ID --plan-json JSON
Submit the exact saved desk-plan-create output. The runner rechecks the live desk,
runtime and current Setup policy. Creation is blocked by default; new desks inherit
commit/publishing gates. Read the operation receipt to learn the outcome. After a
lost reply, query its ID or resubmit the same plan; never invent a replacement ID.`,
	"desk-operation": `Usage: omabit [--config-dir DIR] desk-operation --runner NAME --expected-runner-id ID --id OPERATION_ID
Read a durable desk receipt without dispatching an effect. Acknowledgement alone
is not completion. Interrupted or uncertain work is never replayed automatically.`,
	"service-upgrade-status": `Usage: omabit [--config-dir DIR] service-upgrade-status
Read the existing local owner/enrollment and compare the installed runner with the
current desktop bundle. Reports an unfinished retained attempt before a new target.
No downloads, authorization, key generation, service changes or ship mutations.`,
	"service-upgrade": `Usage: omabit [--config-dir DIR] service-upgrade --expected-runner-id ID --target-sha256 SHA --acknowledge-host-changes
Explicitly update local owner services to the exact target shown by service-upgrade-status.
Run as the desktop user. Downloads pinned launcher dependencies, requests OS authorization,
preserves identities/access/ships and verifies readiness. Busy management refuses safely.
Retry keeps the original executable/bundle; a newer desktop cannot retarget a pending update.
Does not update OS packages, remote runners or the historical offline schema19 bridge.`,
	"theme-preview": `Usage: omabit theme-preview
Read the current local appearance as a palette; no ship or preference writes.`,
	"theme-status": `Usage: omabit [--state-dir DIR] theme-status
Read saved automatic-publishing preferences; no ship traffic.`,
	"theme-sync": `Usage: omabit [--config-dir DIR] [--state-dir DIR] theme-sync
Publish the current local palette to independently enrolled ships with automatic
publishing enabled. Reports partial failure; does not retry failed writes.`,
	"theme-auto": `Usage: omabit [--config-dir DIR] [--state-dir DIR] theme-auto NAME --expected-ship-id ID --enabled=true|false
Set the selected enrollment's automatic-publishing preference. The enrollment ID
must match; this preference change does not itself publish a palette.`,
	"theme-publish": `Usage: omabit [--config-dir DIR] [--state-dir DIR] theme-publish NAME --expected-ship-id ID
Publish the local palette once to the exact enrolled ship. Network write.`,
	"theme-migrate": `Usage: omabit [--config-dir DIR] [--state-dir DIR] theme-migrate [--legacy-dir DIR] [--apply --expected-digest DIGEST]
Default: preview the legacy Theme state and its digest. Apply pauses the legacy
publisher and transfers its sessions/preferences without logout. Review the
preview before applying; do not repeat an uncertain migration blindly.`,
	"copy-code": `Usage: omabit [--config-dir DIR] copy-code --runner NAME --expected-runner-id ID --instance INSTANCE --revision N --container-id FULL_ID
Retrieve the exact running ship's +code and hand it to a private Wayland clipboard
provider for at most 30 seconds. Requires wl-copy. Does not print or save the code;
exiting releases only this process's own clipboard selection.`,
}
