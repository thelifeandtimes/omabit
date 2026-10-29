# Theme appearance ownership and inactive hooks

The development Theme helper holds a shared HOME-local lock for its lifetime,
including transport drain. Different XDG account roots do not bypass that lock.
The selected profile has a persistent generation: same-profile restart keeps it,
while switching away/back invalidates delayed hook intent. Generated Lua checks
the applied owner as well as theme name, so profiles using the same stock theme
cannot both execute their overrides. Exact old include blocks acquire that guard;
the referenced old file is preserved. Edited includes/files and unknown ownership
formats refuse application. New adapter3 code/content receipt is one atomic file.

Hooks now only queue selected-profile intent. The owner handles theme override
suppression/reload. The old second hook-to-UI publication path was removed after
inspection showed it could bypass the ownership decision. Explicit diagnostic IPC
remains supported. Font notifications survive same-profile restart. The installer
includes the ownership module and accepts the previous declaration-file inventory.

213 Python checks pass, including real Lua stable/dev/edge/stable execution (one
configuration call each), previous files retained, real process lock contention,
actual serve exclusion until prior helper exit, delayed hooks, user edits, unknown
formats, missing owner recovery and private installed-module import/upgrade.
20 native QML checks, real SDK/hook tests and separate5-profile+20-legacy QML checks
pass. Connected actual Theme/DMs helper lifecycle remains passing:6 drains,
12 deleted channels,3 total application pokes, retained unknown send and no replay;
state-confirmed invitation/read effects survive another restart. All test scopes
are inactive; retained evidence excludes private keyrings and account metadata.

Supersedes M258's uncoordinated appearance/hook limitation for cooperating development
helpers. No actual compositor, older-helper compatibility, automatic collection
rollback or full activation claim. Unreceipted legacy overrides are preserved and
refused pending explicit migration; this also supersedes the former unconditional
legacy override repair. Production ships, installed plugins and host settings are
untouched. collectionReady remains false.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-theme-appearance-owner`.
