# Profile-scoped Omabit IPC

The actual Omabit Service derives its monitor IPC from its injected manifest ID.
The actual Panel derives widget/setup IPC from moduleName. Default canonical IDs
remain the fallback; experimental qualified instances use separate targets. Monitor
status now reports its module/profile. Panel's native active color explicitly uses
Commons.Color, resolving an ambiguous unqualified Color lookup observed in the test.

`scripts/test-omabit-service-drain-offscreen.py --readers --panel` now loads actual
Service, Panel, monitor and setup readers with copied native Ui controls. Only the
compositor panel/focus, color values and exact-argv tool results are inert. Private
HOME/XDG/PID/network namespaces exclude the host desktop, runners and credentials.
The final source passes five stable/dev/stable/edge/stable cycles. Each verifies
monitor identity plus setup-open/widget-close through scoped IPC, checks canonical
targets absent in the native inventory, then confirms those profile targets disappear
after unloading. Foreground/auth work, delayed readers, three killed-helper fences
and lost-reader-owner refusal still pass. The prior canonical-only service fails
the new profile endpoint test. All six transient scopes are inactive.

Quickshell returns exit0 with the literal `Target not found.` for unknown targets.
The first two trials therefore failed an incorrect exit-code assertion. Trial02
retains the actual response and native target listing; later checks require both
response content and inventory absence. The first real Panel trial also failed on
its ambiguous Color reference, a missing copied icon, and incomplete synthetic
runner display metadata. The product color lookup and fixture inputs were corrected;
the final panel run has none of those warnings/errors. Failed trials remain retained.

This supersedes M286's remaining steady-state Omabit monitor/setup namespace gap,
not durable native admission or a complete profile switch. These actual components
were created with initial property injection in a private shell fixture. The stock
loader's creation-before-injection interval and coexistence with an already loaded
canonical component still need explicit proof/fencing. Prepared generation wrappers
must eventually pin identity before application helpers or IPC start, and a gated
host must expose the proper service payload to widgets. No installed plugin, ship,
production service, tmux session or temp-ur data was accessed or changed.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-scoped-omabit-ipc`.
