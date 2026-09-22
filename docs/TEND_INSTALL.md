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

## Prepare the Gall desk

Create and mount a `%tend` desk from your ship's dojo:

```hoon
|new-desk %tend
|mount %tend
```

Do this before running the release installer. The mounted directory supplies
standard marks and a `sys.kelvin` compatible with that ship's userspace.

## Install the release

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

The installer overlays only Tend-owned files into the mounted desk. It keeps
the desk's standard marks and `sys.kelvin`, and makes a timestamped backup of
the complete pre-install desk. It refuses a directory that does not look like
a fresh or previously installed mounted desk. On an update, differing
Tend-owned files require `--force`; the mounted base files are still preserved.

The default desktop targets are:

- `${XDG_CONFIG_HOME:-$HOME/.config}/omarchy/plugins/io.omabit.tend`
- `$HOME/.local/bin/omabit`

The installer refuses existing desktop files and symbolic-link targets.
`--force` moves existing desktop installations to timestamped backups before
replacing them. For a desktop-only install, omit `--desk-path`.

After the combined install, commit and start the desk:

```hoon
|commit %tend
|rein %tend [& %tend]
```

The desk overlay source is also present as `desk/` in the archive for remote or
custom deployment workflows. Do not replace a mounted desk wholesale with
that directory: it intentionally does not contain the standard base marks,
and its packaged `sys.kelvin` may not match the mounted ship. Never copy a
pier, key material, `+code`, or Eyre cookie into the repository or release
artifact.

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

Then retry `|commit %tend`. You do not need to rerun the installer merely to
complete this recovery. Keep the backup until the desk has committed and Tend
has started successfully.

## Connect

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

To remove the local session and select another ship:

```sh
omabit tend disconnect
```

`omabit tend share LIST SHIP` prints a recipient-bound capability URI in its
result. The invited user normally sees the invitation in-app, but can also run
`omabit tend accept 'omabit://tend/invite/OWNER/TOKEN'`. The URI contains a
single-use invitation token, so share it only with the named recipient.

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
omabit tend add "Buy milk" --list Inbox --tag groceries
omabit tend complete 1 42
omabit tend complete 1 42 43 44
omabit tend move 1 42 43 --section 7 --rank 2048
omabit tend delete 1 42 43 --yes
omabit tend snooze 1 42 --minutes 20
omabit tend snooze 1 42 --until 2026-09-11T09:30:00-07:00
omabit tend share 1 ~sampel-palnet --can-invite
omabit tend invitations
omabit tend accept 7
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

See `docs/TEND_ACCESSIBILITY.md` in the release archive for the full keyboard
reference and assistive-technology release checklist. See
`docs/TEND_OPERATIONS.md` for the tested compatibility baseline, update,
rollback, and troubleshooting procedures.
