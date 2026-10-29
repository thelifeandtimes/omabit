# Native collection artifact identity

`native/collections/artifacts.py` prepares the supported Omabit/Theme/DMs runtime
set from explicit SHA-256/size/executable inventories. Preparation copies only
those files, declares separate non-default profiles/IDs/labels, keeps every QML/JS
and helper beneath a content-derived generation path, and generates a loaded
identity/drain IPC component. It never writes the top-level native manifest,
changes shell configuration, enables a plugin, or touches a ship. Omabit requires
its paired tools and desktop contract; its mutable bootstrap is not an input.

The format1 descriptor binds source and transformed payload inventories, role,
profile and environment. Generated identity code is checked against the pinned
descriptor. Source drift, links, unsupported layouts, altered payloads/wrappers,
unknown formats and stale loaded URLs refuse. Hidden interrupted stages remain;
exact retry verifies existing bytes and re-syncs publication. File hashes and
read-only modes provide same-user integrity, not wallet secrecy or publisher trust.

Theme/DMs use their existing application-profile namespaces within ambient XDG
roots. Only Omabit accepts a per-instance environment. Preparation explicitly
refuses an invented Theme/DMs environment override. A late code review also fixed
literal path text being reinterpreted as QML template placeholders.

62 private tests pass:14 artifact cases plus48 prior collection/recovery cases.
They include a real preparation-process SIGKILL before publication, lost-publication
response/retry, old-generation preservation, tampering, default-profile refusal and
exact loaded observation validation. Full runtime evidence uses the complete copied
native PluginRegistry, its real scanner/watcher, exact native setPluginEnabled IPC,
and copied native service/panel loaders in a no-NIC private namespace. Synthetic
plugin bodies with real QML/JS imports and Python helpers isolate artifact identity.
Ten immutable generations verify. A native rescan selects version2 while the live
kept service/helper truthfully retains version1; treating selection as loaded2 is
rejected. Explicit disable/enable loads2. Stable/dev/stable/edge/stable creates15
distinct instances across the service and overlay paths. Every source hash matches.

Trial01 failed before namespace launch because the extraction expression omitted
an IPC return annotation. Subsequent native and unit trials passed; final trial04
uses final source. Intermediate evidence is retained, and all eight scopes are
inactive. No account/credential files or existing desktop state were accessed.

This narrows M285's missing native identity/preparation gate. It does not qualify
real supported plugin bodies under these wrappers, native bar widgets, the complete
shell/compositor, durable loader admission or a public switch command. Existing
actual-plugin recovery evidence remains M285. The generated inherited runtime
component begins its helpers when instantiated: it must sit behind a durable
admission layer before user-facing installation is safe. Native service enable
IPC is available externally; self-scoped PluginRegistryApi need not be expanded.

Source follow-up: actual Omabit Service/Panel still have canonical monitor/setup
IPC targets. They must derive targets from the experimental module ID before real
native widget integration. Panel service lookup also needs to accommodate a gated
loader payload. Preserve standalone/default behavior while addressing those next.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-native-artifacts`.
