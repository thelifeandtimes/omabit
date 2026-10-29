# Real native bar and Wayland popup qualification

Actual guarded Omabit/Theme/DMs bodies now pass stable/dev/stable with the complete
unchanged native Bar.qml, BarWidgetRegistry, widget registration methods,
PluginBarApi and KeyboardPanel. Native scoped shell callbacks remain integrated.
A private Sway/wlroots headless pixman compositor renders one1280x720 output;
there is no host display, GPU, network route, HOME, keyring or ship access.

Six real keyboard-panel openings, three actual DMs overlays, host/payload popout
ownership, peer navigation and private-socket keyboard delivery pass. Each DMs
cycle first activates the focused Close button with Enter, then reopens and closes
with Escape. Twelve screenshots are retained and representative images were
visually inspected. The run creates exactly nine primaries and nine widgets, with
no duplicate IPC registrations. Guard loss and busy shutdown close already-open
Theme popups; actual DMs surfaces also unmap on shutdown. All guards exit and the
owned compositor exits zero. Closed-admission hooks pass six times.

## Two source fixes demonstrated

- DMs had Keys.onEscapePressed on PanelWindow, which is not an Item. The handler
  now belongs to the card Item, where events from focused buttons bubble. The
  controlled pre-fix trial passes Enter but leaves the overlay mapped on Escape;
  the new source passes both. Installed DMs remains untouched.
- WidgetHost previously closed from the primary phase signal, which can run before
  its derived admitted binding changes. A mapped popup remained open after guard
  loss for the entire bounded wait. It now also closes on admitted changing false.
  The final real-surface test passes this case. No persistent format changes.

The connected native original-guardian/coordinator regression passes after both
fixes: fifteen subscriptions, ten channels deleted, six saved sessions unchanged,
zero new login and a byte-identical unresolved DM without replay. Two original
subreaper receipts verify. All88 private collection tests pass; nine earlier M294
generations still verify unchanged. Source hashes and all18 scopes were checked.

## Preparation, failed trials and limits

Compositor preparation downloads142 signed packages and extracts them only into
a fresh private tree with no package hooks or host installation. The first fetch
failed on a stale database404; fresh private database copies resolved it. The
first surface smoke failed on harness QML syntax; the corrected smoke passes.

The native harness initially derived layout from incrementally registered widgets,
causing rebuilds/duplicate IPC; it now selects a stable three-slot collection from
the native enabled set. A stock empty center slot makes four slots in total.
Disabled entries must be filtered through isEnabled. The native bar retains
placeholder entry APIs until explicit prunePluginBarApis at the ready boundary;
the test records before/after and verifies only the three active APIs remain.
This is not proof of automatic API pruning in the full shell.

Keyboard trials04/05 alone were inconclusive because virtual-keyboard creation
needed a focus roundtrip. The dedicated negative includes a successful Enter
control and delays before/after Escape. Trial07 passed closed-popup cases;08/09
exposed the previously open-popup defect. Trial10 passes the stronger checks.
Earlier failed trials are retained rather than silently reclassified.

Accounts are empty in the visual test; colors and runner/appearance tools are
private fixtures. Theme's unavailable-palette banner is expected with the inert
appearance tool. The compiler scanner emits qs import-path warnings while the
components resolve via QML_IMPORT_PATH. Sway/Qt emits text-input focus teardown
warnings on virtual-keyboard disconnection; successful key assertions do not
qualify IME behavior. Sway proves generic Wayland/layer-shell behavior only, not
Hyprland focus, input-method, monitor or animation semantics. This is not the full
native shell, connected traffic plus compositor in one run, a public profile
command, cross-root ownership or production deployment. Native bar layout must be
coordinated with the collection rather than derived from arrival order.

M297 supersedes M295/M296's simplified-bar/inert-popup limit for this exact private
scope. It narrows M292's closing-UI claim: withdrawal of IPC alone did not prove an
already mapped popup had closed. Broader lifecycle/ownership gates remain.

Evidence: /home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-native-wayland
