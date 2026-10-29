# Native shell API lifetime with real bar surfaces

The private M297 Wayland harness now additionally uses the exact native
publicBarWidgetSnapshot, pluginApiActive, prunePluginApis and syncPluginApis
functions. Registry changes sync APIs before service changes; widget-registry
changes sync APIs as in the native shell. PluginShellApi methods remain unchanged;
only construction/destruction markers and an instance token are added to its
private copy for observation.

Stable/dev/stable passes three complete unload observations: scoped shell caches,
descriptors and bar-state caches empty, no owned bar objects/click targets/popouts,
and recorded destruction of each previously admitted API instance. Nine admitted
API instances are distinct across activations. All13 created APIs, including
inert/recreated guarded hosts, have matching destruction records. Actual popup,
Escape/Enter, guard-loss closure and stale-admission tests still pass. Sources
match, all five scopes are inactive and the owned compositor exits zero.

Trials1/2 incorrectly required every native bar API to disappear while the empty
layout still configured a slot. Trial3 required its shell reference to equal null.
Qt can retain a wrapper for a destroyed QObject: property access yields undefined,
and converting that wrapper to a string can throw (trial4). The final check allows
only configured bar-entry placeholders with no usable shell API and no owned UI
objects. It does not relax the plugin-cache/destruction requirement. On later
profile replacement even that placeholder disappears. All trials are retained.

This narrows M296/M297's API cleanup gap for the tested supported set. The test
still explicitly prunes unused native bar APIs at transition boundaries. It does
not qualify the complete shell, first-party or arbitrary bar-entry capabilities,
revocation of independently captured JavaScript callbacks, Hyprland/IME behavior,
cross-root runtime ownership or public profile commands. No product change was
needed for this checkpoint; installed sources and all real ships remain untouched.

Evidence: /home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-native-api-lifecycle
