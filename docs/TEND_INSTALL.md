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

## CLI examples

```sh
omabit tend today
omabit tend add "Buy milk" --list Inbox --tag groceries
omabit tend complete 1 42
omabit tend snooze 1 42 --minutes 20
omabit tend open 1 42
omabit --json tend list
```

Run `omabit tend --help` for the complete command list.
