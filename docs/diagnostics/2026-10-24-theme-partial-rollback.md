# Partial appearance failure and previous-profile recovery

The first actual Theme/DMs service trial exposed a rollback stall: after a native
theme-set substitute changed the working theme and failed, current/theme.name was
urbit-synced.qualification-dev. The returning stable helper had no applying intent,
so local capture failed before it could read/adopt the stable hub. It also reported
Following shared appearance between retries despite retaining a capture error.
The failed trial, source and desktop events are retained.

Theme now recognizes a temporary managed name only under its lifetime HOME lock
and with an intact owned generated-directory receipt. On initial adoption it can
read the existing authenticated hub, save applying intent, apply it and verify
readback without needing the broken local capture first. It never seeds an empty
hub or invents local state. Offline/missing hubs, unknown/edited directories and
failed readback remain unresolved. Existing uncertain publication intent/IDs are
preserved. An outstanding error no longer reports Following shared appearance.

The repeated actual Service/helper/native-loader/private-keyring fixture passes
failures after theme, window Lua, shell style and font effects. Each failed
candidate is explicitly drained and its exact helper exit observed before stable
is enabled. It restores the previous profile's controlled colors, font, rounding,
shell values, theme file and owner. A deliberately failed restoration remains
pending/Needs attention even after its visible font changed; a later verified
retry succeeds.12 transitions delete28 channels;4 ordinary Talon pokes occur and
zero appearance publications occur. Candidate applying intent survives drains.
All222 Python and58 focused checks pass. The original connected zero-write mode
also remains passing. The applied harness exposes --connected --appearance-rollback
as a separate mode; host-entry/flag guards pass and all scopes are inactive.

This supersedes M260's wholly untested partial-application restoration gap for
this online, explicitly orchestrated private scenario. It is not an automatic
collection rollback command, an immutable offline checkpoint, a compositor test,
or proof of native font/shell restart/background/app-retint side effects. Native
setter source confirms that broader fanout and is hashed for the next gate.
No production ship, installed plugin, host service or desktop configuration changed.
collectionReady remains false.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-theme-partial-rollback`.
