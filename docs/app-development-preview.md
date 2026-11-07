# Persistent Setup preview

Find **Omabit 0.4.0 Preview** in the desktop application launcher. It opens the
actual full Omabit monitor window, with the fleet sidebar and **Setup → App
development and publishing** selected. Close normally; saved fixture permissions
persist. It does not start automatically at login.

The preview uses simulated runners and ships. It does not run a collector,
runner or Urbit, and cannot create, commit or publish a desk. Other panels remain
navigable, with host/ship mutations disabled by the inert service adapter.

```sh
python3 ~/.local/share/omabit-policy-preview-040/launch.py start
python3 ~/.local/share/omabit-policy-preview-040/launch.py status
python3 ~/.local/share/omabit-policy-preview-040/launch.py stop
```

## Suggested review

1. Select Global, a runner, a ship, or a desk in the tree. The selected node's
   editor sits beside it on wider windows, or below it on narrower windows.
   Use **− / +** or **Ctrl+scroll** over the tree to zoom from 25% to 200%.
   Click the percentage to reset to 100%, or **Fit** to show the tree in the
   available space. Scroll or drag to pan; the permission editor stays its normal
   size. When the tree has keyboard focus, Ctrl+plus/minus and Ctrl+0 also work.
   All permissions initially default to Block; previous saved choices are retained.
2. Select `preview-local → ~zod → %base`, set Commit to Clay to Allow and save.
   With **Permission: Commit to Clay** and **Highlight: Allowed**, that desk and
   its path should stand out. Blocked parents keep their Blocked labels, and
   unrelated branches dim. Search matches runner/ship/desk names and their paths.
3. Select `~zod` and allow **Create new desks**. This is a ship permission; desks
   cannot authorize creation of siblings. Use **Review another desk's
   permissions** to inspect a new name. Commit/publish start at Inherit, using
   the saved ship/runner/global values. This field does not create a real desk.
4. In the header, switch **Remote fixture** to **Offline** and save a Global
   change. Its rollout stays pending and retains its last acknowledged values.
   Return to Online, review the current values, and save Global again. A runner
   unavailable at the original review needs that fresh review; Retry pending
   alone cannot authorize it.
5. Select **Online · observer** to inspect a remote branch with editing disabled.
   Try the sidebar, Back to Setup, resizing, scrolling, and Policy details.
   Close and reopen the preview to verify your saved values.

The map displays saved effective permissions. Unsaved edits do not recolor the
map. Sample desk names are fixture inventory; real installed-desk discovery is
not connected yet. The production tree can show saved desk policies and named
scopes without querying Clay. Creation/commit/publication CLI actions remain
subsequent pipeline work.

The connection simulation starts Online each launch. There is no theme picker
and no automatic policy reset. The upgrade from the earlier preview preserves
commit/publish choices and adds desk creation as Block, with scoped Inherit.
Its original JSON is retained as `policy-v1.before-create-desk.json`.
Production, the existing DEV installation and `trial040` are separate.

## Isolation and files

The install is `~/.local/share/omabit-policy-preview-040/`; the separately named
launcher is `~/.local/share/applications/omabit-policy-preview-040.desktop`.
Saved policies are in `data/state/policy.json`, logs in `data/state/window.log`
and `launcher.log`, and a capture of this window in `data/state/preview.png`.
One previous window log is retained. `source.json` records the source commit and
copied file hashes. The panel, Setup pane, tree and controls are copied unchanged;
window activation uses a fixture adapter without compositor-control access.

Bubblewrap supplies private user, PID, mount, IPC and network namespaces, a
read-only source snapshot, and private HOME/XDG directories. Only preview-owned
host data is writable. The sole host socket is the Wayland rendering socket;
host HOME, runtime, SSH, Docker, D-Bus, keyrings and runner sockets are absent.
The launcher uses an exclusive lock and an identity-checked process handle to
close only its own window. It does not install an Omarchy plugin or reload the
shell. The preview uses software rendering and a fixed preview palette.

## Prepare and test another copy

Requires Python 3, Quickshell and Bubblewrap. From the source checkout, use a
**new** output directory. The preparer refuses to overwrite a human preview:

```sh
python3 scripts/prepare-app-development-preview.py \
  --output "$HOME/.local/share/omabit-policy-preview-040" \
  --desktop-entry "$HOME/.local/share/applications/omabit-policy-preview-040.desktop"
python3 scripts/test-app-development-preview.py \
  --output /tmp/omabit-policy-tree-test-UNIQUE
```

The full-panel interaction fixture tests selection, the single-desk Allow path,
creation inheritance, offline/observer behavior, Setup navigation, responsive geometry and real pointer/keyboard
input for zooming, scaled node selection, scrolling and dragging. It uses a fresh networkless offscreen copy. Never run automated policy
mutations against a human's persistent preview.
