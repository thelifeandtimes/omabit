# Guarded widget ownership at the native bar boundary

The negative control demonstrates a real wrapper defect: actual Ui.Panel passes
its own inner object to Bar.switchPanelFrom, while native slots identify the outer
WidgetHost. The copied exact native method therefore cannot find the current slot.
Popout ownership has the same boundary: native slot indicators refer to the host,
while the panel needs to recognize itself as the active popout.

WidgetHost now passes a scoped forwarding object to its payload. Presentation and
shell properties come from the supplied native PluginBarApi. Navigation/popout
owner arguments translate only this payload to its outer host; active popout maps
back to the payload. Peer-widget lookup unwraps matching supported hosts. Closing
admission refuses navigation, new popouts, new command dispatch and interactive
registration while retaining close/release/unregister paths. This is cooperative
lifecycle behavior, not a sandbox or wallet isolation boundary.

The actual Theme/Omabit/DMs widgets pass stable/dev/stable navigation and popout
checks using unchanged native PluginBarApi and copied native bar methods. Ten
closed live-payload probes refuse all three interactive operations; one recreated
host remains inert without a payload. Final native admission cases pass, all88
private collection tests pass, and the connected original-guardian recovery
regression still passes with15 subscriptions and preserved sessions/DM intent.
All nine prior M294 guarded generations remain readable unchanged; new WidgetHost
bytes produce distinct pinned generations. No persistent format migration occurs.

The genuine negative is retained. Positive trial1 passed; trial2's stronger check
incorrectly assumed every fenced host retained a payload. Explicitly recognizing
the refused empty host passes trial3. All six scopes are inactive, final sources
and whitespace checks match. Evidence excludes real credentials and host state.

This narrows M294's bar-ownership integration gap. The complete bar surface,
compositor/focus/animation behavior, native shell capability facades and cross-root
runtime ownership still need qualification. These tests use a fixture layout and
inert KeyboardPanel; they do not claim full desktop compatibility. No installed
plugin, protected ship, tmux or temp-ur data was touched.
Evidence: /home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-native-bar-ownership
