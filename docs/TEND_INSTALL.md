# Installing Tend

Tend has three local pieces: the `%tend` desk on the user's Urbit ship, the
`io.omabit.tend` Omarchy shell plugin, and the `omabit` CLI. No Omabit-operated
account or database is involved.

## Build a release

From the Omabit repository:

```sh
make dist-tend
make check-tend-release
```

The deterministic archive and its checksum are written under `dist/`. The
tarball is the installable release: unpack it, enter the release directory,
and verify its internal manifest before running the installer. For example:

```sh
version=$(cat VERSION)
mkdir -p dist/unpacked
tar -xzf "dist/omabit-tend-$version.tar.gz" -C dist/unpacked
cd "dist/unpacked/omabit-tend-$version"
sha256sum --check SHA256SUMS
```

Use the same release archive for every Omarchy desktop that participates in a
sharing group. One ship publishes the `%tend` desk; the other ships install
that desk through Urbit's normal `%treaty` distribution. There is no central
Omabit service between them.

## Two machines and two ships

For the simplest multiplayer deployment, choose one ship as the Tend desk
publisher and list owner. In the examples below it is `~publisher`. The second
ship is `~participant`. Either ship may be a planet, moon, or comet; Tend does
not branch authorization on ship rank.

Each machine needs:

- a running, independently keyed Urbit ship with a reachable HTTPS domain;
- the current `+code` for that local ship when the desktop is connected;
- Omarchy with Quickshell and the verified Tend runtime dependencies; and
- the same extracted Tend release for installing the Omarchy plugin and CLI.

Only the publisher machine needs to overlay the release's Gall source into a
mounted desk. The participant receives the desk from the publisher with
`|install ~publisher %tend`.

## Prepare the publisher's Gall desk

Create and mount a `%tend` desk from your ship's dojo:

```hoon
|new-desk %tend
|mount %tend
```

Do this before running the release installer. The mounted directory supplies
the four minimal marks and a `sys.kelvin` compatible with that ship's
userspace. The release supplies the additional standard Urbit marks and
libraries that Tend imports, including `%bill`, `%mime`, and `%json`.

## Install the release on the publisher machine

From the root of the extracted release, inspect the plugin (Omarchy plugins
execute unsandboxed in the shell process), then install all three pieces in one
command:

```sh
./install.sh --desk-path /absolute/path/to/your/pier/tend --enable
```

Omit `--enable` if you want to enable the plugin separately after inspection:

```sh
omarchy plugin enable io.omabit.tend
```

The installer overlays Tend-owned files and Tend's bundled Urbit dependencies
into the mounted desk. It keeps the desk's ship-generated minimal marks and
`sys.kelvin`, and makes a timestamped backup of the complete pre-install desk.
It refuses a directory that does not look like a fresh or previously installed
mounted desk. On an update, differing installed files require `--force`; the
mounted base files are still preserved.

The default desktop targets are:

- `${XDG_CONFIG_HOME:-$HOME/.config}/omarchy/plugins/io.omabit.tend`
- `$HOME/.local/bin/omabit`

The installer refuses existing desktop files and symbolic-link targets.
`--force` moves existing desktop installations to timestamped backups before
replacing them. For a desktop-only install, omit `--desk-path`.

After the combined install, commit the desk and install it from the publisher
ship's own Clay source:

```hoon
|commit %tend
|install our %tend
```

The `|install our %tend` step is required for a new locally mounted desk. On a
later source update, `|commit %tend` builds and reloads the already installed
agent. If the agent was intentionally suspended, revive it with
`|revive %tend`; do not reinstall it merely to unsuspend it.

The desk overlay source is also present as `desk/` in the archive for remote or
custom deployment workflows. Do not replace a mounted desk wholesale with
that directory: it intentionally does not contain the four ship-generated
minimal marks, and its packaged `sys.kelvin` may not match the mounted ship.
Never copy a pier, key material, `+code`, or Eyre cookie into the repository or
release artifact.

## Publish the desk and install it on the participant

On the publisher ship, publish the committed desk:

```hoon
:treaty|publish %tend
```

Leave that ship running long enough for the participant to fetch the desk. On
the participant ship, install directly from the publisher:

```hoon
|install ~publisher %tend
```

The participant does not create or mount a `%tend` desk first. A successful
install starts the Gall agent and installs the web files and Landscape docket.
The participant desk follows the publisher's `%treaty` source; later published
desk revisions are delivered through Urbit's normal update path.

On the participant's Omarchy machine, install only the desktop pieces from the
same extracted release:

```sh
./install.sh --enable
```

Do not point this desktop-only command at the publisher's mounted desk. Each
desktop connects over Eyre/HTTP to its own local ship.

## Recover a desk replaced by an older installer

If `|commit %tend` reports deletions of `mar/hoon`, `mar/kelvin`, `mar/noun`,
and `mar/txt`, an older installer replaced the mounted desk with the release
overlay instead of merging it. It created a full backup immediately before the
replacement. Restore the five ship-provided files from that backup while
leaving the installed Tend files in place:

```sh
desk=/absolute/path/to/your/pier/tend
backup=/absolute/path/to/your/pier/tend.backup.TIMESTAMP
install -d "$desk/mar"
for mark in hoon kelvin noun txt; do
  cp -a "$backup/mar/$mark.hoon" "$desk/mar/$mark.hoon"
done
cp -a "$backup/sys.kelvin" "$desk/sys.kelvin"
```

After restoring those files, unpack the latest corrected release into a fresh
directory and run its installer with `--force` so the mounted desk receives
the complete dependency set:

```sh
./install.sh --force --desk-path "$desk"
```

