# Native scoped shell APIs with guarded supported plugins

Actual Omabit/Theme/DMs payloads now run in the private fixture with the unchanged
native PluginShellApi and PluginBarStateApi plus27 exact shell factory/routing
functions. The actual native bar API also receives a scoped shell. No ShellRoot
object is directly supplied to a payload; public manifests omit host trust markers.

Stable/dev/stable verifies nine scoped payload activations: own service lookup
returns only the matching outer service host (DMs has no service), foreign service,
summon and hide requests are refused, and generic configuration mutation is refused.
Actual DMs summon/open/dismiss passes through the copied native routing callbacks
on each cycle. The M295 navigation/popout and closing-admission cases pass with
these APIs, along with the existing stale-permit/guard-loss native checks. All
recorded guards exit; source hashes match and the single scope is inactive.

This is source/private runtime qualification of injection and callbacks, not proof
of a security sandbox. It does not qualify the full shell, API cache revocation/
pruning, real bar layouts, compositor/focus/animation, cross-root ownership or
production deployment. The fixture supplies private layout/cache/config objects;
accounts are empty and desktop/Omabit tools inert. Installed plugins, protected
ships, tmux and temp-ur remain untouched. It narrows M295's scoped-facade gap only.
Evidence: /home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-native-shell-api
