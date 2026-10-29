# DMs profile launcher isolation

Experimental DMs profiles now create distinct Tlon Web launcher filenames,
visible profile labels and exact plugin ownership markers. Removing a dev
launcher preserves stable and edge; a copied foreign marker does not grant
ownership. Existing unmarked-by-instance legacy launchers remain recognized only
by the default profile. The launcher still points at its own plugin helper and
contains no ship URL, cookie or code.

All 39 affected launcher, authentication, bridge and stream tests pass in 0.080s.
Launcher tests write only temporary directories. No installed desktop entry,
plugin, session or configuration was changed. This closes the launcher collision,
not QML/IPC/hook isolation or whole-collection switching.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-dms-profile-launchers`.
