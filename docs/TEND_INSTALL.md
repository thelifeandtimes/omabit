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

The deterministic archive and its checksum are written under `dist/`.

## Install the desktop pieces

Extract the archive, inspect the plugin (Omarchy plugins execute unsandboxed in
the shell process), and run:

```sh
./install.sh
omarchy plugin enable io.omabit.tend
```

Pass `--enable` to combine the second step with installation. The default
targets are:

- `${XDG_CONFIG_HOME:-$HOME/.config}/omarchy/plugins/io.omabit.tend`
- `$HOME/.local/bin/omabit`

The installer refuses existing files and symbolic-link targets. `--force`
moves an existing installation to a timestamped backup before replacing it.

## Install the Gall desk

Create and mount a `%tend` desk from your ship's dojo, then provide that exact
mounted directory to the installer:

```hoon
|new-desk %tend
|mount %tend
```

```sh
./install.sh --desk-path /absolute/path/to/your/pier/tend
```

Then commit and start it:

```hoon
|commit %tend
|rein %tend [& %tend]
```

The desk source is also present as `desk/` in the archive for remote or custom
deployment workflows. Never copy a pier, key material, `+code`, or Eyre cookie
into the repository or release artifact.

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

To remove the local session and select another ship:

```sh
omabit tend disconnect
```

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
omabit tend leave 1 --yes
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
