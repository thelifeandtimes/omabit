package main

// Commands with no ordinary flag parser still expose side-effect-free help.
var desktopCommandHelp = map[string]string{
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
