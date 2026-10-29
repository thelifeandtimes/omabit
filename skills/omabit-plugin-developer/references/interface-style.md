# Omabit plugin presentation conventions

Use the Theme and DMs reference clients together. They are independent plugins;
copy a small presentation component when useful instead of depending on the other
plugin being installed. `PluginHeader.qml` and `ConnectionChoices.qml` are identical
vendored reference components. Their transport and account models remain separate.

## Menubar popup

- Use native `Ui.BarIconButton`, `Ui.KeyboardPanel`, `Ui.PanelHero`,
  `Ui.PanelSeparator`, `Ui.PanelSectionHeader`, `Ui.Button` and `Ui.TextField`.
  Inspect the installed Omarchy SDK before adding properties. Do not replace the
  native border/focus system with a rectangle painted to resemble it.
- Target content width `Style.space(440)`, fitted to the available screen. Cap
  content height at `Style.space(720)` and scroll only when needed. Main sections
  use `Style.space(12)`; related form controls use `Style.space(8)`.
- Popup text uses `Commons.Color.popups.text`; window/dialog text and surfaces use
  `Commons.Color.menu.text/background/border`. Secondary text uses
  `Qt.darker(foreground, 1.4)`. Use `Color.urgent` for actionable failures. Respect
  current light/dark theme and user font scale; never bake a theme's hex colors.
- Header: application title with visible profile suffix, one uppercase metadata
  line, an application icon sized `Style.font.display`, and one trailing action.
  Use `PanelHero.trailingControl` so label widths reserve the action automatically.
  Header actions use `PanelActionButton`, `size: Style.spacing.controlHeight`,
  `bordered: true`, a meaningful tooltip, and `focusable: true`.
- Put a separator immediately below the header. Domain-specific controls follow.
  Use the selected @p in metadata when connected. Empty state has a short helpful
  sentence and one **Sign In** button. Do not fill it with protocol diagnostics.
- Long identities and titles elide with an accessible full label; paragraph text
  wraps. Set `Layout.minimumWidth: 0` on flexible text rows. Keep buttons reachable
  at narrow widths and with larger fonts. Never force an unbounded minimum width.

## Authentication and revocation

The popup's Sign In button closes the popup and opens the sign-in view. Use a
**Sign In** header, application/profile metadata, app icon and Close action.
After the separator present vertical choices with eight-unit spacing:

1. **Connect with Omabit** — bordered primary action; while pending, show
   **Opening Omabit…**, disable duplicates and impose a bounded IPC timeout.
2. **Enter direct credentials** — unbordered secondary action that reveals
   Ship URL and masked +code fields. Clear +code on cancellation, submission,
   transition, connection replacement and destruction, including undo history.

Omabit handoff opens the matching profile's Application Connections pane, with
this plugin selected. If Omabit is absent, explain the direct alternative in the
same view. Standalone authentication must stay usable. A successful connection
returns to application content; backend absence goes to backend setup, not login.

Disconnect must visibly clear old account data and return to sign-in. Ignore
late responses from the previous account generation. Preserve uncertain outbound
write receipts, but do not show old-account warnings as if they belonged to the
signed-out or newly connected account. Never automatically resend an uncertain
message after a session replacement. Session/account state drives the interface;
a successful IPC exit alone is not proof that authentication completed.

## Verify cohesion with behavior

Render the actual native controls with inert transport in a private offscreen Qt
fixture. Compare both reference plugins side by side, signed out and connected,
at normal and narrow widths, with dark/light colors and long identities. Check
header geometry, action sizing, focus, wrapping and state transitions. Capture
compact images only; do not copy piers or a whole desktop for visual testing.

Test real keyboard navigation, primary/secondary sign-in routes, cancellation,
revocation, late old-session events and separate profile state. Offscreen rendering
does not prove Wayland popup placement or compositor focus; verify those in the
separately named DEV installation before release. Do not reload the entire user's
shell or change production plugins to test a new component.