Then retry `|commit %tend`. Keep the original backup until the desk has
committed and Tend has started successfully.

## Connect each desktop and browser

Open the Tend overlay and enter the ship's HTTPS domain plus the current
`+code`, or use the CLI (the code is read from a hidden prompt):

```sh
omabit tend connect https://sampel-palnet.arvo.network
omabit tend status
omabit tend list
```

Planets, moons, and comets use the same flow. Plain HTTP is accepted only for a
loopback development endpoint. Tend saves only the resulting Eyre session
cookie, in a mode-0600 runtime directory; it never saves the `+code`.
Login also requires desktop protocol 1 from the installed desk before either
the cookie or endpoint is saved. Install all components from the same release
and update every ship in a sharing group together; see
`TEND_COMPATIBILITY.md`.

Repeat the connection on the participant machine using the participant ship's
domain and `+code`, not the publisher's credentials. The Omarchy overlay header
shows the authenticated ship so that a user can confirm which identity is
active.

The same installed desk exposes the browser interface at:

```text
https://SHIP-DOMAIN/apps/tend
```

It also supplies a Tend card in Landscape. If the card does not appear after a
successful commit or remote install, reload Landscape and verify that `%tend`
is running before troubleshooting the desktop plugin.

To remove the local session and select another ship:

```sh
omabit tend disconnect
```

`omabit tend share LIST SHIP` prints a recipient-bound capability URI in its
result. The invited user normally sees the invitation in-app, but can also run
`omabit tend accept 'omabit://tend/invite/OWNER/TOKEN'`. The URI contains a
single-use invitation token, so share it only with the named recipient.

## Verify the multiplayer loop

Before treating a two-machine deployment as ready:

1. On the publisher's Omarchy overlay, create a private reminder and confirm it
   appears after reopening the overlay.
2. Create a shared list, invite the participant ship, add a reminder, and
   assign it to that ship.
3. On the participant's `/apps/tend` page, accept the invitation. The
   publisher's overlay must change the participant from pending to member.
4. Edit the assigned reminder in the participant browser. Confirm the changed
   title appears in the publisher overlay.
5. Complete it in the participant browser. Confirm the publisher overlay shows
   it checked and completed.

If the publisher/list-owner ship is Offline, Tend deliberately leaves the
participant replica readable but disables edits. Bring the owner back Online
and wait for the list status to return through Checking to Online before
retrying.

## Optional global shortcuts

The plugin does not modify Hyprland configuration during installation. Add
user-chosen bindings to `~/.config/hypr/bindings.conf`; for example:

```ini
bindd = SUPER, R, Tend, exec, $HOME/.local/bin/omabit tend open
bindd = SUPER SHIFT, R, Tend quick capture, exec, $HOME/.local/bin/omabit tend open --capture
```

Choose different keys if these conflict with local bindings. Hyprland reloads
the file automatically. Inside the overlay, `Ctrl+N` focuses quick add and
`Ctrl+F` focuses search; the full reference is in
`docs/TEND_ACCESSIBILITY.md`.

## CLI examples

```sh
omabit tend today
omabit --json tend lists
omabit --json tend show Inbox 42
omabit tend add "Buy milk" --list Inbox --tag groceries
omabit tend edit Inbox 42 --notes "Use oat milk" --priority high --flagged
omabit tend schedule Inbox 42 --due 2026-09-25T09:00 --timezone America/Los_Angeles --repeat weekly --weekday 5
omabit tend move-list Inbox 42 "Shared chores"
omabit tend complete 1 42
omabit tend complete 1 42 43 44
omabit tend move 1 42 43 --section 7 --rank 2048
omabit tend delete 1 42 43 --yes
omabit tend snooze 1 42 --minutes 20
omabit tend snooze 1 42 --until 2026-09-11T09:30:00-07:00
omabit tend share 1 ~sampel-palnet --can-invite
omabit tend invitations
omabit tend accept 'omabit://tend/invite/sampel-palnet/TOKEN'
omabit tend unshare 1 ~sampel-palnet
omabit tend leave 1
omabit tend open 1 42
omabit tend open --capture
omabit --json tend list
omabit tend export ~/Documents/tend-backup.json
omabit tend validate-backup ~/Documents/tend-backup.json
omabit tend restore ~/Documents/tend-backup.json --yes
```

`tend export` writes a versioned `tend-backup-1` JSON snapshot atomically with
mode 0600. It includes owner-authoritative list content and preferences, omits
visible replicas, and never includes the Eyre cookie or `+code`. Existing files
are refused unless `--force` is explicit; use `-` as the path to stream the
backup to standard output. Keep the result private. `validate-backup` checks an
existing file offline without contacting a ship or changing state. `restore`
repeats validation, requires `--yes`, and succeeds only against empty Gall
state. It does not recreate list sharing; the full contract is documented in
`docs/TEND_BACKUP.md`.

Run `omabit tend --help` for the complete command list.

The release also contains `skills/tend/SKILL.md`, an initial Codex-compatible
agent skill that uses the CLI's JSON mode. To make it discoverable to a local
Codex installation, copy the complete `skills/tend` directory into
`${CODEX_HOME:-$HOME/.codex}/skills/tend`. Review it before installation like
any other local skill. The protocol role model and action coverage are audited
in `docs/TEND_INTERFACE_COVERAGE.md`.

See `docs/TEND_ACCESSIBILITY.md` in the release archive for the full keyboard
reference and assistive-technology release checklist. See
`docs/TEND_OPERATIONS.md` for the tested compatibility baseline, update,
rollback, and troubleshooting procedures.
