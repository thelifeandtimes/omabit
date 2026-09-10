# Tend keyboard and accessibility reference

Tend uses Omarchy's theme colors, spacing, and font sizing. It adds no custom
motion, so desktop reduced-motion preferences do not need a separate override.
Standard Qt controls retain their native focus indicator and tab order.

## Keyboard reference

| Key | Action |
| --- | --- |
| `Escape` | Close the overlay |
| `Ctrl+N` | Focus quick add |
| `Ctrl+Shift+N` | Focus new-list entry |
| `Ctrl+F` | Focus reminder search |
| `Ctrl+,` | Open or close reminder policy settings |
| `Alt+1` … `Alt+6` | Today, Scheduled, All, Flagged, Assigned, Completed |
| `Up` / `Down` | Move through the focused reminder list |
| `Enter` | Open the focused reminder editor |
| `Space` | Complete or uncomplete the focused reminder |
| `Tab` / `Shift+Tab` | Move through controls in either direction |

Quick capture focuses its title field when opened. The full overlay focuses
quick add after an authenticated session is available, or the Eyre URL field
during onboarding.

## Assistive technology

- The bar widget identifies its action, count basis, count, and connection
  state through Qt Accessibility.
- The reminder list exposes list/list-item roles and labels. Completion and
  batch-selection checkboxes include the reminder title in their accessible
  name.
- Fields and non-self-describing combo/spin controls have explicit accessible
  names. Buttons and ordinary checkboxes retain their visible text labels.
- Password entry uses Qt's password echo mode; Tend clears the `+code` field
  immediately after submitting it.
- Owner availability and read-only state are expressed as text in addition to
  color. Connection and mutation controls expose disabled state natively.

Screen-reader speech, compositor focus presentation, high-DPI scaling, and
keyboard layout handling ultimately depend on the installed Qt/Quickshell and
Omarchy versions. Release candidates still require a manual pass on the
supported desktop image with Orca (or the user's preferred AT), 100% and 200%
scaling, keyboard-only operation, and both light and dark themes.

## Release checklist

- Complete onboarding, quick add, list/reminder editing, scheduling, batch
  actions, policy editing, export, disconnect, and reconnect without a pointer.
- Confirm every focusable element has a visible focus state and a meaningful
  spoken name.
- Confirm Today/read-only/connection states remain understandable without
  color perception.
- Confirm the overlay remains usable at 200% scale and with long list/reminder
  titles.
- Confirm no action depends on hover and no animation ignores reduced-motion
  preferences.
