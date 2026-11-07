# Persistent Setup preview

**Omabit 0.4.0 Preview** is a standalone window containing the real
**App development and publishing** Setup pane. Its Python adapter simulates two
runners and three ships. It does not run a collector, runner or Urbit, and cannot
commit or publish a desk. Fixture behavior is for UI review, not qualification of
the runner's policy enforcement.

Find **Omabit 0.4.0 Preview** in the desktop application launcher. Close its window
normally; your saved fixture policies persist. It does not start automatically at
login. From a terminal in the desktop session:

```sh
python3 ~/.local/share/omabit-policy-preview-040/launch.py start
python3 ~/.local/share/omabit-policy-preview-040/launch.py status
python3 ~/.local/share/omabit-policy-preview-040/launch.py stop
```

Review suggestions:

1. Both permissions start **Block**. Change and save Global permissions, then
   select Runner, Ship and Desk to try explicit Allow/Block and Inherit.
2. Use `preview-local` and its Publisher or Project member fixture. For Desk,
   enter a name such as `my-app`. Review the saved effective permissions and
   their source scope. These names do not refer to installed Clay desks.
3. Switch **preview-remote connection** to **Offline**, save a Global change,
   and inspect its pending rollout and last acknowledged values. Return it to
   **Online · operator**, then **Retry pending**.
4. Select **Online · observer** and the remote runner to review disabled editing.
5. Resize the window, scroll, switch the preview palette, and open Policy details.
   Close and reopen the window to check your saved values.

The connection simulation and palette start at Online/Dark each launch; saved
permissions and pending rollouts are retained. There is no automatic fixture
reset. Production, the existing DEV installation and `trial040` are separate.

## Isolation and files

The install is `~/.local/share/omabit-policy-preview-040/`; the separately named
launcher is `~/.local/share/applications/omabit-policy-preview-040.desktop`.
Saved policies are in `data/state/policy.json`, logs in `data/state/window.log`
and `launcher.log`, and an image of this window in `data/state/preview.png`.
One previous window log is retained. `source.json` records the source commit and
the copied file hashes. The pane and shared controls are copied unchanged.

Bubblewrap supplies private user, PID, mount, IPC and network namespaces, a
read-only source snapshot, and private HOME/XDG directories. Only preview data is
writable. Its sole host socket is the Wayland socket used to render the window;
host `/home`, `/run`, SSH, Docker, D-Bus, keyrings and runner sockets are absent.
It uses software rendering and controlled preview palettes. The launcher uses
an exclusive lock and an identity-checked process handle to close only its own
window. It does not install an Omarchy plugin or reload the shell.

## Prepare another isolated copy

Requires the host's Python 3, Quickshell and Bubblewrap. From the source checkout,
use a **new** output directory; the preparer refuses to overwrite an existing
preview or desktop launcher:

```sh
python3 scripts/prepare-app-development-preview.py \
  --output "$HOME/.local/share/omabit-policy-preview-040" \
  --desktop-entry "$HOME/.local/share/applications/omabit-policy-preview-040.desktop"
```

For automated visual checks, prepare a separate copy under `/tmp`, then run
`python3 /tmp/YOUR-FRESH-PREVIEW/launch.py smoke`. This uses an offscreen window,
no host display socket, and exits after capturing the loaded default editor.
Never run automated fixture mutations against a human's persistent preview.
